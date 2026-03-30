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

// TestListTests_RoleBasedFiltering tests listing tests with role-based filtering
func TestListTests_RoleBasedFiltering(t *testing.T) {
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
	createTestUC := testUC.NewCreateTestUseCase(testRepo, logger)
	addQuestionUC := testUC.NewAddQuestionUseCase(questionRepo, testRepo, logger)
	publishTestUC := testUC.NewPublishTestUseCase(testRepo, questionRepo, logger)
	listTestsUC := testUC.NewListTestsUseCase(testRepo)

	// Create password hasher
	passwordHasher := auth.NewPasswordHasher(10)
	hashedPassword, err := passwordHasher.Hash("password123")
	require.NoError(t, err)

	// Create creator 1
	creator1 := &domain.User{
		ID:           uuid.New(),
		Name:         "Creator One",
		Email:        fmt.Sprintf("creator1-%d@example.com", time.Now().Unix()),
		PasswordHash: hashedPassword,
		Role:         domain.UserRole("creator"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Create(ctx, creator1)
	require.NoError(t, err)

	// Create creator 2
	creator2 := &domain.User{
		ID:           uuid.New(),
		Name:         "Creator Two",
		Email:        fmt.Sprintf("creator2-%d@example.com", time.Now().Unix()),
		PasswordHash: hashedPassword,
		Role:         domain.UserRole("creator"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Create(ctx, creator2)
	require.NoError(t, err)

	// Create reviewer
	reviewer := &domain.User{
		ID:           uuid.New(),
		Name:         "Test Reviewer",
		Email:        fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
		PasswordHash: hashedPassword,
		Role:         domain.UserRole("reviewer"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Create(ctx, reviewer)
	require.NoError(t, err)

	// Creator 1: Create 2 tests (1 published, 1 draft)
	test1Creator1 := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator1.ID, "Creator1 Published Test", "Description 1")
	_, err = publishTestUC.Execute(ctx, testUC.PublishTestRequest{TestID: test1Creator1.ID})
	require.NoError(t, err)

	test2Creator1 := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator1.ID, "Creator1 Draft Test", "Description 2")

	// Creator 2: Create 1 published test
	test1Creator2 := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator2.ID, "Creator2 Published Test", "Description 3")
	_, err = publishTestUC.Execute(ctx, testUC.PublishTestRequest{TestID: test1Creator2.ID})
	require.NoError(t, err)

	// Test 1: Creator1 lists all their tests
	t.Run("Creator1_ListAllTests", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   creator1.ID,
			UserRole: "creator",
			Status:   "all",
			Page:     1,
			PageSize: 20,
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Len(t, result.Tests, 2)
	})

	// Test 2: Creator1 lists only published tests
	t.Run("Creator1_ListPublishedTests", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   creator1.ID,
			UserRole: "creator",
			Status:   "published",
			Page:     1,
			PageSize: 20,
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Len(t, result.Tests, 1)
		assert.True(t, result.Tests[0].IsPublished)
		assert.Equal(t, "Creator1 Published Test", result.Tests[0].Title)
	})

	// Test 3: Creator1 lists only draft tests
	t.Run("Creator1_ListDraftTests", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   creator1.ID,
			UserRole: "creator",
			Status:   "draft",
			Page:     1,
			PageSize: 20,
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Len(t, result.Tests, 1)
		assert.False(t, result.Tests[0].IsPublished)
		assert.Equal(t, "Creator1 Draft Test", result.Tests[0].Title)
	})

	// Test 4: Creator2 should only see their own tests
	t.Run("Creator2_ListTests_ShouldOnlySeeOwnTests", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   creator2.ID,
			UserRole: "creator",
			Status:   "all",
			Page:     1,
			PageSize: 20,
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Len(t, result.Tests, 1)
		assert.Equal(t, "Creator2 Published Test", result.Tests[0].Title)
	})

	// Test 5: Reviewer should see all published tests
	t.Run("Reviewer_ListTests_ShouldSeeAllPublished", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   reviewer.ID,
			UserRole: "reviewer",
			Status:   "all",
			Page:     1,
			PageSize: 20,
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total) // Should see both published tests
		assert.Len(t, result.Tests, 2)

		// All tests should be published
		for _, test := range result.Tests {
			assert.True(t, test.IsPublished)
		}
	})

	// Test 6: Pagination test
	t.Run("Pagination_PageSize", func(t *testing.T) {
		req := testUC.ListTestsRequest{
			UserID:   reviewer.ID,
			UserRole: "reviewer",
			Status:   "all",
			Page:     1,
			PageSize: 1, // Only 1 per page
		}

		result, err := listTestsUC.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Tests, 1) // Should only return 1
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Delete questions for all tests
		_, err := pool.Exec(ctx, "DELETE FROM questions WHERE test_id IN ($1, $2, $3)", 
			test1Creator1.ID, test2Creator1.ID, test1Creator2.ID)
		require.NoError(t, err)

		// Delete all tests
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id IN ($1, $2, $3)", 
			test1Creator1.ID, test2Creator1.ID, test1Creator2.ID)
		require.NoError(t, err)

		// Delete users
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2, $3)", 
			creator1.ID, creator2.ID, reviewer.ID)
		require.NoError(t, err)
	})
}

