package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// MsgTypeKatNote is the WS message type used for live note broadcasts. Mirrors
// the constant declared in interfaces/http/ws/client.go; redefined here so
// the application service does not depend on the http/ws package.
const MsgTypeKatNote = "kat.note"

// Default config values; overridable via env vars wired in main.go.
const (
	DefaultKatWindowSeconds          = 120
	DefaultKatStepSeconds            = 20
	DefaultKatMinNewSegments         = 2
	DefaultKatMaxConcurrentRuns      = 4
	DefaultKatCooldownAfterErrorSecs = 60
	DefaultKatRingBufferCapacity     = 20
	DefaultKatPromptVersion          = "kat-v1"
)

// KatMaxTranscriptChars is the per-call char budget for the user prompt.
// When the meeting's accumulated transcript exceeds this, the scheduler
// switches to rolling summarization: it splits segments into chunks,
// summarizes each chunk with the prior summary as context, then runs the
// final structured pass over the last chunk to produce the streaming note.
//
// 150,000 chars ≈ ~37k tokens for English text, leaving ~90k headroom in
// gpt-4o-mini's 128k context window for system prompt + structured output.
const KatMaxTranscriptChars = 150_000

// KatConfig carries the env-bound knobs for the Kat scheduler. Values are
// validated by NewKatNotesService; bad values fall back to defaults with a
// KAT_CONFIG_INVALID warn log.
type KatConfig struct {
	Enabled                bool
	WindowSeconds          int
	StepSeconds            int
	MinNewSegments         int
	MaxConcurrentRuns      int
	CooldownAfterErrorSecs int
	RingBufferCapacity     int
	PromptVersion          string
}

// Clock is an injectable wall-clock so tests can drive deterministic
// sliding-window math. Production wiring uses RealClock{}.
type Clock interface {
	Now() time.Time
}

// RealClock is the default Clock implementation backed by time.Now.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

// KatNotesService runs one ticker per active meeting / call ("cohort entry"),
// loading recent transcript segments via the existing TranscriptRepository,
// calling the Foundry NoteGenerator on a sliding window, and broadcasting the
// resulting KatNote over the WS hub. Notes are held in a per-cohort ring
// buffer (default capacity 20) for the late-join replay path; nothing is
// persisted to disk.
//
// See ../../../.kiro/specs/kat-live-notes/design.md for the lifecycle.
type KatNotesService struct {
	cfg         KatConfig
	transcripts ports.TranscriptRepository
	generator   ports.NoteGenerator
	meetings    ports.MeetingRepository
	calls       ports.CallRepository
	users       ports.UserRepository
	broadcaster ports.WSBroadcaster
	log         *zap.Logger
	clock       Clock

	mu     sync.Mutex
	cohort map[string]*cohortEntry
	sem    chan struct{}

	wg       sync.WaitGroup
	shutdown chan struct{}
}

// cohortEntry holds the per-meeting / per-call live state. The ringBufferMu
// is held only briefly during append + snapshot, so the hub's late-join
// handler can call SnapshotRingBuffer concurrently with a tick without
// blocking either the WS write loop or the generation goroutine.
type cohortEntry struct {
	parentKind string // "meeting" | "call"
	meetingID  *uuid.UUID
	callID     *uuid.UUID
	title      string

	labeller *KatSpeakerLabeller

	stateMu          sync.Mutex
	lastTickAt       time.Time
	lastSuccessRunID uuid.UUID
	lastSummary      string
	lastSegmentCount int
	lastErrorAt      time.Time

	// katEnabled gates whether the scheduler will spend a Foundry / OpenAI
	// call for this cohort. Defaults to false — clients must opt in via the
	// "AI notes" UI toggle before any cost is incurred. The cohort entry is
	// still created on participant.joined so the ring buffer continues to
	// fill from any prior runs (it doesn't), but ticks become no-ops until
	// at least one client flips the toggle on.
	katEnabled bool

	ringBufferMu sync.Mutex
	ringBuffer   []KatNote

	ticker *time.Ticker
	stop   chan struct{}
}

