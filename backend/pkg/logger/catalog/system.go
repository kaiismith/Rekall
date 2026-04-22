// Package catalog is the single source of truth for every log event in the
// Rekall backend. Each exported variable is a logger.LogEvent — a (code, message)
// pair that is emitted with a stable, machine-readable event_code field so that
// log aggregators (Datadog, Loki, CloudWatch Insights …) can filter and alert on
// exact events without parsing free-form strings.
//
// Naming convention: <DOMAIN>_<NOUN>_<OUTCOME>
//
//	DOMAIN  — SYS (process lifecycle), DB (database), HTTP (http layer)
//	NOUN    — what was acted on (CONFIG, SERVER, POOL, REQUEST …)
//	OUTCOME — OK / FAILED / SIGNAL / STARTING / READY / STOPPED …
//
// Severity guidance
//
//	Fatal  — unrecoverable; process will exit immediately after this log
//	Error  — operation failed; requires operator attention
//	Warn   — unexpected but recoverable condition; worth monitoring
//	Info   — normal operational milestone (startup, ready, shutdown)
//	Debug  — high-frequency detail useful only during development
package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Process lifecycle ────────────────────────────────────────────────────────

var (
	// SysInitialising is logged as the very first line before any subsystem starts.
	SysInitialising = logger.LogEvent{
		Code:    "SYS_INITIALISING",
		Message: "rekall backend initialising — loading configuration and subsystems",
	}

	// SysConfigLoaded is logged once all environment variables have been validated.
	SysConfigLoaded = logger.LogEvent{
		Code:    "SYS_CONFIG_LOADED",
		Message: "configuration loaded and validated successfully",
	}

	// SysLoggerReady is logged immediately after the production logger is built.
	SysLoggerReady = logger.LogEvent{
		Code:    "SYS_LOGGER_READY",
		Message: "structured logger initialised",
	}

	// SysRouterConfigured is logged after all routes and middleware are registered.
	SysRouterConfigured = logger.LogEvent{
		Code:    "SYS_ROUTER_CONFIGURED",
		Message: "HTTP router configured with routes and middleware",
	}

	// SysReady is logged when the server is bound and accepting inbound traffic.
	SysReady = logger.LogEvent{
		Code:    "SYS_READY",
		Message: "server is ready and accepting connections",
	}

	// SysShutdownSignal is logged when an OS signal triggers graceful shutdown.
	SysShutdownSignal = logger.LogEvent{
		Code:    "SYS_SHUTDOWN_SIGNAL",
		Message: "OS signal received — initiating graceful shutdown",
	}

	// SysShutdownDraining is logged while the server waits for in-flight requests.
	SysShutdownDraining = logger.LogEvent{
		Code:    "SYS_SHUTDOWN_DRAINING",
		Message: "draining in-flight requests before shutdown",
	}

	// SysShutdownOK is logged once all connections have drained and the process exits cleanly.
	SysShutdownOK = logger.LogEvent{
		Code:    "SYS_SHUTDOWN_OK",
		Message: "server shut down cleanly — all connections drained",
	}

	// SysShutdownTimeout is logged when graceful shutdown exceeds the configured drain timeout.
	SysShutdownTimeout = logger.LogEvent{
		Code:    "SYS_SHUTDOWN_TIMEOUT",
		Message: "graceful shutdown timed out — forcing process exit",
	}

	// SysConfigInvalid is logged when required configuration is missing or malformed.
	SysConfigInvalid = logger.LogEvent{
		Code:    "SYS_CONFIG_INVALID",
		Message: "configuration is invalid or incomplete — required variable is missing",
	}

	// SysLoggerFailed is logged (to stderr) when the logger itself cannot be built.
	SysLoggerFailed = logger.LogEvent{
		Code:    "SYS_LOGGER_FAILED",
		Message: "failed to initialise structured logger",
	}
)

// ─── Database ─────────────────────────────────────────────────────────────────

var (
	// DBPoolOpening is logged just before the connection pool is opened.
	DBPoolOpening = logger.LogEvent{
		Code:    "DB_POOL_OPENING",
		Message: "opening database connection pool",
	}

	// DBConnected is logged after the pool is open and the first ping succeeds.
	DBConnected = logger.LogEvent{
		Code:    "DB_CONNECTED",
		Message: "database connection pool established and verified",
	}

	// DBConnFailed is logged (Fatal) when the initial connection cannot be established.
	DBConnFailed = logger.LogEvent{
		Code:    "DB_CONN_FAILED",
		Message: "failed to establish database connection — verify DSN, credentials, and network access",
	}

	// DBPingFailed is logged when a readiness ping fails (used by the /ready probe).
	DBPingFailed = logger.LogEvent{
		Code:    "DB_PING_FAILED",
		Message: "database liveness ping failed — connection may be lost",
	}

	// DBPoolClosed is logged after the connection pool is closed during shutdown.
	DBPoolClosed = logger.LogEvent{
		Code:    "DB_POOL_CLOSED",
		Message: "database connection pool closed",
	}
)

// ─── HTTP layer ───────────────────────────────────────────────────────────────

var (
	// HTTPRequest is logged by the access-log middleware for every successful request (2xx/3xx).
	HTTPRequest = logger.LogEvent{
		Code:    "HTTP_REQUEST",
		Message: "HTTP request completed successfully",
	}

	// HTTPClientError is logged for 4xx responses — the caller sent a bad request.
	HTTPClientError = logger.LogEvent{
		Code:    "HTTP_CLIENT_ERROR",
		Message: "HTTP request rejected with client error",
	}

	// HTTPServerError is logged for 5xx responses — the server failed to handle the request.
	HTTPServerError = logger.LogEvent{
		Code:    "HTTP_SERVER_ERROR",
		Message: "HTTP request failed with internal server error",
	}

	// HTTPPanic is logged by the recovery middleware when a handler panics.
	// The panic value and full stack trace are included as structured fields.
	HTTPPanic = logger.LogEvent{
		Code:    "HTTP_PANIC",
		Message: "unhandled panic recovered in HTTP handler — request aborted with 500",
	}

	// HTTPServerStarting is logged just before ListenAndServe is called.
	HTTPServerStarting = logger.LogEvent{
		Code:    "HTTP_SERVER_STARTING",
		Message: "HTTP server starting — binding to address",
	}

	// HTTPServerStopped is logged after ListenAndServe returns (not an error path).
	HTTPServerStopped = logger.LogEvent{
		Code:    "HTTP_SERVER_STOPPED",
		Message: "HTTP server stopped listening",
	}
)
