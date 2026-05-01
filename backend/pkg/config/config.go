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
	ASR      ASRConfig
	Kat      KatConfig
}

// KatConfig holds the env-bound knobs for the Kat live-notes scheduler. See
// .kiro/specs/kat-live-notes/requirements.md (Requirement 7) for the full
// contract. Notes are NOT persisted; this config only governs the in-memory
// ring buffer + the Foundry adapter selection.
type KatConfig struct {
	Enabled bool

	// Provider selects the LLM backend: "foundry" (Azure AI Foundry) or
	// "openai" (OpenAI public API or any OpenAI-wire-compatible endpoint).
	// Empty defaults to "foundry" for backward compatibility.
	Provider string

	// Foundry endpoint + deployment + auth selection.
	FoundryEndpoint   string
	FoundryDeployment string
	FoundryAPIVersion string
	// FoundryAPIKey: empty triggers DefaultAzureCredential. Treated as a
	// secret — never logged.
	FoundryAPIKey string

	// OpenAI API key + model + optional base URL. BaseURL is empty for
	// api.openai.com; set it for OpenAI-wire-compatible providers (vLLM,
	// LM Studio, third-party proxies).
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string

	// Per-call deadline applied to whichever provider is selected.
	FoundryRequestTimeout time.Duration

	// Sliding-window scheduler.
	WindowSeconds          int
	StepSeconds            int
	MinNewSegments         int
	MaxConcurrentRuns      int
	CooldownAfterErrorSecs int

	// In-memory ring buffer for late-join replay.
	RingBufferCapacity int

	// Prompt versioning.
	PromptVersion string
}

// ASRConfig holds the Go-side knobs for the standalone C++ ASR service.
// FeatureEnabled is the master switch — when false, /calls/:id/asr-session
// returns 503 ASR_NOT_CONFIGURED and the gRPC client is not dialled at boot.
type ASRConfig struct {
	FeatureEnabled bool

	GRPCAddr        string
	TokenSecret     string
	TokenIssuer     string
	TokenAudience   string
	TokenDefaultTTL time.Duration
	TokenMaxTTL     time.Duration

	// Browser-facing WebSocket base (e.g. wss://asr.rekall.example).
	WSURLBase string

	// mTLS material for the gRPC channel (production).
	GRPCClientCert string
	GRPCClientKey  string
	GRPCServerCA   string
	GRPCServerName string

	// Circuit breaker.
	CircuitBreakerFailures int
	CircuitBreakerCooldown time.Duration
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
	asrTokenDefaultTTL, err := time.ParseDuration(viper.GetString("ASR_TOKEN_DEFAULT_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid ASR_TOKEN_DEFAULT_TTL: %w", err)
	}
	asrTokenMaxTTL, err := time.ParseDuration(viper.GetString("ASR_TOKEN_MAX_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid ASR_TOKEN_MAX_TTL: %w", err)
	}
	asrCircuitCooldown, err := time.ParseDuration(viper.GetString("ASR_CIRCUIT_BREAKER_COOLDOWN"))
	if err != nil {
		return nil, fmt.Errorf("invalid ASR_CIRCUIT_BREAKER_COOLDOWN: %w", err)
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
		ASR: ASRConfig{
			FeatureEnabled:         viper.GetBool("ASR_FEATURE_ENABLED"),
			GRPCAddr:               viper.GetString("ASR_GRPC_ADDR"),
			TokenSecret:            viper.GetString("ASR_TOKEN_SECRET"),
			TokenIssuer:            viper.GetString("ASR_TOKEN_ISSUER"),
			TokenAudience:          viper.GetString("ASR_TOKEN_AUDIENCE"),
			TokenDefaultTTL:        asrTokenDefaultTTL,
			TokenMaxTTL:            asrTokenMaxTTL,
			WSURLBase:              viper.GetString("ASR_WS_URL_BASE"),
			GRPCClientCert:         viper.GetString("ASR_GRPC_CLIENT_CERT"),
			GRPCClientKey:          viper.GetString("ASR_GRPC_CLIENT_KEY"),
			GRPCServerCA:           viper.GetString("ASR_GRPC_SERVER_CA"),
			GRPCServerName:         viper.GetString("ASR_GRPC_SERVER_NAME"),
			CircuitBreakerFailures: viper.GetInt("ASR_CIRCUIT_BREAKER_FAILURES"),
			CircuitBreakerCooldown: asrCircuitCooldown,
		},
		Kat: KatConfig{
			Enabled:                viper.GetBool("KAT_ENABLED"),
			Provider:               viper.GetString("KAT_PROVIDER"),
			FoundryEndpoint:        viper.GetString("KAT_FOUNDRY_ENDPOINT"),
			FoundryDeployment:      viper.GetString("KAT_FOUNDRY_DEPLOYMENT"),
			FoundryAPIVersion:      viper.GetString("KAT_FOUNDRY_API_VERSION"),
			FoundryAPIKey:          viper.GetString("KAT_FOUNDRY_API_KEY"),
			OpenAIAPIKey:           viper.GetString("KAT_OPENAI_API_KEY"),
			OpenAIBaseURL:          viper.GetString("KAT_OPENAI_BASE_URL"),
			OpenAIModel:            viper.GetString("KAT_OPENAI_MODEL"),
			FoundryRequestTimeout:  time.Duration(viper.GetInt("KAT_FOUNDRY_REQUEST_TIMEOUT_MS")) * time.Millisecond,
			WindowSeconds:          viper.GetInt("KAT_WINDOW_SECONDS"),
			StepSeconds:            viper.GetInt("KAT_STEP_SECONDS"),
			MinNewSegments:         viper.GetInt("KAT_MIN_NEW_SEGMENTS"),
			MaxConcurrentRuns:      viper.GetInt("KAT_MAX_CONCURRENT_RUNS"),
			CooldownAfterErrorSecs: viper.GetInt("KAT_COOLDOWN_AFTER_ERROR_SECONDS"),
			RingBufferCapacity:     viper.GetInt("KAT_RING_BUFFER_CAPACITY"),
			PromptVersion:          viper.GetString("KAT_PROMPT_VERSION"),
		},
	}, nil
}

