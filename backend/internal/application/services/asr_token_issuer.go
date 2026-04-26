package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/asr"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

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

// ASRTokenIssuer mints short-lived JWTs binding a user to an asr session.
// It is the Go-side complement of the C++ JWTValidator.
type ASRTokenIssuer struct {
	asrClient       ports.ASRClient
	callRepo        ports.CallRepository
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	signer          *asr.TokenSigner
	cfg             ASRTokenIssuerConfig
	logger          *zap.Logger
}

// NewASRTokenIssuer wires the issuer. callRepo is required for the call flow;
// meetingRepo/participantRepo may be nil when only the call flow is in use
// (the meeting endpoint then returns 503 ASR_NOT_CONFIGURED).
func NewASRTokenIssuer(
	asrClient ports.ASRClient,
	callRepo ports.CallRepository,
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	signer *asr.TokenSigner,
	cfg ASRTokenIssuerConfig,
	logger *zap.Logger,
) *ASRTokenIssuer {
	return &ASRTokenIssuer{
		asrClient:       asrClient,
		callRepo:        callRepo,
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		signer:          signer,
		cfg:             cfg,
		logger:          applogger.WithComponent(logger, "asr_token_issuer"),
	}
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
// Gating: the meeting's `transcription_enabled` flag MUST be true; otherwise
// the endpoint returns 403 TRANSCRIPTION_DISABLED. Ended meetings return
// 410 MEETING_ENDED.
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
	if !meeting.TranscriptionEnabled {
		return nil, apperr.ForbiddenCode("TRANSCRIPTION_DISABLED",
			"live captions are not enabled for this meeting")
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
	return out, nil
}

// _ silences the unused-import warning in environments where the ports
// package is referenced only through the entities namespace below.
var _ = entities.JSONMap{}
