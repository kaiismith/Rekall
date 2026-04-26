package asr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rekall/backend/internal/infrastructure/asr/pb"
)

// MTLSConfig carries paths to PEM files used to mutually authenticate the
// gRPC channel. Empty fields disable mTLS (insecure channel acceptable in
// dev only — production should always set all four).
type MTLSConfig struct {
	ClientCert string
	ClientKey  string
	ServerCA   string
	ServerName string
}

// ClientConfig is the wiring DTO for NewGRPCClient.
type ClientConfig struct {
	Addr                   string
	MTLS                   MTLSConfig
	StartTimeout           time.Duration
	EndTimeout             time.Duration
	HealthTimeout          time.Duration
	CircuitBreakerFailures int
	CircuitBreakerCooldown time.Duration
}

// GRPCClient is the production implementation of ports.ASRClient.
type GRPCClient struct {
	conn    *grpc.ClientConn
	stub    pb.ASRClient
	breaker *CircuitBreaker
	cfg     ClientConfig
}

// NewGRPCClient dials the ASR control plane. Returns an error if the channel
// cannot be established (does not retry — main.go decides how to react).
func NewGRPCClient(cfg ClientConfig) (*GRPCClient, error) {
	if cfg.Addr == "" {
		return nil, errors.New("ASR gRPC addr is required")
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 2 * time.Second
	}
	if cfg.EndTimeout == 0 {
		cfg.EndTimeout = 2 * time.Second
	}
	if cfg.HealthTimeout == 0 {
		cfg.HealthTimeout = 1 * time.Second
	}

	creds, err := dialCreds(cfg.MTLS)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(
		cfg.Addr,
		grpc.WithTransportCredentials(creds),
		// Force the hand-rolled codec — the default proto codec walks
		// MessageInfo descriptors that our hand-maintained pb stubs don't
		// populate, and panics inside protoimpl reflection.
		grpc.WithDefaultCallOptions(grpc.ForceCodec(pb.Codec())),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.WithDefaultServiceConfig(`{
		    "methodConfig": [{
		        "name": [{"service": "rekall.asr.v1.ASR"}],
		        "retryPolicy": {
		            "maxAttempts": 3,
		            "initialBackoff": "0.25s",
		            "maxBackoff": "1s",
		            "backoffMultiplier": 2.0,
		            "retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
		        }
		    }]
		}`),
	)
	if err != nil {
		return nil, fmt.Errorf("asr grpc dial %s: %w", cfg.Addr, err)
	}

	return &GRPCClient{
		conn:    conn,
		stub:    pb.NewASRClient(conn),
		breaker: NewCircuitBreaker(cfg.CircuitBreakerFailures, cfg.CircuitBreakerCooldown),
		cfg:     cfg,
	}, nil
}

