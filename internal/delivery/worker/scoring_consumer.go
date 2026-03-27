package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/infrastructure/metrics"
	"github.com/assessly/assessly-be/internal/usecase/scoring"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Job represents a scoring job from the queue
type Job struct {
	ID      string
	Type    string
	Payload json.RawMessage
}

// QueueConsumer interface for consuming jobs from queue
type QueueConsumer interface {
	Dequeue(ctx context.Context, groupName, consumerName string, count int64, block time.Duration) ([]Job, error)
	AckMessage(ctx context.Context, streamName, groupName, messageID string) error
	DeleteMessage(ctx context.Context, streamName, messageID string) error
	CreateConsumerGroup(ctx context.Context, streamName, groupName string) error
}

// ScoringConsumer handles consuming and processing scoring jobs
type ScoringConsumer struct {
	queue          QueueConsumer
	scoreWithAI    *scoring.ScoreWithAIUseCase
	streamName     string
	groupName      string
	consumerName   string
	batchSize      int64
	blockDuration  time.Duration
	logger         *slog.Logger
}

// NewScoringConsumer creates a new scoring consumer
func NewScoringConsumer(
	queue QueueConsumer,
	scoreWithAI *scoring.ScoreWithAIUseCase,
	streamName, groupName, consumerName string,
	logger *slog.Logger,
) *ScoringConsumer {
	return &ScoringConsumer{
		queue:         queue,
		scoreWithAI:   scoreWithAI,
		streamName:    streamName,
		groupName:     groupName,
		consumerName:  consumerName,
		batchSize:     10, // Process up to 10 jobs at a time
		blockDuration: 5 * time.Second,
		logger:        logger,
	}
}

// Start begins consuming jobs from the queue
func (c *ScoringConsumer) Start(ctx context.Context) error {
	c.logger.Info("starting scoring consumer",
		"stream", c.streamName,
		"group", c.groupName,
		"consumer", c.consumerName,
	)

	// Create consumer group if it doesn't exist
	if err := c.queue.CreateConsumerGroup(ctx, c.streamName, c.groupName); err != nil {
		// Ignore error if group already exists
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			c.logger.Error("failed to create consumer group", "error", err)
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped by context")
			return ctx.Err()
		default:
			if err := c.processJobs(ctx); err != nil {
				c.logger.Error("error processing jobs", "error", err)
				// Continue processing even if there's an error
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processJobs dequeues and processes a batch of jobs
// T119: Records queue depth metrics
func (c *ScoringConsumer) processJobs(ctx context.Context) error {
	jobs, err := c.queue.Dequeue(ctx, c.groupName, c.consumerName, c.batchSize, c.blockDuration)
	if err != nil {
		// If it's a context error, return it
		if err == context.Canceled || err == context.DeadlineExceeded {
			return err
		}
		// For other errors (like no messages), just log and continue
		if err != redis.Nil {
			c.logger.Debug("dequeue error", "error", err)
		}
		return nil
	}

	// T119: Update queue depth metric (approximate)
	metrics.RedisQueueDepth.Set(float64(len(jobs)))

	for _, job := range jobs {
		if err := c.processJob(ctx, job); err != nil {
			c.logger.Error("failed to process job",
				"job_id", job.ID,
				"job_type", job.Type,
				"error", err,
			)
			// T119: Record queue processing error
			metrics.RedisQueueErrorsTotal.Inc()
			// Don't acknowledge failed jobs - they'll be redelivered
			continue
		}

		// T119: Record successful processing
		metrics.RedisQueueProcessedTotal.Inc()

		// Acknowledge successful processing
		if err := c.queue.AckMessage(ctx, c.streamName, c.groupName, job.ID); err != nil {
			c.logger.Error("failed to acknowledge message", "job_id", job.ID, "error", err)
		}

		// Delete the message after acknowledgment
		if err := c.queue.DeleteMessage(ctx, c.streamName, job.ID); err != nil {
			c.logger.Warn("failed to delete message", "job_id", job.ID, "error", err)
		}
	}

	return nil
}

// processJob processes a single scoring job
func (c *ScoringConsumer) processJob(ctx context.Context, job Job) error {
	c.logger.Info("processing job", "job_id", job.ID, "job_type", job.Type)

	switch job.Type {
	case "score_submission":
		return c.handleScoringJob(ctx, job)
	default:
		c.logger.Warn("unknown job type", "job_type", job.Type)
		// Acknowledge unknown job types so they don't get stuck
		return nil
	}
}

// handleScoringJob handles a submission scoring job
func (c *ScoringConsumer) handleScoringJob(ctx context.Context, job Job) error {
	var payload struct {
		SubmissionID string `json:"submission_id"`
	}

	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	submissionID, err := uuid.Parse(payload.SubmissionID)
	if err != nil {
		return fmt.Errorf("invalid submission ID: %w", err)
	}

	// Execute the scoring use case
	req := scoring.ScoreWithAIRequest{
		SubmissionID: submissionID,
	}

	if err := c.scoreWithAI.Execute(ctx, req); err != nil {
		return fmt.Errorf("failed to score submission: %w", err)
	}

	c.logger.Info("submission scored successfully",
		"job_id", job.ID,
		"submission_id", submissionID,
	)

	return nil
}
