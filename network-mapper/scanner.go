package main

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Service struct {
	Name     string
	Type     string
	Domain   string
	Port     int
	Protocol string
	TXT      map[string]string
	Source   string // "mdns", "ssdp", "upnp", etc.
}

type Device struct {
	IP         net.IP
	MAC        string
	MACVendor  string
	Hostname   string
	DeviceType string
	IsGateway  bool
	Ports      []int
	Services   []Service
	UPnPInfo   map[string]string
}

type NetworkInterface struct {
	Name    string
	IP      net.IP
	Subnet  *net.IPNet
	Gateway net.IP
}

type NetworkScanner struct {
	interfaces              []NetworkInterface
	devices                 []Device
	mu                      sync.RWMutex
	disableServiceDiscovery bool
	disableDNSLookup        bool
	scanTimeout             time.Duration
	verbose                 bool
	dnsResolver             *DNSResolver
	scanMode                ScanMode
	networkExpansion        *NetworkExpansion
	progressTracker         *ScanProgress
	scanEstimator           *ScanEstimator
	thoroughness            int
	deviceDetector          *DeviceDetector
}

func NewNetworkScanner() *NetworkScanner {
	return &NetworkScanner{
		interfaces:              make([]NetworkInterface, 0),
		devices:                 make([]Device, 0),
		disableServiceDiscovery: false,
		disableDNSLookup:        false,
		scanTimeout:             5 * time.Second,
		verbose:                 false,
		dnsResolver:             NewDNSResolver(10*time.Second, false),
		scanMode:                ScanModeIntelligent,
		networkExpansion:        NewNetworkExpansion(ScanModeIntelligent, false),
		scanEstimator:           NewScanEstimator(),
		thoroughness:            3,
	}
}

func (ns *NetworkScanner) SetOptions(disableServices bool, disableDNS bool, timeout int, verbose bool) {
	ns.disableServiceDiscovery = disableServices
	ns.disableDNSLookup = disableDNS
	ns.scanTimeout = time.Duration(timeout) * time.Second
	ns.verbose = verbose
	ns.dnsResolver = NewDNSResolver(10*time.Second, verbose)
	ns.networkExpansion = NewNetworkExpansion(ns.scanMode, verbose)
}

func (ns *NetworkScanner) SetScanMode(mode ScanMode) {
	ns.scanMode = mode
	ns.networkExpansion = NewNetworkExpansion(mode, ns.verbose)
}

func (ns *NetworkScanner) SetThoroughness(level int) {
	ns.thoroughness = level
}

func (ns *NetworkScanner) SetDeviceRulesPath(path string) {
	detector, err := NewDeviceDetector(ns.verbose, path)
	if err != nil {
		fmt.Printf("⚠️  Failed to load device detector: %v\n", err)
		fmt.Println("   Falling back to embedded rules")
		detector, _ = NewDeviceDetector(ns.verbose, "")
	}
	ns.deviceDetector = detector

	if ns.verbose && detector != nil {
		ruleCount, version := detector.GetRulesInfo()
		fmt.Printf("🔍 Device detector initialized with %d rules (v%s)\n", ruleCount, version)
	}
}

