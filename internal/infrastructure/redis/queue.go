package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueueClient handles Redis Streams for async job processing
type QueueClient struct {
	client     *redis.Client
	streamName string
}

// Job represents a job in the queue
type Job struct {
	ID        string
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// NewQueueClient creates a new Redis queue client
func NewQueueClient(client *redis.Client, streamName string) *QueueClient {
	return &QueueClient{
		client:     client,
		streamName: streamName,
	}
}

// Enqueue adds a job to the Redis stream
func (q *QueueClient) Enqueue(ctx context.Context, jobType string, payload []byte) (string, error) {
	args := &redis.XAddArgs{
		Stream: q.streamName,
		Values: map[string]interface{}{
			"type":       jobType,
			"payload":    string(payload),
			"created_at": time.Now().Unix(),
		},
	}

	id, err := q.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	return id, nil
}

// Dequeue retrieves jobs from the Redis stream using consumer groups
func (q *QueueClient) Dequeue(ctx context.Context, streamName, groupName, consumerName string, count int64, block time.Duration) ([]Job, error) {
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: consumerName,
		Streams:  []string{streamName, ">"},
		Count:    count,
		Block:    block,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []Job{}, nil
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	var jobs []Job
	for _, stream := range streams {
		for _, message := range stream.Messages {
			jobType, _ := message.Values["type"].(string)
			payloadStr, _ := message.Values["payload"].(string)
			createdAtInt, _ := message.Values["created_at"].(string)

			var createdAt time.Time
			if createdAtInt != "" {
				var ts int64
				fmt.Sscanf(createdAtInt, "%d", &ts)
				createdAt = time.Unix(ts, 0)
			}

			jobs = append(jobs, Job{
				ID:        message.ID,
				Type:      jobType,
				Payload:   json.RawMessage(payloadStr),
				CreatedAt: createdAt,
			})
		}
	}

	return jobs, nil
}

// AckMessage acknowledges a message
func (q *QueueClient) AckMessage(ctx context.Context, streamName, groupName, messageID string) error {
	return q.client.XAck(ctx, streamName, groupName, messageID).Err()
}

// DeleteMessage deletes a message from the stream
func (q *QueueClient) DeleteMessage(ctx context.Context, streamName, messageID string) error {
	return q.client.XDel(ctx, streamName, messageID).Err()
}

// CreateConsumerGroup creates a consumer group for the stream
func (q *QueueClient) CreateConsumerGroup(ctx context.Context, streamName, groupName string) error {
	err := q.client.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}
