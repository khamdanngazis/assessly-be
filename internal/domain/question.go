package domain

import (
	"time"

	"github.com/google/uuid"
)

// Question represents an essay question within a test
type Question struct {
	ID             uuid.UUID `json:"id"`
	TestID         uuid.UUID `json:"test_id"`
	Text           string    `json:"text"`
	ExpectedAnswer string    `json:"expected_answer"`
	OrderNum       int       `json:"order_num"`
	CreatedAt      time.Time `json:"created_at"`
}

// Validate checks if question data is valid
func (q *Question) Validate() error {
	if q.Text == "" {
		return ErrValidation{Field: "text", Message: "question text is required"}
	}
	if q.ExpectedAnswer == "" {
		return ErrValidation{Field: "expected_answer", Message: "expected answer is required"}
	}
	if q.OrderNum <= 0 {
		return ErrValidation{Field: "order_num", Message: "order number must be greater than 0"}
	}
	if len(q.Text) > 10000 {
		return ErrValidation{Field: "text", Message: "question text must be less than 10000 characters"}
	}
	if len(q.ExpectedAnswer) > 10000 {
		return ErrValidation{Field: "expected_answer", Message: "expected answer must be less than 10000 characters"}
	}
	return nil
}
