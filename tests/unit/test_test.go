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
// UpdateTestUseCase Tests
// ============================================================================

func TestUpdateTest_Success(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()

	existingTest := &domain.Test{
		ID:           testID,
		CreatorID:    creatorID,
		Title:        "Old Title",
		Description:  "Old Description",
		AllowRetakes: false,
		IsPublished:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	req := test.UpdateTestRequest{
		TestID:       testID,
		CreatorID:    creatorID,
		Title:        "Updated Title",
		Description:  "Updated Description",
		AllowRetakes: true,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(existingTest, nil)
	mockTestRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *domain.Test) bool {
		return t.ID == testID &&
			t.Title == "Updated Title" &&
			t.Description == "Updated Description" &&
			t.AllowRetakes == true
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Updated Title", result.Title)
	assert.Equal(t, "Updated Description", result.Description)
	assert.True(t, result.AllowRetakes)
	mockTestRepo.AssertExpectations(t)
}

func TestUpdateTest_UnauthorizedCreator(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	creatorID := uuid.New()
	otherCreatorID := uuid.New()
	testID := uuid.New()

	existingTest := &domain.Test{
		ID:           testID,
		CreatorID:    otherCreatorID, // Different creator
		Title:        "Test Title",
		Description:  "Test Description",
		AllowRetakes: false,
		IsPublished:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	req := test.UpdateTestRequest{
		TestID:       testID,
		CreatorID:    creatorID,
		Title:        "Updated Title",
		Description:  "Updated Description",
		AllowRetakes: true,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(existingTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockTestRepo.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestUpdateTest_PublishedTest(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()

	existingTest := &domain.Test{
		ID:           testID,
		CreatorID:    creatorID,
		Title:        "Test Title",
		Description:  "Test Description",
		AllowRetakes: false,
		IsPublished:  true, // Published test
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	req := test.UpdateTestRequest{
		TestID:       testID,
		CreatorID:    creatorID,
		Title:        "Updated Title",
		Description:  "Updated Description",
		AllowRetakes: true,
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(existingTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.ErrorAs(t, err, &validationErr)
	mockTestRepo.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestUpdateTest_TestNotFound(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	testID := uuid.New()

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(nil, domain.ErrNotFound{
		Resource: "test",
		ID:       testID.String(),
	})

	// Act
	result, err := useCase.Execute(context.Background(), test.UpdateTestRequest{
		TestID:       testID,
		CreatorID:    uuid.New(),
		Title:        "Updated Title",
		Description:  "Updated Description",
		AllowRetakes: true,
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var notFoundErr domain.ErrNotFound
	assert.ErrorAs(t, err, &notFoundErr)
	mockTestRepo.AssertExpectations(t)
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestUpdateTest_EmptyTitle(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	req := test.UpdateTestRequest{
		TestID:       uuid.New(),
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
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "title", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
	mockTestRepo.AssertNotCalled(t, "Update")
}

func TestUpdateTest_TitleTooLong(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateTestUseCase(mockTestRepo, logger)

	longTitle := make([]byte, 256)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	req := test.UpdateTestRequest{
		TestID:       uuid.New(),
		CreatorID:    uuid.New(),
		Title:        string(longTitle),
		Description:  "Description",
		AllowRetakes: false,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "title", validationErr.Field)
	mockTestRepo.AssertNotCalled(t, "FindByID")
	mockTestRepo.AssertNotCalled(t, "Update")
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

// ============================================================================
// ListTestsUseCase Tests
// ============================================================================

func TestListTests_CreatorSuccess(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	testID1 := uuid.New()
	testID2 := uuid.New()
	expectedTests := []*domain.Test{
		{
			ID:          testID1,
			CreatorID:   creatorID,
			Title:       "Test 1",
			Description: "Description 1",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          testID2,
			CreatorID:   creatorID,
			Title:       "Test 2",
			Description: "Description 2",
			IsPublished: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	req := test.ListTestsRequest{
		UserID:   creatorID,
		UserRole: "creator",
		Status:   "all",
		Page:     1,
		PageSize: 20,
	}

	mockTestRepo.On("FindByCreatorID", mock.Anything, creatorID, 20, 0).
		Return(expectedTests, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID1).Return([]*domain.Question{}, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID2).Return([]*domain.Question{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Tests, 2)
	assert.Equal(t, "Test 1", result.Tests[0].Title)
	assert.Equal(t, "Test 2", result.Tests[1].Title)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestListTests_CreatorFilterPublished(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	publishedID := uuid.New()
	draftID := uuid.New()
	allTests := []*domain.Test{
		{
			ID:          publishedID,
			CreatorID:   creatorID,
			Title:       "Published Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          draftID,
			CreatorID:   creatorID,
			Title:       "Draft Test",
			IsPublished: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	req := test.ListTestsRequest{
		UserID:   creatorID,
		UserRole: "creator",
		Status:   "published",
		Page:     1,
		PageSize: 20,
	}

	mockTestRepo.On("FindByCreatorID", mock.Anything, creatorID, 20, 0).
		Return(allTests, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, publishedID).Return([]*domain.Question{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Tests, 1)
	assert.Equal(t, "Published Test", result.Tests[0].Title)
	assert.True(t, result.Tests[0].IsPublished)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestListTests_CreatorFilterDraft(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	draftID := uuid.New()
	publishedID := uuid.New()
	allTests := []*domain.Test{
		{
			ID:          publishedID,
			CreatorID:   creatorID,
			Title:       "Published Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          draftID,
			CreatorID:   creatorID,
			Title:       "Draft Test",
			IsPublished: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	req := test.ListTestsRequest{
		UserID:   creatorID,
		UserRole: "creator",
		Status:   "draft",
		Page:     1,
		PageSize: 20,
	}

	mockTestRepo.On("FindByCreatorID", mock.Anything, creatorID, 20, 0).
		Return(allTests, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, draftID).Return([]*domain.Question{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Tests, 1)
	assert.Equal(t, "Draft Test", result.Tests[0].Title)
	assert.False(t, result.Tests[0].IsPublished)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestListTests_ReviewerSuccess(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	reviewerID := uuid.New()
	testID1 := uuid.New()
	testID2 := uuid.New()
	expectedTests := []*domain.Test{
		{
			ID:          testID1,
			CreatorID:   uuid.New(),
			Title:       "Published Test 1",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          testID2,
			CreatorID:   uuid.New(),
			Title:       "Published Test 2",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	req := test.ListTestsRequest{
		UserID:   reviewerID,
		UserRole: "reviewer",
		Status:   "all",
		Page:     1,
		PageSize: 20,
	}

	mockTestRepo.On("FindPublished", mock.Anything, 20, 0).
		Return(expectedTests, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID1).Return([]*domain.Question{}, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID2).Return([]*domain.Question{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Tests, 2)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestListTests_UnauthorizedRole(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	req := test.ListTestsRequest{
		UserID:   uuid.New(),
		UserRole: "participant",
		Status:   "all",
		Page:     1,
		PageSize: 20,
	}

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var unauthorizedErr domain.ErrUnauthorized
	assert.True(t, errors.As(err, &unauthorizedErr))
	mockTestRepo.AssertNotCalled(t, "FindByCreatorID")
	mockTestRepo.AssertNotCalled(t, "FindPublished")
}

func TestListTests_PaginationDefaults(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	req := test.ListTestsRequest{
		UserID:   creatorID,
		UserRole: "creator",
		Status:   "all",
		Page:     0, // Should default to 1
		PageSize: 0, // Should default to 20
	}

	mockTestRepo.On("FindByCreatorID", mock.Anything, creatorID, 20, 0).
		Return([]*domain.Test{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockTestRepo.AssertExpectations(t)
}

func TestListTests_PaginationMaxLimit(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewListTestsUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	req := test.ListTestsRequest{
		UserID:   creatorID,
		UserRole: "creator",
		Status:   "all",
		Page:     1,
		PageSize: 200, // Should be capped at 100
	}

	mockTestRepo.On("FindByCreatorID", mock.Anything, creatorID, 100, 0).
		Return([]*domain.Test{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockTestRepo.AssertExpectations(t)
}

// GetTestUseCase Tests

func TestGetTest_CreatorSuccess(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	testID := uuid.New()
	expectedTest := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test Title",
		Description: "Test Description",
		IsPublished: false,
	}
	expectedQuestions := []*domain.Question{}

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   creatorID,
		UserRole: "creator",
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(expectedTest, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID).Return(expectedQuestions, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testID, result.Test.ID)
	assert.Equal(t, creatorID, result.Test.CreatorID)
	assert.NotNil(t, result.Questions)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestGetTest_CreatorCannotAccessOthersTest(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	otherCreatorID := uuid.New()
	testID := uuid.New()

	otherTest := &domain.Test{
		ID:          testID,
		CreatorID:   otherCreatorID, // Different creator
		Title:       "Other Test",
		Description: "Test Description",
		IsPublished: true,
	}

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   creatorID,
		UserRole: "creator",
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(otherTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockTestRepo.AssertExpectations(t)
}

func TestGetTest_ReviewerCanAccessPublishedTest(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	reviewerID := uuid.New()
	testID := uuid.New()
	publishedTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Published Test",
		Description: "Test Description",
		IsPublished: true,
	}

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   reviewerID,
		UserRole: "reviewer",
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(publishedTest, nil)
	mockQuestionRepo.On("FindByTestID", mock.Anything, testID).Return([]*domain.Question{}, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testID, result.Test.ID)
	assert.True(t, result.Test.IsPublished)
	mockTestRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestGetTest_ReviewerCannotAccessDraftTest(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	reviewerID := uuid.New()
	testID := uuid.New()
	draftTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Draft Test",
		Description: "Test Description",
		IsPublished: false, // Not published
	}

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   reviewerID,
		UserRole: "reviewer",
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(draftTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockTestRepo.AssertExpectations(t)
}

func TestGetTest_TestNotFound(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	creatorID := uuid.New()
	testID := uuid.New()

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   creatorID,
		UserRole: "creator",
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).
		Return(nil, domain.ErrNotFound{Resource: "test", ID: testID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var notFoundErr domain.ErrNotFound
	assert.ErrorAs(t, err, &notFoundErr)
	mockTestRepo.AssertExpectations(t)
}

func TestGetTest_UnauthorizedRole(t *testing.T) {
	// Arrange
	mockTestRepo := new(MockTestRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	useCase := test.NewGetTestUseCase(mockTestRepo, mockQuestionRepo)

	testID := uuid.New()
	participantID := uuid.New()

	someTest := &domain.Test{
		ID:          testID,
		CreatorID:   uuid.New(),
		Title:       "Test",
		Description: "Test Description",
		IsPublished: true,
	}

	req := test.GetTestRequest{
		TestID:   testID,
		UserID:   participantID,
		UserRole: "participant", // Invalid role
	}

	mockTestRepo.On("FindByID", mock.Anything, testID).Return(someTest, nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockTestRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateQuestionUseCase Tests
// ============================================================================

func TestUpdateQuestion_Success(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Old question text",
		ExpectedAnswer: "Old expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test",
		IsPublished: false,
	}

	req := test.UpdateQuestionRequest{
		QuestionID:     questionID,
		TestID:         testID,
		CreatorID:      creatorID,
		Text:           "Updated question text",
		ExpectedAnswer: "Updated expected answer",
		OrderNum:       2,
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)
	mockQuestionRepo.On("Update", mock.Anything, mock.MatchedBy(func(q *domain.Question) bool {
		return q.ID == questionID &&
			q.Text == "Updated question text" &&
			q.ExpectedAnswer == "Updated expected answer" &&
			q.OrderNum == 2
	})).Return(nil)

	// Act
	result, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Updated question text", result.Text)
	assert.Equal(t, "Updated expected answer", result.ExpectedAnswer)
	assert.Equal(t, 2, result.OrderNum)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestUpdateQuestion_UnauthorizedCreator(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	otherCreatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Question text",
		ExpectedAnswer: "Expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   otherCreatorID, // Different creator
		Title:       "Test",
		IsPublished: false,
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)

	// Act
	result, err := useCase.Execute(context.Background(), test.UpdateQuestionRequest{
		QuestionID:     questionID,
		TestID:         testID,
		CreatorID:      creatorID,
		Text:           "Updated text",
		ExpectedAnswer: "Updated answer",
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestUpdateQuestion_PublishedTest(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Question text",
		ExpectedAnswer: "Expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test",
		IsPublished: true, // Published test
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)

	// Act
	result, err := useCase.Execute(context.Background(), test.UpdateQuestionRequest{
		QuestionID:     questionID,
		TestID:         testID,
		CreatorID:      creatorID,
		Text:           "Updated text",
		ExpectedAnswer: "Updated answer",
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	var validationErr domain.ErrValidation
	assert.ErrorAs(t, err, &validationErr)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestUpdateQuestion_QuestionNotFound(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewUpdateQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	questionID := uuid.New()
	testID := uuid.New()
	creatorID := uuid.New()

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(nil, domain.ErrNotFound{Resource: "question", ID: questionID.String()})

	// Act
	result, err := useCase.Execute(context.Background(), test.UpdateQuestionRequest{
		QuestionID:     questionID,
		TestID:         testID,
		CreatorID:      creatorID,
		Text:           "Updated text",
		ExpectedAnswer: "Updated answer",
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockQuestionRepo.AssertExpectations(t)
}

// ============================================================================
// DeleteQuestionUseCase Tests
// ============================================================================

func TestDeleteQuestion_Success(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewDeleteQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Question text",
		ExpectedAnswer: "Expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test",
		IsPublished: false,
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)
	mockQuestionRepo.On("Delete", mock.Anything, questionID).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), test.DeleteQuestionRequest{
		QuestionID: questionID,
		TestID:     testID,
		CreatorID:  creatorID,
	})

	// Assert
	assert.NoError(t, err)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestDeleteQuestion_UnauthorizedCreator(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewDeleteQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	otherCreatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Question text",
		ExpectedAnswer: "Expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   otherCreatorID, // Different creator
		Title:       "Test",
		IsPublished: false,
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)

	// Act
	err := useCase.Execute(context.Background(), test.DeleteQuestionRequest{
		QuestionID: questionID,
		TestID:     testID,
		CreatorID:  creatorID,
	})

	// Assert
	assert.Error(t, err)
	var authErr domain.ErrUnauthorized
	assert.ErrorAs(t, err, &authErr)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestDeleteQuestion_PublishedTest(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewDeleteQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	creatorID := uuid.New()
	testID := uuid.New()
	questionID := uuid.New()

	existingQuestion := &domain.Question{
		ID:             questionID,
		TestID:         testID,
		Text:           "Question text",
		ExpectedAnswer: "Expected answer",
		OrderNum:       1,
		CreatedAt:      time.Now(),
	}

	testEntity := &domain.Test{
		ID:          testID,
		CreatorID:   creatorID,
		Title:       "Test",
		IsPublished: true, // Published test
	}

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(existingQuestion, nil)
	mockTestRepo.On("FindByID", mock.Anything, testID).Return(testEntity, nil)

	// Act
	err := useCase.Execute(context.Background(), test.DeleteQuestionRequest{
		QuestionID: questionID,
		TestID:     testID,
		CreatorID:  creatorID,
	})

	// Assert
	assert.Error(t, err)
	var validationErr domain.ErrValidation
	assert.ErrorAs(t, err, &validationErr)
	mockQuestionRepo.AssertExpectations(t)
	mockTestRepo.AssertExpectations(t)
}

func TestDeleteQuestion_QuestionNotFound(t *testing.T) {
	// Arrange
	mockQuestionRepo := new(MockQuestionRepository)
	mockTestRepo := new(MockTestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	useCase := test.NewDeleteQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)

	questionID := uuid.New()
	testID := uuid.New()
	creatorID := uuid.New()

	mockQuestionRepo.On("FindByID", mock.Anything, questionID).Return(nil, domain.ErrNotFound{Resource: "question", ID: questionID.String()})

	// Act
	err := useCase.Execute(context.Background(), test.DeleteQuestionRequest{
		QuestionID: questionID,
		TestID:     testID,
		CreatorID:  creatorID,
	})

	// Assert
	assert.Error(t, err)
	mockQuestionRepo.AssertExpectations(t)
}
