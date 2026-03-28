package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/assessly/assessly-be/internal/infrastructure/config"
)

// Setup initializes the structured logger
func Setup(cfg *config.Config) {
	level := parseLevel(cfg.Logging.Level)
	
	var handler slog.Handler
	
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Filter sensitive data
				if a.Key == "password" || a.Key == "token" || a.Key == "secret" {
					return slog.Attr {}
				}
				return a
			},
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	
	slog.Info("logger initialized",
		"level", cfg.Logging.Level,
		"format", cfg.Logging.Format,
	)
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
