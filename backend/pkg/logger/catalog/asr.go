package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── ASR (Go-side) events ─────────────────────────────────────────────────────
//
// These are emitted by the ASRTokenIssuer service and the gRPC client when the
// Go backend interacts with the standalone C++ ASR microservice. C++-side
// events use the same `event_code` convention so a single Loki/Datadog query
// joins both surfaces.

var (
	// AsrTokenIssued is logged at Info level when a Session_Token is signed
	// and returned to the frontend. The full token NEVER appears in logs;
	// only the 8-char prefix.
	AsrTokenIssued = logger.LogEvent{
		Code:    "ASR_TOKEN_ISSUED",
		Message: "asr session token issued",
	}

	// AsrTokenIssueFailed covers any failure between caller-validation and
	// JWT signing — gRPC errors, signing errors, repository errors.
	AsrTokenIssueFailed = logger.LogEvent{
		Code:    "ASR_TOKEN_ISSUE_FAILED",
		Message: "asr session token issuance failed",
	}

	// AsrAuthForbidden is logged at Warn when a caller attempts to issue a
	// token for a call they do not own.
	AsrAuthForbidden = logger.LogEvent{
		Code:    "ASR_AUTH_FORBIDDEN",
		Message: "caller does not own the call",
	}

	// AsrSessionEndOk is logged at Info on successful EndSession passthrough.
	AsrSessionEndOk = logger.LogEvent{
		Code:    "ASR_SESSION_END_OK",
		Message: "asr session ended via gRPC",
	}

	// AsrSessionEndFailed is logged when EndSession returned an error.
	AsrSessionEndFailed = logger.LogEvent{
		Code:    "ASR_SESSION_END_FAILED",
		Message: "asr session end failed",
	}

	// AsrCircuitOpen is logged at Warn when the breaker trips.
	AsrCircuitOpen = logger.LogEvent{
		Code:    "ASR_CIRCUIT_OPEN",
		Message: "asr circuit breaker tripped open",
	}

	// AsrCircuitHalfOpen is logged when the breaker admits a half-open probe.
	AsrCircuitHalfOpen = logger.LogEvent{
		Code:    "ASR_CIRCUIT_HALF_OPEN",
		Message: "asr circuit breaker half-open probe admitted",
	}

	// AsrCircuitClosed is logged when a successful call closes the breaker.
	AsrCircuitClosed = logger.LogEvent{
		Code:    "ASR_CIRCUIT_CLOSED",
		Message: "asr circuit breaker closed",
	}

	// AsrGrpcDialFailed is logged at Error level on a non-recoverable dial
	// failure during boot.
	AsrGrpcDialFailed = logger.LogEvent{
		Code:    "ASR_GRPC_DIAL_FAILED",
		Message: "asr grpc dial failed",
	}

	// AsrGrpcResponse is a low-level Debug event capturing per-RPC outcomes.
	AsrGrpcResponse = logger.LogEvent{
		Code:    "ASR_GRPC_RESPONSE",
		Message: "asr grpc rpc response",
	}
)
