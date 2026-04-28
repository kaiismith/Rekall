package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
)

const (
	testSecret = "test-secret-key"
	testIssuer = "rekall-test"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// signToken builds a minimal signed JWT with the given role.
func signToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now().UTC()
	claims := infraauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			Issuer:    testIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		Email: "test@example.com",
		Role:  role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return signed
}

// newRouter builds a test Gin engine with Authenticate + RequireRole on GET /protected.
func newRouter(allowedRoles ...string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	r.Use(middleware.RequireRole(allowedRoles...))
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// ─── Authenticate ─────────────────────────────────────────────────────────────

func TestAuthenticate_MissingHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthenticate_MalformedHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "NotBearer token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthenticate_ValidToken(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Authenticate(testSecret, testIssuer, zap.NewNop()))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, "member"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── RequireRole ──────────────────────────────────────────────────────────────

func TestRequireRole_AdminAllowed(t *testing.T) {
	r := newRouter("admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, "admin"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_MemberDenied(t *testing.T) {
	r := newRouter("admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, "member"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRole_MultipleRolesAllowed(t *testing.T) {
	r := newRouter("admin", "member")

	for _, role := range []string{"admin", "member"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+signToken(t, role))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "role %q should be allowed", role)
	}
}

func TestRequireRole_CaseInsensitive(t *testing.T) {
	r := newRouter("Admin") // registered with capital A

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, "admin")) // token carries lowercase
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRequireRole_WithoutAuthenticateMiddleware ensures RequireRole aborts with
// 401 when claims are absent from the context (e.g. misconfigured chain).
func TestRequireRole_WithoutAuthenticateMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Only RequireRole — no Authenticate first, so claims will be nil.
	r.Use(middleware.RequireRole("admin"))
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
