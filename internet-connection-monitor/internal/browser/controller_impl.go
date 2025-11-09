package browser

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// ErrChromeStartupFailure indicates Chrome failed to start (not an Internet connectivity issue)
var ErrChromeStartupFailure = errors.New("chrome failed to start")

// ControllerImpl is the concrete implementation of the browser controller
type ControllerImpl struct {
	config        *config.BrowserConfig
	allocatorOpts []chromedp.ExecAllocatorOption
	hostname      string
}

// NewControllerImpl creates a new browser controller with chromedp
func NewControllerImpl(cfg *config.BrowserConfig) (*ControllerImpl, error) {
	// Get hostname for metadata
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Build allocator options that will be used for each test
	// Note: We don't create the allocator here - we create a fresh one for each test
	// to force DNS, TCP, and TLS to be refreshed on every test
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.NoSandbox, // Required for Docker
		chromedp.UserAgent(cfg.UserAgent),
		chromedp.WindowSize(cfg.WindowWidth, cfg.WindowHeight),
		chromedp.Flag("log-level", "3"), // Suppress Chrome warnings
		// Disable caches to force fresh connections on each test
		chromedp.Flag("disable-cache", "true"),
		chromedp.Flag("disable-application-cache", "true"),
		chromedp.Flag("disable-offline-load-stale-cache", "true"),
		chromedp.Flag("disk-cache-size", "0"),
		chromedp.Flag("media-cache-size", "0"),
		// Force fresh DNS, TCP, and TLS on every test
		chromedp.Flag("disable-http2", "true"),  // Force HTTP/1.1 (no connection multiplexing)
		chromedp.Flag("disable-quic", "true"),   // Disable HTTP/3
		chromedp.Flag("disable-features", "NetworkService,TLSSessionResumption"), // Disable TLS session cache
	}

	if cfg.Headless {
		opts = append(opts, chromedp.Headless)
	}

	if cfg.DisableImages {
		opts = append(opts, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}

	return &ControllerImpl{
		config:        cfg,
		allocatorOpts: opts,
		hostname:      hostname,
	}, nil
}

// TestSite navigates to a site and collects metrics
func (c *ControllerImpl) TestSite(ctx context.Context, site models.SiteDefinition) (*models.TestResult, error) {
	// Create a fresh allocator context for this test
	// This ensures DNS, TCP, and TLS connections are all refreshed (not cached/reused)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), c.allocatorOpts...)
	defer cancelAlloc()

	// Create a new browser context using the fresh allocator
	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Apply site-specific timeout
	timeout := site.GetTimeout()
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, timeout)
	defer cancelTimeout()

	// Create result
	result := &models.TestResult{
		Timestamp: time.Now(),
		TestID:    uuid.New().String(),
		Site: models.SiteInfo{
			URL:      site.URL,
			Name:     site.GetName(),
			Category: site.Category,
		},
		Status: models.StatusInfo{
			Success: false,
		},
		Metadata: models.TestMetadata{
			Hostname:  c.hostname,
			Version:   "1.1.0",
			UserAgent: c.config.UserAgent,
		},
	}

	startTime := time.Now()

	// Navigate and collect metrics
	var navigationEntry map[string]interface{}

	err := chromedp.Run(taskCtx,
		// Navigate to the URL
		chromedp.Navigate(site.URL),

		// Wait for network idle if configured
		chromedp.ActionFunc(func(ctx context.Context) error {
			if site.WaitForNetworkIdle {
				return chromedp.WaitReady("body", chromedp.ByQuery).Do(ctx)
			}
			return nil
		}),

		// Get performance navigation timing (Level 2 API)
		chromedp.Evaluate(`
			(function() {
				const entry = performance.getEntriesByType('navigation')[0];
				if (!entry) return null;
				return {
					domainLookupStart: entry.domainLookupStart,
					domainLookupEnd: entry.domainLookupEnd,
					connectStart: entry.connectStart,
					connectEnd: entry.connectEnd,
					secureConnectionStart: entry.secureConnectionStart,
					requestStart: entry.requestStart,
					responseStart: entry.responseStart,
					responseEnd: entry.responseEnd,
					domContentLoadedEventEnd: entry.domContentLoadedEventEnd,
					loadEventEnd: entry.loadEventEnd,
					duration: entry.duration,
					transferSize: entry.transferSize,
					encodedBodySize: entry.encodedBodySize,
					decodedBodySize: entry.decodedBodySize
				};
			})()
		`, &navigationEntry),
	)

	totalDuration := time.Since(startTime).Milliseconds()

	// Handle errors
	if err != nil {
		// Check if this is a Chrome startup failure (resource exhaustion, not an Internet issue)
		// These should not be reported as connectivity problems
		if isChromeStartupFailure(err) {
			// Return the special error - test loop will not report this
			return nil, ErrChromeStartupFailure
		}

		result.Status.Success = false
		result.Status.Message = "Failed to load page"
		result.Error = &models.ErrorInfo{
			ErrorType:    categorizeError(err),
			ErrorMessage: err.Error(),
		}
		result.Timings.TotalDurationMs = totalDuration
		return result, nil // Return result even on error (for logging)
	}

	// Extract timing metrics from performance data
	result.Timings = extractTimings(navigationEntry, totalDuration)
	result.Status.Success = true
	result.Status.HTTPStatus = 200 // Navigation succeeded
	result.Status.Message = "Page loaded successfully"

	return result, nil
}

