package outputs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// PrometheusOutput exposes metrics via HTTP endpoint
type PrometheusOutput struct {
	config *config.PrometheusConfig
	server *http.Server

	// Metrics
	testTotal             *prometheus.CounterVec
	testDurationMs        *prometheus.GaugeVec
	testDurationHistogram *prometheus.HistogramVec
	lastSuccessTimestamp  *prometheus.GaugeVec
	dnsLookupMs           *prometheus.GaugeVec
	tcpConnectionMs       *prometheus.GaugeVec
	tlsHandshakeMs        *prometheus.GaugeVec
	timeToFirstByteMs     *prometheus.GaugeVec
}

// NewPrometheusOutput creates a new Prometheus exporter
func NewPrometheusOutput(cfg *config.PrometheusConfig) (*PrometheusOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	p := &PrometheusOutput{
		config: cfg,
	}

	// Register metrics
	p.testTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "internet_monitor_test_total",
			Help: "Total number of tests performed",
		},
		[]string{"site", "status"},
	)

	p.testDurationMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_test_duration_ms",
			Help: "Duration of the most recent test in milliseconds",
		},
		[]string{"site"},
	)

	// Use configured buckets or default
	buckets := cfg.LatencyBuckets
	if len(buckets) == 0 {
		buckets = []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	}

	p.testDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "internet_monitor_test_duration_histogram_ms",
			Help:    "Histogram of test durations in milliseconds",
			Buckets: buckets,
		},
		[]string{"site"},
	)

	p.lastSuccessTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful test",
		},
		[]string{"site"},
	)

	// Detailed timing metrics
	p.dnsLookupMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_dns_lookup_ms",
			Help: "DNS lookup duration in milliseconds",
		},
		[]string{"site"},
	)

	p.tcpConnectionMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_tcp_connection_ms",
			Help: "TCP connection duration in milliseconds",
		},
		[]string{"site"},
	)

	p.tlsHandshakeMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_tls_handshake_ms",
			Help: "TLS handshake duration in milliseconds",
		},
		[]string{"site"},
	)

	p.timeToFirstByteMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internet_monitor_time_to_first_byte_ms",
			Help: "Time to first byte in milliseconds",
		},
		[]string{"site"},
	)

	// Register all metrics
	prometheus.MustRegister(p.testTotal)
	prometheus.MustRegister(p.testDurationMs)
	prometheus.MustRegister(p.testDurationHistogram)
	prometheus.MustRegister(p.lastSuccessTimestamp)
	prometheus.MustRegister(p.dnsLookupMs)
	prometheus.MustRegister(p.tcpConnectionMs)
	prometheus.MustRegister(p.tlsHandshakeMs)
	prometheus.MustRegister(p.timeToFirstByteMs)

	// Create HTTP server
	mux := http.NewServeMux()

	// Register Prometheus handler
	if cfg.IncludeGoMetrics {
		mux.Handle(cfg.Path, promhttp.Handler())
	} else {
		// Create a custom registry without Go metrics
		registry := prometheus.NewRegistry()
		registry.MustRegister(p.testTotal)
		registry.MustRegister(p.testDurationMs)
		registry.MustRegister(p.testDurationHistogram)
		registry.MustRegister(p.lastSuccessTimestamp)
		registry.MustRegister(p.dnsLookupMs)
		registry.MustRegister(p.tcpConnectionMs)
		registry.MustRegister(p.tlsHandshakeMs)
		registry.MustRegister(p.timeToFirstByteMs)
		mux.Handle(cfg.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	}

	addr := fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.Port)
	p.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("Starting Prometheus exporter on %s%s", addr, cfg.Path)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Prometheus server error: %v", err)
		}
	}()

	return p, nil
}

// Write updates Prometheus metrics with the test result
func (p *PrometheusOutput) Write(result *models.TestResult) error {
	if p == nil {
		return nil
	}

	siteName := result.Site.Name
	if siteName == "" {
		siteName = result.Site.URL
	}

	// Increment test counter
	status := "failure"
	if result.Status.Success {
		status = "success"
	}
	p.testTotal.WithLabelValues(siteName, status).Inc()

	// Update duration metrics
	durationMs := float64(result.Timings.TotalDurationMs)
	p.testDurationMs.WithLabelValues(siteName).Set(durationMs)
	p.testDurationHistogram.WithLabelValues(siteName).Observe(durationMs)

	// Update last success timestamp if successful
	if result.Status.Success {
		p.lastSuccessTimestamp.WithLabelValues(siteName).Set(float64(result.Timestamp.Unix()))
	}

	// Update detailed timing metrics
	if result.Timings.DNSLookupMs > 0 {
		p.dnsLookupMs.WithLabelValues(siteName).Set(float64(result.Timings.DNSLookupMs))
	}
	if result.Timings.TCPConnectionMs > 0 {
		p.tcpConnectionMs.WithLabelValues(siteName).Set(float64(result.Timings.TCPConnectionMs))
	}
	if result.Timings.TLSHandshakeMs > 0 {
		p.tlsHandshakeMs.WithLabelValues(siteName).Set(float64(result.Timings.TLSHandshakeMs))
	}
	if result.Timings.TimeToFirstByteMs > 0 {
		p.timeToFirstByteMs.WithLabelValues(siteName).Set(float64(result.Timings.TimeToFirstByteMs))
	}

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

	log.Println("Shutting down Prometheus exporter...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return p.server.Shutdown(ctx)
}
