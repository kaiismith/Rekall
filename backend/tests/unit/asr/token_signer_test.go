package asr_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rekall/backend/internal/infrastructure/asr"
)

const testSecret = "0123456789abcdef0123456789abcdef0123456789ab"

func TestNewTokenSigner_RejectsShortSecret(t *testing.T) {
	if _, err := asr.NewTokenSigner([]byte("short"), "iss", "aud"); err == nil {
		t.Fatalf("expected error for short secret")
	}
}

func TestTokenSigner_RoundTrip(t *testing.T) {
	signer, err := asr.NewTokenSigner([]byte(testSecret), "rekall-backend", "rekall-asr")
	if err != nil {
		t.Fatalf("signer init: %v", err)
	}
	user, call, sid := uuid.New(), uuid.New(), uuid.New()
	expires := time.Now().Add(3 * time.Minute).UTC()

	token, err := signer.Sign(user, call, sid, "small.en", expires)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT with 3 parts, got %d", len(parts))
	}

	parsed, err := jwt.ParseWithClaims(token, &asr.SessionTokenClaims{},
		func(t *jwt.Token) (interface{}, error) { return []byte(testSecret), nil },
		jwt.WithIssuer("rekall-backend"),
		jwt.WithAudience("rekall-asr"),
		jwt.WithExpirationRequired())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims, ok := parsed.Claims.(*asr.SessionTokenClaims)
	if !ok {
		t.Fatalf("claims type assertion failed")
	}
	if claims.SID != sid.String() {
		t.Fatalf("sid: want %s, got %s", sid, claims.SID)
	}
	if claims.CID != call.String() {
		t.Fatalf("cid: want %s, got %s", call, claims.CID)
	}
	if claims.Subject != user.String() {
		t.Fatalf("sub: want %s, got %s", user, claims.Subject)
	}
	if claims.Scope != "asr:stream" {
		t.Fatalf("scope: want asr:stream, got %s", claims.Scope)
	}
	if claims.ID == "" {
		t.Fatalf("jti must be set")
	}
}

func TestTokenSigner_RejectsEmptyIssuerOrAudience(t *testing.T) {
	if _, err := asr.NewTokenSigner([]byte(testSecret), "", "aud"); err == nil {
		t.Fatalf("expected error for empty issuer")
	}
	if _, err := asr.NewTokenSigner([]byte(testSecret), "iss", ""); err == nil {
		t.Fatalf("expected error for empty audience")
	}
}
