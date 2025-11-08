package browser

import (
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// MetricsCollector extracts timing metrics from browser navigation
type MetricsCollector struct {
	// TODO: Add fields for collecting CDP performance metrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// CollectMetrics extracts timing information from the browser
func (m *MetricsCollector) CollectMetrics() (*models.TimingMetrics, error) {
	// TODO: Implement metrics collection from Chrome DevTools Protocol
	// Use Performance.getMetrics() and Navigation Timing API
	// Extract: DNS, TCP, TLS, TTFB, DOM loaded, Network idle
	return &models.TimingMetrics{}, nil
}
