package browser

import (
	"errors"
	"testing"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
)

// TestControllerImpl_ForceFreshConnections verifies that the browser controller
// is configured to force fresh DNS, TCP, and TLS on every test.
// This test will FAIL if the Chrome flags that prevent connection reuse are removed.
func TestControllerImpl_ForceFreshConnections(t *testing.T) {
	cfg := &config.BrowserConfig{
		Headless:      true,
		UserAgent:     "test-agent",
		WindowWidth:   1920,
		WindowHeight:  1080,
		DisableImages: false,
	}

	controller, err := NewControllerImpl(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Verify we have the expected number of allocator options
	// Base options (NoFirstRun, NoDefaultBrowserCheck, DisableGPU, NoSandbox, UserAgent, WindowSize)
	// + log-level flag
	// + 5 cache-disable flags
	// + 3 connection-freshness flags (disable-http2, disable-quic, disable-features)
	// + optional flags (Headless if enabled, blink-settings if images disabled)
	expectedMinOptions := 6 + 1 + 5 + 3 + 1 // +1 for Headless in this test
	if len(controller.allocatorOpts) < expectedMinOptions {
		t.Errorf("Expected at least %d allocator options, got %d", expectedMinOptions, len(controller.allocatorOpts))
		t.Error("CRITICAL: Chrome flags to prevent connection reuse may be missing!")
		t.Error("Verify that NewControllerImpl includes these flags:")
		t.Error("  - chromedp.Flag(\"disable-http2\", \"true\")")
		t.Error("  - chromedp.Flag(\"disable-quic\", \"true\")")
		t.Error("  - chromedp.Flag(\"disable-features\", \"NetworkService,TLSSessionResumption\")")
	}

	// Verify the options slice has the expected structure by checking it's not empty
	// and contains more than just the basic chromedp options
	if len(controller.allocatorOpts) == 0 {
		t.Error("CRITICAL: No allocator options configured!")
		t.Error("Browser will not force fresh connections, causing zero timing values!")
	}
}

// TestExtractTimings_HTTPS tests timing extraction for HTTPS connections
// NOTE: This tests the extraction logic with mock data. The browser is configured
// to force fresh connections on every test, so real-world values should be non-zero.
func TestExtractTimings_HTTPS(t *testing.T) {
	perfData := map[string]interface{}{
		"domainLookupStart":         0.0,
		"domainLookupEnd":           10.5,
		"connectStart":              10.5,
		"connectEnd":                50.2,
		"secureConnectionStart":     30.1,
		"requestStart":              50.2,
		"responseStart":             100.8,
		"responseEnd":               150.0,
		"domContentLoadedEventEnd":  200.0,
		"loadEventEnd":              250.0,
	}

	timings := extractTimings(perfData, 300)

	// Verify extraction logic is mathematically correct
	// DNS lookup: domainLookupEnd - domainLookupStart = 10.5 - 0 = 10ms
	if timings.DNSLookupMs == nil || *timings.DNSLookupMs != 10 {
		t.Errorf("Expected DNS lookup 10ms, got %v", timings.DNSLookupMs)
	}

	// TCP connection: secureConnectionStart - connectStart = 30.1 - 10.5 = 19ms (for HTTPS)
	if timings.TCPConnectionMs == nil || *timings.TCPConnectionMs != 19 {
		t.Errorf("Expected TCP connection 19ms, got %v", timings.TCPConnectionMs)
	}

	// TLS handshake: connectEnd - secureConnectionStart = 50.2 - 30.1 = 20ms
	if timings.TLSHandshakeMs == nil || *timings.TLSHandshakeMs != 20 {
		t.Errorf("Expected TLS handshake 20ms, got %v", timings.TLSHandshakeMs)
	}

	// TTFB: responseStart - requestStart
	if timings.TimeToFirstByteMs == nil || *timings.TimeToFirstByteMs != 50 {
		t.Errorf("Expected TTFB 50ms, got %v", timings.TimeToFirstByteMs)
	}

	// DOM content loaded
	if timings.DOMContentLoadedMs == nil || *timings.DOMContentLoadedMs != 200 {
		t.Errorf("Expected DOM content loaded 200ms, got %v", timings.DOMContentLoadedMs)
	}

	// Full page load
	if timings.FullPageLoadMs == nil || *timings.FullPageLoadMs != 250 {
		t.Errorf("Expected full page load 250ms, got %v", timings.FullPageLoadMs)
	}

	// Network idle (same as load event end)
	if timings.NetworkIdleMs == nil || *timings.NetworkIdleMs != 250 {
		t.Errorf("Expected network idle 250ms, got %v", timings.NetworkIdleMs)
	}

	// Total duration (from parameter)
	if timings.TotalDurationMs != 300 {
		t.Errorf("Expected total duration 300ms, got %d", timings.TotalDurationMs)
	}
}

// TestExtractTimings_HTTP tests timing extraction for HTTP (non-HTTPS) connections
func TestExtractTimings_HTTP(t *testing.T) {
	perfData := map[string]interface{}{
		"domainLookupStart":         0.0,
		"domainLookupEnd":           8.3,
		"connectStart":              8.3,
		"connectEnd":                25.7,
		"secureConnectionStart":     0.0, // No TLS for HTTP
		"requestStart":              25.7,
		"responseStart":             75.2,
		"responseEnd":               120.0,
		"domContentLoadedEventEnd":  180.0,
		"loadEventEnd":              200.0,
	}

	timings := extractTimings(perfData, 220)

	// DNS lookup: 8.3 - 0 = 8ms
	if timings.DNSLookupMs == nil || *timings.DNSLookupMs != 8 {
		t.Errorf("Expected DNS lookup 8ms, got %v", timings.DNSLookupMs)
	}

	// TCP connection: full connection time for HTTP (no TLS)
	// 25.7 - 8.3 = 17ms
	if timings.TCPConnectionMs == nil || *timings.TCPConnectionMs != 17 {
		t.Errorf("Expected TCP connection 17ms, got %v", timings.TCPConnectionMs)
	}

	// TLS handshake: should be nil for HTTP (no TLS)
	if timings.TLSHandshakeMs != nil {
		t.Errorf("Expected TLS handshake nil for HTTP, got %v", timings.TLSHandshakeMs)
	}

	// TTFB
	if timings.TimeToFirstByteMs == nil || *timings.TimeToFirstByteMs != 49 {
		t.Errorf("Expected TTFB 49ms, got %v", timings.TimeToFirstByteMs)
	}
}

// TestExtractTimings_NullData tests handling of nil performance data
func TestExtractTimings_NullData(t *testing.T) {
	timings := extractTimings(nil, 500)

	// All timings should be nil for missing data (not 0)
	if timings.DNSLookupMs != nil {
		t.Errorf("Expected DNS lookup nil for nil data, got %v", timings.DNSLookupMs)
	}
	if timings.TCPConnectionMs != nil {
		t.Errorf("Expected TCP connection nil for nil data, got %v", timings.TCPConnectionMs)
	}
	if timings.TLSHandshakeMs != nil {
		t.Errorf("Expected TLS handshake nil for nil data, got %v", timings.TLSHandshakeMs)
	}
	if timings.TimeToFirstByteMs != nil {
		t.Errorf("Expected TTFB nil for nil data, got %v", timings.TimeToFirstByteMs)
	}

	// Total duration should still be set
	if timings.TotalDurationMs != 500 {
		t.Errorf("Expected total duration 500ms, got %d", timings.TotalDurationMs)
	}
}

// TestExtractTimings_EmptyData tests handling of empty performance data
func TestExtractTimings_EmptyData(t *testing.T) {
	perfData := map[string]interface{}{}
	timings := extractTimings(perfData, 100)

	// All timings should be nil for empty data (not 0)
	if timings.DNSLookupMs != nil {
		t.Errorf("Expected DNS lookup nil for empty data, got %v", timings.DNSLookupMs)
	}
	if timings.TotalDurationMs != 100 {
		t.Errorf("Expected total duration 100ms, got %d", timings.TotalDurationMs)
	}
}

// TestExtractTimings_PartialData tests handling of incomplete performance data
func TestExtractTimings_PartialData(t *testing.T) {
	// Only some timing fields present
	perfData := map[string]interface{}{
		"requestStart":  50.0,
		"responseStart": 120.0,
		// Missing DNS, TCP, TLS data
	}

	timings := extractTimings(perfData, 200)

	// TTFB should be calculated correctly
	if timings.TimeToFirstByteMs == nil || *timings.TimeToFirstByteMs != 70 {
		t.Errorf("Expected TTFB 70ms, got %v", timings.TimeToFirstByteMs)
	}

	// Missing fields should be nil
	if timings.DNSLookupMs != nil {
		t.Errorf("Expected DNS lookup nil for missing data, got %v", timings.DNSLookupMs)
	}
	if timings.TCPConnectionMs != nil {
		t.Errorf("Expected TCP connection nil for missing data, got %v", timings.TCPConnectionMs)
	}
}

// TestExtractTimings_ZeroValues tests handling of zero timing values
func TestExtractTimings_ZeroValues(t *testing.T) {
	perfData := map[string]interface{}{
		"domainLookupStart":     0.0,
		"domainLookupEnd":       0.0, // Same as start - no DNS lookup time
		"connectStart":          0.0,
		"connectEnd":            0.0,
		"secureConnectionStart": 0.0,
		"requestStart":          0.0,
		"responseStart":         0.0,
	}

	timings := extractTimings(perfData, 50)

	// All durations should be nil when end values are 0 (not set)
	// Note: domainLookupEnd is 0, so the condition `if domainLookupEnd > 0` fails
	if timings.DNSLookupMs != nil {
		t.Errorf("Expected DNS lookup nil for zero values, got %v", timings.DNSLookupMs)
	}
	if timings.TCPConnectionMs != nil {
		t.Errorf("Expected TCP connection nil for zero values, got %v", timings.TCPConnectionMs)
	}
}

// TestExtractTimings_InvalidTypes tests handling of wrong data types
func TestExtractTimings_InvalidTypes(t *testing.T) {
	perfData := map[string]interface{}{
		"domainLookupStart": "not-a-number", // Wrong type → treated as 0
		"domainLookupEnd":   10.5,           // Valid
		"connectStart":      nil,            // nil value → treated as 0
		"connectEnd":        50.2,           // Valid
	}

	timings := extractTimings(perfData, 100)

	// Function is resilient - invalid types default to 0, valid values are used
	// So DNS = 10.5 - 0 = 10ms (still calculates correctly with valid end value)
	if timings.DNSLookupMs == nil || *timings.DNSLookupMs != 10 {
		t.Errorf("Expected DNS lookup 10ms (invalid start=0, valid end=10.5), got %v", timings.DNSLookupMs)
	}
	// TCP = 50.2 - 0 = 50ms (still calculates correctly with valid end value)
	if timings.TCPConnectionMs == nil || *timings.TCPConnectionMs != 50 {
		t.Errorf("Expected TCP connection 50ms (nil start=0, valid end=50.2), got %v", timings.TCPConnectionMs)
	}
}

// TestExtractTimings_NegativeValues tests handling of negative values
func TestExtractTimings_NegativeValues(t *testing.T) {
	// This shouldn't happen in real data, but test defensive behavior
	perfData := map[string]interface{}{
		"domainLookupStart": 10.0,
		"domainLookupEnd":   5.0, // End before start!
		"requestStart":      50.0,
		"responseStart":     45.0, // Response before request!
	}

	timings := extractTimings(perfData, 100)

	// Should calculate negative duration (indicates data issue)
	if timings.DNSLookupMs == nil || *timings.DNSLookupMs != -5 {
		t.Errorf("Expected DNS lookup -5ms for reversed values, got %v", timings.DNSLookupMs)
	}
	if timings.TimeToFirstByteMs == nil || *timings.TimeToFirstByteMs != -5 {
		t.Errorf("Expected TTFB -5ms for reversed values, got %v", timings.TimeToFirstByteMs)
	}
}

// TestExtractTimings_RealWorldHTTPS tests with realistic HTTPS timing values
// This simulates a fresh HTTPS request with full DNS, TCP, and TLS handshake
func TestExtractTimings_RealWorldHTTPS(t *testing.T) {
	// Realistic timing values from a fresh HTTPS request
	// Browser is configured to force fresh connections on every test
	perfData := map[string]interface{}{
		"domainLookupStart":         0.0,
		"domainLookupEnd":           15.3,   // ~15ms DNS
		"connectStart":              15.3,
		"connectEnd":                102.7,  // ~87ms total connect time
		"secureConnectionStart":     45.8,   // ~30ms TCP, ~57ms TLS
		"requestStart":              102.7,
		"responseStart":             245.1,  // ~142ms TTFB
		"responseEnd":               450.3,
		"domContentLoadedEventEnd":  892.5,
		"loadEventEnd":              1523.8,
	}

	timings := extractTimings(perfData, 1600)

	// Validate extraction logic produces correct values
	// DNS: 15.3 - 0 = 15ms
	if timings.DNSLookupMs == nil || *timings.DNSLookupMs < 10 || *timings.DNSLookupMs > 20 {
		t.Errorf("DNS lookup outside expected range (10-20ms): %v", timings.DNSLookupMs)
	}

	// TCP: 45.8 - 15.3 = 30ms (for HTTPS, only until TLS starts)
	if timings.TCPConnectionMs == nil || *timings.TCPConnectionMs < 25 || *timings.TCPConnectionMs > 35 {
		t.Errorf("TCP connection outside expected range (25-35ms): %v", timings.TCPConnectionMs)
	}

	// TLS: 102.7 - 45.8 = 56ms
	if timings.TLSHandshakeMs == nil || *timings.TLSHandshakeMs < 55 || *timings.TLSHandshakeMs > 60 {
		t.Errorf("TLS handshake outside expected range (55-60ms): %v", timings.TLSHandshakeMs)
	}

	// TTFB: 245.1 - 102.7 = 142ms
	if timings.TimeToFirstByteMs == nil || *timings.TimeToFirstByteMs < 140 || *timings.TimeToFirstByteMs > 145 {
		t.Errorf("TTFB outside expected range (140-145ms): %v", timings.TimeToFirstByteMs)
	}
}

// TestCategorizeError_Timeout tests timeout error detection
func TestCategorizeError_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: "timeout",
		},
		{
			name:     "context canceled",
			err:      errors.New("context canceled"),
			expected: "timeout",
		},
		{
			name:     "timeout in message",
			err:      errors.New("request timeout occurred"),
			expected: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected error type '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCategorizeError_DNS tests DNS error detection
func TestCategorizeError_DNS(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "dns error",
			err:      errors.New("dns lookup failed"),
			expected: "dns",
		},
		{
			name:     "no such host",
			err:      errors.New("no such host"),
			expected: "dns",
		},
		{
			name:     "DNS in uppercase",
			err:      errors.New("DNS resolution failed"),
			expected: "dns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected error type '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCategorizeError_Connection tests connection error detection
func TestCategorizeError_Connection(t *testing.T) {
	err := errors.New("connection refused")
	result := categorizeError(err)
	if result != "connection_refused" {
		t.Errorf("Expected error type 'connection_refused', got '%s'", result)
	}
}

// TestCategorizeError_TLS tests TLS error detection
func TestCategorizeError_TLS(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "tls error lowercase",
			err:  errors.New("tls handshake failed"),
		},
		{
			name: "TLS error uppercase",
			err:  errors.New("TLS certificate invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != "tls" {
				t.Errorf("Expected error type 'tls', got '%s'", result)
			}
		})
	}
}

// TestCategorizeError_Unknown tests unknown error handling
func TestCategorizeError_Unknown(t *testing.T) {
	err := errors.New("something completely unexpected happened")
	result := categorizeError(err)
	if result != "unknown" {
		t.Errorf("Expected error type 'unknown', got '%s'", result)
	}
}

// TestCategorizeError_Priority tests error type priority
func TestCategorizeError_Priority(t *testing.T) {
	// "context deadline exceeded" should match "timeout" before "context"
	err := errors.New("context deadline exceeded")
	result := categorizeError(err)
	if result != "timeout" {
		t.Errorf("Expected 'timeout' to take priority, got '%s'", result)
	}
}