// Close shuts down the gRPC channel. Idempotent.
func (c *GRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// Breaker exposes the breaker for observability tests.
func (c *GRPCClient) Breaker() *CircuitBreaker { return c.breaker }

// StartSession creates a server-side session and returns its canonical metadata.
func (c *GRPCClient) StartSession(ctx context.Context, in ports.StartSessionInput) (*ports.StartSessionOutput, error) {
	if !c.breaker.Allow() {
		return nil, ports.ErrASRUnavailable
	}
	ctx, cancel := withCorrelation(ctx, c.cfg.StartTimeout)
	defer cancel()

	req := &pb.StartSessionRequest{
		UserId:                   in.UserID.String(),
		CallId:                   in.CallID.String(),
		ModelId:                  in.ModelID,
		Language:                 in.Language,
		RequestedTokenTtlSeconds: uint32(in.RequestedTTL / time.Second),
	}
	resp, err := c.stub.StartSession(ctx, req)
	if err != nil {
		return nil, c.handleErr(err)
	}
	c.breaker.OnSuccess()

	sid, perr := uuid.Parse(resp.GetSessionId())
	if perr != nil {
		return nil, fmt.Errorf("asr returned non-uuid session_id: %w", perr)
	}
	return &ports.StartSessionOutput{
		SessionID:   sid,
		ModelID:     resp.GetModelId(),
		SampleRate:  resp.GetSampleRate(),
		FrameFormat: resp.GetFrameFormat(),
		ExpiresAt:   resp.GetExpiresAt().AsTime(),
	}, nil
}

// EndSession terminates a session; idempotent.
func (c *GRPCClient) EndSession(ctx context.Context, sessionID uuid.UUID) (*ports.EndSessionOutput, error) {
	if !c.breaker.Allow() {
		return nil, ports.ErrASRUnavailable
	}
	ctx, cancel := withCorrelation(ctx, c.cfg.EndTimeout)
	defer cancel()
	resp, err := c.stub.EndSession(ctx, &pb.EndSessionRequest{SessionId: sessionID.String()})
	if err != nil {
		return nil, c.handleErr(err)
	}
	c.breaker.OnSuccess()
	return &ports.EndSessionOutput{
		FinalTranscript: resp.GetFinalTranscript(),
		FinalCount:      resp.GetFinalCount(),
	}, nil
}

// Health returns the live snapshot of the upstream service.
func (c *GRPCClient) Health(ctx context.Context) (*ports.ASRHealth, error) {
	if !c.breaker.Allow() {
		return nil, ports.ErrASRUnavailable
	}
	ctx, cancel := withCorrelation(ctx, c.cfg.HealthTimeout)
	defer cancel()
	resp, err := c.stub.Health(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, c.handleErr(err)
	}
	c.breaker.OnSuccess()
	return &ports.ASRHealth{
		Status:          resp.GetStatus(),
		Version:         resp.GetVersion(),
		UptimeSeconds:   resp.GetUptimeSeconds(),
		LoadedModels:    resp.GetLoadedModels(),
		ActiveSessions:  resp.GetActiveSessions(),
		WorkerPoolSize:  resp.GetWorkerPoolSize(),
		WorkerPoolInUse: resp.GetWorkerPoolInUse(),
	}, nil
}

func (c *GRPCClient) handleErr(err error) error {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.OK:
		return nil
	case codes.ResourceExhausted:
		c.breaker.OnSuccess() // not a transport failure — server told us "no" cleanly
		return ports.ErrASRAtCapacity
	case codes.Unavailable, codes.DeadlineExceeded:
		c.breaker.OnFailure()
		return ports.ErrASRUnavailable
	default:
		c.breaker.OnSuccess() // upstream responded; treat as logical failure
		return err
	}
}

func withCorrelation(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	if cid, ok := parent.Value(correlationKey{}).(string); ok && cid != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-correlation-id", cid)
	}
	return ctx, cancel
}

type correlationKey struct{}

// WithCorrelationID returns a derived context that propagates the correlation
// id as gRPC metadata on every outgoing call. Handlers should pull the id
// from their existing request_id middleware and call this once per request.
func WithCorrelationID(parent context.Context, id string) context.Context {
	return context.WithValue(parent, correlationKey{}, id)
}

func dialCreds(m MTLSConfig) (credentials.TransportCredentials, error) {
	if m.ClientCert == "" && m.ClientKey == "" && m.ServerCA == "" {
		return insecure.NewCredentials(), nil
	}
	if m.ClientCert == "" || m.ClientKey == "" || m.ServerCA == "" {
		return nil, errors.New("ASR mTLS requires all of ClientCert, ClientKey, ServerCA")
	}
	cert, err := loadKeyPair(m.ClientCert, m.ClientKey)
	if err != nil {
		return nil, err
	}
	pool, err := loadCertPool(m.ServerCA)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(buildTLSConfig(cert, pool, m.ServerName)), nil
}

// — keep TLS helpers in their own file so the test build does not need crypto/x509
//   indirection. See tls.go.
//
// The os.ReadFile-only helpers below are exposed for tests to substitute.
var _ = os.ReadFile
