package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/infrastructure/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps pgxpool.Pool for database operations
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// Configure connection pool
	if cfg.Database.MaxConnections > 0 && cfg.Database.MaxConnections <= 2147483647 {
		poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	} else {
		poolConfig.MaxConns = 100 // default safe value
	}
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.Info("database connection established",
		"host", cfg.Database.Host,
		"database", cfg.Database.Name,
		"max_connections", cfg.Database.MaxConnections,
	)

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		slog.Info("database connection closed")
	}
}

// Health checks if database is healthy
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// isPgUniqueViolation checks if the error is a PostgreSQL unique constraint violation
func isPgUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 = unique_violation (SQLSTATE code)
		return pgErr.Code == "23505"
	}
	return false
}
