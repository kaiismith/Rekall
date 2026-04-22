package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// ListCallsFilter holds optional filter criteria for listing calls.
type ListCallsFilter struct {
	UserID *uuid.UUID
	Status *string
}

// CallRepository defines the contract for call persistence operations.
// Infrastructure implementations must satisfy this interface.
type CallRepository interface {
	// Create persists a new call and returns the stored entity.
	Create(ctx context.Context, call *entities.Call) (*entities.Call, error)

	// GetByID retrieves a call by its unique identifier.
	// Returns an error wrapping errors.NotFound when no call exists with the given ID.
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Call, error)

	// List returns a paginated slice of non-deleted calls matching the filter.
	// Returns (calls, total, error).
	List(ctx context.Context, filter ListCallsFilter, page, perPage int) ([]*entities.Call, int, error)

	// Update applies changes to an existing call.
	Update(ctx context.Context, call *entities.Call) (*entities.Call, error)

	// SoftDelete marks a call as deleted without removing the row.
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
