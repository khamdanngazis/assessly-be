package unit

import (
	"context"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/review"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// AddManualReviewUseCase Tests
// ============================================================================

func TestAddManualReview_Success(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)

	answerID := uuid.New()
	reviewerID := uuid.New()
	submissionID := uuid.New()
	questionID := uuid.New()
	
	req := review.AddManualReviewRequest{
		AnswerID:       answerID,
		ReviewerID:     reviewerID,
		ManualScore:    85.5,
		ManualFeedback: "Good answer with minor improvements needed",
	}

	answer := &domain.Answer{
		ID:           answerID,
		SubmissionID: submissionID,
		QuestionID:   questionID,
		Text:         "Student answer",
		CreatedAt:    time.Now(),
	}

	submission := &domain.Submission{
		ID:          submissionID,
		TestID:      uuid.New(),
		AccessEmail: "student@example.com",
		SubmittedAt: time.Now(),
	}

	existingReview := &domain.Review{
		ID:       uuid.New(),
		AnswerID: answerID,
		AIScore:  new(float64),
	}
	*existingReview.AIScore = 80.0

	updatedReview := &domain.Review{
		ID:             existingReview.ID,
		AnswerID:       answerID,
		AIScore:        existingReview.AIScore,
		ManualScore:    &req.ManualScore,
		ManualFeedback: &req.ManualFeedback,
	}

	// Mock expectations
	mockAnswerRepo.On("FindByID", mock.Anything, answerID).Return(answer, nil)
	mockReviewRepo.On("UpsertManualScore", mock.Anything, answerID, reviewerID, req.ManualScore, req.ManualFeedback).Return(nil)
	
	// First call to FindByAnswerID in Execute (line 73)
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).Return(updatedReview, nil).Once()
	
	// Mocks for updateSubmissionTotal
	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(submission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return([]*domain.Answer{answer}, nil)
	
	// Second call to FindByAnswerID inside updateSubmissionTotal loop (line 115)
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).Return(updatedReview, nil).Once()
	
	// Update is called with the submission object after calculating manual total
	mockSubmissionRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.ID == submissionID && s.ManualTotalScore != nil && *s.ManualTotalScore == req.ManualScore
	})).Return(nil)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Review)
	assert.Equal(t, answerID, response.Review.AnswerID)
	assert.Equal(t, req.ManualScore, *response.Review.ManualScore)
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
}

func TestAddManualReview_ScoreTooLow(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)

	req := review.AddManualReviewRequest{
		AnswerID:       uuid.New(),
		ReviewerID:     uuid.New(),
		ManualScore:    -5.0,  // Invalid: below 0
		ManualFeedback: "Too low",
	}

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "score must be between 0 and 100")
	mockAnswerRepo.AssertNotCalled(t, "FindByID")
	mockReviewRepo.AssertNotCalled(t, "UpsertManualScore")
}

func TestAddManualReview_ScoreTooHigh(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)

	req := review.AddManualReviewRequest{
		AnswerID:       uuid.New(),
		ReviewerID:     uuid.New(),
		ManualScore:    150.0,  // Invalid: above 100
		ManualFeedback: "Too high",
	}

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "score must be between 0 and 100")
	mockAnswerRepo.AssertNotCalled(t, "FindByID")
	mockReviewRepo.AssertNotCalled(t, "UpsertManualScore")
}

func TestAddManualReview_AnswerNotFound(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)

	answerID := uuid.New()
	req := review.AddManualReviewRequest{
		AnswerID:       answerID,
		ReviewerID:     uuid.New(),
		ManualScore:    75.0,
		ManualFeedback: "Good work",
	}

	// Mock expectations
	mockAnswerRepo.On("FindByID", mock.Anything, answerID).Return(nil, domain.ErrNotFound{Resource: "answer", ID: answerID.String()})

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "answer not found")
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertNotCalled(t, "UpsertManualScore")
}

