package testloop

import (
	"sync"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// SiteIterator provides round-robin iteration over sites
type SiteIterator struct {
	sites   []models.SiteDefinition
	current int
	mu      sync.Mutex
}

// NewSiteIterator creates a new site iterator
func NewSiteIterator(sites []models.SiteDefinition) *SiteIterator {
	return &SiteIterator{
		sites:   sites,
		current: 0,
	}
}

// Next returns the next site to test in round-robin fashion
func (i *SiteIterator) Next() models.SiteDefinition {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.sites) == 0 {
		// Return empty site if no sites configured
		return models.SiteDefinition{}
	}

	site := i.sites[i.current]
	i.current = (i.current + 1) % len(i.sites)
	return site
}

// Count returns the total number of sites
func (i *SiteIterator) Count() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.sites)
}

// Reset resets the iterator to the first site
func (i *SiteIterator) Reset() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.current = 0
}