// Validate enforces cross-field constraints. Called after Load() in main.go.
// Today only the ASR section has cross-field rules; the rest is validated
// by their respective field-level type checks.
func (c *Config) Validate() error {
	if c.ASR.FeatureEnabled {
		if c.ASR.GRPCAddr == "" {
			return fmt.Errorf("ASR_FEATURE_ENABLED=true requires ASR_GRPC_ADDR")
		}
		if len(c.ASR.TokenSecret) < 32 {
			return fmt.Errorf("ASR_FEATURE_ENABLED=true requires ASR_TOKEN_SECRET ≥ 32 bytes")
		}
		if c.ASR.TokenMaxTTL > 5*time.Minute {
			return fmt.Errorf("ASR_TOKEN_MAX_TTL must be ≤ 5m")
		}
		if c.ASR.TokenDefaultTTL > c.ASR.TokenMaxTTL {
			return fmt.Errorf("ASR_TOKEN_DEFAULT_TTL must be ≤ ASR_TOKEN_MAX_TTL")
		}
		if !c.Server.IsDevelopment() {
			if c.ASR.GRPCClientCert == "" || c.ASR.GRPCClientKey == "" || c.ASR.GRPCServerCA == "" {
				return fmt.Errorf("non-dev environments require ASR_GRPC_CLIENT_CERT, ASR_GRPC_CLIENT_KEY, ASR_GRPC_SERVER_CA")
			}
		}
	}
	return nil
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

	// ASR (master switch defaults off so the wider repo runs without the C++
	// service deployed). When enabled, missing required vars are caught in
	// Config.Validate.
	viper.SetDefault("ASR_FEATURE_ENABLED", false)
	viper.SetDefault("ASR_TOKEN_DEFAULT_TTL", "3m")
	viper.SetDefault("ASR_TOKEN_MAX_TTL", "5m")
	viper.SetDefault("ASR_TOKEN_AUDIENCE", "rekall-asr")
	viper.SetDefault("ASR_TOKEN_ISSUER", "rekall-backend")
	viper.SetDefault("ASR_WS_URL_BASE", "ws://localhost:8081")
	viper.SetDefault("ASR_CIRCUIT_BREAKER_FAILURES", 3)
	viper.SetDefault("ASR_CIRCUIT_BREAKER_COOLDOWN", "30s")

	// Kat live-notes scheduler. Master switch is on by default so a deployment
	// that supplies a Foundry endpoint + key (or managed identity reachable)
	// gets Kat without further opt-in. When KAT_ENABLED=false (or no
	// endpoint/deployment is set), Kat reports `configured=false` via
	// /healthz/kat and the frontend renders the offline panel.
	viper.SetDefault("KAT_ENABLED", true)
	viper.SetDefault("KAT_PROVIDER", "foundry") // "foundry" | "openai"
	viper.SetDefault("KAT_FOUNDRY_API_VERSION", "2024-08-01-preview")
	viper.SetDefault("KAT_OPENAI_MODEL", "gpt-4o-mini")
	viper.SetDefault("KAT_FOUNDRY_REQUEST_TIMEOUT_MS", 15000)
	viper.SetDefault("KAT_WINDOW_SECONDS", 120)
	viper.SetDefault("KAT_STEP_SECONDS", 20)
	viper.SetDefault("KAT_MIN_NEW_SEGMENTS", 2)
	viper.SetDefault("KAT_MAX_CONCURRENT_RUNS", 4)
	viper.SetDefault("KAT_COOLDOWN_AFTER_ERROR_SECONDS", 60)
	viper.SetDefault("KAT_RING_BUFFER_CAPACITY", 20)
	viper.SetDefault("KAT_PROMPT_VERSION", "kat-v1")
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
