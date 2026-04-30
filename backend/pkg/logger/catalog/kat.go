package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Kat live-notes events ────────────────────────────────────────────────────
//
// Emitted by the Foundry adapter (auth + per-call lifecycle), the
// KatNotesService scheduler (tick / generation / cohort lifecycle), and the
// WS broadcaster adapter. Kat does not persist anything — these structured
// log entries are the only forensic surface for failed runs.

var (
	// KatFoundryInitialized is logged once at boot when the Foundry client is
	// constructed successfully. The auth_mode field carries either "api_key"
	// or "managed_identity"; the API key value MUST NEVER appear here.
	KatFoundryInitialized = logger.LogEvent{
		Code:    "KAT_FOUNDRY_INITIALIZED",
		Message: "kat foundry client initialised",
	}

	// KatFoundryUnconfigured is logged at warn when the boot-time Foundry
	// construction failed (missing endpoint / deployment / credential chain).
	// Boot still succeeds; Kat reports IsConfigured() == false.
	KatFoundryUnconfigured = logger.LogEvent{
		Code:    "KAT_FOUNDRY_UNCONFIGURED",
		Message: "kat foundry client not configured",
	}

	// KatFoundryTimeout is logged when the per-call deadline elapses.
	KatFoundryTimeout = logger.LogEvent{
		Code:    "KAT_FOUNDRY_TIMEOUT",
		Message: "kat foundry call timed out",
	}

	// KatFoundryRateLimited is logged after the second 429 response.
	KatFoundryRateLimited = logger.LogEvent{
		Code:    "KAT_FOUNDRY_RATE_LIMITED",
		Message: "kat foundry rate-limited",
	}

	// KatFoundryUnavailable is logged after the second 5xx response.
	KatFoundryUnavailable = logger.LogEvent{
		Code:    "KAT_FOUNDRY_UNAVAILABLE",
		Message: "kat foundry unavailable",
	}

	// KatFoundryParseFailed is logged after the second invalid-JSON response.
	KatFoundryParseFailed = logger.LogEvent{
		Code:    "KAT_FOUNDRY_PARSE_FAILED",
		Message: "kat foundry response parse failed",
	}

	// KatRunStarted is a debug event marking the start of a Generation_Run.
	KatRunStarted = logger.LogEvent{
		Code:    "KAT_RUN_STARTED",
		Message: "kat generation run started",
	}

	// KatRunOk is logged on a successful run — one row pushed to the in-memory
	// ring buffer + one WS broadcast fanned out.
	KatRunOk = logger.LogEvent{
		Code:    "KAT_RUN_OK",
		Message: "kat generation run completed",
	}

	// KatRunFailed is the only forensic surface for a failed run; carries
	// meeting_id / call_id / error_code / latency_ms.
	KatRunFailed = logger.LogEvent{
		Code:    "KAT_RUN_FAILED",
		Message: "kat generation run failed",
	}

	// KatTickNoop is a debug event for "tick fired but not enough new segments".
	KatTickNoop = logger.LogEvent{
		Code:    "KAT_TICK_NOOP",
		Message: "kat tick produced no run",
	}

	// KatBroadcastFailed is logged when the WS broadcaster reports a failure;
	// the in-memory ring buffer is unaffected so a late-join replay still works.
	KatBroadcastFailed = logger.LogEvent{
		Code:    "KAT_BROADCAST_FAILED",
		Message: "kat ws broadcast failed",
	}

	// KatConfigInvalid is logged at warn when a bad env-var value triggers a
	// fallback to the spec default (e.g. window <= step).
	KatConfigInvalid = logger.LogEvent{
		Code:    "KAT_CONFIG_INVALID",
		Message: "kat configuration invalid; substituting defaults",
	}

	// KatConcurrencyDeferred is logged when the global semaphore is full and
	// the tick is skipped rather than queued.
	KatConcurrencyDeferred = logger.LogEvent{
		Code:    "KAT_CONCURRENCY_DEFERRED",
		Message: "kat concurrency cap reached; tick deferred",
	}

	// KatLoadSegmentsFailed is logged when the transcript repo returns an
	// error during a tick. The tick is skipped; no cooldown is set (this is
	// not a Foundry problem).
	KatLoadSegmentsFailed = logger.LogEvent{
		Code:    "KAT_LOAD_SEGMENTS_FAILED",
		Message: "kat tick failed to load transcript segments",
	}
)
