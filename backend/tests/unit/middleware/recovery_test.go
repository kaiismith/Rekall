package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/interfaces/http/middleware"
)

// ─── Recovery ─────────────────────────────────────────────────────────────────

func TestRecovery_CatchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.GET("/boom", func(c *gin.Context) {
		panic("intentional panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()

	// Must not propagate panic past Recovery.
	require.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errField, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", errField["code"])
	assert.NotEmpty(t, errField["message"])
}

func TestRecovery_CatchesPanicWithNonStringValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.GET("/boom", func(c *gin.Context) {
		panic(map[string]string{"detail": "structured panic"})
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()

	require.NotPanics(t, func() { r.ServeHTTP(w, req) })
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecovery_PassThroughNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello", w.Body.String())
}
