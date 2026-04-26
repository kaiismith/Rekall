package asr

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SessionTokenClaims is the payload of the Session_Token issued by the Go
// backend after registering the session via gRPC StartSession. The shape MUST
// match what the C++ JWTValidator expects (see asr/include/.../jwt_validator).
type SessionTokenClaims struct {
	jwt.RegisteredClaims
	SID   string `json:"sid"`
	CID   string `json:"cid"`
	Model string `json:"model"`
	Scope string `json:"scope"`
}

// TokenSigner mints HS256 JWTs for the ASR WebSocket data plane.
type TokenSigner struct {
	secret   []byte
	issuer   string
	audience string
}

// NewTokenSigner returns a signer; secret MUST be at least 32 bytes.
func NewTokenSigner(secret []byte, issuer, audience string) (*TokenSigner, error) {
	if len(secret) < 32 {
		return nil, errors.New("ASR token secret must be at least 32 bytes")
	}
	if issuer == "" || audience == "" {
		return nil, errors.New("ASR token issuer and audience must be non-empty")
	}
	return &TokenSigner{secret: secret, issuer: issuer, audience: audience}, nil
}

// Sign builds a token bound to the supplied session. `expiresAt` SHOULD come
// from the gRPC StartSessionResponse so the issuer and verifier agree on the
// effective lifetime (the ASR server clamps the requested TTL).
func (s *TokenSigner) Sign(
	userID, callID, sessionID uuid.UUID,
	modelID string,
	expiresAt time.Time,
) (string, error) {
	now := time.Now().UTC()
	claims := SessionTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
		SID:   sessionID.String(),
		CID:   callID.String(),
		Model: modelID,
		Scope: "asr:stream",
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = "asr-v1"
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("asr token sign: %w", err)
	}
	return signed, nil
}
