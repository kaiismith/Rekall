package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// OrgMembershipRepository abstracts persistence for OrgMembership records.
type OrgMembershipRepository interface {
	Create(ctx context.Context, m *entities.OrgMembership) error
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*entities.OrgMembership, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.OrgMembership, error)
	Update(ctx context.Context, m *entities.OrgMembership) error
	Delete(ctx context.Context, orgID, userID uuid.UUID) error
}
