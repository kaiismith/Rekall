package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// Sentinel errors returned by TranscriptPersister. Handlers map these onto
// HTTP/WS error codes; tests assert against them directly via errors.Is.
var (
	ErrTranscriptSessionNotFound = errors.New("transcript: session not found")
	ErrTranscriptSessionNotOwned = errors.New("transcript: session not owned by caller")
	ErrTranscriptSessionClosed   = errors.New("transcript: session not active")
	ErrTranscriptInvalidSegment  = errors.New("transcript: invalid segment payload")
)

// EngineSnapshot is the (mode, target, model_id) triple captured at session
// open from asrClient.Health() + StartSessionOutput. Snapshotted to the DB so
// later model rollovers don't retroactively rewrite the audit trail.
type EngineSnapshot struct {
	Mode    string // entities.TranscriptEngineMode{Local|OpenAI|Legacy}
	Target  string // model file path (local) or base_url (openai)
	ModelID string // canonical model name
}

// OpenSessionInput is the payload for TranscriptPersister.OpenSession.
// Exactly one of CallID and MeetingID must be non-nil.
type OpenSessionInput struct {
	SessionID         uuid.UUID
	SpeakerUserID     uuid.UUID
	CallID            *uuid.UUID
	MeetingID         *uuid.UUID
	Engine            EngineSnapshot
	LanguageRequested *string
	SampleRate        int32
	FrameFormat       string
	CorrelationID     *string
	ExpiresAt         time.Time
}

// RecordFinalInput is the payload for TranscriptPersister.RecordFinal.
// CallerUserID is checked against the session's speaker_user_id — a mismatch
// is the security signal that fires TranscriptSessionNotOwned.
type RecordFinalInput struct {
	SessionID    uuid.UUID
	CallerUserID uuid.UUID
	SegmentIndex int32
	Text         string
	Language     *string
	Confidence   *float32
	StartMs      int32
	EndMs        int32
	Words        []entities.WordTiming
}

// CloseSessionInput is the payload for TranscriptPersister.CloseSession.
// When StitchInto is set AND the session is bound to that Call, the persister
// rebuilds calls.transcript from the persisted segments.
type CloseSessionInput struct {
	SessionID    uuid.UUID
	CallerUserID uuid.UUID
	Status       entities.TranscriptSessionStatus
	ErrorCode    *string
	ErrorMessage *string
	StitchInto   *uuid.UUID
}

// TranscriptPersister owns the validation + write path for transcript_sessions
// and transcript_segments. Both the meeting WS hub and the solo-call HTTP
// endpoint share this single source of truth for "what is a valid segment
// write?".
type TranscriptPersister struct {
	repo             ports.TranscriptRepository
	callRepo         ports.CallRepository
	meetingRepo      ports.MeetingRepository
	insightPublisher ports.InsightPublisher
	logger           *zap.Logger
}

// NewTranscriptPersister wires the persister. callRepo and meetingRepo are
// used to resolve scope at session-open time and (for callRepo) to refresh
// the legacy calls.transcript denormalised cache at session close.
//
// The insight publisher is optional and wired in via WithInsightPublisher
// after construction so existing call sites (and tests) don't have to know
// about intellikat. When unset, CloseSession skips the publish step entirely.
func NewTranscriptPersister(
	repo ports.TranscriptRepository,
	callRepo ports.CallRepository,
	meetingRepo ports.MeetingRepository,
	logger *zap.Logger,
) *TranscriptPersister {
	return &TranscriptPersister{
		repo:        repo,
		callRepo:    callRepo,
		meetingRepo: meetingRepo,
		logger:      applogger.WithComponent(logger, "transcript_persister"),
	}
}

// WithInsightPublisher attaches the publisher used to fan a "session closed"
// reference message out to the intellikat consumer. Pass a NoopInsightPublisher
// when the feature is disabled. Returns the persister for fluent wiring at
// the composition root.
func (p *TranscriptPersister) WithInsightPublisher(pub ports.InsightPublisher) *TranscriptPersister {
	p.insightPublisher = pub
	return p
}