// NewKatNotesService validates cfg, applies defaults for any bad values,
// constructs the service, and returns it. The service is NOT started —
// callers must invoke Start() (or the per-cohort hooks) once dependencies
// are wired.
func NewKatNotesService(
	cfg KatConfig,
	transcripts ports.TranscriptRepository,
	generator ports.NoteGenerator,
	meetings ports.MeetingRepository,
	calls ports.CallRepository,
	users ports.UserRepository,
	broadcaster ports.WSBroadcaster,
	log *zap.Logger,
	clock Clock,
) *KatNotesService {
	cfg = sanitizeKatConfig(cfg, log)
	if clock == nil {
		clock = RealClock{}
	}
	return &KatNotesService{
		cfg:         cfg,
		transcripts: transcripts,
		generator:   generator,
		meetings:    meetings,
		calls:       calls,
		users:       users,
		broadcaster: broadcaster,
		log:         log,
		clock:       clock,
		cohort:      make(map[string]*cohortEntry),
		sem:         make(chan struct{}, cfg.MaxConcurrentRuns),
		shutdown:    make(chan struct{}),
	}
}

func sanitizeKatConfig(cfg KatConfig, log *zap.Logger) KatConfig {
	dirty := false
	if cfg.WindowSeconds <= 0 {
		cfg.WindowSeconds = DefaultKatWindowSeconds
		dirty = true
	}
	if cfg.StepSeconds <= 0 {
		cfg.StepSeconds = DefaultKatStepSeconds
		dirty = true
	}
	if cfg.WindowSeconds <= cfg.StepSeconds {
		cfg.WindowSeconds = DefaultKatWindowSeconds
		cfg.StepSeconds = DefaultKatStepSeconds
		dirty = true
	}
	if cfg.MinNewSegments < 1 {
		cfg.MinNewSegments = DefaultKatMinNewSegments
		dirty = true
	}
	if cfg.MaxConcurrentRuns < 1 {
		cfg.MaxConcurrentRuns = DefaultKatMaxConcurrentRuns
		dirty = true
	}
	if cfg.CooldownAfterErrorSecs < 0 {
		cfg.CooldownAfterErrorSecs = DefaultKatCooldownAfterErrorSecs
		dirty = true
	}
	if cfg.RingBufferCapacity < 1 {
		cfg.RingBufferCapacity = DefaultKatRingBufferCapacity
		dirty = true
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = DefaultKatPromptVersion
		dirty = true
	}
	if dirty && log != nil {
		catalog.KatConfigInvalid.Warn(log)
	}
	return cfg
}

// ─── Cohort lifecycle hooks ──────────────────────────────────────────────────

// OnParticipantJoined registers a meeting cohort on first join (or replays
// the existing buffer when one already exists). The hasActiveASR flag is
// retained for backward compatibility but no longer gates cohort creation —
// Kat checks the transcript_segments table directly on each tick. A meeting
// that nobody has spoken in yet will see ticks fire and find zero segments;
// the panel renders an "empty" state rather than waiting for an ASR session
// to start.
func (s *KatNotesService) OnParticipantJoined(meetingID uuid.UUID, userID uuid.UUID, _hasActiveASR bool) {
	if !s.cfg.Enabled || !s.generator.IsConfigured() {
		return
	}
	key := meetingKey(meetingID)

	// Replay first so a late-joiner sees existing notes immediately.
	s.replayRingBufferToUser(key, userID)

	s.mu.Lock()
	if _, ok := s.cohort[key]; ok {
		s.mu.Unlock()
		return
	}
	mid := meetingID
	ent := &cohortEntry{
		parentKind: "meeting",
		meetingID:  &mid,
		labeller:   NewKatSpeakerLabeller(s.users),
		ringBuffer: make([]KatNote, 0, s.cfg.RingBufferCapacity),
		stop:       make(chan struct{}),
	}
	if s.meetings != nil {
		if m, err := s.meetings.GetByID(context.Background(), meetingID); err == nil && m != nil {
			ent.title = m.Title
		}
	}
	s.cohort[key] = ent
	s.mu.Unlock()
	s.startTicker(ent)
}

