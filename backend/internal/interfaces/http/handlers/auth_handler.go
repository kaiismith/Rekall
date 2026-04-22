package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/internal/application/services"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	httputils "github.com/rekall/backend/internal/interfaces/http/utils"
	"github.com/rekall/backend/pkg/constants"
	apperr "github.com/rekall/backend/pkg/errors"
	"go.uber.org/zap"
)

// AuthHandler handles all authentication and account lifecycle endpoints.
type AuthHandler struct {
	service    *services.AuthService
	refreshTTL time.Duration
	logger     *zap.Logger
}

// NewAuthHandler creates an AuthHandler with its required dependencies.
func NewAuthHandler(service *services.AuthService, refreshTTL time.Duration, log *zap.Logger) *AuthHandler {
	return &AuthHandler{service: service, refreshTTL: refreshTTL, logger: log}
}

// Register handles POST /api/v1/auth/register.
//
// @Summary      Register a new user
// @Description  Creates a new user account with email and password. The user receives a verification email and **cannot sign in until their email is confirmed**.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RegisterRequest       true  "Registration payload"
// @Success      201   {object}  dto.UserResponseEnvelope  "Account created — check email for a verification link"
// @Failure      409   {object}  dto.ErrorResponse         "Email already registered (EMAIL_ALREADY_REGISTERED)"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error — see details for field-level errors"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(httputils.ToUserResponse(user)))
}

// Login handles POST /api/v1/auth/login.
//
// @Summary      Sign in
// @Description  Authenticates a user with email and password. On success returns a short-lived JWT **access_token** (15 min) and sets an `HttpOnly Secure SameSite=Strict` **refresh_token** cookie (7 days).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest           true  "Login credentials"
// @Success      200   {object}  dto.LoginResponseEnvelope  "Authenticated successfully"
// @Failure      401   {object}  dto.ErrorResponse          "Invalid credentials (INVALID_CREDENTIALS)"
// @Failure      403   {object}  dto.ErrorResponse          "Email address not yet verified (EMAIL_NOT_VERIFIED)"
// @Failure      422   {object}  dto.ErrorResponse          "Validation error"
// @Failure      500   {object}  dto.ErrorResponse          "Internal server error"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	user, accessToken, rawRefresh, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	handlerhelpers.SetRefreshCookie(c, rawRefresh, h.refreshTTL)
	c.JSON(http.StatusOK, dto.OK(dto.LoginResponse{
		AccessToken: accessToken,
		User:        httputils.ToUserResponse(user),
	}))
}

// Refresh handles POST /api/v1/auth/refresh.
//
// @Summary      Refresh access token
// @Description  Exchanges the `refresh_token` cookie for a new JWT access token. The refresh token is **rotated** — the old one is revoked and a new cookie is issued atomically.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  dto.AccessTokenResponse  "New access token issued"
// @Failure      401  {object}  dto.ErrorResponse        "Refresh token invalid, expired, or revoked (INVALID_REFRESH_TOKEN)"
// @Failure      500  {object}  dto.ErrorResponse        "Internal server error"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	rawRefresh, err := c.Cookie(constants.CookieRefreshToken)
	if err != nil || rawRefresh == "" {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("no refresh token"))
		return
	}

	accessToken, newRawRefresh, err := h.service.RefreshTokens(c.Request.Context(), rawRefresh)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	handlerhelpers.SetRefreshCookie(c, newRawRefresh, h.refreshTTL)
	c.JSON(http.StatusOK, dto.OK(gin.H{"access_token": accessToken}))
}

// Logout handles POST /api/v1/auth/logout.
//
// @Summary      Sign out
// @Description  Revokes the refresh token stored in the `HttpOnly` cookie and clears the cookie. Access tokens are stateless and expire naturally — no server-side invalidation is needed.
// @Tags         Auth
// @Produce      json
// @Success      204  "Signed out — refresh token revoked and cookie cleared"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	rawRefresh, _ := c.Cookie(constants.CookieRefreshToken)
	_ = h.service.Logout(c.Request.Context(), rawRefresh)
	handlerhelpers.ClearRefreshCookie(c)
	c.JSON(http.StatusNoContent, nil)
}

