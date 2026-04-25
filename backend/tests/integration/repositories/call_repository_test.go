package repositories_test

// Integration tests require a running PostgreSQL instance.
// Run with: make test-integration (which starts docker-compose test services)
//
// Set TEST_DATABASE_URL to override the default connection:
//   TEST_DATABASE_URL="host=localhost user=rekall password=rekall_secret dbname=rekall_test port=5432 sslmode=disable" \
//   go test ./tests/integration/...

import (
	"context"
	"os"
	"testing"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

func TestCallRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	// Create a user first (foreign key constraint).
	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "test-call@rekall.io", "Test User", "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })

	// Create call via service.
	callSvc := services.NewCallService(callRepo, nil, nil, zap.NewNop())
	call, err := callSvc.CreateCall(ctx, services.CreateCallInput{
		UserID: user.ID,
		Title:  "Integration Test Call",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, call.ID)
	assert.Equal(t, "pending", call.Status)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, call.ID) })

	// Retrieve via repository.
	fetched, err := callRepo.GetByID(ctx, call.ID)
	require.NoError(t, err)
	assert.Equal(t, call.ID, fetched.ID)
	assert.Equal(t, "Integration Test Call", fetched.Title)
}

func TestCallRepository_SoftDelete(t *testing.T) {
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "test-delete@rekall.io", "Delete User", "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })

	callSvc := services.NewCallService(callRepo, nil, nil, zap.NewNop())
	call, err := callSvc.CreateCall(ctx, services.CreateCallInput{UserID: user.ID, Title: "To Delete"})
	require.NoError(t, err)

	require.NoError(t, callRepo.SoftDelete(ctx, call.ID))

	_, err = callRepo.GetByID(ctx, call.ID)
	assert.Error(t, err, "expected not-found after soft delete")
}
