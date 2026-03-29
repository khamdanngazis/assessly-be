package test

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// ListTestsRequest represents the list tests request
type ListTestsRequest struct {
	UserID   uuid.UUID
	UserRole string
	Status   string // "draft", "published", "all"
	Page     int
	PageSize int
}

// ListTestsResponse represents the list tests response
type ListTestsResponse struct {
	Tests []*domain.Test
	Total int
}

// ListTestsUseCase handles listing tests
type ListTestsUseCase struct {
	testRepo domain.TestRepository
}

// NewListTestsUseCase creates a new list tests use case
func NewListTestsUseCase(testRepo domain.TestRepository) *ListTestsUseCase {
	return &ListTestsUseCase{
		testRepo: testRepo,
	}
}

// Execute lists tests based on user role
func (uc *ListTestsUseCase) Execute(ctx context.Context, req ListTestsRequest) (*ListTestsResponse, error) {
	// Set default pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	offset := (req.Page - 1) * req.PageSize

	var tests []*domain.Test
	var err error

	// Role-based filtering
	switch req.UserRole {
	case "creator":
		// Creators see only their own tests
		tests, err = uc.testRepo.FindByCreatorID(ctx, req.UserID, req.PageSize, offset)
		if err != nil {
			return nil, err
		}

		// Filter by status if requested
		if req.Status == "published" {
			published := make([]*domain.Test, 0)
			for _, t := range tests {
				if t.IsPublished {
					published = append(published, t)
				}
			}
			tests = published
		} else if req.Status == "draft" {
			drafts := make([]*domain.Test, 0)
			for _, t := range tests {
				if !t.IsPublished {
					drafts = append(drafts, t)
				}
			}
			tests = drafts
		}

	case "reviewer":
		// Reviewers see all published tests
		tests, err = uc.testRepo.FindPublished(ctx, req.PageSize, offset)
		if err != nil {
			return nil, err
		}

	default:
		return nil, domain.ErrUnauthorized{
			Message: "insufficient permissions to list tests",
		}
	}

	return &ListTestsResponse{
		Tests: tests,
		Total: len(tests),
	}, nil
}
