package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TestHandler handles test-related HTTP requests
type TestHandler struct {
	createTestUC  *test.CreateTestUseCase
	publishTestUC *test.PublishTestUseCase
	listTestsUC   *test.ListTestsUseCase
	getTestUC     *test.GetTestUseCase
	logger        *slog.Logger
}

// NewTestHandler creates a new test handler
func NewTestHandler(
	createTestUC *test.CreateTestUseCase,
	publishTestUC *test.PublishTestUseCase,
	listTestsUC *test.ListTestsUseCase,
	getTestUC *test.GetTestUseCase,
	logger *slog.Logger,
) *TestHandler {
	return &TestHandler{
		createTestUC:  createTestUC,
		publishTestUC: publishTestUC,
		listTestsUC:   listTestsUC,
		getTestUC:     getTestUC,
		logger:        logger,
	}
}

// CreateTestRequest represents the create test request body
type CreateTestRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	AllowRetakes bool   `json:"allow_retakes"`
}

// TestResponse represents a test in responses
type TestResponse struct {
	ID           string             `json:"id"`
	CreatorID    string             `json:"creator_id"`
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	AllowRetakes bool               `json:"allow_retakes"`
	IsPublished  bool               `json:"is_published"`
	Questions    []QuestionResponse `json:"questions,omitempty"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
}

// ListTests handles listing tests based on user role
func (h *TestHandler) ListTests(w http.ResponseWriter, r *http.Request) {
	// Get user ID and role from context
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid user ID")
		return
	}

	userRole, ok := middleware.GetUserRole(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user role not found")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}

	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := parsePositiveInt(pageStr); err == nil {
			page = p
		}
	}

	if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
		if s, err := parsePositiveInt(sizeStr); err == nil && s <= 100 {
			pageSize = s
		}
	}

	// Execute use case
	resp, err := h.listTestsUC.Execute(r.Context(), test.ListTestsRequest{
		UserID:   userID,
		UserRole: userRole,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Convert tests to response format
	testResponses := make([]TestResponse, len(resp.Tests))
	for i, t := range resp.Tests {
		// Get questions for this test from the questions map
		questions := resp.Questions[t.ID.String()]
		questionResponses := make([]QuestionResponse, len(questions))
		for j, q := range questions {
			questionResponses[j] = QuestionResponse{
				ID:             q.ID.String(),
				TestID:         q.TestID.String(),
				Text:           q.Text,
				ExpectedAnswer: q.ExpectedAnswer,
				OrderNum:       q.OrderNum,
				CreatedAt:      q.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}

		testResponses[i] = TestResponse{
			ID:           t.ID.String(),
			CreatorID:    t.CreatorID.String(),
			Title:        t.Title,
			Description:  t.Description,
			AllowRetakes: t.AllowRetakes,
			IsPublished:  t.IsPublished,
			Questions:    questionResponses,
			CreatedAt:    t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tests": testResponses,
		"pagination": map[string]interface{}{
			"page":      page,
			"page_size": pageSize,
			"total":     resp.Total,
		},
	})
}

// GetTest handles retrieving a single test by ID
func (h *TestHandler) GetTest(w http.ResponseWriter, r *http.Request) {
	// Get test ID from URL
	testIDStr := chi.URLParam(r, "testID")
	if testIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "test ID is required")
		return
	}

	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	// Get user ID and role from context
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid user ID")
		return
	}

	userRole, ok := middleware.GetUserRole(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user role not found")
		return
	}

	// Execute use case
	result, err := h.getTestUC.Execute(r.Context(), test.GetTestRequest{
		TestID:   testID,
		UserID:   userID,
		UserRole: userRole,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Convert questions to response format
	questionResponses := make([]QuestionResponse, len(result.Questions))
	for i, q := range result.Questions {
		questionResponses[i] = QuestionResponse{
			ID:             q.ID.String(),
			TestID:         q.TestID.String(),
			Text:           q.Text,
			ExpectedAnswer: q.ExpectedAnswer,
			OrderNum:       q.OrderNum,
			CreatedAt:      q.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Convert to response format
	testResponse := TestResponse{
		ID:           result.Test.ID.String(),
		CreatorID:    result.Test.CreatorID.String(),
		Title:        result.Test.Title,
		Description:  result.Test.Description,
		AllowRetakes: result.Test.AllowRetakes,
		IsPublished:  result.Test.IsPublished,
		Questions:    questionResponses,
		CreatedAt:    result.Test.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    result.Test.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusOK, testResponse)
}

// CreateTest handles test creation
func (h *TestHandler) CreateTest(w http.ResponseWriter, r *http.Request) {
	var req CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get creator ID from context (set by JWT middleware)
	creatorIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid user ID")
		return
	}

	// Execute use case
	testEntity, err := h.createTestUC.Execute(r.Context(), test.CreateTestRequest{
		CreatorID:    creatorID,
		Title:        req.Title,
		Description:  req.Description,
		AllowRetakes: req.AllowRetakes,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with created test
	resp := TestResponse{
		ID:           testEntity.ID.String(),
		CreatorID:    testEntity.CreatorID.String(),
		Title:        testEntity.Title,
		Description:  testEntity.Description,
		AllowRetakes: testEntity.AllowRetakes,
		IsPublished:  testEntity.IsPublished,
		CreatedAt:    testEntity.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    testEntity.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// PublishTest handles test publication
func (h *TestHandler) PublishTest(w http.ResponseWriter, r *http.Request) {
	testIDStr := chi.URLParam(r, "testID")
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	// Execute use case
	testEntity, err := h.publishTestUC.Execute(r.Context(), test.PublishTestRequest{
		TestID: testID,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with published test
	resp := TestResponse{
		ID:           testEntity.ID.String(),
		CreatorID:    testEntity.CreatorID.String(),
		Title:        testEntity.Title,
		Description:  testEntity.Description,
		AllowRetakes: testEntity.AllowRetakes,
		IsPublished:  testEntity.IsPublished,
		CreatedAt:    testEntity.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    testEntity.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// handleError handles domain errors and converts them to HTTP responses
func (h *TestHandler) handleError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case domain.ErrValidation:
		h.respondError(w, http.StatusBadRequest, e.Error())
	case domain.ErrUnauthorized:
		h.respondError(w, http.StatusUnauthorized, e.Error())
	case domain.ErrConflict:
		h.respondError(w, http.StatusConflict, e.Error())
	case domain.ErrNotFound:
		h.respondError(w, http.StatusNotFound, e.Error())
	case domain.ErrInternal:
		h.logger.Error("internal error", "error", e)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	default:
		h.logger.Error("unexpected error", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// respondJSON writes a JSON response
func (h *TestHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// respondError writes an error JSON response
func (h *TestHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// parsePositiveInt parses a string to a positive integer
func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return n, nil
}
