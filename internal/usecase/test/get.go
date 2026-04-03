package test

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// GetTestRequest represents the get test request
type GetTestRequest struct {
	TestID   uuid.UUID
	UserID   uuid.UUID
	UserRole string // "creator" or "reviewer"
}

// GetTestResponse represents the get test response
type GetTestResponse struct {
	Test      *domain.Test
	Questions []*domain.Question
}

// GetTestUseCase handles getting a single test by ID
type GetTestUseCase struct {
	testRepo     domain.TestRepository
	questionRepo domain.QuestionRepository
}

// NewGetTestUseCase creates a new get test use case
func NewGetTestUseCase(testRepo domain.TestRepository, questionRepo domain.QuestionRepository) *GetTestUseCase {
	return &GetTestUseCase{
		testRepo:     testRepo,
		questionRepo: questionRepo,
	}
}

// Execute retrieves a test by ID with authorization
func (uc *GetTestUseCase) Execute(ctx context.Context, req GetTestRequest) (*GetTestResponse, error) {
	// Find the test
	test, err := uc.testRepo.FindByID(ctx, req.TestID)
	if err != nil {
		return nil, err
	}

	// Authorization: creators can only see their own tests, reviewers can see published tests
	switch req.UserRole {
	case "creator":
		// Creator can only access their own tests
		if test.CreatorID != req.UserID {
			return nil, domain.ErrUnauthorized{Message: "you can only access your own tests"}
		}
	case "reviewer":
		// Reviewer can only access published tests
		if !test.IsPublished {
			return nil, domain.ErrUnauthorized{Message: "reviewers can only access published tests"}
		}
	default:
		return nil, domain.ErrUnauthorized{Message: "insufficient permissions"}
	}

	// Fetch questions for the test
	questions, err := uc.questionRepo.FindByTestID(ctx, test.ID)
	if err != nil {
		// If questions fetch fails, return empty questions array
		questions = []*domain.Question{}
	}

	return &GetTestResponse{
		Test:      test,
		Questions: questions,
	}, nil
}
