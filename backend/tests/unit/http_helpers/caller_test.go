package httphelpers_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	apperr "github.com/rekall/backend/pkg/errors"
)

func init() { gin.SetMode(gin.TestMode) }

// ─── CallerID ────────────────────────────────────────────────────────────────

func TestCallerID_Success(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	userID := uuid.New()
	claims := &infraauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
	}
	c.Set("auth_claims", claims)

	id, err := handlerhelpers.CallerID(c)
	require.NoError(t, err)
	assert.Equal(t, userID, id)
}

func TestCallerID_NoClaims_Unauthorized(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	// Context has no auth_claims set.

	_, err := handlerhelpers.CallerID(c)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestCallerID_InvalidSubjectUUID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	claims := &infraauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "not-a-uuid"},
	}
	c.Set("auth_claims", claims)

	_, err := handlerhelpers.CallerID(c)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}
