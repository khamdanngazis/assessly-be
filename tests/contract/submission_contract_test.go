package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/delivery/http/handler"
	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/assessly/assessly-be/internal/domain"
	submissionUC "github.com/assessly/assessly-be/internal/usecase/submission"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateAccessTokenContract validates POST /api/v1/submissions/access
func TestGenerateAccessTokenContract(t *testing.T) {
	// Setup mocks
	mockTestRepo := &MockTestRepositoryForSubmission{}
	mockTokenGen := &MockAccessTokenGenerator{}
	mockEmailSender := &MockEmailSender{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	generateAccessTokenUC := submissionUC.NewGenerateAccessTokenUseCase(mockTestRepo, mockTokenGen, mockEmailSender, logger)
	submissionHandler := handler.NewSubmissionHandler(generateAccessTokenUC, nil, nil, logger)

	t.Run("should return 200 with message on successful token generation", func(t *testing.T) {
		testID := uuid.New()
		email := "participant@example.com"

		// Prepare mock responses
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				Title:       "Sample Test",
				IsPublished: true,
				CreatedAt:   time.Now(),
			}, nil
		}
		mockTokenGen.GenerateAccessTokenFunc = func(testID uuid.UUID, email string, expiryHours int) (string, error) {
			return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token", nil
		}
		mockEmailSender.SendTestAccessTokenFunc = func(to, testTitle, accessToken, accessURL string) error {
			return nil
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"test_id":    testID.String(),
			"email":      email,
			"access_url": "https://example.com/take-test",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/access", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		submissionHandler.GenerateAccessToken(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate message field
		assert.Contains(t, resp, "message", "response should contain message field")
		assert.IsType(t, "", resp["message"], "message should be string")
	})

	t.Run("should return 400 on invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/access", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		submissionHandler.GenerateAccessToken(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 404 when test not found", func(t *testing.T) {
		testID := uuid.New()
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return nil, domain.ErrNotFound{Resource: "test", ID: id.String()}
		}

		reqBody := map[string]interface{}{
			"test_id":    testID.String(),
			"email":      "participant@example.com",
			"access_url": "https://example.com/take-test",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/access", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		submissionHandler.GenerateAccessToken(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "should return 404 Not Found")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when test not published", func(t *testing.T) {
		testID := uuid.New()
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				Title:       "Unpublished Test",
				IsPublished: false,
				CreatedAt:   time.Now(),
			}, nil
		}

		reqBody := map[string]interface{}{
			"test_id":    testID.String(),
			"email":      "participant@example.com",
			"access_url": "https://example.com/take-test",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/access", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		submissionHandler.GenerateAccessToken(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestGetSubmissionContract validates GET /api/v1/submissions/:id
func TestGetSubmissionContract(t *testing.T) {
	// Setup mocks
	mockSubmissionRepo := &MockSubmissionRepository{}
	mockAnswerRepo := &MockAnswerRepository{}
	mockReviewRepo := &MockReviewRepository{}
	mockTestRepo := &MockTestRepositoryForSubmission{}
	mockTokenValidator := &MockTokenValidator{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	getSubmissionUC := submissionUC.NewGetSubmissionUseCase(mockSubmissionRepo, mockAnswerRepo, mockReviewRepo, mockTestRepo, mockTokenValidator, logger)
	submissionHandler := handler.NewSubmissionHandler(nil, nil, getSubmissionUC, logger)

	t.Run("should return 200 with submission details and scores", func(t *testing.T) {
		submissionID := uuid.New()
		testID := uuid.New()
		answerID := uuid.New()
		questionID := uuid.New()
		aiScore := 85.5
		manualScore := 90.0

		// Prepare mock responses
		mockSubmissionRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
			return &domain.Submission{
				ID:               submissionID,
				TestID:           testID,
				AccessEmail:      "participant@example.com",
				SubmittedAt:      time.Now().Add(-1 * time.Hour),
				AITotalScore:     &aiScore,
				ManualTotalScore: &manualScore,
			}, nil
		}

		mockAnswerRepo.FindBySubmissionIDFunc = func(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
			return []*domain.Answer{
				{
					ID:           answerID,
					SubmissionID: submissionID,
					QuestionID:   questionID,
					Text:         "My answer text",
					CreatedAt:    time.Now().Add(-1 * time.Hour),
				},
			}, nil
		}

		mockReviewRepo.FindByAnswerIDFunc = func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
			return &domain.Review{
				ID:             uuid.New(),
				AnswerID:       answerID,
				AIScore:        &aiScore,
				AIFeedback:     stringPtr("Good answer"),
				ManualScore:    &manualScore,
				ManualFeedback: stringPtr("Excellent work"),
			}, nil
		}

		// Prepare request with URL parameter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+submissionID.String(), nil)
		
		// Setup chi URL params and add user_id for authorization
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", submissionID.String())
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		// Add user_id and user_role to context (simulate creator access)
		creatorID := uuid.New()
		ctx = context.WithValue(ctx, middleware.UserIDKey, creatorID.String())
		ctx = context.WithValue(ctx, middleware.UserRoleKey, "creator")
		req = req.WithContext(ctx)
		
		// Update mockTestRepo to return test with matching creator
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				Title:       "Sample Test",
				CreatorID:   creatorID,
				IsPublished: true,
				CreatedAt:   time.Now(),
			}, nil
		}
		
		w := httptest.NewRecorder()

		// Execute
		submissionHandler.GetSubmission(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate submission object
		submissionObj, ok := resp["submission"].(map[string]interface{})
		require.True(t, ok, "response should contain submission object")
		
		assert.Contains(t, submissionObj, "id", "submission should contain id")
		assert.Contains(t, submissionObj, "test_id", "submission should contain test_id")
		assert.Contains(t, submissionObj, "access_email", "submission should contain access_email")
		assert.Contains(t, submissionObj, "submitted_at", "submission should contain submitted_at")
		assert.Contains(t, submissionObj, "ai_total_score", "submission should contain ai_total_score")
		assert.Contains(t, submissionObj, "manual_total_score", "submission should contain manual_total_score")
		
		// Validate display_score (manual overrides AI)
		assert.Contains(t, submissionObj, "display_score", "submission should contain display_score")
		assert.Equal(t, manualScore, submissionObj["display_score"], "display_score should equal manual score")

		// Validate answers array
		answers, ok := resp["answers"].([]interface{})
		require.True(t, ok, "response should contain answers array")
		assert.NotEmpty(t, answers, "answers should not be empty")
	})

	t.Run("should return 400 on invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/invalid-uuid", nil)
		
		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		submissionHandler.GetSubmission(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 404 when submission not found", func(t *testing.T) {
		submissionID := uuid.New()
		mockSubmissionRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
			return nil, domain.ErrNotFound{Resource: "submission", ID: id.String()}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+submissionID.String(), nil)
		
		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", submissionID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		submissionHandler.GetSubmission(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "should return 404 Not Found")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// Mock implementations for submission contract tests

type MockTestRepositoryForSubmission struct {
	FindByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Test, error)
}

func (m *MockTestRepositoryForSubmission) FindByID(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTestRepositoryForSubmission) Create(ctx context.Context, test *domain.Test) error {
	return nil
}

func (m *MockTestRepositoryForSubmission) Update(ctx context.Context, test *domain.Test) error {
	return nil
}

func (m *MockTestRepositoryForSubmission) FindByCreatorID(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*domain.Test, error) {
	return nil, nil
}

func (m *MockTestRepositoryForSubmission) FindPublished(ctx context.Context, limit, offset int) ([]*domain.Test, error) {
	return nil, nil
}

func (m *MockTestRepositoryForSubmission) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type MockAccessTokenGenerator struct {
	GenerateAccessTokenFunc func(testID uuid.UUID, email string, expiryHours int) (string, error)
}

func (m *MockAccessTokenGenerator) GenerateAccessToken(testID uuid.UUID, email string, expiryHours int) (string, error) {
	if m.GenerateAccessTokenFunc != nil {
		return m.GenerateAccessTokenFunc(testID, email, expiryHours)
	}
	return "", nil
}

type MockEmailSender struct {
	SendTestAccessTokenFunc func(to, testTitle, accessToken, accessURL string) error
}

func (m *MockEmailSender) SendTestAccessToken(to, testTitle, accessToken, accessURL string) error {
	if m.SendTestAccessTokenFunc != nil {
		return m.SendTestAccessTokenFunc(to, testTitle, accessToken, accessURL)
	}
	return nil
}

type MockSubmissionRepository struct {
	FindByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
}

func (m *MockSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSubmissionRepository) Create(ctx context.Context, submission *domain.Submission) error {
	return nil
}

func (m *MockSubmissionRepository) Update(ctx context.Context, submission *domain.Submission) error {
	return nil
}

func (m *MockSubmissionRepository) FindByTestID(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
	return nil, nil
}

func (m *MockSubmissionRepository) CountByTestAndEmail(ctx context.Context, testID uuid.UUID, email string) (int, error) {
	return 0, nil
}

type MockAnswerRepository struct {
	FindByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.Answer, error)
	FindBySubmissionIDFunc func(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error)
}

