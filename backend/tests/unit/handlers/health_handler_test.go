package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/interfaces/http/handlers"
)

// newMockDB returns a gorm.DB backed by go-sqlmock plus the controlling mock
// and the underlying *sql.DB (so tests can close it to simulate failures).
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	// MonitorPingsOption defaults to false → Ping is a no-op on the mock.
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}

// ─── NewHealthHandler ─────────────────────────────────────────────────────────

func TestNewHealthHandler(t *testing.T) {
	db, _, cleanup := newMockDB(t)
	defer cleanup()
	h := handlers.NewHealthHandler(db)
	assert.NotNil(t, h)
}

// ─── Liveness ─────────────────────────────────────────────────────────────────

func TestHealthLiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, cleanup := newMockDB(t)
	defer cleanup()

	h := handlers.NewHealthHandler(db)
	router := gin.New()
	router.GET("/health", h.Liveness)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	data := body["data"].(map[string]any)
	assert.Equal(t, "ok", data["status"])
}

// ─── Readiness ────────────────────────────────────────────────────────────────

func TestHealthReadiness_DBAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, cleanup := newMockDB(t)
	defer cleanup()

	h := handlers.NewHealthHandler(db)
	router := gin.New()
	router.GET("/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	data := body["data"].(map[string]any)
	assert.Equal(t, "ready", data["status"])
	assert.Equal(t, "ok", data["database"])
}

func TestHealthReadiness_DBUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, cleanup := newMockDB(t)
	// Close the underlying sql.DB so any Ping fails.
	cleanup()

	h := handlers.NewHealthHandler(db)
	router := gin.New()
	router.GET("/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errField, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "DB_UNAVAILABLE", errField["code"])
}
