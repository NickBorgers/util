package browser

import (
	"errors"
	"testing"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Helper function to create a pointer to an int64 value
func int64PtrTest(val int64) *int64 {
	return &val
}

func TestInferFailurePhase(t *testing.T) {
	tests := []struct {
		name     string
		timings  *models.TimingMetrics
		url      string
		expected string
	}{
		{
			name:     "no timing data",
			timings:  nil,
			url:      "https://example.com",
			expected: "unknown",
		},
		{
			name: "DNS failure (no DNS timing)",
			timings: &models.TimingMetrics{
				DNSLookupMs:     nil,
				TotalDurationMs: 10000,
			},
			url:      "https://example.com",
			expected: "dns",
		},
		{
			name: "TCP failure (has DNS, no TCP)",
			timings: &models.TimingMetrics{
				DNSLookupMs:     int64PtrTest(12),
				TCPConnectionMs: nil,
				TotalDurationMs: 30000,
			},
			url:      "https://example.com",
			expected: "tcp",
		},
		{
			name: "TLS failure (has DNS+TCP, no TLS) - HTTPS",
			timings: &models.TimingMetrics{
				DNSLookupMs:     int64PtrTest(12),
				TCPConnectionMs: int64PtrTest(25),
				TLSHandshakeMs:  nil,
				TotalDurationMs: 30000,
			},
			url:      "https://example.com",
			expected: "tls",
		},
		{
			name: "HTTP failure (has all connection timing, no TTFB)",
			timings: &models.TimingMetrics{
				DNSLookupMs:       int64PtrTest(12),
				TCPConnectionMs:   int64PtrTest(25),
				TLSHandshakeMs:    int64PtrTest(50),
				TimeToFirstByteMs: nil,
				TotalDurationMs:   30000,
			},
			url:      "https://example.com",
			expected: "http",
		},
		{
			name: "HTTP site TCP failure (skip TLS check)",
			timings: &models.TimingMetrics{
				DNSLookupMs:     int64PtrTest(12),
				TCPConnectionMs: nil,
				TotalDurationMs: 30000,
			},
			url:      "http://example.com",
			expected: "tcp",
		},
		{
			name: "HTTP site success timing but failed (no TTFB)",
			timings: &models.TimingMetrics{
				DNSLookupMs:       int64PtrTest(8),
				TCPConnectionMs:   int64PtrTest(15),
				TimeToFirstByteMs: nil,
				TotalDurationMs:   5000,
			},
			url:      "http://example.com",
			expected: "http",
		},
		{
			name: "All timing present but still failed",
			timings: &models.TimingMetrics{
				DNSLookupMs:       int64PtrTest(8),
				TCPConnectionMs:   int64PtrTest(15),
				TLSHandshakeMs:    int64PtrTest(25),
				TimeToFirstByteMs: int64PtrTest(100),
				TotalDurationMs:   5000,
			},
			url:      "https://example.com",
			expected: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferFailurePhase(tt.timings, tt.url)
			if got != tt.expected {
				t.Errorf("inferFailurePhase() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseErrorType(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		chromeError string
		expected    string
	}{
		{
			name:        "Chrome ERR_NAME_NOT_RESOLVED",
			chromeError: "net::ERR_NAME_NOT_RESOLVED",
			expected:    "ERR_NAME_NOT_RESOLVED",
		},
		{
			name:        "Chrome ERR_CONNECTION_REFUSED",
			chromeError: "net::ERR_CONNECTION_REFUSED",
			expected:    "ERR_CONNECTION_REFUSED",
		},
		{
			name:        "Chrome error with prefix text",
			chromeError: "page load error net::ERR_ABORTED",
			expected:    "ERR_ABORTED",
		},
		{
			name:        "Chrome ERR_CONNECTION_TIMED_OUT",
			chromeError: "net::ERR_CONNECTION_TIMED_OUT",
			expected:    "ERR_CONNECTION_TIMED_OUT",
		},
		{
			name:        "Chrome ERR_CERT_AUTHORITY_INVALID",
			chromeError: "net::ERR_CERT_AUTHORITY_INVALID",
			expected:    "ERR_CERT_AUTHORITY_INVALID",
		},
		{
			name:        "Chrome error with trailing text",
			chromeError: "net::ERR_ABORTED at navigation",
			expected:    "ERR_ABORTED",
		},
		{
			name:     "chromedp timeout (no Chrome error)",
			err:      errors.New("context deadline exceeded"),
			expected: "timeout",
		},
		{
			name:     "chromedp timeout variant",
			err:      errors.New("timeout waiting for page"),
			expected: "timeout",
		},
		{
			name:     "unknown chromedp error",
			err:      errors.New("some other error"),
			expected: "unknown",
		},
		{
			name:     "nil error, no Chrome error",
			err:      nil,
			expected: "unknown",
		},
		{
			name:        "Chrome error takes precedence over chromedp error",
			err:         errors.New("context deadline exceeded"),
			chromeError: "net::ERR_CONNECTION_REFUSED",
			expected:    "ERR_CONNECTION_REFUSED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseErrorType(tt.err, tt.chromeError)
			if got != tt.expected {
				t.Errorf("parseErrorType() = %v, want %v", got, tt.expected)
			}
		})
	}
}
