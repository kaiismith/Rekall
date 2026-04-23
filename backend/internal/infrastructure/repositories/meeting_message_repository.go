package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"gorm.io/gorm"
)

// MeetingMessageRepository implements ports.MeetingMessageRepository using GORM.
type MeetingMessageRepository struct {
	db *gorm.DB
}

func NewMeetingMessageRepository(db *gorm.DB) *MeetingMessageRepository {
	return &MeetingMessageRepository{db: db}
}

func (r *MeetingMessageRepository) Create(ctx context.Context, m *entities.MeetingMessage) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// ListByMeeting returns the most recent messages for a meeting, ordered ASC
// for direct client consumption. The query fetches limit+1 rows to detect
// whether additional older messages exist beyond the page.
func (r *MeetingMessageRepository) ListByMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	before *time.Time,
	limit int,
) ([]*entities.MeetingMessage, bool, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	q := r.db.WithContext(ctx).
		Where("meeting_id = ? AND deleted_at IS NULL", meetingID)
	if before != nil {
		q = q.Where("sent_at < ?", *before)
	}

	var rows []*entities.MeetingMessage
	if err := q.Order("sent_at DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	// Reverse DESC → ASC for client consumption.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	return rows, hasMore, nil
}
