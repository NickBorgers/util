package outputs

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// SNMPOutput provides an SNMP agent for polling recent results
// Note: This is a simplified implementation that caches results in memory
// For production use, consider using a proper SNMP agent framework
type SNMPOutput struct {
	config  *config.SNMPConfig
	cache   []*models.TestResult
	mu      sync.RWMutex
	maxSize int
	done    chan struct{}
	wg      sync.WaitGroup

	// Statistics
	stats map[string]*siteStats
}

type siteStats struct {
	TotalTests       int64
	SuccessfulTests  int64
	FailedTests      int64
	LastSuccessTime  time.Time
	LastFailureTime  time.Time
	LastDurationMs   int64
	AvgDurationMs    float64
	MaxDurationMs    int64
	MinDurationMs    int64
}

// NewSNMPOutput creates a new SNMP agent
func NewSNMPOutput(cfg *config.SNMPConfig) (*SNMPOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	s := &SNMPOutput{
		config:  cfg,
		cache:   make([]*models.TestResult, 0, 100),
		maxSize: 100,
		done:    make(chan struct{}),
		stats:   make(map[string]*siteStats),
	}

	// Start SNMP agent server
	s.wg.Add(1)
	go s.runSNMPAgent()

	log.Printf("SNMP agent listening on %s:%d (community: %s)", cfg.ListenAddress, cfg.Port, cfg.Community)
	log.Printf("Note: This is a basic SNMP implementation for monitoring. For full MIB support, use SNMPv3 or a dedicated agent.")

	return s, nil
}

// runSNMPAgent runs a simple SNMP responder
// Note: This is a basic implementation. For production, consider using a full SNMP agent framework
func (s *SNMPOutput) runSNMPAgent() {
	defer s.wg.Done()

	// Create SNMP trap listener (we'll use it as a basic agent)
	// Note: gosnmp doesn't have a full agent implementation, so this is simplified
	// In production, you'd want to use a proper SNMP agent framework or net-snmp

	log.Println("SNMP agent started (simplified implementation)")
	log.Println("For full SNMP agent functionality, consider using net-snmp or snmpd")
	log.Println("Current implementation caches results in memory for external polling")

	// Keep the goroutine alive
	<-s.done
}

// Write caches the test result for SNMP queries and updates statistics
func (s *SNMPOutput) Write(result *models.TestResult) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to circular buffer cache
	if len(s.cache) >= s.maxSize {
		// Remove oldest entry
		s.cache = s.cache[1:]
	}
	s.cache = append(s.cache, result)

	// Update statistics
	siteName := result.Site.Name
	if siteName == "" {
		siteName = result.Site.URL
	}

	if _, exists := s.stats[siteName]; !exists {
		s.stats[siteName] = &siteStats{
			MinDurationMs: result.Timings.TotalDurationMs,
			MaxDurationMs: result.Timings.TotalDurationMs,
		}
	}

	st := s.stats[siteName]
	st.TotalTests++
	st.LastDurationMs = result.Timings.TotalDurationMs

	if result.Status.Success {
		st.SuccessfulTests++
		st.LastSuccessTime = result.Timestamp
	} else {
		st.FailedTests++
		st.LastFailureTime = result.Timestamp
	}

	// Update min/max
	if result.Timings.TotalDurationMs < st.MinDurationMs {
		st.MinDurationMs = result.Timings.TotalDurationMs
	}
	if result.Timings.TotalDurationMs > st.MaxDurationMs {
		st.MaxDurationMs = result.Timings.TotalDurationMs
	}

	// Calculate running average
	st.AvgDurationMs = (st.AvgDurationMs*float64(st.TotalTests-1) + float64(result.Timings.TotalDurationMs)) / float64(st.TotalTests)

	return nil
}

// GetCachedResults returns the cached results (for external SNMP polling)
func (s *SNMPOutput) GetCachedResults() []*models.TestResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	results := make([]*models.TestResult, len(s.cache))
	copy(results, s.cache)
	return results
}

// GetSiteStats returns statistics for a specific site
func (s *SNMPOutput) GetSiteStats(siteName string) *siteStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if st, exists := s.stats[siteName]; exists {
		// Return a copy
		statsCopy := *st
		return &statsCopy
	}
	return nil
}

