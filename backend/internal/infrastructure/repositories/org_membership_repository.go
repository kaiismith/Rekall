package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"gorm.io/gorm"
)

// OrgMembershipRepository implements ports.OrgMembershipRepository using GORM.
type OrgMembershipRepository struct {
	db *gorm.DB
}

func NewOrgMembershipRepository(db *gorm.DB) *OrgMembershipRepository {
	return &OrgMembershipRepository{db: db}
}

func (r *OrgMembershipRepository) Create(ctx context.Context, m *entities.OrgMembership) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *OrgMembershipRepository) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*entities.OrgMembership, error) {
	var m entities.OrgMembership
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("OrgMembership", orgID.String())
	}
	return &m, err
}

func (r *OrgMembershipRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.OrgMembership, error) {
	var members []*entities.OrgMembership
	err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

func (r *OrgMembershipRepository) Update(ctx context.Context, m *entities.OrgMembership) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *OrgMembershipRepository) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Delete(&entities.OrgMembership{}).Error
}
