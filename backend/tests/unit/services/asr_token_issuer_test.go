package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/asr"
	apperr "github.com/rekall/backend/pkg/errors"
	"go.uber.org/zap"
)

const testSecret = "0123456789abcdef0123456789abcdef0123456789ab"

// ── Stubs ────────────────────────────────────────────────────────────────────

type stubASR struct {
	startResp *ports.StartSessionOutput
	startErr  error
	endResp   *ports.EndSessionOutput
	endErr    error
	startCalls int
	endCalls   int
}

func (s *stubASR) StartSession(_ context.Context, _ ports.StartSessionInput) (*ports.StartSessionOutput, error) {
	s.startCalls++
	return s.startResp, s.startErr
}
func (s *stubASR) EndSession(_ context.Context, _ uuid.UUID) (*ports.EndSessionOutput, error) {
	s.endCalls++
	return s.endResp, s.endErr
}
func (s *stubASR) Health(context.Context) (*ports.ASRHealth, error) { return nil, nil }
func (s *stubASR) Close() error                                     { return nil }

type stubCallRepo struct {
	call    *entities.Call
	getErr  error
}

func (s *stubCallRepo) GetByID(_ context.Context, _ uuid.UUID) (*entities.Call, error) {
	return s.call, s.getErr
}

// Unused interface methods — implement enough to satisfy ports.CallRepository.
func (s *stubCallRepo) Create(_ context.Context, _ *entities.Call) (*entities.Call, error) { return nil, nil }
func (s *stubCallRepo) Update(_ context.Context, _ *entities.Call) (*entities.Call, error) { return nil, nil }
func (s *stubCallRepo) SoftDelete(_ context.Context, _ uuid.UUID) error                    { return nil }
func (s *stubCallRepo) List(_ context.Context, _ ports.ListCallsFilter, _, _ int) ([]*entities.Call, int, error) {
	return nil, 0, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newIssuer(t *testing.T, repo ports.CallRepository, client ports.ASRClient) *services.ASRTokenIssuer {
	t.Helper()
	signer, err := asr.NewTokenSigner([]byte(testSecret), "rekall-backend", "rekall-asr")
	if err != nil {
		t.Fatalf("signer init: %v", err)
	}
	// meetingRepo + participantRepo are nil — these tests cover only the call
	// flow; the meeting flow is exercised by separate handler tests.
	return services.NewASRTokenIssuer(client, repo, nil, nil, signer,
		services.ASRTokenIssuerConfig{
			WSBaseURL:  "ws://test",
			DefaultTTL: 3 * time.Minute,
			MaxTTL:     5 * time.Minute,
		}, zap.NewNop())
}

func aCall(ownerID uuid.UUID, status string) *entities.Call {
	return &entities.Call{ID: uuid.New(), UserID: ownerID, Status: status}
}

func aSuccessOutput() *ports.StartSessionOutput {
	return &ports.StartSessionOutput{
		SessionID:   uuid.New(),
		ModelID:     "small.en",
		SampleRate:  16000,
		FrameFormat: "pcm_s16le_mono",
		ExpiresAt:   time.Now().Add(3 * time.Minute).UTC(),
	}
}

func mustStatus(t *testing.T, err error, want int) *apperr.AppError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with status %d, got nil", want)
	}
	app, ok := apperr.AsAppError(err)
	if !ok {
		t.Fatalf("expected *AppError, got %T (%v)", err, err)
	}
	if app.Status != want {
		t.Fatalf("status: want %d, got %d (%s)", want, app.Status, app.Code)
	}
	return app
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestASRTokenIssuer_RequestSuccess(t *testing.T) {
	owner := uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{startResp: aSuccessOutput()}
	issuer := newIssuer(t, repo, client)

	out, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: owner, CallID: repo.call.ID,
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if out.SessionToken == "" {
		t.Fatalf("token must not be empty")
	}
	if out.WsURL == "" || !contains(out.WsURL, out.SessionToken) {
		t.Fatalf("ws_url should embed token: %q", out.WsURL)
	}
	if client.startCalls != 1 {
		t.Fatalf("expected exactly one StartSession call, got %d", client.startCalls)
	}
}

