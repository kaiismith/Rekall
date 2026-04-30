package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/asr"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// asrHealthTTL is how long a Health() snapshot is reused before re-polling.
// 60 s strikes a balance: short enough to catch a model rollover quickly, long
// enough that bursty session-open traffic doesn't hammer the gRPC server.
const asrHealthTTL = 60 * time.Second

// healthCache memoises ASRClient.Health() for asrHealthTTL so the issuer
// doesn't poll the ASR service on every session-open. A stale snapshot is
// preferred to a hard failure: if the refresh errors we keep using whatever
// we have so the issuer can still finish; the engine snapshot is just
// best-effort metadata, not an auth boundary.
type healthCache struct {
	mu         sync.Mutex
	snapshot   *ports.ASRHealth
	fetchedAt  time.Time
	asrClient  ports.ASRClient
	defaultTTL time.Duration
}

func newHealthCache(asrClient ports.ASRClient) *healthCache {
	return &healthCache{asrClient: asrClient, defaultTTL: asrHealthTTL}
}

// get returns a cached snapshot, refreshing it if older than defaultTTL.
// On refresh failure the previous snapshot (possibly nil) is returned with
// no error — callers must tolerate a nil result.
func (c *healthCache) get(ctx context.Context) *ports.ASRHealth {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.snapshot != nil && time.Since(c.fetchedAt) < c.defaultTTL {
		return c.snapshot
	}
	h, err := c.asrClient.Health(ctx)
	if err != nil || h == nil {
		return c.snapshot // stale-but-non-nil if we have one, else nil
	}
	c.snapshot = h
	c.fetchedAt = time.Now()
	return c.snapshot
}

// ASRTokenIssuerConfig captures the runtime knobs the issuer needs.
type ASRTokenIssuerConfig struct {
	WSBaseURL  string        // e.g. wss://asr.rekall.example
	DefaultTTL time.Duration // server default (clamped server-side too)
	MaxTTL     time.Duration // hard cap; requests above this return 400
}

// ASRSessionPayload is the application-layer DTO returned to the HTTP handler.
type ASRSessionPayload struct {
	SessionID    string
	SessionToken string
	WsURL        string
	ExpiresAt    time.Time
	ModelID      string
	SampleRate   int32
	FrameFormat  string
}

// ASRKatHooks is the slice of KatNotesService that the ASR token issuer
// depends on. Declared as a small interface so tests don't need a real
// scheduler and so a Kat-disabled deployment can pass nil. Both methods are
// best-effort: a panic or error inside them must not affect token issuance.
type ASRKatHooks interface {
	OnCallSessionOpened(callID uuid.UUID)
	OnCallSessionEnded(callID uuid.UUID)
}

// ASRTokenIssuer mints short-lived JWTs binding a user to an asr session.
// It is the Go-side complement of the C++ JWTValidator.
type ASRTokenIssuer struct {
	asrClient       ports.ASRClient
	callRepo        ports.CallRepository
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	persister       *TranscriptPersister // optional: when nil, sessions are not persisted
	katHooks        ASRKatHooks          // optional Kat lifecycle callbacks
	health          *healthCache
	signer          *asr.TokenSigner
	cfg             ASRTokenIssuerConfig
	logger          *zap.Logger
}

// NewASRTokenIssuer wires the issuer. callRepo is required for the call flow;
// meetingRepo/participantRepo may be nil when only the call flow is in use
// (the meeting endpoint then returns 503 ASR_NOT_CONFIGURED).
//
// persister may be nil; when set, every successful session open is recorded
// in transcript_sessions and every End rebuilds calls.transcript from the
// persisted segments. A persister failure does NOT fail the token issuance —
// captions UX must continue working even when persistence is degraded.
func NewASRTokenIssuer(
	asrClient ports.ASRClient,
	callRepo ports.CallRepository,
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	persister *TranscriptPersister,
	signer *asr.TokenSigner,
	cfg ASRTokenIssuerConfig,
	logger *zap.Logger,
) *ASRTokenIssuer {
	return &ASRTokenIssuer{
		asrClient:       asrClient,
		callRepo:        callRepo,
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		persister:       persister,
		health:          newHealthCache(asrClient),
		signer:          signer,
		cfg:             cfg,
		logger:          applogger.WithComponent(logger, "asr_token_issuer"),
	}
}

