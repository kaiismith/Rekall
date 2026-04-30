package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// UserRepository defines the contract for user persistence operations.
// Infrastructure implementations must satisfy this interface.
type UserRepository interface {
	// Create persists a new user and returns the stored entity.
	Create(ctx context.Context, user *entities.User) (*entities.User, error)

	// GetByID retrieves a user by their unique identifier.
	// Returns an error wrapping errors.NotFound when no user exists with the given ID.
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)

	// FindByIDs returns the (non-deleted) users matching the given ids in a
	// single query. Missing ids are silently omitted from the result; callers
	// that need a strict ids → users mapping should build it themselves.
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.User, error)

	// GetByEmail retrieves a user by their email address.
	GetByEmail(ctx context.Context, email string) (*entities.User, error)

	// List returns a paginated slice of non-deleted users.
	List(ctx context.Context, page, perPage int) ([]*entities.User, int, error)

	// Update applies changes to an existing user.
	Update(ctx context.Context, user *entities.User) (*entities.User, error)

	// SoftDelete marks a user as deleted without removing the row.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// SetEmailVerified updates the email_verified flag for the given user.
	SetEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error

	// UpdatePassword replaces the stored password hash for the given user.
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error

	// SetRoleByEmail sets the platform-level role on the user with the given
	// email. No-op if the email is unknown. Used by the AdminReconciler.
	SetRoleByEmail(ctx context.Context, email, role string) error

	// DemoteAdminsExcept downgrades every user currently with role="admin"
	// whose email is NOT in the keepEmails set back to role="member". The
	// AdminReconciler calls this so the env var stays authoritative across
	// boots.
	DemoteAdminsExcept(ctx context.Context, keepEmails []string) (int, error)
}
