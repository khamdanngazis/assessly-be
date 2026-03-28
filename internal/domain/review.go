package domain

import (
	"time"

	"github.com/google/uuid"
)

// Review represents AI and manual scoring for an answer
type Review struct {
	ID              uuid.UUID  `json:"id"`
	AnswerID        uuid.UUID  `json:"answer_id"`
	ReviewerID      *uuid.UUID `json:"reviewer_id"` // NULL when only AI scoring
	AIScore         *float64   `json:"ai_score"`
	AIFeedback      *string    `json:"ai_feedback"`
	AIScoredAt      *time.Time `json:"ai_scored_at"`
	ManualScore     *float64   `json:"manual_score"`
	ManualFeedback  *string    `json:"manual_feedback"`
	ManualScoredAt  *time.Time `json:"manual_scored_at"`
}

// Validate checks if review data is valid
func (r *Review) Validate() error {
	if r.AIScore != nil {
		if *r.AIScore < 0 || *r.AIScore > 100 {
			return ErrValidation{Field: "ai_score", Message: "AI score must be between 0 and 100"}
		}
	}
	if r.ManualScore != nil {
		if *r.ManualScore < 0 || *r.ManualScore > 100 {
			return ErrValidation{Field: "manual_score", Message: "manual score must be between 0 and 100"}
		}
	}
	return nil
}

// HasAIScore returns true if review has AI scoring
func (r *Review) HasAIScore() bool {
	return r.AIScore != nil
}

// HasManualScore returns true if review has manual scoring
func (r *Review) HasManualScore() bool {
	return r.ManualScore != nil
}

// GetFinalScore returns manual score if present, otherwise AI score
func (r *Review) GetFinalScore() *float64 {
	if r.ManualScore != nil {
		return r.ManualScore
	}
	return r.AIScore
}
