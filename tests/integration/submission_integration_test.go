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
	submissionUC "github.com/assessly/assessly-be/internal/usecase/submission"
	testUC "github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubmission_GenerateTokenSubmitRetrieve tests the complete submission flow:
// 1. Creator creates and publishes a test with questions
// 2. Generate access token for participant
// 3. Participant submits test answers
// 4. Retrieve submission (by participant and creator)
func TestSubmission_GenerateTokenSubmitRetrieve(t *testing.T) {
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
	submissionRepo := postgres.NewSubmissionRepository(pool)
	answerRepo := postgres.NewAnswerRepository(pool)
	reviewRepo := postgres.NewReviewRepository(pool)

	// Setup JWT service for access tokens
	jwtSecret := getEnv("JWT_SECRET", "test-secret-key")
	jwtIssuer := getEnv("JWT_ISSUER", "assessly-test")
	jwtService := auth.NewJWTService(jwtSecret, jwtIssuer, 24)

	// Mock services
	emailSender := &mockSubmissionEmailSender{}
	accessTokenGen := &mockAccessTokenGenerator{jwtService: jwtService}
	tokenValidator := &mockSubmissionTokenValidator{jwtService: jwtService}
	scoringQueue := &mockScoringQueue{}

	// Create use cases
	createTestUC := testUC.NewCreateTestUseCase(testRepo, logger)
	addQuestionUC := testUC.NewAddQuestionUseCase(questionRepo, testRepo, logger)
	publishTestUC := testUC.NewPublishTestUseCase(testRepo, questionRepo, logger)
	generateTokenUC := submissionUC.NewGenerateAccessTokenUseCase(testRepo, accessTokenGen, emailSender, logger)
	submitTestUC := submissionUC.NewSubmitTestUseCase(testRepo, questionRepo, submissionRepo, answerRepo, tokenValidator, scoringQueue, logger)
	getSubmissionUC := submissionUC.NewGetSubmissionUseCase(submissionRepo, answerRepo, reviewRepo, testRepo, tokenValidator, logger)

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
	var question1ID, question2ID uuid.UUID
	participantEmail := fmt.Sprintf("participant-%d@example.com", time.Now().Unix())

	// Step 1: Create and publish a test
	t.Run("CreateAndPublishTest", func(t *testing.T) {
		// Create test
		createReq := testUC.CreateTestRequest{
			CreatorID:    creator.ID,
			Title:        "Integration Test - Submission Flow",
			Description:  "Test for submission integration",
			AllowRetakes: false,
		}
		test, err := createTestUC.Execute(ctx, createReq)
		require.NoError(t, err)
		testID = test.ID

		// Add questions
		q1Req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "What is integration testing?",
			ExpectedAnswer: "Integration testing is testing how different modules work together.",
			OrderNum:       0,
		}
		q1, err := addQuestionUC.Execute(ctx, q1Req)
		require.NoError(t, err)
		question1ID = q1.ID

		q2Req := testUC.AddQuestionRequest{
			TestID:         testID,
			Text:           "Why is testing important?",
			ExpectedAnswer: "Testing ensures code quality and prevents bugs.",
			OrderNum:       0,
		}
		q2, err := addQuestionUC.Execute(ctx, q2Req)
		require.NoError(t, err)
		question2ID = q2.ID

		// Publish test
		pubReq := testUC.PublishTestRequest{
			TestID: testID,
		}
		_, err = publishTestUC.Execute(ctx, pubReq)
		require.NoError(t, err)
	})

	// Step 2: Generate access token for participant
	var accessToken string
	t.Run("GenerateAccessToken", func(t *testing.T) {
		req := submissionUC.GenerateAccessTokenRequest{
			TestID:      testID,
			Email:       participantEmail,
			AccessURL:   "https://assessly.com/take-test",
			ExpiryHours: 24,
		}

		err := generateTokenUC.Execute(ctx, req)
		require.NoError(t, err)

		// Verify email was "sent"
		assert.Equal(t, 1, emailSender.CallCount)
		assert.Equal(t, participantEmail, emailSender.LastRecipient)
		accessToken = emailSender.LastAccessToken
		assert.NotEmpty(t, accessToken)
	})

	// Step 3: Try to generate token for unpublished test (should fail)
	t.Run("GenerateToken_UnpublishedTest_ShouldFail", func(t *testing.T) {
		// Create unpublished test
		createReq := testUC.CreateTestRequest{
			CreatorID:   creator.ID,
			Title:       "Unpublished Test",
			Description: "This test is not published",
		}
		unpublishedTest, err := createTestUC.Execute(ctx, createReq)
		require.NoError(t, err)

		// Try to generate token
		req := submissionUC.GenerateAccessTokenRequest{
			TestID:    unpublishedTest.ID,
			Email:     "someone@example.com",
			AccessURL: "https://assessly.com/test",
		}

		err = generateTokenUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not published")
	})

	// Step 4: Participant submits test
	var submissionID uuid.UUID
	t.Run("SubmitTest", func(t *testing.T) {
		req := submissionUC.SubmitTestRequest{
			AccessToken: accessToken,
			Answers: []submissionUC.AnswerInput{
				{
					QuestionID: question1ID,
					Text:       "Integration testing verifies that different components work together correctly.",
				},
				{
					QuestionID: question2ID,
					Text:       "Testing is important to catch bugs early and ensure software quality.",
				},
			},
		}

		submission, err := submitTestUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, submission)
		assert.Equal(t, testID, submission.TestID)
		assert.Equal(t, participantEmail, submission.AccessEmail)
		assert.NotZero(t, submission.SubmittedAt)
		submissionID = submission.ID

		// Verify scoring was queued
		assert.Equal(t, 1, scoringQueue.CallCount)
		assert.Equal(t, submissionID, scoringQueue.LastSubmissionID)
	})

	// Step 5: Verify answers were saved
	t.Run("VerifyAnswersSaved", func(t *testing.T) {
		answers, err := answerRepo.FindBySubmissionID(ctx, submissionID)
		require.NoError(t, err)
		assert.Len(t, answers, 2)

		// Verify answer texts
		answerTexts := make(map[uuid.UUID]string)
		for _, ans := range answers {
			answerTexts[ans.QuestionID] = ans.Text
		}
		assert.Contains(t, answerTexts[question1ID], "Integration testing")
		assert.Contains(t, answerTexts[question2ID], "Testing is important")
	})

	// Step 6: Participant retrieves their submission
	t.Run("GetSubmission_ByParticipant", func(t *testing.T) {
		req := submissionUC.GetSubmissionRequest{
			SubmissionID: submissionID,
			AccessToken:  accessToken,
		}

		resp, err := getSubmissionUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, submissionID, resp.Submission.ID)
		assert.Len(t, resp.Answers, 2)
	})

	// Step 7: Creator retrieves submission
	t.Run("GetSubmission_ByCreator", func(t *testing.T) {
		req := submissionUC.GetSubmissionRequest{
			SubmissionID: submissionID,
			UserID:       &creator.ID,
			UserRole:     "creator",
		}

		resp, err := getSubmissionUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, submissionID, resp.Submission.ID)
		assert.Equal(t, participantEmail, resp.Submission.AccessEmail)
	})

	// Step 8: Unauthorized access should fail
	t.Run("GetSubmission_Unauthorized_ShouldFail", func(t *testing.T) {
		// Try without token or user ID
		req := submissionUC.GetSubmissionRequest{
			SubmissionID: submissionID,
		}

		resp, err := getSubmissionUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "not authorized")
	})

	// Step 9: Submit with invalid token should fail
	t.Run("SubmitTest_InvalidToken_ShouldFail", func(t *testing.T) {
		req := submissionUC.SubmitTestRequest{
			AccessToken: "invalid-token",
			Answers: []submissionUC.AnswerInput{
				{QuestionID: question1ID, Text: "Answer"},
			},
		}

		submission, err := submitTestUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, submission)
	})

	// Step 10: Test retake policy
	t.Run("SubmitTest_RetakeNotAllowed_ShouldFail", func(t *testing.T) {
		// Try to submit again (retakes not allowed)
		req := submissionUC.SubmitTestRequest{
			AccessToken: accessToken,
			Answers: []submissionUC.AnswerInput{
				{QuestionID: question1ID, Text: "Second attempt"},
			},
		}

		submission, err := submitTestUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "retakes not allowed")
	})

	// Step 11: Test with retakes allowed
	t.Run("SubmitTest_RetakeAllowed", func(t *testing.T) {
		// Create new test with retakes allowed
		createReq := testUC.CreateTestRequest{
			CreatorID:    creator.ID,
			Title:        "Test with Retakes",
			Description:  "This test allows retakes",
			AllowRetakes: true,
		}
		retakeTest, err := createTestUC.Execute(ctx, createReq)
		require.NoError(t, err)

		// Add question
		qReq := testUC.AddQuestionRequest{
			TestID:         retakeTest.ID,
			Text:           "Question for retake test",
			ExpectedAnswer: "Answer",
			OrderNum:       0,
		}
		q, err := addQuestionUC.Execute(ctx, qReq)
		require.NoError(t, err)

		// Publish
		pubReq := testUC.PublishTestRequest{
			TestID: retakeTest.ID,
		}
		_, err = publishTestUC.Execute(ctx, pubReq)
		require.NoError(t, err)

		// Generate token
		tokenReq := submissionUC.GenerateAccessTokenRequest{
			TestID:    retakeTest.ID,
			Email:     "retake-test@example.com",
			AccessURL: "https://assessly.com/test",
		}
		err = generateTokenUC.Execute(ctx, tokenReq)
		require.NoError(t, err)
		retakeToken := emailSender.LastAccessToken

		// Submit first time
		submitReq := submissionUC.SubmitTestRequest{
			AccessToken: retakeToken,
			Answers: []submissionUC.AnswerInput{
				{QuestionID: q.ID, Text: "First attempt"},
			},
		}
		_, err = submitTestUC.Execute(ctx, submitReq)
		require.NoError(t, err)

		// Submit second time (should succeed)
		submitReq2 := submissionUC.SubmitTestRequest{
			AccessToken: retakeToken,
			Answers: []submissionUC.AnswerInput{
				{QuestionID: q.ID, Text: "Second attempt"},
			},
		}
		submission2, err := submitTestUC.Execute(ctx, submitReq2)
		require.NoError(t, err)
		assert.NotNil(t, submission2)
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Delete in reverse order of foreign keys
		_, err := pool.Exec(ctx, "DELETE FROM answers WHERE submission_id IN (SELECT id FROM submissions WHERE test_id IN (SELECT id FROM tests WHERE creator_id = $1))", creator.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM submissions WHERE test_id IN (SELECT id FROM tests WHERE creator_id = $1)", creator.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM questions WHERE test_id IN (SELECT id FROM tests WHERE creator_id = $1)", creator.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM tests WHERE creator_id = $1", creator.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", creator.ID)
		require.NoError(t, err)
	})
}

