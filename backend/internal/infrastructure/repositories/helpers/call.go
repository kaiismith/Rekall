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
	return db
}
