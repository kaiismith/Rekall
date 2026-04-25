package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	CORS     CORSConfig
	Auth     AuthConfig
	SMTP     SMTPConfig
	Meeting  MeetingConfig
}

// MeetingConfig holds settings for the meeting feature.
type MeetingConfig struct {
	// WaitingTimeout is how long a meeting stays in 'waiting' before auto-cleanup.
	WaitingTimeout time.Duration
	// MaxDuration is the maximum wall-clock length of an active meeting.
	MaxDuration time.Duration
	// CleanupInterval controls how often the background cleanup job runs.
	CleanupInterval time.Duration
}

// AuthConfig holds JWT and token lifecycle settings.
type AuthConfig struct {
	JWTSecret        string
	JWTIssuer        string
	AppBaseURL       string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	PasswordResetTTL time.Duration
	EmailVerifyTTL   time.Duration
	InvitationTTL    time.Duration

	// PlatformAdminEmails is a list of lowercased emails that should be granted
	// the platform-level "admin" role on each server boot. The reconciler also
	// demotes any existing admins whose email is no longer in this list — so
	// the env var is the source of truth.
	PlatformAdminEmails []string
	// PlatformAdminBootstrapPwd, when non-empty, is used to first-run-create
	// any admin email that does not yet have a User record. Subsequent boots
	// do NOT re-apply the password — rotation goes through the standard reset
	// flow.
	PlatformAdminBootstrapPwd string
}

// SMTPConfig holds outgoing mail settings.
// For local development point at Mailpit (host=localhost, port=1025, no auth, TLS=false).
// For production use any SMTP relay (e.g. SendGrid: host=smtp.sendgrid.net, port=587, TLS=true).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port           string
	Env            string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	SwaggerEnabled bool // SWAGGER_ENABLED — serve /docs UI; default false
}

// IsDevelopment returns true when running in the development environment.
func (s ServerConfig) IsDevelopment() bool {
	return s.Env == "development"
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN builds a PostgreSQL DSN in key=value format accepted by GORM's postgres driver.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		d.Host, d.User, d.Password, d.DBName, d.Port, d.SSLMode,
	)
}

// LoggerConfig holds logging settings.
type LoggerConfig struct {
	Level  string // debug | info | warn | error
	Format string // json | console
}

// CORSConfig holds cross-origin resource sharing settings.
type CORSConfig struct {
	AllowedOrigins []string
}

