package auth

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
)

// ResetTokenGenerator interface for generating reset tokens
type ResetTokenGenerator interface {
	GenerateResetToken(userID string, email string) (string, error)
}

// EmailSender interface for sending emails
type EmailSender interface {
	SendPasswordReset(to, resetToken, resetURL string) error
}

// RequestPasswordResetRequest holds the data for password reset request
type RequestPasswordResetRequest struct {
	Email    string
	ResetURL string // Frontend URL where user will reset password
}

// RequestPasswordResetUseCase handles password reset requests
type RequestPasswordResetUseCase struct {
	userRepo  domain.UserRepository
	tokenGen  ResetTokenGenerator
	emailSender EmailSender
	logger    *slog.Logger
}

// NewRequestPasswordResetUseCase creates a new RequestPasswordReset use case
func NewRequestPasswordResetUseCase(
	userRepo domain.UserRepository,
	tokenGen ResetTokenGenerator,
	emailSender EmailSender,
	logger *slog.Logger,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{
		userRepo:  userRepo,
		tokenGen:  tokenGen,
		emailSender: emailSender,
		logger:    logger,
	}
}

// Execute sends a password reset email with a reset token
func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, req RequestPasswordResetRequest) error {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return err
	}

	// Find user by email
	user, err := uc.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// For security reasons, don't reveal if user exists
		// Log warning but return success to avoid email enumeration
		uc.logger.Warn("password reset requested for non-existent email", "email", req.Email)
		return nil
	}

	// Generate reset token
	resetToken, err := uc.tokenGen.GenerateResetToken(user.ID.String(), user.Email)
	if err != nil {
		uc.logger.Error("failed to generate reset token", "error", err, "user_id", user.ID)
		return domain.ErrInternal{
			Message: "failed to generate reset token",
			Err:     err,
		}
	}

	// Send reset email
	if err := uc.emailSender.SendPasswordReset(user.Email, resetToken, req.ResetURL); err != nil {
		uc.logger.Error("failed to send reset email", "error", err, "user_id", user.ID)
		return domain.ErrInternal{
			Message: "failed to send reset email",
			Err:     err,
		}
	}

	uc.logger.Info("password reset email sent", "user_id", user.ID, "email", user.Email)
	return nil
}

// validateRequest validates the reset request
func (uc *RequestPasswordResetUseCase) validateRequest(req RequestPasswordResetRequest) error {
	if req.Email == "" {
		return domain.ErrValidation{
			Field:   "email",
			Message: "email is required",
		}
	}

	if req.ResetURL == "" {
		return domain.ErrValidation{
			Field:   "reset_url",
			Message: "reset URL is required",
		}
	}

	return nil
}
