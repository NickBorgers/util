package metrics

import (
	"sync"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// ResultsCache stores recent test results in memory (ephemeral)
// This cache is used for SNMP polling and resets on container restart
type ResultsCache struct {
	maxSize int
	results []*models.TestResult
	mu      sync.RWMutex
}

// NewResultsCache creates a new results cache with the specified size
func NewResultsCache(maxSize int) *ResultsCache {
	return &ResultsCache{
		maxSize: maxSize,
		results: make([]*models.TestResult, 0, maxSize),
	}
}

// Add adds a test result to the cache
// If the cache is full, the oldest result is removed
func (c *ResultsCache) Add(result *models.TestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add the new result
	c.results = append(c.results, result)

	// Trim to max size (keep most recent)
	if len(c.results) > c.maxSize {
		c.results = c.results[len(c.results)-c.maxSize:]
	}
}

// GetLast returns the N most recent results
func (c *ResultsCache) GetLast(n int) []*models.TestResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if n > len(c.results) {
		n = len(c.results)
	}

	// Return the last N results
	start := len(c.results) - n
	if start < 0 {
		start = 0
	}

	// Make a copy to avoid race conditions
	results := make([]*models.TestResult, n)
	copy(results, c.results[start:])
	return results
}

// Count returns the current number of cached results
func (c *ResultsCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.results)
}

// Clear empties the cache
func (c *ResultsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = make([]*models.TestResult, 0, c.maxSize)
}
