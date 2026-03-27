package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// loadTestEnv loads the .env.test file for integration tests
func loadTestEnv(t *testing.T) {
	t.Helper()
	
	// Load .env.test from project root
	envPath := filepath.Join("..", "..", ".env.test")
	if err := godotenv.Load(envPath); err != nil {
		t.Logf("Warning: .env.test not found, using default values: %v", err)
	}
}

// getTestDBConfig returns database connection string from environment
func getTestDBConfig() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "12345abc")
	dbname := getEnv("DB_NAME", "assessly_test")
	sslmode := getEnv("DB_SSL_MODE", "disable")
	
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
}

// getPostgresConfig returns connection to postgres database (for creating test db)
func getPostgresConfig() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "12345abc")
	sslmode := getEnv("DB_SSL_MODE", "disable")
	
	return fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s",
		user, password, host, port, sslmode)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// setupTestDatabase creates test database and runs migrations
func setupTestDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	
	dbName := getEnv("DB_NAME", "assessly_test")
	
	// Connect to postgres database to create test database
	postgresConn := getPostgresConfig()
	pool, err := pgxpool.New(ctx, postgresConn)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer pool.Close()
	
	// Try to create test database (ignore error if exists)
	_, err = pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("Note: Could not create database (may already exist): %v", err)
	}
	
	// Connect to test database
	testConn := getTestDBConfig()
	testPool, err := pgxpool.New(ctx, testConn)
	require.NoError(t, err, "Failed to connect to test database")
	
	// Verify connection
	err = testPool.Ping(ctx)
	require.NoError(t, err, "Failed to ping test database")
	
	// Run migrations
	runMigrations(t, ctx, testPool)
	
	t.Logf("Test database '%s' ready", dbName)
	return testPool
}

// runMigrations applies database migrations
func runMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	migrationsDir := filepath.Join("..", "..", "migrations")
	migrations := []string{
		"000001_create_users.up.sql",
		"000002_create_tests.up.sql",
		"000003_create_questions.up.sql",
		"000004_create_submissions.up.sql",
		"000005_create_answers.up.sql",
		"000006_create_reviews.up.sql",
	}

	for _, migration := range migrations {
		filePath := filepath.Join(migrationsDir, migration)
		content, err := os.ReadFile(filePath) // #nosec G304 -- migration files are hardcoded and read-only in test environment
		if err != nil {
			t.Logf("Warning: Could not read migration %s: %v", migration, err)
			continue
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			// Ignore errors for already applied migrations (table already exists, etc)
			t.Logf("Migration %s: %v (continuing...)", migration, err)
		}
	}
}
