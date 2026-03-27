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

// QuestionHandler handles question-related HTTP requests
type QuestionHandler struct {
	addQuestionUC *test.AddQuestionUseCase
	logger        *slog.Logger
}

// NewQuestionHandler creates a new question handler
func NewQuestionHandler(
	addQuestionUC *test.AddQuestionUseCase,
	logger *slog.Logger,
) *QuestionHandler {
	return &QuestionHandler{
		addQuestionUC: addQuestionUC,
		logger:        logger,
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
