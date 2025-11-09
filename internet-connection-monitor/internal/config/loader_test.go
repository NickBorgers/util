package config

import (
	"os"
	"testing"
	"time"
)

// TestParseSimpleSiteList_BasicDomains tests parsing simple domain names
func TestParseSimpleSiteList_BasicDomains(t *testing.T) {
	sitesStr := "google.com,github.com,example.org"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 3 {
		t.Fatalf("Expected 3 sites, got %d", len(sites))
	}

	// Check first site
	if sites[0].URL != "https://google.com" {
		t.Errorf("Expected URL 'https://google.com', got '%s'", sites[0].URL)
	}
	if sites[0].Name != "google" {
		t.Errorf("Expected name 'google', got '%s'", sites[0].Name)
	}

	// Check second site
	if sites[1].URL != "https://github.com" {
		t.Errorf("Expected URL 'https://github.com', got '%s'", sites[1].URL)
	}
	if sites[1].Name != "github" {
		t.Errorf("Expected name 'github', got '%s'", sites[1].Name)
	}

	// Check default values
	if sites[0].TimeoutSeconds != 30 {
		t.Errorf("Expected default timeout 30s, got %d", sites[0].TimeoutSeconds)
	}
	if !sites[0].WaitForNetworkIdle {
		t.Error("Expected WaitForNetworkIdle to be true by default")
	}
}

// TestParseSimpleSiteList_HTTPSURLs tests parsing full HTTPS URLs
func TestParseSimpleSiteList_HTTPSURLs(t *testing.T) {
	sitesStr := "https://www.google.com,https://github.com/trending"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("Expected 2 sites, got %d", len(sites))
	}

	// URL should remain unchanged
	if sites[0].URL != "https://www.google.com" {
		t.Errorf("Expected URL 'https://www.google.com', got '%s'", sites[0].URL)
	}

	// Name should strip protocol and www
	if sites[0].Name != "google" {
		t.Errorf("Expected name 'google', got '%s'", sites[0].Name)
	}

	// Name should strip path
	if sites[1].Name != "github" {
		t.Errorf("Expected name 'github' (path stripped), got '%s'", sites[1].Name)
	}
}

// TestParseSimpleSiteList_HTTPURLs tests parsing HTTP (non-secure) URLs
func TestParseSimpleSiteList_HTTPURLs(t *testing.T) {
	sitesStr := "http://example.com,http://localhost:8080"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("Expected 2 sites, got %d", len(sites))
	}

	// HTTP URLs should remain HTTP
	if sites[0].URL != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got '%s'", sites[0].URL)
	}

	if sites[1].URL != "http://localhost:8080" {
		t.Errorf("Expected URL 'http://localhost:8080', got '%s'", sites[1].URL)
	}
}

