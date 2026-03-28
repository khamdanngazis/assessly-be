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
	testUC "github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateTestContract validates the HTTP contract for POST /api/v1/tests
func TestCreateTestContract(t *testing.T) {
	// Setup mocks
	mockTestRepo := &MockTestRepository{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	createTestUC := testUC.NewCreateTestUseCase(mockTestRepo, logger)
	testHandler := handler.NewTestHandler(createTestUC, nil, logger)

	t.Run("should return 201 with correct response schema on successful test creation", func(t *testing.T) {
		// Prepare mock response
		creatorID := uuid.New()
		mockTestRepo.CreateFunc = func(ctx context.Context, test *domain.Test) error {
			test.ID = uuid.New()
			test.CreatedAt = time.Now()
			test.UpdatedAt = time.Now()
			return nil
		}

		// Prepare request with user context
		reqBody := map[string]interface{}{
			"title":         "Sample Test",
			"description":   "This is a sample test",
			"allow_retakes": true,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		
		// Add user_id to context (simulating JWT middleware)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, creatorID.String())
		req = req.WithContext(ctx)
		
		w := httptest.NewRecorder()

		// Execute
		testHandler.CreateTest(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusCreated, w.Code, "should return 201 Created")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate required fields exist
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "creator_id", "response should contain creator_id field")
		assert.Contains(t, resp, "title", "response should contain title field")
		assert.Contains(t, resp, "description", "response should contain description field")
		assert.Contains(t, resp, "allow_retakes", "response should contain allow_retakes field")
		assert.Contains(t, resp, "is_published", "response should contain is_published field")
		assert.Contains(t, resp, "created_at", "response should contain created_at field")
		assert.Contains(t, resp, "updated_at", "response should contain updated_at field")

		// Validate field types
		assert.IsType(t, "", resp["id"], "id should be string")
		assert.IsType(t, "", resp["creator_id"], "creator_id should be string")
		assert.IsType(t, "", resp["title"], "title should be string")
		assert.IsType(t, "", resp["description"], "description should be string")
		assert.IsType(t, false, resp["allow_retakes"], "allow_retakes should be boolean")
		assert.IsType(t, false, resp["is_published"], "is_published should be boolean")
		assert.IsType(t, "", resp["created_at"], "created_at should be string")
		assert.IsType(t, "", resp["updated_at"], "updated_at should be string")

		// Validate field values
		assert.Equal(t, "Sample Test", resp["title"], "title should match request")
		assert.Equal(t, "This is a sample test", resp["description"], "description should match request")
		assert.Equal(t, true, resp["allow_retakes"], "allow_retakes should match request")
		assert.Equal(t, false, resp["is_published"], "is_published should be false for new tests")
	})

	t.Run("should return 400 on invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, uuid.New().String())
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		testHandler.CreateTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 401 when user not authenticated", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"title":         "Sample Test",
			"description":   "Test description",
			"allow_retakes": false,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		// No user_id in context
		w := httptest.NewRecorder()

		testHandler.CreateTest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "should return 401 Unauthorized")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when validation fails", func(t *testing.T) {
		creatorID := uuid.New()
		
		reqBody := map[string]interface{}{
			"title":         "",  // Empty title
			"description":   "Test description",
			"allow_retakes": false,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, creatorID.String())
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		testHandler.CreateTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestAddQuestionContract validates the HTTP contract for POST /api/v1/tests/:id/questions
func TestAddQuestionContract(t *testing.T) {
	// Setup mocks
	mockQuestionRepo := &MockQuestionRepository{}
	mockTestRepo := &MockTestRepository{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	addQuestionUC := testUC.NewAddQuestionUseCase(mockQuestionRepo, mockTestRepo, logger)
	questionHandler := handler.NewQuestionHandler(addQuestionUC, logger)

	t.Run("should return 201 with correct response schema on successful question addition", func(t *testing.T) {
		testID := uuid.New()
		
		// Mock test exists and is not published
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				IsPublished: false,
			}, nil
		}
		
		mockQuestionRepo.CreateFunc = func(ctx context.Context, question *domain.Question) error {
			question.ID = uuid.New()
			question.CreatedAt = time.Now()
			return nil
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"text":            "What is Go?",
			"expected_answer": "A programming language",
			"order_num":       1,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/questions", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		
		// Add testID to chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		// Execute
		questionHandler.AddQuestion(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusCreated, w.Code, "should return 201 Created")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate required fields exist
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "test_id", "response should contain test_id field")
		assert.Contains(t, resp, "text", "response should contain text field")
		assert.Contains(t, resp, "expected_answer", "response should contain expected_answer field")
		assert.Contains(t, resp, "order_num", "response should contain order_num field")
		assert.Contains(t, resp, "created_at", "response should contain created_at field")

		// Validate field types
		assert.IsType(t, "", resp["id"], "id should be string")
		assert.IsType(t, "", resp["test_id"], "test_id should be string")
		assert.IsType(t, "", resp["text"], "text should be string")
		assert.IsType(t, "", resp["expected_answer"], "expected_answer should be string")
		assert.IsType(t, float64(0), resp["order_num"], "order_num should be number")
		assert.IsType(t, "", resp["created_at"], "created_at should be string")

		// Validate field values
		assert.Equal(t, "What is Go?", resp["text"], "text should match request")
		assert.Equal(t, "A programming language", resp["expected_answer"], "expected_answer should match request")
		assert.Equal(t, float64(1), resp["order_num"], "order_num should match request")
	})

	t.Run("should return 400 on invalid test ID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"text":            "What is Go?",
			"expected_answer": "A programming language",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/invalid-uuid/questions", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		questionHandler.AddQuestion(w, req)

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
			"text":            "What is Go?",
			"expected_answer": "A programming language",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/questions", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		questionHandler.AddQuestion(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "should return 404 Not Found")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when test is already published", func(t *testing.T) {
		testID := uuid.New()
		
		// Mock test exists but is published
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				IsPublished: true,  // Already published
			}, nil
		}

		reqBody := map[string]interface{}{
			"text":            "What is Go?",
			"expected_answer": "A programming language",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/questions", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		questionHandler.AddQuestion(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestPublishTestContract validates the HTTP contract for POST /api/v1/tests/:id/publish
func TestPublishTestContract(t *testing.T) {
	// Setup mocks
	mockTestRepo := &MockTestRepository{}
	mockQuestionRepo := &MockQuestionRepository{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	publishTestUC := testUC.NewPublishTestUseCase(mockTestRepo, mockQuestionRepo, logger)
	testHandler := handler.NewTestHandler(nil, publishTestUC, logger)

	t.Run("should return 200 with correct response schema on successful publish", func(t *testing.T) {
		testID := uuid.New()
		creatorID := uuid.New()
		
		// Mock test exists and is not published, has questions
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				CreatorID:   creatorID,
				Title:       "Sample Test",
				Description: "Test description",
				IsPublished: false,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		}
		
		mockQuestionRepo.CountByTestIDFunc = func(ctx context.Context, testID uuid.UUID) (int, error) {
			return 3, nil  // Has 3 questions
		}
		
		mockTestRepo.UpdateFunc = func(ctx context.Context, test *domain.Test) error {
			test.IsPublished = true
			test.UpdatedAt = time.Now()
			return nil
		}

		// Prepare request
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/publish", nil)
		
		// Add testID to chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		// Execute
		testHandler.PublishTest(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate required fields exist
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "is_published", "response should contain is_published field")

		// Validate field values
		assert.Equal(t, true, resp["is_published"], "is_published should be true after publishing")
	})

	t.Run("should return 400 on invalid test ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/invalid-uuid/publish", nil)
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		testHandler.PublishTest(w, req)

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

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/publish", nil)
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		testHandler.PublishTest(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "should return 404 Not Found")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when test has no questions", func(t *testing.T) {
		testID := uuid.New()
		
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				IsPublished: false,
			}, nil
		}
		
		mockQuestionRepo.CountByTestIDFunc = func(ctx context.Context, testID uuid.UUID) (int, error) {
			return 0, nil  // No questions
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/publish", nil)
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		testHandler.PublishTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when test is already published", func(t *testing.T) {
		testID := uuid.New()
		
		mockTestRepo.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
			return &domain.Test{
				ID:          testID,
				IsPublished: true,  // Already published
			}, nil
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tests/"+testID.String()+"/publish", nil)
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("testID", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		testHandler.PublishTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// Mock implementations for contract tests

type MockTestRepository struct {
	CreateFunc    func(ctx context.Context, test *domain.Test) error
	FindByIDFunc  func(ctx context.Context, id uuid.UUID) (*domain.Test, error)
	UpdateFunc    func(ctx context.Context, test *domain.Test) error
}

func (m *MockTestRepository) Create(ctx context.Context, test *domain.Test) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, test)
	}
	return nil
}

func (m *MockTestRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTestRepository) FindByCreatorID(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*domain.Test, error) {
	return nil, nil
}

func (m *MockTestRepository) FindPublished(ctx context.Context, limit, offset int) ([]*domain.Test, error) {
	return nil, nil
}

func (m *MockTestRepository) Update(ctx context.Context, test *domain.Test) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, test)
	}
	return nil
}

func (m *MockTestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type MockQuestionRepository struct {
	CreateFunc         func(ctx context.Context, question *domain.Question) error
	CountByTestIDFunc  func(ctx context.Context, testID uuid.UUID) (int, error)
}

func (m *MockQuestionRepository) Create(ctx context.Context, question *domain.Question) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, question)
	}
	return nil
}

func (m *MockQuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	return nil, nil
}

func (m *MockQuestionRepository) FindByTestID(ctx context.Context, testID uuid.UUID) ([]*domain.Question, error) {
	return nil, nil
}

func (m *MockQuestionRepository) CountByTestID(ctx context.Context, testID uuid.UUID) (int, error) {
	if m.CountByTestIDFunc != nil {
		return m.CountByTestIDFunc(ctx, testID)
	}
	return 0, nil
}

func (m *MockQuestionRepository) Update(ctx context.Context, question *domain.Question) error {
	return nil
}

func (m *MockQuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
