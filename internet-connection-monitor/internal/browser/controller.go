package browser

import (
	"context"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Controller is the interface for browser automation
type Controller interface {
	TestSite(ctx context.Context, site models.SiteDefinition) (*models.TestResult, error)
	Close() error
}

// NewController creates a new browser controller
func NewController(cfg *config.BrowserConfig) (Controller, error) {
	return NewControllerImpl(cfg)
}