// VerifyEmail handles GET /api/v1/auth/verify?token=<raw>.
//
// @Summary      Verify email address
// @Description  Marks the user's email as verified using the single-use token delivered in the verification email. The token is valid for 24 hours.
// @Tags         Auth
// @Produce      json
// @Param        token  query     string             true  "Raw verification token from the email link"
// @Success      200    {object}  dto.MessageResponse  "Email verified successfully"
// @Failure      400    {object}  dto.ErrorResponse    "Token missing, invalid, expired, or already used (INVALID_OR_EXPIRED_TOKEN)"
// @Failure      500    {object}  dto.ErrorResponse    "Internal server error"
// @Router       /api/v1/auth/verify [get]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("missing token"))
		return
	}

	if err := h.service.VerifyEmail(c.Request.Context(), token); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(gin.H{"message": "email verified successfully"}))
}

// ResendVerification handles POST /api/v1/auth/verify/resend.
//
// @Summary      Resend verification email
// @Description  Generates a new verification token, invalidates any prior pending token, and resends the verification email to the given address.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ResendVerificationRequest  true  "Email address to resend verification to"
// @Success      200   {object}  dto.MessageResponse            "Verification email sent (if the address is registered and unverified)"
// @Failure      400   {object}  dto.ErrorResponse              "Email is already verified (EMAIL_ALREADY_VERIFIED)"
// @Failure      422   {object}  dto.ErrorResponse              "Validation error"
// @Failure      500   {object}  dto.ErrorResponse              "Internal server error"
// @Router       /api/v1/auth/verify/resend [post]
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	if err := h.service.ResendVerification(c.Request.Context(), req.Email); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(gin.H{"message": "if that address is registered and unverified, a new email has been sent"}))
}

// ForgotPassword handles POST /api/v1/auth/password/forgot.
//
// @Summary      Request password reset
// @Description  Sends a password reset link to the given email address. **Always returns 200 regardless of whether the email is registered** to prevent user enumeration. The reset link is valid for 1 hour.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ForgotPasswordRequest  true  "Email address"
// @Success      200   {object}  dto.MessageResponse        "Reset email dispatched (if account exists)"
// @Failure      422   {object}  dto.ErrorResponse          "Validation error"
// @Failure      500   {object}  dto.ErrorResponse          "Internal server error"
// @Router       /api/v1/auth/password/forgot [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	_ = h.service.ForgotPassword(c.Request.Context(), req.Email)
	c.JSON(http.StatusOK, dto.OK(gin.H{"message": "if an account with that email exists, a reset link has been sent"}))
}

// ResetPassword handles POST /api/v1/auth/password/reset.
//
// @Summary      Reset password
// @Description  Sets a new password using a valid, unexpired password reset token. On success all active refresh tokens for the user are revoked, forcing re-login on all devices. Password must be 8+ characters with at least one letter and one digit.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ResetPasswordRequest  true  "Reset token and new password"
// @Success      200   {object}  dto.MessageResponse       "Password updated — all sessions terminated"
// @Failure      400   {object}  dto.ErrorResponse         "Token invalid, expired, or already used (INVALID_OR_EXPIRED_TOKEN)"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error — password complexity requirements not met"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/auth/password/reset [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(gin.H{"message": "password updated successfully"}))
}

// Me handles GET /api/v1/auth/me (requires Authenticate middleware).
//
// @Summary      Get current user
// @Description  Returns the full profile of the authenticated user. Requires a valid JWT access token in the `Authorization: Bearer <token>` header.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.UserResponseEnvelope  "Authenticated user profile"
// @Failure      401  {object}  dto.ErrorResponse         "Missing or invalid token (MISSING_TOKEN / INVALID_TOKEN)"
// @Failure      403  {object}  dto.ErrorResponse         "Email address not verified (EMAIL_NOT_VERIFIED)"
// @Failure      500  {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	id, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToUserResponse(user)))
}