// Mock implementations

type mockSubmissionEmailSender struct {
	CallCount       int
	LastRecipient   string
	LastAccessToken string
}

func (m *mockSubmissionEmailSender) SendPasswordReset(to, token, resetURL string) error {
	return nil
}

func (m *mockSubmissionEmailSender) SendTestAccessToken(to, testTitle, accessToken, accessURL string) error {
	m.CallCount++
	m.LastRecipient = to
	m.LastAccessToken = accessToken
	return nil
}

type mockAccessTokenGenerator struct {
	jwtService *auth.JWTService
}

func (m *mockAccessTokenGenerator) GenerateAccessToken(testID uuid.UUID, email string, expiryHours int) (string, error) {
	// Generate JWT with role "participant"
	return m.jwtService.GenerateToken(testID, email, "participant")
}

type mockSubmissionTokenValidator struct {
	jwtService *auth.JWTService
}

func (m *mockSubmissionTokenValidator) ValidateToken(tokenString string) (string, string, string, error) {
	claims, err := m.jwtService.ValidateToken(tokenString)
	if err != nil {
		return "", "", "", domain.ErrUnauthorized{Message: "invalid token"}
	}
	// For participant tokens, UserID field contains testID
	return claims.UserID, claims.Email, claims.Role, nil
}

type mockScoringQueue struct {
	CallCount        int
	LastSubmissionID uuid.UUID
}

func (m *mockScoringQueue) Enqueue(ctx context.Context, submissionID uuid.UUID) error {
	m.CallCount++
	m.LastSubmissionID = submissionID
	return nil
}
