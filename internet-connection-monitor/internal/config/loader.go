package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv(cfg *Config) error {
	// General settings
	if v := os.Getenv("INTER_TEST_DELAY"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid INTER_TEST_DELAY: %w", err)
		}
		cfg.General.InterTestDelay = d
	}

	if v := os.Getenv("GLOBAL_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid GLOBAL_TIMEOUT: %w", err)
		}
		cfg.General.GlobalTimeout = d
	}

	if v := os.Getenv("CACHE_SIZE"); v != "" {
		var size int
		fmt.Sscanf(v, "%d", &size)
		if size > 0 {
			cfg.General.CacheSize = size
		}
	}

	// Sites from comma-separated list
	if v := os.Getenv("SITES"); v != "" {
		sites, err := ParseSimpleSiteList(v)
		if err != nil {
			return fmt.Errorf("invalid SITES: %w", err)
		}
		cfg.Sites.List = sites
	}

	// Browser settings
	if v := os.Getenv("BROWSER_HEADLESS"); v != "" {
		cfg.Browser.Headless = v == "true" || v == "1"
	}

	if v := os.Getenv("BROWSER_USER_AGENT"); v != "" {
		cfg.Browser.UserAgent = v
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}

	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	// Elasticsearch
	if v := os.Getenv("ES_ENABLED"); v != "" {
		cfg.Elasticsearch.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("ES_ENDPOINT"); v != "" {
		cfg.Elasticsearch.Endpoint = v
	}

	if v := os.Getenv("ES_INDEX_PATTERN"); v != "" {
		cfg.Elasticsearch.IndexPattern = v
	}

	if v := os.Getenv("ES_USERNAME"); v != "" {
		cfg.Elasticsearch.Username = v
	}

	if v := os.Getenv("ES_PASSWORD"); v != "" {
		cfg.Elasticsearch.Password = v
	}

	if v := os.Getenv("ES_API_KEY"); v != "" {
		cfg.Elasticsearch.APIKey = v
	}

	if v := os.Getenv("ES_BULK_SIZE"); v != "" {
		var size int
		fmt.Sscanf(v, "%d", &size)
		if size > 0 {
			cfg.Elasticsearch.BulkSize = size
		}
	}

	if v := os.Getenv("ES_FLUSH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid ES_FLUSH_INTERVAL: %w", err)
		}
		cfg.Elasticsearch.FlushInterval = d
	}

	// SNMP
	if v := os.Getenv("SNMP_ENABLED"); v != "" {
		cfg.SNMP.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("SNMP_PORT"); v != "" {
		var port int
		fmt.Sscanf(v, "%d", &port)
		if port > 0 {
			cfg.SNMP.Port = port
		}
	}

	if v := os.Getenv("SNMP_COMMUNITY"); v != "" {
		cfg.SNMP.Community = v
	}

	if v := os.Getenv("SNMP_LISTEN_ADDRESS"); v != "" {
		cfg.SNMP.ListenAddress = v
	}

	// Prometheus
	if v := os.Getenv("PROM_ENABLED"); v != "" {
		cfg.Prometheus.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("PROM_PORT"); v != "" {
		var port int
		fmt.Sscanf(v, "%d", &port)
		if port > 0 {
			cfg.Prometheus.Port = port
		}
	}

	if v := os.Getenv("PROM_PATH"); v != "" {
		cfg.Prometheus.Path = v
	}

	if v := os.Getenv("PROM_LISTEN_ADDRESS"); v != "" {
		cfg.Prometheus.ListenAddress = v
	}

	// Advanced
	if v := os.Getenv("HEALTH_CHECK_ENABLED"); v != "" {
		cfg.Advanced.HealthCheckEnabled = v == "true" || v == "1"
	}

	if v := os.Getenv("HEALTH_CHECK_PORT"); v != "" {
		var port int
		fmt.Sscanf(v, "%d", &port)
		if port > 0 {
			cfg.Advanced.HealthCheckPort = port
		}
	}

	if v := os.Getenv("HEALTH_CHECK_LISTEN_ADDRESS"); v != "" {
		cfg.Advanced.HealthCheckListenAddress = v
	}

	return nil
}

// ParseSimpleSiteList parses a comma-separated list of domains/URLs
func ParseSimpleSiteList(sitesStr string) ([]models.SiteDefinition, error) {
	if sitesStr == "" {
		return nil, nil
	}

	parts := strings.Split(sitesStr, ",")
	sites := make([]models.SiteDefinition, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Normalize to full URL
		url := part
		if !strings.HasPrefix(part, "http://") && !strings.HasPrefix(part, "https://") {
			url = "https://" + part
		}

		// Derive name from domain
		name := part
		name = strings.TrimPrefix(name, "https://")
		name = strings.TrimPrefix(name, "http://")
		name = strings.TrimPrefix(name, "www.")
		if idx := strings.Index(name, "/"); idx > 0 {
			name = name[:idx]
		}
		if idx := strings.Index(name, "."); idx > 0 {
			name = name[:idx]
		}

		sites = append(sites, models.SiteDefinition{
			URL:                url,
			Name:               name,
			TimeoutSeconds:     30,
			WaitForNetworkIdle: true,
		})
	}

	return sites, nil
}
