package main

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type SubnetCandidate struct {
	Network     *net.IPNet
	Priority    int
	Source      string // "interface", "adjacent", "gateway", "discovered"
	GatewayIP   net.IP
	IsActive    bool
	DeviceCount int
}

type IntelligentDiscovery struct {
	candidates   []SubnetCandidate
	mu           sync.RWMutex
	verbose      bool
	timeout      time.Duration
	thoroughness int // 1-5, higher = more thorough
}

func NewIntelligentDiscovery(verbose bool, timeout time.Duration) *IntelligentDiscovery {
	return NewIntelligentDiscoveryWithThoroughness(verbose, timeout, 3)
}

func NewIntelligentDiscoveryWithThoroughness(verbose bool, timeout time.Duration, thoroughness int) *IntelligentDiscovery {
	return &IntelligentDiscovery{
		candidates:   make([]SubnetCandidate, 0),
		verbose:      verbose,
		timeout:      timeout,
		thoroughness: thoroughness,
	}
}

// DiscoverActiveSubnets intelligently discovers active subnets using heuristics
func (id *IntelligentDiscovery) DiscoverActiveSubnets(interfaces []NetworkInterface) []SubnetCandidate {
	if id.verbose {
		fmt.Println("🧠 Starting intelligent subnet discovery...")
	}

	// Phase 1: Add interface subnets as high-priority candidates
	id.addInterfaceSubnets(interfaces)

	// Phase 2: Generate adjacent subnet candidates
	id.generateAdjacentSubnets(interfaces)

	// Phase 3: Generate common network range candidates
	id.generateCommonSubnets(interfaces)

	// Phase 4: Probe gateway IPs to validate active subnets
	id.probeGatewayIPs()

	// Phase 5: Sort by priority and activity
	id.sortCandidates()

	// Phase 6: Filter to only active subnets or high-priority ones
	activeSubnets := id.getActiveSubnets()

	// Always show discovery results for intelligent mode (increased output)
	fmt.Printf("🎯 Intelligent discovery found %d active subnets from %d candidates\n",
		len(activeSubnets), len(id.candidates))

	// Show more detail when using intelligent discovery
	maxShow := 8 // Increased from 5
	if id.verbose {
		maxShow = len(activeSubnets) // Show all in verbose mode
	}

	for i, subnet := range activeSubnets {
		if i < maxShow {
			if id.verbose {
				fmt.Printf("   [%d] %s (Priority: %d, Source: %s, Active: %v)\n",
					i+1, subnet.Network.String(), subnet.Priority, subnet.Source, subnet.IsActive)
			} else {
				fmt.Printf("   [%d] %s (Source: %s)\n",
					i+1, subnet.Network.String(), subnet.Source)
			}
		}
	}
	if len(activeSubnets) > maxShow {
		fmt.Printf("   ... and %d more (use --verbose for details)\n", len(activeSubnets)-maxShow)
	}

	return activeSubnets
}

// addInterfaceSubnets adds the current interface subnets as highest priority
func (id *IntelligentDiscovery) addInterfaceSubnets(interfaces []NetworkInterface) {
	for _, iface := range interfaces {
		// For large interface subnets (larger than /24), create a smaller targeted subnet
		// around the interface IP to avoid scanning huge ranges
		network := iface.Subnet
		maskSize, _ := iface.Subnet.Mask.Size()

		// If the interface subnet is larger than /24, create a /24 around the interface IP
		if maskSize < 24 {
			ip := iface.IP.To4()
			if ip != nil {
				// Create a /24 subnet around the interface IP
				targetedIP := net.IPv4(ip[0], ip[1], ip[2], 0)
				network = &net.IPNet{
					IP:   targetedIP,
					Mask: net.CIDRMask(24, 32),
				}

				if id.verbose {
					fmt.Printf("🎯 Large interface subnet detected (%s), targeting /24 around interface: %s\n",
						iface.Subnet.String(), network.String())
				}
			}
		}

		candidate := SubnetCandidate{
			Network:   network,
			Priority:  100, // Highest priority
			Source:    "interface",
			GatewayIP: iface.Gateway,
			IsActive:  true, // Assume interface subnets are active
		}
		id.candidates = append(id.candidates, candidate)
	}
}

