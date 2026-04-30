package services

import (
	"context"
	"errors"
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

// OnParticipantJoined registers a meeting in the cohort if this is the first
// active participant + ASR session. It also replays the cohort's current
// ring-buffer contents to userID via the broadcaster's SendToUser so the
// joiner sees recent notes immediately (without an HTTP round-trip).
func (s *KatNotesService) OnParticipantJoined(meetingID uuid.UUID, userID uuid.UUID, hasActiveASR bool) {
	if !s.cfg.Enabled || !s.generator.IsConfigured() {
		return
	}
	key := meetingKey(meetingID)

	// Replay first: even if hasActiveASR is false (e.g. the new participant
	// is a viewer who isn't running ASR locally), they should still see the
	// notes accumulated from other speakers' streams.
	s.replayRingBufferToUser(key, userID)

	if !hasActiveASR {
		// Lazy: only join the cohort once at least one ASR session is open.
		return
	}

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
	if s.cfg.CooldownAfterErrorSecs > 0 && !ent.lastErrorAt.IsZero() &&
		now.Sub(ent.lastErrorAt) < time.Duration(s.cfg.CooldownAfterErrorSecs)*time.Second {
		ent.stateMu.Unlock()
		return
	}
	lastSummary := ent.lastSummary
	lastCount := ent.lastSegmentCount
	ent.stateMu.Unlock()

	windowEnd := now
	windowStart := now.Add(-time.Duration(s.cfg.WindowSeconds) * time.Second)

	segs, err := s.loadSegments(ent, windowStart, windowEnd)
	if err != nil {
		catalog.KatLoadSegmentsFailed.Warn(s.log, zap.Error(err))
		return
	}

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
	lastCount := ent.lastSegmentCount
	lastSummary := ent.lastSummary
	ent.stateMu.Unlock()

	windowEnd := now
	windowStart := now.Add(-time.Duration(s.cfg.WindowSeconds) * time.Second)
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

	in := ports.NoteGeneratorInput{
		PromptVersion:   s.cfg.PromptVersion,
		MeetingTitle:    ent.title,
		SpeakerLabels:   labels,
		Segments:        lite,
		PreviousSummary: previousSummary,
	}
	out, err := s.generator.Generate(ctx, in)

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

	catalog.KatRunOk.Info(s.log,
		zap.String("run_id", runID.String()),
		zap.String("parent_kind", ent.parentKind),
		zap.Int("segments", len(segs)),
		zap.Int32("latency_ms", out.LatencyMs),
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
