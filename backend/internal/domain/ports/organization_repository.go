package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// OrganizationRepository abstracts persistence for Organization aggregates.
type OrganizationRepository interface {
	Create(ctx context.Context, org *entities.Organization) (*entities.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*entities.Organization, error)
	Update(ctx context.Context, org *entities.Organization) (*entities.Organization, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Organization, error)
}
