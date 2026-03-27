package domain

import (
	"time"

	"github.com/google/uuid"
)

// Answer represents a participant's response to a question
type Answer struct {
	ID           uuid.UUID `json:"id"`
	SubmissionID uuid.UUID `json:"submission_id"`
	QuestionID   uuid.UUID `json:"question_id"`
	Text         string    `json:"text"`
	CreatedAt    time.Time `json:"created_at"`
}

// Validate checks if answer data is valid
func (a *Answer) Validate() error {
	if a.Text == "" {
		return ErrValidation{Field: "text", Message: "answer text is required"}
	}
	if len(a.Text) > 50000 {
		return ErrValidation{Field: "text", Message: "answer text must be less than 50000 characters"}
	}
	return nil
}
