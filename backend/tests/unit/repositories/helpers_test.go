package repositories_test

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	repohelpers "github.com/rekall/backend/internal/infrastructure/repositories/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ApplyCallFilter ──────────────────────────────────────────────────────────

func TestApplyCallFilter_NoFields(t *testing.T) {
	db, mock := newMockDB(t)

	// Without any filter fields, the query should have no WHERE clauses.
	filtered := repohelpers.ApplyCallFilter(db.Model(&entities.Call{}), ports.ListCallsFilter{})
	require.NotNil(t, filtered)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	var count int64
	err := filtered.Count(&count).Error
	require.NoError(t, err)
}

func TestApplyCallFilter_UserIDOnly(t *testing.T) {
	db, mock := newMockDB(t)

	userID := uuid.New()
	filtered := repohelpers.ApplyCallFilter(db.Model(&entities.Call{}), ports.ListCallsFilter{
		UserID: &userID,
	})
	require.NotNil(t, filtered)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls" WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	var count int64
	err := filtered.Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestApplyCallFilter_StatusOnly(t *testing.T) {
	db, mock := newMockDB(t)

	status := "done"
	filtered := repohelpers.ApplyCallFilter(db.Model(&entities.Call{}), ports.ListCallsFilter{
		Status: &status,
	})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls" WHERE status = $1`)).
		WithArgs(status).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	var count int64
	err := filtered.Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestApplyCallFilter_BothFields(t *testing.T) {
	db, mock := newMockDB(t)

	userID := uuid.New()
	status := "done"
	filtered := repohelpers.ApplyCallFilter(db.Model(&entities.Call{}), ports.ListCallsFilter{
		UserID: &userID,
		Status: &status,
	})

	// Both WHERE clauses chained.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "calls" WHERE user_id = $1 AND status = $2`)).
		WithArgs(userID, status).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	var count int64
	err := filtered.Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
