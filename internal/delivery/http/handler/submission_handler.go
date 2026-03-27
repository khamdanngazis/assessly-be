package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/submission"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GenerateAccessTokenRequest represents the HTTP request body
type GenerateAccessTokenRequest struct {
	TestID      string `json:"test_id"`
	Email       string `json:"email"`
	AccessURL   string `json:"access_url"`
	ExpiryHours int    `json:"expiry_hours,omitempty"`
}

// SubmitTestRequest represents the HTTP request body
type SubmitTestRequest struct {
	AccessToken string         `json:"access_token"`
	Answers     []AnswerInput  `json:"answers"`
}

// AnswerInput represents an answer in the request
type AnswerInput struct {
	QuestionID string `json:"question_id"`
	Text       string `json:"text"`
}

// AnswerResponse represents the HTTP response for an answer
type AnswerResponse struct {
	ID           string          `json:"id"`
	SubmissionID string          `json:"submission_id"`
	QuestionID   string          `json:"question_id"`
	Text         string          `json:"text"`
	CreatedAt    string          `json:"created_at"`
	Review       *ReviewResponse `json:"review,omitempty"`
}

// GetSubmissionResponse represents the HTTP response for get submission
type GetSubmissionResponse struct {
	Submission SubmissionResponse `json:"submission"`
	Answers    []AnswerResponse   `json:"answers"`
}

// Handler handles submission HTTP requests
type Handler struct {
	generateAccessTokenUC *submission.GenerateAccessTokenUseCase
	submitTestUC          *submission.SubmitTestUseCase
	getSubmissionUC       *submission.GetSubmissionUseCase
	logger                *slog.Logger
}

// NewSubmissionHandler creates a new submission handler
func NewSubmissionHandler(
	generateAccessTokenUC *submission.GenerateAccessTokenUseCase,
	submitTestUC *submission.SubmitTestUseCase,
	getSubmissionUC *submission.GetSubmissionUseCase,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		generateAccessTokenUC: generateAccessTokenUC,
		submitTestUC:          submitTestUC,
		getSubmissionUC:       getSubmissionUC,
		logger:                logger,
	}
}

// GenerateAccessToken handles POST /api/v1/submissions/access
func (h *Handler) GenerateAccessToken(w http.ResponseWriter, r *http.Request) {
	var req GenerateAccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse test ID
	testID, err := uuid.Parse(req.TestID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid test ID")
		return
	}

	// Execute use case
	ucReq := submission.GenerateAccessTokenRequest{
		TestID:      testID,
		Email:       req.Email,
		AccessURL:   req.AccessURL,
		ExpiryHours: req.ExpiryHours,
	}

	if err := h.generateAccessTokenUC.Execute(r.Context(), ucReq); err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"message": "access token sent successfully",
	})
}

// SubmitTest handles POST /api/v1/submissions
func (h *Handler) SubmitTest(w http.ResponseWriter, r *http.Request) {
	var req SubmitTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert answers
	answers := make([]submission.AnswerInput, len(req.Answers))
	for i, ans := range req.Answers {
		questionID, err := uuid.Parse(ans.QuestionID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid question ID")
			return
		}
		answers[i] = submission.AnswerInput{
			QuestionID: questionID,
			Text:       ans.Text,
		}
	}

	// Execute use case
	ucReq := submission.SubmitTestRequest{
		AccessToken: req.AccessToken,
		Answers:     answers,
	}

	sub, err := h.submitTestUC.Execute(r.Context(), ucReq)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	// Convert submission to response
	response := SubmissionResponse{
		ID:               sub.ID,
		TestID:           sub.TestID,
		AccessEmail:      sub.AccessEmail,
		SubmittedAt:      sub.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"),
		AITotalScore:     sub.AITotalScore,
		ManualTotalScore: sub.ManualTotalScore,
	}

	h.respondJSON(w, http.StatusCreated, response)
}

