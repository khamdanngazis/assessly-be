package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/usecase/review"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ReviewHandler handles review-related HTTP requests
type ReviewHandler struct {
	addManualReview *review.AddManualReviewUseCase
	getReview       *review.GetReviewUseCase
	listSubmissions *review.ListSubmissionsUseCase
	logger          *slog.Logger
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(
	addManualReview *review.AddManualReviewUseCase,
	getReview       *review.GetReviewUseCase,
	listSubmissions *review.ListSubmissionsUseCase,
	logger          *slog.Logger,
) *ReviewHandler {
	return &ReviewHandler{
		addManualReview: addManualReview,
		getReview:       getReview,
		listSubmissions: listSubmissions,
		logger:          logger,
	}
}

// AddManualReviewRequest is the HTTP request body for adding manual review
type AddManualReviewRequest struct {
	ManualScore    float64 `json:"manual_score"`
	ManualFeedback string  `json:"manual_feedback"`
}

// HandleAddManualReview handles PUT /api/v1/reviews/:answerId
// Requires reviewer role (enforced by middleware)
func (h *ReviewHandler) HandleAddManualReview(w http.ResponseWriter, r *http.Request) {
	// Get answer ID from URL
	answerIDStr := chi.URLParam(r, "answerId")
	answerID, err := uuid.Parse(answerIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid answer ID")
		return
	}

	// Get reviewer ID from context (set by auth middleware)
	reviewerID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "reviewer ID not found")
		return
	}

	// Parse request body
	var req AddManualReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	resp, err := h.addManualReview.Execute(r.Context(), review.AddManualReviewRequest{
		AnswerID:       answerID,
		ReviewerID:     reviewerID,
		ManualScore:    req.ManualScore,
		ManualFeedback: req.ManualFeedback,
	})
	if err != nil {
		h.logger.Error("failed to add manual review", "error", err)
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response
	response := ReviewResponse{
		ID:             resp.Review.ID.String(),
		AnswerID:       resp.Review.AnswerID.String(),
		AIScore:        resp.Review.AIScore,
		AIFeedback:     resp.Review.AIFeedback,
		ManualScore:    resp.Review.ManualScore,
		ManualFeedback: resp.Review.ManualFeedback,
	}

	if resp.Review.AIScoredAt != nil {
		scoredAt := resp.Review.AIScoredAt.Format("2006-01-02T15:04:05Z07:00")
		response.AIScoredAt = &scoredAt
	}

	if resp.Review.ManualScoredAt != nil {
		scoredAt := resp.Review.ManualScoredAt.Format("2006-01-02T15:04:05Z07:00")
		response.ManualScoredAt = &scoredAt
	}

	if resp.Review.ReviewerID != nil {
		reviewerID := resp.Review.ReviewerID.String()
		response.ReviewerID = &reviewerID
	}

	// T090: Set display fields - prioritize manual over AI
	if resp.Review.ManualScore != nil {
		response.DisplayScore = resp.Review.ManualScore
		response.DisplayFeedback = resp.Review.ManualFeedback
	} else if resp.Review.AIScore != nil {
		response.DisplayScore = resp.Review.AIScore
		response.DisplayFeedback = resp.Review.AIFeedback
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetReview handles GET /api/v1/reviews/:answerId
// Accessible to reviewers and the test creator
func (h *ReviewHandler) HandleGetReview(w http.ResponseWriter, r *http.Request) {
	// Get answer ID from URL
	answerIDStr := chi.URLParam(r, "answerId")
	answerID, err := uuid.Parse(answerIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid answer ID")
		return
	}

	// Execute use case
	resp, err := h.getReview.Execute(r.Context(), review.GetReviewRequest{
		AnswerID: answerID,
	})
	if err != nil {
		h.logger.Error("failed to get review", "error", err)
		h.respondError(w, http.StatusNotFound, "review not found")
		return
	}

	// Build response
	response := ReviewResponse{
		ID:             resp.Review.ID.String(),
		AnswerID:       resp.Review.AnswerID.String(),
		AIScore:        resp.Review.AIScore,
		AIFeedback:     resp.Review.AIFeedback,
		ManualScore:    resp.Review.ManualScore,
		ManualFeedback: resp.Review.ManualFeedback,
	}

	if resp.Review.AIScoredAt != nil {
		scoredAt := resp.Review.AIScoredAt.Format("2006-01-02T15:04:05Z07:00")
		response.AIScoredAt = &scoredAt
	}

	if resp.Review.ManualScoredAt != nil {
		scoredAt := resp.Review.ManualScoredAt.Format("2006-01-02T15:04:05Z07:00")
		response.ManualScoredAt = &scoredAt
	}

	if resp.Review.ReviewerID != nil {
		reviewerID := resp.Review.ReviewerID.String()
		response.ReviewerID = &reviewerID
	}

	// T090: Set display fields
	if resp.Review.ManualScore != nil {
		response.DisplayScore = resp.Review.ManualScore
		response.DisplayFeedback = resp.Review.ManualFeedback
	} else if resp.Review.AIScore != nil {
		response.DisplayScore = resp.Review.AIScore
		response.DisplayFeedback = resp.Review.AIFeedback
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleListSubmissions handles GET /api/v1/tests/:testId/submissions
// Accessible to reviewers
func (h *ReviewHandler) HandleListSubmissions(w http.ResponseWriter, r *http.Request) {
	// Get test ID from URL
	testIDStr := chi.URLParam(r, "testId")
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	// Execute use case
	resp, err := h.listSubmissions.Execute(r.Context(), review.ListSubmissionsRequest{
		TestID: testID,
	})
	if err != nil {
		h.logger.Error("failed to list submissions", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to list submissions")
		return
	}

	// Build response
	submissions := make([]SubmissionResponse, len(resp.Submissions))
	for i, sub := range resp.Submissions {
		submissions[i] = SubmissionResponse{
			ID:               sub.ID,
			TestID:           sub.TestID,
			AccessEmail:      sub.AccessEmail,
			SubmittedAt:      sub.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"),
			AITotalScore:     sub.AITotalScore,
			ManualTotalScore: sub.ManualTotalScore,
		}
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"submissions": submissions,
	})
}

// respondJSON sends a JSON response
func (h *ReviewHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// respondError sends an error response
func (h *ReviewHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error("failed to encode error response", "error", err)
	}
}
