package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	RoleCreator  UserRole = "creator"
	RoleReviewer UserRole = "reviewer"
)

// User represents an authenticated user (creator or reviewer)
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks if user data is valid
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrValidation{Field: "email", Message: "email is required"}
	}
	if u.Role != RoleCreator && u.Role != RoleReviewer {
		return ErrValidation{Field: "role", Message: "invalid role"}
	}
	return nil
}

// IsCreator returns true if user is a creator
func (u *User) IsCreator() bool {
	return u.Role == RoleCreator
}

// IsReviewer returns true if user is a reviewer
func (u *User) IsReviewer() bool {
	return u.Role == RoleReviewer
}