// Load reads configuration from environment variables (and an optional .env file).
// It returns an error if any required variable is missing or a duration cannot be parsed.
func Load() (*Config, error) {
	// Best-effort: load .env for local development. Ignore missing file error.
	_ = godotenv.Load()

	viper.AutomaticEnv()

	setDefaults()

	if err := validateRequired(); err != nil {
		return nil, err
	}

	readTimeout, err := time.ParseDuration(viper.GetString("SERVER_READ_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(viper.GetString("SERVER_WRITE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT: %w", err)
	}
	idleTimeout, err := time.ParseDuration(viper.GetString("SERVER_IDLE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_IDLE_TIMEOUT: %w", err)
	}
	connMaxLifetime, err := time.ParseDuration(viper.GetString("DB_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
	}
	accessTTL, err := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	resetTTL, err := time.ParseDuration(viper.GetString("PASSWORD_RESET_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid PASSWORD_RESET_TTL: %w", err)
	}
	verifyTTL, err := time.ParseDuration(viper.GetString("EMAIL_VERIFY_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid EMAIL_VERIFY_TTL: %w", err)
	}
	inviteTTL, err := time.ParseDuration(viper.GetString("INVITATION_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid INVITATION_TTL: %w", err)
	}
	meetingWaitingTimeout, err := time.ParseDuration(viper.GetString("MEETING_WAITING_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid MEETING_WAITING_TIMEOUT: %w", err)
	}
	meetingMaxDuration, err := time.ParseDuration(viper.GetString("MEETING_MAX_DURATION"))
	if err != nil {
		return nil, fmt.Errorf("invalid MEETING_MAX_DURATION: %w", err)
	}
	meetingCleanupInterval, err := time.ParseDuration(viper.GetString("MEETING_CLEANUP_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid MEETING_CLEANUP_INTERVAL: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port:           viper.GetString("SERVER_PORT"),
			Env:            viper.GetString("SERVER_ENV"),
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			SwaggerEnabled: viper.GetBool("SWAGGER_ENABLED"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("DB_HOST"),
			Port:            viper.GetString("DB_PORT"),
			User:            viper.GetString("DB_USER"),
			Password:        viper.GetString("DB_PASSWORD"),
			DBName:          viper.GetString("DB_NAME"),
			SSLMode:         viper.GetString("DB_SSL_MODE"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: connMaxLifetime,
		},
		Logger: LoggerConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(viper.GetString("CORS_ALLOWED_ORIGINS")),
		},
		Auth: AuthConfig{
			JWTSecret:                 viper.GetString("JWT_SECRET"),
			JWTIssuer:                 viper.GetString("JWT_ISSUER"),
			AppBaseURL:                viper.GetString("APP_BASE_URL"),
			AccessTokenTTL:            accessTTL,
			RefreshTokenTTL:           refreshTTL,
			PasswordResetTTL:          resetTTL,
			EmailVerifyTTL:            verifyTTL,
			InvitationTTL:             inviteTTL,
			PlatformAdminEmails:       parseAdminEmails(viper.GetString("PLATFORM_ADMIN_EMAILS")),
			PlatformAdminBootstrapPwd: viper.GetString("PLATFORM_ADMIN_BOOTSTRAP_PASSWORD"),
		},
		SMTP: SMTPConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			Username: viper.GetString("SMTP_USER"),
			Password: viper.GetString("SMTP_PASSWORD"),
			From:     viper.GetString("SMTP_FROM"),
			UseTLS:   viper.GetBool("SMTP_TLS"),
		},
		Meeting: MeetingConfig{
			WaitingTimeout:  meetingWaitingTimeout,
			MaxDuration:     meetingMaxDuration,
			CleanupInterval: meetingCleanupInterval,
		},
	}, nil
}

func setDefaults() {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_ENV", "development")
	viper.SetDefault("SERVER_READ_TIMEOUT", "15s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "15s")
	viper.SetDefault("SERVER_IDLE_TIMEOUT", "60s")

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "5m")

	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	viper.SetDefault("SWAGGER_ENABLED", false)

	viper.SetDefault("JWT_ISSUER", "rekall")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")
	viper.SetDefault("PASSWORD_RESET_TTL", "1h")
	viper.SetDefault("EMAIL_VERIFY_TTL", "24h")
	viper.SetDefault("INVITATION_TTL", "168h")
	viper.SetDefault("APP_BASE_URL", "http://localhost:5173")

	viper.SetDefault("MEETING_WAITING_TIMEOUT", "10m")
	viper.SetDefault("MEETING_MAX_DURATION", "8h")
	viper.SetDefault("MEETING_CLEANUP_INTERVAL", "5m")

	viper.SetDefault("SMTP_PORT", 1025)
	viper.SetDefault("SMTP_FROM", "noreply@rekall.local")
	viper.SetDefault("SMTP_TLS", false)
}

func validateRequired() error {
	required := []string{"DB_USER", "DB_PASSWORD", "DB_NAME", "JWT_SECRET", "SMTP_HOST", "SMTP_FROM"}
	for _, key := range required {
		if viper.GetString(key) == "" {
			return fmt.Errorf("required environment variable %q is not set", key)
		}
	}
	return nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseAdminEmails parses the comma-separated PLATFORM_ADMIN_EMAILS value into
// a deduplicated, trimmed, lowercased list. Empty input yields an empty slice
// (no admins). The reconciler treats this slice as the source of truth — any
// user whose email is in it gets role=admin on boot, and any current admin
// whose email is NOT in it gets demoted to member.
func parseAdminEmails(s string) []string {
	parts := strings.Split(s, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		e := strings.ToLower(strings.TrimSpace(p))
		if e == "" {
			continue
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}
