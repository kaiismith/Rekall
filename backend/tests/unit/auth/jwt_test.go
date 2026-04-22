package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret = "test-secret-key-32-bytes-long!!"
	testIssuer = "rekall-test"
)

func newUser() *entities.User {
	return &entities.User{
		ID:    uuid.New(),
		Email: "alice@example.com",
		Role:  "member",
	}
}

// ─── SignAccessToken ──────────────────────────────────────────────────────────

func TestSignAccessToken_ReturnsNonEmptyToken(t *testing.T) {
	token, err := infraauth.SignAccessToken(newUser(), testSecret, testIssuer, 15*time.Minute)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestSignAccessToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	u1, u2 := newUser(), newUser()
	tok1, _ := infraauth.SignAccessToken(u1, testSecret, testIssuer, 15*time.Minute)
	tok2, _ := infraauth.SignAccessToken(u2, testSecret, testIssuer, 15*time.Minute)

	assert.NotEqual(t, tok1, tok2)
}

// ─── ParseAccessToken ─────────────────────────────────────────────────────────

func TestParseAccessToken_ValidToken(t *testing.T) {
	user := newUser()
	token, err := infraauth.SignAccessToken(user, testSecret, testIssuer, 15*time.Minute)
	require.NoError(t, err)

	claims, err := infraauth.ParseAccessToken(token, testSecret, testIssuer)

	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.Subject)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)
	assert.Equal(t, testIssuer, claims.Issuer)
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	token, err := infraauth.SignAccessToken(newUser(), testSecret, testIssuer, 15*time.Minute)
	require.NoError(t, err)

	_, err = infraauth.ParseAccessToken(token, "wrong-secret", testIssuer)

	assert.Error(t, err)
}

func TestParseAccessToken_WrongIssuer(t *testing.T) {
	token, err := infraauth.SignAccessToken(newUser(), testSecret, testIssuer, 15*time.Minute)
	require.NoError(t, err)

	_, err = infraauth.ParseAccessToken(token, testSecret, "other-issuer")

	assert.Error(t, err)
}

func TestParseAccessToken_ExpiredToken(t *testing.T) {
	token, err := infraauth.SignAccessToken(newUser(), testSecret, testIssuer, -1*time.Second)
	require.NoError(t, err)

	_, err = infraauth.ParseAccessToken(token, testSecret, testIssuer)

	assert.Error(t, err)
}

func TestParseAccessToken_MalformedToken(t *testing.T) {
	_, err := infraauth.ParseAccessToken("not.a.valid.jwt", testSecret, testIssuer)

	assert.Error(t, err)
}

func TestParseAccessToken_EmptyToken(t *testing.T) {
	_, err := infraauth.ParseAccessToken("", testSecret, testIssuer)

	assert.Error(t, err)
}

// ─── SubjectAsUUID ────────────────────────────────────────────────────────────

func TestSubjectAsUUID_ValidSubject(t *testing.T) {
	user := newUser()
	token, err := infraauth.SignAccessToken(user, testSecret, testIssuer, 15*time.Minute)
	require.NoError(t, err)

	claims, err := infraauth.ParseAccessToken(token, testSecret, testIssuer)
	require.NoError(t, err)

	id, err := claims.SubjectAsUUID()

	require.NoError(t, err)
	assert.Equal(t, user.ID, id)
}

func TestSubjectAsUUID_InvalidSubject(t *testing.T) {
	claims := &infraauth.Claims{}
	claims.Subject = "not-a-uuid"

	_, err := claims.SubjectAsUUID()

	assert.Error(t, err)
}
