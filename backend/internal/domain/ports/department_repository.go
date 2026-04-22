package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// DepartmentRepository abstracts persistence for Department aggregates.
type DepartmentRepository interface {
	Create(ctx context.Context, dept *entities.Department) (*entities.Department, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Department, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.Department, error)
	Update(ctx context.Context, dept *entities.Department) (*entities.Department, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// DepartmentMembershipRepository abstracts persistence for DepartmentMembership records.
type DepartmentMembershipRepository interface {
	Create(ctx context.Context, m *entities.DepartmentMembership) error
	GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error)
	ListByDept(ctx context.Context, deptID uuid.UUID) ([]*entities.DepartmentMembership, error)
	Update(ctx context.Context, m *entities.DepartmentMembership) error
	Delete(ctx context.Context, deptID, userID uuid.UUID) error
}
