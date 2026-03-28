package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/assessly/assessly-be/internal/delivery/worker"
	"github.com/google/uuid"
)

// SubmissionQueueAdapter adapts QueueClient to scoring use case interface
type SubmissionQueueAdapter struct {
	queue *QueueClient
}

// NewSubmissionQueueAdapter creates a new adapter
func NewSubmissionQueueAdapter(queue *QueueClient) *SubmissionQueueAdapter {
	return &SubmissionQueueAdapter{queue: queue}
}

// Enqueue adds a submission ID to the scoring queue
func (a *SubmissionQueueAdapter) Enqueue(ctx context.Context, submissionID uuid.UUID) error {
	payload, err := json.Marshal(map[string]string{
		"submission_id": submissionID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = a.queue.Enqueue(ctx, "score_submission", payload)
	return err
}

// QueueConsumerAdapter adapts QueueClient to worker consumer interface
type QueueConsumerAdapter struct {
	queue      *QueueClient
	streamName string
}

// NewQueueConsumerAdapter creates a new adapter
func NewQueueConsumerAdapter(queue *QueueClient, streamName string) *QueueConsumerAdapter {
	return &QueueConsumerAdapter{
		queue:      queue,
		streamName: streamName,
	}
}

// Dequeue retrieves jobs from the queue
func (a *QueueConsumerAdapter) Dequeue(ctx context.Context, groupName, consumerName string, count int64, block time.Duration) ([]worker.Job, error) {
	jobs, err := a.queue.Dequeue(ctx, a.streamName, groupName, consumerName, count, block)
	if err != nil {
		return nil, err
	}

	workerJobs := make([]worker.Job, len(jobs))
	for i, job := range jobs {
		workerJobs[i] = worker.Job{
			ID:      job.ID,
			Type:    job.Type,
			Payload: job.Payload,
		}
	}

	return workerJobs, nil
}

// AckMessage acknowledges a processed message
func (a *QueueConsumerAdapter) AckMessage(ctx context.Context, streamName, groupName, messageID string) error {
	return a.queue.AckMessage(ctx, streamName, groupName, messageID)
}

// DeleteMessage deletes a message
func (a *QueueConsumerAdapter) DeleteMessage(ctx context.Context, streamName, messageID string) error {
	return a.queue.DeleteMessage(ctx, streamName, messageID)
}

// CreateConsumerGroup creates a consumer group
func (a *QueueConsumerAdapter) CreateConsumerGroup(ctx context.Context, streamName, groupName string) error {
	return a.queue.CreateConsumerGroup(ctx, streamName, groupName)
}

