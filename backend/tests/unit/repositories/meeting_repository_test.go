package repositories_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	apperr "github.com/rekall/backend/pkg/errors"
)

func meetingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "code", "title", "type", "host_id", "status", "max_participants",
		"started_at", "ended_at", "created_at", "updated_at",
	})
}

// ─── NewMeetingRepository ─────────────────────────────────────────────────────

func TestNewMeetingRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewMeetingRepository(db))
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestMeeting_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "meetings"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	m := &entities.Meeting{Code: "abc", HostID: uuid.New(), Type: "open", Status: "waiting", MaxParticipants: 50}
	require.NoError(t, repo.Create(context.Background(), m))
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func TestMeeting_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE id = $1`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnRows(meetingRows().AddRow(id, "c", "T", "open", uuid.New(), "waiting", 50, nil, nil, time.Now(), time.Now()))

	m, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, m.ID)
}

func TestMeeting_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

// ─── GetByCode ────────────────────────────────────────────────────────────────

func TestMeeting_GetByCode_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE code = $1`)).
		WithArgs("abc", sqlmock.AnyArg()).
		WillReturnRows(meetingRows().AddRow(uuid.New(), "abc", "T", "open", uuid.New(), "waiting", 50, nil, nil, time.Now(), time.Now()))

	m, err := repo.GetByCode(context.Background(), "abc")
	require.NoError(t, err)
	assert.Equal(t, "abc", m.Code)
}

func TestMeeting_GetByCode_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings"`)).
		WithArgs("xyz", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByCode(context.Background(), "xyz")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

// ─── Update / Delete ──────────────────────────────────────────────────────────

func TestMeeting_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "meetings"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	m := &entities.Meeting{ID: uuid.New(), Code: "c", HostID: uuid.New(), Type: "open", Status: "ended", MaxParticipants: 50}
	require.NoError(t, repo.Update(context.Background(), m))
}

func TestMeeting_Delete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "meetings"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Delete(context.Background(), uuid.New()))
}

// ─── ListByHost ───────────────────────────────────────────────────────────────

func TestMeeting_ListByHost_NoStatus(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	hostID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1`)).
		WithArgs(hostID).
		WillReturnRows(meetingRows().AddRow(uuid.New(), "c", "T", "open", hostID, "waiting", 50, nil, nil, time.Now(), time.Now()))

	list, err := repo.ListByHost(context.Background(), hostID, "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMeeting_ListByHost_WithStatus(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	hostID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 AND status = $2`)).
		WithArgs(hostID, "active").
		WillReturnRows(meetingRows())

	list, err := repo.ListByHost(context.Background(), hostID, "active")
	require.NoError(t, err)
	assert.Empty(t, list)
}

// ─── CountActiveByHost ────────────────────────────────────────────────────────

func TestMeeting_CountActiveByHost_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	hostID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "meetings" WHERE host_id = $1 AND status IN`)).
		WithArgs(hostID, "waiting", "active").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	n, err := repo.CountActiveByHost(context.Background(), hostID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

// ─── FindStaleWaiting / FindStaleActive / FindActiveWithNoParticipants ─────────

func TestMeeting_FindStaleWaiting_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE status = $1 AND created_at < $2`)).
		WithArgs("waiting", sqlmock.AnyArg()).
		WillReturnRows(meetingRows())

	list, err := repo.FindStaleWaiting(context.Background(), time.Hour)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_FindStaleActive_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE status = $1 AND started_at < $2`)).
		WithArgs("active", sqlmock.AnyArg()).
		WillReturnRows(meetingRows())

	list, err := repo.FindStaleActive(context.Background(), 8*time.Hour)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_FindActiveWithNoParticipants_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE status = $1 AND id NOT IN`)).
		WithArgs("active").
		WillReturnRows(meetingRows())

	list, err := repo.FindActiveWithNoParticipants(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}

// ─── ListByUser ───────────────────────────────────────────────────────────────

