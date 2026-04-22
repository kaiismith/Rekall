package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ─── Router factory ───────────────────────────────────────────────────────────

func newAuthRouter(h *handlers.AuthHandler, authed bool, userID uuid.UUID) *gin.Engine {
	r := gin.New()

	// Public routes
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.GET("/auth/verify", h.VerifyEmail)
	r.POST("/auth/verify/resend", h.ResendVerification)
	r.POST("/auth/password/forgot", h.ForgotPassword)
	r.POST("/auth/password/reset", h.ResetPassword)

	// Protected
	me := r.Group("/auth")
	if authed {
		me.Use(injectClaims(userID, "member"))
	} else {
		me.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	}
	me.GET("/me", h.Me)

	return r
}

func newAuthService(userRepo *mockUserRepo, tokenRepo *mockTokenRepo, mailer *mockMailer) *services.AuthService {
	return services.NewAuthService(
		userRepo, tokenRepo, mailer,
		testSecret, testIssuer, "http://localhost:5173",
		15*time.Minute, 7*24*time.Hour, time.Hour, 24*time.Hour,
		zap.NewNop(),
	)
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestRegisterHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, apperr.NotFound("User", "alice@example.com"))
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", Role: "member",
	}, nil)
	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.AnythingOfType("*entities.EmailVerificationToken")).Return(nil)
	mailer.On("Send", mock.Anything, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	w := doRequest(r, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"email": "alice@example.com", "password": "Password1!", "full_name": "Alice",
	}))

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
}

func TestRegisterHandler_InvalidBody(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	// Missing password
	w := doRequest(r, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"email": "alice@example.com",
	}))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	existing := &entities.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(existing, nil)

	w := doRequest(r, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"email": "alice@example.com", "password": "Password1!", "full_name": "Alice",
	}))

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", Role: "member",
		PasswordHash: hashedPassword(t, "Password1!"), EmailVerified: true,
	}, nil)
	tokenRepo.On("CreateRefreshToken", mock.Anything, mock.AnythingOfType("*entities.RefreshToken")).Return(nil)

	w := doRequest(r, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"email": "alice@example.com", "password": "Password1!",
	}))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])
	// Refresh cookie should be set
	assert.NotEmpty(t, w.Result().Cookies())
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(&entities.User{
		ID: uuid.New(), Email: "alice@example.com",
		PasswordHash: hashedPassword(t, "correct-horse"), EmailVerified: true,
	}, nil)

	w := doRequest(r, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"email": "alice@example.com", "password": "wrong",
	}))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_UnverifiedEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(&entities.User{
		ID: uuid.New(), Email: "alice@example.com",
		PasswordHash: hashedPassword(t, "Password1!"), EmailVerified: false,
	}, nil)

	w := doRequest(r, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"email": "alice@example.com", "password": "Password1!",
	}))

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLoginHandler_InvalidBody(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	w := doRequest(r, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{"email": "only"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogoutHandler_AlwaysNoContent(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	// No cookie present — should still 204
	w := doRequest(r, http.MethodPost, "/auth/logout", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func TestVerifyEmailHandler_MissingToken(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	w := doRequest(r, http.MethodGet, "/auth/verify", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyEmailHandler_InvalidToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	tokenRepo.On("GetVerificationToken", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("token", "x"))

	w := doRequest(r, http.MethodGet, "/auth/verify?token=invalid-raw-token", nil)
	// Service always returns 400 for any token lookup failure (anti-enumeration)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

func TestForgotPasswordHandler_AlwaysOK(t *testing.T) {
	// Anti-enumeration: responds 200 regardless of whether email exists
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, apperr.NotFound("User", "unknown@example.com"))

	w := doRequest(r, http.MethodPost, "/auth/password/forgot", jsonBody(t, map[string]string{
		"email": "unknown@example.com",
	}))
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── Me ───────────────────────────────────────────────────────────────────────

func TestMeHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	userID := uuid.New()
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, true, userID)

	userRepo.On("GetByID", mock.Anything, userID).Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", Role: "member",
	}, nil)

	w := doRequest(r, http.MethodGet, "/auth/me", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "alice@example.com", data["email"])
}

func TestMeHandler_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	userID := uuid.New()
	r := newAuthRouter(h, true, userID)

	userRepo.On("GetByID", mock.Anything, userID).Return(nil, apperr.NotFound("User", userID.String()))

	w := doRequest(r, http.MethodGet, "/auth/me", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMeHandler_Unauthenticated(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	// authed=false → real JWT middleware
	r := newAuthRouter(h, false, uuid.Nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

func TestRefreshHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userID := uuid.New()
	rawRefresh := "raw-refresh-token-xyz"
	hash := infraauth.HashToken(rawRefresh)

	tokenRepo.On("GetRefreshToken", mock.Anything, hash).Return(&entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", Role: "member",
	}, nil)
	tokenRepo.On("RevokeRefreshToken", mock.Anything, hash).Return(nil)
	tokenRepo.On("CreateRefreshToken", mock.Anything, mock.AnythingOfType("*entities.RefreshToken")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: rawRefresh})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])
}

func TestRefreshHandler_NoCookie(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	// Token not found in repo
	tokenRepo.On("GetRefreshToken", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("token", "x"))

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── ResendVerification ──────────────────────────────────────────────────────

func TestResendVerificationHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", EmailVerified: false,
	}, nil)
	tokenRepo.On("InvalidatePendingVerificationTokens", mock.Anything, userID).Return(nil)
	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.AnythingOfType("*entities.EmailVerificationToken")).Return(nil)
	mailer.On("Send", mock.Anything, mock.Anything).Return(nil)

	w := doRequest(r, http.MethodPost, "/auth/verify/resend", jsonBody(t, map[string]string{
		"email": "alice@example.com",
	}))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResendVerificationHandler_InvalidBody(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	w := doRequest(r, http.MethodPost, "/auth/verify/resend", jsonBody(t, map[string]string{}))

	// Empty email fails DTO validation (422)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResendVerificationHandler_AlreadyVerified(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(&entities.User{
		ID: uuid.New(), Email: "alice@example.com", EmailVerified: true,
	}, nil)

	w := doRequest(r, http.MethodPost, "/auth/verify/resend", jsonBody(t, map[string]string{
		"email": "alice@example.com",
	}))

	// Should return 400 for already verified email
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

func TestResetPasswordHandler_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	userID := uuid.New()
	rawToken := "raw-reset-token"
	hash := infraauth.HashToken(rawToken)

	tokenRepo.On("GetPasswordResetToken", mock.Anything, hash).Return(&entities.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil)
	userRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("MarkPasswordResetTokenUsed", mock.Anything, hash).Return(nil)
	tokenRepo.On("RevokeAllRefreshTokens", mock.Anything, userID).Return(nil)

	w := doRequest(r, http.MethodPost, "/auth/password/reset", jsonBody(t, map[string]string{
		"token":    rawToken,
		"password": "NewPassword1",
	}))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResetPasswordHandler_InvalidBody(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	// Missing password
	w := doRequest(r, http.MethodPost, "/auth/password/reset", jsonBody(t, map[string]string{
		"token": "abc",
	}))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResetPasswordHandler_InvalidToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	h := handlers.NewAuthHandler(svc, 7*24*time.Hour, zap.NewNop())
	r := newAuthRouter(h, false, uuid.Nil)

	tokenRepo.On("GetPasswordResetToken", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("token", "x"))

	w := doRequest(r, http.MethodPost, "/auth/password/reset", jsonBody(t, map[string]string{
		"token":    "invalid-token",
		"password": "NewPassword1",
	}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
