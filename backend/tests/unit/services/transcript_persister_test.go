package services_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── In-memory fake TranscriptRepository ─────────────────────────────────────
//
// Keeps the persister tests focused on the validation + auth logic. We use a
// fake (not a testify mock) here because we want to assert end-state — counters
// incremented exactly once, segments dedup'd by (session_id, segment_index) —
// not the call sequence.

type fakeTranscriptRepo struct {
	mu               sync.Mutex
	sessions         map[uuid.UUID]*entities.TranscriptSession
	segments         map[string]*entities.TranscriptSegment // key = session_id + ":" + segment_index
	stitchedCallText map[uuid.UUID]string
}

func newFakeTranscriptRepo() *fakeTranscriptRepo {
	return &fakeTranscriptRepo{
		sessions:         map[uuid.UUID]*entities.TranscriptSession{},
		segments:         map[string]*entities.TranscriptSegment{},
		stitchedCallText: map[uuid.UUID]string{},
	}
}

func (f *fakeTranscriptRepo) CreateSession(_ context.Context, s *entities.TranscriptSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.sessions[s.ID]; exists {
		return errors.New("duplicate session")
	}
	cp := *s
	f.sessions[s.ID] = &cp
	return nil
}

func (f *fakeTranscriptRepo) GetSession(_ context.Context, id uuid.UUID) (*entities.TranscriptSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok {
		return nil, apperr.NotFound("TranscriptSession", id.String())
	}
	cp := *s
	return &cp, nil
}

func (f *fakeTranscriptRepo) UpdateSessionStatus(_ context.Context, id uuid.UUID, status entities.TranscriptSessionStatus, errCode, errMsg *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok {
		return apperr.NotFound("TranscriptSession", id.String())
	}
	s.Status = status
	if status != entities.TranscriptSessionStatusActive {
		now := time.Now().UTC()
		s.EndedAt = &now
	}
	if status == entities.TranscriptSessionStatusErrored {
		s.ErrorCode = errCode
		s.ErrorMessage = errMsg
	}
	return nil
}

func (f *fakeTranscriptRepo) UpsertSegment(_ context.Context, seg *entities.TranscriptSegment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := segmentKey(seg.SessionID, seg.SegmentIndex)
	_, existed := f.segments[key]
	cp := *seg
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	f.segments[key] = &cp
	if !existed {
		if s, ok := f.sessions[seg.SessionID]; ok {
			s.FinalizedSegmentCount++
			s.AudioSecondsTotal += float64(seg.EndMs-seg.StartMs) / 1000.0
		}
	}
	return nil
}