// replayRingBufferToUser sends each note currently in the cohort's ring
// buffer to userID via WSBroadcaster.SendToUser. No-op when there's no
// cohort entry or the buffer is empty (the joining client stays in
// `warming_up` until the next successful tick).
func (s *KatNotesService) replayRingBufferToUser(key string, userID uuid.UUID) {
	s.mu.Lock()
	ent, ok := s.cohort[key]
	s.mu.Unlock()
	if !ok || s.broadcaster == nil {
		return
	}
	ent.ringBufferMu.Lock()
	notes := make([]KatNote, len(ent.ringBuffer))
	copy(notes, ent.ringBuffer)
	ent.ringBufferMu.Unlock()
	for _, n := range notes {
		s.broadcaster.SendToUser(userID, MsgTypeKatNote, n)
	}
}

// OnParticipantLeft removes the meeting cohort entry when isLast is true,
// stopping the ticker and dropping the ring buffer. No carry-over.
func (s *KatNotesService) OnParticipantLeft(meetingID uuid.UUID, isLast bool) {
	if !isLast {
		return
	}
	s.removeCohort(meetingKey(meetingID))
}

// OnMeetingEnded removes the cohort entry; runFinalTick (in tickLoop) handles
// the trailing run on its way out if there are pending segments.
func (s *KatNotesService) OnMeetingEnded(meetingID uuid.UUID) {
	s.removeCohort(meetingKey(meetingID))
}

// OnCallSessionOpened registers a solo call in the cohort.
func (s *KatNotesService) OnCallSessionOpened(callID uuid.UUID) {
	if !s.cfg.Enabled || !s.generator.IsConfigured() {
		return
	}
	key := callKey(callID)
	s.mu.Lock()
	if _, ok := s.cohort[key]; ok {
		s.mu.Unlock()
		return
	}
	cid := callID
	ent := &cohortEntry{
		parentKind: "call",
		callID:     &cid,
		labeller:   NewKatSpeakerLabeller(s.users),
		ringBuffer: make([]KatNote, 0, s.cfg.RingBufferCapacity),
		stop:       make(chan struct{}),
	}
	if s.calls != nil {
		if c, err := s.calls.GetByID(context.Background(), callID); err == nil && c != nil {
			ent.title = c.Title
		}
	}
	s.cohort[key] = ent
	s.mu.Unlock()
	s.startTicker(ent)
}

// OnCallSessionEnded drops the call cohort entry.
func (s *KatNotesService) OnCallSessionEnded(callID uuid.UUID) {
	s.removeCohort(callKey(callID))
}

// SetKatEnabled toggles the Foundry / OpenAI cost gate for a meeting cohort.
// While disabled, ticks become no-ops; the cohort entry stays alive so the
// ring buffer is preserved and the next enable resumes immediately. The
// caller is the WS hub, aggregating per-client AI-notes preferences.
func (s *KatNotesService) SetKatEnabled(meetingID uuid.UUID, enabled bool) {
	s.setKatEnabledByKey(meetingKey(meetingID), enabled)
}

// SetKatEnabledForCall is the solo-call analogue of SetKatEnabled.
func (s *KatNotesService) SetKatEnabledForCall(callID uuid.UUID, enabled bool) {
	s.setKatEnabledByKey(callKey(callID), enabled)
}

func (s *KatNotesService) setKatEnabledByKey(key string, enabled bool) {
	s.mu.Lock()
	ent, ok := s.cohort[key]
	s.mu.Unlock()
	if !ok {
		return
	}
	ent.stateMu.Lock()
	ent.katEnabled = enabled
	ent.stateMu.Unlock()
}

func (s *KatNotesService) removeCohort(key string) {
	s.mu.Lock()
	ent, ok := s.cohort[key]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.cohort, key)
	s.mu.Unlock()
	close(ent.stop)
}

// SnapshotRingBuffer returns a copy of the current ring-buffer contents for
// the given meeting. Returns an empty slice when the meeting is not in the
// cohort. Safe to call from the WS hub's late-join handler.
func (s *KatNotesService) SnapshotRingBuffer(meetingID uuid.UUID) []KatNote {
	return s.snapshot(meetingKey(meetingID))
}

// SnapshotRingBufferForCall is the solo-call analogue.
func (s *KatNotesService) SnapshotRingBufferForCall(callID uuid.UUID) []KatNote {
	return s.snapshot(callKey(callID))
}

