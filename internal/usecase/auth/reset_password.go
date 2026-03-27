package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// TokenValidator interface for validating tokens
type TokenValidator interface {
	ValidateToken(tokenString string) (userID string, email string, role string, err error)
}

// ResetPasswordRequest holds the data for password reset
type ResetPasswordRequest struct {
	Token       string
	NewPassword string
}

// ResetPasswordUseCase handles password reset
type ResetPasswordUseCase struct {
	userRepo domain.UserRepository
	validator TokenValidator
	hasher   PasswordHasher
	logger   *slog.Logger
}

// NewResetPasswordUseCase creates a new ResetPassword use case
func NewResetPasswordUseCase(
	userRepo domain.UserRepository,
	validator TokenValidator,
	hasher PasswordHasher,
	logger *slog.Logger,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:  userRepo,
		validator: validator,
		hasher:    hasher,
		logger:    logger,
	}
}

// Execute resets a user's password using a reset token
func (uc *ResetPasswordUseCase) Execute(ctx context.Context, req ResetPasswordRequest) error {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return err
	}

	// Validate reset token
	userIDStr, email, role, err := uc.validator.ValidateToken(req.Token)
	if err != nil {
		uc.logger.Warn("invalid reset token", "error", err)
		return domain.ErrUnauthorized{
			Message: "invalid or expired reset token",
		}
	}

	// Check if role is "reset" (special role for reset tokens)
	if role != "reset" {
		uc.logger.Warn("token is not a reset token", "role", role)
		return domain.ErrUnauthorized{
			Message: "invalid reset token",
		}
	}

	// Parse user ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		uc.logger.Error("invalid user ID in token", "error", err, "user_id", userIDStr)
		return domain.ErrUnauthorized{
			Message: "invalid reset token",
		}
	}

	// Find user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		uc.logger.Error("user not found", "error", err, "user_id", userID)
		return domain.ErrUnauthorized{
			Message: "invalid reset token",
		}
	}

	// Verify email matches
	if user.Email != email {
		uc.logger.Warn("email mismatch in reset token", "token_email", email, "user_email", user.Email)
		return domain.ErrUnauthorized{
			Message: "invalid reset token",
		}
	}

	// Hash new password
	hashedPassword, err := uc.hasher.Hash(req.NewPassword)
	if err != nil {
		uc.logger.Error("failed to hash new password", "error", err, "user_id", userID)
		return domain.ErrInternal{
			Message: "failed to update password",
			Err:     err,
		}
	}

	// Update user password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.Error("failed to update user password", "error", err, "user_id", userID)
		return domain.ErrInternal{
			Message: "failed to update password",
			Err:     err,
		}
	}

	uc.logger.Info("password reset successful", "user_id", userID, "email", user.Email)
	return nil
}

// validateRequest validates the reset password request
func (uc *ResetPasswordUseCase) validateRequest(req ResetPasswordRequest) error {
	if req.Token == "" {
		return domain.ErrValidation{
			Field:   "token",
			Message: "reset token is required",
		}
	}

	if req.NewPassword == "" {
		return domain.ErrValidation{
			Field:   "new_password",
			Message: "new password is required",
		}
	}

	// Password strength validation
	if len(req.NewPassword) < 8 {
		return domain.ErrValidation{
			Field:   "new_password",
			Message: "password must be at least 8 characters long",
		}
	}

	return nil
}
