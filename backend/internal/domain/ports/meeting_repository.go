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
	// Scope, when non-nil, restricts the result to meetings matching the given
	// scope. Nil preserves the caller's default visibility (host + participant).
	Scope *ScopeFilter
	// Page is 1-indexed. Zero/negative values default to 1 in the repo.
	Page int
	// PerPage caps the page size. Zero/negative values default to 20; the
	// repository clamps to a reasonable upper bound (200) so a malicious
	// request can't ask for an unbounded page.
	PerPage int
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
	// enriched with duration and up to 3 participant previews. The result is
	// paginated by filter.Page / filter.PerPage; the second return value is
	// the total count of matching rows so callers can compute total_pages.
	ListByUser(ctx context.Context, userID uuid.UUID, filter ListMeetingsFilter) ([]*MeetingListItem, int, error)

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