// generateAdjacentSubnets creates candidates for subnets adjacent to interface subnets
func (id *IntelligentDiscovery) generateAdjacentSubnets(interfaces []NetworkInterface) {
	for _, iface := range interfaces {
		adjacents := id.getAdjacentSubnets(iface.Subnet)
		for _, subnet := range adjacents {
			candidate := SubnetCandidate{
				Network:   subnet,
				Priority:  80, // High priority
				Source:    "adjacent",
				GatewayIP: id.guessGatewayIP(subnet),
			}
			id.candidates = append(id.candidates, candidate)
		}
	}
}

// generateCommonSubnets creates candidates for common network ranges
func (id *IntelligentDiscovery) generateCommonSubnets(interfaces []NetworkInterface) {
	// For each interface, generate common subnet variations
	for _, iface := range interfaces {
		ip := iface.IP.To4()
		if ip == nil {
			continue
		}

		// Generate common /24 subnets in the same /16 range
		for thirdOctet := 0; thirdOctet <= 255; thirdOctet += 1 {
			// Skip the current interface subnet
			if thirdOctet == int(ip[2]) {
				continue
			}

			// Thoroughness affects which ranges we include
			if !id.isCommonThirdOctetForThoroughness(thirdOctet) {
				continue
			}

			subnetIP := net.IPv4(ip[0], ip[1], byte(thirdOctet), 0)
			subnet := &net.IPNet{
				IP:   subnetIP,
				Mask: net.CIDRMask(24, 32),
			}

			candidate := SubnetCandidate{
				Network:   subnet,
				Priority:  60, // Medium priority
				Source:    "common",
				GatewayIP: id.guessGatewayIP(subnet),
			}
			id.candidates = append(id.candidates, candidate)
		}
	}
}

// isCommonThirdOctetForThoroughness returns true for commonly used third octets based on thoroughness
func (id *IntelligentDiscovery) isCommonThirdOctetForThoroughness(octet int) bool {
	// Define ranges by thoroughness level
	switch id.thoroughness {
	case 1: // Minimal - only most common
		common := []int{0, 1}
		return id.containsInt(common, octet)
	case 2: // Light
		common := []int{0, 1, 10, 100}
		return id.containsInt(common, octet)
	case 3: // Default
		common := []int{0, 1, 10, 20, 30, 50, 100, 168, 254}
		return id.containsInt(common, octet)
	case 4: // Thorough
		common := []int{0, 1, 2, 5, 10, 11, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 150, 168, 200, 250, 254}
		return id.containsInt(common, octet)
	case 5: // Exhaustive - try many more ranges
		// For level 5, check every 10th number plus commons
		if octet%10 == 0 || octet%10 == 1 || octet%10 == 5 {
			return true
		}
		common := []int{0, 1, 2, 5, 10, 11, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 150, 168, 200, 250, 254}
		return id.containsInt(common, octet)
	default:
		return id.isCommonThirdOctet(octet)
	}
}

// Helper function for old behavior
func (id *IntelligentDiscovery) isCommonThirdOctet(octet int) bool {
	common := []int{0, 1, 10, 20, 30, 50, 100, 168, 200, 254}
	return id.containsInt(common, octet)
}

// containsInt checks if slice contains integer
func (id *IntelligentDiscovery) containsInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// getAdjacentSubnets returns subnets adjacent to the given subnet
func (id *IntelligentDiscovery) getAdjacentSubnets(subnet *net.IPNet) []*net.IPNet {
	var adjacents []*net.IPNet
	ip := subnet.IP.To4()
	if ip == nil {
		return adjacents
	}

	maskSize, _ := subnet.Mask.Size()

	// For /24 networks, check adjacent /24s
	if maskSize == 24 {
		// Previous /24
		if ip[2] > 0 {
			prevIP := net.IPv4(ip[0], ip[1], ip[2]-1, 0)
			prevSubnet := &net.IPNet{IP: prevIP, Mask: subnet.Mask}
			adjacents = append(adjacents, prevSubnet)
		}

		// Next /24
		if ip[2] < 255 {
			nextIP := net.IPv4(ip[0], ip[1], ip[2]+1, 0)
			nextSubnet := &net.IPNet{IP: nextIP, Mask: subnet.Mask}
			adjacents = append(adjacents, nextSubnet)
		}
	}

	// For /16 networks, check common /24 subnets within
	if maskSize == 16 {
		commonThirds := []byte{0, 1, 10, 20, 50, 100, 168, 254}
		for _, third := range commonThirds {
			subnetIP := net.IPv4(ip[0], ip[1], third, 0)
			subnet24 := &net.IPNet{IP: subnetIP, Mask: net.CIDRMask(24, 32)}
			adjacents = append(adjacents, subnet24)
		}
	}

	return adjacents
}

