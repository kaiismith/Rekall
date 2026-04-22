package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// ListMeetingsFilter controls filtering and ordering for ListByUser.
type ListMeetingsFilter struct {
	// Status is the user-facing status label: "in_progress", "complete",
	// "processing", or "failed". Nil means no filter.
	Status *string
	// Sort is one of: created_at_desc (default), created_at_asc,
	// duration_desc, duration_asc, title_asc, title_desc.
	Sort string
}

// ParticipantPreview is a lightweight user snapshot used in list responses.
type ParticipantPreview struct {
	UserID   uuid.UUID
	FullName string
	Initials string
}

// MeetingListItem bundles a Meeting with metadata computed at list time.
type MeetingListItem struct {
	*entities.Meeting
	// DurationSeconds is nil for in-progress meetings (ended_at not yet set).
	DurationSeconds     *int64
	ParticipantPreviews []ParticipantPreview
}

// MeetingRepository abstracts persistence for Meeting records.
type MeetingRepository interface {
	Create(ctx context.Context, m *entities.Meeting) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Meeting, error)
	GetByCode(ctx context.Context, code string) (*entities.Meeting, error)
	Update(ctx context.Context, m *entities.Meeting) error
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByHost returns all meetings created by the given host, optionally
	// filtered by status.
	ListByHost(ctx context.Context, hostID uuid.UUID, status string) ([]*entities.Meeting, error)

	// ListByUser returns meetings where the user is the host or a participant,
	// enriched with duration and up to 3 participant previews.
	ListByUser(ctx context.Context, userID uuid.UUID, filter ListMeetingsFilter) ([]*MeetingListItem, error)

	// CountActiveByHost counts meetings with status 'waiting' or 'active' for
	// the given host. Used to enforce the per-host limit.
	CountActiveByHost(ctx context.Context, hostID uuid.UUID) (int64, error)

	// Cleanup queries — used by the background cleanup job.

	// FindStaleWaiting returns waiting meetings that have not started within the
	// given timeout duration (i.e. created_at + timeout < now).
	FindStaleWaiting(ctx context.Context, timeout time.Duration) ([]*entities.Meeting, error)

	// FindStaleActive returns active meetings that have exceeded the maximum
	// allowed duration (i.e. started_at + maxDuration < now).
	FindStaleActive(ctx context.Context, maxDuration time.Duration) ([]*entities.Meeting, error)

	// FindActiveWithNoParticipants returns active meetings that currently have
	// zero participants with left_at IS NULL — meetings that were abandoned or
	// whose server process crashed before marking them ended.
	FindActiveWithNoParticipants(ctx context.Context) ([]*entities.Meeting, error)
}
