package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// MeetingMessageRepository abstracts persistence for MeetingMessage records.
type MeetingMessageRepository interface {
	Create(ctx context.Context, m *entities.MeetingMessage) error

	// ListByMeeting returns messages for a meeting ordered by sent_at ASC.
	// When before is non-nil, only messages strictly older than that timestamp
	// are returned (used for cursor-based pagination of older history).
	// limit is clamped to [1, 100] by the implementation.
	// hasMore reports whether additional older rows exist beyond the page.
	ListByMeeting(
		ctx context.Context,
		meetingID uuid.UUID,
		before *time.Time,
		limit int,
	) (messages []*entities.MeetingMessage, hasMore bool, err error)
}
