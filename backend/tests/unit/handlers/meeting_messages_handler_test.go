package handlers_test

import (
	"context"
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

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── Mock MeetingMessageRepository ───────────────────────────────────────────

type mockMessageRepo struct{ mock.Mock }

func (m *mockMessageRepo) Create(ctx context.Context, msg *entities.MeetingMessage) error {
	return m.Called(ctx, msg).Error(0)
}

func (m *mockMessageRepo) ListByMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	before *time.Time,
	limit int,
) ([]*entities.MeetingMessage, bool, error) {
	args := m.Called(ctx, meetingID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]*entities.MeetingMessage), args.Bool(1), args.Error(2)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newMeetingHandlerWithChat(
	svc *services.MeetingService,
	chat *services.ChatMessageService,
) *handlers.MeetingHandler {
	manager := wsHub.NewHubManager(nil, nil, zap.NewNop())
	return handlers.NewMeetingHandler(svc, chat, nil, manager, nil, "http://rekall.test", zap.NewNop())
}

func newMessagesRouter(h *handlers.MeetingHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	authed := r.Group("/")
	authed.Use(injectClaims(callerID, "member"))
	authed.GET("/meetings/:code/messages", h.ListMessages)
	return r
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestListMessages_HostSuccess(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	meetingSvc := newMeetingService(mr, pr)
	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(meetingSvc, chatSvc)
	r := newMessagesRouter(h, hostID)

	m := activeMeeting(hostID)
	mr.On("GetByCode", mock.Anything, m.Code).Return(m, nil)
	expected := []*entities.MeetingMessage{
		{ID: uuid.New(), MeetingID: m.ID, UserID: hostID, Body: "hi", SentAt: time.Now()},
	}
	mrepo.On("ListByMeeting", mock.Anything, m.ID, (*time.Time)(nil), 50).
		Return(expected, false, nil)

	w := doRequest(r, http.MethodGet, "/meetings/"+m.Code+"/messages", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	msgs := data["messages"].([]interface{})
	assert.Len(t, msgs, 1)
	assert.False(t, data["has_more"].(bool))
}

func TestListMessages_NonParticipantForbidden(t *testing.T) {
	hostID := uuid.New()
	outsiderID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(newMeetingService(mr, pr), chatSvc)
	r := newMessagesRouter(h, outsiderID)

	m := activeMeeting(hostID)
	mr.On("GetByCode", mock.Anything, m.Code).Return(m, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, m.ID, outsiderID).
		Return(nil, apperr.NotFound("MeetingParticipant", m.ID.String()))

	w := doRequest(r, http.MethodGet, "/meetings/"+m.Code+"/messages", nil)

	require.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errBody := resp["error"].(map[string]interface{})
	assert.Equal(t, "NOT_A_PARTICIPANT", errBody["code"])
}

func TestListMessages_MeetingNotFound(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(newMeetingService(mr, pr), chatSvc)
	r := newMessagesRouter(h, uuid.New())

	mr.On("GetByCode", mock.Anything, "nope-none-nada").
		Return(nil, apperr.NotFound("Meeting", "nope-none-nada"))

	w := doRequest(r, http.MethodGet, "/meetings/nope-none-nada/messages", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListMessages_InvalidBefore_BadRequest(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(newMeetingService(mr, pr), chatSvc)
	r := newMessagesRouter(h, hostID)

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/messages?before=not-a-timestamp", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListMessages_InvalidLimit_BadRequest(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(newMeetingService(mr, pr), chatSvc)
	r := newMessagesRouter(h, hostID)

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/messages?limit=-1", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListMessages_LimitClampedTo100(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mrepo := new(mockMessageRepo)

	chatSvc := services.NewChatMessageService(mr, pr, mrepo, zap.NewNop())
	h := newMeetingHandlerWithChat(newMeetingService(mr, pr), chatSvc)
	r := newMessagesRouter(h, hostID)

	m := activeMeeting(hostID)
	mr.On("GetByCode", mock.Anything, m.Code).Return(m, nil)
	// Handler clamps limit=1000 → 100 before calling the service; the repo
	// receives the clamped value.
	mrepo.On("ListByMeeting", mock.Anything, m.ID, (*time.Time)(nil), 100).
		Return([]*entities.MeetingMessage{}, false, nil)

	w := doRequest(r, http.MethodGet, "/meetings/"+m.Code+"/messages?limit=1000", nil)
	require.Equal(t, http.StatusOK, w.Code)
	mrepo.AssertExpectations(t)
}

func TestListMessages_NoChatServiceConfigured_ReturnsInternal(t *testing.T) {
	// The handler is constructed with chatService=nil (via the existing
	// newMeetingHandler). Requests must return 500 rather than panic.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMessagesRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/messages", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