// Close shuts down the browser controller
// Note: Each test now creates and cleans up its own browser instance,
// so there's no persistent browser to shut down
func (c *ControllerImpl) Close() error {
	// No persistent browser allocator to clean up
	// Each TestSite() call creates and disposes of its own browser instance
	return nil
}

// extractTimings converts performance navigation timing data to our metrics structure
//
// The browser is configured to force fresh DNS, TCP, and TLS on every test by disabling
// HTTP/2, QUIC, and TLS session resumption. This ensures accurate timing measurements
// for every connection phase, allowing us to detect network issues in DNS resolution,
// TCP handshakes, and TLS negotiation.
func extractTimings(perfData map[string]interface{}, totalMs int64) models.TimingMetrics {
	timings := models.TimingMetrics{
		TotalDurationMs: totalMs,
	}

	if perfData == nil {
		return timings
	}

	// Helper to safely get float64 from interface{}
	getFloat := func(key string) float64 {
		if val, ok := perfData[key]; ok {
			if f, ok := val.(float64); ok {
				return f
			}
		}
		return 0
	}

	// Extract timing values from Navigation Timing Level 2 API
	// All times are relative to navigationStart (0)
	domainLookupStart := getFloat("domainLookupStart")
	domainLookupEnd := getFloat("domainLookupEnd")
	connectStart := getFloat("connectStart")
	connectEnd := getFloat("connectEnd")
	secureConnectionStart := getFloat("secureConnectionStart")
	requestStart := getFloat("requestStart")
	responseStart := getFloat("responseStart")
	domContentLoadedEventEnd := getFloat("domContentLoadedEventEnd")
	loadEventEnd := getFloat("loadEventEnd")

	// Calculate individual timing components (durations)
	// The browser is forced to create fresh connections, so these values should be non-zero
	// for successful requests. Zero values indicate either an error or missing performance data.

	// DNS lookup duration
	if domainLookupEnd > 0 {
		timings.DNSLookupMs = int64(domainLookupEnd - domainLookupStart)
	}

	// TCP connection duration
	if connectEnd > 0 {
		if secureConnectionStart > 0 {
			// For HTTPS: TCP time is from connectStart to secureConnectionStart
			timings.TCPConnectionMs = int64(secureConnectionStart - connectStart)
		} else {
			// For HTTP: TCP time is the full connection time
			timings.TCPConnectionMs = int64(connectEnd - connectStart)
		}
	}

	// TLS handshake duration (only for HTTPS connections)
	if secureConnectionStart > 0 && connectEnd > secureConnectionStart {
		timings.TLSHandshakeMs = int64(connectEnd - secureConnectionStart)
	}

	// Time to first byte (TTFB): from request start to response start
	if responseStart > 0 {
		timings.TimeToFirstByteMs = int64(responseStart - requestStart)
	}

	// DOM content loaded (when HTML is parsed and DOM is ready)
	if domContentLoadedEventEnd > 0 {
		timings.DOMContentLoadedMs = int64(domContentLoadedEventEnd)
	}

	// Full page load (when all resources are loaded)
	if loadEventEnd > 0 {
		timings.FullPageLoadMs = int64(loadEventEnd)
		timings.NetworkIdleMs = int64(loadEventEnd) // Network idle ≈ load complete
	}

	return timings
}

// isChromeStartupFailure detects if Chrome failed to start (not a connectivity issue)
func isChromeStartupFailure(err error) bool {
	errStr := strings.ToLower(err.Error())

	// Chrome startup failures typically contain these phrases
	return strings.Contains(errStr, "chrome failed to start") ||
		strings.Contains(errStr, "failed to start chrome") ||
		strings.Contains(errStr, "failed to allocate") ||
		strings.Contains(errStr, "cannot start chrome")
}

// categorizeError determines the error type
func categorizeError(err error) string {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "context deadline exceeded"):
		return "timeout"
	case strings.Contains(errStr, "context canceled"):
		return "timeout"
	case strings.Contains(errStr, "dns"):
		return "dns"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "tls"):
		return "tls"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "no such host"):
		return "dns"
	default:
		return "unknown"
	}
}