func (ns *NetworkScanner) Run() {
	fmt.Printf("🌐 Network Mapper v1.0 - %s\n", runtime.GOOS)
	fmt.Println("==========================================")

	fmt.Println("📡 Discovering network interfaces...")
	if err := ns.discoverInterfaces(); err != nil {
		fmt.Printf("❌ Error discovering interfaces: %v\n", err)
		return
	}

	fmt.Printf("✅ Found %d network interfaces\n", len(ns.interfaces))
	ns.displayInterfaces()

	fmt.Println("\n🔍 Scanning for devices...")
	ns.scanDevices()

	fmt.Printf("✅ Found %d devices\n", len(ns.devices))

	if !ns.disableDNSLookup {
		fmt.Println("\n🌐 Performing reverse DNS lookups...")
		ns.performBulkDNSLookup()
	} else {
		fmt.Println("⏭️  Skipping DNS lookups (disabled)")
	}

	if !ns.disableServiceDiscovery {
		fmt.Println("\n🔍 Initializing service discovery...")
		mvl := NewMACVendorLookup()
		mvl.Initialize()

		sd := NewServiceDiscoveryWithVerbose(ns.verbose)
		sd.DiscoverServices(ns.interfaces, ns.scanTimeout)

		fmt.Println("\n🔗 Merging service information...")
		ns.devices = sd.MergeWithDevices(ns.devices)
		ns.enhanceDevicesWithVendorInfo(ns.devices, mvl)

		fmt.Println("🧠 Enhancing device identification...")
		for i := range ns.devices {
			if ns.deviceDetector != nil {
				// Use new YAML-based detection
				deviceType := ns.deviceDetector.DetectDeviceType(&ns.devices[i])
				ns.devices[i].DeviceType = deviceType
			} else {
				// Fallback to old method if detector failed to initialize
				ns.enhanceDeviceTypeWithServices(&ns.devices[i])
			}
		}

		fmt.Println("💾 Scanning DHCP lease information...")
		dhcpLeases := ns.scanDHCPLeases()
		ns.enhanceDevicesWithDHCPInfo(ns.devices, dhcpLeases)

		fmt.Printf("✅ Enhanced %d devices with service information\n", len(ns.devices))
	} else {
		fmt.Println("⏭️  Skipping service discovery (disabled)")
	}

	// Log unknown devices for future research (if enabled)
	if ns.deviceDetector != nil {
		ns.logUnknownDevices()
	}

	fmt.Println("\n🎨 Generating network visualization...")
	ns.visualizeNetwork()
}

func (ns *NetworkScanner) discoverInterfaces() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			var ipnet *net.IPNet

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				ipnet = v
			case *net.IPAddr:
				ip = v.IP
				ipnet = &net.IPNet{IP: ip, Mask: ip.DefaultMask()}
			}

			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			gateway := ns.findGateway(ipnet)

			netInterface := NetworkInterface{
				Name:    iface.Name,
				IP:      ip,
				Subnet:  ipnet,
				Gateway: gateway,
			}

			ns.interfaces = append(ns.interfaces, netInterface)
		}
	}

	return nil
}

func (ns *NetworkScanner) findGateway(subnet *net.IPNet) net.IP {
	switch runtime.GOOS {
	case "linux":
		return ns.findGatewayLinux(subnet)
	case "darwin":
		return ns.findGatewayDarwin(subnet)
	case "windows":
		return ns.findGatewayWindows(subnet)
	default:
		ip := subnet.IP.To4()
		if ip != nil {
			gateway := make(net.IP, len(ip))
			copy(gateway, ip)
			gateway[3] = 1
			return gateway
		}
		return nil
	}
}

func (ns *NetworkScanner) displayInterfaces() {
	for i, iface := range ns.interfaces {
		fmt.Printf("  [%d] %s: %s/%s (Gateway: %s)\n",
			i+1, iface.Name, iface.IP.String(),
			iface.Subnet.Mask.String(), iface.Gateway.String())
	}
}

func (ns *NetworkScanner) scanDevices() {
	// Use intelligent discovery for comprehensive, firewall-test, and intelligent modes
	if ns.scanMode == ScanModeComprehensive || ns.scanMode == ScanModeFirewallTest || ns.scanMode == ScanModeIntelligent {
		ns.scanDevicesIntelligent()
		return
	}

	// Use traditional scanning for quick and normal modes
	ns.scanDevicesTraditional()
}