func TestASRTokenIssuer_RejectsNonOwner(t *testing.T) {
	owner, caller := uuid.New(), uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: caller, CallID: repo.call.ID,
	})
	app := mustStatus(t, err, http.StatusForbidden)
	if app.Code != "ASR_ACCESS_DENIED" {
		t.Fatalf("code: want ASR_ACCESS_DENIED, got %s", app.Code)
	}
	if client.startCalls != 0 {
		t.Fatalf("StartSession must not be called when caller is forbidden")
	}
}

func TestASRTokenIssuer_RejectsFinalisedCall(t *testing.T) {
	owner := uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "done")}
	client := &stubASR{}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: owner, CallID: repo.call.ID,
	})
	app := mustStatus(t, err, http.StatusConflict)
	if app.Code != "CALL_ALREADY_FINALISED" {
		t.Fatalf("code: want CALL_ALREADY_FINALISED, got %s", app.Code)
	}
}

func TestASRTokenIssuer_AtCapacity(t *testing.T) {
	owner := uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{startErr: ports.ErrASRAtCapacity}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: owner, CallID: repo.call.ID,
	})
	app := mustStatus(t, err, http.StatusServiceUnavailable)
	if app.Code != "ASR_AT_CAPACITY" {
		t.Fatalf("code: want ASR_AT_CAPACITY, got %s", app.Code)
	}
	if app.RetryAfterSeconds != 5 {
		t.Fatalf("Retry-After: want 5, got %d", app.RetryAfterSeconds)
	}
}

func TestASRTokenIssuer_Unavailable(t *testing.T) {
	owner := uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{startErr: ports.ErrASRUnavailable}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: owner, CallID: repo.call.ID,
	})
	app := mustStatus(t, err, http.StatusServiceUnavailable)
	if app.Code != "ASR_UNAVAILABLE" {
		t.Fatalf("code: want ASR_UNAVAILABLE, got %s", app.Code)
	}
}

func TestASRTokenIssuer_TTLOverMaxRejected(t *testing.T) {
	owner := uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{startResp: aSuccessOutput()}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: owner, CallID: repo.call.ID, RequestedTTL: 10 * time.Minute,
	})
	mustStatus(t, err, http.StatusBadRequest)
	if client.startCalls != 0 {
		t.Fatalf("validation should run before StartSession is called")
	}
}

func TestASRTokenIssuer_End(t *testing.T) {
	owner, sid := uuid.New(), uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{endResp: &ports.EndSessionOutput{
		FinalTranscript: "the quick brown fox",
		FinalCount:      3,
	}}
	issuer := newIssuer(t, repo, client)

	out, err := issuer.End(context.Background(), owner, repo.call.ID, sid)
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if out.FinalCount != 3 {
		t.Fatalf("final_count: want 3, got %d", out.FinalCount)
	}
	if client.endCalls != 1 {
		t.Fatalf("EndSession not called")
	}
}

func TestASRTokenIssuer_End_ForwardsUnavailable(t *testing.T) {
	owner, sid := uuid.New(), uuid.New()
	repo := &stubCallRepo{call: aCall(owner, "pending")}
	client := &stubASR{endErr: ports.ErrASRUnavailable}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.End(context.Background(), owner, repo.call.ID, sid)
	app := mustStatus(t, err, http.StatusServiceUnavailable)
	if app.Code != "ASR_UNAVAILABLE" {
		t.Fatalf("code: want ASR_UNAVAILABLE, got %s", app.Code)
	}
}

func TestASRTokenIssuer_Repo404(t *testing.T) {
	repo := &stubCallRepo{getErr: apperr.NotFound("Call", uuid.New().String())}
	client := &stubASR{}
	issuer := newIssuer(t, repo, client)

	_, err := issuer.Request(context.Background(), services.RequestInput{
		CallerID: uuid.New(), CallID: uuid.New(),
	})
	mustStatus(t, err, http.StatusNotFound)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// silence unused-import: errors is referenced through apperr internals only.
var _ = errors.Is
