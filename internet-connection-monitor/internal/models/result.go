package models

import "time"

// TestResult represents the outcome of testing a single site
type TestResult struct {
	// Timestamp when the test was performed
	Timestamp time.Time `json:"@timestamp"`

	// TestID is a unique identifier for this test
	TestID string `json:"test_id"`

	// Site information
	Site SiteInfo `json:"site"`

	// Status information
	Status StatusInfo `json:"status"`

	// Timings collected during the test
	Timings TimingMetrics `json:"timings"`

	// Error information (if test failed)
	Error *ErrorInfo `json:"error,omitempty"`

	// Metadata about the test environment
	Metadata TestMetadata `json:"metadata,omitempty"`
}

// SiteInfo contains information about the tested site
type SiteInfo struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
}

// StatusInfo contains the result status
type StatusInfo struct {
	Success    bool   `json:"success"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Message    string `json:"message,omitempty"`
}

// TimingMetrics contains all timing measurements in milliseconds
type TimingMetrics struct {
	// DNSLookupMs is the time spent resolving DNS (nil if not available)
	DNSLookupMs *int64 `json:"dns_lookup_ms,omitempty"`

	// TCPConnectionMs is the time to establish TCP connection (nil if not available)
	TCPConnectionMs *int64 `json:"tcp_connection_ms,omitempty"`

	// TLSHandshakeMs is the time for TLS negotiation (nil if not available)
	TLSHandshakeMs *int64 `json:"tls_handshake_ms,omitempty"`

	// TimeToFirstByteMs is the time until first byte received (nil if not available)
	TimeToFirstByteMs *int64 `json:"time_to_first_byte_ms,omitempty"`

	// DOMContentLoadedMs is when the DOM is fully loaded (nil if not available)
	DOMContentLoadedMs *int64 `json:"dom_content_loaded_ms,omitempty"`

	// FullPageLoadMs is when the page load event fires (nil if not available)
	FullPageLoadMs *int64 `json:"full_page_load_ms,omitempty"`

	// NetworkIdleMs is when network activity has stopped (nil if not available)
	NetworkIdleMs *int64 `json:"network_idle_ms,omitempty"`

	// TotalDurationMs is the total time from start to completion (always present)
	TotalDurationMs int64 `json:"total_duration_ms"`
}

// ErrorInfo contains error details when a test fails
type ErrorInfo struct {
	// ErrorType is Chrome's error code (e.g., "ERR_NAME_NOT_RESOLVED", "ERR_ABORTED", "timeout")
	// For chromedp errors without Chrome codes, use simplified types: "timeout", "unknown"
	ErrorType string `json:"error_type"`

	// ErrorMessage is the human-readable error message
	ErrorMessage string `json:"error_message"`

	// FailurePhase indicates which network layer failed (inferred from timing)
	// Values: "dns", "tcp", "tls", "http", "unknown"
	// Empty for successful requests
	FailurePhase string `json:"failure_phase,omitempty"`

	// StackTrace contains the error stack (for debugging)
	StackTrace string `json:"stack_trace,omitempty"`
}

// TestMetadata contains information about the test environment
type TestMetadata struct {
	// Hostname of the monitor instance
	Hostname string `json:"hostname,omitempty"`

	// Version of the monitor software
	Version string `json:"version,omitempty"`

	// Browser user agent
	UserAgent string `json:"user_agent,omitempty"`
}
