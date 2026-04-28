package services

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/rekall/backend/internal/application/helpers"
	apputils "github.com/rekall/backend/internal/application/utils"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	infraemail "github.com/rekall/backend/internal/infrastructure/email"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

const bcryptCost = 12

var emailRegexp = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// AuthService orchestrates all account-lifecycle operations:
// registration, email verification, login, token refresh, logout, and password reset.
type AuthService struct {
	userRepo   ports.UserRepository
	tokenRepo  ports.TokenRepository
	mailer     ports.EmailSender
	jwtSecret  string
	jwtIssuer  string
	appBaseURL string
	accessTTL  time.Duration
	refreshTTL time.Duration
	resetTTL   time.Duration
	verifyTTL  time.Duration
	logger     *zap.Logger
}

// NewAuthService creates an AuthService with all required dependencies.
func NewAuthService(
	userRepo ports.UserRepository,
	tokenRepo ports.TokenRepository,
	mailer ports.EmailSender,
	jwtSecret, jwtIssuer, appBaseURL string,
	accessTTL, refreshTTL, resetTTL, verifyTTL time.Duration,
	log *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		mailer:     mailer,
		jwtSecret:  jwtSecret,
		jwtIssuer:  jwtIssuer,
		appBaseURL: appBaseURL,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		resetTTL:   resetTTL,
		verifyTTL:  verifyTTL,
		logger:     applogger.WithComponent(log, "auth_service"),
	}
}

// Register validates input, creates a user, and dispatches a verification email.
func (s *AuthService) Register(ctx context.Context, email, password, fullName string) (*entities.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	fullName = strings.TrimSpace(fullName)

	if !emailRegexp.MatchString(email) {
		return nil, apperr.Unprocessable("invalid email address", nil)
	}
	if fullName == "" {
		return nil, apperr.Unprocessable("full_name is required", nil)
	}
	if err := apputils.ValidatePassword(password); err != nil {
		return nil, err
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !apperr.IsNotFound(err) {
		return nil, apperr.Internal("failed to check email availability")
	}
	if existing != nil {
		return nil, apperr.Conflict("an account with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, apperr.Internal("failed to process registration")
	}

	now := time.Now().UTC()
	user := &entities.User{
		ID:            uuid.New(),
		Email:         email,
		FullName:      fullName,
		Role:          "member",
		PasswordHash:  string(hash),
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, apperr.Internal("failed to create account")
	}

	if err := helpers.SendVerificationEmail(ctx, created, s.tokenRepo, s.verifyTTL, s.appBaseURL, s.mailer); err != nil {
		// Non-fatal: user exists but email may not arrive; they can resend.
		catalog.VerificationEmailFailed.Error(s.logger,
			zap.String("user_id", created.ID.String()),
			zap.Error(err),
		)
	}

	catalog.UserRegistered.Info(s.logger,
		zap.String("user_id", created.ID.String()),
		zap.String("email", created.Email),
	)
	return created, nil
}

// VerifyEmail confirms an email address using a raw token from the verification link.
func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	hash := infraauth.HashToken(rawToken)
	tok, err := s.tokenRepo.GetVerificationToken(ctx, hash)
	if err != nil || !tok.IsValid() {
		catalog.VerificationTokenInvalid.Warn(s.logger)
		return apperr.BadRequest("verification link is invalid or has expired")
	}

	if err := s.userRepo.SetEmailVerified(ctx, tok.UserID, true); err != nil {
		return apperr.Internal("failed to verify email")
	}
	if err := s.tokenRepo.MarkVerificationTokenUsed(ctx, hash); err != nil {
		catalog.VerificationTokenMarkFailed.Error(s.logger, zap.Error(err))
	}

	catalog.EmailVerified.Info(s.logger, zap.String("user_id", tok.UserID.String()))
	return nil
}

// ResendVerification dispatches a fresh verification email if the user is not yet verified.
func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if apperr.IsNotFound(err) {
		return nil // don't reveal whether the email exists
	}
	if err != nil {
		return apperr.Internal("failed to resend verification")
	}
	if user.EmailVerified {
		catalog.EmailAlreadyVerified.Warn(s.logger, zap.String("user_id", user.ID.String()))
		return apperr.BadRequest("this email address is already verified")
	}

	_ = s.tokenRepo.InvalidatePendingVerificationTokens(ctx, user.ID)
	return helpers.SendVerificationEmail(ctx, user, s.tokenRepo, s.verifyTTL, s.appBaseURL, s.mailer)
}

// Login checks credentials and returns an access token + raw refresh token on success.
func (s *AuthService) Login(ctx context.Context, email, password string) (*entities.User, string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if apperr.IsNotFound(err) || err == nil && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		catalog.LoginFailedBadCredentials.Warn(s.logger, zap.String("email", email))
		return nil, "", "", apperr.Unauthorized("invalid email or password")
	}
	if err != nil {
		return nil, "", "", apperr.Internal("failed to authenticate")
	}
	if !user.EmailVerified {
		catalog.LoginFailedUnverified.Warn(s.logger,
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
		)
		return nil, "", "", apperr.Forbidden("please verify your email address before signing in")
	}

	accessToken, err := infraauth.SignAccessToken(user, s.jwtSecret, s.jwtIssuer, s.accessTTL)
	if err != nil {
		return nil, "", "", apperr.Internal("failed to issue access token")
	}

	rawRefresh, refreshToken, err := helpers.NewRefreshToken(user.ID, s.refreshTTL)
	if err != nil {
		return nil, "", "", err
	}
	if err := s.tokenRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, "", "", apperr.Internal("failed to create session")
	}

	catalog.LoginSuccess.Info(s.logger,
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)
	return user, accessToken, rawRefresh, nil
}

