package entities

import (
	"time"

	"github.com/google/uuid"
)

// User represents a platform member who owns and manages calls.
type User struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email         string     `gorm:"uniqueIndex;not null"                           json:"email"`
	FullName      string     `gorm:"column:full_name;not null"                      json:"full_name"`
	Role          string     `gorm:"not null;default:member"                        json:"role"` // "admin" | "member"
	PasswordHash  string     `gorm:"column:password_hash"                           json:"-"`
	EmailVerified bool       `gorm:"column:email_verified;not null;default:false"   json:"email_verified"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"                                 json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index"                                          json:"deleted_at,omitempty"`
}

// TableName tells GORM which table to use for this model.
func (User) TableName() string { return "users" }

// IsAdmin reports whether the user holds the admin role.
func (u *User) IsAdmin() bool { return u.Role == "admin" }

// IsDeleted reports whether the user has been soft-deleted.
func (u *User) IsDeleted() bool { return u.DeletedAt != nil }

// IsEmailVerified reports whether the user's email address has been confirmed.
func (u *User) IsEmailVerified() bool { return u.EmailVerified }
