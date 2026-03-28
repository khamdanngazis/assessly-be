package unit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/submission"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// GenerateAccessTokenUseCase Tests
// ============================================================================

func TestGenerateAccessToken_Success(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockTokenGen := new(MockAccessTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)

	testID := uuid.New()
	publishedTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Sample Test",
		IsPublished: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := submission.GenerateAccessTokenRequest{
		TestID:      testID,
		Email:       "participant@example.com",
		AccessURL:   "https://example.com/test",
		ExpiryHours: 24,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)
	mockTokenGen.On("GenerateAccessToken", testID, "participant@example.com", 24).
		Return("access-token-123", nil)
	mockEmailSender.On("SendTestAccessToken", "participant@example.com", "Sample Test", "access-token-123", "https://example.com/test").
		Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	mockTestRepo.AssertExpectations(t)
	mockTokenGen.AssertExpectations(t)
	mockEmailSender.AssertExpectations(t)
}

func TestGenerateAccessToken_EmptyEmail(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockTokenGen := new(MockAccessTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)

	req := submission.GenerateAccessTokenRequest{
		TestID:      uuid.New(),
		Email:       "",
		AccessURL:   "https://example.com/test",
		ExpiryHours: 24,
	}

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "email", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
}

func TestGenerateAccessToken_EmptyAccessURL(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockTokenGen := new(MockAccessTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)

	req := submission.GenerateAccessTokenRequest{
		TestID:      uuid.New(),
		Email:       "test@example.com",
		AccessURL:   "",
		ExpiryHours: 24,
	}

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "access_url", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
}

func TestGenerateAccessToken_TestNotFound(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockTokenGen := new(MockAccessTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)

	testID := uuid.New()
	req := submission.GenerateAccessTokenRequest{
		TestID:      testID,
		Email:       "test@example.com",
		AccessURL:   "https://example.com/test",
		ExpiryHours: 24,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).
		Return(nil, domain.ErrNotFound{Resource: "test", ID: testID.String()})

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	mockTestRepo.AssertExpectations(t)
	mockTokenGen.AssertNotCalled(t, "GenerateAccessToken")
	mockEmailSender.AssertNotCalled(t, "SendTestAccessToken")
}

func TestGenerateAccessToken_TestNotPublished(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockTokenGen := new(MockAccessTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)

	testID := uuid.New()
	draftTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Draft Test",
		IsPublished: false, // Not published
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := submission.GenerateAccessTokenRequest{
		TestID:      testID,
		Email:       "test@example.com",
		AccessURL:   "https://example.com/test",
		ExpiryHours: 24,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(draftTest, nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "test_id", validationErr.Field)
	assert.Contains(t, validationErr.Message, "not published")
	mockTestRepo.AssertExpectations(t)
	mockTokenGen.AssertNotCalled(t, "GenerateAccessToken")
	mockEmailSender.AssertNotCalled(t, "SendTestAccessToken")
}

// ============================================================================
// SubmitTestUseCase Tests
// ============================================================================

func TestSubmitTest_Success(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockValidator := new(MockTokenValidator)
	mockScoringQueue := new(MockScoringQueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewSubmitTestUseCase(
		mockTestRepo,
		mockQuestionRepo,
		mockSubmissionRepo,
		mockAnswerRepo,
		mockValidator,
		mockScoringQueue,
		logger,
	)

	testID := uuid.New()
	q1ID := uuid.New()
	q2ID := uuid.New()

	publishedTest := &domain.Test{
		ID:           testID,
		CreatorID:    uuid.New(),
		Title:        "Test",
		IsPublished:  true,
		AllowRetakes: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	questions := []*domain.Question{
		{ID: q1ID, TestID: testID, Text: "Q1", ExpectedAnswer: "A1", OrderNum: 1},
		{ID: q2ID, TestID: testID, Text: "Q2", ExpectedAnswer: "A2", OrderNum: 2},
	}

	req := submission.SubmitTestRequest{
		AccessToken: "valid-token",
		Answers: []submission.AnswerInput{
			{QuestionID: q1ID, Text: "My answer 1"},
			{QuestionID: q2ID, Text: "My answer 2"},
		},
	}

	mockValidator.On("ValidateToken", "valid-token").
		Return(testID.String(), "participant@example.com", "participant", nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)
	mockSubmissionRepo.On("CountByTestAndEmail", mock.Anything, testID, "participant@example.com").
		Return(0, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID).Return(questions, nil)
	mockSubmissionRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.TestID == testID && s.AccessEmail == "participant@example.com"
	})).Return(nil)
	mockAnswerRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(answers []*domain.Answer) bool {
		return len(answers) == 2
	})).Return(nil)
	mockScoringQueue.On("Enqueue", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testID, result.TestID)
	assert.Equal(t, "participant@example.com", result.AccessEmail)
	mockValidator.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockScoringQueue.AssertExpectations(t)
}

