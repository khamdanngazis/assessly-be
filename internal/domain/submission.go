package domain

import (
	"time"

	"github.com/google/uuid"
)

// Submission represents a participant's test submission
type Submission struct {
	ID               uuid.UUID  `json:"id"`
	TestID           uuid.UUID  `json:"test_id"`
	AccessEmail      string     `json:"access_email"`
	SubmittedAt      time.Time  `json:"submitted_at"`
	AITotalScore     *float64   `json:"ai_total_score"`
	ManualTotalScore *float64   `json:"manual_total_score"`
}

// Validate checks if submission data is valid
func (s *Submission) Validate() error {
	if s.AccessEmail == "" {
		return ErrValidation{Field: "access_email", Message: "participant email is required"}
	}
	// Basic email validation (more thorough validation should be done at application layer)
	if len(s.AccessEmail) > 255 {
		return ErrValidation{Field: "access_email", Message: "email must be less than 255 characters"}
	}
	return nil
}
