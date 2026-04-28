package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// TokenRepository abstracts all single-use and session token persistence.
type TokenRepository interface {
	// ── Refresh tokens ─────────────────────────────────────────────────────────

	CreateRefreshToken(ctx context.Context, token *entities.RefreshToken) error
	GetRefreshToken(ctx context.Context, hash string) (*entities.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, hash string) error
	// RevokeAllRefreshTokens revokes every active refresh token for a user.
	// Called after a successful password reset to force re-login on all devices.
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error

	// ── Email verification tokens ───────────────────────────────────────────────

	CreateVerificationToken(ctx context.Context, token *entities.EmailVerificationToken) error
	GetVerificationToken(ctx context.Context, hash string) (*entities.EmailVerificationToken, error)
	MarkVerificationTokenUsed(ctx context.Context, hash string) error
	// InvalidatePendingVerificationTokens deletes all unused verification tokens for a user.
	// Called before issuing a fresh token so only one active token exists per user.
	InvalidatePendingVerificationTokens(ctx context.Context, userID uuid.UUID) error

	// ── Password reset tokens ───────────────────────────────────────────────────

	CreatePasswordResetToken(ctx context.Context, token *entities.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, hash string) (*entities.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, hash string) error
	// InvalidatePendingPasswordResetTokens deletes all unused reset tokens for a user.
	InvalidatePendingPasswordResetTokens(ctx context.Context, userID uuid.UUID) error
}
