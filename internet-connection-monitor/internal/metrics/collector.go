package metrics

import (
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Collector aggregates metrics from test results
type Collector struct {
	cache      *ResultsCache
	// TODO: Add references to output modules
}

// NewCollector creates a new metrics collector
func NewCollector(cacheSize int) *Collector {
	return &Collector{
		cache: NewResultsCache(cacheSize),
	}
}

// RecordResult processes a test result and updates all metrics
func (c *Collector) RecordResult(result *models.TestResult) error {
	// Add to cache for SNMP polling
	c.cache.Add(result)

	// TODO: Update Prometheus metrics
	// TODO: Push to Elasticsearch
	// TODO: Log to stdout

	return nil
}

// GetRecentResults returns the N most recent test results
func (c *Collector) GetRecentResults(n int) []*models.TestResult {
	return c.cache.GetLast(n)
}
