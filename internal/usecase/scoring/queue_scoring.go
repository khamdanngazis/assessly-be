package scoring

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// QueueEnqueuer interface for enqueuing jobs
type QueueEnqueuer interface {
	Enqueue(ctx context.Context, submissionID uuid.UUID) error
}

// QueueScoringRequest holds the data for queuing a scoring job
type QueueScoringRequest struct {
	SubmissionID uuid.UUID
}

// QueueAIScoringUseCase handles queuing submissions for AI scoring
type QueueAIScoringUseCase struct {
	queue  QueueEnqueuer
	logger *slog.Logger
}

// NewQueueAIScoringUseCase creates a new QueueAIScoring use case
func NewQueueAIScoringUseCase(
	queue QueueEnqueuer,
	logger *slog.Logger,
) *QueueAIScoringUseCase {
	return &QueueAIScoringUseCase{
		queue:  queue,
		logger: logger,
	}
}

// Execute enqueues a submission for AI scoring
func (uc *QueueAIScoringUseCase) Execute(ctx context.Context, req QueueScoringRequest) error {
	// Enqueue the submission ID into Redis
	if err := uc.queue.Enqueue(ctx, req.SubmissionID); err != nil {
		uc.logger.Error("failed to enqueue submission for AI scoring",
			"submission_id", req.SubmissionID,
			"error", err,
		)
		return err
	}

	uc.logger.Info("submission queued for AI scoring", "submission_id", req.SubmissionID)
	return nil
}
