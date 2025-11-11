package browser

import (
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// inferFailurePhase determines which network layer failed based on timing data
// Logic: If we have timing for phase X but not X+1, failure was in X+1
func inferFailurePhase(timings *models.TimingMetrics, siteURL string) string {
	if timings == nil {
		return "unknown"
	}

	// Check what timing data we have (in order of network stack)
	hasDNS := timings.DNSLookupMs != nil
	hasTCP := timings.TCPConnectionMs != nil
	hasTLS := timings.TLSHandshakeMs != nil
	hasTTFB := timings.TimeToFirstByteMs != nil

	// Determine if this is an HTTPS site (should have TLS)
	isHTTPS := strings.HasPrefix(siteURL, "https://")

	// Infer phase based on what completed
	if !hasDNS {
		return "dns" // Failed before DNS completed
	}
	if !hasTCP {
		return "tcp" // DNS worked, TCP didn't
	}
	if isHTTPS && !hasTLS {
		return "tls" // TCP worked, TLS didn't (only for HTTPS)
	}
	if !hasTTFB {
		return "http" // Connection established, HTTP request failed
	}

	// Has all timing but still failed? Likely HTTP/application layer
	return "http"
}

// parseErrorType extracts the Chrome error code from error text
// Returns the error code (e.g., "ERR_NAME_NOT_RESOLVED") or a fallback
func parseErrorType(err error, chromeError string) string {
	// If we have Chrome error text, parse it
	if chromeError != "" {
		// "net::ERR_NAME_NOT_RESOLVED" → "ERR_NAME_NOT_RESOLVED"
		if strings.Contains(chromeError, "net::") {
			parts := strings.Split(chromeError, "net::")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
		// "page load error net::ERR_ABORTED" → "ERR_ABORTED"
		if strings.Contains(chromeError, "net::ERR_") {
			start := strings.Index(chromeError, "ERR_")
			if start != -1 {
				// Extract until space or end of string
				errCode := chromeError[start:]
				if spaceIdx := strings.Index(errCode, " "); spaceIdx != -1 {
					errCode = errCode[:spaceIdx]
				}
				return errCode
			}
		}
		return chromeError
	}

	// Fallback to simple categorization for chromedp errors without Chrome codes
	if err == nil {
		return "unknown"
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "timeout") {
		return "timeout"
	}

	return "unknown"
}

// mergeNetworkTiming combines Network.responseReceived timing into our TimingMetrics
// Chrome gives us two sources of timing: Performance API and Network events
// This merges them to get the most complete picture
func mergeNetworkTiming(timings *models.TimingMetrics, networkTiming *network.ResourceTiming) {
	if networkTiming == nil {
		return
	}

	// Network.ResourceTiming provides:
	// - requestTime (baseline in seconds since epoch)
	// - dnsStart/dnsEnd (relative to requestTime, in milliseconds)
	// - connectStart/connectEnd
	// - sslStart/sslEnd
	// - sendStart/sendEnd
	// - receiveHeadersEnd

	// Only fill in if we don't already have the data from Performance API
	if timings.DNSLookupMs == nil && networkTiming.DNSStart >= 0 && networkTiming.DNSEnd >= 0 {
		duration := int64(networkTiming.DNSEnd - networkTiming.DNSStart)
		timings.DNSLookupMs = &duration
	}

	if timings.TCPConnectionMs == nil && networkTiming.ConnectStart >= 0 && networkTiming.ConnectEnd >= 0 {
		// For HTTPS: TCP is connectStart to sslStart
		// For HTTP: TCP is connectStart to connectEnd
		var duration int64
		if networkTiming.SslStart >= 0 {
			duration = int64(networkTiming.SslStart - networkTiming.ConnectStart)
		} else {
			duration = int64(networkTiming.ConnectEnd - networkTiming.ConnectStart)
		}
		timings.TCPConnectionMs = &duration
	}

	if timings.TLSHandshakeMs == nil && networkTiming.SslStart >= 0 && networkTiming.SslEnd >= 0 {
		duration := int64(networkTiming.SslEnd - networkTiming.SslStart)
		timings.TLSHandshakeMs = &duration
	}
}