// TestParseSimpleSiteList_WithWWW tests parsing domains with www prefix
func TestParseSimpleSiteList_WithWWW(t *testing.T) {
	sitesStr := "www.google.com,www.github.com"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// URLs should get https:// prefix
	if sites[0].URL != "https://www.google.com" {
		t.Errorf("Expected URL 'https://www.google.com', got '%s'", sites[0].URL)
	}

	// Names should strip www
	if sites[0].Name != "google" {
		t.Errorf("Expected name 'google', got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_WithPaths tests parsing URLs with paths
func TestParseSimpleSiteList_WithPaths(t *testing.T) {
	sitesStr := "example.com/api/status,github.com/trending"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Full URL with path should be preserved
	if sites[0].URL != "https://example.com/api/status" {
		t.Errorf("Expected URL with path, got '%s'", sites[0].URL)
	}

	// Name should strip path
	if sites[0].Name != "example" {
		t.Errorf("Expected name 'example' (path stripped), got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_WithWhitespace tests parsing with extra whitespace
func TestParseSimpleSiteList_WithWhitespace(t *testing.T) {
	sitesStr := "  google.com  ,  github.com  ,  example.com  "
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 3 {
		t.Fatalf("Expected 3 sites, got %d", len(sites))
	}

	// Whitespace should be trimmed
	if sites[0].URL != "https://google.com" {
		t.Errorf("Expected whitespace to be trimmed, got '%s'", sites[0].URL)
	}
}

// TestParseSimpleSiteList_EmptyString tests parsing empty string
func TestParseSimpleSiteList_EmptyString(t *testing.T) {
	sites, err := ParseSimpleSiteList("")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sites != nil {
		t.Errorf("Expected nil for empty string, got %d sites", len(sites))
	}
}

// TestParseSimpleSiteList_EmptyElements tests parsing with empty elements
func TestParseSimpleSiteList_EmptyElements(t *testing.T) {
	sitesStr := "google.com,,github.com,,"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Empty elements should be skipped
	if len(sites) != 2 {
		t.Fatalf("Expected 2 sites (empty elements skipped), got %d", len(sites))
	}

	if sites[0].Name != "google" {
		t.Errorf("Expected first site to be google, got '%s'", sites[0].Name)
	}
	if sites[1].Name != "github" {
		t.Errorf("Expected second site to be github, got '%s'", sites[1].Name)
	}
}

// TestParseSimpleSiteList_SingleSite tests parsing single site
func TestParseSimpleSiteList_SingleSite(t *testing.T) {
	sitesStr := "google.com"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("Expected 1 site, got %d", len(sites))
	}

	if sites[0].Name != "google" {
		t.Errorf("Expected name 'google', got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_Subdomains tests parsing subdomains
func TestParseSimpleSiteList_Subdomains(t *testing.T) {
	sitesStr := "api.github.com,status.example.com"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Name should use first part before dot
	if sites[0].Name != "api" {
		t.Errorf("Expected name 'api', got '%s'", sites[0].Name)
	}
	if sites[1].Name != "status" {
		t.Errorf("Expected name 'status', got '%s'", sites[1].Name)
	}
}

// TestParseSimpleSiteList_MixedFormats tests parsing mixed URL formats
func TestParseSimpleSiteList_MixedFormats(t *testing.T) {
	sitesStr := "google.com,https://github.com,http://example.com,www.cloudflare.com"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 4 {
		t.Fatalf("Expected 4 sites, got %d", len(sites))
	}

	// Check various formats are handled correctly
	expectedURLs := []string{
		"https://google.com",
		"https://github.com",
		"http://example.com",
		"https://www.cloudflare.com",
	}

	for i, expected := range expectedURLs {
		if sites[i].URL != expected {
			t.Errorf("Site %d: expected URL '%s', got '%s'", i, expected, sites[i].URL)
		}
	}
}

// TestParseSimpleSiteList_IPAddresses tests parsing IP addresses
func TestParseSimpleSiteList_IPAddresses(t *testing.T) {
	sitesStr := "192.168.1.1,10.0.0.1:8080"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("Expected 2 sites, got %d", len(sites))
	}

	// IP addresses should get https:// prefix
	if sites[0].URL != "https://192.168.1.1" {
		t.Errorf("Expected URL 'https://192.168.1.1', got '%s'", sites[0].URL)
	}

	// Name should be the IP address (first part before dot)
	if sites[0].Name != "192" {
		t.Errorf("Expected name '192', got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_Localhost tests parsing localhost
func TestParseSimpleSiteList_Localhost(t *testing.T) {
	sitesStr := "localhost:3000,localhost"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("Expected 2 sites, got %d", len(sites))
	}

	if sites[0].URL != "https://localhost:3000" {
		t.Errorf("Expected URL 'https://localhost:3000', got '%s'", sites[0].URL)
	}

	if sites[0].Name != "localhost:3000" {
		t.Errorf("Expected name 'localhost:3000', got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_SpecialCharacters tests handling special characters
func TestParseSimpleSiteList_SpecialCharacters(t *testing.T) {
	sitesStr := "example.com/api?key=value,test.com#fragment"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// URLs with query params and fragments should be preserved
	if sites[0].URL != "https://example.com/api?key=value" {
		t.Errorf("Expected query params preserved, got '%s'", sites[0].URL)
	}

	// Name should still be extracted from domain
	if sites[0].Name != "example" {
		t.Errorf("Expected name 'example', got '%s'", sites[0].Name)
	}
}

// TestParseSimpleSiteList_TrailingSlashes tests handling trailing slashes
func TestParseSimpleSiteList_TrailingSlashes(t *testing.T) {
	sitesStr := "example.com/,test.com/api/"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Trailing slashes should be preserved
	if sites[0].URL != "https://example.com/" {
		t.Errorf("Expected trailing slash preserved, got '%s'", sites[0].URL)
	}
}

// TestParseSimpleSiteList_LongList tests parsing a long list
func TestParseSimpleSiteList_LongList(t *testing.T) {
	sitesStr := "a.com,b.com,c.com,d.com,e.com,f.com,g.com,h.com,i.com,j.com"
	sites, err := ParseSimpleSiteList(sitesStr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sites) != 10 {
		t.Fatalf("Expected 10 sites, got %d", len(sites))
	}

	// Check all sites have correct defaults
	for i, site := range sites {
		if site.TimeoutSeconds != 30 {
			t.Errorf("Site %d: expected timeout 30s, got %d", i, site.TimeoutSeconds)
		}
		if !site.WaitForNetworkIdle {
			t.Errorf("Site %d: expected WaitForNetworkIdle true", i)
		}
	}
}

// TestLoadFromEnv_HealthCheckListenAddress tests loading HEALTH_CHECK_LISTEN_ADDRESS from environment
func TestLoadFromEnv_HealthCheckListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "localhost binding",
			envValue: "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			name:     "all interfaces binding",
			envValue: "0.0.0.0",
			expected: "0.0.0.0",
		},
		{
			name:     "specific IP binding",
			envValue: "192.168.1.100",
			expected: "192.168.1.100",
		},
		{
			name:     "IPv6 localhost",
			envValue: "::1",
			expected: "::1",
		},
		{
			name:     "IPv6 all interfaces",
			envValue: "::",
			expected: "::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("HEALTH_CHECK_LISTEN_ADDRESS", tt.envValue)
			defer os.Unsetenv("HEALTH_CHECK_LISTEN_ADDRESS")

			// Create default config and load from env
			cfg := DefaultConfig()
			err := LoadFromEnv(cfg)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg.Advanced.HealthCheckListenAddress != tt.expected {
				t.Errorf("Expected HealthCheckListenAddress '%s', got '%s'", tt.expected, cfg.Advanced.HealthCheckListenAddress)
			}
		})
	}
}

// TestLoadFromEnv_HealthCheckListenAddress_NotSet tests default value when env var not set
func TestLoadFromEnv_HealthCheckListenAddress_NotSet(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("HEALTH_CHECK_LISTEN_ADDRESS")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should retain default value
	if cfg.Advanced.HealthCheckListenAddress != "0.0.0.0" {
		t.Errorf("Expected default HealthCheckListenAddress '0.0.0.0', got '%s'", cfg.Advanced.HealthCheckListenAddress)
	}
}

// TestLoadFromEnv_MultipleAdvancedSettings tests loading multiple advanced settings
func TestLoadFromEnv_MultipleAdvancedSettings(t *testing.T) {
	os.Setenv("HEALTH_CHECK_ENABLED", "true")
	os.Setenv("HEALTH_CHECK_PORT", "8888")
	os.Setenv("HEALTH_CHECK_LISTEN_ADDRESS", "127.0.0.1")
	defer func() {
		os.Unsetenv("HEALTH_CHECK_ENABLED")
		os.Unsetenv("HEALTH_CHECK_PORT")
		os.Unsetenv("HEALTH_CHECK_LISTEN_ADDRESS")
	}()

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !cfg.Advanced.HealthCheckEnabled {
		t.Error("Expected HealthCheckEnabled to be true")
	}

	if cfg.Advanced.HealthCheckPort != 8888 {
		t.Errorf("Expected HealthCheckPort 8888, got %d", cfg.Advanced.HealthCheckPort)
	}

	if cfg.Advanced.HealthCheckListenAddress != "127.0.0.1" {
		t.Errorf("Expected HealthCheckListenAddress '127.0.0.1', got '%s'", cfg.Advanced.HealthCheckListenAddress)
	}
}

// TestLoadFromEnv_PrometheusListenAddress tests Prometheus listen address configuration
func TestLoadFromEnv_PrometheusListenAddress(t *testing.T) {
	os.Setenv("PROM_LISTEN_ADDRESS", "127.0.0.1")
	defer os.Unsetenv("PROM_LISTEN_ADDRESS")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Prometheus.ListenAddress != "127.0.0.1" {
		t.Errorf("Expected Prometheus ListenAddress '127.0.0.1', got '%s'", cfg.Prometheus.ListenAddress)
	}
}

// TestLoadFromEnv_SNMPListenAddress tests SNMP listen address configuration
func TestLoadFromEnv_SNMPListenAddress(t *testing.T) {
	os.Setenv("SNMP_LISTEN_ADDRESS", "127.0.0.1")
	defer os.Unsetenv("SNMP_LISTEN_ADDRESS")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.SNMP.ListenAddress != "127.0.0.1" {
		t.Errorf("Expected SNMP ListenAddress '127.0.0.1', got '%s'", cfg.SNMP.ListenAddress)
	}
}

// TestLoadFromEnv_InterTestDelay tests loading time duration from environment
func TestLoadFromEnv_InterTestDelay(t *testing.T) {
	os.Setenv("INTER_TEST_DELAY", "5s")
	defer os.Unsetenv("INTER_TEST_DELAY")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.General.InterTestDelay != 5*time.Second {
		t.Errorf("Expected InterTestDelay 5s, got %v", cfg.General.InterTestDelay)
	}
}

// TestLoadFromEnv_InvalidDuration tests error handling for invalid duration
func TestLoadFromEnv_InvalidDuration(t *testing.T) {
	os.Setenv("INTER_TEST_DELAY", "invalid")
	defer os.Unsetenv("INTER_TEST_DELAY")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err == nil {
		t.Fatal("Expected error for invalid duration, got nil")
	}
}

// TestLoadFromEnv_Sites tests loading sites from environment
func TestLoadFromEnv_Sites(t *testing.T) {
	os.Setenv("SITES", "google.com,github.com,example.com")
	defer os.Unsetenv("SITES")

	cfg := DefaultConfig()
	err := LoadFromEnv(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(cfg.Sites.List) != 3 {
		t.Errorf("Expected 3 sites, got %d", len(cfg.Sites.List))
	}

	if cfg.Sites.List[0].Name != "google" {
		t.Errorf("Expected first site name 'google', got '%s'", cfg.Sites.List[0].Name)
	}
}
