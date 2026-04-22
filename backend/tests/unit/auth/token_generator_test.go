package auth_test

import (
	"encoding/hex"
	"testing"

	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── GenerateRawToken ─────────────────────────────────────────────────────────

func TestGenerateRawToken_ReturnsNonEmptyString(t *testing.T) {
	token, err := infraauth.GenerateRawToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateRawToken_IsValidHex(t *testing.T) {
	token, err := infraauth.GenerateRawToken()
	require.NoError(t, err)

	decoded, err := hex.DecodeString(token)
	require.NoError(t, err, "token should be valid hex")
	assert.Len(t, decoded, 32, "should decode to 32 bytes")
}

func TestGenerateRawToken_IsUnique(t *testing.T) {
	tok1, err1 := infraauth.GenerateRawToken()
	tok2, err2 := infraauth.GenerateRawToken()

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, tok1, tok2)
}

// ─── HashToken ────────────────────────────────────────────────────────────────

func TestHashToken_ReturnsNonEmptyString(t *testing.T) {
	hash := infraauth.HashToken("some-raw-token")

	assert.NotEmpty(t, hash)
}

func TestHashToken_IsValidHex(t *testing.T) {
	hash := infraauth.HashToken("some-raw-token")

	decoded, err := hex.DecodeString(hash)
	require.NoError(t, err, "hash should be valid hex")
	assert.Len(t, decoded, 32, "SHA-256 produces 32 bytes")
}

func TestHashToken_IsDeterministic(t *testing.T) {
	raw := "same-input"

	assert.Equal(t, infraauth.HashToken(raw), infraauth.HashToken(raw))
}

func TestHashToken_DifferentInputsDifferentHashes(t *testing.T) {
	assert.NotEqual(t, infraauth.HashToken("token-a"), infraauth.HashToken("token-b"))
}

func TestHashToken_RawTokenDoesNotAppearInHash(t *testing.T) {
	raw := "super-secret-raw-value"
	hash := infraauth.HashToken(raw)

	assert.NotContains(t, hash, raw)
}