// RefreshTokens validates the raw refresh token, rotates it, and returns a new access token.
func (s *AuthService) RefreshTokens(ctx context.Context, rawRefresh string) (string, string, error) {
	hash := infraauth.HashToken(rawRefresh)
	tok, err := s.tokenRepo.GetRefreshToken(ctx, hash)
	if err != nil || !tok.IsValid() {
		return "", "", apperr.Unauthorized("refresh token is invalid or expired")
	}

	user, err := s.userRepo.GetByID(ctx, tok.UserID)
	if err != nil {
		return "", "", apperr.Unauthorized("user not found")
	}

	accessToken, err := infraauth.SignAccessToken(user, s.jwtSecret, s.jwtIssuer, s.accessTTL)
	if err != nil {
		return "", "", apperr.Internal("failed to issue access token")
	}

	rawNew, newTok, err := helpers.NewRefreshToken(user.ID, s.refreshTTL)
	if err != nil {
		return "", "", err
	}

	// Rotate atomically: revoke old, create new
	if err := s.tokenRepo.RevokeRefreshToken(ctx, hash); err != nil {
		return "", "", apperr.Internal("failed to rotate session")
	}
	if err := s.tokenRepo.CreateRefreshToken(ctx, newTok); err != nil {
		return "", "", apperr.Internal("failed to create session")
	}

	catalog.TokenRefreshed.Debug(s.logger, zap.String("user_id", user.ID.String()))
	return accessToken, rawNew, nil
}

// Logout revokes the given refresh token. Idempotent — no error if token not found.
func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	hash := infraauth.HashToken(rawRefresh)
	_ = s.tokenRepo.RevokeRefreshToken(ctx, hash) // best-effort
	catalog.Logout.Info(s.logger)
	return nil
}

// ForgotPassword sends a password reset email. Always succeeds to prevent user enumeration.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil // silently succeed
	}

	_ = s.tokenRepo.InvalidatePendingPasswordResetTokens(ctx, user.ID)

	raw, err := infraauth.GenerateRawToken()
	if err != nil {
		return nil
	}
	tok := &entities.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: infraauth.HashToken(raw),
		ExpiresAt: time.Now().UTC().Add(s.resetTTL),
	}
	if err := s.tokenRepo.CreatePasswordResetToken(ctx, tok); err != nil {
		return nil
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appBaseURL, raw)
	_ = s.mailer.Send(ctx, ports.EmailMessage{
		To:      user.Email,
		Subject: infraemail.PasswordResetEmailSubject(),
		Body:    infraemail.PasswordResetEmailBody(user.FullName, resetURL),
	})

	catalog.PasswordResetRequested.Info(s.logger, zap.String("user_id", user.ID.String()))
	return nil
}

// ResetPassword validates the token, updates the password, and revokes all sessions.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if err := apputils.ValidatePassword(newPassword); err != nil {
		return err
	}

	hash := infraauth.HashToken(rawToken)
	tok, err := s.tokenRepo.GetPasswordResetToken(ctx, hash)
	if err != nil || !tok.IsValid() {
		catalog.PasswordResetTokenInvalid.Warn(s.logger)
		return apperr.BadRequest("reset link is invalid or has expired")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return apperr.Internal("failed to process password reset")
	}

	if err := s.userRepo.UpdatePassword(ctx, tok.UserID, string(newHash)); err != nil {
		return apperr.Internal("failed to update password")
	}
	_ = s.tokenRepo.MarkPasswordResetTokenUsed(ctx, hash)
	_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, tok.UserID)

	catalog.PasswordResetCompleted.Info(s.logger, zap.String("user_id", tok.UserID.String()))
	return nil
}

// UpdateMe applies an in-place update to the calling user's profile. Only
// the full_name field is editable via this path; email/role/verified state
// require admin-managed flows.
func (s *AuthService) UpdateMe(ctx context.Context, userID uuid.UUID, fullName string) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.FullName = fullName
	return s.userRepo.Update(ctx, user)
}

// ChangePassword verifies the caller's current password, installs the new
// one, revokes every other refresh token for the user, and issues a fresh
// access+refresh pair for THIS session so the tab that made the change stays
// signed in. Failing the bcrypt comparison returns INVALID_CURRENT_PASSWORD.
func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID uuid.UUID,
	currentPassword, newPassword string,
) (accessToken string, rawRefresh string, err error) {
	if err := apputils.ValidatePassword(newPassword); err != nil {
		return "", "", err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return "", "", &apperr.AppError{
			Status:  http.StatusBadRequest,
			Code:    "INVALID_CURRENT_PASSWORD",
			Message: "current password is incorrect",
		}
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return "", "", apperr.Internal("failed to process password change")
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return "", "", apperr.Internal("failed to update password")
	}

	// Revoke every existing refresh token, then mint a fresh one for this
	// session so the tab that initiated the change is not kicked out.
	if err := s.tokenRepo.RevokeAllRefreshTokens(ctx, userID); err != nil {
		s.logger.Warn("password change: failed to revoke other sessions", zap.Error(err))
	}

	accessToken, err = infraauth.SignAccessToken(user, s.jwtSecret, s.jwtIssuer, s.accessTTL)
	if err != nil {
		return "", "", apperr.Internal("failed to issue access token")
	}

	rawRefresh, refreshToken, err := helpers.NewRefreshToken(userID, s.refreshTTL)
	if err != nil {
		return "", "", err
	}
	if err := s.tokenRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return "", "", apperr.Internal("failed to create session")
	}

	s.logger.Info("password changed",
		zap.String("event_code", "PASSWORD_CHANGED"),
		zap.String("user_id", userID.String()),
	)
	return accessToken, rawRefresh, nil
}

// GetUser retrieves a user by ID — used by the auth middleware after JWT parsing.
func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
