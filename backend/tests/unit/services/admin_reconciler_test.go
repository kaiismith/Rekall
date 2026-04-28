package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── Reconcile: bootstrap path ───────────────────────────────────────────────

func TestAdminReconciler_BootstrapsMissingUser(t *testing.T) {
	repo := new(mockUserRepo)
	emails := []string{"alice@example.com"}
	rec := services.NewAdminReconciler(repo, emails, "secret123!", zap.NewNop())

	// First lookup misses → triggers bootstrap.
	repo.On("GetByEmail", mock.Anything, "alice@example.com").
		Return(nil, apperr.NotFound("User", "alice@example.com")).Once()
	repo.On("Create", mock.Anything, mock.MatchedBy(func(u *entities.User) bool {
		return u.Email == "alice@example.com" && u.Role == "admin" && u.EmailVerified
	})).Return(&entities.User{Email: "alice@example.com", Role: "admin"}, nil).Once()

	// After bootstrap, step-2 lookup succeeds → already admin → no SetRoleByEmail.
	repo.On("GetByEmail", mock.Anything, "alice@example.com").
		Return(&entities.User{Email: "alice@example.com", Role: "admin"}, nil).Once()
	repo.On("DemoteAdminsExcept", mock.Anything, emails).Return(0, nil)

	res, err := rec.Reconcile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 0, res.Promoted)
	repo.AssertExpectations(t)
}

// ─── Reconcile: promote existing user ────────────────────────────────────────

func TestAdminReconciler_PromotesExistingMember(t *testing.T) {
	repo := new(mockUserRepo)
	emails := []string{"bob@example.com"}
	rec := services.NewAdminReconciler(repo, emails, "", zap.NewNop()) // no bootstrap pwd

	repo.On("GetByEmail", mock.Anything, "bob@example.com").
		Return(&entities.User{ID: uuid.New(), Email: "bob@example.com", Role: "member"}, nil).Once()
	repo.On("SetRoleByEmail", mock.Anything, "bob@example.com", "admin").Return(nil)
	repo.On("DemoteAdminsExcept", mock.Anything, emails).Return(0, nil)

	res, err := rec.Reconcile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created)
	assert.Equal(t, 1, res.Promoted)
	repo.AssertExpectations(t)
}

// ─── Reconcile: idempotent — already admin ───────────────────────────────────

func TestAdminReconciler_AlreadyAdminIsNoOp(t *testing.T) {
	repo := new(mockUserRepo)
	emails := []string{"carol@example.com"}
	rec := services.NewAdminReconciler(repo, emails, "", zap.NewNop())

	repo.On("GetByEmail", mock.Anything, "carol@example.com").
		Return(&entities.User{Email: "carol@example.com", Role: "admin"}, nil).Once()
	repo.On("DemoteAdminsExcept", mock.Anything, emails).Return(0, nil)

	res, err := rec.Reconcile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created)
	assert.Equal(t, 0, res.Promoted)
	repo.AssertNotCalled(t, "SetRoleByEmail", mock.Anything, mock.Anything, mock.Anything)
}

// ─── Reconcile: demote unlisted admin ────────────────────────────────────────

func TestAdminReconciler_DemotesUnlistedAdmins(t *testing.T) {
	repo := new(mockUserRepo)
	emails := []string{} // empty list → every existing admin gets demoted.
	rec := services.NewAdminReconciler(repo, emails, "", zap.NewNop())

	repo.On("DemoteAdminsExcept", mock.Anything, emails).Return(3, nil)

	res, err := rec.Reconcile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res.Demoted)
}

// ─── Reconcile: skip unknown email when no bootstrap pwd ─────────────────────

func TestAdminReconciler_SkipsUnknownWhenNoBootstrapPassword(t *testing.T) {
	repo := new(mockUserRepo)
	emails := []string{"ghost@example.com"}
	rec := services.NewAdminReconciler(repo, emails, "", zap.NewNop()) // no pwd → no create

	repo.On("GetByEmail", mock.Anything, "ghost@example.com").
		Return(nil, apperr.NotFound("User", "ghost@example.com"))
	repo.On("DemoteAdminsExcept", mock.Anything, emails).Return(0, nil)

	res, err := rec.Reconcile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created)
	assert.Equal(t, 0, res.Promoted)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SetRoleByEmail", mock.Anything, mock.Anything, mock.Anything)
}
