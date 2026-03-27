package test

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// AddQuestionRequest holds the data for adding a question
type AddQuestionRequest struct {
	TestID         uuid.UUID
	Text           string
	ExpectedAnswer string
	OrderNum       int
}

// AddQuestionUseCase handles adding questions to tests
type AddQuestionUseCase struct {
	questionRepo domain.QuestionRepository
	testRepo     domain.TestRepository
	logger       *slog.Logger
}

// NewAddQuestionUseCase creates a new AddQuestion use case
func NewAddQuestionUseCase(
	questionRepo domain.QuestionRepository,
	testRepo domain.TestRepository,
	logger *slog.Logger,
) *AddQuestionUseCase {
	return &AddQuestionUseCase{
		questionRepo: questionRepo,
		testRepo:     testRepo,
		logger:       logger,
	}
}

// Execute adds a question to a test
func (uc *AddQuestionUseCase) Execute(ctx context.Context, req AddQuestionRequest) (*domain.Question, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Check if test exists
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return nil, err
	}

	// Don't allow adding questions to published tests
	if test.IsPublished {
		return nil, domain.ErrValidation{
			Field:   "test_id",
			Message: "cannot add questions to a published test",
		}
	}

	// If order_num is 0, auto-assign the next available number
	orderNum := req.OrderNum
	if orderNum == 0 {
		count, err := uc.questionRepo.CountByTestID(ctx, req.TestID)
		if err != nil {
			uc.logger.Error("failed to count questions", "error", err, "test_id", req.TestID)
			return nil, err
		}
		orderNum = count + 1
	}

	// Create question entity
	question := &domain.Question{
		ID:             uuid.New(),
		TestID:         req.TestID,
		Text:           req.Text,
		ExpectedAnswer: req.ExpectedAnswer,
		OrderNum:       orderNum,
		CreatedAt:      time.Now(),
	}

	// Validate question entity
	if err := question.Validate(); err != nil {
		return nil, err
	}

	// Save question to database
	if err := uc.questionRepo.Create(ctx, question); err != nil {
		uc.logger.Error("failed to create question", "error", err, "test_id", req.TestID)
		return nil, err
	}

	uc.logger.Info("question added successfully", "question_id", question.ID, "test_id", req.TestID, "order_num", orderNum)
	return question, nil
}

// validateRequest validates the add question request
func (uc *AddQuestionUseCase) validateRequest(req AddQuestionRequest) error {
	if req.Text == "" {
		return domain.ErrValidation{
			Field:   "text",
			Message: "question text is required",
		}
	}

	if req.ExpectedAnswer == "" {
		return domain.ErrValidation{
			Field:   "expected_answer",
			Message: "expected answer is required",
		}
	}

	if len(req.Text) > 10000 {
		return domain.ErrValidation{
			Field:   "text",
			Message: "question text must be less than 10000 characters",
		}
	}

	if len(req.ExpectedAnswer) > 10000 {
		return domain.ErrValidation{
			Field:   "expected_answer",
			Message: "expected answer must be less than 10000 characters",
		}
	}

	return nil
}
