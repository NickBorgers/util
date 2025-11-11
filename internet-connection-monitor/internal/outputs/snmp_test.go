package outputs

import (
	"testing"
	"time"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

func TestOIDCompare(t *testing.T) {
	tests := []struct {
		name     string
		oid1     string
		oid2     string
		expected int
	}{
		{
			name:     "Equal OIDs",
			oid1:     ".1.3.6.1.4.1.99999.1.1.0",
			oid2:     ".1.3.6.1.4.1.99999.1.1.0",
			expected: 0,
		},
		{
			name:     "First OID less than second",
			oid1:     ".1.3.6.1.4.1.99999.1.1.0",
			oid2:     ".1.3.6.1.4.1.99999.1.2.0",
			expected: -1,
		},
		{
			name:     "First OID greater than second",
			oid1:     ".1.3.6.1.4.1.99999.2.1.0",
			oid2:     ".1.3.6.1.4.1.99999.1.1.0",
			expected: 1,
		},
		{
			name:     "Shorter OID first",
			oid1:     ".1.3.6.1.4.1.99999.1",
			oid2:     ".1.3.6.1.4.1.99999.1.1",
			expected: -1,
		},
		{
			name:     "Longer OID first",
			oid1:     ".1.3.6.1.4.1.99999.1.1.0",
			oid2:     ".1.3.6.1.4.1.99999.1.1",
			expected: 1,
		},
		{
			name:     "OIDs without leading dot",
			oid1:     "1.3.6.1.4.1.99999.1.1.0",
			oid2:     "1.3.6.1.4.1.99999.1.2.0",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := oidCompare(tt.oid1, tt.oid2)
			if result != tt.expected {
				t.Errorf("oidCompare(%s, %s) = %d, expected %d",
					tt.oid1, tt.oid2, result, tt.expected)
			}
		})
	}
}

func TestSortOIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "Already sorted",
			input: []string{
				".1.3.6.1.4.1.99999.1.1.0",
				".1.3.6.1.4.1.99999.1.2.0",
				".1.3.6.1.4.1.99999.1.3.0",
			},
			expected: []string{
				".1.3.6.1.4.1.99999.1.1.0",
				".1.3.6.1.4.1.99999.1.2.0",
				".1.3.6.1.4.1.99999.1.3.0",
			},
		},
		{
			name: "Reverse order",
			input: []string{
				".1.3.6.1.4.1.99999.1.3.0",
				".1.3.6.1.4.1.99999.1.2.0",
				".1.3.6.1.4.1.99999.1.1.0",
			},
			expected: []string{
				".1.3.6.1.4.1.99999.1.1.0",
				".1.3.6.1.4.1.99999.1.2.0",
				".1.3.6.1.4.1.99999.1.3.0",
			},
		},
		{
			name: "Mixed lengths",
			input: []string{
				".1.3.6.1.4.1.99999.1.1.0.1",
				".1.3.6.1.4.1.99999.1.1.0",
				".1.3.6.1.4.1.99999.1.1",
			},
			expected: []string{
				".1.3.6.1.4.1.99999.1.1",
				".1.3.6.1.4.1.99999.1.1.0",
				".1.3.6.1.4.1.99999.1.1.0.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since sortOIDs modifies in place
			oids := make([]string, len(tt.input))
			copy(oids, tt.input)

			sortOIDs(oids)

			if len(oids) != len(tt.expected) {
				t.Fatalf("Expected %d OIDs, got %d", len(tt.expected), len(oids))
			}

			for i := range oids {
				if oids[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %s, got %s",
						i, tt.expected[i], oids[i])
				}
			}
		})
	}
}

