package groq

import (
	"context"

	"github.com/assessly/assessly-be/internal/usecase/scoring"
)

// ScorerAdapter adapts GroqClient to scoring use case interface
type ScorerAdapter struct {
	client *Client
}

// NewScorerAdapter creates a new adapter
func NewScorerAdapter(client *Client) *ScorerAdapter {
	return &ScorerAdapter{client: client}
}

// ScoreAnswer scores an answer using Groq AI
func (a *ScorerAdapter) ScoreAnswer(ctx context.Context, question, expectedAnswer, actualAnswer string) (*scoring.ScoreResult, error) {
	score, feedback, err := a.client.ScoreAnswer(ctx, question, expectedAnswer, actualAnswer)
	if err != nil {
		return nil, err
	}

	return &scoring.ScoreResult{
		Score:    score,
		Feedback: feedback,
	}, nil
}
