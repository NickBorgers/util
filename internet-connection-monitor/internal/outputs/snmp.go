package outputs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// SNMPOutput provides an SNMP agent for polling recent results
type SNMPOutput struct {
	config  *config.SNMPConfig
	cache   []*models.TestResult
	mu      sync.RWMutex
	maxSize int
	done    chan struct{}
	wg      sync.WaitGroup

	// Statistics
	stats map[string]*siteStats

	// SNMP server
	snmpConn   *net.UDPConn
	httpServer *http.Server

	// OID tree for efficient lookups
	oidTree map[string]oidHandler

	// Trap destinations
	trapDestinations []*gosnmp.GoSNMP
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

// oidHandler is a function that returns a value for an OID
type oidHandler func() interface{}

// OID constants based on enterprise OID
const (
	// Base enterprise OID: .1.3.6.1.4.1.99999 (example, should be registered)
	baseOID = ".1.3.6.1.4.1.99999"

	// Branch 1: General statistics
	generalStatsOID     = baseOID + ".1"
	cacheSize           = generalStatsOID + ".1.0"  // Current cache size
	maxCacheSize        = generalStatsOID + ".2.0"  // Max cache size
	monitoredSitesCount = generalStatsOID + ".3.0"  // Number of monitored sites
	totalTestsRun       = generalStatsOID + ".4.0"  // Total tests across all sites
	totalSuccesses      = generalStatsOID + ".5.0"  // Total successes
	totalFailures       = generalStatsOID + ".6.0"  // Total failures

	// Branch 2: Per-site statistics (table)
	// Format: .1.3.6.1.4.1.99999.2.<siteIndex>.<metric>
	siteStatsOID = baseOID + ".2"
	// siteStatsOID + ".<index>.1" = site name
	// siteStatsOID + ".<index>.2" = total tests for site
	// siteStatsOID + ".<index>.3" = successful tests
	// siteStatsOID + ".<index>.4" = failed tests
	// siteStatsOID + ".<index>.5" = last duration (ms)
	// siteStatsOID + ".<index>.6" = average duration (ms)
	// siteStatsOID + ".<index>.7" = min duration (ms)
	// siteStatsOID + ".<index>.8" = max duration (ms)
	// siteStatsOID + ".<index>.9" = last success time (Unix timestamp)
	// siteStatsOID + ".<index>.10" = last failure time (Unix timestamp)

	// Branch 3: Recent test results (table, last N tests)
	recentTestsOID = baseOID + ".3"
	// recentTestsOID + ".<index>.1" = site name
	// recentTestsOID + ".<index>.2" = timestamp (Unix)
	// recentTestsOID + ".<index>.3" = success (1/0)
	// recentTestsOID + ".<index>.4" = total duration (ms)
	// recentTestsOID + ".<index>.5" = HTTP status code
)

// Trap types
const (
	trapTestFailure     = 1
	trapServiceDegraded = 2
	trapServiceRecovered = 3
)

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
		oidTree: make(map[string]oidHandler),
	}

	// Initialize OID tree with handlers
	s.initializeOIDTree()

	// Parse trap destinations if configured
	if err := s.initializeTrapDestinations(); err != nil {
		log.Printf("Warning: Failed to initialize trap destinations: %v", err)
	}

	// Start SNMP agent server
	if err := s.startSNMPServer(); err != nil {
		return nil, fmt.Errorf("failed to start SNMP server: %w", err)
	}

	// Start HTTP API server (for easier monitoring and debugging)
	if err := s.startHTTPServer(); err != nil {
		log.Printf("Warning: Failed to start HTTP API server: %v", err)
	}

	log.Printf("SNMP agent listening on %s:%d (community: %s)", cfg.ListenAddress, cfg.Port, cfg.Community)
	log.Printf("SNMP HTTP API listening on %s:%d/snmp/data", cfg.ListenAddress, cfg.Port+1)
	log.Printf("Enterprise OID: %s", cfg.EnterpriseOID)

	return s, nil
}

// initializeOIDTree sets up the OID tree with handlers for all metrics
func (s *SNMPOutput) initializeOIDTree() {
	// General statistics
	s.oidTree[cacheSize] = func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return len(s.cache)
	}

	s.oidTree[maxCacheSize] = func() interface{} {
		return s.maxSize
	}

	s.oidTree[monitoredSitesCount] = func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return len(s.stats)
	}

	s.oidTree[totalTestsRun] = func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var total int64
		for _, st := range s.stats {
			total += st.TotalTests
		}
		return total
	}

	s.oidTree[totalSuccesses] = func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var total int64
		for _, st := range s.stats {
			total += st.SuccessfulTests
		}
		return total
	}

	s.oidTree[totalFailures] = func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var total int64
		for _, st := range s.stats {
			total += st.FailedTests
		}
		return total
	}
}