func (ns *NetworkScanner) scanDevicesIntelligent() {
	fmt.Println("🧠 Using intelligent subnet discovery...")

	// Create intelligent discovery instance with thoroughness level
	intelligentDiscovery := NewIntelligentDiscoveryWithThoroughness(ns.verbose, ns.scanTimeout, ns.thoroughness)

	// Discover active subnets using heuristics
	activeSubnets := intelligentDiscovery.DiscoverActiveSubnets(ns.interfaces)

	// Convert to scan ranges
	var scanRanges []ScanRange
	var totalIPs uint32

	for i, subnet := range activeSubnets {
		scanRange := ScanRange{
			Network:     subnet.Network,
			Priority:    subnet.Priority,
			Description: fmt.Sprintf("Intelligent discovery - %s", subnet.Source),
		}
		scanRanges = append(scanRanges, scanRange)

		// Calculate IPs in this range
		startIP := subnet.Network.IP
		endIP := ns.getLastIP(subnet.Network)
		rangeIPs := ns.countIPsInRange(startIP, endIP)
		totalIPs += rangeIPs

		// Show more details for intelligent scanning (increased output)
		maxShow := 5
		if !ns.verbose {
			maxShow = 8 // Show more in non-verbose mode for intelligent scanning
		}

		if i < maxShow || ns.verbose {
			fmt.Printf("🎯 Will scan: %s (%d IPs, %s)\n",
				subnet.Network.String(), rangeIPs, subnet.Source)
		}
	}

	if len(activeSubnets) > 8 && !ns.verbose {
		fmt.Printf("   ... and %d more subnets (use --verbose to see all)\n", len(activeSubnets)-8)
	} else if len(activeSubnets) > 5 && ns.verbose {
		fmt.Printf("   ... and %d more subnets\n", len(activeSubnets)-5)
	}

	// Initialize progress tracking
	ns.progressTracker = NewScanProgress(len(scanRanges), totalIPs, ns.verbose)

	// Show scan information
	fmt.Printf("📊 Scan mode: Intelligent discovery (%s)\n", ns.networkExpansion.GetScanModeDescription(ns.scanMode))
	fmt.Printf("🎯 Discovered %d active subnets covering %d IP addresses\n", len(scanRanges), totalIPs)
	ns.printScanRanges(scanRanges)

	estimate := ns.scanEstimator.GetEstimateDescription(totalIPs, ns.scanMode)
	fmt.Printf("⏱️  Estimated scan time: %s\n", estimate)

	fmt.Println() // Add space before progress tracking

	// Scan the discovered ranges
	var wg sync.WaitGroup

	for i, scanRange := range scanRanges {
		wg.Add(1)
		go func(rangeIndex int, sr ScanRange) {
			defer wg.Done()
			ns.scanRangeWithProgress(rangeIndex, sr)
		}(i, scanRange)
	}

	wg.Wait()

	// Show final progress and summary
	ns.progressTracker.ForceUpdate()
	ns.progressTracker.ShowFinalSummary()
}

func (ns *NetworkScanner) scanDevicesTraditional() {
	// Get expanded scan ranges based on current scan mode
	scanRanges := ns.networkExpansion.ExpandScanRanges(ns.interfaces)

	// Calculate total IPs to scan and provide estimate
	var totalIPs uint32
	for _, sr := range scanRanges {
		startIP := sr.Network.IP
		endIP := ns.getLastIP(sr.Network)
		rangeIPs := ns.countIPsInRange(startIP, endIP)
		totalIPs += rangeIPs
	}

	// Initialize progress tracking
	ns.progressTracker = NewScanProgress(len(scanRanges), totalIPs, ns.verbose)

	// Show scan information
	fmt.Printf("📊 Scan mode: %s\n", ns.networkExpansion.GetScanModeDescription(ns.scanMode))
	fmt.Printf("🎯 Expanded to %d scan range(s) covering %d IP addresses\n", len(scanRanges), totalIPs)
	ns.printScanRanges(scanRanges)

	estimate := ns.scanEstimator.GetEstimateDescription(totalIPs, ns.scanMode)
	fmt.Printf("⏱️  Estimated scan time: %s\n", estimate)

	if ns.verbose {
		fmt.Printf("📋 Scan ranges:\n")
		for i, sr := range scanRanges {
			rangeIPs := ns.countIPsInRange(sr.Network.IP, ns.getLastIP(sr.Network))
			fmt.Printf("   [%d] %s - %s (%d IPs, priority %d)\n",
				i+1, sr.Network.String(), sr.Description, rangeIPs, sr.Priority)
		}
	}

	fmt.Println() // Add space before progress tracking

	var wg sync.WaitGroup

	for i, scanRange := range scanRanges {
		wg.Add(1)
		go func(rangeIndex int, sr ScanRange) {
			defer wg.Done()
			ns.scanRangeWithProgress(rangeIndex, sr)
		}(i, scanRange)
	}

	wg.Wait()

	// Show final progress and summary
	ns.progressTracker.ForceUpdate()
	ns.progressTracker.ShowFinalSummary()
}