func (s *KatNotesService) snapshot(key string) []KatNote {
	s.mu.Lock()
	ent, ok := s.cohort[key]
	s.mu.Unlock()
	if !ok {
		return []KatNote{}
	}
	ent.ringBufferMu.Lock()
	out := make([]KatNote, len(ent.ringBuffer))
	copy(out, ent.ringBuffer)
	ent.ringBufferMu.Unlock()
	return out
}

// ─── Ticker loop ─────────────────────────────────────────────────────────────

func (s *KatNotesService) startTicker(ent *cohortEntry) {
	ent.ticker = time.NewTicker(time.Duration(s.cfg.StepSeconds) * time.Second)
	s.wg.Add(1)
	go s.tickLoop(ent)
}

func (s *KatNotesService) tickLoop(ent *cohortEntry) {
	defer s.wg.Done()
	defer ent.ticker.Stop()
	for {
		select {
		case <-s.shutdown:
			return
		case <-ent.stop:
			s.runFinalTick(ent)
			return
		case t := <-ent.ticker.C:
			s.tick(ent, t.UTC())
		}
	}
}

// Tick is exposed for tests so they can drive ticks deterministically with a
// fake clock without waiting on time.Ticker.
func (s *KatNotesService) Tick(meetingOrCallID uuid.UUID, isMeeting bool) {
	key := callKey(meetingOrCallID)
	if isMeeting {
		key = meetingKey(meetingOrCallID)
	}
	s.mu.Lock()
	ent, ok := s.cohort[key]
	s.mu.Unlock()
	if !ok {
		return
	}
	s.tick(ent, s.clock.Now())
}

func (s *KatNotesService) tick(ent *cohortEntry, now time.Time) {
	ent.stateMu.Lock()
	// Per-meeting opt-in gate: tick is a no-op while no client has the
	// "AI notes" toggle on. This stops Foundry / OpenAI cost when nobody is
	// looking at the panel.
	if !ent.katEnabled {
		ent.stateMu.Unlock()
		return
	}
	if s.cfg.CooldownAfterErrorSecs > 0 && !ent.lastErrorAt.IsZero() &&
		now.Sub(ent.lastErrorAt) < time.Duration(s.cfg.CooldownAfterErrorSecs)*time.Second {
		ent.stateMu.Unlock()
		return
	}
	lastSummary := ent.lastSummary
	lastCount := ent.lastSegmentCount
	ent.stateMu.Unlock()

	// Load EVERY segment from the meeting / call so the model summarizes
	// the conversation in full. Earlier versions used a 120s sliding window,
	// which meant long meetings lost their early context (introductions,
	// decisions made in minute 1) by the time minute 30 came around. Now we
	// pass the full transcript and rely on chunking (below) when it grows
	// past the per-call token budget.
	windowEnd := now
	windowStart := time.Unix(0, 0) // far past — matches all segments

	segs, err := s.loadSegments(ent, windowStart, windowEnd)
	if err != nil {
		catalog.KatLoadSegmentsFailed.Warn(s.log, zap.Error(err))
		return
	}

	// Empty-window path: no segments persisted in the meeting's transcript
	// table for the current sliding window. Skip the OpenAI call entirely
	// and emit an `empty_window` broadcast so the frontend can render
	// "There's nothing to take notes" instead of staying stuck on
	// "Warming up". Also reset lastSegmentCount so the next tick that
	// finds segments treats them all as new.
	if len(segs) == 0 {
		ent.stateMu.Lock()
		ent.lastTickAt = now
		ent.lastSegmentCount = 0
		ent.stateMu.Unlock()
		s.broadcastEmptyWindow(ent, windowStart, windowEnd, now)
		catalog.KatTickNoop.Debug(s.log, zap.Int("segments", 0))
		return
	}

	// Gate: don't burn an OpenAI call if nothing new arrived since the
	// previous successful summary. Keeps the panel from rerendering the
	// same content every tick during long silences.
	if len(segs)-lastCount < s.cfg.MinNewSegments {
		ent.stateMu.Lock()
		ent.lastTickAt = now
		ent.stateMu.Unlock()
		catalog.KatTickNoop.Debug(s.log, zap.Int("segments", len(segs)))
		return
	}

	select {
	case s.sem <- struct{}{}:
	default:
		catalog.KatConcurrencyDeferred.Warn(s.log)
		return
	}

	go func() {
		defer func() { <-s.sem }()
		s.runGeneration(ent, segs, lastSummary, windowStart, windowEnd, now)
	}()
}

