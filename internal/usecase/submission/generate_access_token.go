package submission

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// AccessTokenGenerator interface for generating access tokens
type AccessTokenGenerator interface {
	GenerateAccessToken(testID uuid.UUID, email string, expiryHours int) (string, error)
}

// EmailSender interface for sending emails
type EmailSender interface {
	SendTestAccessToken(to, testTitle, accessToken, accessURL string) error
}

// GenerateAccessTokenRequest holds the data for generating an access token
type GenerateAccessTokenRequest struct {
	TestID     uuid.UUID
	Email      string
	AccessURL  string // Frontend URL where participant will take the test
	ExpiryHours int    // Token expiry in hours (default: 24)
}

// GenerateAccessTokenUseCase handles generating access tokens for anonymous participants
type GenerateAccessTokenUseCase struct {
	testRepo     domain.TestRepository
	tokenGen     AccessTokenGenerator
	emailSender  EmailSender
	logger       *slog.Logger
}

// NewGenerateAccessTokenUseCase creates a new GenerateAccessToken use case
func NewGenerateAccessTokenUseCase(
	testRepo domain.TestRepository,
	tokenGen AccessTokenGenerator,
	emailSender EmailSender,
	logger *slog.Logger,
) *GenerateAccessTokenUseCase {
	return &GenerateAccessTokenUseCase{
		testRepo:    testRepo,
		tokenGen:    tokenGen,
		emailSender: emailSender,
		logger:      logger,
	}
}

// Execute generates an access token and sends it via email
func (uc *GenerateAccessTokenUseCase) Execute(ctx context.Context, req GenerateAccessTokenRequest) error {
	// Validate input
	if err := uc.validateRequest(req); err != nil {
		return err
	}

	// Find test
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		uc.logger.Warn("test not found", "test_id", req.TestID)
		return err
	}

	// Check if test is published
	if !test.IsPublished {
		return domain.ErrValidation{
			Field:   "test_id",
			Message: "test is not published",
		}
	}

	// Set default expiry if not provided
	expiryHours := req.ExpiryHours
	if expiryHours == 0 {
		expiryHours = 24 // Default 24 hours
	}

	// Generate access token
	accessToken, err := uc.tokenGen.GenerateAccessToken(req.TestID, req.Email, expiryHours)
	if err != nil {
		uc.logger.Error("failed to generate access token", "error", err, "test_id", req.TestID)
		return domain.ErrInternal{
			Message: "failed to generate access token",
			Err:     err,
		}
	}

	// Send access token via email
	if err := uc.emailSender.SendTestAccessToken(req.Email, test.Title, accessToken, req.AccessURL); err != nil {
		uc.logger.Error("failed to send access token email", "error", err, "email", req.Email)
		return domain.ErrInternal{
			Message: "failed to send access token email",
			Err:     err,
		}
	}

	uc.logger.Info("access token generated and sent", "test_id", req.TestID, "email", req.Email)
	return nil
}

// validateRequest validates the generate access token request
func (uc *GenerateAccessTokenUseCase) validateRequest(req GenerateAccessTokenRequest) error {
	if req.Email == "" {
		return domain.ErrValidation{
			Field:   "email",
			Message: "email is required",
		}
	}

	if req.AccessURL == "" {
		return domain.ErrValidation{
			Field:   "access_url",
			Message: "access URL is required",
		}
	}

	return nil
}