func (ns *NetworkScanner) scanRangeWithProgress(rangeIndex int, scanRange ScanRange) {
	// Calculate the range of IPs to scan
	startIP := scanRange.Network.IP
	endIP := ns.getLastIP(scanRange.Network)
	totalIPs := ns.countIPsInRange(startIP, endIP)

	// Start range tracking with CIDR information
	ns.progressTracker.StartRangeWithCIDR(rangeIndex, scanRange.Description, scanRange.Network.String(), totalIPs)

	if totalIPs > 10000 && ns.verbose {
		fmt.Printf("⚠️  Large range detected (%d IPs). Consider using --scan-mode quick for faster scanning.\n", totalIPs)
	}

	// Track devices found in this range
	devicesFoundStart := len(ns.devices)

	// Scan IPs in the range
	current := make(net.IP, len(startIP))
	copy(current, startIP)

	var scanWg sync.WaitGroup
	semaphore := make(chan struct{}, 100) // Limit concurrent scans
	var scannedCount uint32

	// Channel to track active scans for progress display
	activeCounter := make(chan int, 200)
	go func() {
		activeCount := 0
		for delta := range activeCounter {
			activeCount += delta
			ns.progressTracker.SetActiveScans(activeCount)
		}
	}()

	for !current.Equal(endIP) {
		// Skip network and broadcast addresses
		if ns.isNetworkOrBroadcast(current, scanRange.Network) {
			ns.incrementIP(current)
			continue
		}

		scanWg.Add(1)
		semaphore <- struct{}{}
		activeCounter <- 1 // Increment active scans

		go func(ip net.IP) {
			defer scanWg.Done()
			defer func() {
				<-semaphore
				activeCounter <- -1 // Decrement active scans
			}()

			if ns.pingHost(ip.String()) {
				ports := ns.scanCommonPorts(ip)

				// Find which original interface this relates to for gateway detection
				isGateway := false
				for _, iface := range ns.interfaces {
					if ip.Equal(iface.Gateway) {
						isGateway = true
						break
					}
				}

				device := Device{
					IP:         make(net.IP, len(ip)),
					MAC:        ns.getMACAddress(ip),
					MACVendor:  "",
					Hostname:   "",        // Will be populated in bulk DNS lookup
					DeviceType: "Unknown", // Will be set by device detector later
					IsGateway:  isGateway,
					Ports:      ports,
					Services:   make([]Service, 0),
					UPnPInfo:   make(map[string]string),
				}
				copy(device.IP, ip)

				ns.mu.Lock()
				ns.devices = append(ns.devices, device)
				ns.mu.Unlock()
			}

			// Update progress every few IPs to avoid overwhelming the display
			count := atomic.AddUint32(&scannedCount, 1)
			if count%10 == 0 {
				ns.progressTracker.IncrementScanned(10)
			}
		}(ns.copyIP(current))

		ns.incrementIP(current)
		if ns.isAfter(current, endIP) {
			break
		}
	}

	scanWg.Wait()
	close(activeCounter)

	// Final progress update for this range
	finalCount := atomic.LoadUint32(&scannedCount)
	if remaining := finalCount % 10; remaining > 0 {
		ns.progressTracker.IncrementScanned(remaining)
	}

	// Complete range tracking
	devicesFound := len(ns.devices) - devicesFoundStart
	ns.progressTracker.CompleteRangeWithCIDR(rangeIndex, scanRange.Description, scanRange.Network.String(), devicesFound)
}