func TestSubmitTest_EmptyAccessToken(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockValidator := new(MockTokenValidator)
	mockScoringQueue := new(MockScoringQueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewSubmitTestUseCase(
		mockTestRepo,
		mockQuestionRepo,
		mockSubmissionRepo,
		mockAnswerRepo,
		mockValidator,
		mockScoringQueue,
		logger,
	)

	req := submission.SubmitTestRequest{
		AccessToken: "",
		Answers: []submission.AnswerInput{
			{QuestionID: uuid.New(), Text: "Answer"},
		},
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "access_token", validationErr.Field)
	mockValidator.AssertNotCalled(t, "ValidateToken")
}

func TestSubmitTest_InvalidAccessToken(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockValidator := new(MockTokenValidator)
	mockScoringQueue := new(MockScoringQueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewSubmitTestUseCase(
		mockTestRepo,
		mockQuestionRepo,
		mockSubmissionRepo,
		mockAnswerRepo,
		mockValidator,
		mockScoringQueue,
		logger,
	)

	req := submission.SubmitTestRequest{
		AccessToken: "invalid-token",
		Answers: []submission.AnswerInput{
			{QuestionID: uuid.New(), Text: "Answer"},
		},
	}

	mockValidator.On("ValidateToken", "invalid-token").
		Return("", "", "", errors.New("invalid token"))

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var unauthorizedErr domain.ErrUnauthorized
	assert.True(t, errors.As(err, &unauthorizedErr))
	mockValidator.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "FindByID")
}

func TestSubmitTest_WrongRole(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockValidator := new(MockTokenValidator)
	mockScoringQueue := new(MockScoringQueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewSubmitTestUseCase(
		mockTestRepo,
		mockQuestionRepo,
		mockSubmissionRepo,
		mockAnswerRepo,
		mockValidator,
		mockScoringQueue,
		logger,
	)

	req := submission.SubmitTestRequest{
		AccessToken: "creator-token",
		Answers: []submission.AnswerInput{
			{QuestionID: uuid.New(), Text: "Answer"},
		},
	}

	mockValidator.On("ValidateToken", "creator-token").
		Return(uuid.New().String(), "creator@example.com", "creator", nil) // Wrong role

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var unauthorizedErr domain.ErrUnauthorized
	assert.True(t, errors.As(err, &unauthorizedErr))
	mockValidator.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "FindByID")
}

func TestSubmitTest_RetakesNotAllowed(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockValidator := new(MockTokenValidator)
	mockScoringQueue := new(MockScoringQueuer)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewSubmitTestUseCase(
		mockTestRepo,
		mockQuestionRepo,
		mockSubmissionRepo,
		mockAnswerRepo,
		mockValidator,
		mockScoringQueue,
		logger,
	)

	testID := uuid.New()
	publishedTest := &domain.Test{
		ID:           testID,
		CreatorID:    uuid.New(),
		Title:        "Test",
		IsPublished:  true,
		AllowRetakes: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	req := submission.SubmitTestRequest{
		AccessToken: "valid-token",
		Answers: []submission.AnswerInput{
			{QuestionID: uuid.New(), Text: "Answer"},
		},
	}

	mockValidator.On("ValidateToken", "valid-token").
		Return(testID.String(), "participant@example.com", "participant", nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)
	mockSubmissionRepo.On("CountByTestAndEmail", mock.Anything, testID, "participant@example.com").
		Return(1, nil) // Already submitted once

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "test_id", validationErr.Field)
	assert.Contains(t, validationErr.Message, "retakes not allowed")
	mockValidator.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "FindByTestID")
}

// ============================================================================
// GetSubmissionUseCase Tests
// ============================================================================

