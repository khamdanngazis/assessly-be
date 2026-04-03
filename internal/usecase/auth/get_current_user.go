package auth

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// GetCurrentUserRequest holds the request for getting current user
type GetCurrentUserRequest struct {
	UserID uuid.UUID
}

// GetCurrentUserUseCase handles getting current user information
type GetCurrentUserUseCase struct {
	userRepo domain.UserRepository
}

// NewGetCurrentUserUseCase creates a new GetCurrentUser use case
func NewGetCurrentUserUseCase(userRepo domain.UserRepository) *GetCurrentUserUseCase {
	return &GetCurrentUserUseCase{
		userRepo: userRepo,
	}
}

// Execute retrieves the current user information
func (uc *GetCurrentUserUseCase) Execute(ctx context.Context, req GetCurrentUserRequest) (*domain.User, error) {
	// Find user by ID
	user, err := uc.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
