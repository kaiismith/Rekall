package config_test

import (
	"testing"
	"time"

	"github.com/rekall/backend/pkg/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setRequired plants the minimum env vars needed to pass validateRequired.
// t.Setenv restores each value automatically after the test.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DB_USER", "rekall")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "rekall_db")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_FROM", "noreply@rekall.local")
}

// reset clears viper's global state between tests so defaults and overrides
// from one test cannot bleed into the next.
func reset() { viper.Reset() }

// ─── Defaults ─────────────────────────────────────────────────────────────────

func TestConfig_Defaults(t *testing.T) {
	reset()
	setRequired(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	// Server
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Env)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)
	assert.False(t, cfg.Server.SwaggerEnabled)

	// Database
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	// Auth
	assert.Equal(t, "rekall", cfg.Auth.JWTIssuer)
	assert.Equal(t, 15*time.Minute, cfg.Auth.AccessTokenTTL)
	assert.Equal(t, 168*time.Hour, cfg.Auth.RefreshTokenTTL)

	// Meeting
	assert.Equal(t, 10*time.Minute, cfg.Meeting.WaitingTimeout)
	assert.Equal(t, 8*time.Hour, cfg.Meeting.MaxDuration)
	assert.Equal(t, 5*time.Minute, cfg.Meeting.CleanupInterval)

	// Logger
	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)
}

// ─── Overrides ────────────────────────────────────────────────────────────────

func TestConfig_MeetingOverrides(t *testing.T) {
	reset()
	setRequired(t)
	t.Setenv("MEETING_WAITING_TIMEOUT", "30m")
	t.Setenv("MEETING_MAX_DURATION", "12h")
	t.Setenv("MEETING_CLEANUP_INTERVAL", "2m")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 30*time.Minute, cfg.Meeting.WaitingTimeout)
	assert.Equal(t, 12*time.Hour, cfg.Meeting.MaxDuration)
	assert.Equal(t, 2*time.Minute, cfg.Meeting.CleanupInterval)
}

func TestConfig_ServerOverrides(t *testing.T) {
	reset()
	setRequired(t)
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_ENV", "production")
	t.Setenv("SWAGGER_ENABLED", "true")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "production", cfg.Server.Env)
	assert.True(t, cfg.Server.SwaggerEnabled)
}

// ─── Required variable validation ────────────────────────────────────────────

func TestConfig_MissingRequired_ReturnsError(t *testing.T) {
	// SMTP_FROM is excluded: it has a viper default ("noreply@rekall.local"),
	// so setting the env var to "" causes viper to return the default rather
	// than "", making the required check impossible to trigger via t.Setenv.
	cases := []struct {
		omit string
	}{
		{"DB_USER"},
		{"DB_PASSWORD"},
		{"DB_NAME"},
		{"JWT_SECRET"},
		{"SMTP_HOST"},
	}

	for _, tc := range cases {
		t.Run("missing_"+tc.omit, func(t *testing.T) {
			reset()
			setRequired(t)
			// Blank out the one required var being tested.
			t.Setenv(tc.omit, "")

			_, err := config.Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.omit)
		})
	}
}

// ─── Invalid duration strings ─────────────────────────────────────────────────

func TestConfig_InvalidDuration_ReturnsError(t *testing.T) {
	cases := []struct {
		key string
	}{
		{"SERVER_READ_TIMEOUT"},
		{"SERVER_WRITE_TIMEOUT"},
		{"SERVER_IDLE_TIMEOUT"},
		{"DB_CONN_MAX_LIFETIME"},
		{"JWT_ACCESS_TTL"},
		{"JWT_REFRESH_TTL"},
		{"PASSWORD_RESET_TTL"},
		{"EMAIL_VERIFY_TTL"},
		{"INVITATION_TTL"},
		{"MEETING_WAITING_TIMEOUT"},
		{"MEETING_MAX_DURATION"},
		{"MEETING_CLEANUP_INTERVAL"},
	}

	for _, tc := range cases {
		t.Run("invalid_"+tc.key, func(t *testing.T) {
			reset()
			setRequired(t)
			t.Setenv(tc.key, "not-a-duration")

			_, err := config.Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.key)
		})
	}
}

// ─── CORS origin splitting ────────────────────────────────────────────────────

func TestConfig_CORSOrigins_Split(t *testing.T) {
	reset()
	setRequired(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://app.rekall.io, https://admin.rekall.io , http://localhost:3000")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, []string{
		"http://app.rekall.io",
		"https://admin.rekall.io",
		"http://localhost:3000",
	}, cfg.CORS.AllowedOrigins)
}

func TestConfig_CORSOrigins_Default(t *testing.T) {
	reset()
	setRequired(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Contains(t, cfg.CORS.AllowedOrigins, "http://localhost:3000")
	assert.Contains(t, cfg.CORS.AllowedOrigins, "http://localhost:5173")
}

// ─── DatabaseConfig.DSN ───────────────────────────────────────────────────────

func TestConfig_DSN_Format(t *testing.T) {
	reset()
	setRequired(t)
	t.Setenv("DB_HOST", "db.rekall.io")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_SSL_MODE", "require")

	cfg, err := config.Load()
	require.NoError(t, err)

	dsn := cfg.Database.DSN()
	assert.Contains(t, dsn, "host=db.rekall.io")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "user=rekall")
	assert.Contains(t, dsn, "dbname=rekall_db")
	assert.Contains(t, dsn, "sslmode=require")
	assert.Contains(t, dsn, "TimeZone=UTC")
}

// ─── ServerConfig.IsDevelopment ──────────────────────────────────────────────

func TestConfig_IsDevelopment(t *testing.T) {
	reset()
	setRequired(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.True(t, cfg.Server.IsDevelopment()) // default env = "development"

	reset()
	setRequired(t)
	t.Setenv("SERVER_ENV", "production")

	cfg, err = config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.Server.IsDevelopment())
}
