package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// MeetingRepository implements ports.MeetingRepository using GORM.
type MeetingRepository struct {
	db *gorm.DB
}

func NewMeetingRepository(db *gorm.DB) *MeetingRepository {
	return &MeetingRepository{db: db}
}

func (r *MeetingRepository) Create(ctx context.Context, m *entities.Meeting) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MeetingRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Meeting, error) {
	var m entities.Meeting
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Meeting", id.String())
	}
	return &m, err
}

func (r *MeetingRepository) GetByCode(ctx context.Context, code string) (*entities.Meeting, error) {
	var m entities.Meeting
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Meeting", code)
	}
	return &m, err
}

func (r *MeetingRepository) Update(ctx context.Context, m *entities.Meeting) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *MeetingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Meeting{}, "id = ?", id).Error
}

func (r *MeetingRepository) ListByHost(ctx context.Context, hostID uuid.UUID, status string) ([]*entities.Meeting, error) {
	var meetings []*entities.Meeting
	q := r.db.WithContext(ctx).Where("host_id = ?", hostID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Find(&meetings).Error
	return meetings, err
}

func (r *MeetingRepository) CountActiveByHost(ctx context.Context, hostID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Meeting{}).
		Where("host_id = ? AND status IN ?", hostID, []string{entities.MeetingStatusWaiting, entities.MeetingStatusActive}).
		Count(&count).Error
	return count, err
}

func (r *MeetingRepository) FindStaleWaiting(ctx context.Context, timeout time.Duration) ([]*entities.Meeting, error) {
	var meetings []*entities.Meeting
	cutoff := time.Now().UTC().Add(-timeout)
	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", entities.MeetingStatusWaiting, cutoff).
		Find(&meetings).Error
	return meetings, err
}

func (r *MeetingRepository) FindStaleActive(ctx context.Context, maxDuration time.Duration) ([]*entities.Meeting, error) {
	var meetings []*entities.Meeting
	cutoff := time.Now().UTC().Add(-maxDuration)
	err := r.db.WithContext(ctx).
		Where("status = ? AND started_at < ?", entities.MeetingStatusActive, cutoff).
		Find(&meetings).Error
	return meetings, err
}

func (r *MeetingRepository) FindActiveWithNoParticipants(ctx context.Context) ([]*entities.Meeting, error) {
	var meetings []*entities.Meeting
	err := r.db.WithContext(ctx).
		Where(`status = ? AND id NOT IN (
			SELECT DISTINCT meeting_id FROM meeting_participants WHERE left_at IS NULL
		)`, entities.MeetingStatusActive).
		Find(&meetings).Error
	return meetings, err
}

