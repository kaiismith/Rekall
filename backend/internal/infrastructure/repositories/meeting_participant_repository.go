package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
)

// MeetingParticipantRepository implements ports.MeetingParticipantRepository using GORM.
type MeetingParticipantRepository struct {
	db *gorm.DB
}

func NewMeetingParticipantRepository(db *gorm.DB) *MeetingParticipantRepository {
	return &MeetingParticipantRepository{db: db}
}

func (r *MeetingParticipantRepository) Create(ctx context.Context, p *entities.MeetingParticipant) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *MeetingParticipantRepository) GetByMeetingAndUser(ctx context.Context, meetingID, userID uuid.UUID) (*entities.MeetingParticipant, error) {
	var p entities.MeetingParticipant
	err := r.db.WithContext(ctx).
		Where("meeting_id = ? AND user_id = ?", meetingID, userID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("MeetingParticipant", meetingID.String())
	}
	return &p, err
}

func (r *MeetingParticipantRepository) Update(ctx context.Context, p *entities.MeetingParticipant) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *MeetingParticipantRepository) ListActive(ctx context.Context, meetingID uuid.UUID) ([]*entities.MeetingParticipant, error) {
	var participants []*entities.MeetingParticipant
	err := r.db.WithContext(ctx).
		Where("meeting_id = ? AND left_at IS NULL", meetingID).
		Find(&participants).Error
	return participants, err
}

func (r *MeetingParticipantRepository) CountActive(ctx context.Context, meetingID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.MeetingParticipant{}).
		Where("meeting_id = ? AND left_at IS NULL", meetingID).
		Count(&count).Error
	return count, err
}

func (r *MeetingParticipantRepository) MarkAllLeft(ctx context.Context, meetingID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&entities.MeetingParticipant{}).
		Where("meeting_id = ? AND left_at IS NULL", meetingID).
		Update("left_at", now).Error
}