// SetKatHooks installs optional Kat lifecycle callbacks. Safe to call after
// construction; nil disables the hooks.
func (s *ASRTokenIssuer) SetKatHooks(h ASRKatHooks) { s.katHooks = h }

// engineSnapshotFor returns the (mode, target, model) triple to record on a
// new transcript_sessions row. If Health() is unavailable, fall back to the
// model_id we already know plus "unknown" engine markers — the row is still
// useful, just less precise.
func (s *ASRTokenIssuer) engineSnapshotFor(ctx context.Context, modelID string) EngineSnapshot {
	h := s.health.get(ctx)
	mode := entities.TranscriptEngineModeLocal
	target := ""
	if h != nil {
		if h.EngineMode != "" {
			mode = h.EngineMode
		}
		target = h.EngineTarget
	}
	// Diagnostic: shows the raw Health() result so an `engine_mode='local'`
	// row in the DB despite an OpenAI-mode ASR can be traced to either a
	// nil Health (gRPC failure) or empty EngineMode (C++ field not populated).
	if h == nil {
		s.logger.Info("DIAG engineSnapshotFor: Health returned nil",
			zap.String("fallback_mode", mode))
	} else {
		s.logger.Info("DIAG engineSnapshotFor: Health snapshot",
			zap.String("h.engine_mode", h.EngineMode),
			zap.String("h.engine_target", h.EngineTarget),
			zap.String("h.status", h.Status),
			zap.String("h.version", h.Version),
			zap.Uint32("h.workers", h.WorkerPoolSize))
	}
	return EngineSnapshot{Mode: mode, Target: target, ModelID: modelID}
}

// RequestInput is the application-layer input for Request().
type RequestInput struct {
	CallerID     uuid.UUID
	CallID       uuid.UUID
	ModelID      string        // optional
	Language     string        // optional BCP-47
	RequestedTTL time.Duration // 0 → server default
}

// Request validates the caller, registers a session over gRPC, and signs a
// JWT bound to the resulting session_id.
func (s *ASRTokenIssuer) Request(ctx context.Context, in RequestInput) (*ASRSessionPayload, error) {
	if in.RequestedTTL > 0 && in.RequestedTTL > s.cfg.MaxTTL {
		return nil, apperr.BadRequest(fmt.Sprintf(
			"requested_token_ttl_seconds exceeds max (%s)", s.cfg.MaxTTL))
	}

	call, err := s.callRepo.GetByID(ctx, in.CallID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("Call", in.CallID.String())
		}
		return nil, apperr.Internal("failed to look up call")
	}
	if call.UserID != in.CallerID {
		catalog.AsrAuthForbidden.Warn(s.logger,
			zap.String("caller_id", in.CallerID.String()),
			zap.String("call_id", in.CallID.String()),
		)
		return nil, apperr.ForbiddenCode("ASR_ACCESS_DENIED",
			"caller does not own the call")
	}
	if call.Status == "done" {
		return nil, apperr.ConflictCode("CALL_ALREADY_FINALISED",
			"call has been finalised")
	}

	out, err := s.asrClient.StartSession(ctx, ports.StartSessionInput{
		UserID:       in.CallerID,
		CallID:       in.CallID,
		ModelID:      in.ModelID,
		Language:     in.Language,
		RequestedTTL: in.RequestedTTL,
	})
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrASRAtCapacity):
			return nil, apperr.ServiceUnavailable("ASR_AT_CAPACITY",
				"asr service is at capacity", 5)
		case errors.Is(err, ports.ErrASRUnavailable):
			catalog.AsrCircuitOpen.Warn(s.logger,
				zap.String("call_id", in.CallID.String()),
			)
			return nil, apperr.ServiceUnavailable("ASR_UNAVAILABLE",
				"asr service is unavailable", 0)
		default:
			catalog.AsrTokenIssueFailed.Error(s.logger,
				zap.Error(err),
				zap.String("call_id", in.CallID.String()),
			)
			return nil, apperr.Internal("asr StartSession failed")
		}
	}

	token, err := s.signer.Sign(in.CallerID, in.CallID, out.SessionID,
		out.ModelID, out.ExpiresAt)
	if err != nil {
		catalog.AsrTokenIssueFailed.Error(s.logger,
			zap.Error(err),
			zap.String("session_id", out.SessionID.String()),
		)
		return nil, apperr.Internal("asr token sign failed")
	}

	prefix := token
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	catalog.AsrTokenIssued.Info(s.logger,
		zap.String("user_id", in.CallerID.String()),
		zap.String("call_id", in.CallID.String()),
		zap.String("session_id", out.SessionID.String()),
		zap.String("model_id", out.ModelID),
		zap.String("token_prefix", prefix),
		zap.Time("expires_at", out.ExpiresAt),
	)

	// Best-effort persistence open: a failure does NOT fail token issuance.
	if s.persister != nil {
		callID := in.CallID
		var langPtr *string
		if in.Language != "" {
			lang := in.Language
			langPtr = &lang
		}
		if err := s.persister.OpenSession(ctx, OpenSessionInput{
			SessionID:         out.SessionID,
			SpeakerUserID:     in.CallerID,
			CallID:            &callID,
			Engine:            s.engineSnapshotFor(ctx, out.ModelID),
			LanguageRequested: langPtr,
			SampleRate:        out.SampleRate,
			FrameFormat:       out.FrameFormat,
			ExpiresAt:         out.ExpiresAt,
		}); err != nil {
			s.logger.Warn("transcript session open failed; persistence degraded for this call",
				zap.Error(err),
				zap.String("session_id", out.SessionID.String()),
			)
		}
	}

	// Kat hook: register the call cohort so the live-notes scheduler starts
	// ticking against this call's transcript. Off the request path; failure
	// is opaque to the caller.
	if s.katHooks != nil {
		callID := in.CallID
		go func() {
			defer func() { _ = recover() }()
			s.katHooks.OnCallSessionOpened(callID)
		}()
	}

	return &ASRSessionPayload{
		SessionID:    out.SessionID.String(),
		SessionToken: token,
		WsURL: fmt.Sprintf("%s/v1/asr/stream?token=%s",
			s.cfg.WSBaseURL, token),
		ExpiresAt:   out.ExpiresAt,
		ModelID:     out.ModelID,
		SampleRate:  out.SampleRate,
		FrameFormat: out.FrameFormat,
	}, nil
}

