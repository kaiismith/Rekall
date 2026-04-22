package entities

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored, hashed refresh token used to obtain new access tokens.
// The raw token is sent to the client via cookie; only the SHA-256 hash is persisted.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	TokenHash string     `gorm:"column:token_hash;uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// IsExpired reports whether the token has passed its expiry time.
func (t *RefreshToken) IsExpired() bool { return time.Now().UTC().After(t.ExpiresAt) }

// IsRevoked reports whether the token has been explicitly revoked.
func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

// IsValid reports whether the token can be used to obtain a new access token.
func (t *RefreshToken) IsValid() bool { return !t.IsExpired() && !t.IsRevoked() }

// EmailVerificationToken is a single-use token emailed to confirm a new account address.
type EmailVerificationToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	TokenHash string     `gorm:"column:token_hash;uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
}

func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }

func (t *EmailVerificationToken) IsExpired() bool { return time.Now().UTC().After(t.ExpiresAt) }
func (t *EmailVerificationToken) IsUsed() bool    { return t.UsedAt != nil }
func (t *EmailVerificationToken) IsValid() bool   { return !t.IsExpired() && !t.IsUsed() }

// PasswordResetToken is a single-use token emailed to authorise a password change.
type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	TokenHash string     `gorm:"column:token_hash;uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

func (t *PasswordResetToken) IsExpired() bool { return time.Now().UTC().After(t.ExpiresAt) }
func (t *PasswordResetToken) IsUsed() bool    { return t.UsedAt != nil }
func (t *PasswordResetToken) IsValid() bool   { return !t.IsExpired() && !t.IsUsed() }
