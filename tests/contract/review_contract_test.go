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
	"github.com/assessly/assessly-be/internal/domain"
	reviewUC "github.com/assessly/assessly-be/internal/usecase/review"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddManualReviewContract validates PUT /api/v1/reviews/:answerId
func TestAddManualReviewContract(t *testing.T) {
	// Setup mocks
	mockReviewRepo := &MockReviewRepositoryForReview{}
	mockAnswerRepo := &MockAnswerRepositoryForReview{}
	mockSubmissionRepo := &MockSubmissionRepositoryForReview{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	addManualReviewUC := reviewUC.NewAddManualReviewUseCase(mockReviewRepo, mockAnswerRepo, mockSubmissionRepo, logger)
	reviewHandler := handler.NewReviewHandler(addManualReviewUC, nil, nil, logger)

	t.Run("should return 200 with updated review on success", func(t *testing.T) {
		answerID := uuid.New()
		reviewerID := uuid.New()
		submissionID := uuid.New()
		manualScore := 90.0
		aiScore := 85.0

		// Prepare mock responses
		mockAnswerRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
			return &domain.Answer{
				ID:           answerID,
				SubmissionID: submissionID,
				QuestionID:   uuid.New(),
				Text:         "Answer text",
				CreatedAt:    time.Now(),
			}, nil
		}

		mockReviewRepo.UpsertManualScoreFunc = func(ctx context.Context, answerID, reviewerID uuid.UUID, score float64, feedback string) error {
			return nil
		}

		mockReviewRepo.FindByAnswerIDFunc = func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
			return &domain.Review{
				ID:             uuid.New(),
				AnswerID:       answerID,
				ReviewerID:     &reviewerID,
				AIScore:        &aiScore,
				AIFeedback:     stringPtr("Good answer"),
				ManualScore:    &manualScore,
				ManualFeedback: stringPtr("Excellent answer with great depth"),
			}, nil
		}

		mockAnswerRepo.FindBySubmissionIDFunc = func(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
			return []*domain.Answer{
				{ID: answerID, SubmissionID: submissionID},
			}, nil
		}

		mockSubmissionRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
			return &domain.Submission{
				ID:          submissionID,
				TestID:      uuid.New(),
				AccessEmail: "participant@example.com",
			}, nil
		}

		mockSubmissionRepo.UpdateFunc = func(ctx context.Context, submission *domain.Submission) error {
			return nil
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"manual_score":    manualScore,
			"manual_feedback": "Excellent answer with great depth",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/"+answerID.String(), bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")

		// Setup chi URL params and context with user_id
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", answerID.String())
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = context.WithValue(ctx, "user_id", reviewerID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		// Execute
		reviewHandler.HandleAddManualReview(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate review fields
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "answer_id", "response should contain answer_id field")
		assert.Contains(t, resp, "manual_score", "response should contain manual_score field")
		assert.Contains(t, resp, "manual_feedback", "response should contain manual_feedback field")

		// Verify manual score was set
		assert.Equal(t, manualScore, resp["manual_score"], "manual_score should match request")
		assert.Equal(t, "Excellent answer with great depth", resp["manual_feedback"], "manual_feedback should match request")
	})

	t.Run("should return 400 on invalid JSON body", func(t *testing.T) {
		answerID := uuid.New()
		reviewerID := uuid.New()

		req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/"+answerID.String(), bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		// Setup chi URL params and context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", answerID.String())
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = context.WithValue(ctx, "user_id", reviewerID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		reviewHandler.HandleAddManualReview(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 on invalid answer ID", func(t *testing.T) {
		reviewerID := uuid.New()

		reqBody := map[string]interface{}{
			"manual_score":    90.0,
			"manual_feedback": "Great answer",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/invalid-uuid", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")

		// Setup chi URL params and context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", "invalid-uuid")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = context.WithValue(ctx, "user_id", reviewerID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		reviewHandler.HandleAddManualReview(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		answerID := uuid.New()

		reqBody := map[string]interface{}{
			"manual_score":    90.0,
			"manual_feedback": "Excellent answer",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/"+answerID.String(), bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")

		// Setup chi URL params but NO user_id in context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", answerID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		reviewHandler.HandleAddManualReview(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "should return 401 Unauthorized")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestGetReviewContract validates GET /api/v1/reviews/:answerId
func TestGetReviewContract(t *testing.T) {
	// Setup mocks
	mockReviewRepo := &MockReviewRepositoryForReview{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	getReviewUC := reviewUC.NewGetReviewUseCase(mockReviewRepo, logger)
	reviewHandler := handler.NewReviewHandler(nil, getReviewUC, nil, logger)

	t.Run("should return 200 with review details", func(t *testing.T) {
		answerID := uuid.New()
		reviewerID := uuid.New()
		aiScore := 85.5
		manualScore := 90.0

		// Prepare mock response
		mockReviewRepo.FindByAnswerIDFunc = func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
			return &domain.Review{
				ID:             uuid.New(),
				AnswerID:       answerID,
				ReviewerID:     &reviewerID,
				AIScore:        &aiScore,
				AIFeedback:     stringPtr("Good answer"),
				ManualScore:    &manualScore,
				ManualFeedback: stringPtr("Excellent work"),
			}, nil
		}

		// Prepare request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/"+answerID.String(), nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", answerID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		// Execute
		reviewHandler.HandleGetReview(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate review fields
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "answer_id", "response should contain answer_id field")
		assert.Contains(t, resp, "ai_score", "response should contain ai_score field")
		assert.Contains(t, resp, "ai_feedback", "response should contain ai_feedback field")
		assert.Contains(t, resp, "manual_score", "response should contain manual_score field")
		assert.Contains(t, resp, "manual_feedback", "response should contain manual_feedback field")

		// Validate display_score (manual overrides AI)
		assert.Contains(t, resp, "display_score", "response should contain display_score field")
		assert.Equal(t, manualScore, resp["display_score"], "display_score should equal manual score when present")
		assert.Contains(t, resp, "display_feedback", "response should contain display_feedback field")
	})

	t.Run("should return 400 on invalid answer ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/invalid-uuid", nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		reviewHandler.HandleGetReview(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 404 when review not found", func(t *testing.T) {
		answerID := uuid.New()
		mockReviewRepo.FindByAnswerIDFunc = func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
			return nil, domain.ErrNotFound{Resource: "review", ID: answerID.String()}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/"+answerID.String(), nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("answerId", answerID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		reviewHandler.HandleGetReview(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "should return 404 Not Found")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestListSubmissionsContract validates GET /api/v1/tests/:testId/submissions
func TestListSubmissionsContract(t *testing.T) {
	// Setup mocks
	mockSubmissionRepo := &MockSubmissionRepositoryForReview{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	listSubmissionsUC := reviewUC.NewListSubmissionsUseCase(mockSubmissionRepo, logger)
	reviewHandler := handler.NewReviewHandler(nil, nil, listSubmissionsUC, logger)

	t.Run("should return 200 with list of submissions", func(t *testing.T) {
		testID := uuid.New()
		submission1ID := uuid.New()
		submission2ID := uuid.New()

		// Prepare mock response
		mockSubmissionRepo.FindByTestIDFunc = func(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
			aiScore1 := 85.0
			manualScore1 := 90.0
			aiScore2 := 75.0

			return []*domain.Submission{
				{
					ID:               submission1ID,
					TestID:           testID,
					AccessEmail:      "participant1@example.com",
					SubmittedAt:      time.Now().Add(-2 * time.Hour),
					AITotalScore:     &aiScore1,
					ManualTotalScore: &manualScore1,
				},
				{
					ID:               submission2ID,
					TestID:           testID,
					AccessEmail:      "participant2@example.com",
					SubmittedAt:      time.Now().Add(-1 * time.Hour),
					AITotalScore:     &aiScore2,
					ManualTotalScore: nil,
				},
			}, nil
		}

		// Prepare request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/"+testID.String()+"/submissions", nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testId", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		// Execute
		reviewHandler.HandleListSubmissions(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate submissions array
		submissions, ok := resp["submissions"].([]interface{})
		require.True(t, ok, "response should contain submissions array")
		assert.Len(t, submissions, 2, "should return 2 submissions")

		// Validate first submission structure
		firstSubmission := submissions[0].(map[string]interface{})
		assert.Contains(t, firstSubmission, "id", "submission should contain id")
		assert.Contains(t, firstSubmission, "test_id", "submission should contain test_id")
		assert.Contains(t, firstSubmission, "access_email", "submission should contain access_email")
		assert.Contains(t, firstSubmission, "submitted_at", "submission should contain submitted_at")
		assert.Contains(t, firstSubmission, "ai_total_score", "submission should contain ai_total_score")
		assert.Contains(t, firstSubmission, "manual_total_score", "submission should contain manual_total_score")
	})

	t.Run("should return 400 on invalid test ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/invalid-uuid/submissions", nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testId", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		reviewHandler.HandleListSubmissions(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 200 with empty array when no submissions", func(t *testing.T) {
		testID := uuid.New()

		mockSubmissionRepo.FindByTestIDFunc = func(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
			return []*domain.Submission{}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/"+testID.String()+"/submissions", nil)

		// Setup chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testId", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		reviewHandler.HandleListSubmissions(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		submissions, ok := resp["submissions"].([]interface{})
		require.True(t, ok, "response should contain submissions array")
		assert.Empty(t, submissions, "submissions array should be empty")
	})
}

// Mock implementations for review contract tests

type MockReviewRepositoryForReview struct {
	FindByAnswerIDFunc     func(ctx context.Context, answerID uuid.UUID) (*domain.Review, error)
	UpsertManualScoreFunc  func(ctx context.Context, answerID, reviewerID uuid.UUID, score float64, feedback string) error
	UpsertAIScoreFunc      func(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error
}

func (m *MockReviewRepositoryForReview) FindByAnswerID(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
	if m.FindByAnswerIDFunc != nil {
		return m.FindByAnswerIDFunc(ctx, answerID)
	}
	return nil, nil
}

func (m *MockReviewRepositoryForReview) UpsertManualScore(ctx context.Context, answerID, reviewerID uuid.UUID, score float64, feedback string) error {
	if m.UpsertManualScoreFunc != nil {
		return m.UpsertManualScoreFunc(ctx, answerID, reviewerID, score, feedback)
	}
	return nil
}

func (m *MockReviewRepositoryForReview) UpsertAIScore(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error {
	if m.UpsertAIScoreFunc != nil {
		return m.UpsertAIScoreFunc(ctx, answerID, score, feedback)
	}
	return nil
}

func (m *MockReviewRepositoryForReview) Create(ctx context.Context, review *domain.Review) error {
	return nil
}

func (m *MockReviewRepositoryForReview) Update(ctx context.Context, review *domain.Review) error {
	return nil
}

type MockAnswerRepositoryForReview struct {
	FindByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.Answer, error)
	FindBySubmissionIDFunc func(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error)
}

func (m *MockAnswerRepositoryForReview) FindByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAnswerRepositoryForReview) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
	if m.FindBySubmissionIDFunc != nil {
		return m.FindBySubmissionIDFunc(ctx, submissionID)
	}
	return nil, nil
}

func (m *MockAnswerRepositoryForReview) Create(ctx context.Context, answer *domain.Answer) error {
	return nil
}

func (m *MockAnswerRepositoryForReview) CreateBatch(ctx context.Context, answers []*domain.Answer) error {
	return nil
}

type MockSubmissionRepositoryForReview struct {
	FindByIDFunc     func(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
	FindByTestIDFunc func(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error)
	UpdateFunc       func(ctx context.Context, submission *domain.Submission) error
}

func (m *MockSubmissionRepositoryForReview) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSubmissionRepositoryForReview) FindByTestID(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
	if m.FindByTestIDFunc != nil {
		return m.FindByTestIDFunc(ctx, testID, limit, offset)
	}
	return nil, nil
}

func (m *MockSubmissionRepositoryForReview) Update(ctx context.Context, submission *domain.Submission) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, submission)
	}
	return nil
}

func (m *MockSubmissionRepositoryForReview) Create(ctx context.Context, submission *domain.Submission) error {
	return nil
}

func (m *MockSubmissionRepositoryForReview) CountByTestAndEmail(ctx context.Context, testID uuid.UUID, email string) (int, error) {
	return 0, nil
}
