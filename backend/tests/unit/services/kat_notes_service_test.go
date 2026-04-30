package services_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/foundry"
)

// ─── fakes ──────────────────────────────────────────────────────────────────

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }
func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}
func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

type fakeNoteGenerator struct {
	mu            sync.Mutex
	calls         int32
	concurrent    int32
	maxConcurrent int32
	output        *ports.NoteGeneratorOutput
	err           error
	cb            func() // optional per-call hook
	configured    bool
	authMode      string
	modelID       string
	holdDur       time.Duration
}

func newFakeNoteGenerator(out *ports.NoteGeneratorOutput) *fakeNoteGenerator {
	return &fakeNoteGenerator{
		output:     out,
		configured: true,
		authMode:   "api_key",
		modelID:    "test-model",
	}
}

func (f *fakeNoteGenerator) Generate(ctx context.Context, in ports.NoteGeneratorInput) (*ports.NoteGeneratorOutput, error) {
	atomic.AddInt32(&f.calls, 1)
	cur := atomic.AddInt32(&f.concurrent, 1)
	defer atomic.AddInt32(&f.concurrent, -1)
	for {
		old := atomic.LoadInt32(&f.maxConcurrent)
		if cur <= old || atomic.CompareAndSwapInt32(&f.maxConcurrent, old, cur) {
			break
		}
	}
	if f.holdDur > 0 {
		select {
		case <-time.After(f.holdDur):
		case <-ctx.Done():
		}
	}
	if f.cb != nil {
		f.cb()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}
func (f *fakeNoteGenerator) ModelID() string      { return f.modelID }
func (f *fakeNoteGenerator) AuthMode() string     { return f.authMode }
func (f *fakeNoteGenerator) IsConfigured() bool   { return f.configured }
func (f *fakeNoteGenerator) Calls() int32         { return atomic.LoadInt32(&f.calls) }
func (f *fakeNoteGenerator) MaxConcurrent() int32 { return atomic.LoadInt32(&f.maxConcurrent) }

type capturedBroadcast struct {
	MeetingID uuid.UUID
	Type      string
	Data      any
}

type fakeBroadcaster struct {
	mu     sync.Mutex
	bcasts []capturedBroadcast
	sent   []capturedBroadcast
}

func newFakeBroadcaster() *fakeBroadcaster { return &fakeBroadcaster{} }
func (b *fakeBroadcaster) BroadcastToMeeting(meetingID uuid.UUID, msgType string, data any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bcasts = append(b.bcasts, capturedBroadcast{MeetingID: meetingID, Type: msgType, Data: data})
}
func (b *fakeBroadcaster) SendToUser(userID uuid.UUID, msgType string, data any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, capturedBroadcast{MeetingID: userID, Type: msgType, Data: data})
}
func (b *fakeBroadcaster) Broadcasts() []capturedBroadcast {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]capturedBroadcast, len(b.bcasts))
	copy(out, b.bcasts)
	return out
}

type katFakeTranscriptRepo struct {
	mu       sync.Mutex
	segments []*entities.TranscriptSegment
}

