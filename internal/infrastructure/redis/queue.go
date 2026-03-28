package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueueClient handles Redis Streams for async job processing
type QueueClient struct {
	client     *redis.Client
	streamName string
	logger     *slog.Logger
}

// Job represents a job in the queue
type Job struct {
	ID        string
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// NewQueueClient creates a new Redis queue client
func NewQueueClient(client *redis.Client, streamName string, logger *slog.Logger) *QueueClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueueClient{
		client:     client,
		streamName: streamName,
		logger:     logger,
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
			// Extract job type (handle string or []byte)
			jobType := q.extractString(message.Values["type"], message.ID, "type")
			
			// Extract payload (handle string or []byte)
			payloadStr := q.extractString(message.Values["payload"], message.ID, "payload")
			
			// Extract created_at timestamp (handle int64, string, or []byte)
			createdAt := q.extractTimestamp(message.Values["created_at"], message.ID)

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

// extractString safely extracts a string value from Redis message values
// Handles string, []byte, and other types with logging
func (q *QueueClient) extractString(value interface{}, messageID, fieldName string) string {
	if value == nil {
		q.logger.Warn("redis message field is nil",
			"message_id", messageID,
			"field", fieldName)
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		q.logger.Error("unexpected type for redis message field",
			"message_id", messageID,
			"field", fieldName,
			"type", fmt.Sprintf("%T", v),
			"value", v)
		return fmt.Sprintf("%v", v) // Fallback to string conversion
	}
}

// extractTimestamp safely extracts a Unix timestamp from Redis message values
// Handles int64, string, []byte with proper type conversion and logging
func (q *QueueClient) extractTimestamp(value interface{}, messageID string) time.Time {
	if value == nil {
		q.logger.Warn("redis message created_at field is nil",
			"message_id", messageID)
		return time.Time{} // Zero time
	}

	var ts int64
	var err error

	switch v := value.(type) {
	case int64:
		// Direct int64 from Redis
		ts = v
	case string:
		// String representation of integer
		ts, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			q.logger.Error("failed to parse created_at string as int64",
				"message_id", messageID,
				"value", v,
				"error", err)
			return time.Time{}
		}
	case []byte:
		// Byte slice representation
		ts, err = strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			q.logger.Error("failed to parse created_at bytes as int64",
				"message_id", messageID,
				"value", string(v),
				"error", err)
			return time.Time{}
		}
	case int:
		// int type (less common but possible)
		ts = int64(v)
	default:
		q.logger.Error("unexpected type for created_at field",
			"message_id", messageID,
			"type", fmt.Sprintf("%T", v),
			"value", v)
		return time.Time{}
	}

	return time.Unix(ts, 0)
}
