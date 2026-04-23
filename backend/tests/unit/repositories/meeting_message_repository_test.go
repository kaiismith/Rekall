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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chatMessageRows(ids ...uuid.UUID) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"id", "meeting_id", "user_id", "body", "sent_at", "deleted_at"})
	base := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)
	for i, id := range ids {
		rows.AddRow(id, uuid.New(), uuid.New(), "msg", base.Add(time.Duration(i)*time.Second), nil)
	}
	return rows
}

func TestNewMeetingMessageRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewMeetingMessageRepository(db))
}

func TestMeetingMessage_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "meeting_messages"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "sent_at"}).
			AddRow(uuid.New(), time.Now()))
	mock.ExpectCommit()

	msg := &entities.MeetingMessage{
		MeetingID: uuid.New(),
		UserID:    uuid.New(),
		Body:      "hello",
	}
	require.NoError(t, repo.Create(context.Background(), msg))
}

func TestMeetingMessage_ListByMeeting_DefaultLimit_EmptyResult(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	meetingID := uuid.New()
	// Default limit 50 → query uses LIMIT 51 to detect has_more.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "meeting_messages" WHERE meeting_id = $1 AND deleted_at IS NULL ORDER BY sent_at DESC LIMIT $2`,
	)).WithArgs(meetingID, 51).
		WillReturnRows(chatMessageRows())

	msgs, hasMore, err := repo.ListByMeeting(context.Background(), meetingID, nil, 0)
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Empty(t, msgs)
}

func TestMeetingMessage_ListByMeeting_HasMoreDetection(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	meetingID := uuid.New()
	// limit=3 → query fetches 4 rows; the 4th triggers has_more=true.
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "meeting_messages" WHERE meeting_id = $1 AND deleted_at IS NULL ORDER BY sent_at DESC LIMIT $2`,
	)).WithArgs(meetingID, 4).
		WillReturnRows(chatMessageRows(ids...))

	msgs, hasMore, err := repo.ListByMeeting(context.Background(), meetingID, nil, 3)
	require.NoError(t, err)
	assert.True(t, hasMore)
	assert.Len(t, msgs, 3)
}

func TestMeetingMessage_ListByMeeting_LimitClampedToMax(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	meetingID := uuid.New()
	// Request 1000 → clamped to 100; query LIMIT 101.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "meeting_messages" WHERE meeting_id = $1 AND deleted_at IS NULL ORDER BY sent_at DESC LIMIT $2`,
	)).WithArgs(meetingID, 101).
		WillReturnRows(chatMessageRows())

	_, _, err := repo.ListByMeeting(context.Background(), meetingID, nil, 1000)
	require.NoError(t, err)
}

func TestMeetingMessage_ListByMeeting_WithBeforeCursor(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	meetingID := uuid.New()
	before := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "meeting_messages" WHERE (meeting_id = $1 AND deleted_at IS NULL) AND sent_at < $2 ORDER BY sent_at DESC LIMIT $3`,
	)).WithArgs(meetingID, before, 51).
		WillReturnRows(chatMessageRows())

	_, _, err := repo.ListByMeeting(context.Background(), meetingID, &before, 50)
	require.NoError(t, err)
}

func TestMeetingMessage_ListByMeeting_ResultsReversedAscending(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingMessageRepository(db)

	meetingID := uuid.New()
	// Rows in DESC order from the query; repo must reverse to ASC.
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	base := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "meeting_id", "user_id", "body", "sent_at", "deleted_at"})
	rows.AddRow(id1, meetingID, uuid.New(), "newest", base.Add(2*time.Second), nil)
	rows.AddRow(id2, meetingID, uuid.New(), "middle", base.Add(1*time.Second), nil)
	rows.AddRow(id3, meetingID, uuid.New(), "oldest", base, nil)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "meeting_messages" WHERE meeting_id = $1 AND deleted_at IS NULL ORDER BY sent_at DESC LIMIT $2`,
	)).WithArgs(meetingID, 51).
		WillReturnRows(rows)

	msgs, _, err := repo.ListByMeeting(context.Background(), meetingID, nil, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	// ASC order on output.
	assert.Equal(t, "oldest", msgs[0].Body)
	assert.Equal(t, "middle", msgs[1].Body)
	assert.Equal(t, "newest", msgs[2].Body)
}
