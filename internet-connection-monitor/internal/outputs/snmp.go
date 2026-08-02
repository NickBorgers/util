package outputs

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
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

	// SNMP agent lifecycle
	listener   *net.UDPConn
	actualPort int
	startTime  time.Time

	// Site indexing for stable OIDs
	siteIndex     map[string]int
	nextSiteIndex int

	startupCh chan error
	closeOnce sync.Once
}

type siteStats struct {
	TotalTests      int64
	SuccessfulTests int64
	FailedTests     int64
	LastSuccessTime time.Time
	LastFailureTime time.Time
	LastDurationMs  int64
	AvgDurationMs   float64
	MaxDurationMs   int64
	MinDurationMs   int64
}

// NewSNMPOutput creates a new SNMP agent
func NewSNMPOutput(cfg *config.SNMPConfig) (*SNMPOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	s := &SNMPOutput{
		config:    cfg,
		cache:     make([]*models.TestResult, 0, 100),
		maxSize:   100,
		done:      make(chan struct{}),
		stats:     make(map[string]*siteStats),
		siteIndex: make(map[string]int),
		startTime: time.Now(),
		startupCh: make(chan error, 1),
	}

	// Start SNMP agent server
	s.wg.Add(1)
	go s.runSNMPAgent()

	if err := s.waitForStartup(); err != nil {
		return nil, err
	}

	log.Printf("SNMP agent listening on %s:%d (community: %s)", cfg.ListenAddress, s.Port(), cfg.Community)
	log.Printf("Note: This is a basic SNMP implementation for monitoring. For full MIB support, use SNMPv3 or a dedicated agent.")

	return s, nil
}

