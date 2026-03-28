package review

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// GetReviewRequest holds the answer ID
type GetReviewRequest struct {
	AnswerID uuid.UUID
}

// GetReviewResponse contains the review with AI and manual scores
type GetReviewResponse struct {
	Review *domain.Review
}

// GetReviewUseCase handles retrieving a review for an answer
type GetReviewUseCase struct {
	reviewRepo domain.ReviewRepository
	logger     *slog.Logger
}

// NewGetReviewUseCase creates a new GetReview use case
func NewGetReviewUseCase(
	reviewRepo domain.ReviewRepository,
	logger *slog.Logger,
) *GetReviewUseCase {
	return &GetReviewUseCase{
		reviewRepo: reviewRepo,
		logger:     logger,
	}
}

// Execute retrieves the review (AI and manual scores) for an answer
func (uc *GetReviewUseCase) Execute(ctx context.Context, req GetReviewRequest) (*GetReviewResponse, error) {
	// Get the review
	review, err := uc.reviewRepo.FindByAnswerID(ctx, req.AnswerID)
	if err != nil {
		uc.logger.Error("failed to get review", "answer_id", req.AnswerID, "error", err)
		return nil, fmt.Errorf("review not found")
	}

	uc.logger.Info("review retrieved", "answer_id", req.AnswerID)

	return &GetReviewResponse{
		Review: review,
	}, nil
}