// End forwards EndSession to the ASR side. Idempotent.
func (s *ASRTokenIssuer) End(ctx context.Context, callerID, callID, sessionID uuid.UUID) (*ports.EndSessionOutput, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("Call", callID.String())
		}
		return nil, apperr.Internal("failed to look up call")
	}
	if call.UserID != callerID {
		return nil, apperr.ForbiddenCode("ASR_ACCESS_DENIED", "caller does not own the call")
	}
	out, err := s.asrClient.EndSession(ctx, sessionID)
	if err != nil {
		catalog.AsrSessionEndFailed.Warn(s.logger,
			zap.Error(err),
			zap.String("session_id", sessionID.String()),
		)
		if errors.Is(err, ports.ErrASRUnavailable) {
			return nil, apperr.ServiceUnavailable("ASR_UNAVAILABLE",
				"asr service is unavailable", 0)
		}
		return nil, apperr.Internal("asr EndSession failed")
	}
	catalog.AsrSessionEndOk.Info(s.logger,
		zap.String("session_id", sessionID.String()),
		zap.Uint32("final_count", out.FinalCount),
	)

	// Best-effort persistence close: stitch the persisted segments into the
	// legacy calls.transcript so existing read paths keep working. The gRPC
	// EndSession's FinalTranscript is intentionally NOT used — the segment
	// table is the source of truth.
	if s.persister != nil {
		callID := callID
		if err := s.persister.CloseSession(ctx, CloseSessionInput{
			SessionID:    sessionID,
			CallerUserID: callerID,
			Status:       entities.TranscriptSessionStatusEnded,
			StitchInto:   &callID,
		}); err != nil {
			// Benign: the session never had a transcript_sessions row (session
			// predates persistence rollout, or OpenSession failed at issue
			// time). Log at Debug — there's nothing to "close" so this is a
			// no-op, not a failure.
			if errors.Is(err, ErrTranscriptSessionNotFound) {
				s.logger.Debug("transcript session close skipped: no row to close",
					zap.String("session_id", sessionID.String()),
				)
			} else {
				s.logger.Warn("transcript session close failed",
					zap.Error(err),
					zap.String("session_id", sessionID.String()),
				)
			}
		}
	}

	// Kat hook: drop the call cohort entry so the live-notes scheduler stops
	// ticking. Off the request path; failure is opaque to the caller.
	if s.katHooks != nil {
		cid := callID
		go func() {
			defer func() { _ = recover() }()
			s.katHooks.OnCallSessionEnded(cid)
		}()
	}

	return out, nil
}

