package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/interfaces/http/middleware"
	"github.com/rekall/backend/pkg/constants"
)

// ─── RequestID ────────────────────────────────────────────────────────────────

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())

	var capturedID string
	r.GET("/ping", func(c *gin.Context) {
		capturedID = c.GetString(constants.CtxKeyRequestID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Generated a UUID-shaped ID
	assert.NotEmpty(t, capturedID)
	_, err := uuid.Parse(capturedID)
	require.NoError(t, err, "generated request ID should be a valid UUID")

	// Response header should echo the same ID
	assert.Equal(t, capturedID, w.Header().Get(constants.HeaderRequestID))
}

func TestRequestID_ReusesClientHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())

	clientID := "client-supplied-trace-id"
	var capturedID string
	r.GET("/ping", func(c *gin.Context) {
		capturedID = c.GetString(constants.CtxKeyRequestID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(constants.HeaderRequestID, clientID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, clientID, capturedID)
	assert.Equal(t, clientID, w.Header().Get(constants.HeaderRequestID))
}

func TestRequestID_GeneratesUniqueIDsPerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())

	ids := make([]string, 3)
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(constants.CtxKeyRequestID))
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		ids[i] = w.Body.String()
	}

	assert.NotEqual(t, ids[0], ids[1])
	assert.NotEqual(t, ids[1], ids[2])
	assert.NotEqual(t, ids[0], ids[2])
}