// initializeTrapDestinations parses and initializes trap destinations
func (s *SNMPOutput) initializeTrapDestinations() error {
	// Trap destinations would be configured via environment variable
	// Format: host:port,host:port
	// For now, this is a placeholder for future configuration
	s.trapDestinations = make([]*gosnmp.GoSNMP, 0)
	return nil
}

// startSNMPServer starts the SNMP UDP server
func (s *SNMPOutput) startSNMPServer() error {
	addr := fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	s.snmpConn = conn

	// Start SNMP packet handler
	s.wg.Add(1)
	go s.handleSNMPPackets()

	return nil
}

// handleSNMPPackets processes incoming SNMP requests
func (s *SNMPOutput) handleSNMPPackets() {
	defer s.wg.Done()
	defer s.snmpConn.Close()

	buffer := make([]byte, 65535)

	for {
		select {
		case <-s.done:
			return
		default:
			// Set read deadline to allow checking done channel
			s.snmpConn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, remoteAddr, err := s.snmpConn.ReadFromUDP(buffer)
			if err != nil {
				// Timeout is expected, continue
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("SNMP read error: %v", err)
				continue
			}

			// Process the SNMP packet
			go s.processSNMPPacket(buffer[:n], remoteAddr)
		}
	}
}

// processSNMPPacket handles a single SNMP request
func (s *SNMPOutput) processSNMPPacket(data []byte, remoteAddr *net.UDPAddr) {
	// Decode SNMP packet
	packet, err := gosnmp.UnmarshalMessage(data)
	if err != nil {
		log.Printf("Failed to unmarshal SNMP packet: %v", err)
		return
	}

	// Verify community string
	if packet.Community != s.config.Community {
		log.Printf("SNMP request with invalid community from %s", remoteAddr)
		return
	}

	// Handle different PDU types
	var response *gosnmp.SnmpPacket
	switch packet.PDUType {
	case gosnmp.GetRequest:
		response = s.handleGetRequest(packet)
	case gosnmp.GetNextRequest:
		response = s.handleGetNextRequest(packet)
	case gosnmp.GetBulkRequest:
		response = s.handleGetBulkRequest(packet)
	default:
		log.Printf("Unsupported SNMP PDU type: %v", packet.PDUType)
		return
	}

	if response == nil {
		return
	}

	// Marshal response
	responseData, err := response.MarshalMsg()
	if err != nil {
		log.Printf("Failed to marshal SNMP response: %v", err)
		return
	}

	// Send response
	_, err = s.snmpConn.WriteToUDP(responseData, remoteAddr)
	if err != nil {
		log.Printf("Failed to send SNMP response: %v", err)
	}
}

// handleGetRequest processes SNMP GET requests
func (s *SNMPOutput) handleGetRequest(packet *gosnmp.SnmpPacket) *gosnmp.SnmpPacket {
	response := &gosnmp.SnmpPacket{
		Version:   packet.Version,
		Community: packet.Community,
		PDUType:   gosnmp.GetResponse,
		RequestID: packet.RequestID,
		Variables: make([]gosnmp.SnmpPDU, 0, len(packet.Variables)),
	}

	for _, reqVar := range packet.Variables {
		pdu := s.getOIDValue(reqVar.Name)
		response.Variables = append(response.Variables, pdu)
	}

	return response
}

// handleGetNextRequest processes SNMP GETNEXT requests
func (s *SNMPOutput) handleGetNextRequest(packet *gosnmp.SnmpPacket) *gosnmp.SnmpPacket {
	response := &gosnmp.SnmpPacket{
		Version:   packet.Version,
		Community: packet.Community,
		PDUType:   gosnmp.GetResponse,
		RequestID: packet.RequestID,
		Variables: make([]gosnmp.SnmpPDU, 0, len(packet.Variables)),
	}

	for _, reqVar := range packet.Variables {
		pdu := s.getNextOID(reqVar.Name)
		response.Variables = append(response.Variables, pdu)
	}

	return response
}

// handleGetBulkRequest processes SNMP GETBULK requests
func (s *SNMPOutput) handleGetBulkRequest(packet *gosnmp.SnmpPacket) *gosnmp.SnmpPacket {
	response := &gosnmp.SnmpPacket{
		Version:   packet.Version,
		Community: packet.Community,
		PDUType:   gosnmp.GetResponse,
		RequestID: packet.RequestID,
		Variables: make([]gosnmp.SnmpPDU, 0),
	}

	// GETBULK MaxRepetitions
	maxReps := packet.MaxRepetitions
	if maxReps == 0 {
		maxReps = 10 // Default
	}

	for _, reqVar := range packet.Variables {
		currentOID := reqVar.Name
		for i := uint32(0); i < maxReps; i++ {
			pdu := s.getNextOID(currentOID)
			if pdu.Type == gosnmp.EndOfMibView {
				break
			}
			response.Variables = append(response.Variables, pdu)
			currentOID = pdu.Name
		}
	}

	return response
}

