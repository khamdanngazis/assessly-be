package unit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// CreateTestUseCase Tests
// ============================================================================

func TestCreateTest_Success(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewCreateTestUseCase(mockTestRepo, logger)

	creatorID := uuid.New()
	req := test.CreateTestRequest{
		CreatorID:    creatorID,
		Title:        "Sample Test",
		Description:  "A sample test description",
		AllowRetakes: true,
	}

	mockTestRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *domain.Test) bool {
		return t.Title == "Sample Test" &&
			t.CreatorID == creatorID &&
			t.Description == "A sample test description" &&
			t.AllowRetakes == true &&
			t.IsPublished == false
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Sample Test", result.Title)
	assert.Equal(t, creatorID, result.CreatorID)
	assert.Equal(t, "A sample test description", result.Description)
	assert.True(t, result.AllowRetakes)
	assert.False(t, result.IsPublished)
	mockTestRepo.AssertExpectations(t)
}

func TestCreateTest_EmptyTitle(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewCreateTestUseCase(mockTestRepo, logger)

	req := test.CreateTestRequest{
		CreatorID:    uuid.New(),
		Title:        "",
		Description:  "Description",
		AllowRetakes: false,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "title", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "Create")
}

func TestCreateTest_TitleTooLong(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewCreateTestUseCase(mockTestRepo, logger)

	longTitle := string(make([]byte, 256))
	for i := range longTitle {
		longTitle = longTitle[:i] + "a" + longTitle[i+1:]
	}

	req := test.CreateTestRequest{
		CreatorID:    uuid.New(),
		Title:        longTitle,
		Description:  "Description",
		AllowRetakes: false,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "title", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "Create")
}

func TestCreateTest_RepositoryError(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewCreateTestUseCase(mockTestRepo, logger)

	req := test.CreateTestRequest{
		CreatorID:    uuid.New(),
		Title:        "Valid Title",
		Description:  "Valid Description",
		AllowRetakes: false,
	}

	mockTestRepo.On("Create", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
	mockTestRepo.AssertExpectations(t)
}

// ============================================================================
// AddQuestionUseCase Tests
// ============================================================================

func TestAddQuestion_Success(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	testID := uuid.New()
	existingTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.AddQuestionRequest{
		TestID:         testID,
		Text:           "What is Go?",
		ExpectedAnswer: "A programming language",
		OrderNum:       1,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(existingTest, nil)
	mockQuestionRepo.On("Create", mock.Anything, mock.MatchedBy(func(q *domain.Question) bool {
		return q.TestID == testID &&
			q.Text == "What is Go?" &&
			q.ExpectedAnswer == "A programming language" &&
			q.OrderNum == 1
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testID, result.TestID)
	assert.Equal(t, "What is Go?", result.Text)
	assert.Equal(t, "A programming language", result.ExpectedAnswer)
	assert.Equal(t, 1, result.OrderNum)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestAddQuestion_AutoAssignOrderNum(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	testID := uuid.New()
	existingTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.AddQuestionRequest{
		TestID:         testID,
		Text:           "What is Go?",
		ExpectedAnswer: "A programming language",
		OrderNum:       0, // Auto-assign
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(existingTest, nil)
	mockQuestionRepo.On("CountByTestID", mock.Anything, testID).Return(5, nil)
	mockQuestionRepo.On("Create", mock.Anything, mock.MatchedBy(func(q *domain.Question) bool {
		return q.OrderNum == 6 // Should be count + 1
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 6, result.OrderNum)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestAddQuestion_TestNotFound(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	testID := uuid.New()
	req := test.AddQuestionRequest{
		TestID:         testID,
		Text:           "What is Go?",
		ExpectedAnswer: "A programming language",
		OrderNum:       1,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).
		Return(nil, domain.ErrNotFound{Resource: "test", ID: testID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "Create")
}

func TestAddQuestion_PublishedTest(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	testID := uuid.New()
	publishedTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: true, // Already published
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.AddQuestionRequest{
		TestID:         testID,
		Text:           "What is Go?",
		ExpectedAnswer: "A programming language",
		OrderNum:       1,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "test_id", validationErr.Field)
	assert.Contains(t, validationErr.Message, "published")
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "Create")
}

func TestAddQuestion_EmptyText(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	req := test.AddQuestionRequest{
		TestID:         uuid.New(),
		Text:           "",
		ExpectedAnswer: "An answer",
		OrderNum:       1,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "text", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
	mockQuestionRepo.AssertNotCalled(t, "Create")
}

func TestAddQuestion_EmptyExpectedAnswer(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	req := test.AddQuestionRequest{
		TestID:         uuid.New(),
		Text:           "What is Go?",
		ExpectedAnswer: "",
		OrderNum:       1,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "expected_answer", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
	mockQuestionRepo.AssertNotCalled(t, "Create")
}

// ============================================================================
// PublishTestUseCase Tests
// ============================================================================

func TestPublishTest_Success(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewPublishTestUseCase(mockTestRepo, mockQuestionRepo, logger)

	testID := uuid.New()
	draftTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.PublishTestRequest{
		TestID: testID,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(draftTest, nil)
	mockQuestionRepo.On("CountByTestID", mock.Anything, testID).Return(5, nil)
	mockTestRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *domain.Test) bool {
		return t.ID == testID && t.IsPublished == true
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsPublished)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestPublishTest_TestNotFound(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewPublishTestUseCase(mockTestRepo, mockQuestionRepo, logger)

	testID := uuid.New()
	req := test.PublishTestRequest{
		TestID: testID,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).
		Return(nil, domain.ErrNotFound{Resource: "test", ID: testID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "CountByTestID")
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestPublishTest_AlreadyPublished(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewPublishTestUseCase(mockTestRepo, mockQuestionRepo, logger)

	testID := uuid.New()
	publishedTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: true, // Already published
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.PublishTestRequest{
		TestID: testID,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "test_id", validationErr.Field)
	assert.Contains(t, validationErr.Message, "already published")
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertNotCalled(t, "CountByTestID")
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestPublishTest_NoQuestions(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewPublishTestUseCase(mockTestRepo, mockQuestionRepo, logger)

	testID := uuid.New()
	draftTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		IsPublished: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := test.PublishTestRequest{
		TestID: testID,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(draftTest, nil)
	mockQuestionRepo.On("CountByTestID", mock.Anything, testID).Return(0, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "questions", validationErr.Field)
	assert.Contains(t, validationErr.Message, "at least one question")
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "Update")
}
