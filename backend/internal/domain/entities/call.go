package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JSONMap is a map[string]interface{} that reads/writes PostgreSQL JSONB columns.
// It implements driver.Valuer and sql.Scanner so it works with GORM and database/sql
// without importing any ORM package into the domain layer.
type JSONMap map[string]interface{}

// Value serialises the map to JSON for storage.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("JSONMap.Value: %w", err)
	}
	return string(b), nil
}

// Scan deserialises a JSON value from the database into the map.
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("JSONMap.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(raw, j)
}

// Call is the central aggregate root representing a recorded conversation.
type Call struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"   json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;column:user_id;not null;index"          json:"user_id"`
	Title        string     `gorm:"not null"                                         json:"title"`
	DurationSec  int        `gorm:"column:duration_sec;not null;default:0"           json:"duration_sec"`
	Status       string     `gorm:"not null;default:pending"                         json:"status"` // pending | processing | done | failed
	RecordingURL *string    `gorm:"column:recording_url"                             json:"recording_url,omitempty"`
	Transcript   *string    `gorm:"type:text"                                        json:"transcript,omitempty"`
	Metadata     JSONMap    `gorm:"type:jsonb;not null;default:'{}'"                 json:"metadata"`
	// Scope attaches this call to an organization or department. NULL on both
	// means the call is an Open Item — not attached to any team.
	ScopeType    *string    `gorm:"column:scope_type"                                json:"scope_type,omitempty"`
	ScopeID      *uuid.UUID `gorm:"type:uuid;column:scope_id"                        json:"scope_id,omitempty"`
	StartedAt    *time.Time `gorm:"column:started_at"                                json:"started_at,omitempty"`
	EndedAt      *time.Time `gorm:"column:ended_at"                                  json:"ended_at,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"                                   json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"                                   json:"updated_at"`
	// GORM v2 recognises *time.Time named DeletedAt as a soft-delete field automatically.
	DeletedAt    *time.Time `gorm:"index"                                            json:"deleted_at,omitempty"`
}

// TableName tells GORM which table to use for this model.
func (Call) TableName() string { return "calls" }

// IsPending reports whether the call is still awaiting processing.
func (c *Call) IsPending() bool { return c.Status == "pending" }

// IsProcessing reports whether the call is currently being processed.
func (c *Call) IsProcessing() bool { return c.Status == "processing" }

// IsDone reports whether the call has been successfully processed.
func (c *Call) IsDone() bool { return c.Status == "done" }

// IsFailed reports whether the call processing failed.
func (c *Call) IsFailed() bool { return c.Status == "failed" }

// IsDeleted reports whether the call has been soft-deleted.
func (c *Call) IsDeleted() bool { return c.DeletedAt != nil }
