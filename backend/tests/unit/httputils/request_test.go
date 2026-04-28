package httputils_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httputils "github.com/rekall/backend/internal/interfaces/http/utils"
)

func init() { gin.SetMode(gin.TestMode) }

// ─── ParseUUID ────────────────────────────────────────────────────────────────

func TestParseUUID_Valid(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	id := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	got, err := httputils.ParseUUID(c, "id")
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestParseUUID_Invalid(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	_, err := httputils.ParseUUID(c, "id")
	require.Error(t, err)
}

// ─── QueryInt ────────────────────────────────────────────────────────────────

func TestQueryInt_UsesDefaultWhenAbsent(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?", nil)

	got := httputils.QueryInt(c, "page", 42)
	assert.Equal(t, 42, got)
}

func TestQueryInt_ParsesValidValue(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=7", nil)

	got := httputils.QueryInt(c, "page", 1)
	assert.Equal(t, 7, got)
}

func TestQueryInt_InvalidStringUsesDefault(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=abc", nil)

	got := httputils.QueryInt(c, "page", 1)
	assert.Equal(t, 1, got)
}

func TestQueryInt_ZeroUsesDefault(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=0", nil)

	// Zero is below the minimum (1), so the default kicks in.
	got := httputils.QueryInt(c, "page", 5)
	assert.Equal(t, 5, got)
}

func TestQueryInt_NegativeUsesDefault(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=-3", nil)

	got := httputils.QueryInt(c, "page", 5)
	assert.Equal(t, 5, got)
}
