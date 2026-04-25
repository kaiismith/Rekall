package ports

import "github.com/google/uuid"

// ScopeFilter identifies which slice of items a list query is restricted to.
//
// Three shapes are valid:
//
//	{Kind: ScopeKindOrganization, ID: <orgID>} — rows where scope_type='organization' AND scope_id=<orgID>
//	{Kind: ScopeKindDepartment,   ID: <deptID>} — rows where scope_type='department'   AND scope_id=<deptID>
//	{Kind: ScopeKindOpen}                       — rows where scope_type IS NULL
//
// A nil *ScopeFilter means "no scope filter applied" — preserve the caller's
// existing visibility rules (host + participant for meetings, owner for calls).
type ScopeFilter struct {
	Kind ScopeKind
	ID   uuid.UUID // zero value for ScopeKindOpen
}

// ScopeKind enumerates the allowed scope filter values.
type ScopeKind string

const (
	ScopeKindOrganization ScopeKind = "organization"
	ScopeKindDepartment   ScopeKind = "department"
	ScopeKindOpen         ScopeKind = "open"
)

// IsValid reports whether k is one of the three accepted scope kinds.
func (k ScopeKind) IsValid() bool {
	return k == ScopeKindOrganization || k == ScopeKindDepartment || k == ScopeKindOpen
}