// getOIDValue retrieves the value for a specific OID
func (s *SNMPOutput) getOIDValue(oid string) gosnmp.SnmpPDU {
	// Check if OID is in our tree
	if handler, exists := s.oidTree[oid]; exists {
		value := handler()
		return s.createSNMPPDU(oid, value)
	}

	// Check if it's a table OID (site stats or recent tests)
	if strings.HasPrefix(oid, siteStatsOID) {
		return s.getSiteStatsOID(oid)
	}

	if strings.HasPrefix(oid, recentTestsOID) {
		return s.getRecentTestOID(oid)
	}

	// OID not found
	return gosnmp.SnmpPDU{
		Name:  oid,
		Type:  gosnmp.NoSuchInstance,
		Value: nil,
	}
}

// getNextOID finds the next OID in the tree
func (s *SNMPOutput) getNextOID(oid string) gosnmp.SnmpPDU {
	// Get all OIDs and sort them
	allOIDs := s.getAllOIDs()

	// Find the next OID
	for _, nextOID := range allOIDs {
		if oidCompare(oid, nextOID) < 0 {
			return s.getOIDValue(nextOID)
		}
	}

	// End of MIB
	return gosnmp.SnmpPDU{
		Name:  oid,
		Type:  gosnmp.EndOfMibView,
		Value: nil,
	}
}

// getAllOIDs returns all available OIDs in sorted order
func (s *SNMPOutput) getAllOIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	oids := make([]string, 0)

	// Add static OIDs
	for oid := range s.oidTree {
		oids = append(oids, oid)
	}

	// Add site stats OIDs
	siteIndex := 1
	for siteName := range s.stats {
		_ = siteName // Use index instead of name for OID
		for metric := 1; metric <= 10; metric++ {
			oid := fmt.Sprintf("%s.%d.%d", siteStatsOID, siteIndex, metric)
			oids = append(oids, oid)
		}
		siteIndex++
	}

	// Add recent test OIDs
	for i := 0; i < len(s.cache); i++ {
		for metric := 1; metric <= 5; metric++ {
			oid := fmt.Sprintf("%s.%d.%d", recentTestsOID, i+1, metric)
			oids = append(oids, oid)
		}
	}

	// Sort OIDs
	sortOIDs(oids)

	return oids
}

// getSiteStatsOID retrieves site statistics OID value
func (s *SNMPOutput) getSiteStatsOID(oid string) gosnmp.SnmpPDU {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse OID: .1.3.6.1.4.1.99999.2.<index>.<metric>
	parts := strings.Split(strings.TrimPrefix(oid, siteStatsOID+"."), ".")
	if len(parts) != 2 {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	metric, err := strconv.Atoi(parts[1])
	if err != nil {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	// Get site by index
	siteNames := make([]string, 0, len(s.stats))
	for name := range s.stats {
		siteNames = append(siteNames, name)
	}

	if index < 1 || index > len(siteNames) {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	siteName := siteNames[index-1]
	st := s.stats[siteName]

	// Return metric value
	switch metric {
	case 1: // site name
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.OctetString, Value: siteName}
	case 2: // total tests
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(st.TotalTests)}
	case 3: // successful tests
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(st.SuccessfulTests)}
	case 4: // failed tests
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(st.FailedTests)}
	case 5: // last duration
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint(st.LastDurationMs)}
	case 6: // average duration
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint(st.AvgDurationMs)}
	case 7: // min duration
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint(st.MinDurationMs)}
	case 8: // max duration
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint(st.MaxDurationMs)}
	case 9: // last success time
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(st.LastSuccessTime.Unix())}
	case 10: // last failure time
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(st.LastFailureTime.Unix())}
	default:
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}
}

// getRecentTestOID retrieves recent test result OID value
func (s *SNMPOutput) getRecentTestOID(oid string) gosnmp.SnmpPDU {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse OID: .1.3.6.1.4.1.99999.3.<index>.<metric>
	parts := strings.Split(strings.TrimPrefix(oid, recentTestsOID+"."), ".")
	if len(parts) != 2 {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	metric, err := strconv.Atoi(parts[1])
	if err != nil {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	// Check index bounds
	if index < 1 || index > len(s.cache) {
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}

	result := s.cache[index-1]

	// Return metric value
	switch metric {
	case 1: // site name
		siteName := result.Site.Name
		if siteName == "" {
			siteName = result.Site.URL
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.OctetString, Value: siteName}
	case 2: // timestamp
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: uint64(result.Timestamp.Unix())}
	case 3: // success
		success := 0
		if result.Status.Success {
			success = 1
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Integer, Value: success}
	case 4: // total duration
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint(result.Timings.TotalDurationMs)}
	case 5: // HTTP status
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Integer, Value: result.Status.HTTPStatus}
	default:
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.NoSuchInstance, Value: nil}
	}
}

