package admin_test

// Live database integration tests for the platform-admin reconciler and the
// admin-only org-create gate (Tasks 21.2 / 21.3 / 21.5).
//
// Requires TEST_DATABASE_URL pointing at a Postgres instance with the
// migrations applied — when unset, every test in this file skips so the
// regular `go test ./...` run on a developer machine without docker stays
// fast.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	"github.com/rekall/backend/pkg/constants"
)

const testJWTIssuer = "rekall-test"

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// uniqueEmail produces a per-test email address so parallel runs don't collide
// on the users.email unique index.
func uniqueEmail(prefix string) string {
	return prefix + "-" + uuid.NewString()[:8] + "@rekall-test.io"
}

// ─── Reconciler ──────────────────────────────────────────────────────────────

func TestReconciler_PromotesAndDemotes(t *testing.T) {
	db := testDB(t)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ssword!"), 10)
	carolEmail := strings.ToLower(uniqueEmail("carol"))
	dianaEmail := strings.ToLower(uniqueEmail("diana"))

	carol := &entities.User{
		ID:            uuid.New(),
		Email:         carolEmail,
		FullName:      "Carol",
		Role:          "member",
		PasswordHash:  string(hash),
		EmailVerified: true,
	}
	diana := &entities.User{
		ID:            uuid.New(),
		Email:         dianaEmail,
		FullName:      "Diana",
		Role:          "admin", // currently admin; should be demoted
		PasswordHash:  string(hash),
		EmailVerified: true,
	}
	_, err := userRepo.Create(ctx, carol)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, diana)
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, carol.ID); _ = userRepo.SoftDelete(ctx, diana.ID) })

	// Reconcile with carol listed as the sole admin → promote carol, demote diana.
	rec := services.NewAdminReconciler(userRepo, []string{carolEmail}, "", zap.NewNop())
	res, err := rec.Reconcile(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Promoted, 1)
	assert.GreaterOrEqual(t, res.Demoted, 1)

	// Verify in DB.
	gotCarol, err := userRepo.GetByEmail(ctx, carolEmail)
	require.NoError(t, err)
	assert.Equal(t, "admin", gotCarol.Role)
	gotDiana, err := userRepo.GetByEmail(ctx, dianaEmail)
	require.NoError(t, err)
	assert.Equal(t, "member", gotDiana.Role)
}

func TestReconciler_BootstrapCreatesUser(t *testing.T) {
	db := testDB(t)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	email := strings.ToLower(uniqueEmail("bootstrap"))
	rec := services.NewAdminReconciler(userRepo, []string{email}, "p@ssword!", zap.NewNop())

	res, err := rec.Reconcile(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Created, 1)

	got, err := userRepo.GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Role)
	assert.True(t, got.EmailVerified)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, got.ID) })
}

// ─── HTTP — admin-gated org create ───────────────────────────────────────────

func TestHTTP_OrgCreate_NonAdmin_403(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(t, db)

	// Plain member token.
	memberID, memberEmail := seedUser(t, db, "member")
	t.Cleanup(func() { _ = repositories.NewUserRepository(db).SoftDelete(context.Background(), memberID) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations",
		strings.NewReader(`{"name":"Should-Fail"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issueToken(t, memberID, "member"))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "non-admin org create must be 403")
	_ = memberEmail
}

func TestHTTP_OrgCreate_Admin_201(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(t, db)

	adminID, _ := seedUser(t, db, "admin")
	t.Cleanup(func() { _ = repositories.NewUserRepository(db).SoftDelete(context.Background(), adminID) })

	body := `{"name":"Admin-Created Org"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issueToken(t, adminID, "admin"))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHTTP_OrgCreate_AdminWithUnknownOwnerEmail_422(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(t, db)

	adminID, _ := seedUser(t, db, "admin")
	t.Cleanup(func() { _ = repositories.NewUserRepository(db).SoftDelete(context.Background(), adminID) })

	body := `{"name":"Bad Owner","owner_email":"ghost@example.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issueToken(t, adminID, "admin"))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

const testJWTSecret = "integration-test-secret"

// newTestRouter wires the slice of the production router we care about for
// these tests: middleware.Authenticate + middleware.RequireRole + the org
// handler. We deliberately bypass the full RouterDeps wiring so the test
// stays focused on the admin gate.
func newTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	memberRepo := repositories.NewOrgMembershipRepository(db)
	inviteRepo := repositories.NewInvitationRepository(db)
	userRepo := repositories.NewUserRepository(db)
	orgRepo := repositories.NewOrganizationRepository(db)

	orgSvc := services.NewOrganizationService(
		orgRepo, memberRepo, inviteRepo, userRepo, nil,
		"http://localhost", time.Hour, zap.NewNop(),
	)
	orgH := handlers.NewOrganizationHandler(orgSvc, zap.NewNop())

	api := r.Group("/api/v1")
	api.Use(middleware.Authenticate(testJWTSecret, testJWTIssuer, zap.NewNop()))
	orgs := api.Group("/organizations")
	orgs.POST("", middleware.RequireRole(constants.UserRoleAdmin), orgH.Create)
	return r
}

// seedUser creates a user with the given role and returns its ID + email.
func seedUser(t *testing.T, db *gorm.DB, role string) (uuid.UUID, string) {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("p@ssword!"), 10)
	u := &entities.User{
		ID:            uuid.New(),
		Email:         uniqueEmail(role),
		FullName:      strings.ToUpper(role[:1]) + role[1:],
		Role:          role,
		PasswordHash:  string(hash),
		EmailVerified: true,
	}
	_, err := repositories.NewUserRepository(db).Create(context.Background(), u)
	require.NoError(t, err)
	return u.ID, u.Email
}

// issueToken mints a JWT with the same shape the real auth flow produces so
// the middleware accepts it byte-for-byte.
func issueToken(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()
	user := &entities.User{
		ID:    userID,
		Email: uniqueEmail("token"),
		Role:  role,
	}
	tok, err := infraauth.SignAccessToken(user, testJWTSecret, testJWTIssuer, 5*time.Minute)
	require.NoError(t, err)
	return tok
}