func (s *KatNotesService) runFinalTick(ent *cohortEntry) {
	now := s.clock.Now()
	ent.stateMu.Lock()
	if !ent.katEnabled {
		// Honour the same opt-in gate as the regular tick — nobody enabled
		// AI notes, so don't spend a final Foundry call on the way out.
		ent.stateMu.Unlock()
		return
	}
	lastCount := ent.lastSegmentCount
	lastSummary := ent.lastSummary
	ent.stateMu.Unlock()

	windowEnd := now
	windowStart := time.Unix(0, 0) // full transcript — matches the regular tick path
	segs, err := s.loadSegments(ent, windowStart, windowEnd)
	if err != nil || len(segs)-lastCount < s.cfg.MinNewSegments {
		return
	}
	s.runGeneration(ent, segs, lastSummary, windowStart, windowEnd, now)
}

func (s *KatNotesService) loadSegments(
	ent *cohortEntry,
	from, to time.Time,
) ([]*entities.TranscriptSegment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ent.parentKind == "meeting" {
		return s.transcripts.ListSegmentsByMeetingInRange(ctx, *ent.meetingID, from, to)
	}
	return s.transcripts.ListSegmentsByCallInRange(ctx, *ent.callID, from, to)
}

func (s *KatNotesService) runGeneration(
	ent *cohortEntry,
	segs []*entities.TranscriptSegment,
	previousSummary string,
	windowStart, windowEnd, now time.Time,
) {
	runID := uuid.New()
	catalog.KatRunStarted.Debug(s.log,
		zap.String("run_id", runID.String()),
		zap.String("parent_kind", ent.parentKind),
		zap.Int("segments", len(segs)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lite, labels := s.buildLiteSegments(ctx, ent, segs)

	// Suppress the placeholder "Brief greeting/setup." from being fed back
	// as PreviousSummary. Otherwise the model anchors on it and tends to
	// re-emit the same placeholder even when the new window has substantive
	// content. Empty PreviousSummary lets the model judge fresh.
	prevForPrompt := previousSummary
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(prevForPrompt)), "brief greeting") {
		prevForPrompt = ""
	}

	// Hierarchical fold: when the full transcript would blow past the
	// per-call char budget, split it into chunks and summarize each one
	// sequentially. Each non-final chunk's summary feeds into the next call
	// as PreviousSummary, so the model carries the meeting's earlier context
	// forward without ever seeing the raw earlier segments again. Only the
	// final chunk runs with the streaming callback so the user sees the
	// typewriter effect on the live structured output.
	chunks := chunkSegmentsByCharBudget(lite, KatMaxTranscriptChars)
	rollingSummary := prevForPrompt
	for i := 0; i < len(chunks)-1; i++ {
		partial := chunks[i]
		partialIn := ports.NoteGeneratorInput{
			PromptVersion:   s.cfg.PromptVersion,
			MeetingTitle:    ent.title,
			SpeakerLabels:   labels,
			Segments:        partial,
			PreviousSummary: rollingSummary,
		}
		s.log.Info("kat: rolling chunk start",
			zap.String("run_id", runID.String()),
			zap.Int("chunk_index", i),
			zap.Int("chunk_count", len(chunks)),
			zap.Int("chunk_segments", len(partial)),
		)
		partialOut, perr := s.generator.Generate(ctx, partialIn, nil)
		if perr != nil {
			ent.stateMu.Lock()
			ent.lastErrorAt = now
			ent.stateMu.Unlock()
			catalog.KatRunFailed.Warn(s.log,
				zap.String("run_id", runID.String()),
				zap.String("parent_kind", ent.parentKind),
				zap.Int("chunk_index", i),
				zap.Error(perr),
			)
			return
		}
		rollingSummary = partialOut.Summary
		s.log.Info("kat: rolling chunk done",
			zap.String("run_id", runID.String()),
			zap.Int("chunk_index", i),
			zap.String("rolling_summary", rollingSummary),
		)
	}

	// Final chunk: structured streaming pass that produces the broadcast note.
	// In the common (small-meeting) case chunks has length 1 and this runs
	// over the whole transcript — same shape as before chunking existed.
	finalSegs := chunks[len(chunks)-1]
	in := ports.NoteGeneratorInput{
		PromptVersion:   s.cfg.PromptVersion,
		MeetingTitle:    ent.title,
		SpeakerLabels:   labels,
		Segments:        finalSegs,
		PreviousSummary: rollingSummary,
	}

	// Log the full transcript fed into the model so we can debug
	// "Brief greeting/setup" / empty-output cases — usually they mean the
	// model legitimately saw only filler segments, but having the input
	// side visible in the log lets us confirm vs. (e.g.) speaker labels
	// being mangled or segment text getting truncated.
	transcriptForLog := make([]string, 0, len(finalSegs))
	for _, seg := range finalSegs {
		transcriptForLog = append(transcriptForLog, "["+seg.SpeakerLabel+"] "+seg.Text)
	}
	s.log.Info("kat: generation input",
		zap.String("run_id", runID.String()),
		zap.String("parent_kind", ent.parentKind),
		zap.Int("total_segments", len(lite)),
		zap.Int("chunks", len(chunks)),
		zap.Int("final_chunk_segments", len(finalSegs)),
		zap.String("prompt_version", s.cfg.PromptVersion),
		zap.String("meeting_title", ent.title),
		zap.String("previous_summary", rollingSummary),
		zap.Strings("transcript", transcriptForLog),
	)

	// Stream the partial response over WS as tokens arrive. The chunk
	// callback fires off the OpenAI hot path (every ~250ms throttle) — we
	// hand each partial to broadcastStreamingChunk which builds a transient
	// kat.note { status: 'streaming', summary } and fans it to participants.
	streamRunID := runID // captured by closure
	out, err := s.generator.Generate(ctx, in, func(partial string) {
		s.broadcastStreamingChunk(ent, streamRunID, partial, windowStart, windowEnd, now)
	})

	if err != nil {
		ent.stateMu.Lock()
		ent.lastErrorAt = now
		ent.stateMu.Unlock()
		catalog.KatRunFailed.Warn(s.log,
			zap.String("run_id", runID.String()),
			zap.String("parent_kind", ent.parentKind),
			zap.Error(err),
		)
		return
	}

	loIdx, hiIdx := segmentIndexBounds(segs)
	note := KatNote{
		ID:               uuid.New(),
		RunID:            runID,
		MeetingID:        ent.meetingID,
		CallID:           ent.callID,
		WindowStartedAt:  windowStart,
		WindowEndedAt:    windowEnd,
		SegmentIndexLo:   loIdx,
		SegmentIndexHi:   hiIdx,
		Summary:          out.Summary,
		KeyPoints:        out.KeyPoints,
		OpenQuestions:    out.OpenQuestions,
		ModelID:          out.ModelID,
		PromptVersion:    s.cfg.PromptVersion,
		PromptTokens:     out.PromptTokens,
		CompletionTokens: out.CompletionTokens,
		LatencyMs:        out.LatencyMs,
		Status:           KatNoteStatusOK,
		CreatedAt:        now,
	}

	s.pushToRingBuffer(ent, note)

	ent.stateMu.Lock()
	ent.lastSuccessRunID = runID
	ent.lastSummary = note.Summary
	ent.lastSegmentCount = len(segs)
	ent.lastTickAt = now
	ent.lastErrorAt = time.Time{}
	ent.stateMu.Unlock()

	s.broadcastNote(ent, note)

	// Log the full note (summary + bullets) at Info so we can diff what the
	// panel SHOULD show against what actually rendered. The transcript text
	// the model saw is logged below on the input side.
	catalog.KatRunOk.Info(s.log,
		zap.String("run_id", runID.String()),
		zap.String("parent_kind", ent.parentKind),
		zap.Int("segments", len(segs)),
		zap.Int32("latency_ms", out.LatencyMs),
		zap.String("note_summary", note.Summary),
		zap.Strings("note_key_points", note.KeyPoints),
		zap.Strings("note_open_questions", note.OpenQuestions),
		zap.Int32p("prompt_tokens", note.PromptTokens),
		zap.Int32p("completion_tokens", note.CompletionTokens),
	)
}

