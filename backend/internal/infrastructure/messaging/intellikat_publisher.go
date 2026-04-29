// Package messaging contains adapters that publish reference messages to
// out-of-process consumers. Intellikat's transcript-insights pipeline is the
// first such consumer; future search/indexing/archival workers attach as
// additional Service Bus subscriptions on the same topic.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/pkg/config"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

const (
	schemaVersion  = "1"
	publishTimeout = 5 * time.Second
)

// JobReferencePayload is the on-the-wire shape of the message body. It is
// versioned (`SchemaVersion`) so consumers can reject unknown shapes
// explicitly. Crucially: NO transcript text, no per-word timings, no PII.
type JobReferencePayload struct {
	SchemaVersion       string         `json:"schema_version"`
	JobID               string         `json:"job_id"`
	EventType           string         `json:"event_type"`
	TranscriptSessionID string         `json:"transcript_session_id"`
	Scope               scopePayload   `json:"scope"`
	SegmentIndexRange   *segmentRange  `json:"segment_index_range,omitempty"`
	SpeakerUserID       string         `json:"speaker_user_id"`
	EngineSnapshot      engineSnapshot `json:"engine_snapshot"`
	OccurredAt          string         `json:"occurred_at"`
	CorrelationID       string         `json:"correlation_id,omitempty"`
}

type scopePayload struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type segmentRange struct {
	From int32 `json:"from"`
	To   int32 `json:"to"`
}

type engineSnapshot struct {
	EngineMode string `json:"engine_mode"`
	ModelID    string `json:"model_id"`
}

// IntellikatPublisher implements ports.InsightPublisher over Azure Service
// Bus. When PublishEnabled is false the constructor returns a no-op
// publisher so callers don't need to special-case the disabled state.
type IntellikatPublisher struct {
	enabled bool
	topic   string
	client  *azservicebus.Client
	sender  *azservicebus.Sender
	logger  *zap.Logger
}

// NewIntellikatPublisher constructs the adapter. Returns a NoopInsightPublisher
// when PublishEnabled is false; that way the wider DI graph stays simple.
func NewIntellikatPublisher(cfg config.IntellikatConfig, logger *zap.Logger) (ports.InsightPublisher, error) {
	log := applogger.WithComponent(logger, "intellikat_publisher")
	if !cfg.PublishEnabled {
		log.Info("intellikat publish disabled — operating as no-op")
		return &NoopInsightPublisher{}, nil
	}

	var (
		client *azservicebus.Client
		err    error
	)
	if cfg.FullyQualifiedNamespace != "" {
		credential, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("intellikat publisher: managed identity: %w", credErr)
		}
		client, err = azservicebus.NewClient(cfg.FullyQualifiedNamespace, credential, nil)
	} else {
		client, err = azservicebus.NewClientFromConnectionString(cfg.ConnectionString, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("intellikat publisher: service bus client: %w", err)
	}

	sender, err := client.NewSender(cfg.Topic, nil)
	if err != nil {
		_ = client.Close(context.Background())
		return nil, fmt.Errorf("intellikat publisher: sender for topic %q: %w", cfg.Topic, err)
	}

	return &IntellikatPublisher{
		enabled: true,
		topic:   cfg.Topic,
		client:  client,
		sender:  sender,
		logger:  log,
	}, nil
}

// PublishSessionClosed emits the reference message for one closed session.
// Failures are logged at warn — never bubbled up — so a Service Bus outage
// can't roll back a successful DB close.
func (p *IntellikatPublisher) PublishSessionClosed(
	ctx context.Context,
	session *entities.TranscriptSession,
	correlationID string,
) {
	if !p.enabled || session == nil {
		return
	}

	payload := JobReferencePayload{
		SchemaVersion:       schemaVersion,
		JobID:               uuid.NewString(),
		EventType:           "transcript.session.closed",
		TranscriptSessionID: session.ID.String(),
		Scope:               scopeOf(session),
		SegmentIndexRange:   segmentRangeOf(session),
		SpeakerUserID:       session.SpeakerUserID.String(),
		EngineSnapshot: engineSnapshot{
			EngineMode: session.EngineMode,
			ModelID:    session.ModelID,
		},
		OccurredAt:    time.Now().UTC().Format(time.RFC3339Nano),
		CorrelationID: correlationID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		catalog.IntellikatPublishFailed.Warn(p.logger,
			zap.String("transcript_session_id", payload.TranscriptSessionID),
			zap.Error(err),
		)
		return
	}

	contentType := "application/json"
	msg := &azservicebus.Message{
		Body:          body,
		ContentType:   &contentType,
		MessageID:     &payload.JobID,
		CorrelationID: &correlationID,
		ApplicationProperties: map[string]any{
			"event_type":     payload.EventType,
			"schema_version": payload.SchemaVersion,
		},
	}

	sendCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if err := p.sender.SendMessage(sendCtx, msg, nil); err != nil {
		catalog.IntellikatPublishFailed.Warn(p.logger,
			zap.String("transcript_session_id", payload.TranscriptSessionID),
			zap.String("topic", p.topic),
			zap.Error(err),
		)
		return
	}

	catalog.IntellikatPublishOK.Info(p.logger,
		zap.String("transcript_session_id", payload.TranscriptSessionID),
		zap.String("job_id", payload.JobID),
		zap.String("event_type", payload.EventType),
	)
}

// Close releases the sender + client. Safe to call on the no-op too.
func (p *IntellikatPublisher) Close() error {
	if !p.enabled {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()
	if p.sender != nil {
		_ = p.sender.Close(ctx)
	}
	if p.client != nil {
		return p.client.Close(ctx)
	}
	return nil
}

func scopeOf(s *entities.TranscriptSession) scopePayload {
	if s.MeetingID != nil {
		return scopePayload{Kind: "meeting", ID: s.MeetingID.String()}
	}
	if s.CallID != nil {
		return scopePayload{Kind: "call", ID: s.CallID.String()}
	}
	// Defensive — schema CHECK guarantees one of the two.
	return scopePayload{Kind: "unknown", ID: uuid.Nil.String()}
}

func segmentRangeOf(s *entities.TranscriptSession) *segmentRange {
	if s.FinalizedSegmentCount <= 0 {
		return nil
	}
	return &segmentRange{From: 0, To: s.FinalizedSegmentCount - 1}
}

// NoopInsightPublisher is the disabled-flag fallback. Centralises the
// "intellikat off" branch so callers stay flag-agnostic.
type NoopInsightPublisher struct{}

func (n *NoopInsightPublisher) PublishSessionClosed(
	_ context.Context, _ *entities.TranscriptSession, _ string,
) {
}

func (n *NoopInsightPublisher) Close() error { return nil }
