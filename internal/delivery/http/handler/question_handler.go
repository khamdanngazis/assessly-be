package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// QuestionHandler handles question-related HTTP requests
type QuestionHandler struct {
	addQuestionUC    *test.AddQuestionUseCase
	updateQuestionUC *test.UpdateQuestionUseCase
	deleteQuestionUC *test.DeleteQuestionUseCase
	logger           *slog.Logger
}

// NewQuestionHandler creates a new question handler
func NewQuestionHandler(
	addQuestionUC *test.AddQuestionUseCase,
	updateQuestionUC *test.UpdateQuestionUseCase,
	deleteQuestionUC *test.DeleteQuestionUseCase,
	logger *slog.Logger,
) *QuestionHandler {
	return &QuestionHandler{
		addQuestionUC:    addQuestionUC,
		updateQuestionUC: updateQuestionUC,
		deleteQuestionUC: deleteQuestionUC,
		logger:           logger,
	}
}

// AddQuestionRequest represents the add question request body
type AddQuestionRequest struct {
	Text           string `json:"text"`
	ExpectedAnswer string `json:"expected_answer"`
	OrderNum       int    `json:"order_num,omitempty"`
}

// QuestionResponse represents a question in responses
type QuestionResponse struct {
	ID             string `json:"id"`
	TestID         string `json:"test_id"`
	Text           string `json:"text"`
	ExpectedAnswer string `json:"expected_answer,omitempty"` // Hide from participants
	OrderNum       int    `json:"order_num"`
	CreatedAt      string `json:"created_at"`
}

// AddQuestion handles adding a question to a test
func (h *QuestionHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	testIDStr := chi.URLParam(r, "testID")
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	var req AddQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	questionEntity, err := h.addQuestionUC.Execute(r.Context(), test.AddQuestionRequest{
		TestID:         testID,
		Text:           req.Text,
		ExpectedAnswer: req.ExpectedAnswer,
		OrderNum:       req.OrderNum,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with created question
	resp := QuestionResponse{
		ID:             questionEntity.ID.String(),
		TestID:         questionEntity.TestID.String(),
		Text:           questionEntity.Text,
		ExpectedAnswer: questionEntity.ExpectedAnswer,
		OrderNum:       questionEntity.OrderNum,
		CreatedAt:      questionEntity.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// UpdateQuestionRequest represents the update question request body
type UpdateQuestionRequest struct {
	Text           string `json:"text"`
	ExpectedAnswer string `json:"expected_answer"`
	OrderNum       int    `json:"order_num,omitempty"`
}

// UpdateQuestion handles updating a question
func (h *QuestionHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	// Extract creator ID from JWT context
	creatorIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid user ID")
		return
	}

	// Extract test ID and question ID from URL
	testIDStr := chi.URLParam(r, "testID")
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	questionIDStr := chi.URLParam(r, "questionID")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid question ID")
		return
	}

	// Parse request body
	var req UpdateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	questionEntity, err := h.updateQuestionUC.Execute(r.Context(), test.UpdateQuestionRequest{
		QuestionID:     questionID,
		TestID:         testID,
		CreatorID:      creatorID,
		Text:           req.Text,
		ExpectedAnswer: req.ExpectedAnswer,
		OrderNum:       req.OrderNum,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with updated question
	resp := QuestionResponse{
		ID:             questionEntity.ID.String(),
		TestID:         questionEntity.TestID.String(),
		Text:           questionEntity.Text,
		ExpectedAnswer: questionEntity.ExpectedAnswer,
		OrderNum:       questionEntity.OrderNum,
		CreatedAt:      questionEntity.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteQuestion handles deleting a question
func (h *QuestionHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	// Extract creator ID from JWT context
	creatorIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid user ID")
		return
	}

	// Extract test ID and question ID from URL
	testIDStr := chi.URLParam(r, "testID")
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	questionIDStr := chi.URLParam(r, "questionID")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid question ID")
		return
	}

	// Execute use case
	err = h.deleteQuestionUC.Execute(r.Context(), test.DeleteQuestionRequest{
		QuestionID: questionID,
		TestID:     testID,
		CreatorID:  creatorID,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with success (no content)
	w.WriteHeader(http.StatusNoContent)
}

// handleError handles domain errors and converts them to HTTP responses
func (h *QuestionHandler) handleError(w http.ResponseWriter, err error) {
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
func (h *QuestionHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// respondError writes an error JSON response
func (h *QuestionHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
