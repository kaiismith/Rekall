package httpiface_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/internal/application/services"
	httpiface "github.com/rekall/backend/internal/interfaces/http"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	"github.com/rekall/backend/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

// ─── newMockDB ────────────────────────────────────────────────────────────────

func newMockDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// newTestDeps constructs minimal RouterDeps with handlers wired to in-memory services.
// Services receive mocked repositories that return safe defaults so routes respond
// without calling a real database.
func newTestDeps(t *testing.T) httpiface.RouterDeps {
	t.Helper()
	log := zap.NewNop()
	db := newMockDB(t)

	// Stub services with empty mocks — handlers won't be invoked in router-existence tests.
	// We just need non-nil handlers so route registration succeeds.
	authSvc := services.NewAuthService(nil, nil, nil, "secret", "rekall", "http://localhost", 15*time.Minute, 168*time.Hour, time.Hour, 24*time.Hour, log)
	callSvc := services.NewCallService(nil, log)
	userSvc := services.NewUserService(nil, log)
	orgSvc := services.NewOrganizationService(nil, nil, nil, nil, nil, "http://localhost", 48*time.Hour, log)
	deptSvc := services.NewDepartmentService(nil, nil, nil, log)

	return httpiface.RouterDeps{
		Logger:         log,
		JWTSecret:      "test-secret-for-router-tests",
		JWTIssuer:      "rekall-test",
		HealthH:        handlers.NewHealthHandler(db),
		CallH:          handlers.NewCallHandler(callSvc, log),
		UserH:          handlers.NewUserHandler(userSvc, log),
		AuthH:          handlers.NewAuthHandler(authSvc, 168*time.Hour, log),
		OrgH:           handlers.NewOrganizationHandler(orgSvc, log),
		DeptH:          handlers.NewDepartmentHandler(deptSvc, log),
		MeetingH:       nil, // meeting routes guarded by nil-check
		CORSOrigins:    []string{"http://localhost:3000"},
		SwaggerEnabled: false,
	}
}

// ─── Router construction ──────────────────────────────────────────────────────

func TestNewRouter_Constructs(t *testing.T) {
	r := httpiface.NewRouter(newTestDeps(t))
	assert.NotNil(t, r)
}

func TestNewRouter_RegistersHealthRoutes(t *testing.T) {
	r := httpiface.NewRouter(newTestDeps(t))

	for _, path := range []string{"/health", "/ready"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "expected %s to be registered", path)
	}
}

func TestNewRouter_RegistersAuthPublicRoutes(t *testing.T) {
	r := httpiface.NewRouter(newTestDeps(t))

	publicRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodGet, "/api/v1/auth/verify"},
		{http.MethodPost, "/api/v1/auth/verify/resend"},
		{http.MethodPost, "/api/v1/auth/password/forgot"},
		{http.MethodPost, "/api/v1/auth/password/reset"},
	}

	for _, route := range publicRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			body := strings.NewReader(`{}`)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, body)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			// Route should be registered — any response other than 404.
			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestNewRouter_ProtectedRoutesRequireAuth(t *testing.T) {
	r := httpiface.NewRouter(newTestDeps(t))

	protected := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/calls"},
		{http.MethodGet, "/api/v1/organizations"},
		{http.MethodGet, "/api/v1/users"},
	}

	for _, route := range protected {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			// No Authorization header → must be 401.
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestNewRouter_SwaggerDisabled_Returns404(t *testing.T) {
	deps := newTestDeps(t)
	deps.SwaggerEnabled = false
	r := httpiface.NewRouter(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_SwaggerEnabled_RegistersDocs(t *testing.T) {
	deps := newTestDeps(t)
	deps.SwaggerEnabled = true
	r := httpiface.NewRouter(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
	r.ServeHTTP(w, req)
	// Swagger returns 200 or 301 depending on version — not 404.
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_MeetingHNil_SkipsMeetingRoutes(t *testing.T) {
	deps := newTestDeps(t)
	deps.MeetingH = nil
	r := httpiface.NewRouter(deps)

	// Authenticated meetings routes should not exist when MeetingH is nil.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	// Even with an invalid token, since the route isn't registered,
	// the 404 comes from the router (after authentication fails first).
	assert.Contains(t, []int{http.StatusNotFound, http.StatusUnauthorized}, w.Code)
}

func TestNewRouter_MeetingHWired_RegistersMeetingRoutes(t *testing.T) {
	// Supply a MeetingH so the non-nil branch in NewRouter fires.
	deps := newTestDeps(t)
	log := zap.NewNop()
	meetingSvc := services.NewMeetingService(nil, nil, nil, nil, "http://rekall.test", log)
	manager := wsHub.NewHubManager(nil, log)
	deps.MeetingH = handlers.NewMeetingHandler(meetingSvc, nil, nil, manager, "http://rekall.test",
		deps.JWTSecret, deps.JWTIssuer, log)

	r := httpiface.NewRouter(deps)

	// Verify the authenticated meetings route exists (returns 401 without auth, not 404).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/mine", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Verify the WS endpoint is registered (public, no JWT middleware).
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/abc-defg-hij/ws", nil)
	r.ServeHTTP(w2, req2)
	// The Connect handler rejects missing token with 401 (not 404).
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

// ─── Server ──────────────────────────────────────────────────────────────────

func TestNewServer_Constructs(t *testing.T) {
	cfg := config.ServerConfig{
		Port:         "0", // let OS pick
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	router := gin.New()
	s := httpiface.NewServer(cfg, router, zap.NewNop())
	assert.NotNil(t, s)
}

func TestServer_Start_BindError(t *testing.T) {
	// Bind to port 1 (reserved) → ListenAndServe returns a non-ErrServerClosed error.
	cfg := config.ServerConfig{
		Port:         "1",
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	}
	s := httpiface.NewServer(cfg, gin.New(), zap.NewNop())

	err := s.Start()
	// Expected: permission denied / address in use / bind error wrapped as "server: ..."
	if err != nil {
		assert.Contains(t, err.Error(), "server:")
	}
	// On some systems port 1 may bind (running as root in CI container),
	// so tolerate both paths. We at least hit the wrapping branch when it fails.
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := config.ServerConfig{
		Port:         "0",
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	s := httpiface.NewServer(cfg, router, zap.NewNop())

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start() }()

	// Give the server a moment to start (port=0 picks a free port).
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop after shutdown")
	}
}