func TestMeeting_ListByUser_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
		WithArgs(userID, userID).
		WillReturnRows(meetingRows())

	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{})
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_ListByUser_WithResultsAndPreviews(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	meetingID := uuid.New()
	started := time.Now().Add(-time.Hour)
	ended := time.Now()

	// Primary query
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
		WithArgs(userID, userID).
		WillReturnRows(meetingRows().AddRow(
			meetingID, "abc", "Test", "open", userID, "ended", 50, started, ended, time.Now(), time.Now(),
		))

	// Preview query
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT mp.meeting_id, u.id AS user_id, u.full_name FROM meeting_participants`)).
		WillReturnRows(sqlmock.NewRows([]string{"meeting_id", "user_id", "full_name"}).
			AddRow(meetingID, uuid.New(), "Alice Smith").
			AddRow(meetingID, uuid.New(), "Bob Jones"))

	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, meetingID, list[0].Meeting.ID)
	assert.Len(t, list[0].ParticipantPreviews, 2)
	require.NotNil(t, list[0].DurationSeconds)
	assert.Greater(t, *list[0].DurationSeconds, int64(3000))
}

func TestMeeting_ListByUser_StatusInProgress(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	status := "in_progress"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE (host_id = $1 OR id IN`)).
		WithArgs(userID, userID, "waiting", "active").
		WillReturnRows(meetingRows())

	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{Status: &status})
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_ListByUser_StatusComplete(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	status := "complete"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE (host_id = $1 OR id IN`)).
		WithArgs(userID, userID, "ended").
		WillReturnRows(meetingRows())

	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{Status: &status})
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_ListByUser_StatusProcessing_EmptyShortCircuit(t *testing.T) {
	db, _ := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	// "processing" and "failed" short-circuit to empty without any DB query.
	userID := uuid.New()
	status := "processing"
	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{Status: &status})
	require.NoError(t, err)
	assert.Empty(t, list)

	status = "failed"
	list, err = repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{Status: &status})
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMeeting_ListByUser_QueryError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
		WithArgs(userID, userID).
		WillReturnError(assert.AnError)

	_, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{})
	require.Error(t, err)
}

func TestMeeting_ListByUser_PreviewError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
		WithArgs(userID, userID).
		WillReturnRows(meetingRows().AddRow(
			uuid.New(), "abc", "Test", "open", userID, "waiting", 50, nil, nil, time.Now(), time.Now(),
		))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT mp.meeting_id`)).
		WillReturnError(assert.AnError)

	_, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{})
	require.Error(t, err)
}

// ─── listSortExpr / meetingInitials (exercised via ListByUser) ────────────────

func TestMeeting_ListByUser_SortVariants(t *testing.T) {
	sorts := []string{"created_at_asc", "duration_desc", "duration_asc", "title_asc", "title_desc", "unknown-sort"}
	for _, s := range sorts {
		t.Run("sort="+s, func(t *testing.T) {
			db, mock := newMockDB(t)
			repo := repositories.NewMeetingRepository(db)

			userID := uuid.New()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
				WithArgs(userID, userID).
				WillReturnRows(meetingRows())

			_, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{Sort: s})
			require.NoError(t, err)
		})
	}
}

func TestMeeting_ListByUser_PreviewsInitials(t *testing.T) {
	// Exercises meetingInitials with: empty name, single-word, two-word, three-word.
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingRepository(db)

	userID := uuid.New()
	meetingID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meetings" WHERE host_id = $1 OR id IN`)).
		WithArgs(userID, userID).
		WillReturnRows(meetingRows().AddRow(
			meetingID, "abc", "T", "open", userID, "waiting", 50, nil, nil, time.Now(), time.Now(),
		))

	// Four participants with distinct name shapes; only first 3 are included
	// in the preview (cap), exercising the "len < 3" guard.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT mp.meeting_id`)).
		WillReturnRows(sqlmock.NewRows([]string{"meeting_id", "user_id", "full_name"}).
			AddRow(meetingID, uuid.New(), "").                 // empty → "?"
			AddRow(meetingID, uuid.New(), "Cher").             // single-word → "C"
			AddRow(meetingID, uuid.New(), "Mary Jane Watson"). // three-word → "MW"
			AddRow(meetingID, uuid.New(), "Fourth One"))       // capped out

	list, err := repo.ListByUser(context.Background(), userID, ports.ListMeetingsFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].ParticipantPreviews, 3) // capped at 3
	assert.Equal(t, "?", list[0].ParticipantPreviews[0].Initials)
	assert.Equal(t, "C", list[0].ParticipantPreviews[1].Initials)
	assert.Equal(t, "MW", list[0].ParticipantPreviews[2].Initials)
}
