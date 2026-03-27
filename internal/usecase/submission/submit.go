package submission

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// TokenValidator interface for validating access tokens
type TokenValidator interface {
	ValidateToken(tokenString string) (testID string, email string, role string, err error)
}

// ScoringQueuer interface for queuing AI scoring
type ScoringQueuer interface {
	Enqueue(ctx context.Context, submissionID uuid.UUID) error
}

// AnswerInput represents an answer to a question
type AnswerInput struct {
	QuestionID uuid.UUID
	Text       string
}

// SubmitTestRequest holds the data for submitting a test
type SubmitTestRequest struct {
	AccessToken string
	Answers     []AnswerInput
}

// SubmitTestUseCase handles test submission by participants
type SubmitTestUseCase struct {
	testRepo       domain.TestRepository
	questionRepo   domain.QuestionRepository
	submissionRepo domain.SubmissionRepository
	answerRepo     domain.AnswerRepository
	validator      TokenValidator
	scoringQueue   ScoringQueuer
	logger         *slog.Logger
}

// NewSubmitTestUseCase creates a new SubmitTest use case
func NewSubmitTestUseCase(
	testRepo domain.TestRepository,
	questionRepo domain.QuestionRepository,
	submissionRepo domain.SubmissionRepository,
	answerRepo domain.AnswerRepository,
	validator TokenValidator,
	scoringQueue ScoringQueuer,
	logger *slog.Logger,
) *SubmitTestUseCase {
	return &SubmitTestUseCase{
		testRepo:       testRepo,
		questionRepo:   questionRepo,
		submissionRepo: submissionRepo,
		answerRepo:     answerRepo,
		validator:      validator,
		scoringQueue:   scoringQueue,
		logger:         logger,
	}
}

// Execute submits a test with answers
func (uc *SubmitTestUseCase) Execute(ctx context.Context, req SubmitTestRequest) (*domain.Submission, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Validate access token
	testIDStr, email, role, err := uc.validator.ValidateToken(req.AccessToken)
	if err != nil {
		uc.logger.Warn("invalid access token", "error", err)
		return nil, domain.ErrUnauthorized{
			Message: "invalid or expired access token",
		}
	}

	// Check if role is "participant"
	if role != "participant" {
		uc.logger.Warn("token is not a participant token", "role", role)
		return nil, domain.ErrUnauthorized{
			Message: "invalid access token",
		}
	}

	// Parse test ID
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		uc.logger.Error("invalid test ID in token", "error", err, "test_id", testIDStr)
		return nil, domain.ErrUnauthorized{
			Message: "invalid access token",
		}
	}

	// Find test
	test, err := uc.testRepo.FindByID(ctx, testID)
	if err != nil {
		uc.logger.Error("test not found", "error", err, "test_id", testID)
		return nil, err
	}

	// Check if test is published
	if !test.IsPublished {
		return nil, domain.ErrValidation{
			Field:   "test_id",
			Message: "test is not published",
		}
	}

	// Check retake policy
	if !test.AllowRetakes {
		count, err := uc.submissionRepo.CountByTestAndEmail(ctx, testID, email)
		if err != nil {
			uc.logger.Error("failed to count submissions", "error", err, "test_id", testID)
			return nil, err
		}
		if count > 0 {
			return nil, domain.ErrValidation{
				Field:   "test_id",
				Message: "retakes not allowed for this test",
			}
		}
	}

	// Get all questions for the test
	questions, err := uc.questionRepo.FindByTestID(ctx, testID)
	if err != nil {
		uc.logger.Error("failed to get questions", "error", err, "test_id", testID)
		return nil, err
	}

	// Validate that all questions are answered
	questionMap := make(map[uuid.UUID]bool)
	for _, q := range questions {
		questionMap[q.ID] = false
	}
	for _, ans := range req.Answers {
		if _, exists := questionMap[ans.QuestionID]; !exists {
			return nil, domain.ErrValidation{
				Field:   "answers",
				Message: "invalid question ID in answers",
			}
		}
		questionMap[ans.QuestionID] = true
	}
	for qID, answered := range questionMap {
		if !answered {
			return nil, domain.ErrValidation{
				Field:   "answers",
				Message: "all questions must be answered: missing question " + qID.String(),
			}
		}
	}

	// Create submission
	now := time.Now()
	submission := &domain.Submission{
		ID:               uuid.New(),
		TestID:           testID,
		AccessEmail:      email,
		SubmittedAt:      now,
		AITotalScore:     nil, // Will be set by AI scoring worker
		ManualTotalScore: nil, // Will be set by reviewer if needed
	}

	if err := uc.submissionRepo.Create(ctx, submission); err != nil {
		uc.logger.Error("failed to create submission", "error", err, "test_id", testID)
		return nil, err
	}

	// Create answers
	answers := make([]*domain.Answer, len(req.Answers))
	for i, ansInput := range req.Answers {
		answer := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   ansInput.QuestionID,
			Text:         ansInput.Text,
			CreatedAt:    now,
		}
		if err := answer.Validate(); err != nil {
			return nil, err
		}
		answers[i] = answer
	}

	if err := uc.answerRepo.CreateBatch(ctx, answers); err != nil {
		uc.logger.Error("failed to create answers", "error", err, "submission_id", submission.ID)
		return nil, err
	}

	// Queue for AI scoring asynchronously (fire and forget)
	if err := uc.scoringQueue.Enqueue(ctx, submission.ID); err != nil {
		// Log error but don't fail the submission
		uc.logger.Error("failed to queue AI scoring", "error", err, "submission_id", submission.ID)
	}

	uc.logger.Info("test submitted successfully", "submission_id", submission.ID, "test_id", testID, "email", email)
	return submission, nil
}

// validateRequest validates the submit test request
func (uc *SubmitTestUseCase) validateRequest(req SubmitTestRequest) error {
	if req.AccessToken == "" {
		return domain.ErrValidation{
			Field:   "access_token",
			Message: "access token is required",
		}
	}

	if len(req.Answers) == 0 {
		return domain.ErrValidation{
			Field:   "answers",
			Message: "at least one answer is required",
		}
	}

	return nil
}