func (s *KatNotesService) buildLiteSegments(
	ctx context.Context,
	ent *cohortEntry,
	segs []*entities.TranscriptSegment,
) ([]ports.TranscriptSegmentLite, map[uuid.UUID]string) {
	lite := make([]ports.TranscriptSegmentLite, 0, len(segs))
	labels := make(map[uuid.UUID]string)
	for _, seg := range segs {
		label, ok := labels[seg.SpeakerUserID]
		if !ok {
			label = ent.labeller.Label(ctx, seg.SpeakerUserID)
			labels[seg.SpeakerUserID] = label
		}
		lite = append(lite, ports.TranscriptSegmentLite{
			SpeakerLabel: label,
			Text:         seg.Text,
			StartMs:      seg.StartMs,
			EndMs:        seg.EndMs,
		})
	}
	return lite, labels
}

func (s *KatNotesService) pushToRingBuffer(ent *cohortEntry, note KatNote) {
	ent.ringBufferMu.Lock()
	defer ent.ringBufferMu.Unlock()
	if len(ent.ringBuffer) >= s.cfg.RingBufferCapacity {
		// FIFO eviction of the oldest entry.
		ent.ringBuffer = append(ent.ringBuffer[1:], note)
		return
	}
	ent.ringBuffer = append(ent.ringBuffer, note)
}