// runSNMPAgent runs a simple SNMP responder
// Note: This is a basic implementation. For production, consider using a full SNMP agent framework
func (s *SNMPOutput) runSNMPAgent() {
	defer s.wg.Done()

	addr := fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		s.signalStartupError(fmt.Errorf("resolve UDP address: %w", err))
		return
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		s.signalStartupError(fmt.Errorf("listen UDP: %w", err))
		return
	}

	s.mu.Lock()
	s.listener = listener
	if udpAddr.Port == 0 {
		if la, ok := listener.LocalAddr().(*net.UDPAddr); ok {
			s.actualPort = la.Port
		}
	} else {
		s.actualPort = udpAddr.Port
	}
	s.mu.Unlock()

	s.signalStartupReady()

	buffer := make([]byte, 65535)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		if err := listener.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			log.Printf("SNMP agent deadline error: %v", err)
		}

		n, remoteAddr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			select {
			case <-s.done:
				return
			default:
				log.Printf("SNMP agent read error: %v", err)
				continue
			}
		}

		packet := make([]byte, n)
		copy(packet, buffer[:n])
		s.handleRequest(remoteAddr, packet)
	}
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
		if _, ok := s.siteIndex[siteName]; !ok {
			s.nextSiteIndex++
			s.siteIndex[siteName] = s.nextSiteIndex
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
	data["monitored_sites"] = len(s.siteIndex)
	data["uptime_seconds"] = int(time.Since(s.startTime).Seconds())

	// Per-site metrics
	sites := make(map[string]interface{})
	for siteName, st := range s.stats {
		sites[siteName] = map[string]interface{}{
			"total_tests":       st.TotalTests,
			"successful_tests":  st.SuccessfulTests,
			"failed_tests":      st.FailedTests,
			"last_success_time": st.LastSuccessTime.Unix(),
			"last_failure_time": st.LastFailureTime.Unix(),
			"last_duration_ms":  st.LastDurationMs,
			"avg_duration_ms":   st.AvgDurationMs,
			"max_duration_ms":   st.MaxDurationMs,
			"min_duration_ms":   st.MinDurationMs,
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

	s.closeOnce.Do(func() {
		close(s.done)
		s.mu.Lock()
		if s.listener != nil {
			_ = s.listener.Close()
		}
		s.mu.Unlock()
	})

	// Wait for goroutine to finish
	s.wg.Wait()

	s.mu.RLock()
	defer s.mu.RUnlock()

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

// Port returns the UDP port the SNMP agent is bound to.
// When configured with port 0, this returns the dynamically assigned port.
func (s *SNMPOutput) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.actualPort != 0 {
		return s.actualPort
	}
	return s.config.Port
}

func (s *SNMPOutput) waitForStartup() error {
	select {
	case err := <-s.startupCh:
		if err != nil {
			return fmt.Errorf("failed to start SNMP agent: %w", err)
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for SNMP agent to start")
	}
}

func (s *SNMPOutput) signalStartupReady() {
	select {
	case s.startupCh <- nil:
	default:
	}
}

func (s *SNMPOutput) signalStartupError(err error) {
	select {
	case s.startupCh <- err:
	default:
	}
}

func (s *SNMPOutput) handleRequest(remote *net.UDPAddr, packet []byte) {
	snmpPacket, err := gosnmp.Default.SnmpDecodePacket(packet)
	if err != nil {
		log.Printf("SNMP decode error from %s: %v", remote, err)
		return
	}

	if snmpPacket == nil {
		return
	}

	if snmpPacket.Version != gosnmp.Version2c && snmpPacket.Version != gosnmp.Version1 {
		log.Printf("SNMP unsupported version %v from %s", snmpPacket.Version, remote)
		return
	}

	if snmpPacket.Community != s.config.Community {
		log.Printf("SNMP unauthorized community from %s", remote)
		return
	}

	sortedOIDs, valueMap := s.buildOIDSnapshot()

	response := &gosnmp.SnmpPacket{
		Version:        snmpPacket.Version,
		Community:      snmpPacket.Community,
		PDUType:        gosnmp.GetResponse,
		RequestID:      snmpPacket.RequestID,
		MsgID:          snmpPacket.MsgID,
		NonRepeaters:   snmpPacket.NonRepeaters,
		MaxRepetitions: snmpPacket.MaxRepetitions,
	}

	switch snmpPacket.PDUType {
	case gosnmp.GetRequest:
		response.Variables = s.handleGet(snmpPacket.Variables, valueMap)
	case gosnmp.GetNextRequest:
		response.Variables = s.handleGetNext(snmpPacket.Variables, valueMap, sortedOIDs)
	case gosnmp.GetBulkRequest:
		response.Variables = s.handleGetBulk(snmpPacket, valueMap, sortedOIDs)
	default:
		log.Printf("SNMP unsupported PDU type %v from %s", snmpPacket.PDUType, remote)
		response.Error = gosnmp.GenErr
		response.Variables = snmpPacket.Variables
	}

	respBytes, err := response.MarshalMsg()
	if err != nil {
		log.Printf("SNMP marshal error to %s: %v", remote, err)
		return
	}

	s.mu.RLock()
	listener := s.listener
	s.mu.RUnlock()

	if listener == nil {
		return
	}

	if _, err := listener.WriteToUDP(respBytes, remote); err != nil {
		log.Printf("SNMP write error to %s: %v", remote, err)
	}
}

func (s *SNMPOutput) handleGet(vars []gosnmp.SnmpPDU, valueMap map[string]gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	results := make([]gosnmp.SnmpPDU, 0, len(vars))
	for _, vb := range vars {
		oid := normalizeOID(vb.Name)
		if val, ok := valueMap[oid]; ok {
			results = append(results, val)
			continue
		}
		results = append(results, gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchObject})
	}
	return results
}

func (s *SNMPOutput) handleGetNext(vars []gosnmp.SnmpPDU, valueMap map[string]gosnmp.SnmpPDU, sortedOIDs []string) []gosnmp.SnmpPDU {
	results := make([]gosnmp.SnmpPDU, 0, len(vars))
	for _, vb := range vars {
		oid := normalizeOID(vb.Name)
		nextOID, ok := nextOID(sortedOIDs, oid)
		if !ok {
			results = append(results, gosnmp.SnmpPDU{Name: oid, Type: gosnmp.EndOfMibView})
			continue
		}
		results = append(results, valueMap[nextOID])
	}
	return results
}

func (s *SNMPOutput) handleGetBulk(packet *gosnmp.SnmpPacket, valueMap map[string]gosnmp.SnmpPDU, sortedOIDs []string) []gosnmp.SnmpPDU {
	vars := packet.Variables
	nonRepeaters := int(packet.NonRepeaters)
	if nonRepeaters > len(vars) {
		nonRepeaters = len(vars)
	}
	maxRepetitions := int(packet.MaxRepetitions)
	if maxRepetitions <= 0 {
		maxRepetitions = 1
	}

	results := make([]gosnmp.SnmpPDU, 0, len(vars)*maxRepetitions)

	for i := 0; i < nonRepeaters; i++ {
		oid := normalizeOID(vars[i].Name)
		next, ok := nextOID(sortedOIDs, oid)
		if !ok {
			results = append(results, gosnmp.SnmpPDU{Name: oid, Type: gosnmp.EndOfMibView})
			continue
		}
		results = append(results, valueMap[next])
	}

	for i := nonRepeaters; i < len(vars); i++ {
		oid := normalizeOID(vars[i].Name)
		current := oid
		for r := 0; r < maxRepetitions; r++ {
			next, ok := nextOID(sortedOIDs, current)
			if !ok {
				results = append(results, gosnmp.SnmpPDU{Name: current, Type: gosnmp.EndOfMibView})
				break
			}
			val := valueMap[next]
			results = append(results, val)
			current = val.Name
		}
	}

	return results
}

func (s *SNMPOutput) buildOIDSnapshot() ([]string, map[string]gosnmp.SnmpPDU) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	base := normalizeOID(s.config.EnterpriseOID)
	if base == "." {
		base = ".1.3.6.1.4.1.99999"
	}

	values := make(map[string]gosnmp.SnmpPDU)

	cacheSize := uint32(len(s.cache))
	maxSize := uint32(s.maxSize)
	siteCount := uint32(len(s.siteIndex))
	uptime := uint32(time.Since(s.startTime).Seconds())

	values[fmt.Sprintf("%s.1.0", base)] = gaugePDU(fmt.Sprintf("%s.1.0", base), cacheSize)
	values[fmt.Sprintf("%s.2.0", base)] = gaugePDU(fmt.Sprintf("%s.2.0", base), maxSize)
	values[fmt.Sprintf("%s.3.0", base)] = gaugePDU(fmt.Sprintf("%s.3.0", base), siteCount)
	values[fmt.Sprintf("%s.4.0", base)] = timeTicksPDU(fmt.Sprintf("%s.4.0", base), uptime)

	type siteEntry struct {
		name  string
		index int
		stats *siteStats
	}

	entries := make([]siteEntry, 0, len(s.stats))
	for name, st := range s.stats {
		idx, ok := s.siteIndex[name]
		if !ok {
			continue
		}
		statsCopy := *st
		entries = append(entries, siteEntry{name: name, index: idx, stats: &statsCopy})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].index == entries[j].index {
			return entries[i].name < entries[j].name
		}
		return entries[i].index < entries[j].index
	})

	siteBase := fmt.Sprintf("%s.5", base)
	for _, entry := range entries {
		prefix := fmt.Sprintf("%s.%d", siteBase, entry.index)
		values[fmt.Sprintf("%s.1", prefix)] = octetStringPDU(fmt.Sprintf("%s.1", prefix), entry.name)
		values[fmt.Sprintf("%s.2", prefix)] = counterPDU(fmt.Sprintf("%s.2", prefix), uint32(entry.stats.TotalTests))
		values[fmt.Sprintf("%s.3", prefix)] = counterPDU(fmt.Sprintf("%s.3", prefix), uint32(entry.stats.SuccessfulTests))
		values[fmt.Sprintf("%s.4", prefix)] = counterPDU(fmt.Sprintf("%s.4", prefix), uint32(entry.stats.FailedTests))

		if !entry.stats.LastSuccessTime.IsZero() {
			values[fmt.Sprintf("%s.5", prefix)] = gaugePDU(fmt.Sprintf("%s.5", prefix), uint32(entry.stats.LastSuccessTime.Unix()))
		} else {
			values[fmt.Sprintf("%s.5", prefix)] = gaugePDU(fmt.Sprintf("%s.5", prefix), 0)
		}
		if !entry.stats.LastFailureTime.IsZero() {
			values[fmt.Sprintf("%s.6", prefix)] = gaugePDU(fmt.Sprintf("%s.6", prefix), uint32(entry.stats.LastFailureTime.Unix()))
		} else {
			values[fmt.Sprintf("%s.6", prefix)] = gaugePDU(fmt.Sprintf("%s.6", prefix), 0)
		}

		values[fmt.Sprintf("%s.7", prefix)] = gaugePDU(fmt.Sprintf("%s.7", prefix), uint32(entry.stats.LastDurationMs))
		values[fmt.Sprintf("%s.8", prefix)] = gaugePDU(fmt.Sprintf("%s.8", prefix), uint32(math.Round(entry.stats.AvgDurationMs)))
		values[fmt.Sprintf("%s.9", prefix)] = gaugePDU(fmt.Sprintf("%s.9", prefix), uint32(entry.stats.MaxDurationMs))
		values[fmt.Sprintf("%s.10", prefix)] = gaugePDU(fmt.Sprintf("%s.10", prefix), uint32(entry.stats.MinDurationMs))
	}

	oids := make([]string, 0, len(values))
	for oid := range values {
		oids = append(oids, oid)
	}

	sort.Slice(oids, func(i, j int) bool {
		return compareOIDs(oids[i], oids[j]) < 0
	})

	return oids, values
}