// Helper function to create a test with one question
func createTestWithQuestion(t *testing.T, ctx context.Context, createTestUC *testUC.CreateTestUseCase, 
	addQuestionUC *testUC.AddQuestionUseCase, creatorID uuid.UUID, title, description string) *domain.Test {
	
	// Create test
	req := testUC.CreateTestRequest{
		CreatorID:   creatorID,
		Title:       title,
		Description: description,
	}
	test, err := createTestUC.Execute(ctx, req)
	require.NoError(t, err)

	// Add one question so it can be published
	addReq := testUC.AddQuestionRequest{
		TestID:         test.ID,
		Text:           "Sample question for " + title,
		ExpectedAnswer: "Sample answer",
		OrderNum:       1,
	}
	_, err = addQuestionUC.Execute(ctx, addReq)
	require.NoError(t, err)

	return test
}

// TestGetTest_RoleBasedAccess tests getting a single test with role-based access control
func TestGetTest_RoleBasedAccess(t *testing.T) {
if testing.Short() {
t.Skip("Skipping integration test in short mode")
}

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
createTestUC := testUC.NewCreateTestUseCase(testRepo, logger)
addQuestionUC := testUC.NewAddQuestionUseCase(questionRepo, testRepo, logger)
publishTestUC := testUC.NewPublishTestUseCase(testRepo, questionRepo, logger)
getTestUC := testUC.NewGetTestUseCase(testRepo)

// Create password hasher
passwordHasher := auth.NewPasswordHasher(10)
hashedPassword, err := passwordHasher.Hash("password123")
require.NoError(t, err)

// Create creator 1
creator1 := &domain.User{
ID:           uuid.New(),
Name:         "Creator One",
Email:        fmt.Sprintf("creator1-%d@example.com", time.Now().Unix()),
PasswordHash: hashedPassword,
Role:         domain.UserRole("creator"),
CreatedAt:    time.Now(),
UpdatedAt:    time.Now(),
}
err = userRepo.Create(ctx, creator1)
require.NoError(t, err)

// Create creator 2
creator2 := &domain.User{
ID:           uuid.New(),
Name:         "Creator Two",
Email:        fmt.Sprintf("creator2-%d@example.com", time.Now().Unix()),
PasswordHash: hashedPassword,
Role:         domain.UserRole("creator"),
CreatedAt:    time.Now(),
UpdatedAt:    time.Now(),
}
err = userRepo.Create(ctx, creator2)
require.NoError(t, err)

// Create reviewer
reviewer := &domain.User{
ID:           uuid.New(),
Name:         "Test Reviewer",
Email:        fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
PasswordHash: hashedPassword,
Role:         domain.UserRole("reviewer"),
CreatedAt:    time.Now(),
UpdatedAt:    time.Now(),
}
err = userRepo.Create(ctx, reviewer)
require.NoError(t, err)

// Creator 1: Create 1 published test and 1 draft test
publishedTest := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator1.ID, "Creator1 Published Test", "Description 1")
_, err = publishTestUC.Execute(ctx, testUC.PublishTestRequest{TestID: publishedTest.ID})
require.NoError(t, err)

