package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assessly/assessly-be/internal/delivery/http/handler"
	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/assessly/assessly-be/internal/delivery/http/router"
	"github.com/assessly/assessly-be/internal/infrastructure/auth"
	"github.com/assessly/assessly-be/internal/infrastructure/config"
	"github.com/assessly/assessly-be/internal/infrastructure/email"
	"github.com/assessly/assessly-be/internal/infrastructure/logging"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	"github.com/assessly/assessly-be/internal/infrastructure/redis"
	authUC "github.com/assessly/assessly-be/internal/usecase/auth"
	reviewUC "github.com/assessly/assessly-be/internal/usecase/review"
	submissionUC "github.com/assessly/assessly-be/internal/usecase/submission"
	testUC "github.com/assessly/assessly-be/internal/usecase/test"
	"github.com/google/uuid"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Setup logging
	logging.Setup(cfg)
	slog.Info("starting Assessly API server", "env", cfg.Server.Env, "port", cfg.Server.Port)

	ctx := context.Background()

	// Connect to database
	db, err := postgres.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run database migrations
	if err := runMigrations(cfg); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisClient, err := redis.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize infrastructure components
	jwtService := auth.NewJWTService(cfg.JWT.Secret, "assessly", cfg.JWT.ExpiryHours)
	passwordHasher := auth.NewPasswordHasher(12) // bcrypt cost of 12
	emailSender := email.NewEmailSender(email.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     parsePort(cfg.SMTP.Port),
		Username: cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
	
	// Initialize repositories
	userRepo := postgres.NewUserRepository(db.Pool)
	testRepo := postgres.NewTestRepository(db.Pool)
	questionRepo := postgres.NewQuestionRepository(db.Pool)
	submissionRepo := postgres.NewSubmissionRepository(db.Pool)
	answerRepo := postgres.NewAnswerRepository(db.Pool)
	reviewRepo := postgres.NewReviewRepository(db.Pool)
	
	// Initialize queue for AI scoring
	queueClient := redis.NewQueueClient(redisClient.Redis, "submissions", slog.Default())
	queueAdapter := redis.NewSubmissionQueueAdapter(queueClient)
	
	// Initialize auth use cases
	registerUC := authUC.NewRegisterUserUseCase(userRepo, passwordHasher, slog.Default())
	loginUC := authUC.NewLoginUserUseCase(userRepo, passwordHasher, jwtService, slog.Default())
	
	// Initialize test use cases
	createTestUC := testUC.NewCreateTestUseCase(testRepo, slog.Default())
	addQuestionUC := testUC.NewAddQuestionUseCase(questionRepo, testRepo, slog.Default())
	publishTestUC := testUC.NewPublishTestUseCase(testRepo, questionRepo, slog.Default())
	
	// Initialize submission use cases
	generateAccessTokenUC := submissionUC.NewGenerateAccessTokenUseCase(
		testRepo,
		&accessTokenGeneratorAdapter{jwtService: jwtService},
		emailSender,
		slog.Default(),
	)
	submitTestUC := submissionUC.NewSubmitTestUseCase(
		testRepo,
		questionRepo,
		submissionRepo,
		answerRepo,
		&tokenValidatorAdapter{jwtService: jwtService},
		queueAdapter,
		slog.Default(),
	)
	getSubmissionUC := submissionUC.NewGetSubmissionUseCase(
		submissionRepo,
		answerRepo,
		reviewRepo,
		testRepo,
		&tokenValidatorAdapter{jwtService: jwtService},
		slog.Default(),
	)
	
	// Create token validator wrapper for reset password use case
	tokenValidator := &tokenValidatorAdapter{jwtService: jwtService}
	requestResetUC := authUC.NewRequestPasswordResetUseCase(
		userRepo,
		&resetTokenGeneratorAdapter{jwtService: jwtService},
		emailSender,
		slog.Default(),
	)
	resetPasswordUC := authUC.NewResetPasswordUseCase(userRepo, tokenValidator, passwordHasher, slog.Default())
	
	// Initialize review use cases
	listSubmissionsUC := reviewUC.NewListSubmissionsUseCase(submissionRepo, slog.Default())
	addManualReviewUC := reviewUC.NewAddManualReviewUseCase(reviewRepo, answerRepo, submissionRepo, slog.Default())
	getReviewUC := reviewUC.NewGetReviewUseCase(reviewRepo, slog.Default())
	
	// Initialize HTTP handlers
	authHandler := handler.NewAuthHandler(registerUC, loginUC, requestResetUC, resetPasswordUC, slog.Default())
	testHandler := handler.NewTestHandler(createTestUC, publishTestUC, slog.Default())
	questionHandler := handler.NewQuestionHandler(addQuestionUC, slog.Default())
	submissionHandler := handler.NewSubmissionHandler(
		generateAccessTokenUC,
		submitTestUC,
		getSubmissionUC,
		&testAccessTokenGeneratorAdapter{jwtService: jwtService}, // For testing endpoints
		slog.Default(),
	)
	reviewHandler := handler.NewReviewHandler(
		addManualReviewUC,
		getReviewUC,
		listSubmissionsUC,
		slog.Default(),
	)
	
	// T115: Initialize metrics handler
	metricsHandler := handler.NewMetricsHandler()
	
	// Create JWT middleware
	jwtMiddleware := middleware.JWTAuth(middleware.JWTConfig{
		SecretKey: []byte(cfg.JWT.Secret),
		Logger:    slog.Default(),
	})

	// Setup router with middleware
	r := router.New()
	
	// Apply global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(slog.Default()))
	r.Use(middleware.Logger(slog.Default()))
	
	// Apply CORS middleware
	corsConfig := middleware.DefaultCORSConfig()
	// Note: AllowedOrigins would be configured here if needed
	// corsConfig.AllowedOrigins = []string{"http://localhost:3000"}
	r.Use(middleware.CORS(corsConfig))
	
	// Setup routes (T115: pass metricsHandler, T089: add reviewHandler)
	r.SetupRoutes(
		healthCheckHandler(db, redisClient),
		metricsHandler,
		authHandler,
		testHandler,
		questionHandler,
		submissionHandler,
		reviewHandler,
		jwtMiddleware,
		slog.Default(),
	)
	
	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server listening", "port", cfg.Server.Port, "addr", fmt.Sprintf("http://localhost:%s", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

// runMigrations runs database migrations
func runMigrations(cfg *config.Config) error {
	slog.Info("running database migrations...")
	
	// TODO: Implement migration runner using golang-migrate
	// For now, migrations should be run manually with:
	// migrate -path ./migrations -database "postgres://..." up
	
	slog.Info("migrations completed (manual execution required)")
	return nil
}

// healthCheckHandler returns a simple health check handler
func healthCheckHandler(db *postgres.DB, redis *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		
		// Check database
		dbStatus := "connected"
		if err := db.Health(ctx); err != nil {
			dbStatus = "disconnected"
			slog.Error("database health check failed", "error", err)
		}

		// Check Redis
		redisStatus := "connected"
		if err := redis.Health(ctx); err != nil {
			redisStatus = "disconnected"
			slog.Error("redis health check failed", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		
		if dbStatus == "connected" && redisStatus == "connected" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"healthy","database":"%s","redis":"%s"}`, dbStatus, redisStatus)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","database":"%s","redis":"%s"}`, dbStatus, redisStatus)
		}
	}
}

