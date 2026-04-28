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

// OrganizationRepository implements ports.OrganizationRepository using GORM.
type OrganizationRepository struct {
	db *gorm.DB
}

func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	if err := r.db.WithContext(ctx).Create(org).Error; err != nil {
		return nil, err
	}
	return org, nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error) {
	var org entities.Organization
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&org).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Organization", id.String())
	}
	return &org, err
}

func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*entities.Organization, error) {
	var org entities.Organization
	err := r.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&org).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Organization", slug)
	}
	return &org, err
}

func (r *OrganizationRepository) Update(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	if err := r.db.WithContext(ctx).Save(org).Error; err != nil {
		return nil, err
	}
	return org, nil
}

func (r *OrganizationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.Organization{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error
}

func (r *OrganizationRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Organization, error) {
	var orgs []*entities.Organization
	err := r.db.WithContext(ctx).
		Joins("JOIN org_memberships ON org_memberships.org_id = organizations.id AND org_memberships.user_id = ?", userID).
		Where("organizations.deleted_at IS NULL").
		Order("organizations.created_at DESC").
		Find(&orgs).Error
	return orgs, err
}
