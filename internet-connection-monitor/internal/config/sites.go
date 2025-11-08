package config

import (
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// DefaultSites returns the default list of sites to monitor
func DefaultSites() []models.SiteDefinition {
	return []models.SiteDefinition{
		{
			URL:                    "https://www.google.com",
			Name:                   "google",
			Category:               "search",
			TimeoutSeconds:         30,
			WaitForNetworkIdle:     true,
		},
		{
			URL:                    "https://github.com",
			Name:                   "github",
			Category:               "development",
			TimeoutSeconds:         30,
			WaitForNetworkIdle:     true,
		},
		{
			URL:                    "https://www.cloudflare.com",
			Name:                   "cloudflare",
			Category:               "infrastructure",
			TimeoutSeconds:         30,
			WaitForNetworkIdle:     true,
		},
		{
			URL:                    "https://www.wikipedia.org",
			Name:                   "wikipedia",
			Category:               "reference",
			TimeoutSeconds:         30,
			WaitForNetworkIdle:     true,
		},
		{
			URL:                    "https://example.com",
			Name:                   "example",
			Category:               "test",
			TimeoutSeconds:         15,
			WaitForNetworkIdle:     false,
		},
	}
}

// ParseSimpleSiteList is implemented in loader.go to avoid circular dependencies
