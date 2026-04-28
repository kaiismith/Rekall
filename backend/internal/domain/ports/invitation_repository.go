package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// InvitationRepository abstracts persistence for organization invitations.
type InvitationRepository interface {
	// Upsert creates a new invitation or refreshes the token and expiry of an existing
	// pending invitation for the same (org_id, email) pair.
	Upsert(ctx context.Context, inv *entities.Invitation) error
	GetByTokenHash(ctx context.Context, hash string) (*entities.Invitation, error)
	GetPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*entities.Invitation, error)
	MarkAccepted(ctx context.Context, hash string) error
}
