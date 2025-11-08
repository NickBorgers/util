package models

import "time"

// SiteDefinition represents a website to monitor
type SiteDefinition struct {
	// URL is the full URL to test (e.g., "https://www.google.com")
	URL string `yaml:"url" json:"url"`

	// Name is a short, human-readable identifier (e.g., "google")
	Name string `yaml:"name" json:"name"`

	// Category groups sites by type (e.g., "search", "social", "infrastructure")
	Category string `yaml:"category" json:"category"`

	// TimeoutSeconds is the maximum time to wait for this site to load
	TimeoutSeconds int `yaml:"timeout_seconds" json:"timeout_seconds"`

	// WaitForNetworkIdle determines if we should wait for network to be idle
	WaitForNetworkIdle bool `yaml:"wait_for_network_idle" json:"wait_for_network_idle"`

	// ExpectedElements are DOM selectors that should be present for the test to succeed
	ExpectedElements []string `yaml:"expected_elements" json:"expected_elements,omitempty"`

	// CustomHeaders to send with the request
	CustomHeaders map[string]string `yaml:"custom_headers" json:"custom_headers,omitempty"`
}

// GetTimeout returns the timeout duration for this site
func (s *SiteDefinition) GetTimeout() time.Duration {
	if s.TimeoutSeconds <= 0 {
		return 30 * time.Second // Default timeout
	}
	return time.Duration(s.TimeoutSeconds) * time.Second
}

// GetName returns the site name, deriving it from URL if not set
func (s *SiteDefinition) GetName() string {
	if s.Name != "" {
		return s.Name
	}
	// TODO: Derive name from URL if not provided
	return "unknown"
}