// oidCompare compares two OIDs lexicographically
func oidCompare(oid1, oid2 string) int {
	// Remove leading dots
	oid1 = strings.TrimPrefix(oid1, ".")
	oid2 = strings.TrimPrefix(oid2, ".")

	parts1 := strings.Split(oid1, ".")
	parts2 := strings.Split(oid2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])

		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}

	if len(parts1) < len(parts2) {
		return -1
	} else if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// sortOIDs sorts OIDs in lexicographic order
func sortOIDs(oids []string) {
	// Simple bubble sort for OIDs
	for i := 0; i < len(oids); i++ {
		for j := i + 1; j < len(oids); j++ {
			if oidCompare(oids[i], oids[j]) > 0 {
				oids[i], oids[j] = oids[j], oids[i]
			}
		}
	}
}

// startHTTPServer starts an HTTP API server for easier SNMP data access
func (s *SNMPOutput) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snmp/data", s.handleSNMPDataRequest)
	mux.HandleFunc("/snmp/mib", s.handleMIBRequest)
	mux.HandleFunc("/snmp/oids", s.handleOIDListRequest)

	addr := fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.Port+1)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("SNMP HTTP server error: %v", err)
		}
	}()

	return nil
}

// handleSNMPDataRequest handles HTTP requests for SNMP data in JSON format
func (s *SNMPOutput) handleSNMPDataRequest(w http.ResponseWriter, r *http.Request) {
	data := s.GetSNMPData()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding SNMP data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleMIBRequest handles HTTP requests for MIB data
func (s *SNMPOutput) handleMIBRequest(w http.ResponseWriter, r *http.Request) {
	mib := s.ExportMIBData()

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, mib)
}

// handleOIDListRequest handles HTTP requests for the list of available OIDs
func (s *SNMPOutput) handleOIDListRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	oids := s.getAllOIDs()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(oids); err != nil {
		log.Printf("Error encoding OID list: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

		// Send trap for test failure (async, don't block)
		go s.sendTestFailureTrap(siteName, result.Status.ErrorMessage)
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

	// Check for service degradation (high failure rate)
	if st.TotalTests > 10 {
		failureRate := float64(st.FailedTests) / float64(st.TotalTests)
		if failureRate > 0.5 {
			go s.sendServiceDegradedTrap(siteName, failureRate)
		}
	}

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

// SendTrap sends an SNMP trap for critical events
func (s *SNMPOutput) SendTrap(trapType string, message string) error {
	if s == nil || s.config == nil || len(s.trapDestinations) == 0 {
		return nil
	}

	// Build trap PDU
	pdu := gosnmp.SnmpPDU{
		Name:  s.config.EnterpriseOID + ".0.1",
		Type:  gosnmp.OctetString,
		Value: message,
	}

	// Send to all configured trap destinations
	for _, dest := range s.trapDestinations {
		trap := gosnmp.SnmpTrap{
			Variables: []gosnmp.SnmpPDU{pdu},
		}

		_, err := dest.SendTrap(trap)
		if err != nil {
			log.Printf("Failed to send SNMP trap to %s: %v", dest.Target, err)
		} else {
			log.Printf("SNMP trap sent to %s: %s", dest.Target, message)
		}
	}

	return nil
}

// sendTestFailureTrap sends a trap when a test fails
func (s *SNMPOutput) sendTestFailureTrap(siteName, errorMsg string) {
	if len(s.trapDestinations) == 0 {
		return
	}

	message := fmt.Sprintf("Test failure for %s: %s", siteName, errorMsg)
	s.SendTrap("test_failure", message)
}

// sendServiceDegradedTrap sends a trap when service is degraded (high failure rate)
func (s *SNMPOutput) sendServiceDegradedTrap(siteName string, failureRate float64) {
	if len(s.trapDestinations) == 0 {
		return
	}

	message := fmt.Sprintf("Service degraded for %s: %.1f%% failure rate", siteName, failureRate*100)
	s.SendTrap("service_degraded", message)
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

	// Signal shutdown to SNMP packet handler
	close(s.done)

	// Shutdown HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down SNMP HTTP server: %v", err)
		}
	}

	// Wait for SNMP packet handler to finish
	s.wg.Wait()

	// Close trap connections
	for _, dest := range s.trapDestinations {
		if dest.Conn != nil {
			dest.Conn.Close()
		}
	}

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