func (f *fakeTranscriptRepo) ListSegmentsBySession(_ context.Context, id uuid.UUID) ([]*entities.TranscriptSegment, error) {
	return nil, nil
}
func (f *fakeTranscriptRepo) ListSegmentsByCall(_ context.Context, _ uuid.UUID, _, _ int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *fakeTranscriptRepo) ListSegmentsByMeeting(_ context.Context, _ uuid.UUID, _, _ int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *fakeTranscriptRepo) ListSpeakerUserIDsByMeeting(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (f *fakeTranscriptRepo) ListSegmentsByMeetingInRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*entities.TranscriptSegment, error) {
	return []*entities.TranscriptSegment{}, nil
}
func (f *fakeTranscriptRepo) ListSegmentsByCallInRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*entities.TranscriptSegment, error) {
	return []*entities.TranscriptSegment{}, nil
}
func (f *fakeTranscriptRepo) ListSessionsByCall(_ context.Context, _ uuid.UUID) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *fakeTranscriptRepo) ListSessionsByMeeting(_ context.Context, _ uuid.UUID) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *fakeTranscriptRepo) FindExpiredActive(_ context.Context, limit int) ([]*entities.TranscriptSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	var out []*entities.TranscriptSession
	for _, s := range f.sessions {
		if s.Status == entities.TranscriptSessionStatusActive && s.ExpiresAt.Before(now) {
			cp := *s
			out = append(out, &cp)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (f *fakeTranscriptRepo) StitchSession(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (f *fakeTranscriptRepo) StitchCall(_ context.Context, callID uuid.UUID) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.stitchedCallText[callID]; ok {
		return v, nil
	}
	return "stitched", nil
}
func (f *fakeTranscriptRepo) StitchMeeting(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

func segmentKey(sessionID uuid.UUID, idx int32) string {
	return fmt.Sprintf("%s:%d", sessionID, idx)
}

// ─── Test helpers ────────────────────────────────────────────────────────────

func transcriptPersisterDeps(t *testing.T, callID uuid.UUID, scope *uuid.UUID) (
	*services.TranscriptPersister, *fakeTranscriptRepo, *mockCallRepo, *mockMeetingRepo,
) {
	t.Helper()
	tr := newFakeTranscriptRepo()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)

	cr.On("GetByID", mock.Anything, callID).Return(&entities.Call{
		ID:      callID,
		ScopeID: scope,
	}, nil).Maybe()
	cr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Call")).Return(
		&entities.Call{ID: callID},
		nil,
	).Maybe()

	p := services.NewTranscriptPersister(tr, cr, mr, zap.NewNop())
	return p, tr, cr, mr
}

func sampleOpenInput(callID, speaker uuid.UUID) services.OpenSessionInput {
	return services.OpenSessionInput{
		SessionID:     uuid.New(),
		SpeakerUserID: speaker,
		CallID:        &callID,
		Engine: services.EngineSnapshot{
			Mode:    entities.TranscriptEngineModeOpenAI,
			Target:  "https://api.openai.com/v1/",
			ModelID: "whisper-1",
		},
		SampleRate:  16000,
		FrameFormat: "pcm_s16le_mono",
		ExpiresAt:   time.Now().Add(15 * time.Minute),
	}
}

func sampleFinal(sessionID, caller uuid.UUID, idx int32) services.RecordFinalInput {
	return services.RecordFinalInput{
		SessionID:    sessionID,
		CallerUserID: caller,
		SegmentIndex: idx,
		Text:         "hello world",
		StartMs:      idx * 1000,
		EndMs:        (idx + 1) * 1000,
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestTranscriptPersister_OpenSession_RejectsBothCallAndMeeting(t *testing.T) {
	p, _, _, _ := transcriptPersisterDeps(t, uuid.New(), nil)
	mid := uuid.New()
	cid := uuid.New()
	in := sampleOpenInput(cid, uuid.New())
	in.MeetingID = &mid // both set → invalid
	err := p.OpenSession(context.Background(), in)
	assert.ErrorIs(t, err, services.ErrTranscriptInvalidSegment)
}

func TestTranscriptPersister_OpenSession_RejectsNeitherCallNorMeeting(t *testing.T) {
	p, _, _, _ := transcriptPersisterDeps(t, uuid.New(), nil)
	in := sampleOpenInput(uuid.New(), uuid.New())
	in.CallID = nil
	in.MeetingID = nil
	err := p.OpenSession(context.Background(), in)
	assert.ErrorIs(t, err, services.ErrTranscriptInvalidSegment)
}

func TestTranscriptPersister_OpenSession_HappyPath(t *testing.T) {
	callID := uuid.New()
	p, repo, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, uuid.New())
	require.NoError(t, p.OpenSession(context.Background(), in))

	got, err := repo.GetSession(context.Background(), in.SessionID)
	require.NoError(t, err)
	assert.Equal(t, callID, *got.CallID)
	assert.Equal(t, entities.TranscriptSessionStatusActive, got.Status)
	assert.Equal(t, "whisper-1", got.ModelID)
}

func TestTranscriptPersister_RecordFinal_RejectsInvalidPayload(t *testing.T) {
	callID := uuid.New()
	speaker := uuid.New()
	p, _, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))

	cases := map[string]services.RecordFinalInput{
		"empty text":       {SessionID: in.SessionID, CallerUserID: speaker, SegmentIndex: 0, StartMs: 0, EndMs: 1000},
		"end before start": {SessionID: in.SessionID, CallerUserID: speaker, SegmentIndex: 0, Text: "ok", StartMs: 1000, EndMs: 500},
		"negative index":   {SessionID: in.SessionID, CallerUserID: speaker, SegmentIndex: -1, Text: "ok", StartMs: 0, EndMs: 1000},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			err := p.RecordFinal(context.Background(), payload)
			assert.ErrorIs(t, err, services.ErrTranscriptInvalidSegment)
		})
	}
}