func (s *KatNotesService) broadcastNote(ent *cohortEntry, note KatNote) {
	if s.broadcaster == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			catalog.KatBroadcastFailed.Warn(s.log,
				zap.String("run_id", note.RunID.String()),
			)
		}
	}()
	if ent.meetingID != nil {
		s.broadcaster.BroadcastToMeeting(*ent.meetingID, MsgTypeKatNote, note)
		return
	}
	// Solo call: broadcast to everyone connected to the user that owns the
	// call; in v1 this is the call owner only, but the Hub adapter handles
	// fan-out. We use BroadcastToMeeting with a sentinel? — no: solo calls
	// use SendToUser; main.go wires the call owner's UserID via a closure
	// adapter that translates BroadcastToMeeting into SendToUser. Here we
	// fall back to a no-op broadcast if no meeting is bound; the hub adapter
	// is responsible for the call routing.
	if ent.callID != nil {
		// The broadcaster's caller-side adapter is expected to know how to
		// route a solo-call note (it owns the call->user mapping). We pass
		// the callID inside the payload so the adapter can recover routing
		// without an extra arg on the port. In v1 the WSBroadcaster impl is
		// expected to inspect note.CallID and route accordingly.
		s.broadcaster.BroadcastToMeeting(*ent.callID, MsgTypeKatNote, note)
	}
}

// broadcastStreamingChunk emits a transient KatNote with status='streaming'
// carrying the running raw text from the LLM. The frontend uses RunID to
// dedupe consecutive partials and replace previous render with the latest.
// NOT pushed to the ring buffer (chunks are intermediate state, not durable
// notes).
func (s *KatNotesService) broadcastStreamingChunk(
	ent *cohortEntry,
	runID uuid.UUID,
	partial string,
	windowStart, windowEnd, now time.Time,
) {
	if s.broadcaster == nil {
		return
	}
	note := KatNote{
		ID:              runID, // stable across chunks for the same run, so FE can replace
		RunID:           runID,
		MeetingID:       ent.meetingID,
		CallID:          ent.callID,
		WindowStartedAt: windowStart,
		WindowEndedAt:   windowEnd,
		Summary:         partial, // raw plain-text partial; FE renders progressively
		KeyPoints:       []string{},
		OpenQuestions:   []string{},
		ModelID:         s.generator.ModelID(),
		PromptVersion:   s.cfg.PromptVersion,
		Status:          KatNoteStatusStreaming,
		CreatedAt:       now,
	}
	defer func() { _ = recover() }()
	if ent.meetingID != nil {
		s.broadcaster.BroadcastToMeeting(*ent.meetingID, MsgTypeKatNote, note)
		return
	}
	if ent.callID != nil {
		s.broadcaster.BroadcastToMeeting(*ent.callID, MsgTypeKatNote, note)
	}
}