// ListByUser returns meetings where userID is the host or a participant,
// enriched with computed duration and up to 3 participant previews.
func (r *MeetingRepository) ListByUser(ctx context.Context, userID uuid.UUID, filter ports.ListMeetingsFilter) ([]*ports.MeetingListItem, error) {
	var meetings []*entities.Meeting

	q := r.db.WithContext(ctx)

	// When a scope filter is active, membership in org/department scopes is the
	// authorisation gate (enforced by the service); every meeting in the scope
	// is visible to its members regardless of host/participant status.
	//
	// For Open scope we layer the host-or-participant rule on top because open
	// items are addressed personally ("my open meetings"), not by group
	// membership. No scope filter preserves the historical default.
	if filter.Scope != nil {
		switch filter.Scope.Kind {
		case ports.ScopeKindOpen:
			q = q.Where(
				"scope_type IS NULL AND (host_id = ? OR id IN (SELECT meeting_id FROM meeting_participants WHERE user_id = ?))",
				userID, userID,
			)
		case ports.ScopeKindOrganization:
			q = q.Where("scope_type = ? AND scope_id = ?", entities.MeetingScopeOrg, filter.Scope.ID)
		case ports.ScopeKindDepartment:
			q = q.Where("scope_type = ? AND scope_id = ?", entities.MeetingScopeDept, filter.Scope.ID)
		}
	} else {
		q = q.Where(
			"host_id = ? OR id IN (SELECT meeting_id FROM meeting_participants WHERE user_id = ?)",
			userID, userID,
		)
	}

	// Translate the user-facing status label to internal meeting statuses.
	if filter.Status != nil {
		switch *filter.Status {
		case "in_progress":
			q = q.Where("status IN ?", []string{entities.MeetingStatusWaiting, entities.MeetingStatusActive})
		case "complete":
			q = q.Where("status = ?", entities.MeetingStatusEnded)
		case "processing", "failed":
			// Reserved for future pipeline use — always empty for now.
			return []*ports.MeetingListItem{}, nil
		}
	}

	if err := q.Order(listSortExpr(filter.Sort)).Find(&meetings).Error; err != nil {
		return nil, err
	}
	if len(meetings) == 0 {
		return []*ports.MeetingListItem{}, nil
	}

	// Batch-fetch participant previews for all returned meetings.
	meetingIDs := make([]uuid.UUID, len(meetings))
	for i, m := range meetings {
		meetingIDs[i] = m.ID
	}

	var previewRows []struct {
		MeetingID uuid.UUID `gorm:"column:meeting_id"`
		UserID    uuid.UUID `gorm:"column:user_id"`
		FullName  string    `gorm:"column:full_name"`
	}
	if err := r.db.WithContext(ctx).
		Table("meeting_participants mp").
		Select("mp.meeting_id, u.id AS user_id, u.full_name").
		Joins("JOIN users u ON u.id = mp.user_id").
		Where("mp.meeting_id IN ?", meetingIDs).
		Order("mp.meeting_id, mp.joined_at ASC NULLS LAST").
		Scan(&previewRows).Error; err != nil {
		return nil, err
	}

	// Cap at 3 previews per meeting.
	previewMap := make(map[uuid.UUID][]ports.ParticipantPreview, len(meetings))
	for _, row := range previewRows {
		if len(previewMap[row.MeetingID]) < 3 {
			previewMap[row.MeetingID] = append(previewMap[row.MeetingID], ports.ParticipantPreview{
				UserID:   row.UserID,
				FullName: row.FullName,
				Initials: meetingInitials(row.FullName),
			})
		}
	}

	items := make([]*ports.MeetingListItem, len(meetings))
	for i, m := range meetings {
		item := &ports.MeetingListItem{
			Meeting:             m,
			ParticipantPreviews: previewMap[m.ID],
		}
		if m.StartedAt != nil && m.EndedAt != nil {
			d := int64(m.EndedAt.Sub(*m.StartedAt).Seconds())
			item.DurationSeconds = &d
		}
		items[i] = item
	}
	return items, nil
}

// listSortExpr maps the sort key from the API to an ORDER BY clause.
func listSortExpr(sort string) string {
	switch sort {
	case "created_at_asc":
		return "created_at ASC"
	case "duration_desc":
		return "(CASE WHEN ended_at IS NOT NULL AND started_at IS NOT NULL THEN EXTRACT(EPOCH FROM (ended_at - started_at)) ELSE NULL END) DESC NULLS LAST"
	case "duration_asc":
		return "(CASE WHEN ended_at IS NOT NULL AND started_at IS NOT NULL THEN EXTRACT(EPOCH FROM (ended_at - started_at)) ELSE NULL END) ASC NULLS LAST"
	case "title_asc":
		return "title ASC"
	case "title_desc":
		return "title DESC"
	default: // "created_at_desc" and any unrecognised value
		return "created_at DESC"
	}
}

// meetingInitials returns up to two uppercase initials from a full name.
func meetingInitials(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "?"
	}
	first := strings.ToUpper(string([]rune(parts[0])[:1]))
	if len(parts) == 1 {
		return first
	}
	return first + strings.ToUpper(string([]rune(parts[len(parts)-1])[:1]))
}
