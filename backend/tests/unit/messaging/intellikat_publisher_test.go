package messaging_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/infrastructure/messaging"
	"github.com/rekall/backend/pkg/config"
)

// ── disabled publisher returns a no-op ───────────────────────────────────────

func TestNewIntellikatPublisher_DisabledReturnsNoop(t *testing.T) {
	pub, err := messaging.NewIntellikatPublisher(
		config.IntellikatConfig{PublishEnabled: false},
		zap.NewNop(),
	)
	require.NoError(t, err)

	_, isNoop := pub.(*messaging.NoopInsightPublisher)
	assert.True(t, isNoop, "disabled flag must yield NoopInsightPublisher")

	// No-op must not panic on calls and must close cleanly.
	pub.PublishSessionClosed(context.Background(), &entities.TranscriptSession{}, "cid")
	pub.PublishSessionClosed(context.Background(), nil, "")
	assert.NoError(t, pub.Close())
}

// ── on-the-wire JSON shape ───────────────────────────────────────────────────
//
// The publisher requires a real Service Bus broker to exercise the send path;
// success/failure coverage of the network-side behaviour lives in an
// integration test gated by SERVICEBUS_CONNECTION_STRING.
//
// What we CAN verify here is the contract: the JobReferencePayload carries
// IDs only and never the transcript text.

func TestJobReferencePayload_JSONShapeAndNoTextLeak(t *testing.T) {
	sessionID := uuid.New()
	speakerID := uuid.New()
	callID := uuid.New()
	jobID := uuid.New().String()

	payload := messaging.JobReferencePayload{
		SchemaVersion:       "1",
		JobID:               jobID,
		EventType:           "transcript.session.closed",
		TranscriptSessionID: sessionID.String(),
		SpeakerUserID:       speakerID.String(),
		OccurredAt:          time.Now().UTC().Format(time.RFC3339Nano),
		CorrelationID:       "cid-test",
	}

	bytes, err := json.Marshal(payload)
	require.NoError(t, err)
	body := string(bytes)

	assert.Contains(t, body, `"schema_version":"1"`)
	assert.Contains(t, body, `"event_type":"transcript.session.closed"`)
	assert.Contains(t, body, `"transcript_session_id":"`+sessionID.String()+`"`)
	assert.Contains(t, body, `"speaker_user_id":"`+speakerID.String()+`"`)
	assert.Contains(t, body, `"correlation_id":"cid-test"`)

	// Round-trip and verify a synthetic transcript text was NEVER on the wire.
	var roundTrip map[string]any
	require.NoError(t, json.Unmarshal(bytes, &roundTrip))
	for k := range roundTrip {
		assert.NotContains(t, k, "text", "payload must not carry any *text fields: %s", k)
		assert.NotContains(t, k, "content", "payload must not carry any *content fields: %s", k)
	}

	_ = callID
}