func (m *MockAnswerRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAnswerRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
	if m.FindBySubmissionIDFunc != nil {
		return m.FindBySubmissionIDFunc(ctx, submissionID)
	}
	return nil, nil
}

func (m *MockAnswerRepository) Create(ctx context.Context, answer *domain.Answer) error {
	return nil
}

func (m *MockAnswerRepository) CreateBatch(ctx context.Context, answers []*domain.Answer) error {
	return nil
}

type MockReviewRepository struct {
	FindByAnswerIDFunc func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error)
}

func (m *MockReviewRepository) FindByAnswerID(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
	if m.FindByAnswerIDFunc != nil {
		return m.FindByAnswerIDFunc(ctx, answerID)
	}
	return nil, nil
}

func (m *MockReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	return nil
}

func (m *MockReviewRepository) Update(ctx context.Context, review *domain.Review) error {
	return nil
}

func (m *MockReviewRepository) UpsertManualScore(ctx context.Context, answerID, reviewerID uuid.UUID, score float64, feedback string) error {
	return nil
}

func (m *MockReviewRepository) UpsertAIScore(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error {
	return nil
}

type MockTokenValidator struct {
	ValidateTokenFunc func(tokenString string) (testID string, email string, role string, err error)
}

func (m *MockTokenValidator) ValidateToken(tokenString string) (testID string, email string, role string, err error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(tokenString)
	}
	return "", "", "", nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