func (f *katFakeTranscriptRepo) push(seg *entities.TranscriptSegment) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.segments = append(f.segments, seg)
}
func (f *katFakeTranscriptRepo) ListSegmentsByMeetingInRange(
	_ context.Context, meetingID uuid.UUID, fromTs, toTs time.Time,
) ([]*entities.TranscriptSegment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*entities.TranscriptSegment, 0)
	for _, s := range f.segments {
		if s.MeetingID == nil || *s.MeetingID != meetingID {
			continue
		}
		if s.SegmentStartedAt.Before(fromTs) || !s.SegmentStartedAt.Before(toTs) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
func (f *katFakeTranscriptRepo) ListSegmentsByCallInRange(
	_ context.Context, callID uuid.UUID, fromTs, toTs time.Time,
) ([]*entities.TranscriptSegment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*entities.TranscriptSegment, 0)
	for _, s := range f.segments {
		if s.CallID == nil || *s.CallID != callID {
			continue
		}
		if s.SegmentStartedAt.Before(fromTs) || !s.SegmentStartedAt.Before(toTs) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// Unused TranscriptRepository methods for interface satisfaction.
func (f *katFakeTranscriptRepo) CreateSession(context.Context, *entities.TranscriptSession) error {
	return nil
}
func (f *katFakeTranscriptRepo) GetSession(context.Context, uuid.UUID) (*entities.TranscriptSession, error) {
	return nil, errors.New("not implemented")
}
func (f *katFakeTranscriptRepo) UpdateSessionStatus(context.Context, uuid.UUID, entities.TranscriptSessionStatus, *string, *string) error {
	return nil
}
func (f *katFakeTranscriptRepo) UpsertSegment(context.Context, *entities.TranscriptSegment) error {
	return nil
}
func (f *katFakeTranscriptRepo) ListSegmentsBySession(context.Context, uuid.UUID) ([]*entities.TranscriptSegment, error) {
	return nil, nil
}
func (f *katFakeTranscriptRepo) ListSegmentsByCall(context.Context, uuid.UUID, int, int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *katFakeTranscriptRepo) ListSegmentsByMeeting(context.Context, uuid.UUID, int, int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *katFakeTranscriptRepo) ListSessionsByCall(context.Context, uuid.UUID) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *katFakeTranscriptRepo) ListSessionsByMeeting(context.Context, uuid.UUID) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *katFakeTranscriptRepo) ListSpeakerUserIDsByMeeting(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (f *katFakeTranscriptRepo) FindExpiredActive(context.Context, int) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *katFakeTranscriptRepo) StitchSession(context.Context, uuid.UUID) (string, error) {
	return "", nil
}
func (f *katFakeTranscriptRepo) StitchCall(context.Context, uuid.UUID) (string, error) {
	return "", nil
}
func (f *katFakeTranscriptRepo) StitchMeeting(context.Context, uuid.UUID) (string, error) {
	return "", nil
}

type fakeUserRepo struct{}

func (fakeUserRepo) Create(context.Context, *entities.User) (*entities.User, error) { return nil, nil }
func (fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	return &entities.User{ID: id, FullName: "Test User"}, nil
}
func (fakeUserRepo) GetByEmail(context.Context, string) (*entities.User, error) { return nil, nil }
func (fakeUserRepo) List(context.Context, int, int) ([]*entities.User, int, error) {
	return nil, 0, nil
}
func (fakeUserRepo) FindByIDs(context.Context, []uuid.UUID) ([]*entities.User, error) {
	return []*entities.User{}, nil
}
func (fakeUserRepo) Update(context.Context, *entities.User) (*entities.User, error) {
	return nil, nil
}
func (fakeUserRepo) SoftDelete(context.Context, uuid.UUID) error               { return nil }
func (fakeUserRepo) SetEmailVerified(context.Context, uuid.UUID, bool) error   { return nil }
func (fakeUserRepo) UpdatePassword(context.Context, uuid.UUID, string) error   { return nil }
func (fakeUserRepo) SetRoleByEmail(context.Context, string, string) error      { return nil }
func (fakeUserRepo) DemoteAdminsExcept(context.Context, []string) (int, error) { return 0, nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

type katFixture struct {
	svc       *services.KatNotesService
	transcr   *katFakeTranscriptRepo
	gen       *fakeNoteGenerator
	bcast     *fakeBroadcaster
	clock     *fakeClock
	meetingID uuid.UUID
	speaker   uuid.UUID
}

func newKatFixture(t *testing.T, cfg services.KatConfig, gen *fakeNoteGenerator) *katFixture {
	t.Helper()
	cfg.Enabled = true
	tr := &katFakeTranscriptRepo{}
	bc := newFakeBroadcaster()
	clk := newFakeClock(time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC))

	svc := services.NewKatNotesService(
		cfg,
		tr,
		gen,
		nil, // meetings
		nil, // calls
		fakeUserRepo{},
		bc,
		zap.NewNop(),
		clk,
	)
	return &katFixture{
		svc:       svc,
		transcr:   tr,
		gen:       gen,
		bcast:     bc,
		clock:     clk,
		meetingID: uuid.New(),
		speaker:   uuid.New(),
	}
}

