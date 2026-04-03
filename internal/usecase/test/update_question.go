package test

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// UpdateQuestionRequest holds the data for updating a question
type UpdateQuestionRequest struct {
	QuestionID     uuid.UUID
	TestID         uuid.UUID
	CreatorID      uuid.UUID // For authorization
	Text           string
	ExpectedAnswer string
	OrderNum       int
}

// UpdateQuestionUseCase handles updating questions
type UpdateQuestionUseCase struct {
	questionRepo domain.QuestionRepository
	testRepo     domain.TestRepository
	logger       *slog.Logger
}

// NewUpdateQuestionUseCase creates a new UpdateQuestion use case
func NewUpdateQuestionUseCase(
	questionRepo domain.QuestionRepository,
	testRepo domain.TestRepository,
	logger *slog.Logger,
) *UpdateQuestionUseCase {
	return &UpdateQuestionUseCase{
		questionRepo: questionRepo,
		testRepo:     testRepo,
		logger:       logger,
	}
}

// Execute updates a question
func (uc *UpdateQuestionUseCase) Execute(ctx context.Context, req UpdateQuestionRequest) (*domain.Question, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Check if question exists
	question, err := uc.questionRepo.FindByID(ctx, req.QuestionID)
	if err != nil {
		uc.logger.Warn("question not found", "question_id", req.QuestionID)
		return nil, err
	}

	// Check if test exists and belongs to creator
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return nil, err
	}

	// Verify ownership
	if test.CreatorID != req.CreatorID {
		return nil, domain.ErrUnauthorized{Message: "you can only update questions in your own tests"}
	}

	// Don't allow updating questions in published tests
	if test.IsPublished {
		return nil, domain.ErrValidation{
			Field:   "test_id",
			Message: "cannot update questions in a published test",
		}
	}

	// Verify question belongs to the test
	if question.TestID != req.TestID {
		return nil, domain.ErrValidation{
			Field:   "question_id",
			Message: "question does not belong to this test",
		}
	}

	// Update question fields
	question.Text = req.Text
	question.ExpectedAnswer = req.ExpectedAnswer
	if req.OrderNum > 0 {
		question.OrderNum = req.OrderNum
	}

	// Validate updated question
	if err := question.Validate(); err != nil {
		return nil, err
	}

	// Save updates to database
	if err := uc.questionRepo.Update(ctx, question); err != nil {
		uc.logger.Error("failed to update question", "error", err, "question_id", req.QuestionID)
		return nil, err
	}

	uc.logger.Info("question updated successfully", "question_id", question.ID, "test_id", req.TestID)
	return question, nil
}

// validateRequest validates the update question request
func (uc *UpdateQuestionUseCase) validateRequest(req UpdateQuestionRequest) error {
	if req.Text == "" {
		return domain.ErrValidation{Field: "text", Message: "question text is required"}
	}
	if req.ExpectedAnswer == "" {
		return domain.ErrValidation{Field: "expected_answer", Message: "expected answer is required"}
	}
	if len(req.Text) > 10000 {
		return domain.ErrValidation{Field: "text", Message: "question text must be less than 10000 characters"}
	}
	if len(req.ExpectedAnswer) > 10000 {
		return domain.ErrValidation{Field: "expected_answer", Message: "expected answer must be less than 10000 characters"}
	}
	return nil
}
