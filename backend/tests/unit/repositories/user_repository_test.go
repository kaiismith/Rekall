package repositories_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Rows shared by multiple user tests.
func userRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "email", "full_name", "role", "password_hash", "email_verified", "created_at", "updated_at",
	})
}

// ─── NewUserRepository ────────────────────────────────────────────────────────

func TestNewUserRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewUserRepository(db))
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestUserRepository_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	user := &entities.User{Email: "a@b.com", FullName: "Alice", Role: "member"}
	result, err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserRepository_Create_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Create(context.Background(), &entities.User{Email: "x@y.z"})
	require.Error(t, err)
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func TestUserRepository_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnRows(userRows().AddRow(id, "a@b.com", "Alice", "member", "hash", true, time.Now(), time.Now()))

	u, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, u.ID)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUserRepository_GetByID_OtherError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(assert.AnError)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.False(t, apperr.IsNotFound(err))
}

// ─── GetByEmail ───────────────────────────────────────────────────────────────

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
		WithArgs("a@b.com", sqlmock.AnyArg()).
		WillReturnRows(userRows().AddRow(uuid.New(), "a@b.com", "Alice", "member", "h", true, time.Now(), time.Now()))

	u, err := repo.GetByEmail(context.Background(), "a@b.com")
	require.NoError(t, err)
	assert.Equal(t, "a@b.com", u.Email)
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
		WithArgs("x@x.com", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByEmail(context.Background(), "x@x.com")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUserRepository_GetByEmail_OtherError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs("x@x.com", sqlmock.AnyArg()).
		WillReturnError(assert.AnError)

	_, err := repo.GetByEmail(context.Background(), "x@x.com")
	require.Error(t, err)
	assert.False(t, apperr.IsNotFound(err))
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestUserRepository_List_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRows().
			AddRow(uuid.New(), "a@b.com", "Alice", "member", "h", true, time.Now(), time.Now()).
			AddRow(uuid.New(), "c@d.com", "Bob", "member", "h", true, time.Now(), time.Now()))

	users, total, err := repo.List(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
}

func TestUserRepository_List_CountError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WillReturnError(assert.AnError)

	_, _, err := repo.List(context.Background(), 1, 10)
	require.Error(t, err)
}

func TestUserRepository_List_SelectError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnError(assert.AnError)

	_, _, err := repo.List(context.Background(), 1, 10)
	require.Error(t, err)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestUserRepository_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	u := &entities.User{ID: uuid.New(), Email: "a@b.com", FullName: "Alice", Role: "member"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.Update(context.Background(), u)
	require.NoError(t, err)
	assert.Equal(t, u.ID, result.ID)
}

func TestUserRepository_Update_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	u := &entities.User{ID: uuid.New(), Email: "a@b.com"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Update(context.Background(), u)
	require.Error(t, err)
}

// ─── SoftDelete ───────────────────────────────────────────────────────────────

func TestUserRepository_SoftDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SoftDelete(context.Background(), id))
}

func TestUserRepository_SoftDelete_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.SoftDelete(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUserRepository_SoftDelete_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users"`)).
		WithArgs(id).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	require.Error(t, repo.SoftDelete(context.Background(), id))
}

// ─── SetEmailVerified ─────────────────────────────────────────────────────────

func TestUserRepository_SetEmailVerified_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email_verified"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SetEmailVerified(context.Background(), id, true))
}

// ─── UpdatePassword ───────────────────────────────────────────────────────────

func TestUserRepository_UpdatePassword_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "password_hash"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdatePassword(context.Background(), id, "new-hash"))
}