// tokenValidatorAdapter adapts JWT service to TokenValidator interface
type tokenValidatorAdapter struct {
	jwtService *auth.JWTService
}

func (a *tokenValidatorAdapter) ValidateToken(tokenString string) (userID string, email string, role string, err error) {
	claims, err := a.jwtService.ValidateToken(tokenString)
	if err != nil {
		return "", "", "", err
	}
	return claims.UserID, claims.Email, claims.Role, nil
}

// resetTokenGeneratorAdapter adapts JWT service to ResetTokenGenerator interface
type resetTokenGeneratorAdapter struct {
	jwtService *auth.JWTService
}

func (a *resetTokenGeneratorAdapter) GenerateResetToken(userID string, email string) (string, error) {
	// Parse string UUID back to uuid.UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}
	return a.jwtService.GenerateResetToken(uid, email)
}

// accessTokenGeneratorAdapter adapts JWT service to AccessTokenGenerator interface
type accessTokenGeneratorAdapter struct {
	jwtService *auth.JWTService
}

func (a *accessTokenGeneratorAdapter) GenerateAccessToken(testID uuid.UUID, email string, expiryHours int) (string, error) {
	return a.jwtService.GenerateAccessToken(testID, email, expiryHours)
}

// testAccessTokenGeneratorAdapter adapts JWT service for handler interface (testID as string)
type testAccessTokenGeneratorAdapter struct {
	jwtService *auth.JWTService
}

func (a *testAccessTokenGeneratorAdapter) GenerateAccessToken(testIDStr, email string, expiryHours int) (string, error) {
	testID, err := uuid.Parse(testIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid test ID: %w", err)
	}
	return a.jwtService.GenerateAccessToken(testID, email, expiryHours)
}

// parsePort converts port string to int, defaults to 587 for SMTP if invalid
func parsePort(portStr string) int {
	port := 587 // Default SMTP port
	if n, err := fmt.Sscanf(portStr, "%d", &port); err != nil || n != 1 {
		return 587
	}
	return port
}

