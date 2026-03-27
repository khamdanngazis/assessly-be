package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler exposes Prometheus metrics
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// ServeHTTP handles GET /metrics requests
// Returns Prometheus-formatted metrics
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use Prometheus built-in handler
	promhttp.Handler().ServeHTTP(w, r)
}