draftTest := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator1.ID, "Creator1 Draft Test", "Description 2")

// Creator 2: Create 1 published test
creator2Test := createTestWithQuestion(t, ctx, createTestUC, addQuestionUC, creator2.ID, "Creator2 Test", "Description 3")
_, err = publishTestUC.Execute(ctx, testUC.PublishTestRequest{TestID: creator2Test.ID})
require.NoError(t, err)

t.Run("Creator1_CanAccessOwnPublishedTest", func(t *testing.T) {
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   publishedTest.ID,
UserID:   creator1.ID,
UserRole: "creator",
})

require.NoError(t, err)
assert.NotNil(t, result)
assert.Equal(t, publishedTest.ID, result.ID)
assert.Equal(t, "Creator1 Published Test", result.Title)
assert.True(t, result.IsPublished)
})

t.Run("Creator1_CanAccessOwnDraftTest", func(t *testing.T) {
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   draftTest.ID,
UserID:   creator1.ID,
UserRole: "creator",
})

require.NoError(t, err)
assert.NotNil(t, result)
assert.Equal(t, draftTest.ID, result.ID)
assert.Equal(t, "Creator1 Draft Test", result.Title)
assert.False(t, result.IsPublished)
})

t.Run("Creator1_CannotAccessCreator2Test", func(t *testing.T) {
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   creator2Test.ID,
UserID:   creator1.ID,
UserRole: "creator",
})

require.Error(t, err)
assert.Nil(t, result)
var authErr domain.ErrUnauthorized
assert.ErrorAs(t, err, &authErr)
})

t.Run("Reviewer_CanAccessPublishedTest", func(t *testing.T) {
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   publishedTest.ID,
UserID:   reviewer.ID,
UserRole: "reviewer",
})

require.NoError(t, err)
assert.NotNil(t, result)
assert.Equal(t, publishedTest.ID, result.ID)
assert.True(t, result.IsPublished)
})

t.Run("Reviewer_CannotAccessDraftTest", func(t *testing.T) {
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   draftTest.ID,
UserID:   reviewer.ID,
UserRole: "reviewer",
})

require.Error(t, err)
assert.Nil(t, result)
var authErr domain.ErrUnauthorized
assert.ErrorAs(t, err, &authErr)
})

t.Run("TestNotFound", func(t *testing.T) {
nonExistentID := uuid.New()
result, err := getTestUC.Execute(ctx, testUC.GetTestRequest{
TestID:   nonExistentID,
UserID:   creator1.ID,
UserRole: "creator",
})

require.Error(t, err)
assert.Nil(t, result)
var notFoundErr domain.ErrNotFound
assert.ErrorAs(t, err, &notFoundErr)
})

// Cleanup
t.Run("Cleanup", func(t *testing.T) {
// Delete questions first
_, err := pool.Exec(ctx, "DELETE FROM questions WHERE test_id IN ($1, $2, $3)",
publishedTest.ID, draftTest.ID, creator2Test.ID)
require.NoError(t, err)

// Delete tests
_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id IN ($1, $2, $3)",
publishedTest.ID, draftTest.ID, creator2Test.ID)
require.NoError(t, err)

// Delete users
_, err = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2, $3)",
creator1.ID, creator2.ID, reviewer.ID)
require.NoError(t, err)
})
}
