package outputs

import (
	"net/http"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// PrometheusOutput exposes metrics via HTTP endpoint
type PrometheusOutput struct {
	config *config.PrometheusConfig
	// TODO: Add Prometheus metrics (counters, gauges, histograms)
	server *http.Server
}

// NewPrometheusOutput creates a new Prometheus exporter
func NewPrometheusOutput(cfg *config.PrometheusConfig) (*PrometheusOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// TODO: Create Prometheus metrics:
	// - Counter: test_total{site, status}
	// - Gauge: test_duration_ms{site}
	// - Histogram: test_duration_histogram{site}
	// - Gauge: last_success_timestamp{site}

	// TODO: Start HTTP server on cfg.Port with cfg.Path handler

	return &PrometheusOutput{
		config: cfg,
	}, nil
}

// Write updates Prometheus metrics with the test result
func (p *PrometheusOutput) Write(result *models.TestResult) error {
	if p == nil {
		return nil
	}

	// TODO: Update metrics:
	// - Increment test_total counter
	// - Set test_duration_ms gauge
	// - Observe histogram
	// - Update last_success_timestamp if successful

	return nil
}

// Name returns the output module name
func (p *PrometheusOutput) Name() string {
	return "prometheus"
}

// Close shuts down the HTTP server
func (p *PrometheusOutput) Close() error {
	if p == nil || p.server == nil {
		return nil
	}

	// TODO: Gracefully shut down HTTP server
	return nil
}
