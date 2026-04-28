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

// DepartmentRepository implements ports.DepartmentRepository using GORM.
type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) Create(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	if err := r.db.WithContext(ctx).Create(dept).Error; err != nil {
		return nil, err
	}
	return dept, nil
}

func (r *DepartmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Department, error) {
	var dept entities.Department
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Department", id.String())
	}
	return &dept, err
}

func (r *DepartmentRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.Department, error) {
	var depts []*entities.Department
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND deleted_at IS NULL", orgID).
		Order("created_at ASC").
		Find(&depts).Error
	return depts, err
}

func (r *DepartmentRepository) Update(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	if err := r.db.WithContext(ctx).Save(dept).Error; err != nil {
		return nil, err
	}
	return dept, nil
}

func (r *DepartmentRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&entities.Department{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("Department", id.String())
	}
	return nil
}