// OpenSession inserts a transcript_sessions row keyed by the ASR-issued
// session_id. Scope is resolved from the parent Call or Meeting and snapshotted
// onto the row (so later scope changes don't rewrite history).
//
// Returns ErrTranscriptInvalidSegment only on hard contract violations:
// missing/both parent ids, or missing UUIDs. Soft fields (engine snapshot
// fields that the upstream ASR service may not have populated yet) are
// substituted with `"unknown"` so the row still inserts — provenance is
// best-effort, but persistence must not be blocked by upstream omissions.
func (p *TranscriptPersister) OpenSession(ctx context.Context, in OpenSessionInput) error {
	if (in.CallID == nil) == (in.MeetingID == nil) {
		return ErrTranscriptInvalidSegment
	}
	if in.SessionID == uuid.Nil || in.SpeakerUserID == uuid.Nil {
		return ErrTranscriptInvalidSegment
	}
	if in.Engine.Mode == "" {
		in.Engine.Mode = entities.TranscriptEngineModeLocal
	}
	if in.Engine.ModelID == "" {
		// The C++ ASR service occasionally returns an empty model_id on
		// StartSession (e.g. OpenAI engine before the upstream wires the
		// canonical model name). Substitute a sentinel so the NOT NULL
		// column constraint holds; downstream consumers can filter on
		// `model_id = 'unknown'` to flag these.
		in.Engine.ModelID = "unknown"
	}

	var (
		scopeType *string
		scopeID   *uuid.UUID
	)
	switch {
	case in.CallID != nil:
		c, err := p.callRepo.GetByID(ctx, *in.CallID)
		if err != nil {
			return fmt.Errorf("transcript: load call for scope: %w", err)
		}
		scopeType, scopeID = c.ScopeType, c.ScopeID
	case in.MeetingID != nil:
		m, err := p.meetingRepo.GetByID(ctx, *in.MeetingID)
		if err != nil {
			return fmt.Errorf("transcript: load meeting for scope: %w", err)
		}
		scopeType, scopeID = m.ScopeType, m.ScopeID
	}

	row := &entities.TranscriptSession{
		ID:                in.SessionID,
		SpeakerUserID:     in.SpeakerUserID,
		CallID:            in.CallID,
		MeetingID:         in.MeetingID,
		ScopeType:         scopeType,
		ScopeID:           scopeID,
		EngineMode:        in.Engine.Mode,
		EngineTarget:      in.Engine.Target,
		ModelID:           in.Engine.ModelID,
		LanguageRequested: in.LanguageRequested,
		SampleRate:        in.SampleRate,
		FrameFormat:       in.FrameFormat,
		CorrelationID:     in.CorrelationID,
		Status:            entities.TranscriptSessionStatusActive,
		StartedAt:         time.Now().UTC(),
		ExpiresAt:         in.ExpiresAt,
	}
	if err := p.repo.CreateSession(ctx, row); err != nil {
		catalog.TranscriptSessionOpenFailed.Warn(p.logger,
			zap.String("session_id", in.SessionID.String()),
			zap.Error(err),
		)
		return err
	}
	catalog.TranscriptSessionOpened.Info(p.logger,
		zap.String("session_id", in.SessionID.String()),
		zap.String("speaker_user_id", in.SpeakerUserID.String()),
		zap.String("engine_mode", in.Engine.Mode),
		zap.String("model_id", in.Engine.ModelID),
	)
	return nil
}

// RecordFinal upserts a transcript_segments row for one ASR `final` event.
// Validation: text non-empty, end_ms > start_ms, segment_index >= 0,
// session exists, caller owns the session, session is still active.
//
// On idempotent retransmission (same session_id + segment_index) the row is
// updated in place and the session counter is NOT re-incremented (handled in
// the repo via the xmax = 0 trick).
func (p *TranscriptPersister) RecordFinal(ctx context.Context, in RecordFinalInput) error {
	if in.Text == "" || in.EndMs <= in.StartMs || in.SegmentIndex < 0 {
		return ErrTranscriptInvalidSegment
	}

	sess, err := p.repo.GetSession(ctx, in.SessionID)
	if err != nil {
		var ae *apperr.AppError
		if errors.As(err, &ae) && ae.Code == "NOT_FOUND" {
			return ErrTranscriptSessionNotFound
		}
		return err
	}
	if sess.SpeakerUserID != in.CallerUserID {
		catalog.TranscriptSessionNotOwned.Warn(p.logger,
			zap.String("session_id", in.SessionID.String()),
			zap.String("session_speaker", sess.SpeakerUserID.String()),
			zap.String("caller", in.CallerUserID.String()),
		)
		return ErrTranscriptSessionNotOwned
	}
	if !sess.IsActive() {
		return ErrTranscriptSessionClosed
	}

	segStartedAt := sess.StartedAt.Add(time.Duration(in.StartMs) * time.Millisecond)
	seg := &entities.TranscriptSegment{
		SessionID:        in.SessionID,
		SegmentIndex:     in.SegmentIndex,
		SpeakerUserID:    in.CallerUserID,
		CallID:           sess.CallID,
		MeetingID:        sess.MeetingID,
		Text:             in.Text,
		Language:         in.Language,
		Confidence:       in.Confidence,
		StartMs:          in.StartMs,
		EndMs:            in.EndMs,
		Words:            entities.WordTimings(in.Words),
		EngineMode:       sess.EngineMode,
		ModelID:          sess.ModelID,
		SegmentStartedAt: segStartedAt,
	}
	if err := p.repo.UpsertSegment(ctx, seg); err != nil {
		catalog.TranscriptPersistFailed.Warn(p.logger,
			zap.String("session_id", in.SessionID.String()),
			zap.Int32("segment_index", in.SegmentIndex),
			zap.Error(err),
		)
		return err
	}
	catalog.TranscriptSegmentPersisted.Debug(p.logger,
		zap.String("session_id", in.SessionID.String()),
		zap.Int32("segment_index", in.SegmentIndex),
		zap.Int32("duration_ms", in.EndMs-in.StartMs),
	)
	return nil
}