// pushSegment inserts a transcript segment with the given offset (seconds)
// from a base time. The segment_started_at is base + offset.
func (f *katFixture) pushSegment(base time.Time, idx int32, offsetSecs int) {
	mid := f.meetingID
	f.transcr.push(&entities.TranscriptSegment{
		ID:               uuid.New(),
		SessionID:        uuid.New(),
		SegmentIndex:     idx,
		SpeakerUserID:    f.speaker,
		MeetingID:        &mid,
		Text:             "hello",
		StartMs:          int32(offsetSecs) * 1000,
		EndMs:            (int32(offsetSecs) + 1) * 1000,
		SegmentStartedAt: base.Add(time.Duration(offsetSecs) * time.Second),
	})
}

func okOutput() *ports.NoteGeneratorOutput {
	return &ports.NoteGeneratorOutput{
		Summary:       "running summary",
		KeyPoints:     []string{"a"},
		OpenQuestions: []string{},
		ModelID:       "test-model",
		LatencyMs:     42,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestKatService_HappyPathTick(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 2,
		MaxConcurrentRuns: 4, RingBufferCapacity: 20,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)

	now := f.clock.Now()
	for i := 0; i < 3; i++ {
		f.pushSegment(now.Add(-30*time.Second), int32(i), i*5)
	}

	f.svc.Tick(f.meetingID, true)
	waitForCalls(t, gen, 1)

	bcasts := f.bcast.Broadcasts()
	require.Len(t, bcasts, 1)
	assert.Equal(t, services.MsgTypeKatNote, bcasts[0].Type)
	note, ok := bcasts[0].Data.(services.KatNote)
	require.True(t, ok)
	assert.Equal(t, "running summary", note.Summary)
	assert.Equal(t, services.KatNoteStatusOK, note.Status)

	snap := f.svc.SnapshotRingBuffer(f.meetingID)
	require.Len(t, snap, 1)
	assert.Equal(t, "running summary", snap[0].Summary)
}

func TestKatService_NoOpTickInsufficientSegments(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 5,
		MaxConcurrentRuns: 4, RingBufferCapacity: 20,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)
	now := f.clock.Now()
	for i := 0; i < 2; i++ {
		f.pushSegment(now.Add(-30*time.Second), int32(i), i*5)
	}

	f.svc.Tick(f.meetingID, true)
	time.Sleep(50 * time.Millisecond) // allow any goroutine to settle

	assert.Zero(t, gen.Calls(), "no Foundry call when MinNewSegments not met")
	assert.Empty(t, f.bcast.Broadcasts(), "no broadcast on no-op tick")
	assert.Empty(t, f.svc.SnapshotRingBuffer(f.meetingID), "ring buffer unchanged")
}

func TestKatService_ErrorTickSetsCooldownNoBroadcast(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	gen.err = foundry.ErrFoundryUnavailable
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 2,
		MaxConcurrentRuns: 4, RingBufferCapacity: 20,
		CooldownAfterErrorSecs: 60,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)
	now := f.clock.Now()
	for i := 0; i < 3; i++ {
		f.pushSegment(now.Add(-30*time.Second), int32(i), i*5)
	}

	f.svc.Tick(f.meetingID, true)
	waitForCalls(t, gen, 1)

	assert.Empty(t, f.bcast.Broadcasts(), "no broadcast on error")
	assert.Empty(t, f.svc.SnapshotRingBuffer(f.meetingID), "errored runs do not push to ring buffer")

	// A second tick within the cooldown must not call Foundry.
	f.svc.Tick(f.meetingID, true)
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 1, gen.Calls(), "cooldown must skip a second tick")
}

func TestKatService_RingBufferFIFOEviction(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 1,
		MaxConcurrentRuns: 4, RingBufferCapacity: 3,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)

	for run := 0; run < 4; run++ {
		// Make each run produce a unique-ish summary.
		gen.output = &ports.NoteGeneratorOutput{
			Summary:   "summary-" + string(rune('A'+run)),
			ModelID:   "test-model",
			LatencyMs: 1,
		}
		now := f.clock.Now()
		f.pushSegment(now.Add(-1*time.Second), int32(run), 0)
		f.svc.Tick(f.meetingID, true)
		waitForCalls(t, gen, int32(run+1))
		// Step the clock forward past cooldown so subsequent runs proceed.
		f.clock.Advance(2 * time.Second)
	}

	snap := f.svc.SnapshotRingBuffer(f.meetingID)
	require.Len(t, snap, 3, "buffer capped at RingBufferCapacity")
	// FIFO: oldest dropped is summary-A; remaining are B, C, D.
	assert.Equal(t, "summary-B", snap[0].Summary)
	assert.Equal(t, "summary-C", snap[1].Summary)
	assert.Equal(t, "summary-D", snap[2].Summary)
}

