package test

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// PublishTestRequest holds the data for publishing a test
type PublishTestRequest struct {
	TestID uuid.UUID
}

// PublishTestUseCase handles test publication
type PublishTestUseCase struct {
	testRepo     domain.TestRepository
	questionRepo domain.QuestionRepository
	logger       *slog.Logger
}

// NewPublishTestUseCase creates a new PublishTest use case
func NewPublishTestUseCase(
	testRepo domain.TestRepository,
	questionRepo domain.QuestionRepository,
	logger *slog.Logger,
) *PublishTestUseCase {
	return &PublishTestUseCase{
		testRepo:     testRepo,
		questionRepo: questionRepo,
		logger:       logger,
	}
}

// Execute publishes a test
func (uc *PublishTestUseCase) Execute(ctx context.Context, req PublishTestRequest) (*domain.Test, error) {
	// Find test
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return nil, err
	}

	// Check if already published
	if test.IsPublished {
		return nil, domain.ErrValidation{
			Field:   "test_id",
			Message: "test is already published",
		}
	}

	// Count questions
	questionCount, err := uc.questionRepo.CountByTestID(ctx, req.TestID)
	if err != nil {
		uc.logger.Error("failed to count questions", "error", err, "test_id", req.TestID)
		return nil, domain.ErrInternal{
			Message: "failed to count questions",
			Err:     err,
		}
	}

	// Validate test can be published
	if err := test.CanBePublished(questionCount); err != nil {
		return nil, err
	}

	// Update test to published
	test.IsPublished = true
	test.UpdatedAt = time.Now()

	if err := uc.testRepo.Update(ctx, test); err != nil {
		uc.logger.Error("failed to publish test", "error", err, "test_id", req.TestID)
		return nil, err
	}

	uc.logger.Info("test published successfully", "test_id", test.ID, "question_count", questionCount)
	return test, nil
}
