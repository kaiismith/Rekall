package repohelpers

import (
	"github.com/rekall/backend/internal/domain/ports"
	"gorm.io/gorm"
)

// ApplyCallFilter chains WHERE conditions onto the query for each populated filter field.
func ApplyCallFilter(db *gorm.DB, f ports.ListCallsFilter) *gorm.DB {
	if f.UserID != nil {
		db = db.Where("user_id = ?", *f.UserID)
	}
	if f.Status != nil {
		db = db.Where("status = ?", *f.Status)
	}
	if f.Scope != nil {
		switch f.Scope.Kind {
		case ports.ScopeKindOpen:
			db = db.Where("scope_type IS NULL")
		case ports.ScopeKindOrganization:
			db = db.Where("scope_type = ? AND scope_id = ?", "organization", f.Scope.ID)
		case ports.ScopeKindDepartment:
			db = db.Where("scope_type = ? AND scope_id = ?", "department", f.Scope.ID)
		}
	}
	return db
}
