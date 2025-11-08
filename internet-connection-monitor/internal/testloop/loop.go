package testloop

import (
	"context"
	"log/slog"
	"time"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/browser"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/metrics"
)

// TestLoop manages the continuous testing cycle
type TestLoop struct {
	config     *config.Config
	iterator   *SiteIterator
	browser    browser.Controller
	dispatcher *metrics.Dispatcher
	logger     *slog.Logger
	stopChan   chan struct{}
}

// NewTestLoop creates a new continuous test loop
func NewTestLoop(cfg *config.Config, browserCtrl browser.Controller, dispatcher *metrics.Dispatcher) (*TestLoop, error) {
	iterator := NewSiteIterator(cfg.Sites.List)

	return &TestLoop{
		config:     cfg,
		iterator:   iterator,
		browser:    browserCtrl,
		dispatcher: dispatcher,
		logger:     slog.Default(),
		stopChan:   make(chan struct{}),
	}, nil
}

// Run starts the continuous testing loop
// This is the main loop that runs forever, testing sites serially
func (t *TestLoop) Run(ctx context.Context) error {
	t.logger.Info("Starting continuous test loop",
		"sites", t.iterator.Count(),
		"inter_test_delay", t.config.General.InterTestDelay,
	)

	ticker := time.NewTicker(t.config.General.InterTestDelay)
	defer ticker.Stop()

	// Test immediately on start
	t.runSingleTest(ctx)

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("Test loop stopped by context")
			return ctx.Err()

		case <-t.stopChan:
			t.logger.Info("Test loop stopped by Stop() call")
			return nil

		case <-ticker.C:
			t.runSingleTest(ctx)
		}
	}
}

// runSingleTest executes one test iteration
func (t *TestLoop) runSingleTest(ctx context.Context) {
	// Get next site
	site := t.iterator.Next()

	t.logger.Debug("Testing site", "site", site.Name, "url", site.URL)

	// Test the site
	result, err := t.browser.TestSite(ctx, site)
	if err != nil {
		t.logger.Error("Failed to test site",
			"site", site.Name,
			"error", err,
		)
		return
	}

	// Dispatch result to all outputs
	t.dispatcher.Dispatch(result)
}

// Stop gracefully stops the test loop
func (t *TestLoop) Stop() error {
	close(t.stopChan)
	return nil
}
