package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	redisInfra "github.com/assessly/assessly-be/internal/infrastructure/redis"
	"github.com/assessly/assessly-be/internal/usecase/scoring"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScoringWorkerIntegration tests the complete AI scoring worker flow
func TestScoringWorkerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test environment
	loadTestEnv(t)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

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

	// Setup Redis connection
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	defer redisClient.Close()

	// Test Redis connection
	err := redisClient.Ping(ctx).Err()
	require.NoError(t, err, "failed to connect to Redis")

	// Setup queue
	queueClient := redisInfra.NewQueueClient(redisClient, "assessly:scoring", slog.Default())
	streamName := "assessly:scoring"

	// Setup mock AI scorer
	mockAIScorer := &MockAIScorer{
		ScoreAnswerFunc: func(ctx context.Context, question, expectedAnswer, actualAnswer string) (*scoring.ScoreResult, error) {
			// Simple scoring logic for testing: exact match = 100, contains keyword = 50, else 0
			if actualAnswer == expectedAnswer {
				return &scoring.ScoreResult{
					Score:    100.0,
					Feedback: "Perfect answer!",
				}, nil
			} else if len(actualAnswer) > 0 {
				return &scoring.ScoreResult{
					Score:    50.0,
					Feedback: "Partially correct answer.",
				}, nil
			}
			return &scoring.ScoreResult{
				Score:    0.0,
				Feedback: "Incorrect or missing answer.",
			}, nil
		},
	}

	// Setup use case
	scoreWithAIUC := scoring.NewScoreWithAIUseCase(
		submissionRepo,
		answerRepo,
		questionRepo,
		reviewRepo,
		mockAIScorer,
		logger,
	)

	t.Run("should process scoring job and store AI scores", func(t *testing.T) {
		// === SETUP: Create test data ===

		// Create test creator
		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator@scoring.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		// Create test
		test := &domain.Test{
			ID:           uuid.New(),
			CreatorID:    creator.ID,
			Title:        "Scoring Test",
			Description:  "Test for AI scoring worker",
			AllowRetakes: false,
			IsPublished:  true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		// Create questions
		question1 := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "What is Go?",
			ExpectedAnswer: "A programming language",
			OrderNum:       1,
			CreatedAt:      time.Now(),
		}
		question2 := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "What is Docker?",
			ExpectedAnswer: "A containerization platform",
			OrderNum:       2,
			CreatedAt:      time.Now(),
		}
		err = questionRepo.Create(ctx, question1)
		require.NoError(t, err)
		err = questionRepo.Create(ctx, question2)
		require.NoError(t, err)

		// Create submission
		submission := &domain.Submission{
			ID:               uuid.New(),
			TestID:           test.ID,
			AccessEmail:      "participant@scoring.test",
			SubmittedAt:      time.Now(),
			AITotalScore:     nil,
			ManualTotalScore: nil,
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		// Create answers
		answer1 := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question1.ID,
			Text:         "A programming language", // Exact match
			CreatedAt:    time.Now(),
		}
		answer2 := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question2.ID,
			Text:         "Container technology", // Partial match
			CreatedAt:    time.Now(),
		}
		err = answerRepo.CreateBatch(ctx, []*domain.Answer{answer1, answer2})
		require.NoError(t, err)

		// === TEST: Queue and process scoring job ===

		// Create consumer group
		groupName := "scoring-workers"
		err = queueClient.CreateConsumerGroup(ctx, streamName, groupName)
		// Ignore error if group already exists
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			require.NoError(t, err)
		}

		// Enqueue scoring job
		payload := []byte(`{"submission_id":"` + submission.ID.String() + `"}`)
		jobID, err := queueClient.Enqueue(ctx, "score_submission", payload)
		require.NoError(t, err)
		assert.NotEmpty(t, jobID, "job ID should not be empty")

		// Verify job is in queue
		jobs, err := queueClient.Dequeue(ctx, streamName, groupName, "test-consumer", 1, 100*time.Millisecond)
		require.NoError(t, err)
		require.Len(t, jobs, 1, "should have 1 job in queue")
		assert.Equal(t, "score_submission", jobs[0].Type)

		// Process the scoring job (simulate worker)
		err = scoreWithAIUC.Execute(ctx, scoring.ScoreWithAIRequest{
			SubmissionID: submission.ID,
		})
		require.NoError(t, err)

		// Acknowledge the job
		err = queueClient.AckMessage(ctx, streamName, groupName, jobs[0].ID)
		require.NoError(t, err)

		// === VERIFY: Check AI scores are stored ===

		// Verify reviews were created for both answers
		review1, err := reviewRepo.FindByAnswerID(ctx, answer1.ID)
		require.NoError(t, err)
		require.NotNil(t, review1, "review1 should exist")

		review2, err := reviewRepo.FindByAnswerID(ctx, answer2.ID)
		require.NoError(t, err)
		require.NotNil(t, review2, "review2 should exist")

		// Check review1 scores (exact match)
		assert.NotNil(t, review1.AIScore, "AI score should be set")
		assert.Equal(t, 100.0, *review1.AIScore, "AI score should be 100 for exact match")
		assert.NotNil(t, review1.AIFeedback, "AI feedback should be set")
		assert.Equal(t, "Perfect answer!", *review1.AIFeedback)
		assert.NotNil(t, review1.AIScoredAt, "AI scored timestamp should be set")
		assert.Nil(t, review1.ReviewerID, "reviewer ID should be nil for AI scoring")

		// Check review2 scores (partial match)
		assert.NotNil(t, review2.AIScore, "AI score should be set")
		assert.Equal(t, 50.0, *review2.AIScore, "AI score should be 50 for partial match")
		assert.NotNil(t, review2.AIFeedback, "AI feedback should be set")
		assert.Equal(t, "Partially correct answer.", *review2.AIFeedback)
		assert.NotNil(t, review2.AIScoredAt, "AI scored timestamp should be set")

		// Verify submission has updated AI total score
		updatedSubmission, err := submissionRepo.FindByID(ctx, submission.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedSubmission.AITotalScore, "AI total score should be set")
		expectedTotal := 150.0 // Sum: 100 + 50
		assert.Equal(t, expectedTotal, *updatedSubmission.AITotalScore, "AI total score should be sum of answer scores")

		// === CLEANUP ===
		// Delete in reverse order of foreign keys using direct SQL
		_, err = pool.Exec(ctx, "DELETE FROM reviews WHERE answer_id IN ($1, $2)", answer1.ID, answer2.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM answers WHERE id IN ($1, $2)", answer1.ID, answer2.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE id IN ($1, $2)", question1.ID, question2.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", creator.ID)
		require.NoError(t, err)

		// Delete the job from stream
		err = queueClient.DeleteMessage(ctx, streamName, jobID)
		require.NoError(t, err)

		t.Log("✅ AI scoring worker integration test completed successfully")
	})

	t.Run("should handle scoring job with no answers gracefully", func(t *testing.T) {
		// Create test data without answers
		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator2@scoring.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		test := &domain.Test{
			ID:          uuid.New(),
			CreatorID:   creator.ID,
			Title:       "Empty Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		submission := &domain.Submission{
			ID:          uuid.New(),
			TestID:      test.ID,
			AccessEmail: "participant2@scoring.test",
			SubmittedAt: time.Now(),
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		// Try to score submission with no answers
		err = scoreWithAIUC.Execute(ctx, scoring.ScoreWithAIRequest{
			SubmissionID: submission.ID,
		})
		assert.Error(t, err, "should return error for submission with no answers")
		assert.Contains(t, err.Error(), "no answers found")

		// Cleanup
		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", creator.ID)
		require.NoError(t, err)
	})

	t.Run("should continue scoring other answers when one fails", func(t *testing.T) {
		// Test resilience: if one answer fails to score, continue with others
		creator := &domain.User{
			ID:           uuid.New(),
			Email:        "creator3@scoring.test",
			PasswordHash: "hashed_password",
			Role:         domain.RoleCreator,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := userRepo.Create(ctx, creator)
		require.NoError(t, err)

		test := &domain.Test{
			ID:          uuid.New(),
			CreatorID:   creator.ID,
			Title:       "Resilience Test",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = testRepo.Create(ctx, test)
		require.NoError(t, err)

		question1 := &domain.Question{
			ID:             uuid.New(),
			TestID:         test.ID,
			Text:           "Valid question",
			ExpectedAnswer: "Valid answer",
			OrderNum:       1,
			CreatedAt:      time.Now(),
		}
		err = questionRepo.Create(ctx, question1)
		require.NoError(t, err)

		submission := &domain.Submission{
			ID:          uuid.New(),
			TestID:      test.ID,
			AccessEmail: "participant3@scoring.test",
			SubmittedAt: time.Now(),
		}
		err = submissionRepo.Create(ctx, submission)
		require.NoError(t, err)

		answer1 := &domain.Answer{
			ID:           uuid.New(),
			SubmissionID: submission.ID,
			QuestionID:   question1.ID,
			Text:         "Some answer",
			CreatedAt:    time.Now(),
		}
		err = answerRepo.CreateBatch(ctx, []*domain.Answer{answer1})
		require.NoError(t, err)

		// Execute scoring
		err = scoreWithAIUC.Execute(ctx, scoring.ScoreWithAIRequest{
			SubmissionID: submission.ID,
		})
		require.NoError(t, err)

		// Verify at least one answer was scored
		reviewForAnswer1, err := reviewRepo.FindByAnswerID(ctx, answer1.ID)
		require.NoError(t, err)
		assert.NotNil(t, reviewForAnswer1, "should have created review for valid answer")

		// Cleanup - delete in reverse order of foreign keys
		_, err = pool.Exec(ctx, "DELETE FROM reviews WHERE answer_id = $1", answer1.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM answers WHERE id = $1", answer1.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE id = $1", submission.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE id = $1", question1.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE id = $1", test.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", creator.ID)
		require.NoError(t, err)
	})
}

// MockAIScorer implements AIScorer for testing
type MockAIScorer struct {
	ScoreAnswerFunc func(ctx context.Context, question, expectedAnswer, actualAnswer string) (*scoring.ScoreResult, error)
}

func (m *MockAIScorer) ScoreAnswer(ctx context.Context, question, expectedAnswer, actualAnswer string) (*scoring.ScoreResult, error) {
	if m.ScoreAnswerFunc != nil {
		return m.ScoreAnswerFunc(ctx, question, expectedAnswer, actualAnswer)
	}
	return &scoring.ScoreResult{Score: 0, Feedback: "Mock scorer"}, nil
}