// MeetingRequestInput is the input for RequestForMeeting().
type MeetingRequestInput struct {
	CallerID     uuid.UUID
	MeetingCode  string        // path param; resolved to meeting.id internally
	ModelID      string        // optional
	Language     string        // optional BCP-47
	RequestedTTL time.Duration // 0 → server default
}

// RequestForMeeting issues an ASR session token bound to a meeting.
//
// Authorization: caller must be an active participant of the meeting (i.e.
// have a meeting_participants row with joined_at IS NOT NULL AND left_at IS
// NULL). The host is implicitly an active participant once they've joined.
//
// Captions are a per-user opt-in: any active participant may request a
// session at any time. There is no meeting-wide flag and no host approval.
// Ended meetings return 410 MEETING_ENDED.
//
// The C++ ASR service has no notion of "call" vs "meeting" — both flows pass
// a UUID in the JWT's `cid` claim. Here we use meeting.id so the C++ side's
// SessionInfo and audit logs remain coherent.
func (s *ASRTokenIssuer) RequestForMeeting(ctx context.Context, in MeetingRequestInput) (*ASRSessionPayload, error) {
	if s.meetingRepo == nil || s.participantRepo == nil {
		return nil, apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
			"meeting transcription is not enabled in this environment", 0)
	}
	if in.RequestedTTL > 0 && in.RequestedTTL > s.cfg.MaxTTL {
		return nil, apperr.BadRequest(fmt.Sprintf(
			"requested_token_ttl_seconds exceeds max (%s)", s.cfg.MaxTTL))
	}

	meeting, err := s.meetingRepo.GetByCode(ctx, in.MeetingCode)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("Meeting", in.MeetingCode)
		}
		return nil, apperr.Internal("failed to look up meeting")
	}
	if meeting.IsEnded() {
		return nil, apperr.Gone("MEETING_ENDED", "meeting has ended")
	}

	// Active-participant check: host doesn't get a free pass — they too must
	// have actually joined the room before requesting captions.
	part, err := s.participantRepo.GetByMeetingAndUser(ctx, meeting.ID, in.CallerID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.ForbiddenCode("ASR_ACCESS_DENIED",
				"caller is not a participant of this meeting")
		}
		return nil, apperr.Internal("failed to verify meeting participation")
	}
	if part == nil || part.JoinedAt == nil || part.LeftAt != nil {
		return nil, apperr.ForbiddenCode("ASR_ACCESS_DENIED",
			"caller is not currently in this meeting")
	}

	out, err := s.asrClient.StartSession(ctx, ports.StartSessionInput{
		UserID:       in.CallerID,
		CallID:       meeting.ID, // reuses the cid claim slot for meeting.id
		ModelID:      in.ModelID,
		Language:     in.Language,
		RequestedTTL: in.RequestedTTL,
	})
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrASRAtCapacity):
			return nil, apperr.ServiceUnavailable("ASR_AT_CAPACITY",
				"asr service is at capacity", 5)
		case errors.Is(err, ports.ErrASRUnavailable):
			catalog.AsrCircuitOpen.Warn(s.logger,
				zap.String("meeting_code", in.MeetingCode),
			)
			return nil, apperr.ServiceUnavailable("ASR_UNAVAILABLE",
				"asr service is unavailable", 0)
		default:
			catalog.AsrTokenIssueFailed.Error(s.logger,
				zap.Error(err),
				zap.String("meeting_code", in.MeetingCode),
			)
			return nil, apperr.Internal("asr StartSession failed")
		}
	}

	token, err := s.signer.Sign(in.CallerID, meeting.ID, out.SessionID,
		out.ModelID, out.ExpiresAt)
	if err != nil {
		catalog.AsrTokenIssueFailed.Error(s.logger,
			zap.Error(err),
			zap.String("session_id", out.SessionID.String()),
		)
		return nil, apperr.Internal("asr token sign failed")
	}

	prefix := token
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	catalog.AsrTokenIssued.Info(s.logger,
		zap.String("user_id", in.CallerID.String()),
		zap.String("meeting_code", in.MeetingCode),
		zap.String("meeting_id", meeting.ID.String()),
		zap.String("session_id", out.SessionID.String()),
		zap.String("model_id", out.ModelID),
		zap.String("token_prefix", prefix),
		zap.Time("expires_at", out.ExpiresAt),
	)

	// Best-effort persistence open: meeting session bound to meeting.id.
	if s.persister != nil {
		meetingID := meeting.ID
		var langPtr *string
		if in.Language != "" {
			lang := in.Language
			langPtr = &lang
		}
		if err := s.persister.OpenSession(ctx, OpenSessionInput{
			SessionID:         out.SessionID,
			SpeakerUserID:     in.CallerID,
			MeetingID:         &meetingID,
			Engine:            s.engineSnapshotFor(ctx, out.ModelID),
			LanguageRequested: langPtr,
			SampleRate:        out.SampleRate,
			FrameFormat:       out.FrameFormat,
			ExpiresAt:         out.ExpiresAt,
		}); err != nil {
			s.logger.Warn("transcript session open failed; persistence degraded for this meeting",
				zap.Error(err),
				zap.String("session_id", out.SessionID.String()),
			)
		}
	}

	return &ASRSessionPayload{
		SessionID:    out.SessionID.String(),
		SessionToken: token,
		WsURL: fmt.Sprintf("%s/v1/asr/stream?token=%s",
			s.cfg.WSBaseURL, token),
		ExpiresAt:   out.ExpiresAt,
		ModelID:     out.ModelID,
		SampleRate:  out.SampleRate,
		FrameFormat: out.FrameFormat,
	}, nil
}