func TestAddManualReview_UpsertError(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)

	answerID := uuid.New()
	submissionID := uuid.New()
	reviewerID := uuid.New()
	
	req := review.AddManualReviewRequest{
		AnswerID:       answerID,
		ReviewerID:     reviewerID,
		ManualScore:    90.0,
		ManualFeedback: "Excellent",
	}

	answer := &domain.Answer{
		ID:           answerID,
		SubmissionID: submissionID,
		QuestionID:   uuid.New(),
		Text:         "Answer",
		CreatedAt:    time.Now(),
	}

	// Mock expectations
	mockAnswerRepo.On("FindByID", mock.Anything, answerID).Return(answer, nil)
	mockReviewRepo.On("UpsertManualScore", mock.Anything, answerID, reviewerID, req.ManualScore, req.ManualFeedback).
		Return(assert.AnError)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertNotCalled(t, "UpdateManualTotal")
}

// ============================================================================
// GetReviewUseCase Tests
// ============================================================================

func TestGetReview_Success(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	logger := getTestLogger()

	useCase := review.NewGetReviewUseCase(mockReviewRepo, logger)

	answerID := uuid.New()
	req := review.GetReviewRequest{
		AnswerID: answerID,
	}

	aiScore := 85.0
	manualScore := 90.0
	expectedReview := &domain.Review{
		ID:             uuid.New(),
		AnswerID:       answerID,
		AIScore:        &aiScore,
		AIFeedback:     new(string),
		ManualScore:    &manualScore,
		ManualFeedback: new(string),
	}
	*expectedReview.AIFeedback = "AI feedback here"
	*expectedReview.ManualFeedback = "Manual feedback here"

	// Mock expectations
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).Return(expectedReview, nil)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Review)
	assert.Equal(t, answerID, response.Review.AnswerID)
	assert.Equal(t, aiScore, *response.Review.AIScore)
	assert.Equal(t, manualScore, *response.Review.ManualScore)
	mockReviewRepo.AssertExpectations(t)
}

func TestGetReview_NotFound(t *testing.T) {
	// Arrange
	mockReviewRepo := new(MockReviewRepository)
	logger := getTestLogger()

	useCase := review.NewGetReviewUseCase(mockReviewRepo, logger)

	answerID := uuid.New()
	req := review.GetReviewRequest{
		AnswerID: answerID,
	}

	// Mock expectations
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).
		Return(nil, domain.ErrNotFound{Resource: "review", ID: answerID.String()})

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "review not found")
	mockReviewRepo.AssertExpectations(t)
}

// ============================================================================
// ListSubmissionsUseCase Tests
// ============================================================================

func TestListSubmissions_Success(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewListSubmissionsUseCase(mockSubmissionRepo, logger)

	testID := uuid.New()
	req := review.ListSubmissionsRequest{
		TestID: testID,
	}

	aiScore1 := 85.0
	aiScore2 := 92.0
	expectedSubmissions := []*domain.Submission{
		{
			ID:          uuid.New(),
			TestID:      testID,
			AccessEmail: "student1@example.com",
			AITotalScore: &aiScore1,
			SubmittedAt: time.Now(),
		},
		{
			ID:          uuid.New(),
			TestID:      testID,
			AccessEmail: "student2@example.com",
			AITotalScore: &aiScore2,
			SubmittedAt: time.Now(),
		},
	}

	// Mock expectations
	mockSubmissionRepo.On("FindByTestID", mock.Anything, testID, 1000, 0).Return(expectedSubmissions, nil)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Submissions, 2)
	assert.Equal(t, "student1@example.com", response.Submissions[0].AccessEmail)
	assert.Equal(t, "student2@example.com", response.Submissions[1].AccessEmail)
	mockSubmissionRepo.AssertExpectations(t)
}

func TestListSubmissions_EmptyList(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewListSubmissionsUseCase(mockSubmissionRepo, logger)

	testID := uuid.New()
	req := review.ListSubmissionsRequest{
		TestID: testID,
	}

	// Mock expectations
	mockSubmissionRepo.On("FindByTestID", mock.Anything, testID, 1000, 0).Return([]*domain.Submission{}, nil)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Submissions, 0)
	mockSubmissionRepo.AssertExpectations(t)
}

func TestListSubmissions_RepositoryError(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	logger := getTestLogger()

	useCase := review.NewListSubmissionsUseCase(mockSubmissionRepo, logger)

	testID := uuid.New()
	req := review.ListSubmissionsRequest{
		TestID: testID,
	}

	// Mock expectations
	mockSubmissionRepo.On("FindByTestID", mock.Anything, testID, 1000, 0).Return(nil, assert.AnError)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	mockSubmissionRepo.AssertExpectations(t)
}