// guessGatewayIP guesses the most likely gateway IP for a subnet
func (id *IntelligentDiscovery) guessGatewayIP(subnet *net.IPNet) net.IP {
	ip := subnet.IP.To4()
	if ip == nil {
		return nil
	}

	// Most common: .1
	gateway := net.IPv4(ip[0], ip[1], ip[2], 1)
	return gateway
}

// probeGatewayIPs probes gateway IPs to determine if subnets are active
func (id *IntelligentDiscovery) probeGatewayIPs() {
	if id.verbose {
		fmt.Printf("🔍 Probing %d subnet candidates for activity...\n", len(id.candidates))
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit concurrent probes

	for i := range id.candidates {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			candidate := &id.candidates[index]

			// Try multiple common gateway IPs
			gatewayIPs := id.getCommonGatewayIPs(candidate.Network)

			for _, gwIP := range gatewayIPs {
				if id.pingHost(gwIP.String()) {
					id.mu.Lock()
					candidate.IsActive = true
					candidate.GatewayIP = gwIP
					id.mu.Unlock()

					if id.verbose {
						fmt.Printf("✅ Found active gateway %s in subnet %s\n",
							gwIP.String(), candidate.Network.String())
					}
					break
				}
			}
		}(i)
	}

	wg.Wait()
}

// getCommonGatewayIPs returns common gateway IP addresses for a subnet
func (id *IntelligentDiscovery) getCommonGatewayIPs(subnet *net.IPNet) []net.IP {
	ip := subnet.IP.To4()
	if ip == nil {
		return nil
	}

	var gateways []net.IP

	// Common gateway addresses in order of likelihood
	commonLastOctets := []byte{1, 254, 10, 100, 50}

	for _, lastOctet := range commonLastOctets {
		gateway := net.IPv4(ip[0], ip[1], ip[2], lastOctet)
		// Make sure it's within the subnet
		if subnet.Contains(gateway) {
			gateways = append(gateways, gateway)
		}
	}

	return gateways
}

// pingHost checks if a host responds to ping (reuse existing implementation)
func (id *IntelligentDiscovery) pingHost(host string) bool {
	// We'll reference the existing pingHost method from NetworkScanner
	// For now, we'll implement a simple version
	return pingHostQuick(host, id.timeout)
}

// sortCandidates sorts candidates by priority and activity
func (id *IntelligentDiscovery) sortCandidates() {
	sort.Slice(id.candidates, func(i, j int) bool {
		// Active subnets first
		if id.candidates[i].IsActive != id.candidates[j].IsActive {
			return id.candidates[i].IsActive
		}
		// Then by priority
		return id.candidates[i].Priority > id.candidates[j].Priority
	})
}

// getActiveSubnets returns only active subnets plus high-priority inactive ones
func (id *IntelligentDiscovery) getActiveSubnets() []SubnetCandidate {
	var active []SubnetCandidate

	for _, candidate := range id.candidates {
		// Include if active OR if it's a high-priority interface subnet
		if candidate.IsActive || (candidate.Priority >= 100) {
			active = append(active, candidate)
		}
	}

	// Limit based on thoroughness to avoid excessive scanning
	maxSubnets := id.getMaxSubnetsForThoroughness()
	if len(active) > maxSubnets {
		active = active[:maxSubnets]
	}

	return active
}

// getMaxSubnetsForThoroughness returns the maximum number of subnets based on thoroughness
func (id *IntelligentDiscovery) getMaxSubnetsForThoroughness() int {
	switch id.thoroughness {
	case 1:
		return 5 // Very limited
	case 2:
		return 10 // Light
	case 3:
		return 20 // Default
	case 4:
		return 40 // Thorough
	case 5:
		return 80 // Exhaustive
	default:
		return 20
	}
}

// Quick ping implementation with shorter timeout
func pingHostQuick(host string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "80"), timeout/2)
	if err == nil {
		conn.Close()
		return true
	}

	// Try ICMP-style check by attempting connection to common ports
	commonPorts := []string{"22", "53", "80", "443"}
	for _, port := range commonPorts {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout/10)
		if err == nil {
			conn.Close()
			return true
		}
	}

	return false
}