// GetSubmission handles GET /api/v1/submissions/:id
func (h *Handler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	// Get submission ID from URL
	submissionIDStr := chi.URLParam(r, "id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid submission ID")
		return
	}

	// Get access token from query params or header
	accessToken := r.URL.Query().Get("token")
	if accessToken == "" {
		accessToken = r.Header.Get("X-Access-Token")
	}

	// Get user ID from context (if authenticated)
	var userID *uuid.UUID
	if userIDValue := r.Context().Value("user_id"); userIDValue != nil {
		if uid, ok := userIDValue.(uuid.UUID); ok {
			userID = &uid
		}
	}

	// Execute use case
	ucReq := submission.GetSubmissionRequest{
		SubmissionID: submissionID,
		AccessToken:  accessToken,
		UserID:       userID,
	}

	result, err := h.getSubmissionUC.Execute(r.Context(), ucReq)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	// Convert to response
	response := GetSubmissionResponse{
		Submission: SubmissionResponse{
			ID:               result.Submission.ID,
			TestID:           result.Submission.TestID,
			AccessEmail:      result.Submission.AccessEmail,
			SubmittedAt:      result.Submission.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"),
			AITotalScore:     result.Submission.AITotalScore,
			ManualTotalScore: result.Submission.ManualTotalScore,
		},
		Answers: make([]AnswerResponse, len(result.Answers)),
	}

	// T090: Set display score - prioritize manual over AI
	if result.Submission.ManualTotalScore != nil {
		response.Submission.DisplayScore = result.Submission.ManualTotalScore
	} else if result.Submission.AITotalScore != nil {
		response.Submission.DisplayScore = result.Submission.AITotalScore
	}

	for i, ansWithReview := range result.Answers {
		ansResp := AnswerResponse{
			ID:           ansWithReview.Answer.ID.String(),
			SubmissionID: ansWithReview.Answer.SubmissionID.String(),
			QuestionID:   ansWithReview.Answer.QuestionID.String(),
			Text:         ansWithReview.Answer.Text,
			CreatedAt:    ansWithReview.Answer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add review if exists
		if ansWithReview.Review != nil {
			review := ansWithReview.Review
			reviewResp := &ReviewResponse{
				ID:         review.ID.String(),
				AIScore:    review.AIScore,
				AIFeedback: review.AIFeedback,
			}

			if review.AIScoredAt != nil {
				scoredAt := review.AIScoredAt.Format("2006-01-02T15:04:05Z07:00")
				reviewResp.AIScoredAt = &scoredAt
			}

			reviewResp.ManualScore = review.ManualScore
			reviewResp.ManualFeedback = review.ManualFeedback

			if review.ManualScoredAt != nil {
				scoredAt := review.ManualScoredAt.Format("2006-01-02T15:04:05Z07:00")
				reviewResp.ManualScoredAt = &scoredAt
			}

			if review.ReviewerID != nil {
				reviewerID := review.ReviewerID.String()
				reviewResp.ReviewerID = &reviewerID
			}

			// T090: Set display fields - prioritize manual score over AI score
			if review.ManualScore != nil {
				reviewResp.DisplayScore = review.ManualScore
				reviewResp.DisplayFeedback = review.ManualFeedback
			} else if review.AIScore != nil {
				reviewResp.DisplayScore = review.AIScore
				reviewResp.DisplayFeedback = review.AIFeedback
			}

			ansResp.Review = reviewResp
		}

		response.Answers[i] = ansResp
	}

	h.respondJSON(w, http.StatusOK, response)
}

// handleUseCaseError maps use case errors to HTTP responses
func (h *Handler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case domain.ErrValidation:
		h.respondError(w, http.StatusBadRequest, e.Message)
	case domain.ErrNotFound:
		h.respondError(w, http.StatusNotFound, e.Error())
	case domain.ErrUnauthorized:
		h.respondError(w, http.StatusUnauthorized, e.Message)
	case domain.ErrConflict:
		h.respondError(w, http.StatusConflict, e.Message)
	default:
		h.logger.Error("internal error", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
