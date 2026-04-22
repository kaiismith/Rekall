package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// MeetingParticipantRepository abstracts persistence for MeetingParticipant records.
type MeetingParticipantRepository interface {
	Create(ctx context.Context, p *entities.MeetingParticipant) error
	GetByMeetingAndUser(ctx context.Context, meetingID, userID uuid.UUID) (*entities.MeetingParticipant, error)
	Update(ctx context.Context, p *entities.MeetingParticipant) error

	// ListActive returns all participants in a meeting who have not yet left
	// (left_at IS NULL).
	ListActive(ctx context.Context, meetingID uuid.UUID) ([]*entities.MeetingParticipant, error)

	// CountActive returns the number of participants currently in the meeting
	// (left_at IS NULL). Uses the partial index for efficiency.
	CountActive(ctx context.Context, meetingID uuid.UUID) (int64, error)

	// MarkAllLeft sets left_at = now for every active participant in the meeting.
	// Called when a meeting is ended (normally or by the cleanup job).
	MarkAllLeft(ctx context.Context, meetingID uuid.UUID) error
}
