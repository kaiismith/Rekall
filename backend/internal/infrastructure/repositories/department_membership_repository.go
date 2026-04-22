package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"gorm.io/gorm"
)

// DepartmentMembershipRepository implements ports.DepartmentMembershipRepository using GORM.
type DepartmentMembershipRepository struct {
	db *gorm.DB
}

func NewDepartmentMembershipRepository(db *gorm.DB) *DepartmentMembershipRepository {
	return &DepartmentMembershipRepository{db: db}
}

func (r *DepartmentMembershipRepository) Create(ctx context.Context, m *entities.DepartmentMembership) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *DepartmentMembershipRepository) GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error) {
	var m entities.DepartmentMembership
	err := r.db.WithContext(ctx).
		Where("department_id = ? AND user_id = ?", deptID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("DepartmentMembership", deptID.String())
	}
	return &m, err
}

func (r *DepartmentMembershipRepository) ListByDept(ctx context.Context, deptID uuid.UUID) ([]*entities.DepartmentMembership, error) {
	var members []*entities.DepartmentMembership
	err := r.db.WithContext(ctx).
		Where("department_id = ?", deptID).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

func (r *DepartmentMembershipRepository) Update(ctx context.Context, m *entities.DepartmentMembership) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *DepartmentMembershipRepository) Delete(ctx context.Context, deptID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("department_id = ? AND user_id = ?", deptID, userID).
		Delete(&entities.DepartmentMembership{}).Error
}
