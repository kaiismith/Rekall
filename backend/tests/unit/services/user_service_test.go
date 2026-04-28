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

// ─── Helper ───────────────────────────────────────────────────────────────────

func newUserService(repo *mockUserRepo) *services.UserService {
	return services.NewUserService(repo, zap.NewNop())
}

// ─── CreateUser ───────────────────────────────────────────────────────────────

func TestCreateUser_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	created := &entities.User{ID: uuid.New(), Email: "alice@example.com", FullName: "Alice", Role: "member"}

	repo.On("GetByEmail", ctx, "alice@example.com").Return(nil, apperr.NotFound("User", "alice@example.com"))
	repo.On("Create", ctx, mock.AnythingOfType("*entities.User")).Return(created, nil)

	user, err := svc.CreateUser(ctx, "alice@example.com", "Alice", "member")

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "member", user.Role)
}

func TestCreateUser_DefaultsToMemberRole(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	created := &entities.User{ID: uuid.New(), Email: "bob@example.com", FullName: "Bob", Role: "member"}

	repo.On("GetByEmail", ctx, "bob@example.com").Return(nil, apperr.NotFound("User", "bob@example.com"))
	repo.On("Create", ctx, mock.MatchedBy(func(u *entities.User) bool {
		return u.Role == "member"
	})).Return(created, nil)

	user, err := svc.CreateUser(ctx, "bob@example.com", "Bob", "") // empty role

	require.NoError(t, err)
	assert.Equal(t, "member", user.Role)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	existing := &entities.User{ID: uuid.New(), Email: "alice@example.com"}
	repo.On("GetByEmail", ctx, "alice@example.com").Return(existing, nil)

	_, err := svc.CreateUser(ctx, "alice@example.com", "Alice", "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 409, appErr.Status)
}

func TestCreateUser_MissingEmail(t *testing.T) {
	svc := newUserService(new(mockUserRepo))

	_, err := svc.CreateUser(context.Background(), "", "Alice", "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateUser_MissingFullName(t *testing.T) {
	svc := newUserService(new(mockUserRepo))

	_, err := svc.CreateUser(context.Background(), "alice@example.com", "", "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

// ─── GetUser ──────────────────────────────────────────────────────────────────

func TestGetUser_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	id := uuid.New()
	user := &entities.User{ID: id, Email: "alice@example.com", Role: "member"}
	repo.On("GetByID", ctx, id).Return(user, nil)

	result, err := svc.GetUser(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestGetUser_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, apperr.NotFound("User", id.String()))

	_, err := svc.GetUser(ctx, id)

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestGetUser_RepoError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, assert.AnError)

	_, err := svc.GetUser(ctx, id)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestCreateUser_RepoGetByEmailError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "a@b.com").Return(nil, assert.AnError)

	_, err := svc.CreateUser(ctx, "a@b.com", "Alice", "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestCreateUser_RepoCreateError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "a@b.com").Return(nil, apperr.NotFound("User", "a@b.com"))
	repo.On("Create", ctx, mock.AnythingOfType("*entities.User")).Return(nil, assert.AnError)

	_, err := svc.CreateUser(ctx, "a@b.com", "Alice", "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestDeleteUser_SoftDeleteError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	id := uuid.New()
	user := &entities.User{ID: id, Email: "a@b.com", Role: "member"}
	repo.On("GetByID", ctx, id).Return(user, nil)
	repo.On("SoftDelete", ctx, id).Return(assert.AnError)

	err := svc.DeleteUser(ctx, id)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── ListUsers ────────────────────────────────────────────────────────────────

func TestListUsers_ReturnsPaginatedResults(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	users := []*entities.User{
		{ID: uuid.New(), Email: "a@example.com"},
		{ID: uuid.New(), Email: "b@example.com"},
	}
	repo.On("List", ctx, 1, 20).Return(users, 2, nil)

	result, total, err := svc.ListUsers(ctx, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestListUsers_EmptyResult(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	repo.On("List", ctx, 1, 20).Return([]*entities.User{}, 0, nil)

	result, total, err := svc.ListUsers(ctx, 1, 20)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

func TestListUsers_RepoError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	repo.On("List", ctx, 1, 20).Return([]*entities.User(nil), 0, assert.AnError)

	_, _, err := svc.ListUsers(ctx, 1, 20)
	require.Error(t, err)
}

// ─── DeleteUser ───────────────────────────────────────────────────────────────

func TestDeleteUser_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()

	id := uuid.New()
	user := &entities.User{ID: id, Email: "alice@example.com", Role: "member"}
	repo.On("GetByID", ctx, id).Return(user, nil)
	repo.On("SoftDelete", ctx, id).Return(nil)

	require.NoError(t, svc.DeleteUser(ctx, id))
	repo.AssertExpectations(t)
}

func TestDeleteUser_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := newUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, apperr.NotFound("User", id.String()))

	err := svc.DeleteUser(ctx, id)

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}
