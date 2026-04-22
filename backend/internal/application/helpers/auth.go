package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	infraemail "github.com/rekall/backend/internal/infrastructure/email"
	apperr "github.com/rekall/backend/pkg/errors"
)

// NewRefreshToken generates a raw refresh token and its hashed entity record.
func NewRefreshToken(userID uuid.UUID, ttl time.Duration) (string, *entities.RefreshToken, error) {
	raw, err := infraauth.GenerateRawToken()
	if err != nil {
		return "", nil, apperr.Internal("failed to generate session token")
	}
	tok := &entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: infraauth.HashToken(raw),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	return raw, tok, nil
}

// SendVerificationEmail creates a verification token and dispatches the email.
func SendVerificationEmail(
	ctx context.Context,
	user *entities.User,
	tokenRepo ports.TokenRepository,
	ttl time.Duration,
	appBaseURL string,
	mailer ports.EmailSender,
) error {
	raw, err := infraauth.GenerateRawToken()
	if err != nil {
		return err
	}
	tok := &entities.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: infraauth.HashToken(raw),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := tokenRepo.CreateVerificationToken(ctx, tok); err != nil {
		return err
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", appBaseURL, raw)
	return mailer.Send(ctx, ports.EmailMessage{
		To:      user.Email,
		Subject: infraemail.VerificationEmailSubject(),
		Body:    infraemail.VerificationEmailBody(user.FullName, verifyURL),
	})
}
