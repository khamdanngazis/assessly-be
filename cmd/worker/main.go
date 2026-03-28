package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/assessly/assessly-be/internal/delivery/worker"
	"github.com/assessly/assessly-be/internal/infrastructure/config"
	"github.com/assessly/assessly-be/internal/infrastructure/groq"
	"github.com/assessly/assessly-be/internal/infrastructure/logging"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	"github.com/assessly/assessly-be/internal/infrastructure/redis"
	"github.com/assessly/assessly-be/internal/usecase/scoring"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Setup logging
	logging.Setup(cfg)
	slog.Info("starting Assessly scoring worker", "env", cfg.Server.Env)

	ctx := context.Background()

	// Connect to database
	db, err := postgres.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Connect to Redis
	redisClient, err := redis.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize repositories
	submissionRepo := postgres.NewSubmissionRepository(db.Pool)
	answerRepo := postgres.NewAnswerRepository(db.Pool)
	questionRepo := postgres.NewQuestionRepository(db.Pool)
	reviewRepo := postgres.NewReviewRepository(db.Pool)

	// Initialize Groq AI client
	groqClient := groq.NewClient(groq.Config{
		APIKey:         cfg.Groq.APIKey,
		Model:          cfg.Groq.Model,
		BaseURL:        cfg.Groq.APIURL,
		TimeoutSeconds: cfg.Groq.TimeoutSeconds,
		MaxRetries:     cfg.Groq.MaxRetries,
	})
	groqAdapter := groq.NewScorerAdapter(groqClient)

	// Initialize Redis queue
	queueClient := redis.NewQueueClient(redisClient.Redis, "submissions", slog.Default())
	queueAdapter := redis.NewQueueConsumerAdapter(queueClient, "submissions")

	// Initialize scoring use case
	scoreWithAIUC := scoring.NewScoreWithAIUseCase(
		submissionRepo,
		answerRepo,
		questionRepo,
		reviewRepo,
		groqAdapter,
		slog.Default(),
	)

	// Create scoring consumer
	consumer := worker.NewScoringConsumer(
		queueAdapter,
		scoreWithAIUC,
		"submissions",
		"scoring-workers",
		getConsumerName(),
		slog.Default(),
	)

	// Start consumer
	slog.Info("starting consumer", "consumer", getConsumerName())

	// Create context for graceful shutdown
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start consumer in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := consumer.Start(workerCtx); err != nil && err != context.Canceled {
			slog.Error("consumer error", "error", err)
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("received shutdown signal")
	case err := <-errChan:
		slog.Error("worker failed", "error", err)
	}

	// Cancel context to stop consumer
	cancel()

	slog.Info("worker stopped gracefully")
}

// getConsumerName generates a unique consumer name for this worker instance
func getConsumerName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "worker-unknown"
	}
	return hostname
}
