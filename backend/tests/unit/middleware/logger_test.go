package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/interfaces/http/middleware"
)

// ─── Logger middleware (access log) ───────────────────────────────────────────

func TestLogger_LogsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok?q=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_LogsClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.GET("/bad", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusBadRequest)
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogger_LogsServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.GET("/fail", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogger_SurfacesGinErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.GET("/errs", func(c *gin.Context) {
		_ = c.Error(errors.New("handler error"))
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/errs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Logger should process c.Errors branch; no panic is the assertion here.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogger_CapturesRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.POST("/resource/:id", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodPost, "/resource/42?x=y", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ─── CORS ─────────────────────────────────────────────────────────────────────

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS([]string{"http://localhost:5173"}))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// gin-contrib/cors sets Access-Control-Allow-Origin for allowed origins
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS([]string{"http://localhost:5173"}))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// OPTIONS preflight should return 2xx (204 No Content is typical)
	assert.True(t, w.Code < 300, "expected 2xx but got %d", w.Code)
}
