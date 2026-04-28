package repositories_test

// Migration test for the calls scope columns (Task 4.7).
//
// The 000013_add_scope_to_calls migration adds nullable scope_type +
// scope_id columns. This test asserts that a pre-existing call (created
// before scope columns existed, or simply without scope set) still loads
// correctly after the migration runs — both fields are NULL.
//
// Requires the migration runner to have been executed against TEST_DATABASE_URL
// (handled by `make test-integration`).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/repositories"
)

func TestCallScopeColumns_LegacyCall_HasNullScope(t *testing.T) {
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "scope-legacy@rekall.io", "Scope Legacy", "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })

	// Create the call without setting scope — exercises the same code path
	// pre-existing rows hit on first read after the migration.
	callSvc := services.NewCallService(callRepo, nil, nil, zap.NewNop())
	call, err := callSvc.CreateCall(ctx, services.CreateCallInput{
		UserID: user.ID,
		Title:  "Legacy call (no scope)",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, call.ID) })

	// Read back — the migration's schema must expose scope_type and scope_id
	// as nullable so the entity's *string and *uuid.UUID fields stay nil.
	fetched, err := callRepo.GetByID(ctx, call.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched.ScopeType, "legacy call must have NULL scope_type after migration")
	assert.Nil(t, fetched.ScopeID, "legacy call must have NULL scope_id after migration")
}

func TestCallScopeColumns_OrgScope_PersistsRoundTrip(t *testing.T) {
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "scope-org@rekall.io", "Scope Org", "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })

	scopeType := "organization"
	scopeID := uuid.New()

	// Bypass the service so we don't need a live OrgMembership row — this
	// test asserts the column round-trip, not the membership gate.
	created, err := callRepo.Create(ctx, &entities.Call{
		UserID:    user.ID,
		Title:     "Org-scoped call",
		Status:    "pending",
		Metadata:  entities.JSONMap{},
		ScopeType: &scopeType,
		ScopeID:   &scopeID,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, created.ID) })

	fetched, err := callRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.ScopeType)
	require.NotNil(t, fetched.ScopeID)
	assert.Equal(t, "organization", *fetched.ScopeType)
	assert.Equal(t, scopeID, *fetched.ScopeID)
}

func TestCallScopeColumns_PartialIndex_FilterByOrg(t *testing.T) {
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "scope-filter@rekall.io", "Scope Filter", "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })

	scopeType := "organization"
	orgID := uuid.New()

	// One open + one org-scoped row.
	openCall, err := callRepo.Create(ctx, &entities.Call{
		UserID:   user.ID,
		Title:    "Open",
		Status:   "pending",
		Metadata: entities.JSONMap{},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, openCall.ID) })

	scopedCall, err := callRepo.Create(ctx, &entities.Call{
		UserID:    user.ID,
		Title:     "Scoped",
		Status:    "pending",
		Metadata:  entities.JSONMap{},
		ScopeType: &scopeType,
		ScopeID:   &orgID,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, scopedCall.ID) })

	// Filter using the new ports.ScopeFilter — should hit the partial index
	// and return only the scoped row.
	calls, _, err := callRepo.List(ctx, ports.ListCallsFilter{
		UserID: &user.ID,
		Scope:  &ports.ScopeFilter{Kind: ports.ScopeKindOrganization, ID: orgID},
	}, 1, 50)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, scopedCall.ID, calls[0].ID)

	// Filter to open scope — should return only the open row.
	openOnly, _, err := callRepo.List(ctx, ports.ListCallsFilter{
		UserID: &user.ID,
		Scope:  &ports.ScopeFilter{Kind: ports.ScopeKindOpen},
	}, 1, 50)
	require.NoError(t, err)
	require.Len(t, openOnly, 1)
	assert.Equal(t, openCall.ID, openOnly[0].ID)
}
