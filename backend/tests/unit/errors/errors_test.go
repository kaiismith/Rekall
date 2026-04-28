package errors_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── Constructors ─────────────────────────────────────────────────────────────

func TestNotFound(t *testing.T) {
	err := apperr.NotFound("user", "abc-123")

	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Contains(t, err.Message, "user")
	assert.Contains(t, err.Message, "abc-123")
}

func TestBadRequest(t *testing.T) {
	err := apperr.BadRequest("field is required")

	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "BAD_REQUEST", err.Code)
	assert.Equal(t, "field is required", err.Message)
}

func TestUnprocessable(t *testing.T) {
	details := map[string]string{"email": "invalid format"}
	err := apperr.Unprocessable("validation failed", details)

	assert.Equal(t, http.StatusUnprocessableEntity, err.Status)
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, "validation failed", err.Message)
	assert.Equal(t, details, err.Details)
}

func TestUnprocessable_NilDetails(t *testing.T) {
	err := apperr.Unprocessable("bad input", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, err.Status)
	assert.Nil(t, err.Details)
}

func TestUnauthorized(t *testing.T) {
	err := apperr.Unauthorized("authentication required")

	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "UNAUTHORIZED", err.Code)
	assert.Equal(t, "authentication required", err.Message)
}

func TestForbidden(t *testing.T) {
	err := apperr.Forbidden("insufficient permissions")

	assert.Equal(t, http.StatusForbidden, err.Status)
	assert.Equal(t, "FORBIDDEN", err.Code)
	assert.Equal(t, "insufficient permissions", err.Message)
}

func TestConflict(t *testing.T) {
	err := apperr.Conflict("email already in use")

	assert.Equal(t, http.StatusConflict, err.Status)
	assert.Equal(t, "CONFLICT", err.Code)
	assert.Equal(t, "email already in use", err.Message)
}

func TestInternal(t *testing.T) {
	err := apperr.Internal("something went wrong")

	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "INTERNAL_ERROR", err.Code)
	assert.Equal(t, "something went wrong", err.Message)
}

// ─── Error() string ───────────────────────────────────────────────────────────

func TestAppError_ErrorString(t *testing.T) {
	err := apperr.BadRequest("missing field")

	assert.Contains(t, err.Error(), "BAD_REQUEST")
	assert.Contains(t, err.Error(), "missing field")
}

// ─── AsAppError ───────────────────────────────────────────────────────────────

func TestAsAppError_WithAppError(t *testing.T) {
	orig := apperr.NotFound("org", "xyz")

	appErr, ok := apperr.AsAppError(orig)

	require.True(t, ok)
	assert.Equal(t, orig, appErr)
}

func TestAsAppError_WithWrappedAppError(t *testing.T) {
	orig := apperr.Unauthorized("token expired")
	wrapped := errors.Join(errors.New("outer"), orig)

	appErr, ok := apperr.AsAppError(wrapped)

	require.True(t, ok)
	assert.Equal(t, orig, appErr)
}

func TestAsAppError_WithPlainError(t *testing.T) {
	_, ok := apperr.AsAppError(errors.New("plain error"))

	assert.False(t, ok)
}

func TestAsAppError_WithNil(t *testing.T) {
	_, ok := apperr.AsAppError(nil)

	assert.False(t, ok)
}

// ─── IsNotFound ───────────────────────────────────────────────────────────────

func TestIsNotFound_True(t *testing.T) {
	assert.True(t, apperr.IsNotFound(apperr.NotFound("call", "123")))
}

func TestIsNotFound_FalseForOtherStatus(t *testing.T) {
	assert.False(t, apperr.IsNotFound(apperr.BadRequest("bad")))
	assert.False(t, apperr.IsNotFound(apperr.Unauthorized("unauth")))
	assert.False(t, apperr.IsNotFound(apperr.Internal("oops")))
}

func TestIsNotFound_FalseForPlainError(t *testing.T) {
	assert.False(t, apperr.IsNotFound(errors.New("not found")))
}

// ─── IsForbidden ──────────────────────────────────────────────────────────────

func TestIsForbidden_True(t *testing.T) {
	assert.True(t, apperr.IsForbidden(apperr.Forbidden("not allowed")))
}

func TestIsForbidden_FalseForOtherStatus(t *testing.T) {
	assert.False(t, apperr.IsForbidden(apperr.NotFound("x", "1")))
	assert.False(t, apperr.IsForbidden(apperr.Unauthorized("unauth")))
	assert.False(t, apperr.IsForbidden(apperr.Internal("oops")))
}

func TestIsForbidden_FalseForPlainError(t *testing.T) {
	assert.False(t, apperr.IsForbidden(errors.New("forbidden")))
}
