package handler

import "github.com/google/uuid"

// Shared response types for handlers

// SubmissionResponse represents a submission in API responses
type SubmissionResponse struct {
	ID               uuid.UUID `json:"id"`
	TestID           uuid.UUID `json:"test_id"`
	AccessEmail      string    `json:"access_email"`
	SubmittedAt      string    `json:"submitted_at"`
	AITotalScore     *float64  `json:"ai_total_score,omitempty"`
	ManualTotalScore *float64  `json:"manual_total_score,omitempty"`
	// T090: Display score prioritizes manual over AI
	DisplayScore     *float64  `json:"display_score,omitempty"`
}

// ReviewResponse represents review data in API responses
// T090: Includes display fields that prioritize manual over AI
type ReviewResponse struct {
	ID              string   `json:"id"`
	AnswerID        string   `json:"answer_id"`
	AIScore         *float64 `json:"ai_score,omitempty"`
	AIFeedback      *string  `json:"ai_feedback,omitempty"`
	AIScoredAt      *string  `json:"ai_scored_at,omitempty"`
	ManualScore     *float64 `json:"manual_score,omitempty"`
	ManualFeedback  *string  `json:"manual_feedback,omitempty"`
	ManualScoredAt  *string  `json:"manual_scored_at,omitempty"`
	ReviewerID      *string  `json:"reviewer_id,omitempty"`
	// T090: Display fields prioritize manual over AI
	DisplayScore    *float64 `json:"display_score,omitempty"`
	DisplayFeedback *string  `json:"display_feedback,omitempty"`
}