// Helper methods for IP range scanning
func (ns *NetworkScanner) getLastIP(network *net.IPNet) net.IP {
	ip := network.IP.To4()
	mask := network.Mask
	lastIP := make(net.IP, len(ip))

	for i := 0; i < len(ip); i++ {
		lastIP[i] = ip[i] | ^mask[i]
	}

	return lastIP
}

func (ns *NetworkScanner) countIPsInRange(start, end net.IP) uint32 {
	startInt := ns.ipToUint32(start)
	endInt := ns.ipToUint32(end)
	if endInt >= startInt {
		return endInt - startInt + 1
	}
	return 0
}

func (ns *NetworkScanner) ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 + uint32(ip[1])<<16 + uint32(ip[2])<<8 + uint32(ip[3])
}

func (ns *NetworkScanner) incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func (ns *NetworkScanner) copyIP(ip net.IP) net.IP {
	copied := make(net.IP, len(ip))
	copy(copied, ip)
	return copied
}

func (ns *NetworkScanner) isAfter(ip1, ip2 net.IP) bool {
	return ns.ipToUint32(ip1) > ns.ipToUint32(ip2)
}

func (ns *NetworkScanner) isNetworkOrBroadcast(ip net.IP, network *net.IPNet) bool {
	// Skip network address (first IP)
	if ip.Equal(network.IP) {
		return true
	}

	// Skip broadcast address (last IP)
	lastIP := ns.getLastIP(network)
	return ip.Equal(lastIP)
}

func (ns *NetworkScanner) performBulkDNSLookup() {
	// Collect all IPs that need DNS lookup
	var ips []net.IP
	for _, device := range ns.devices {
		ips = append(ips, device.IP)
	}

	// Perform bulk DNS lookup
	hostnames := ns.dnsResolver.BulkLookup(ips)

	// Update devices with hostnames
	for i := range ns.devices {
		ipStr := ns.devices[i].IP.String()
		if hostname, exists := hostnames[ipStr]; exists && hostname != "" {
			ns.devices[i].Hostname = hostname
		}
	}

	// Show DNS lookup statistics
	successful, total := ns.dnsResolver.GetCacheStats()
	fmt.Printf("✅ DNS lookups complete: %d/%d successful\n", successful, total)
}

// logUnknownDevices writes unknown devices to a log file for future research
func (ns *NetworkScanner) logUnknownDevices() {
	var unknownDevices []Device

	for _, device := range ns.devices {
		if device.DeviceType == "Unknown" || device.DeviceType == "" {
			unknownDevices = append(unknownDevices, device)
		}
	}

	if len(unknownDevices) == 0 {
		return
	}

	// TODO: Implement structured data collection for research agent
	// This could write to a structured log file (JSON/YAML) that the
	// background research agent can process later
	if ns.verbose {
		fmt.Printf("📊 Found %d unknown devices for future research\n", len(unknownDevices))
		fmt.Println("💾 Research data ready for background agent processing")
	}
}

// printScanRanges displays the CIDR ranges that will be scanned
func (ns *NetworkScanner) printScanRanges(scanRanges []ScanRange) {
	fmt.Printf("📡 Target ranges:\n")
	for i, sr := range scanRanges {
		rangeIPs := ns.countIPsInRange(sr.Network.IP, ns.getLastIP(sr.Network))
		if i < 5 || ns.verbose { // Show first 5 ranges in non-verbose, all if verbose
			fmt.Printf("   • %s (%d IPs) - %s\n", sr.Network.String(), rangeIPs, sr.Description)
		} else if i == 5 && !ns.verbose {
			fmt.Printf("   • ... and %d more ranges (use --verbose to see all)\n", len(scanRanges)-5)
			break
		}
	}
}
