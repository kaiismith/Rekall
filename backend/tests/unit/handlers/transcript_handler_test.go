package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	apperr "github.com/rekall/backend/pkg/errors"
)

// Reuses the in-memory fakeTranscriptRepoH from asr_handler_segments_test.go,
// which already implements ports.TranscriptRepository.

func newCallTranscriptRouter(h *handlers.TranscriptHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/api/v1/calls/:id/transcript", h.GetCallTranscript)
	return r
}

func newMeetingTranscriptRouter(h *handlers.TranscriptHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/api/v1/meetings/:code/transcript", h.GetMeetingTranscript)
	return r
}

func TestGetCallTranscript_200(t *testing.T) {
	tr := newFakeTranscriptRepoH()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	owner := uuid.New()
	callID := uuid.New()
	cr.On("GetByID", mock.Anything, callID).Return(&entities.Call{
		ID: callID, UserID: owner,
	}, nil)

	// Seed one session + one segment.
	sid := openTestSession(t, tr, callID, owner)
	require.NoError(t, tr.UpsertSegment(t.Context(), &entities.TranscriptSegment{
		SessionID:        sid,
		SegmentIndex:     0,
		SpeakerUserID:    owner,
		CallID:           &callID,
		Text:             "hello world",
		StartMs:          0,
		EndMs:            1500,
		EngineMode:       entities.TranscriptEngineModeOpenAI,
		ModelID:          "whisper-1",
		SegmentStartedAt: time.Now().UTC(),
	}))

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := newCallTranscriptRouter(h, owner)

	w := doRequest(router, http.MethodGet, "/api/v1/calls/"+callID.String()+"/transcript", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var env dto.CallTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Success)
	require.NotNil(t, env.Data.Session)
	assert.Equal(t, sid, env.Data.Session.ID)
	assert.Equal(t, "whisper-1", env.Data.Session.ModelID)
	require.Len(t, env.Data.Segments, 0,
		"fakeTranscriptRepoH.ListSegmentsByCall returns nil/0; full coverage lives in the integration test")
}

func TestGetCallTranscript_403_NotOwner(t *testing.T) {
	tr := newFakeTranscriptRepoH()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	owner := uuid.New()
	intruder := uuid.New()
	callID := uuid.New()
	cr.On("GetByID", mock.Anything, callID).Return(&entities.Call{
		ID: callID, UserID: owner,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := newCallTranscriptRouter(h, intruder)

	w := doRequest(router, http.MethodGet, "/api/v1/calls/"+callID.String()+"/transcript", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetCallTranscript_404_CallMissing(t *testing.T) {
	tr := newFakeTranscriptRepoH()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	caller := uuid.New()
	missing := uuid.New()
	cr.On("GetByID", mock.Anything, missing).Return(nil, apperr.NotFound("Call", missing.String()))

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := newCallTranscriptRouter(h, caller)

	w := doRequest(router, http.MethodGet, "/api/v1/calls/"+missing.String()+"/transcript", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCallTranscript_400_InvalidID(t *testing.T) {
	h := handlers.NewTranscriptHandler(
		newFakeTranscriptRepoH(), new(mockCallRepo),
		new(mockMeetingRepo), new(mockParticipantRepo), zap.NewNop(),
	)
	router := newCallTranscriptRouter(h, uuid.New())

	w := doRequest(router, http.MethodGet, "/api/v1/calls/not-a-uuid/transcript", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMeetingTranscript_200_Host(t *testing.T) {
	tr := newFakeTranscriptRepoH()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	host := uuid.New()
	meetingID := uuid.New()
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := newMeetingTranscriptRouter(h, host)

	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var env dto.MeetingTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Success)
	assert.NotNil(t, env.Data.Sessions)
	assert.NotNil(t, env.Data.Segments)
}

func TestGetMeetingTranscript_403_NotParticipant(t *testing.T) {
	tr := newFakeTranscriptRepoH()
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	host := uuid.New()
	intruder := uuid.New()
	meetingID := uuid.New()
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, meetingID, intruder).Return(
		nil, apperr.NotFound("MeetingParticipant", intruder.String()))

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := newMeetingTranscriptRouter(h, intruder)

	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
