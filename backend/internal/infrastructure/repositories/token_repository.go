package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"gorm.io/gorm"
)

// TokenRepository implements ports.TokenRepository using GORM.
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a TokenRepository backed by the given GORM connection.
func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// ── Refresh tokens ─────────────────────────────────────────────────────────────

func (r *TokenRepository) CreateRefreshToken(ctx context.Context, token *entities.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) GetRefreshToken(ctx context.Context, hash string) (*entities.RefreshToken, error) {
	var t entities.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("RefreshToken", hash)
	}
	return &t, err
}

func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, hash string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.RefreshToken{}).
		Where("token_hash = ?", hash).
		Update("revoked_at", now).Error
}

func (r *TokenRepository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// ── Email verification tokens ──────────────────────────────────────────────────

func (r *TokenRepository) CreateVerificationToken(ctx context.Context, token *entities.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) GetVerificationToken(ctx context.Context, hash string) (*entities.EmailVerificationToken, error) {
	var t entities.EmailVerificationToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("EmailVerificationToken", hash)
	}
	return &t, err
}

func (r *TokenRepository) MarkVerificationTokenUsed(ctx context.Context, hash string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.EmailVerificationToken{}).
		Where("token_hash = ?", hash).
		Update("used_at", now).Error
}

func (r *TokenRepository) InvalidatePendingVerificationTokens(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Delete(&entities.EmailVerificationToken{}).Error
}

// ── Password reset tokens ──────────────────────────────────────────────────────

func (r *TokenRepository) CreatePasswordResetToken(ctx context.Context, token *entities.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) GetPasswordResetToken(ctx context.Context, hash string) (*entities.PasswordResetToken, error) {
	var t entities.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("PasswordResetToken", hash)
	}
	return &t, err
}

func (r *TokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, hash string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.PasswordResetToken{}).
		Where("token_hash = ?", hash).
		Update("used_at", now).Error
}

func (r *TokenRepository) InvalidatePendingPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Delete(&entities.PasswordResetToken{}).Error
}
