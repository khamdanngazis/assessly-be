package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/auth"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	testUC "github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestManagement_CreateAddQuestionsPublish tests the complete test management flow:
// 1. Creator creates a test
// 2. Creator adds multiple questions
// 3. Creator publishes the test
// 4. Verify test cannot be modified after publishing
func TestTestManagement_CreateAddQuestionsPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test environment
	loadTestEnv(t)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Setup test database
	pool := setupTestDatabase(t, ctx)
	defer pool.Close()

	// Setup repositories
	userRepo := postgres.NewUserRepository(pool)
	testRepo := postgres.NewTestRepository(pool)
	questionRepo := postgres.NewQuestionRepository(pool)

	// Create use cases
	createTestUC :=testUC.NewCreateTestUseCase(testRepo, logger)
	addQuestionUC := testUC.NewAddQuestionUseCase(questionRepo, testRepo, logger)
	publishTestUC := testUC.NewPublishTestUseCase(testRepo, questionRepo, logger)

	// Create a test creator user
	passwordHasher := auth.NewPasswordHasher(10)
	hashedPassword, err := passwordHasher.Hash("password123")
	require.NoError(t, err)

	creator := &domain.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("creator-%d@example.com", time.Now().Unix()),
		PasswordHash: hashedPassword,
		Role:         domain.UserRole("creator"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Create(ctx, creator)
	require.NoError(t, err)

	// Test data
	var testID uuid.UUID
	testTitle := "Software Engineering Mid-term Exam"
	testDescription := "Covers OOP, Design Patterns, and Testing"

	// Step 1: Create a test
	t.Run("CreateTest", func(t *testing.T) {
		req := testUC.CreateTestRequest{
			CreatorID:   creator.ID,
			Title:       testTitle,
			Description: testDescription,
		}

		test, err := createTestUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, test)
		assert.Equal(t, testTitle, test.Title)
		assert.Equal(t, testDescription, test.Description)
		assert.Equal(t, creator.ID, test.CreatorID)
		assert.False(t, test.IsPublished)
		testID = test.ID
	})

	// Step 2: Add questions to the test

	t.Run("AddQuestion1", func(t *testing.T) {
		req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "What is the main principle of Object-Oriented Programming?",
			ExpectedAnswer: "The main principles are: Encapsulation, Inheritance, Polymorphism, and Abstraction.",
			OrderNum:       0, // Auto-assign
		}

		question, err := addQuestionUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, question)
		assert.Equal(t, req.Text, question.Text)
		assert.Equal(t, req.ExpectedAnswer, question.ExpectedAnswer)
		assert.Equal(t, 1, question.OrderNum) // First question
	})

	t.Run("AddQuestion2", func(t *testing.T) {
		req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "Explain the Singleton design pattern and when to use it.",
			ExpectedAnswer: "Singleton ensures a class has only one instance and provides a global point of access to it. Use when you need exactly one instance of a class.",
			OrderNum:       0,
		}

		question, err := addQuestionUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, question)
		assert.Equal(t, 2, question.OrderNum) // Second question
	})

	t.Run("AddQuestion3_WithSpecificOrder", func(t *testing.T) {
		req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "What is Test-Driven Development (TDD)?",
			ExpectedAnswer: "TDD is a software development process where you write tests before writing the actual code. The cycle is: Red (failing test) → Green (make it pass) → Refactor.",
			OrderNum:       3,
		}

		question, err := addQuestionUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, question)
		assert.Equal(t, 3, question.OrderNum)
	})

	// Step 3: Verify questions were added
	t.Run("VerifyQuestionsAdded", func(t *testing.T) {
		questions, err := questionRepo.FindByTestID(ctx, testID)
		require.NoError(t, err)
		assert.Len(t, questions, 3)
		
		// Verify order
		assert.Equal(t, 1, questions[0].OrderNum)
		assert.Equal(t, 2, questions[1].OrderNum)
		assert.Equal(t, 3, questions[2].OrderNum)
	})

	// Step 4: Publish the test
	t.Run("PublishTest", func(t *testing.T) {
		req := testUC.PublishTestRequest{
			TestID: testID,
		}

		test, err := publishTestUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, test)
		assert.True(t, test.IsPublished)
	})

	// Step 5: Adding question to published test should fail
	t.Run("AddQuestion_AfterPublish_ShouldFail", func(t *testing.T) {
		req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "This should fail",
			ExpectedAnswer: "Because test is published",
			OrderNum:       0,
		}

		question, err := addQuestionUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, question)
		assert.Contains(t, err.Error(), "published")
	})

	// Step 6: Publishing already published test should fail
	t.Run("PublishTest_AlreadyPublished_ShouldFail", func(t *testing.T) {
		req := testUC.PublishTestRequest{
			TestID: testID,
		}

		test, err := publishTestUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, test)
		assert.Contains(t, err.Error(), "already published")
	})

	// Step 7: Verify test cannot be modified after publishing
	t.Run("VerifyTestImmutableAfterPublish", func(t *testing.T) {
		// Retrieve test
		test, err := testRepo.FindByID(ctx, testID)
		require.NoError(t, err)
		assert.True(t, test.IsPublished)
		
		// Questions should still be there
		questions, err := questionRepo.FindByTestID(ctx, testID)
		require.NoError(t, err)
		assert.Len(t, questions, 3)
	})

	// Step 8: Test edge cases
	t.Run("CreateTest_EmptyTitle_ShouldFail", func(t *testing.T) {
		req := testUC.CreateTestRequest{
			CreatorID:   creator.ID,
			Title:       "",
			Description: "Test with no title",
		}

		test, err := createTestUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, test)
	})

	t.Run("AddQuestion_EmptyText_ShouldFail", func(t *testing.T) {
		// Create new unpublished test
		req := testUC.CreateTestRequest{
			CreatorID:   creator.ID,
			Title:       "Test for validation",
			Description: "Test validation",
		}
		newTest, err := createTestUC.Execute(ctx, req)
		require.NoError(t, err)

		// Try to add question with empty text
		addReq := testUC.AddQuestionRequest{
			TestID:         newTest.ID,
			Text:           "",
			ExpectedAnswer: "This should fail",
			OrderNum:       0,
		}

		question, err := addQuestionUC.Execute(ctx, addReq)
		assert.Error(t, err)
		assert.Nil(t, question)
	})

	t.Run("PublishTest_NoQuestions_ShouldFail", func(t *testing.T) {
		// Create test without questions
		req := testUC.CreateTestRequest{
			CreatorID:   creator.ID,
			Title:       "Test without questions",
			Description: "Cannot be published",
		}
		emptyTest, err := createTestUC.Execute(ctx, req)
		require.NoError(t, err)

		// Try to publish
		pubReq := testUC.PublishTestRequest{
			TestID: emptyTest.ID,
		}

		test, err := publishTestUC.Execute(ctx, pubReq)
		assert.Error(t, err)
		assert.Nil(t, test)
		assert.Contains(t, err.Error(), "at least one question")
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Delete questions
		_, err := pool.Exec(ctx, "DELETE FROM questions WHERE test_id = $1", testID)
		require.NoError(t, err)

		// Delete tests
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE creator_id = $1", creator.ID)
		require.NoError(t, err)

		// Delete user
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", creator.ID)
		require.NoError(t, err)
	})
}
