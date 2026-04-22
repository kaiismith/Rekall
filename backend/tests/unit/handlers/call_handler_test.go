package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Router factory ───────────────────────────────────────────────────────────

func newCallRouter(h *handlers.CallHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/calls", h.List)
	r.GET("/calls/:id", h.Get)
	r.POST("/calls", h.Create)
	r.PATCH("/calls/:id", h.Update)
	r.DELETE("/calls/:id", h.Delete)
	return r
}

func newCallService(repo *mockCallRepo) *services.CallService {
	return services.NewCallService(repo, zap.NewNop())
}

func sampleCall(userID uuid.UUID) *entities.Call {
	return &entities.Call{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "Q1 Sales Call",
		Status:    "pending",
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestCallListHandler_Success(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	call := sampleCall(callerID)
	repo.On("List", mock.Anything, ports.ListCallsFilter{}, 1, 20).Return([]*entities.Call{call}, 1, nil)

	w := doRequest(r, http.MethodGet, "/calls", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	meta := body["meta"].(map[string]interface{})
	assert.Equal(t, float64(1), meta["total"])
}

func TestCallListHandler_FilterByUserID(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	filter := ports.ListCallsFilter{UserID: &callerID}
	repo.On("List", mock.Anything, filter, 1, 20).Return([]*entities.Call{}, 0, nil)

	w := doRequest(r, http.MethodGet, "/calls?user_id="+callerID.String(), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}

func TestCallListHandler_InvalidUserID(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/calls?user_id=not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCallListHandler_FilterByStatus(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	status := "processing"
	filter := ports.ListCallsFilter{Status: &status}
	repo.On("List", mock.Anything, filter, 1, 20).Return([]*entities.Call{}, 0, nil)

	w := doRequest(r, http.MethodGet, "/calls?status=processing", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}

// ─── Get ──────────────────────────────────────────────────────────────────────

func TestCallGetHandler_Success(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	call := sampleCall(callerID)
	repo.On("GetByID", mock.Anything, call.ID).Return(call, nil)

	w := doRequest(r, http.MethodGet, "/calls/"+call.ID.String(), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
}

func TestCallGetHandler_InvalidUUID(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/calls/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCallGetHandler_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	callID := uuid.New()
	repo.On("GetByID", mock.Anything, callID).Return(nil, apperr.NotFound("Call", callID.String()))

	w := doRequest(r, http.MethodGet, "/calls/"+callID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestCallCreateHandler_Success(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	call := sampleCall(callerID)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Call")).Return(call, nil)

	w := doRequest(r, http.MethodPost, "/calls", jsonBody(t, map[string]interface{}{
		"user_id": callerID.String(), "title": "Q1 Sales Call",
	}))

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCallCreateHandler_MissingTitle(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	callerID := uuid.New()
	// Title is required by the service, but the DTO binding requires user_id and title
	w := doRequest(r, http.MethodPost, "/calls", jsonBody(t, map[string]interface{}{
		"user_id": callerID.String(),
	}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCallCreateHandler_MissingUserID(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	w := doRequest(r, http.MethodPost, "/calls", jsonBody(t, map[string]string{"title": "Test"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestCallUpdateHandler_Success(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	call := sampleCall(callerID)
	updated := *call
	updated.Title = "Updated Title"
	repo.On("GetByID", mock.Anything, call.ID).Return(call, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Call")).Return(&updated, nil)

	w := doRequest(r, http.MethodPatch, "/calls/"+call.ID.String(), jsonBody(t, map[string]string{"title": "Updated Title"}))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
}

func TestCallUpdateHandler_InvalidUUID(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	w := doRequest(r, http.MethodPatch, "/calls/bad-uuid", jsonBody(t, map[string]string{"title": "x"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCallUpdateHandler_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	callID := uuid.New()
	repo.On("GetByID", mock.Anything, callID).Return(nil, apperr.NotFound("Call", callID.String()))

	w := doRequest(r, http.MethodPatch, "/calls/"+callID.String(), jsonBody(t, map[string]string{"title": "x"}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestCallDeleteHandler_Success(t *testing.T) {
	repo := new(mockCallRepo)
	callerID := uuid.New()
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, callerID)

	call := sampleCall(callerID)
	repo.On("GetByID", mock.Anything, call.ID).Return(call, nil)
	repo.On("SoftDelete", mock.Anything, call.ID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/calls/"+call.ID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCallDeleteHandler_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	h := handlers.NewCallHandler(newCallService(repo), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	callID := uuid.New()
	repo.On("GetByID", mock.Anything, callID).Return(nil, apperr.NotFound("Call", callID.String()))

	w := doRequest(r, http.MethodDelete, "/calls/"+callID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallDeleteHandler_InvalidUUID(t *testing.T) {
	h := handlers.NewCallHandler(newCallService(new(mockCallRepo)), zap.NewNop())
	r := newCallRouter(h, uuid.New())

	w := doRequest(r, http.MethodDelete, "/calls/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
