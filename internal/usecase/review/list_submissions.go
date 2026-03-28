package review

import (
	"context"
	"log/slog"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
)

// ListSubmissionsRequest holds the filter criteria
type ListSubmissionsRequest struct {
	TestID uuid.UUID
}

// ListSubmissionsResponse contains the list of submissions
type ListSubmissionsResponse struct {
	Submissions []*domain.Submission
}

// ListSubmissionsUseCase handles listing submissions for reviewers
type ListSubmissionsUseCase struct {
	submissionRepo domain.SubmissionRepository
	logger         *slog.Logger
}

// NewListSubmissionsUseCase creates a new ListSubmissions use case
func NewListSubmissionsUseCase(
	submissionRepo domain.SubmissionRepository,
	logger *slog.Logger,
) *ListSubmissionsUseCase {
	return &ListSubmissionsUseCase{
		submissionRepo: submissionRepo,
		logger:         logger,
	}
}

// Execute retrieves all submissions for a test (accessible to reviewers)
func (uc *ListSubmissionsUseCase) Execute(ctx context.Context, req ListSubmissionsRequest) (*ListSubmissionsResponse, error) {
	// Get all submissions for the test (no pagination for now - get all)
	submissions, err := uc.submissionRepo.FindByTestID(ctx, req.TestID, 1000, 0)
	if err != nil {
		uc.logger.Error("failed to list submissions", "test_id", req.TestID, "error", err)
		return nil, err
	}

	uc.logger.Info("submissions listed successfully",
		"test_id", req.TestID,
		"count", len(submissions),
	)

	return &ListSubmissionsResponse{
		Submissions: submissions,
	}, nil
}
