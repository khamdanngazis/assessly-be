package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/assessly/assessly-be/internal/infrastructure/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logger logs HTTP requests with method, path, status, and duration
// T116: Also records request duration metrics for Prometheus
func Logger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := newResponseWriter(w)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)
			statusStr := strconv.Itoa(wrapped.statusCode)

			// T116: Record request duration metrics
			metrics.HTTPRequestDuration.WithLabelValues(
				r.Method,
				r.URL.Path,
				statusStr,
			).Observe(duration.Seconds())

			// Record request count
			metrics.HTTPRequestsTotal.WithLabelValues(
				r.Method,
				r.URL.Path,
				statusStr,
			).Inc()

			// Log request details
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}
