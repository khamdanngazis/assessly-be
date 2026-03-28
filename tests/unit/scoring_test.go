package unit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/scoring"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// QueueAIScoringUseCase Tests
// ============================================================================

func TestQueueAIScoring_Success(t *testing.T) {
	// Arrange
	mockQueue := new(MockQueueEnqueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewQueueAIScoringUseCase(mockQueue, logger)

	submissionID := uuid.New()
	req := scoring.QueueScoringRequest{
		SubmissionID: submissionID,
	}

	mockQueue.On("Enqueue", mock.Anything, submissionID).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	mockQueue.AssertExpectations(t)
}

func TestQueueAIScoring_EnqueueError(t *testing.T) {
	// Arrange
	mockQueue := new(MockQueueEnqueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewQueueAIScoringUseCase(mockQueue, logger)

	submissionID := uuid.New()
	req := scoring.QueueScoringRequest{
		SubmissionID: submissionID,
	}

	mockQueue.On("Enqueue", mock.Anything, submissionID).
		Return(errors.New("redis connection failed"))

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis connection failed")
	mockQueue.AssertExpectations(t)
}

// ============================================================================
// ScoreWithAIUseCase Tests
// ============================================================================

func TestScoreWithAI_Success(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockAIScorer := new(MockAIScorer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewScoreWithAIUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockQuestionRepo,
		mockReviewRepo,
		mockAIScorer,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	q1ID := uuid.New()
	q2ID := uuid.New()
	a1ID := uuid.New()
	a2ID := uuid.New()

	submission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	answers := []*domain.Answer{
		{
			ID:           a1ID,
			SubmissionID: submissionID,
			QuestionID:   q1ID,
			Text:         "Go is a programming language",
			CreatedAt:    time.Now(),
		},
		{
			ID:           a2ID,
			SubmissionID: submissionID,
			QuestionID:   q2ID,
			Text:         "Goroutines are lightweight threads",
			CreatedAt:    time.Now(),
		},
	}

	questions := []*domain.Question{
		{ID: q1ID, TestID: testID, Text: "What is Go?", ExpectedAnswer: "A programming language", OrderNum: 1},
		{ID: q2ID, TestID: testID, Text: "What are goroutines?", ExpectedAnswer: "Lightweight threads", OrderNum: 2},
	}

	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(submission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockQuestionRepo.On("FindByID", mock.Anything, q1ID).Return(questions[0], nil)
	mockQuestionRepo.On("FindByID", mock.Anything, q2ID).Return(questions[1], nil)
	mockAIScorer.On("ScoreAnswer", mock.Anything, "What is Go?", "A programming language", "Go is a programming language").
		Return(&scoring.ScoreResult{Score: 85.0, Feedback: "Good answer"}, nil)
	mockAIScorer.On("ScoreAnswer", mock.Anything, "What are goroutines?", "Lightweight threads", "Goroutines are lightweight threads").
		Return(&scoring.ScoreResult{Score: 90.0, Feedback: "Excellent answer"}, nil)
	mockReviewRepo.On("UpsertAIScore", mock.Anything, a1ID, 85.0, "Good answer").Return(nil)
	mockReviewRepo.On("UpsertAIScore", mock.Anything, a2ID, 90.0, "Excellent answer").Return(nil)
	mockSubmissionRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.ID == submissionID && s.AITotalScore != nil && *s.AITotalScore == 175.0 // Sum: 85 + 90
	})).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
	mockAIScorer.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
}

func TestScoreWithAI_SubmissionNotFound(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockAIScorer := new(MockAIScorer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewScoreWithAIUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockQuestionRepo,
		mockReviewRepo,
		mockAIScorer,
		logger,
	)

	submissionID := uuid.New()
	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).
		Return(nil, domain.ErrNotFound{Resource: "submission", ID: submissionID.String()})

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertNotCalled(t, "FindBySubmissionID")
}

func TestScoreWithAI_NoAnswers(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockAIScorer := new(MockAIScorer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewScoreWithAIUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockQuestionRepo,
		mockReviewRepo,
		mockAIScorer,
		logger,
	)

	submissionID := uuid.New()
	submission := &domain.Submission{
		ID:          submissionID,
		TestID:      uuid.New(),
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(submission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return([]*domain.Answer{}, nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no answers found")
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "FindByID")
	mockAIScorer.AssertNotCalled(t, "ScoreAnswer")
}

func TestScoreWithAI_AIServiceError(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockAIScorer := new(MockAIScorer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewScoreWithAIUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockQuestionRepo,
		mockReviewRepo,
		mockAIScorer,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	q1ID := uuid.New()
	a1ID := uuid.New()

	submission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	answers := []*domain.Answer{
		{
			ID:           a1ID,
			SubmissionID: submissionID,
			QuestionID:   q1ID,
			Text:         "Answer",
			CreatedAt:    time.Now(),
		},
	}

	question := &domain.Question{
		ID:             q1ID,
		TestID:         testID,
		Text:           "Question",
		ExpectedAnswer: "Expected",
		OrderNum:       1,
	}

	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(submission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockQuestionRepo.On("FindByID", mock.Anything, q1ID).Return(question, nil)
	mockAIScorer.On("ScoreAnswer", mock.Anything, "Question", "Expected", "Answer").
		Return(nil, errors.New("AI service unavailable"))

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	// The use case continues even with AI errors (logs and skips), so no error returned
	// But the submission is not updated since no scores were recorded
	assert.NoError(t, err)
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
	mockAIScorer.AssertExpectations(t)
	mockReviewRepo.AssertNotCalled(t, "UpsertAIScore")
	mockSubmissionRepo.AssertNotCalled(t, "Update")
}

func TestScoreWithAI_PartialSuccess(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockAIScorer := new(MockAIScorer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := scoring.NewScoreWithAIUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockQuestionRepo,
		mockReviewRepo,
		mockAIScorer,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	q1ID := uuid.New()
	q2ID := uuid.New()
	a1ID := uuid.New()
	a2ID := uuid.New()

	submission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	answers := []*domain.Answer{
		{ID: a1ID, SubmissionID: submissionID, QuestionID: q1ID, Text: "Answer 1", CreatedAt: time.Now()},
		{ID: a2ID, SubmissionID: submissionID, QuestionID: q2ID, Text: "Answer 2", CreatedAt: time.Now()},
	}

	questions := []*domain.Question{
		{ID: q1ID, TestID: testID, Text: "Q1", ExpectedAnswer: "E1", OrderNum: 1},
		{ID: q2ID, TestID: testID, Text: "Q2", ExpectedAnswer: "E2", OrderNum: 2},
	}

	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(submission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockQuestionRepo.On("FindByID", mock.Anything, q1ID).Return(questions[0], nil)
	mockQuestionRepo.On("FindByID", mock.Anything, q2ID).Return(questions[1], nil)
	mockAIScorer.On("ScoreAnswer", mock.Anything, "Q1", "E1", "Answer 1").
		Return(&scoring.ScoreResult{Score: 80.0, Feedback: "Good"}, nil)
	mockAIScorer.On("ScoreAnswer", mock.Anything, "Q2", "E2", "Answer 2").
		Return(nil, errors.New("AI error")) // Second one fails
	mockReviewRepo.On("UpsertAIScore", mock.Anything, a1ID, 80.0, "Good").Return(nil)
	mockSubmissionRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.ID == submissionID && s.AITotalScore != nil && *s.AITotalScore == 80.0 // Only first answer scored
	})).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err) // Use case continues despite partial failures
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
	mockAIScorer.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
}
