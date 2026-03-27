package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Request Metrics (T116)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// Error Rate Metrics (T117)
	HTTPErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "path", "status", "error_type"},
	)

	// AI Scoring Metrics (T118)
	AIScoringDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ai_scoring_duration_seconds",
			Help:    "Duration of AI scoring operations in seconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
		},
	)

	AIScoringTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_scoring_total",
			Help: "Total number of AI scoring attempts",
		},
		[]string{"status"}, // success, error
	)

	AIScoringErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ai_scoring_errors_total",
			Help: "Total number of AI scoring errors",
		},
	)

	// Redis Queue Metrics (T119)
	RedisQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_queue_depth",
			Help: "Current depth of Redis scoring queue (pending messages)",
		},
	)

	RedisQueueProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_queue_processed_total",
			Help: "Total number of messages processed from Redis queue",
		},
	)

	RedisQueueErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_queue_errors_total",
			Help: "Total number of errors processing Redis queue messages",
		},
	)

	// Worker Metrics
	WorkerGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_goroutines_active",
			Help: "Number of active worker goroutines",
		},
	)
)
