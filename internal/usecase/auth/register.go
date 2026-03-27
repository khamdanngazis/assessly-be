package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// PasswordHasher interface for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// RegisterUserRequest holds the data for user registration
type RegisterUserRequest struct {
	Email    string
	Password string
	Role     domain.UserRole
}

// RegisterUserUseCase handles user registration
type RegisterUserUseCase struct {
	userRepo domain.UserRepository
	hasher   PasswordHasher
	logger   *slog.Logger
}

// NewRegisterUserUseCase creates a new RegisterUser use case
func NewRegisterUserUseCase(
	userRepo domain.UserRepository,
	hasher PasswordHasher,
	logger *slog.Logger,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepo: userRepo,
		hasher:   hasher,
		logger:   logger,
	}
}

// Execute registers a new user
func (uc *RegisterUserUseCase) Execute(ctx context.Context, req RegisterUserRequest) (*domain.User, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, err := uc.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Only proceed if error is NotFound
		if _, ok := err.(domain.ErrNotFound); !ok {
			uc.logger.Error("failed to check user existence", "error", err, "email", req.Email)
			return nil, domain.ErrInternal{
				Message: "failed to check user existence",
				Err:     err,
			}
		}
	}
	if existingUser != nil {
		return nil, domain.ErrConflict{
			Resource: "user",
			Message:  "email already registered",
		}
	}

	// Hash password
	hashedPassword, err := uc.hasher.Hash(req.Password)
	if err != nil {
		uc.logger.Error("failed to hash password", "error", err)
		return nil, domain.ErrInternal{
			Message: "failed to hash password",
			Err:     err,
		}
	}

	// Create user entity
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         req.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate user entity
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Save user to database
	if err := uc.userRepo.Create(ctx, user); err != nil {
		uc.logger.Error("failed to create user", "error", err, "email", req.Email)
		return nil, err
	}

	uc.logger.Info("user registered successfully", "user_id", user.ID, "email", user.Email, "role", user.Role)
	return user, nil
}

// validateRequest validates the registration request
func (uc *RegisterUserUseCase) validateRequest(req RegisterUserRequest) error {
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

	// Password strength validation
	if len(req.Password) < 8 {
		return domain.ErrValidation{
			Field:   "password",
			Message: "password must be at least 8 characters long",
		}
	}

	if req.Role != domain.RoleCreator && req.Role != domain.RoleReviewer {
		return domain.ErrValidation{
			Field:   "role",
			Message: fmt.Sprintf("role must be either '%s' or '%s'", domain.RoleCreator, domain.RoleReviewer),
		}
	}

	return nil
}