// EndForMeeting forwards EndSession after verifying the caller is/was a
// participant of `meetingCode`. Idempotent.
func (s *ASRTokenIssuer) EndForMeeting(ctx context.Context, callerID uuid.UUID, meetingCode string, sessionID uuid.UUID) (*ports.EndSessionOutput, error) {
	if s.meetingRepo == nil || s.participantRepo == nil {
		return nil, apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
			"meeting transcription is not enabled in this environment", 0)
	}
	meeting, err := s.meetingRepo.GetByCode(ctx, meetingCode)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("Meeting", meetingCode)
		}
		return nil, apperr.Internal("failed to look up meeting")
	}
	part, err := s.participantRepo.GetByMeetingAndUser(ctx, meeting.ID, callerID)
	if err != nil || part == nil {
		return nil, apperr.ForbiddenCode("ASR_ACCESS_DENIED",
			"caller is not a participant of this meeting")
	}
	out, err := s.asrClient.EndSession(ctx, sessionID)
	if err != nil {
		catalog.AsrSessionEndFailed.Warn(s.logger,
			zap.Error(err),
			zap.String("session_id", sessionID.String()),
		)
		if errors.Is(err, ports.ErrASRUnavailable) {
			return nil, apperr.ServiceUnavailable("ASR_UNAVAILABLE",
				"asr service is unavailable", 0)
		}
		return nil, apperr.Internal("asr EndSession failed")
	}
	catalog.AsrSessionEndOk.Info(s.logger,
		zap.String("session_id", sessionID.String()),
		zap.String("meeting_code", meetingCode),
		zap.Uint32("final_count", out.FinalCount),
	)

	// Best-effort persistence close. Meeting transcripts have no analogous
	// denormalised cache to refresh, so StitchInto is left nil.
	if s.persister != nil {
		if err := s.persister.CloseSession(ctx, CloseSessionInput{
			SessionID:    sessionID,
			CallerUserID: callerID,
			Status:       entities.TranscriptSessionStatusEnded,
		}); err != nil {
			// Benign: see the matching branch in End(). A pre-persistence
			// session has no row to close — log at Debug and move on.
			if errors.Is(err, ErrTranscriptSessionNotFound) {
				s.logger.Debug("transcript session close skipped: no row to close",
					zap.String("session_id", sessionID.String()),
				)
			} else {
				s.logger.Warn("transcript session close failed",
					zap.Error(err),
					zap.String("session_id", sessionID.String()),
				)
			}
		}
	}
	return out, nil
}

// _ silences the unused-import warning in environments where the ports
// package is referenced only through the entities namespace below.
var _ = entities.JSONMap{}