// CloseSession flips the session row to a terminal state. When StitchInto
// names the session's bound Call, calls.transcript is rebuilt from the
// persisted segments (replacing the legacy gRPC FinalTranscript path).
//
// Stitch failures DO NOT roll back the status flip — better to have an
// 'ended' session with a stale calls.transcript than a perpetually 'active'
// session that the cleanup job retries forever.
func (p *TranscriptPersister) CloseSession(ctx context.Context, in CloseSessionInput) error {
	sess, err := p.repo.GetSession(ctx, in.SessionID)
	if err != nil {
		var ae *apperr.AppError
		if errors.As(err, &ae) && ae.Code == "NOT_FOUND" {
			return ErrTranscriptSessionNotFound
		}
		return err
	}
	if in.CallerUserID != uuid.Nil && sess.SpeakerUserID != in.CallerUserID {
		catalog.TranscriptSessionNotOwned.Warn(p.logger,
			zap.String("session_id", in.SessionID.String()),
			zap.String("session_speaker", sess.SpeakerUserID.String()),
			zap.String("caller", in.CallerUserID.String()),
		)
		return ErrTranscriptSessionNotOwned
	}

	status := in.Status
	if status == "" || status == entities.TranscriptSessionStatusActive {
		status = entities.TranscriptSessionStatusEnded
	}

	if err := p.repo.UpdateSessionStatus(ctx, in.SessionID, status, in.ErrorCode, in.ErrorMessage); err != nil {
		return err
	}

	catalog.TranscriptSessionClosed.Info(p.logger,
		zap.String("session_id", in.SessionID.String()),
		zap.String("status", string(status)),
		zap.Int32("segments", sess.FinalizedSegmentCount),
	)

	if in.StitchInto != nil && sess.CallID != nil && *in.StitchInto == *sess.CallID {
		text, err := p.repo.StitchCall(ctx, *sess.CallID)
		if err != nil {
			catalog.TranscriptStitchFailed.Error(p.logger,
				zap.String("session_id", in.SessionID.String()),
				zap.String("call_id", sess.CallID.String()),
				zap.Error(err),
			)
			return nil
		}
		call, err := p.callRepo.GetByID(ctx, *sess.CallID)
		if err != nil {
			catalog.TranscriptStitchFailed.Error(p.logger,
				zap.String("session_id", in.SessionID.String()),
				zap.String("call_id", sess.CallID.String()),
				zap.Error(err),
			)
			return nil
		}
		call.Transcript = &text
		if _, err := p.callRepo.Update(ctx, call); err != nil {
			catalog.TranscriptStitchFailed.Error(p.logger,
				zap.String("session_id", in.SessionID.String()),
				zap.String("call_id", sess.CallID.String()),
				zap.Error(err),
			)
		}
	}

	// Re-fetch the now-closed session so the publisher sees the final
	// status / counters. Best-effort — a publish failure is logged but
	// never propagated; the DB close is the source of truth.
	if p.insightPublisher != nil {
		closed, err := p.repo.GetSession(ctx, in.SessionID)
		if err == nil && closed != nil {
			correlationID := ""
			if closed.CorrelationID != nil {
				correlationID = *closed.CorrelationID
			}
			// Run on a background goroutine so the close response isn't
			// blocked by Service Bus latency. Use context.Background to
			// avoid cancellation when the request context returns.
			go p.insightPublisher.PublishSessionClosed(
				context.Background(), closed, correlationID,
			)
		}
	}
	return nil
}