func TestGetSubmission_SuccessWithAccessToken(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockTestRepo := new(MockTestRepository)
	mockValidator := new(MockTokenValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGetSubmissionUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockReviewRepo,
		mockTestRepo,
		mockValidator,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	answerID := uuid.New()

	testSubmission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	answers := []*domain.Answer{
		{
			ID:           answerID,
			SubmissionID: submissionID,
			QuestionID:   uuid.New(),
			Text:         "My answer",
			CreatedAt:    time.Now(),
		},
	}

	req := submission.GetSubmissionRequest{
		SubmissionID: submissionID,
		AccessToken:  "valid-token",
		UserID:       nil,
		UserRole:     "", // Participant access via token
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(testSubmission, nil)
	mockValidator.On("ValidateToken", "valid-token").
		Return(testID.String(), "participant@example.com", "participant", nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).
		Return(nil, domain.ErrNotFound{Resource: "review", ID: answerID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, submissionID, result.Submission.ID)
	assert.Equal(t, 1, len(result.Answers))
	assert.Equal(t, answerID, result.Answers[0].Answer.ID)
	assert.Nil(t, result.Answers[0].Review) // No review yet
	mockSubmissionRepo.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
}

func TestGetSubmission_SuccessWithCreator(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockTestRepo := new(MockTestRepository)
	mockValidator := new(MockTokenValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGetSubmissionUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockReviewRepo,
		mockTestRepo,
		mockValidator,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	creatorID := uuid.New()
	answerID := uuid.New()

	testSubmission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	test := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test",
		IsPublished: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	answers := []*domain.Answer{
		{
			ID:           answerID,
			SubmissionID: submissionID,
			QuestionID:   uuid.New(),
			Text:         "My answer",
			CreatedAt:    time.Now(),
		},
	}

	req := submission.GetSubmissionRequest{
		SubmissionID: submissionID,
		AccessToken:  "",
		UserID:       &creatorID,
		UserRole:     "creator",
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(testSubmission, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(test, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).
		Return(nil, domain.ErrNotFound{Resource: "review", ID: answerID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, submissionID, result.Submission.ID)
	mockSubmissionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
}

func TestGetSubmission_Unauthorized(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockTestRepo := new(MockTestRepository)
	mockValidator := new(MockTokenValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGetSubmissionUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockReviewRepo,
		mockTestRepo,
		mockValidator,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	wrongUserID := uuid.New()
	actualCreatorID := uuid.New()

	testSubmission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	test := &domain.Test{
		ID:          testID,
		CreatorID:   actualCreatorID, // Different from the requesting user
		Title:       "Test",
		IsPublished: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := submission.GetSubmissionRequest{
		SubmissionID: submissionID,
		AccessToken:  "",
		UserID:       &wrongUserID,
		UserRole:     "creator", // Wrong creator
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(testSubmission, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(test, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var unauthorizedErr domain.ErrUnauthorized
	assert.True(t, errors.As(err, &unauthorizedErr))
	mockSubmissionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
	mockAnswerRepo.AssertNotCalled(t, "FindBySubmissionID")
}

func TestGetSubmission_SuccessWithReviewer(t *testing.T) {
	// Arrange
	mockSubmissionRepo := new(MockSubmissionRepository)
	mockAnswerRepo := new(MockAnswerRepository)
	mockReviewRepo := new(MockReviewRepository)
	mockTestRepo := new(MockTestRepository)
	mockValidator := new(MockTokenValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := submission.NewGetSubmissionUseCase(
		mockSubmissionRepo,
		mockAnswerRepo,
		mockReviewRepo,
		mockTestRepo,
		mockValidator,
		logger,
	)

	submissionID := uuid.New()
	testID := uuid.New()
	reviewerID := uuid.New()
	answerID := uuid.New()

	testSubmission := &domain.Submission{
		ID:          submissionID,
		TestID:      testID,
		AccessEmail: "participant@example.com",
		SubmittedAt: time.Now(),
	}

	answers := []*domain.Answer{
		{
			ID:           answerID,
			SubmissionID: submissionID,
			QuestionID:   uuid.New(),
			Text:         "My answer",
			CreatedAt:    time.Now(),
		},
	}

	req := submission.GetSubmissionRequest{
		SubmissionID: submissionID,
		AccessToken:  "",
		UserID:       &reviewerID,
		UserRole:     "reviewer",
	}

	mockSubmissionRepo.On("FindByID", mock.Anything, submissionID).Return(testSubmission, nil)
	mockAnswerRepo.On("FindBySubmissionID", mock.Anything, submissionID).Return(answers, nil)
	mockReviewRepo.On("FindByAnswerID", mock.Anything, answerID).
		Return(nil, domain.ErrNotFound{Resource: "review", ID: answerID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, submissionID, result.Submission.ID)
	assert.Equal(t, 1, len(result.Answers))
	assert.Equal(t, answerID, result.Answers[0].Answer.ID)
	mockSubmissionRepo.AssertExpectations(t)
	mockAnswerRepo.AssertExpectations(t)
	mockReviewRepo.AssertExpectations(t)
	// TestRepo should NOT be called for reviewers (they can access all submissions)
	mockTestRepo.AssertNotCalled(t, "FindByID")
}
