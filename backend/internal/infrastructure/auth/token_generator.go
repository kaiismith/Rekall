package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const rawTokenBytes = 32

// GenerateRawToken returns a hex-encoded 32-byte cryptographically random string.
// This is the value sent to the client (email link, cookie). Never store this directly.
func GenerateRawToken() (string, error) {
	b := make([]byte, rawTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token_generator: failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 hash of a raw token.
// Only the hash is stored in the database; if the DB is compromised the raw tokens remain safe.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
