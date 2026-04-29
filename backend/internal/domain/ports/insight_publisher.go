package ports

import (
	"context"

	"github.com/rekall/backend/internal/domain/entities"
)

// InsightPublisher emits a small reference message to Service Bus when a
// transcript session reaches a terminal state. The message body carries
// IDENTIFIERS only — no transcript text — and the downstream consumer
// (intellikat) reads the canonical data from the shared DB.
//
// PublishSessionClosed is best-effort: a failure must NOT roll back the
// session close. A future "reprocess" admin tool covers missed publishes.
type InsightPublisher interface {
	PublishSessionClosed(ctx context.Context, session *entities.TranscriptSession, correlationID string)
	Close() error
}
