package test

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// DeleteQuestionRequest holds the data for deleting a question
type DeleteQuestionRequest struct {
	QuestionID uuid.UUID
	TestID     uuid.UUID
	CreatorID  uuid.UUID // For authorization
}

// DeleteQuestionUseCase handles deleting questions
type DeleteQuestionUseCase struct {
	questionRepo domain.QuestionRepository
	testRepo     domain.TestRepository
	logger       *slog.Logger
}

// NewDeleteQuestionUseCase creates a new DeleteQuestion use case
func NewDeleteQuestionUseCase(
	questionRepo domain.QuestionRepository,
	testRepo domain.TestRepository,
	logger *slog.Logger,
) *DeleteQuestionUseCase {
	return &DeleteQuestionUseCase{
		questionRepo: questionRepo,
		testRepo:     testRepo,
		logger:       logger,
	}
}

// Execute deletes a question
func (uc *DeleteQuestionUseCase) Execute(ctx context.Context, req DeleteQuestionRequest) error {
	// Check if question exists
	question, err := uc.questionRepo.FindByID(ctx, req.QuestionID)
	if err != nil {
		uc.logger.Warn("question not found", "question_id", req.QuestionID)
		return err
	}

	// Check if test exists and belongs to creator
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return err
	}

	// Verify ownership
	if test.CreatorID != req.CreatorID {
		return domain.ErrUnauthorized{Message: "you can only delete questions in your own tests"}
	}

	// Don't allow deleting questions from published tests
	if test.IsPublished {
		return domain.ErrValidation{
			Field:   "test_id",
			Message: "cannot delete questions from a published test",
		}
	}

	// Verify question belongs to the test
	if question.TestID != req.TestID {
		return domain.ErrValidation{
			Field:   "question_id",
			Message: "question does not belong to this test",
		}
	}

	// Delete the question
	if err := uc.questionRepo.Delete(ctx, req.QuestionID); err != nil {
		uc.logger.Error("failed to delete question", "error", err, "question_id", req.QuestionID)
		return err
	}

	uc.logger.Info("question deleted successfully", "question_id", req.QuestionID, "test_id", req.TestID)
	return nil
}
