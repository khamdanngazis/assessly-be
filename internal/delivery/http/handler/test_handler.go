package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TestHandler handles test-related HTTP requests
type TestHandler struct {
	createTestUC  *test.CreateTestUseCase
	publishTestUC *test.PublishTestUseCase
	logger        *slog.Logger
}

// NewTestHandler creates a new test handler
func NewTestHandler(
	createTestUC *test.CreateTestUseCase,
	publishTestUC *test.PublishTestUseCase,
	logger *slog.Logger,
) *TestHandler {
	return &TestHandler{
		createTestUC:  createTestUC,
		publishTestUC: publishTestUC,
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
	ID           string `json:"id"`
	CreatorID    string `json:"creator_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	AllowRetakes bool   `json:"allow_retakes"`
	IsPublished  bool   `json:"is_published"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CreateTest handles test creation
func (h *TestHandler) CreateTest(w http.ResponseWriter, r *http.Request) {
	var req CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get creator ID from context (set by JWT middleware)
	creatorIDStr := r.Context().Value("user_id")
	if creatorIDStr == nil {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	creatorID, err := uuid.Parse(creatorIDStr.(string))
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
