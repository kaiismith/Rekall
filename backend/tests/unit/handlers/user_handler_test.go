package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Router factory ───────────────────────────────────────────────────────────

func newUserRouter(h *handlers.UserHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "admin"))
	r.GET("/users", h.List)
	r.GET("/users/:id", h.Get)
	r.POST("/users", h.Create)
	r.DELETE("/users/:id", h.Delete)
	return r
}

func newUserService(repo *mockUserRepo) *services.UserService {
	return services.NewUserService(repo, zap.NewNop())
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestUserListHandler_Success(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	repo.On("List", mock.Anything, 1, 20).Return([]*entities.User{
		{ID: uuid.New(), Email: "a@example.com", Role: "member"},
		{ID: uuid.New(), Email: "b@example.com", Role: "admin"},
	}, 2, nil)

	w := doRequest(r, http.MethodGet, "/users", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	meta := body["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["total"])
}

func TestUserListHandler_ServiceError(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	repo.On("List", mock.Anything, 1, 20).Return([]*entities.User(nil), 0, assert.AnError)

	w := doRequest(r, http.MethodGet, "/users", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserListHandler_CustomPage(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	repo.On("List", mock.Anything, 2, 10).Return([]*entities.User{}, 0, nil)

	w := doRequest(r, http.MethodGet, "/users?page=2&per_page=10", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}

// ─── Get ──────────────────────────────────────────────────────────────────────

func TestUserGetHandler_Success(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	userID := uuid.New()
	repo.On("GetByID", mock.Anything, userID).Return(&entities.User{
		ID: userID, Email: "alice@example.com", Role: "member",
	}, nil)

	w := doRequest(r, http.MethodGet, "/users/"+userID.String(), nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserGetHandler_InvalidUUID(t *testing.T) {
	h := handlers.NewUserHandler(newUserService(new(mockUserRepo)), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/users/not-a-uuid", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserGetHandler_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	userID := uuid.New()
	repo.On("GetByID", mock.Anything, userID).Return(nil, apperr.NotFound("User", userID.String()))

	w := doRequest(r, http.MethodGet, "/users/"+userID.String(), nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestUserCreateHandler_Success(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	userID := uuid.New()
	repo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, apperr.NotFound("User", "new@example.com"))
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(&entities.User{
		ID: userID, Email: "new@example.com", FullName: "New User", Role: "member",
	}, nil)

	w := doRequest(r, http.MethodPost, "/users", jsonBody(t, map[string]string{
		"email": "new@example.com", "full_name": "New User",
	}))

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUserCreateHandler_InvalidBody(t *testing.T) {
	h := handlers.NewUserHandler(newUserService(new(mockUserRepo)), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	// Missing full_name
	w := doRequest(r, http.MethodPost, "/users", jsonBody(t, map[string]string{"email": "x@x.com"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUserCreateHandler_DuplicateEmail(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	existing := &entities.User{ID: uuid.New(), Email: "dup@example.com"}
	repo.On("GetByEmail", mock.Anything, "dup@example.com").Return(existing, nil)

	w := doRequest(r, http.MethodPost, "/users", jsonBody(t, map[string]string{
		"email": "dup@example.com", "full_name": "Dup User",
	}))

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestUserDeleteHandler_Success(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	userID := uuid.New()
	repo.On("GetByID", mock.Anything, userID).Return(&entities.User{
		ID: userID, Email: "alice@example.com", Role: "member",
	}, nil)
	repo.On("SoftDelete", mock.Anything, userID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/users/"+userID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUserDeleteHandler_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	h := handlers.NewUserHandler(newUserService(repo), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	userID := uuid.New()
	repo.On("GetByID", mock.Anything, userID).Return(nil, apperr.NotFound("User", userID.String()))

	w := doRequest(r, http.MethodDelete, "/users/"+userID.String(), nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserDeleteHandler_InvalidUUID(t *testing.T) {
	h := handlers.NewUserHandler(newUserService(new(mockUserRepo)), zap.NewNop())
	r := newUserRouter(h, uuid.New())

	w := doRequest(r, http.MethodDelete, "/users/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