func TestTranscriptPersister_RecordFinal_RejectsUnknownSession(t *testing.T) {
	p, _, _, _ := transcriptPersisterDeps(t, uuid.New(), nil)
	err := p.RecordFinal(context.Background(), sampleFinal(uuid.New(), uuid.New(), 0))
	assert.ErrorIs(t, err, services.ErrTranscriptSessionNotFound)
}

func TestTranscriptPersister_RecordFinal_RejectsWrongOwner(t *testing.T) {
	callID := uuid.New()
	owner := uuid.New()
	p, _, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, owner)
	require.NoError(t, p.OpenSession(context.Background(), in))

	intruder := uuid.New()
	err := p.RecordFinal(context.Background(), sampleFinal(in.SessionID, intruder, 0))
	assert.ErrorIs(t, err, services.ErrTranscriptSessionNotOwned)
}

func TestTranscriptPersister_RecordFinal_RejectsClosedSession(t *testing.T) {
	callID := uuid.New()
	speaker := uuid.New()
	p, repo, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))
	require.NoError(t, repo.UpdateSessionStatus(context.Background(), in.SessionID, entities.TranscriptSessionStatusEnded, nil, nil))

	err := p.RecordFinal(context.Background(), sampleFinal(in.SessionID, speaker, 0))
	assert.ErrorIs(t, err, services.ErrTranscriptSessionClosed)
}

func TestTranscriptPersister_RecordFinal_HappyPathIncrementsCounters(t *testing.T) {
	callID := uuid.New()
	speaker := uuid.New()
	p, repo, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))

	for i := int32(0); i < 3; i++ {
		require.NoError(t, p.RecordFinal(context.Background(), sampleFinal(in.SessionID, speaker, i)))
	}

	got, err := repo.GetSession(context.Background(), in.SessionID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, got.FinalizedSegmentCount)
	assert.InDelta(t, 3.0, got.AudioSecondsTotal, 0.001)
}

func TestTranscriptPersister_RecordFinal_DuplicateDoesNotDoubleCount(t *testing.T) {
	callID := uuid.New()
	speaker := uuid.New()
	p, repo, _, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))

	final := sampleFinal(in.SessionID, speaker, 0)
	require.NoError(t, p.RecordFinal(context.Background(), final))
	require.NoError(t, p.RecordFinal(context.Background(), final))

	got, err := repo.GetSession(context.Background(), in.SessionID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, got.FinalizedSegmentCount, "duplicate (session_id, segment_index) must not double-count")
}

func TestTranscriptPersister_CloseSession_RewritesCallTranscript(t *testing.T) {
	callID := uuid.New()
	speaker := uuid.New()
	p, repo, cr, _ := transcriptPersisterDeps(t, callID, nil)
	repo.stitchedCallText[callID] = "hello world from segments"

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))

	require.NoError(t, p.CloseSession(context.Background(), services.CloseSessionInput{
		SessionID:    in.SessionID,
		CallerUserID: speaker,
		Status:       entities.TranscriptSessionStatusEnded,
		StitchInto:   &callID,
	}))

	got, err := repo.GetSession(context.Background(), in.SessionID)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusEnded, got.Status)

	cr.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(c *entities.Call) bool {
		return c.Transcript != nil && *c.Transcript == "hello world from segments"
	}))
}

func TestTranscriptPersister_CloseSession_StitchInto_DiffersFromSessionCall_NoStitch(t *testing.T) {
	callID := uuid.New()
	otherID := uuid.New()
	speaker := uuid.New()
	p, _, cr, _ := transcriptPersisterDeps(t, callID, nil)

	in := sampleOpenInput(callID, speaker)
	require.NoError(t, p.OpenSession(context.Background(), in))

	require.NoError(t, p.CloseSession(context.Background(), services.CloseSessionInput{
		SessionID:    in.SessionID,
		CallerUserID: speaker,
		Status:       entities.TranscriptSessionStatusEnded,
		StitchInto:   &otherID, // does NOT match the session's call_id
	}))

	cr.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// Compile-time check that the fake satisfies the port.
var _ ports.TranscriptRepository = (*fakeTranscriptRepo)(nil)
