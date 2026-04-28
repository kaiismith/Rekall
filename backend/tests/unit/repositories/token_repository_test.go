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
	"github.com/rekall/backend/internal/infrastructure/repositories"
	apperr "github.com/rekall/backend/pkg/errors"
)

func TestNewTokenRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewTokenRepository(db))
}

// ─── Refresh tokens ───────────────────────────────────────────────────────────

func TestToken_CreateRefreshToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "refresh_tokens"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	tok := &entities.RefreshToken{
		UserID:    uuid.New(),
		TokenHash: "hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreateRefreshToken(context.Background(), tok))
}

func TestToken_GetRefreshToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "refresh_tokens" WHERE token_hash = $1`)).
		WithArgs("hash", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at",
		}).AddRow(uuid.New(), uuid.New(), "hash", time.Now().Add(time.Hour), time.Now()))

	tok, err := repo.GetRefreshToken(context.Background(), "hash")
	require.NoError(t, err)
	assert.Equal(t, "hash", tok.TokenHash)
}

func TestToken_GetRefreshToken_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "refresh_tokens"`)).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetRefreshToken(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestToken_RevokeRefreshToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "refresh_tokens" SET "revoked_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.RevokeRefreshToken(context.Background(), "hash"))
}

func TestToken_RevokeAllRefreshTokens_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "refresh_tokens" SET "revoked_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	require.NoError(t, repo.RevokeAllRefreshTokens(context.Background(), uuid.New()))
}

// ─── Email verification tokens ────────────────────────────────────────────────

func TestToken_CreateVerificationToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "email_verification_tokens"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	tok := &entities.EmailVerificationToken{
		UserID: uuid.New(), TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreateVerificationToken(context.Background(), tok))
}

func TestToken_GetVerificationToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "email_verification_tokens" WHERE token_hash = $1`)).
		WithArgs("h", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at",
		}).AddRow(uuid.New(), uuid.New(), "h", time.Now().Add(time.Hour), time.Now()))

	tok, err := repo.GetVerificationToken(context.Background(), "h")
	require.NoError(t, err)
	assert.Equal(t, "h", tok.TokenHash)
}

func TestToken_GetVerificationToken_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "email_verification_tokens"`)).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetVerificationToken(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestToken_MarkVerificationTokenUsed_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "email_verification_tokens" SET "used_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.MarkVerificationTokenUsed(context.Background(), "hash"))
}

func TestToken_InvalidatePendingVerificationTokens_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "email_verification_tokens"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.InvalidatePendingVerificationTokens(context.Background(), uuid.New()))
}

// ─── Password reset tokens ────────────────────────────────────────────────────

func TestToken_CreatePasswordResetToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "password_reset_tokens"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	tok := &entities.PasswordResetToken{
		UserID: uuid.New(), TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreatePasswordResetToken(context.Background(), tok))
}

func TestToken_GetPasswordResetToken_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "password_reset_tokens" WHERE token_hash = $1`)).
		WithArgs("h", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at",
		}).AddRow(uuid.New(), uuid.New(), "h", time.Now().Add(time.Hour), time.Now()))

	tok, err := repo.GetPasswordResetToken(context.Background(), "h")
	require.NoError(t, err)
	assert.Equal(t, "h", tok.TokenHash)
}

func TestToken_GetPasswordResetToken_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "password_reset_tokens"`)).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetPasswordResetToken(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestToken_MarkPasswordResetTokenUsed_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "password_reset_tokens" SET "used_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.MarkPasswordResetTokenUsed(context.Background(), "hash"))
}

func TestToken_InvalidatePendingPasswordResetTokens_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "password_reset_tokens"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.InvalidatePendingPasswordResetTokens(context.Background(), uuid.New()))
}
