package database

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rekall/backend/pkg/config"
)

// New opens a GORM database connection backed by PostgreSQL and configures
// the underlying sql.DB connection pool from the provided config.
// It pings the database before returning to surface connection issues early.
func New(cfg config.DatabaseConfig, isDevelopment bool) (*gorm.DB, error) {
	logLevel := logger.Error
	if isDevelopment {
		logLevel = logger.Info
	}

	// `record not found` is a normal control-flow signal in this codebase
	// (e.g. admitDirect's "is there already a meeting_participants row?"
	// pre-check before INSERT). Suppress the noisy log line — handlers still
	// receive `gorm.ErrRecordNotFound` and switch on it explicitly.
	gormLogger := logger.New(
		stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              true, // cache prepared statements
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("database: failed to open connection: %w", err)
	}

	// Configure the underlying connection pool.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := Ping(context.Background(), db); err != nil {
		return nil, err
	}

	return db, nil
}

// Ping verifies the database is reachable. Used for readiness checks.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: failed to get sql.DB for ping: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database: ping failed: %w", err)
	}
	return nil
}
