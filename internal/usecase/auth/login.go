package auth

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// PasswordComparer interface for password comparison
type PasswordComparer interface {
	Compare(hashedPassword, password string) error
}

// TokenGenerator interface for JWT token generation
type TokenGenerator interface {
	GenerateToken(userID uuid.UUID, email, role string) (string, error)
}

// LoginUserRequest holds the data for user login
type LoginUserRequest struct {
	Email    string
	Password string
}

// LoginUserResponse holds the login response data
type LoginUserResponse struct {
	User  *domain.User
	Token string
}

// LoginUserUseCase handles user login
type LoginUserUseCase struct {
	userRepo domain.UserRepository
	comparer PasswordComparer
	tokenGen TokenGenerator
	logger   *slog.Logger
}

// NewLoginUserUseCase creates a new LoginUser use case
func NewLoginUserUseCase(
	userRepo domain.UserRepository,
	comparer PasswordComparer,
	tokenGen TokenGenerator,
	logger *slog.Logger,
) *LoginUserUseCase {
	return &LoginUserUseCase{
		userRepo: userRepo,
		comparer: comparer,
		tokenGen: tokenGen,
		logger:   logger,
	}
}

// Execute authenticates a user and returns a JWT token
func (uc *LoginUserUseCase) Execute(ctx context.Context, req LoginUserRequest) (*LoginUserResponse, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Find user by email
	user, err := uc.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if user exists or not
		uc.logger.Warn("login attempt for non-existent user", "email", req.Email)
		return nil, domain.ErrUnauthorized{
			Message: "invalid email or password",
		}
	}

	// Compare password
	if err := uc.comparer.Compare(user.PasswordHash, req.Password); err != nil {
		uc.logger.Warn("failed login attempt", "user_id", user.ID, "email", req.Email)
		return nil, domain.ErrUnauthorized{
			Message: "invalid email or password",
		}
	}

	// Generate JWT token
	token, err := uc.tokenGen.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		uc.logger.Error("failed to generate token", "error", err, "user_id", user.ID)
		return nil, domain.ErrInternal{
			Message: "failed to generate authentication token",
			Err:     err,
		}
	}

	uc.logger.Info("user logged in successfully", "user_id", user.ID, "email", user.Email)

	return &LoginUserResponse{
		User:  user,
		Token: token,
	}, nil
}

// validateRequest validates the login request
func (uc *LoginUserUseCase) validateRequest(req LoginUserRequest) error {
	if req.Email == "" {
		return domain.ErrValidation{
			Field:   "email",
			Message: "email is required",
		}
	}

	if req.Password == "" {
		return domain.ErrValidation{
			Field:   "password",
			Message: "password is required",
		}
	}

	return nil
}
