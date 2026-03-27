package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/assessly/assessly-be/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client for Redis operations
type Client struct {
	Redis *redis.Client
}

// New creates a new Redis client
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis: %w", err)
	}

	slog.Info("Redis connection established",
		"addr", cfg.RedisAddr(),
		"db", cfg.Redis.DB,
	)

	return &Client{Redis: client}, nil
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	if c.Redis != nil {
		err := c.Redis.Close()
		if err == nil {
			slog.Info("Redis connection closed")
		}
		return err
	}
	return nil
}

// Health checks if Redis is healthy
func (c *Client) Health(ctx context.Context) error {
	return c.Redis.Ping(ctx).Err()
}