func TestSNMPOutputWrite(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       false, // Don't start actual server in test
		Port:          1161,
		Community:     "test",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.99999",
	}

	// Create SNMP output without starting servers
	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 100),
		maxSize: 100,
		stats:   make(map[string]*siteStats),
		oidTree: make(map[string]oidHandler),
	}

	// Test writing a successful result
	successResult := &models.TestResult{
		Timestamp: time.Now(),
		Site: models.Site{
			URL:  "https://example.com",
			Name: "example",
		},
		Status: models.Status{
			Success:    true,
			HTTPStatus: 200,
		},
		Timings: models.Timings{
			TotalDurationMs: 1500,
		},
	}

	err := s.Write(successResult)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify cache
	if len(s.cache) != 1 {
		t.Errorf("Expected cache size 1, got %d", len(s.cache))
	}

	// Verify stats
	stats := s.GetSiteStats("example")
	if stats == nil {
		t.Fatal("Expected stats for 'example' site, got nil")
	}

	if stats.TotalTests != 1 {
		t.Errorf("Expected TotalTests=1, got %d", stats.TotalTests)
	}

	if stats.SuccessfulTests != 1 {
		t.Errorf("Expected SuccessfulTests=1, got %d", stats.SuccessfulTests)
	}

	if stats.FailedTests != 0 {
		t.Errorf("Expected FailedTests=0, got %d", stats.FailedTests)
	}

	if stats.LastDurationMs != 1500 {
		t.Errorf("Expected LastDurationMs=1500, got %d", stats.LastDurationMs)
	}

	// Test writing a failed result
	failureResult := &models.TestResult{
		Timestamp: time.Now(),
		Site: models.Site{
			URL:  "https://example.com",
			Name: "example",
		},
		Status: models.Status{
			Success:      false,
			HTTPStatus:   0,
			ErrorMessage: "Connection timeout",
		},
		Timings: models.Timings{
			TotalDurationMs: 30000,
		},
	}

	err = s.Write(failureResult)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify updated stats
	stats = s.GetSiteStats("example")
	if stats.TotalTests != 2 {
		t.Errorf("Expected TotalTests=2, got %d", stats.TotalTests)
	}

	if stats.SuccessfulTests != 1 {
		t.Errorf("Expected SuccessfulTests=1, got %d", stats.SuccessfulTests)
	}

	if stats.FailedTests != 1 {
		t.Errorf("Expected FailedTests=1, got %d", stats.FailedTests)
	}

	if stats.MinDurationMs != 1500 {
		t.Errorf("Expected MinDurationMs=1500, got %d", stats.MinDurationMs)
	}

	if stats.MaxDurationMs != 30000 {
		t.Errorf("Expected MaxDurationMs=30000, got %d", stats.MaxDurationMs)
	}
}

func TestSNMPOutputCircularBuffer(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       false,
		Port:          1161,
		Community:     "test",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.99999",
	}

	// Create SNMP output with small cache
	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 3),
		maxSize: 3,
		stats:   make(map[string]*siteStats),
		oidTree: make(map[string]oidHandler),
	}

	// Write 5 results (more than cache size)
	for i := 0; i < 5; i++ {
		result := &models.TestResult{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Site: models.Site{
				URL:  "https://example.com",
				Name: "example",
			},
			Status: models.Status{
				Success:    true,
				HTTPStatus: 200,
			},
			Timings: models.Timings{
				TotalDurationMs: int64(1000 + i*100),
			},
		}

		err := s.Write(result)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Cache should only have last 3 results
	if len(s.cache) != 3 {
		t.Errorf("Expected cache size 3, got %d", len(s.cache))
	}

	// Verify oldest results were removed (should have results 3, 4, 5)
	if s.cache[0].Timings.TotalDurationMs != 1200 {
		t.Errorf("Expected first cached result duration 1200, got %d",
			s.cache[0].Timings.TotalDurationMs)
	}

	if s.cache[2].Timings.TotalDurationMs != 1400 {
		t.Errorf("Expected last cached result duration 1400, got %d",
			s.cache[2].Timings.TotalDurationMs)
	}

	// Stats should still count all 5 tests
	stats := s.GetSiteStats("example")
	if stats.TotalTests != 5 {
		t.Errorf("Expected TotalTests=5, got %d", stats.TotalTests)
	}
}

