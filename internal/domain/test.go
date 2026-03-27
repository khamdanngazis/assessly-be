package domain

import (
	"time"

	"github.com/google/uuid"
)

// Test represents a test definition created by a user
type Test struct {
	ID           uuid.UUID `json:"id"`
	CreatorID    uuid.UUID `json:"creator_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	AllowRetakes bool      `json:"allow_retakes"`
	IsPublished  bool      `json:"is_published"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks if test data is valid
func (t *Test) Validate() error {
	if t.Title == "" {
		return ErrValidation{Field: "title", Message: "title is required"}
	}
	if len(t.Title) > 255 {
		return ErrValidation{Field: "title", Message: "title must be less than 255 characters"}
	}
	return nil
}

// CanBePublished checks if test can be published
func (t *Test) CanBePublished(questionCount int) error {
	if questionCount == 0 {
		return ErrValidation{Field: "questions", Message: "test must have at least one question to be published"}
	}
	return nil
}
