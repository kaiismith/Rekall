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

// ─── InvitationRepository ─────────────────────────────────────────────────────

func TestNewInvitationRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewInvitationRepository(db))
}

func TestInvitation_Upsert_InsertsWhenAbsent(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	orgID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations" WHERE org_id = $1 AND email = $2 AND accepted_at IS NULL`)).
		WithArgs(orgID, "new@example.com", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "invitations"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	inv := &entities.Invitation{
		OrgID:     orgID,
		Email:     "new@example.com",
		TokenHash: "hash",
		Role:      "member",
		InvitedBy: uuid.New(),
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.Upsert(context.Background(), inv))
}

func TestInvitation_Upsert_UpdatesExisting(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	orgID := uuid.New()
	existingID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations" WHERE org_id = $1 AND email = $2 AND accepted_at IS NULL`)).
		WithArgs(orgID, "existing@example.com", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "email", "token_hash", "role", "invited_by", "expires_at", "created_at",
		}).AddRow(existingID, orgID, "existing@example.com", "old-hash", "member", uuid.New(), time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "invitations"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	inv := &entities.Invitation{
		OrgID:     orgID,
		Email:     "existing@example.com",
		TokenHash: "new-hash",
		Role:      "admin",
		InvitedBy: uuid.New(),
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.Upsert(context.Background(), inv))
}

func TestInvitation_Upsert_LookupError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	err := repo.Upsert(context.Background(), &entities.Invitation{
		OrgID: uuid.New(), Email: "x@y.z", ExpiresAt: time.Now(),
	})
	require.Error(t, err)
}

func TestInvitation_GetByTokenHash_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations" WHERE token_hash = $1`)).
		WithArgs("hash", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "email", "token_hash", "role", "invited_by", "expires_at", "created_at",
		}).AddRow(uuid.New(), uuid.New(), "a@b.com", "hash", "member", uuid.New(), time.Now().Add(time.Hour), time.Now()))

	inv, err := repo.GetByTokenHash(context.Background(), "hash")
	require.NoError(t, err)
	assert.Equal(t, "hash", inv.TokenHash)
}

func TestInvitation_GetByTokenHash_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations"`)).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByTokenHash(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestInvitation_GetPendingByOrgAndEmail_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	orgID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations" WHERE org_id = $1 AND email = $2 AND accepted_at IS NULL`)).
		WithArgs(orgID, "a@b.com", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "email", "token_hash", "role", "invited_by", "expires_at", "created_at",
		}).AddRow(uuid.New(), orgID, "a@b.com", "hash", "member", uuid.New(), time.Now().Add(time.Hour), time.Now()))

	inv, err := repo.GetPendingByOrgAndEmail(context.Background(), orgID, "a@b.com")
	require.NoError(t, err)
	assert.Equal(t, "a@b.com", inv.Email)
}

func TestInvitation_GetPendingByOrgAndEmail_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	orgID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invitations"`)).
		WithArgs(orgID, "missing@x.com", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetPendingByOrgAndEmail(context.Background(), orgID, "missing@x.com")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestInvitation_MarkAccepted_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "invitations" SET "accepted_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.MarkAccepted(context.Background(), "hash"))
}

func TestInvitation_MarkAccepted_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "invitations" SET "accepted_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.MarkAccepted(context.Background(), "hash")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestInvitation_MarkAccepted_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewInvitationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "invitations" SET "accepted_at"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	require.Error(t, repo.MarkAccepted(context.Background(), "hash"))
}

// ─── MeetingParticipantRepository ─────────────────────────────────────────────

func TestNewMeetingParticipantRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewMeetingParticipantRepository(db))
}

func TestParticipant_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "meeting_participants"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	p := &entities.MeetingParticipant{MeetingID: uuid.New(), UserID: uuid.New(), Role: "participant"}
	require.NoError(t, repo.Create(context.Background(), p))
}

func TestParticipant_GetByMeetingAndUser_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mID, uID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meeting_participants" WHERE meeting_id = $1 AND user_id = $2`)).
		WithArgs(mID, uID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "meeting_id", "user_id", "role", "created_at",
		}).AddRow(uuid.New(), mID, uID, "participant", time.Now()))

	p, err := repo.GetByMeetingAndUser(context.Background(), mID, uID)
	require.NoError(t, err)
	assert.Equal(t, mID, p.MeetingID)
}

func TestParticipant_GetByMeetingAndUser_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mID, uID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meeting_participants"`)).
		WithArgs(mID, uID, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByMeetingAndUser(context.Background(), mID, uID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestParticipant_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "meeting_participants"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	p := &entities.MeetingParticipant{ID: uuid.New(), MeetingID: uuid.New(), UserID: uuid.New(), Role: "participant"}
	require.NoError(t, repo.Update(context.Background(), p))
}

func TestParticipant_ListActive_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "meeting_participants" WHERE meeting_id = $1 AND left_at IS NULL`)).
		WithArgs(mID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "meeting_id", "user_id", "role", "created_at",
		}).AddRow(uuid.New(), mID, uuid.New(), "participant", time.Now()))

	list, err := repo.ListActive(context.Background(), mID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestParticipant_CountActive_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "meeting_participants" WHERE meeting_id = $1 AND left_at IS NULL`)).
		WithArgs(mID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	n, err := repo.CountActive(context.Background(), mID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestParticipant_MarkAllLeft_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewMeetingParticipantRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "meeting_participants" SET "left_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	require.NoError(t, repo.MarkAllLeft(context.Background(), uuid.New()))
}
