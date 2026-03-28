package test

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// CreateTestRequest holds the data for creating a test
type CreateTestRequest struct {
	CreatorID    uuid.UUID
	Title        string
	Description  string
	AllowRetakes bool
}

// CreateTestUseCase handles test creation
type CreateTestUseCase struct {
	testRepo domain.TestRepository
	logger   *slog.Logger
}

// NewCreateTestUseCase creates a new CreateTest use case
func NewCreateTestUseCase(
	testRepo domain.TestRepository,
	logger *slog.Logger,
) *CreateTestUseCase {
	return &CreateTestUseCase{
		testRepo: testRepo,
		logger:   logger,
	}
}

// Execute creates a new test as a draft
func (uc *CreateTestUseCase) Execute(ctx context.Context, req CreateTestRequest) (*domain.Test, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Create test entity (draft by default)
	now := time.Now()
	test := &domain.Test{
		ID:           uuid.New(),
		CreatorID:    req.CreatorID,
		Title:        req.Title,
		Description:  req.Description,
		AllowRetakes: req.AllowRetakes,
		IsPublished:  false, // Start as draft
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate test entity
	if err := test.Validate(); err != nil {
		return nil, err
	}

	// Save test to database
	if err := uc.testRepo.Create(ctx, test); err != nil {
		uc.logger.Error("failed to create test", "error", err, "creator_id", req.CreatorID)
		return nil, err
	}

	uc.logger.Info("test created successfully", "test_id", test.ID, "creator_id", req.CreatorID, "title", test.Title)
	return test, nil
}

// validateRequest validates the create test request
func (uc *CreateTestUseCase) validateRequest(req CreateTestRequest) error {
	if req.Title == "" {
		return domain.ErrValidation{
			Field:   "title",
			Message: "test title is required",
		}
	}

	if len(req.Title) > 255 {
		return domain.ErrValidation{
			Field:   "title",
			Message: "test title must be less than 255 characters",
		}
	}

	return nil
}
