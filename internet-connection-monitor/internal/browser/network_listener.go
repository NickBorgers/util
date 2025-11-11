package browser

import (
	"context"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// NetworkEventCapture stores network events for the main document request
type NetworkEventCapture struct {
	errorText   string                  // Raw Chrome error (e.g., "net::ERR_NAME_NOT_RESOLVED")
	timing      *network.ResourceTiming // Partial timing data if available
	hasResponse bool                    // Did we get a response event?
}

// SetupNetworkListener configures event listeners to capture network data
// Call this before navigation begins
func SetupNetworkListener(ctx context.Context) *NetworkEventCapture {
	capture := &NetworkEventCapture{}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventLoadingFailed:
			// Only capture main document request (not images, CSS, etc.)
			if e.Type == network.ResourceTypeDocument {
				capture.errorText = e.ErrorText
			}
		case *network.EventResponseReceived:
			// Capture timing data from response
			if e.Type == network.ResourceTypeDocument {
				capture.timing = e.Response.Timing
				capture.hasResponse = true
			}
		}
	})

	return capture
}

// GetErrorText returns the captured Chrome error text
func (n *NetworkEventCapture) GetErrorText() string {
	return n.errorText
}

// GetTiming returns the captured network timing data
func (n *NetworkEventCapture) GetTiming() *network.ResourceTiming {
	return n.timing
}

// HasResponse returns true if a response event was captured
func (n *NetworkEventCapture) HasResponse() bool {
	return n.hasResponse
}