func gaugePDU(oid string, value uint32) gosnmp.SnmpPDU {
	return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: value}
}

func counterPDU(oid string, value uint32) gosnmp.SnmpPDU {
	return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter32, Value: value}
}

func timeTicksPDU(oid string, value uint32) gosnmp.SnmpPDU {
	return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.TimeTicks, Value: value * 100}
}

func octetStringPDU(oid string, value string) gosnmp.SnmpPDU {
	return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.OctetString, Value: []byte(value)}
}

func normalizeOID(oid string) string {
	trimmed := strings.TrimSpace(oid)
	if trimmed == "" {
		return "."
	}
	if !strings.HasPrefix(trimmed, ".") {
		trimmed = "." + trimmed
	}
	for strings.HasSuffix(trimmed, ".") && len(trimmed) > 1 {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

func nextOID(sorted []string, current string) (string, bool) {
	for _, oid := range sorted {
		if compareOIDs(oid, current) > 0 {
			return oid, true
		}
	}
	return "", false
}

func compareOIDs(a, b string) int {
	if a == b {
		return 0
	}
	ap := strings.Split(strings.TrimPrefix(a, "."), ".")
	bp := strings.Split(strings.TrimPrefix(b, "."), ".")

	maxLen := len(ap)
	if len(bp) > maxLen {
		maxLen = len(bp)
	}

	for i := 0; i < maxLen; i++ {
		ai := 0
		if i < len(ap) {
			if v, err := strconv.Atoi(ap[i]); err == nil {
				ai = v
			}
		}
		bi := 0
		if i < len(bp) {
			if v, err := strconv.Atoi(bp[i]); err == nil {
				bi = v
			}
		}

		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}

	if len(ap) < len(bp) {
		return -1
	}
	if len(ap) > len(bp) {
		return 1
	}
	return 0
}
