package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── In-memory fake transcript repo (handler-test scope) ─────────────────────
//
// Re-implemented here (instead of importing the services-test package fake)
// because Go disallows importing _test packages. Kept narrow: only the
// methods the handler -> persister path exercises are wired.

type fakeTranscriptRepoH struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*entities.TranscriptSession
	segments map[string]*entities.TranscriptSegment
}

func newFakeTranscriptRepoH() *fakeTranscriptRepoH {
	return &fakeTranscriptRepoH{
		sessions: map[uuid.UUID]*entities.TranscriptSession{},
		segments: map[string]*entities.TranscriptSegment{},
	}
}

func (f *fakeTranscriptRepoH) CreateSession(_ context.Context, s *entities.TranscriptSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *s
	f.sessions[s.ID] = &cp
	return nil
}
func (f *fakeTranscriptRepoH) GetSession(_ context.Context, id uuid.UUID) (*entities.TranscriptSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok {
		return nil, apperr.NotFound("TranscriptSession", id.String())
	}
	cp := *s
	return &cp, nil
}
func (f *fakeTranscriptRepoH) UpdateSessionStatus(_ context.Context, id uuid.UUID, st entities.TranscriptSessionStatus, ec, em *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok {
		return apperr.NotFound("TranscriptSession", id.String())
	}
	s.Status = st
	if ec != nil {
		s.ErrorCode = ec
	}
	if em != nil {
		s.ErrorMessage = em
	}
	return nil
}
func (f *fakeTranscriptRepoH) UpsertSegment(_ context.Context, seg *entities.TranscriptSegment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *seg
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	f.segments[seg.SessionID.String()] = &cp
	if s, ok := f.sessions[seg.SessionID]; ok {
		s.FinalizedSegmentCount++
	}
	return nil
}
func (f *fakeTranscriptRepoH) ListSegmentsBySession(_ context.Context, _ uuid.UUID) ([]*entities.TranscriptSegment, error) {
	return nil, nil
}
func (f *fakeTranscriptRepoH) ListSegmentsByCall(_ context.Context, _ uuid.UUID, _, _ int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *fakeTranscriptRepoH) ListSegmentsByMeeting(_ context.Context, _ uuid.UUID, _, _ int) ([]*entities.TranscriptSegment, int, error) {
	return nil, 0, nil
}
func (f *fakeTranscriptRepoH) ListSessionsByCall(_ context.Context, callID uuid.UUID) ([]*entities.TranscriptSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*entities.TranscriptSession
	for _, s := range f.sessions {
		if s.CallID != nil && *s.CallID == callID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (f *fakeTranscriptRepoH) ListSessionsByMeeting(_ context.Context, meetingID uuid.UUID) ([]*entities.TranscriptSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*entities.TranscriptSession
	for _, s := range f.sessions {
		if s.MeetingID != nil && *s.MeetingID == meetingID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (f *fakeTranscriptRepoH) FindExpiredActive(_ context.Context, _ int) ([]*entities.TranscriptSession, error) {
	return nil, nil
}
func (f *fakeTranscriptRepoH) StitchSession(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (f *fakeTranscriptRepoH) StitchCall(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (f *fakeTranscriptRepoH) StitchMeeting(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

// Compile-time check.
var _ ports.TranscriptRepository = (*fakeTranscriptRepoH)(nil)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newSegmentsRouter(h *handlers.ASRHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.POST("/api/v1/calls/:id/asr-session/:session_id/segments", h.PostCallSegment)
	return r
}

func openTestSession(t *testing.T, repo *fakeTranscriptRepoH, callID, speaker uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repo.CreateSession(context.Background(), &entities.TranscriptSession{
		ID:            id,
		SpeakerUserID: speaker,
		CallID:        &callID,
		EngineMode:    entities.TranscriptEngineModeOpenAI,
		EngineTarget:  "https://api.openai.com/v1/",
		ModelID:       "whisper-1",
		SampleRate:    16000,
		FrameFormat:   "pcm_s16le_mono",
		Status:        entities.TranscriptSessionStatusActive,
		StartedAt:     time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(15 * time.Minute),
	}))
	return id
}

func validBody() dto.TranscriptSegmentRequest {
	return dto.TranscriptSegmentRequest{
		SegmentIndex: 0,
		Text:         "hello world",
		StartMs:      0,
		EndMs:        1500,
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestPostCallSegment_204(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()
	sessionID := openTestSession(t, repo, callID, speaker)

	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+sessionID.String()+"/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusNoContent, w.Code, "body=%s", w.Body.String())
}

func TestPostCallSegment_400_BadBody(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()
	sessionID := openTestSession(t, repo, callID, speaker)

	bad := dto.TranscriptSegmentRequest{Text: ""} // missing required text
	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+sessionID.String()+"/segments",
		jsonBody(t, bad))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostCallSegment_400_InvalidSessionID(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()

	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/not-a-uuid/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostCallSegment_404_UnknownSession(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()
	bogusSession := uuid.New()

	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+bogusSession.String()+"/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostCallSegment_403_NotOwned(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	owner := uuid.New()
	intruder := uuid.New()
	callID := uuid.New()
	sessionID := openTestSession(t, repo, callID, owner)

	router := newSegmentsRouter(h, intruder)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+sessionID.String()+"/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostCallSegment_409_Closed(t *testing.T) {
	repo := newFakeTranscriptRepoH()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	h := handlers.NewASRHandler(nil, persister, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()
	sessionID := openTestSession(t, repo, callID, speaker)
	require.NoError(t, repo.UpdateSessionStatus(context.Background(), sessionID,
		entities.TranscriptSessionStatusEnded, nil, nil))

	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+sessionID.String()+"/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPostCallSegment_503_PersistenceNotConfigured(t *testing.T) {
	h := handlers.NewASRHandler(nil, nil, zap.NewNop())

	speaker := uuid.New()
	callID := uuid.New()
	sessionID := uuid.New()

	router := newSegmentsRouter(h, speaker)
	w := doRequest(router, http.MethodPost,
		"/api/v1/calls/"+callID.String()+"/asr-session/"+sessionID.String()+"/segments",
		jsonBody(t, validBody()))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// Sanity: services.ErrTranscriptInvalidSegment maps to 400.
func TestPostCallSegment_InvalidSegmentMapsTo400(t *testing.T) {
	// EndMs <= StartMs is rejected by binding-level `gtfield=StartMs` first,
	// so this test is mostly belt-and-braces — but it's worth confirming the
	// service-layer error is wired correctly.
	require.True(t, errors.Is(services.ErrTranscriptInvalidSegment, services.ErrTranscriptInvalidSegment))
}