// broadcastEmptyWindow emits a placeholder KatNote with status='empty_window'
// so the frontend can render "There's nothing to take notes" instead of
// staying in the warming-up state forever. The note is NOT pushed to the
// ring buffer (late-joiners shouldn't see stale empty markers); it's a
// transient signal only.
func (s *KatNotesService) broadcastEmptyWindow(ent *cohortEntry, windowStart, windowEnd, now time.Time) {
	if s.broadcaster == nil {
		return
	}
	note := KatNote{
		ID:              uuid.New(),
		RunID:           uuid.New(),
		MeetingID:       ent.meetingID,
		CallID:          ent.callID,
		WindowStartedAt: windowStart,
		WindowEndedAt:   windowEnd,
		Summary:         "",
		KeyPoints:       []string{},
		OpenQuestions:   []string{},
		ModelID:         s.generator.ModelID(),
		PromptVersion:   s.cfg.PromptVersion,
		Status:          KatNoteStatusEmptyWindow,
		CreatedAt:       now,
	}
	defer func() { _ = recover() }()
	if ent.meetingID != nil {
		s.broadcaster.BroadcastToMeeting(*ent.meetingID, MsgTypeKatNote, note)
		return
	}
	if ent.callID != nil {
		s.broadcaster.BroadcastToMeeting(*ent.callID, MsgTypeKatNote, note)
	}
}

// chunkSegmentsByCharBudget splits segs into consecutive chunks where each
// chunk's combined transcript text length stays under maxChars. Always returns
// at least one chunk (possibly empty) so callers can index chunks[len-1].
//
// A segment that is itself larger than maxChars goes into its own chunk —
// the model will still see it but the budget is informational only at that
// point. In practice transcript segments are short (<500 chars), so this
// edge case is rare.
func chunkSegmentsByCharBudget(segs []ports.TranscriptSegmentLite, maxChars int) [][]ports.TranscriptSegmentLite {
	if len(segs) == 0 {
		return [][]ports.TranscriptSegmentLite{{}}
	}
	if maxChars <= 0 {
		return [][]ports.TranscriptSegmentLite{segs}
	}
	var chunks [][]ports.TranscriptSegmentLite
	var cur []ports.TranscriptSegmentLite
	curLen := 0
	for _, seg := range segs {
		// Approximate per-segment cost: speaker prefix + text + line break.
		segLen := len(seg.Text) + len(seg.SpeakerLabel) + 4
		if curLen+segLen > maxChars && len(cur) > 0 {
			chunks = append(chunks, cur)
			cur = nil
			curLen = 0
		}
		cur = append(cur, seg)
		curLen += segLen
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

func segmentIndexBounds(segs []*entities.TranscriptSegment) (lo, hi int32) {
	if len(segs) == 0 {
		return 0, 0
	}
	lo = segs[0].SegmentIndex
	hi = segs[0].SegmentIndex
	for _, seg := range segs[1:] {
		if seg.SegmentIndex < lo {
			lo = seg.SegmentIndex
		}
		if seg.SegmentIndex > hi {
			hi = seg.SegmentIndex
		}
	}
	return lo, hi
}

// ─── Shutdown ────────────────────────────────────────────────────────────────

// Shutdown stops every cohort ticker and waits for in-flight runs to drain
// (capped by ctx). Ring buffers are dropped along with the cohort entries —
// the next process start sees an empty cohort.
func (s *KatNotesService) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	close(s.shutdown)
	for k, ent := range s.cohort {
		select {
		case <-ent.stop:
			// already closed
		default:
			close(ent.stop)
		}
		delete(s.cohort, k)
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return errors.New("kat: shutdown timed out")
	}
}

func meetingKey(id uuid.UUID) string { return "meeting:" + id.String() }
func callKey(id uuid.UUID) string    { return "call:" + id.String() }
