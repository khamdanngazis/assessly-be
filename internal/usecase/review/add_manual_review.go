package review

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// AddManualReviewRequest holds the manual review data
type AddManualReviewRequest struct {
	AnswerID       uuid.UUID
	ReviewerID     uuid.UUID
	ManualScore    float64
	ManualFeedback string
}

// AddManualReviewResponse contains the updated review
type AddManualReviewResponse struct {
	Review *domain.Review
}

// AddManualReviewUseCase handles adding or updating manual reviews
type AddManualReviewUseCase struct {
	reviewRepo     domain.ReviewRepository
	answerRepo     domain.AnswerRepository
	submissionRepo domain.SubmissionRepository
	logger         *slog.Logger
}

// NewAddManualReviewUseCase creates a new AddManualReview use case
func NewAddManualReviewUseCase(
	reviewRepo domain.ReviewRepository,
	answerRepo domain.AnswerRepository,
	submissionRepo domain.SubmissionRepository,
	logger *slog.Logger,
) *AddManualReviewUseCase {
	return &AddManualReviewUseCase{
		reviewRepo:     reviewRepo,
		answerRepo:     answerRepo,
		submissionRepo: submissionRepo,
		logger:         logger,
	}
}

// Execute adds or updates a manual review for an answer
func (uc *AddManualReviewUseCase) Execute(ctx context.Context, req AddManualReviewRequest) (*AddManualReviewResponse, error) {
	// Validate score range (0-100)
	if req.ManualScore < 0 || req.ManualScore > 100 {
		return nil, fmt.Errorf("manual score must be between 0 and 100")
	}

	// Verify answer exists
	answer, err := uc.answerRepo.FindByID(ctx, req.AnswerID)
	if err != nil {
		uc.logger.Error("failed to find answer", "answer_id", req.AnswerID, "error", err)
		return nil, fmt.Errorf("answer not found")
	}

	// Upsert manual score (creates review if not exists, updates if exists)
	if err := uc.reviewRepo.UpsertManualScore(ctx, req.AnswerID, req.ReviewerID, req.ManualScore, req.ManualFeedback); err != nil {
		uc.logger.Error("failed to upsert manual score",
			"answer_id", req.AnswerID,
			"reviewer_id", req.ReviewerID,
			"error", err,
		)
		return nil, err
	}

	// Get the updated review
	review, err := uc.reviewRepo.FindByAnswerID(ctx, req.AnswerID)
	if err != nil {
		uc.logger.Error("failed to get review", "answer_id", req.AnswerID, "error", err)
		return nil, err
	}

	// Update submission total score
	if err := uc.updateSubmissionTotal(ctx, answer.SubmissionID); err != nil {
		uc.logger.Warn("failed to update submission total", "submission_id", answer.SubmissionID, "error", err)
		// Continue anyway - the review was saved
	}

	uc.logger.Info("manual review added",
		"answer_id", req.AnswerID,
		"reviewer_id", req.ReviewerID,
		"manual_score", req.ManualScore,
	)

	return &AddManualReviewResponse{
		Review: review,
	}, nil
}

// updateSubmissionTotal recalculates the manual total score for a submission
func (uc *AddManualReviewUseCase) updateSubmissionTotal(ctx context.Context, submissionID uuid.UUID) error {
	// Get the submission
	submission, err := uc.submissionRepo.FindByID(ctx, submissionID)
	if err != nil {
		return err
	}

	// Get all answers for the submission
	answers, err := uc.answerRepo.FindBySubmissionID(ctx, submissionID)
	if err != nil {
		return err
	}

	// Calculate total manual score
	var manualTotal *float64
	var hasManualScores bool

	for _, answer := range answers {
		review, err := uc.reviewRepo.FindByAnswerID(ctx, answer.ID)
		if err != nil {
			continue // Skip if no review
		}

		if review.ManualScore != nil {
			hasManualScores = true
			if manualTotal == nil {
				zero := 0.0
				manualTotal = &zero
			}
			*manualTotal += *review.ManualScore
		}
	}

	// Update submission with manual total
	if hasManualScores {
		submission.ManualTotalScore = manualTotal
		return uc.submissionRepo.Update(ctx, submission)
	}

	return nil
}
