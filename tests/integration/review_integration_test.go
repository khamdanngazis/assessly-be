package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualReviewFlow tests the manual review functionality
// Reviewers can view AI-scored submissions and add/override scores
func TestManualReviewFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test environment
	loadTestEnv(t)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_ = logger // Suppress unused warning

	// Setup database connection
	pool := setupTestDatabase(t, ctx)
	defer pool.Close()

	// Setup repositories
	userRepo := postgres.NewUserRepository(pool)
	testRepo := postgres.NewTestRepository(pool)
	questionRepo := postgres.NewQuestionRepository(pool)
	submissionRepo := postgres.NewSubmissionRepository(pool)
	answerRepo := postgres.NewAnswerRepository(pool)
	reviewRepo := postgres.NewReviewRepository(pool)

	t.Run("should allow reviewer to add manual score to AI-scored answer", func(t *testing.T) {
		// === SETUP: Create test data with AI scoring ===

		// Create test creator
		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		// Create reviewer
		reviewer := &domain.User{
			ID:           uuid.New(),
			Email:        "reviewer@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleReviewer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = userRepo.Create(ctx, reviewer)
		require.NoError(t, err)

		// Create test
		test := &domain.Test{
			ID:          uuid.New(),
			CreatorID:   creator.ID,
			Title:       "Manual Review Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		// Create question
		question := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "What is Kubernetes?",
			ExpectedAnswer: "Container orchestration platform",
			OrderNum:       1,
			CreatedAt:      time.Now(),
		}
		err = questionRepo.Create(ctx, question)
		require.NoError(t, err)

		// Create submission
		submission := &domain.Submission{
			ID:          uuid.New(),
			TestID:      test.ID,
			AccessEmail: "participant@review.test",
			SubmittedAt: time.Now(),
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		// Create answer
		answer := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question.ID,
			Text:         "A tool for managing containerized applications",
			CreatedAt:    time.Now(),
		}
		err = answerRepo.CreateBatch(ctx, []*domain.Answer{answer})
		require.NoError(t, err)

		// Simulate AI scoring by directly using repository method
		aiScore := 70.0
		aiFeedback := "Good answer but missing some key concepts"
		err = reviewRepo.UpsertAIScore(ctx, answer.ID, aiScore, aiFeedback)
		require.NoError(t, err)

		// === TEST: Reviewer retrieves submission and adds manual score ===

		// Verify AI score was stored
		reviewBeforeManual, err := reviewRepo.FindByAnswerID(ctx, answer.ID)
		require.NoError(t, err)
		require.NotNil(t, reviewBeforeManual)
		assert.NotNil(t, reviewBeforeManual.AIScore, "AI score should be set")
		assert.Equal(t, aiScore, *reviewBeforeManual.AIScore)
		assert.Nil(t, reviewBeforeManual.ManualScore, "Manual score should not be set yet")
		assert.Nil(t, reviewBeforeManual.ReviewerID, "Reviewer ID should be nil for AI-only scoring")

		// Reviewer adds manual score (override AI score)
		manualScore := 85.0
		manualFeedback := "Excellent understanding. Added technical depth needed."
		err = reviewRepo.UpsertManualScore(ctx, answer.ID, reviewer.ID, manualScore, manualFeedback)
		require.NoError(t, err)

		// === VERIFICATION ===

		// Verify manual score was added
		reviewAfterManual, err := reviewRepo.FindByAnswerID(ctx, answer.ID)
		require.NoError(t, err)
		require.NotNil(t, reviewAfterManual)

		// Check AI score is still present
		assert.NotNil(t, reviewAfterManual.AIScore, "AI score should still exist")
		assert.Equal(t, aiScore, *reviewAfterManual.AIScore)

		// Check manual score is now set
		assert.NotNil(t, reviewAfterManual.ManualScore, "Manual score should be set")
		assert.Equal(t, manualScore, *reviewAfterManual.ManualScore)
		assert.NotNil(t, reviewAfterManual.ManualFeedback, "Manual feedback should be set")
		assert.Equal(t, manualFeedback, *reviewAfterManual.ManualFeedback)

		// Check reviewer ID is recorded
		assert.NotNil(t, reviewAfterManual.ReviewerID, "Reviewer ID should be set")
		assert.Equal(t, reviewer.ID, *reviewAfterManual.ReviewerID)

		// Check timestamps
		assert.NotNil(t, reviewAfterManual.AIScoredAt, "AI scored timestamp should be set")
		assert.NotNil(t, reviewAfterManual.ManualScoredAt, "Manual scored timestamp should be set")

		// === CLEANUP ===
		_, err = pool.Exec(ctx, "DELETE FROM reviews WHERE answer_id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM answers WHERE id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE id = $1", question.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", creator.ID, reviewer.ID)
		require.NoError(t, err)

		t.Log("✅ Manual review flow test completed successfully")
	})

	t.Run("should allow reviewer to update existing manual score", func(t *testing.T) {
		// === SETUP: Create test data ===

		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator2@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		reviewer := &domain.User{
			ID:           uuid.New(),
			Email:        "reviewer2@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleReviewer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = userRepo.Create(ctx, reviewer)
		require.NoError(t, err)

		test := &domain.Test{
			ID:          uuid.New(),
			CreatorID:   creator.ID,
			Title:       "Review Update Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		question := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "Explain Docker",
			ExpectedAnswer: "Containerization platform",
			OrderNum:       1,
			CreatedAt:      time.Now(),
		}
		err = questionRepo.Create(ctx, question)
		require.NoError(t, err)

		submission := &domain.Submission{
			ID:          uuid.New(),
			TestID:      test.ID,
			AccessEmail: "participant2@review.test",
			SubmittedAt: time.Now(),
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		answer := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question.ID,
			Text:         "Platform for running containers",
			CreatedAt:    time.Now(),
		}
		err = answerRepo.CreateBatch(ctx, []*domain.Answer{answer})
		require.NoError(t, err)

		// Add initial manual score
		initialScore := 60.0
		initialFeedback := "Needs more detail"
		err = reviewRepo.UpsertManualScore(ctx, answer.ID, reviewer.ID, initialScore, initialFeedback)
		require.NoError(t, err)

		// Verify initial score
		reviewBefore, err := reviewRepo.FindByAnswerID(ctx, answer.ID)
		require.NoError(t, err)
		require.NotNil(t, reviewBefore.ManualScore)
		assert.Equal(t, initialScore, *reviewBefore.ManualScore)

		// === TEST: Update manual score ===

		updatedScore := 75.0
		updatedFeedback := "Improved answer with additional context"
		err = reviewRepo.UpsertManualScore(ctx, answer.ID, reviewer.ID, updatedScore, updatedFeedback)
		require.NoError(t, err)

		// === VERIFICATION ===

		reviewAfter, err := reviewRepo.FindByAnswerID(ctx, answer.ID)
		require.NoError(t, err)
		require.NotNil(t, reviewAfter)

		// Verify score was updated
		assert.NotNil(t, reviewAfter.ManualScore, "Manual score should be set")
		assert.Equal(t, updatedScore, *reviewAfter.ManualScore, "Score should be updated")
		assert.NotNil(t, reviewAfter.ManualFeedback)
		assert.Equal(t, updatedFeedback, *reviewAfter.ManualFeedback, "Feedback should be updated")
		assert.Equal(t, reviewer.ID, *reviewAfter.ReviewerID, "Reviewer ID should remain same")

		// === CLEANUP ===
		_, err = pool.Exec(ctx, "DELETE FROM reviews WHERE answer_id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM answers WHERE id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE id = $1", question.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", creator.ID, reviewer.ID)
		require.NoError(t, err)

		t.Log("✅ Manual review update test completed successfully")
	})

	t.Run("should allow reviewer to add manual score without prior AI score", func(t *testing.T) {
		// === SETUP: Create test data without AI scoring ===

		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator3@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		reviewer := &domain.User{
			ID:           uuid.New(),
			Email:        "reviewer3@review.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleReviewer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = userRepo.Create(ctx, reviewer)
		require.NoError(t, err)

		test := &domain.Test{
			ID:          uuid.New(),
			CreatorID:   creator.ID,
			Title:       "Manual-Only Review Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		question := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "What is microservices?",
			ExpectedAnswer: "Architectural pattern for distributed systems",
			OrderNum:       1,
			CreatedAt:      time.Now(),
		}
		err = questionRepo.Create(ctx, question)
		require.NoError(t, err)

		submission := &domain.Submission{
			ID:          uuid.New(),
			TestID:      test.ID,
			AccessEmail: "participant3@review.test",
			SubmittedAt: time.Now(),
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		answer := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question.ID,
			Text:         "Breaking down applications into smaller services",
			CreatedAt:    time.Now(),
		}
		err = answerRepo.CreateBatch(ctx, []*domain.Answer{answer})
		require.NoError(t, err)

		// === TEST: Add manual score directly (no AI scoring) ===

		manualScore := 90.0
		manualFeedback := "Excellent concise explanation"
		err = reviewRepo.UpsertManualScore(ctx, answer.ID, reviewer.ID, manualScore, manualFeedback)
		require.NoError(t, err)

		// === VERIFICATION ===

		review, err := reviewRepo.FindByAnswerID(ctx, answer.ID)
		require.NoError(t, err)
		require.NotNil(t, review)

		// Check no AI score
		assert.Nil(t, review.AIScore, "AI score should be nil")
		assert.Nil(t, review.AIFeedback, "AI feedback should be nil")
		assert.Nil(t, review.AIScoredAt, "AI scored timestamp should be nil")

		// Check manual score is set
		assert.NotNil(t, review.ManualScore, "Manual score should be set")
		assert.Equal(t, manualScore, *review.ManualScore)
		assert.NotNil(t, review.ManualFeedback)
		assert.Equal(t, manualFeedback, *review.ManualFeedback)
		assert.NotNil(t, review.ReviewerID)
		assert.Equal(t, reviewer.ID, *review.ReviewerID)
		assert.NotNil(t, review.ManualScoredAt)

		// === CLEANUP ===
		_, err = pool.Exec(ctx, "DELETE FROM reviews WHERE answer_id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM answers WHERE id = $1", answer.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE id = $1", question.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", creator.ID, reviewer.ID)
		require.NoError(t, err)

		t.Log("✅ Manual-only review test completed successfully")
	})
}
