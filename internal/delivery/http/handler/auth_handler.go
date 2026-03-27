package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/auth"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	registerUC      *auth.RegisterUserUseCase
	loginUC         *auth.LoginUserUseCase
	requestResetUC  *auth.RequestPasswordResetUseCase
	resetPasswordUC *auth.ResetPasswordUseCase
	logger          *slog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	registerUC *auth.RegisterUserUseCase,
	loginUC *auth.LoginUserUseCase,
	requestResetUC *auth.RequestPasswordResetUseCase,
	resetPasswordUC *auth.ResetPasswordUseCase,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUC:      registerUC,
		loginUC:         loginUC,
		requestResetUC:  requestResetUC,
		resetPasswordUC: resetPasswordUC,
		logger:          logger,
	}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert role string to domain type
	var role domain.UserRole
	switch req.Role {
	case "creator":
		role = domain.RoleCreator
	case "reviewer":
		role = domain.RoleReviewer
	default:
		h.respondError(w, http.StatusBadRequest, "role must be 'creator' or 'reviewer'")
		return
	}

	// Execute use case
	user, err := h.registerUC.Execute(r.Context(), auth.RegisterUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Role:     role,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with created user (without password)
	resp := RegisterResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string         `json:"token"`
	User  UserResponse   `json:"user"`
}

// UserResponse represents user data in responses
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	result, err := h.loginUC.Execute(r.Context(), auth.LoginUserRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Respond with token and user data
	resp := LoginResponse{
		Token: result.Token,
		User: UserResponse{
			ID:    result.User.ID.String(),
			Email: result.User.Email,
			Role:  string(result.User.Role),
		},
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// RequestResetRequest represents the password reset request body
type RequestResetRequest struct {
	Email string `json:"email"`
}

// RequestResetResponse represents the password reset response
type RequestResetResponse struct {
	Message string `json:"message"`
}

// RequestPasswordReset handles password reset requests
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req RequestResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get reset URL from request header or use default
	resetURL := r.Header.Get("X-Reset-URL")
	if resetURL == "" {
		resetURL = "http://localhost:3000/reset-password" // Default frontend URL
	}

	// Execute use case
	err := h.requestResetUC.Execute(r.Context(), auth.RequestPasswordResetRequest{
		Email:    req.Email,
		ResetURL: resetURL,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Always return success to avoid email enumeration
	resp := RequestResetResponse{
		Message: "If the email exists, a password reset link has been sent",
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// ResetPasswordRequest represents the password reset request body
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPasswordResponse represents the password reset response
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPassword handles password reset with token
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	err := h.resetPasswordUC.Execute(r.Context(), auth.ResetPasswordRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	resp := ResetPasswordResponse{
		Message: "Password reset successfully",
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// handleError handles domain errors and converts them to HTTP responses
func (h *AuthHandler) handleError(w http.ResponseWriter, err error) {
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
func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// respondError writes an error JSON response
func (h *AuthHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
