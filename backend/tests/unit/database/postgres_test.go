package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rekall/backend/internal/infrastructure/database"
	"github.com/rekall/backend/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ─── Ping ─────────────────────────────────────────────────────────────────────

func TestPing_Succeeds(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, database.Ping(context.Background(), db))
}

func TestPing_FailsWhenConnClosed(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	_ = sqlDB.Close() // close the underlying conn first

	err = database.Ping(context.Background(), db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database:")
}

func TestPing_FailsWithCancelledContext(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = database.Ping(ctx, db)
	require.Error(t, err)
}

// ─── New ──────────────────────────────────────────────────────────────────────

// TestNew_UnreachableDSN exercises the error branch in New() when the DSN
// points at an unreachable Postgres. The connection attempt fails during
// gorm.Open's initial handshake OR during Ping.
func TestNew_UnreachableDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            "1", // reserved port, nothing listening
		User:            "nobody",
		Password:        "nopass",
		DBName:          "nodb",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
	}

	// Development mode — exercise the info log-level branch.
	_, err := database.New(cfg, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database:")
}

func TestNew_UnreachableDSN_ProductionLogLevel(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            "1",
		User:            "nobody",
		Password:        "nopass",
		DBName:          "nodb",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
	}

	// Production mode — exercise the default log-level branch (logger.Error).
	_, err := database.New(cfg, false)
	require.Error(t, err)
}
