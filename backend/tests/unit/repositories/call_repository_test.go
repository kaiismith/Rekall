package repositories_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newMockDB returns a gorm.DB backed by go-sqlmock.
// Uses QueryMatcherRegexp so tests don't need to match GORM's exact SQL.
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

// ─── NewCallRepository ────────────────────────────────────────────────────────

func TestNewCallRepository(t *testing.T) {
	db, _ := newMockDB(t)
	repo := repositories.NewCallRepository(db)
	assert.NotNil(t, repo)
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestCallRepository_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "calls"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	call := &entities.Call{
		UserID: uuid.New(),
		Title:  "Test",
		Status: "pending",
	}
	result, err := repo.Create(context.Background(), call)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCallRepository_Create_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "calls"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	call := &entities.Call{UserID: uuid.New(), Title: "X", Status: "pending"}
	_, err := repo.Create(context.Background(), call)
	require.Error(t, err)
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func TestCallRepository_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()
	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls" WHERE id = $1`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "title", "duration_sec", "status", "metadata", "created_at", "updated_at",
		}).AddRow(id, userID, "Test Call", 60, "done", "{}", time.Now(), time.Now()))

	call, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, call.ID)
	assert.Equal(t, "Test Call", call.Title)
}

func TestCallRepository_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestCallRepository_GetByID_OtherError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(assert.AnError)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.False(t, apperr.IsNotFound(err))
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestCallRepository_List_NoFilters(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	// COUNT
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// SELECT
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "title", "duration_sec", "status", "metadata", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), uuid.New(), "Call 1", 10, "done", "{}", time.Now(), time.Now()).
		AddRow(uuid.New(), uuid.New(), "Call 2", 20, "pending", "{}", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls"`)).
		WillReturnRows(rows)

	list, total, err := repo.List(context.Background(), ports.ListCallsFilter{}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, list, 2)
}

func TestCallRepository_List_WithFilters(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	userID := uuid.New()
	status := "done"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "title", "duration_sec", "status", "metadata", "created_at", "updated_at",
	}).AddRow(uuid.New(), userID, "Only", 30, status, "{}", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls"`)).
		WillReturnRows(rows)

	list, total, err := repo.List(context.Background(), ports.ListCallsFilter{
		UserID: &userID,
		Status: &status,
	}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, list, 1)
}

func TestCallRepository_List_CountError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls"`)).
		WillReturnError(assert.AnError)

	_, _, err := repo.List(context.Background(), ports.ListCallsFilter{}, 1, 10)
	require.Error(t, err)
}

func TestCallRepository_List_SelectError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "calls"`)).
		WillReturnError(assert.AnError)

	_, _, err := repo.List(context.Background(), ports.ListCallsFilter{}, 1, 10)
	require.Error(t, err)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestCallRepository_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	call := &entities.Call{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Title:  "Updated",
		Status: "done",
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "calls"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.Update(context.Background(), call)
	require.NoError(t, err)
	assert.Equal(t, call.ID, result.ID)
}

func TestCallRepository_Update_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	call := &entities.Call{ID: uuid.New(), UserID: uuid.New(), Title: "X", Status: "pending"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "calls"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Update(context.Background(), call)
	require.Error(t, err)
}

// ─── SoftDelete ───────────────────────────────────────────────────────────────

func TestCallRepository_SoftDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "calls" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SoftDelete(context.Background(), id)
	require.NoError(t, err)
}

func TestCallRepository_SoftDelete_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "calls" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected → not found
	mock.ExpectCommit()

	err := repo.SoftDelete(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestCallRepository_SoftDelete_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewCallRepository(db)

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "calls" WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	err := repo.SoftDelete(context.Background(), id)
	require.Error(t, err)
}
