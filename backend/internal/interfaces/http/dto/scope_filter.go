package dto

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ParseScopeFilter reads the optional filter[scope_type] / filter[scope_id]
// query parameters and returns a *ports.ScopeFilter.
//
// Returns (nil, nil) when the caller did not supply any scope parameters —
// the handler should preserve the endpoint's default visibility rules.
//
// Returns (nil, apperr.BadRequest) when the parameters are malformed:
//   - scope_type is set to a value other than organization / department / open
//   - scope_type is organization or department but scope_id is missing or not a UUID
//   - scope_id is set without scope_type
func ParseScopeFilter(c *gin.Context) (*ports.ScopeFilter, error) {
	scopeType := c.Query("filter[scope_type]")
	scopeIDStr := c.Query("filter[scope_id]")

	if scopeType == "" && scopeIDStr == "" {
		return nil, nil
	}
	if scopeType == "" {
		return nil, apperr.BadRequest("filter[scope_id] set without filter[scope_type]")
	}

	kind := ports.ScopeKind(scopeType)
	if !kind.IsValid() {
		return nil, apperr.BadRequest("filter[scope_type] must be one of: organization, department, open")
	}

	if kind == ports.ScopeKindOpen {
		if scopeIDStr != "" {
			return nil, apperr.BadRequest("filter[scope_id] must be omitted when filter[scope_type]=open")
		}
		return &ports.ScopeFilter{Kind: ports.ScopeKindOpen}, nil
	}

	// organization or department — scope_id is required and must parse as a UUID.
	if scopeIDStr == "" {
		return nil, apperr.BadRequest("filter[scope_id] is required when filter[scope_type] is organization or department")
	}
	id, err := uuid.Parse(scopeIDStr)
	if err != nil {
		return nil, apperr.BadRequest("filter[scope_id] must be a UUID")
	}
	return &ports.ScopeFilter{Kind: kind, ID: id}, nil
}
