package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/assessly/assessly-be/internal/infrastructure/metrics"
)

// Recovery recovers from panics and logs the error
// T117: Also records error rate metrics for Prometheus
func Recovery(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// T117: Record panic as error metric
					metrics.HTTPErrorsTotal.WithLabelValues(
						r.Method,
						r.URL.Path,
						strconv.Itoa(http.StatusInternalServerError),
						"panic",
					).Inc()

					// Log the panic with stack trace
					logger.Error("panic recovered",
						"error", err,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr,
					)

					// Return 500 Internal Server Error
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte(`{"error":"internal server error"}`)); err != nil {
						// Error writing response, connection may be closed
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RequestID adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate simple request ID from timestamp
			requestID = time.Now().Format("20060102150405.000000")
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}
