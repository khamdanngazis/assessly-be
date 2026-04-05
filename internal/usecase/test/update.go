package test

import (
	"context"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// UpdateTestRequest holds the data for updating a test
type UpdateTestRequest struct {
	TestID       uuid.UUID
	CreatorID    uuid.UUID
	Title        string
	Description  string
	AllowRetakes bool
}

// UpdateTestUseCase handles test updates
type UpdateTestUseCase struct {
	testRepo domain.TestRepository
	logger   *slog.Logger
}

// NewUpdateTestUseCase creates a new UpdateTest use case
func NewUpdateTestUseCase(
	testRepo domain.TestRepository,
	logger *slog.Logger,
) *UpdateTestUseCase {
	return &UpdateTestUseCase{
		testRepo: testRepo,
		logger:   logger,
	}
}

// Execute updates an existing test
func (uc *UpdateTestUseCase) Execute(ctx context.Context, req UpdateTestRequest) (*domain.Test, error) {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Find the test
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return nil, err
	}

	// Verify ownership
	if test.CreatorID != req.CreatorID {
		uc.logger.Warn("unauthorized test update attempt",
			"test_id", req.TestID,
			"creator_id", test.CreatorID,
			"requested_by", req.CreatorID)
		return nil, domain.ErrUnauthorized{
			Message: "you can only update your own tests",
		}
	}

	// Prevent updates to published tests (except allow_retakes)
	if test.IsPublished {
		uc.logger.Warn("attempt to update published test", "test_id", req.TestID)
		return nil, domain.ErrValidation{
			Message: "cannot update title or description of a published test",
		}
	}

	// Update fields
	test.Title = req.Title
	test.Description = req.Description
	test.AllowRetakes = req.AllowRetakes
	test.UpdatedAt = time.Now()

	// Validate updated test
	if err := test.Validate(); err != nil {
		return nil, err
	}

	// Save changes
	if err := uc.testRepo.Update(ctx, test); err != nil {
		uc.logger.Error("failed to update test", "error", err, "test_id", req.TestID)
		return nil, err
	}

	uc.logger.Info("test updated successfully", "test_id", test.ID, "title", test.Title)
	return test, nil
}

// validateRequest validates the update test request
func (uc *UpdateTestUseCase) validateRequest(req UpdateTestRequest) error {
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