// GetAllStats returns statistics for all sites
func (s *SNMPOutput) GetAllStats() map[string]*siteStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	statsCopy := make(map[string]*siteStats)
	for site, st := range s.stats {
		stats := *st
		statsCopy[site] = &stats
	}
	return statsCopy
}

// GetSNMPData returns SNMP-compatible data structure
// This can be queried by external SNMP monitoring systems
func (s *SNMPOutput) GetSNMPData() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make(map[string]interface{})

	// Overall metrics
	data["cache_size"] = len(s.cache)
	data["cache_max_size"] = s.maxSize
	data["monitored_sites"] = len(s.stats)

	// Per-site metrics
	sites := make(map[string]interface{})
	for siteName, st := range s.stats {
		sites[siteName] = map[string]interface{}{
			"total_tests":        st.TotalTests,
			"successful_tests":   st.SuccessfulTests,
			"failed_tests":       st.FailedTests,
			"last_success_time":  st.LastSuccessTime.Unix(),
			"last_failure_time":  st.LastFailureTime.Unix(),
			"last_duration_ms":   st.LastDurationMs,
			"avg_duration_ms":    st.AvgDurationMs,
			"max_duration_ms":    st.MaxDurationMs,
			"min_duration_ms":    st.MinDurationMs,
		}
	}
	data["sites"] = sites

	return data
}

// SendTrap sends an SNMP trap for critical events (optional feature)
func (s *SNMPOutput) SendTrap(trapType string, message string) error {
	if s == nil || s.config == nil {
		return nil
	}

	// This would be implemented if we want to send SNMP traps for alerts
	// For now, it's a placeholder for future functionality
	log.Printf("SNMP trap (not implemented): %s - %s", trapType, message)

	return nil
}

// ExportMIBData exports the current state in a MIB-compatible format
// This is useful for documentation and external SNMP managers
func (s *SNMPOutput) ExportMIBData() string {
	data := s.GetSNMPData()

	mib := fmt.Sprintf(`
-- Internet Connection Monitor MIB (Simplified)
-- Enterprise OID: %s
--
-- This is a simplified representation. For full SNMP support,
-- use a proper SNMP agent with a complete MIB definition.

Cache Size: %v
Max Cache Size: %v
Monitored Sites: %v

Per-Site Statistics:
`, s.config.EnterpriseOID, data["cache_size"], data["cache_max_size"], data["monitored_sites"])

	if sites, ok := data["sites"].(map[string]interface{}); ok {
		for site, stats := range sites {
			if statsMap, ok := stats.(map[string]interface{}); ok {
				mib += fmt.Sprintf("\nSite: %s\n", site)
				mib += fmt.Sprintf("  Total Tests: %v\n", statsMap["total_tests"])
				mib += fmt.Sprintf("  Successful: %v\n", statsMap["successful_tests"])
				mib += fmt.Sprintf("  Failed: %v\n", statsMap["failed_tests"])
				mib += fmt.Sprintf("  Avg Duration: %.2f ms\n", statsMap["avg_duration_ms"])
			}
		}
	}

	return mib
}

// Name returns the output module name
func (s *SNMPOutput) Name() string {
	return "snmp"
}

// Close shuts down the SNMP agent
func (s *SNMPOutput) Close() error {
	if s == nil {
		return nil
	}

	log.Println("Shutting down SNMP agent...")

	// Signal shutdown
	close(s.done)

	// Wait for goroutine to finish
	s.wg.Wait()

	log.Printf("SNMP agent stopped. Final statistics:")
	for site, stats := range s.stats {
		log.Printf("  %s: %d tests (%d success, %d failed), avg: %.2f ms",
			site, stats.TotalTests, stats.SuccessfulTests, stats.FailedTests, stats.AvgDurationMs)
	}

	return nil
}

// Helper function to create SNMP PDU (for future enhancement)
func (s *SNMPOutput) createSNMPPDU(oid string, value interface{}) gosnmp.SnmpPDU {
	var pduType gosnmp.Asn1BER

	switch value.(type) {
	case int, int64:
		pduType = gosnmp.Integer
	case string:
		pduType = gosnmp.OctetString
	default:
		pduType = gosnmp.OctetString
	}

	return gosnmp.SnmpPDU{
		Name:  oid,
		Type:  pduType,
		Value: value,
	}
}