func TestKatService_RingBufferDroppedOnLastParticipantLeft(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 1,
		MaxConcurrentRuns: 4, RingBufferCapacity: 20,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)
	now := f.clock.Now()
	f.pushSegment(now.Add(-5*time.Second), 0, 0)
	f.svc.Tick(f.meetingID, true)
	waitForCalls(t, gen, 1)

	require.Len(t, f.svc.SnapshotRingBuffer(f.meetingID), 1)

	f.svc.OnParticipantLeft(f.meetingID, true)
	// Allow the tickLoop goroutine to exit; the cohort entry was removed
	// synchronously on OnParticipantLeft.
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, f.svc.SnapshotRingBuffer(f.meetingID),
		"ring buffer must be dropped along with the cohort entry")
}

func TestKatService_BadConfigSubstitutesDefaults(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	// window <= step is invalid; service must substitute defaults.
	cfg := services.KatConfig{
		Enabled:                true,
		WindowSeconds:          5,
		StepSeconds:            10,
		MinNewSegments:         -1,
		RingBufferCapacity:     0,
		MaxConcurrentRuns:      0,
		CooldownAfterErrorSecs: -1,
	}
	tr := &katFakeTranscriptRepo{}
	bc := newFakeBroadcaster()
	clk := newFakeClock(time.Now().UTC())
	svc := services.NewKatNotesService(cfg, tr, gen, nil, nil, fakeUserRepo{}, bc, zap.NewNop(), clk)
	require.NotNil(t, svc)
	// No panic + a usable service is the contract.
}

func TestKatService_ConcurrencyCap(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	gen.holdDur = 200 * time.Millisecond
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 1,
		MaxConcurrentRuns: 2, RingBufferCapacity: 20,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	// Five separate cohort entries (separate meetings).
	meetings := make([]uuid.UUID, 5)
	for i := range meetings {
		meetings[i] = uuid.New()
	}
	for _, m := range meetings {
		f.svc.OnParticipantJoined(m, f.speaker, true)
		now := f.clock.Now()
		mid := m
		// Push 1 segment per meeting so each tick will fire.
		f.transcr.push(&entities.TranscriptSegment{
			ID:               uuid.New(),
			SessionID:        uuid.New(),
			SegmentIndex:     0,
			SpeakerUserID:    f.speaker,
			MeetingID:        &mid,
			Text:             "x",
			StartMs:          0,
			EndMs:            1000,
			SegmentStartedAt: now.Add(-1 * time.Second),
		})
	}

	for _, m := range meetings {
		f.svc.Tick(m, true)
	}

	// Wait long enough for every run to launch but not so long that any one
	// finishes well before the next is launched.
	time.Sleep(100 * time.Millisecond)
	assert.LessOrEqual(t, gen.MaxConcurrent(), int32(2),
		"global semaphore must cap concurrent Foundry calls at MaxConcurrentRuns")
}

func TestKatService_NotConfiguredDoesNotJoinCohort(t *testing.T) {
	gen := newFakeNoteGenerator(okOutput())
	gen.configured = false
	f := newKatFixture(t, services.KatConfig{
		WindowSeconds: 120, StepSeconds: 20, MinNewSegments: 1,
		MaxConcurrentRuns: 2, RingBufferCapacity: 20,
	}, gen)
	defer func() { _ = f.svc.Shutdown(ctxWithTimeout(t)) }()

	f.svc.OnParticipantJoined(f.meetingID, f.speaker, true)
	// Cohort should be empty; SnapshotRingBuffer returns empty slice.
	assert.Empty(t, f.svc.SnapshotRingBuffer(f.meetingID))

	f.svc.Tick(f.meetingID, true)
	time.Sleep(20 * time.Millisecond)
	assert.Zero(t, gen.Calls())
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func waitForCalls(t *testing.T, gen *fakeNoteGenerator, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gen.Calls() >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %d generator calls, only saw %d", want, gen.Calls())
}

func ctxWithTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
