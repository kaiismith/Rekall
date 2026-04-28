package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrASRUnavailable is returned by ASRClient implementations when the upstream
// service is unreachable, the circuit breaker is open, or the gRPC channel
// times out before a response. Callers should map this to HTTP 503.
var ErrASRUnavailable = errors.New("asr service is unavailable")

// ErrASRAtCapacity is returned when the upstream gRPC call returned
// RESOURCE_EXHAUSTED. Callers should map this to HTTP 503 + Retry-After.
var ErrASRAtCapacity = errors.New("asr service at capacity")

// StartSessionInput is the application-layer input for ASRClient.StartSession.
// The Go side does not pass arbitrary metadata — keep the surface minimal.
type StartSessionInput struct {
	UserID       uuid.UUID
	CallID       uuid.UUID
	ModelID      string        // optional; "" → server default
	Language     string        // optional BCP-47
	RequestedTTL time.Duration // 0 → server default; clamped to [60s, 300s]
}

// StartSessionOutput mirrors the canonical response after the ASR side has
// applied any model fallback and TTL clamping.
type StartSessionOutput struct {
	SessionID   uuid.UUID
	ModelID     string // canonical id actually selected
	SampleRate  int32  // always 16000 for whisper
	FrameFormat string // "pcm_s16le_mono"
	ExpiresAt   time.Time
}

// EndSessionOutput carries the stitched final transcript so the backend can
// persist it once the WebSocket has closed.
type EndSessionOutput struct {
	FinalTranscript string
	FinalCount      uint32
}

// ASRHealth is the live snapshot returned by ASRClient.Health, used by the
// circuit breaker probe and a future /health endpoint.
type ASRHealth struct {
	Status          string // SERVING | NOT_SERVING
	Version         string
	UptimeSeconds   uint64
	LoadedModels    []string
	ActiveSessions  uint32
	WorkerPoolSize  uint32
	WorkerPoolInUse uint32
	// Engine selection surfaced by the upstream service. "local" | "openai".
	EngineMode string
	// Local engine: default model id. OpenAI engine: base url. Empty otherwise.
	EngineTarget string
}

// ASRClient is the application-facing port to the standalone C++ ASR service.
// Implementations live under internal/infrastructure/asr (gRPC).
type ASRClient interface {
	StartSession(ctx context.Context, in StartSessionInput) (*StartSessionOutput, error)
	EndSession(ctx context.Context, sessionID uuid.UUID) (*EndSessionOutput, error)
	Health(ctx context.Context) (*ASRHealth, error)
	// Close releases the underlying gRPC channel. Idempotent.
	Close() error
}
