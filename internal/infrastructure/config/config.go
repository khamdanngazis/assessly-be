package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Groq     GroqConfig
	SMTP     SMTPConfig
	Logging  LoggingConfig
	Worker   WorkerConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port string
	Env  string
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host           string
	Port           string
	User           string
	Password       string
	Name           string
	SSLMode        string
	MaxConnections int
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret                string
	ExpiryHours           int
	ResetTokenExpiryHours int
}

// GroqConfig holds Groq AI configuration
type GroqConfig struct {
	APIKey         string
	Model          string
	APIURL         string
	MaxRetries     int
	TimeoutSeconds int
}

// SMTPConfig holds email service configuration
type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	FromName string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	Concurrency      int
	QueueName        string
	DeadLetterQueue  string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnv("DB_PORT", "5432"),
			User:           getEnv("DB_USER", ""),
			Password:       getEnv("DB_PASSWORD", ""),
			Name:           getEnv("DB_NAME", ""),
			SSLMode:        getEnv("DB_SSL_MODE", "disable"),
			MaxConnections: getEnvAsInt("DB_MAX_CONNECTIONS", 20),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:                getEnv("JWT_SECRET", ""),
			ExpiryHours:           getEnvAsInt("JWT_EXPIRY_HOURS", 24),
			ResetTokenExpiryHours: getEnvAsInt("JWT_RESET_TOKEN_EXPIRY_HOURS", 1),
		},
		Groq: GroqConfig{
			APIKey:         getEnv("GROQ_API_KEY", ""),
			Model:          getEnv("GROQ_MODEL", "llama-3-70b-8192"),
			APIURL:         getEnv("GROQ_API_URL", "https://api.groq.com/openai/v1"),
			MaxRetries:     getEnvAsInt("GROQ_MAX_RETRIES", 3),
			TimeoutSeconds: getEnvAsInt("GROQ_TIMEOUT_SECONDS", 30),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     getEnv("SMTP_PORT", "1025"),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASS", ""),
			From:     getEnv("SMTP_FROM", "noreply@assessly.local"),
			FromName: getEnv("SMTP_FROM_NAME", "Assessly Platform"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Worker: WorkerConfig{
			Concurrency:     getEnvAsInt("WORKER_CONCURRENCY", 5),
			QueueName:       getEnv("WORKER_QUEUE_NAME", "ai-scoring"),
			DeadLetterQueue: getEnv("WORKER_DEAD_LETTER_QUEUE", "ai-scoring-dlq"),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if required configuration fields are set
func (c *Config) Validate() error {
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.Groq.APIKey == "" {
		return fmt.Errorf("GROQ_API_KEY is required")
	}
	return nil
}

// DatabaseURL returns PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// RedisAddr returns Redis address
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets environment variable as integer or returns default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