func TestSNMPOutputGetSNMPData(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       false,
		Port:          1161,
		Community:     "test",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.99999",
	}

	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 100),
		maxSize: 100,
		stats:   make(map[string]*siteStats),
		oidTree: make(map[string]oidHandler),
	}

	// Add some test data
	result := &models.TestResult{
		Timestamp: time.Now(),
		Site: models.Site{
			URL:  "https://example.com",
			Name: "example",
		},
		Status: models.Status{
			Success:    true,
			HTTPStatus: 200,
		},
		Timings: models.Timings{
			TotalDurationMs: 1234,
		},
	}

	s.Write(result)

	// Get SNMP data
	data := s.GetSNMPData()

	// Verify structure
	if data["cache_size"].(int) != 1 {
		t.Errorf("Expected cache_size=1, got %v", data["cache_size"])
	}

	if data["cache_max_size"].(int) != 100 {
		t.Errorf("Expected cache_max_size=100, got %v", data["cache_max_size"])
	}

	if data["monitored_sites"].(int) != 1 {
		t.Errorf("Expected monitored_sites=1, got %v", data["monitored_sites"])
	}

	// Verify site data
	sites, ok := data["sites"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected sites to be a map")
	}

	exampleSite, ok := sites["example"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected example site data")
	}

	if exampleSite["total_tests"].(int64) != 1 {
		t.Errorf("Expected total_tests=1, got %v", exampleSite["total_tests"])
	}

	if exampleSite["successful_tests"].(int64) != 1 {
		t.Errorf("Expected successful_tests=1, got %v", exampleSite["successful_tests"])
	}

	if exampleSite["last_duration_ms"].(int64) != 1234 {
		t.Errorf("Expected last_duration_ms=1234, got %v", exampleSite["last_duration_ms"])
	}
}

func TestSNMPOutputGetAllStats(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       false,
		Port:          1161,
		Community:     "test",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.99999",
	}

	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 100),
		maxSize: 100,
		stats:   make(map[string]*siteStats),
		oidTree: make(map[string]oidHandler),
	}

	// Add results for multiple sites
	sites := []string{"example.com", "github.com", "google.com"}
	for _, siteName := range sites {
		result := &models.TestResult{
			Timestamp: time.Now(),
			Site: models.Site{
				URL:  "https://" + siteName,
				Name: siteName,
			},
			Status: models.Status{
				Success:    true,
				HTTPStatus: 200,
			},
			Timings: models.Timings{
				TotalDurationMs: 1000,
			},
		}
		s.Write(result)
	}

	// Get all stats
	allStats := s.GetAllStats()

	if len(allStats) != 3 {
		t.Errorf("Expected 3 sites in stats, got %d", len(allStats))
	}

	for _, siteName := range sites {
		stats, ok := allStats[siteName]
		if !ok {
			t.Errorf("Expected stats for site %s, but not found", siteName)
			continue
		}

		if stats.TotalTests != 1 {
			t.Errorf("Site %s: expected TotalTests=1, got %d",
				siteName, stats.TotalTests)
		}
	}
}

func TestSNMPOutputExportMIBData(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       false,
		Port:          1161,
		Community:     "test",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.99999",
	}

	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 100),
		maxSize: 100,
		stats:   make(map[string]*siteStats),
		oidTree: make(map[string]oidHandler),
	}

	// Add test data
	result := &models.TestResult{
		Timestamp: time.Now(),
		Site: models.Site{
			URL:  "https://example.com",
			Name: "example",
		},
		Status: models.Status{
			Success:    true,
			HTTPStatus: 200,
		},
		Timings: models.Timings{
			TotalDurationMs: 1234,
		},
	}
	s.Write(result)

	// Export MIB data
	mib := s.ExportMIBData()

	// Verify MIB contains expected information
	if mib == "" {
		t.Error("Expected non-empty MIB data")
	}

	// Check for key components
	expectedStrings := []string{
		"Internet Connection Monitor MIB",
		".1.3.6.1.4.1.99999",
		"example",
		"Total Tests: 1",
		"Successful: 1",
		"Failed: 0",
	}

	for _, expected := range expectedStrings {
		if !contains(mib, expected) {
			t.Errorf("Expected MIB to contain '%s'", expected)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
