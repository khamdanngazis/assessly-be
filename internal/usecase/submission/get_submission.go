package submission

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// GetSubmissionRequest holds the data for getting a submission
type GetSubmissionRequest struct {
	SubmissionID uuid.UUID
	// Either access token (for participant) or user ID (for reviewer/creator)
	AccessToken string
	UserID      *uuid.UUID
	UserRole    string // "creator" or "reviewer"
}

// GetSubmissionResponse holds the submission with answers and reviews
type GetSubmissionResponse struct {
	Submission *domain.Submission
	Answers    []*AnswerWithReview
}

// AnswerWithReview combines an answer with its review (if any)
type AnswerWithReview struct {
	Answer *domain.Answer
	Review *domain.Review
}

// GetSubmissionUseCase handles retrieving submission details
type GetSubmissionUseCase struct {
	submissionRepo domain.SubmissionRepository
	answerRepo     domain.AnswerRepository
	reviewRepo     domain.ReviewRepository
	testRepo       domain.TestRepository
	validator      TokenValidator
	logger         *slog.Logger
}

// NewGetSubmissionUseCase creates a new GetSubmission use case
func NewGetSubmissionUseCase(
	submissionRepo domain.SubmissionRepository,
	answerRepo domain.AnswerRepository,
	reviewRepo domain.ReviewRepository,
	testRepo domain.TestRepository,
	validator TokenValidator,
	logger *slog.Logger,
) *GetSubmissionUseCase {
	return &GetSubmissionUseCase{
		submissionRepo: submissionRepo,
		answerRepo:     answerRepo,
		reviewRepo:     reviewRepo,
		testRepo:       testRepo,
		validator:      validator,
		logger:         logger,
	}
}

// Execute retrieves a submission with answers
func (uc *GetSubmissionUseCase) Execute(ctx context.Context, req GetSubmissionRequest) (*GetSubmissionResponse, error) {
	// Find submission
	submission, err := uc.submissionRepo.FindByID(ctx, req.SubmissionID)
	if err != nil {
		uc.logger.Warn("submission not found", "submission_id", req.SubmissionID)
		return nil, err
	}

	// Validate access
	authorized := false

	// Check if authenticated user (reviewer/creator)
	if req.UserID != nil {
		// Reviewers can see all submissions
		if req.UserRole == "reviewer" {
			authorized = true
			uc.logger.Info("reviewer accessing submission",
				"user_id", req.UserID,
				"submission_id", req.SubmissionID)
		} else if req.UserRole == "creator" {
			// Creators can see submissions for their tests
			test, err := uc.testRepo.FindByID(ctx, submission.TestID)
			if err == nil && test.CreatorID == *req.UserID {
				authorized = true
				uc.logger.Info("creator accessing own test submission",
					"user_id", req.UserID,
					"test_id", submission.TestID,
					"submission_id", req.SubmissionID)
			}
		}
	}

	// Check if participant with valid access token
	if !authorized && req.AccessToken != "" {
		testIDStr, email, role, err := uc.validator.ValidateToken(req.AccessToken)
		if err == nil && role == "participant" {
			testID, err := uuid.Parse(testIDStr)
			if err == nil && testID == submission.TestID && email == submission.AccessEmail {
				authorized = true
			}
		}
	}

	if !authorized {
		uc.logger.Warn("unauthorized access to submission", "submission_id", req.SubmissionID)
		return nil, domain.ErrUnauthorized{
			Message: "not authorized to view this submission",
		}
	}

	// Get answers
	answers, err := uc.answerRepo.FindBySubmissionID(ctx, req.SubmissionID)
	if err != nil {
		uc.logger.Error("failed to get answers", "error", err, "submission_id", req.SubmissionID)
		return nil, err
	}

	// Get reviews for each answer
	answersWithReviews := make([]*AnswerWithReview, len(answers))
	for i, answer := range answers {
		answersWithReviews[i] = &AnswerWithReview{
			Answer: answer,
			Review: nil,
		}

		// Try to find review for this answer (may not exist yet)
		review, err := uc.reviewRepo.FindByAnswerID(ctx, answer.ID)
		if err != nil {
			// If review doesn't exist, that's okay - it might not be scored yet
			if _, ok := err.(domain.ErrNotFound); !ok {
				uc.logger.Error("failed to get review", "error", err, "answer_id", answer.ID)
			}
			continue
		}
		answersWithReviews[i].Review = review
	}

	uc.logger.Info("submission retrieved", "submission_id", req.SubmissionID)

	return &GetSubmissionResponse{
		Submission: submission,
		Answers:    answersWithReviews,
	}, nil
}
