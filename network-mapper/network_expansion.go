package main

import (
	"fmt"
	"net"
	"sort"
)

// RFC1918 private network ranges
var (
	RFC1918Networks = []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},       // 10.0.0.0/8
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},    // 172.16.0.0/12
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},   // 192.168.0.0/16
	}
)

type ScanMode int

const (
	ScanModeQuick ScanMode = iota        // Only immediate subnet
	ScanModeNormal                       // Intelligent expansion within RFC1918
	ScanModeComprehensive                // Full RFC1918 range scanning
	ScanModeFirewallTest                 // Targeted firewall testing ranges
)

type NetworkExpansion struct {
	mode    ScanMode
	verbose bool
}

func NewNetworkExpansion(mode ScanMode, verbose bool) *NetworkExpansion {
	return &NetworkExpansion{
		mode:    mode,
		verbose: verbose,
	}
}

// ExpandScanRanges takes the detected network interfaces and expands them
// into comprehensive scan ranges based on the scan mode
func (ne *NetworkExpansion) ExpandScanRanges(interfaces []NetworkInterface) []ScanRange {
	var ranges []ScanRange

	for _, iface := range interfaces {
		expanded := ne.expandInterface(iface)
		ranges = append(ranges, expanded...)
	}

	// Deduplicate and merge overlapping ranges
	return ne.mergeRanges(ranges)
}

type ScanRange struct {
	Network     *net.IPNet
	Description string
	Priority    int // Lower = higher priority
	Source      string // Which interface triggered this range
}

func (ne *NetworkExpansion) expandInterface(iface NetworkInterface) []ScanRange {
	switch ne.mode {
	case ScanModeQuick:
		return ne.expandQuick(iface)
	case ScanModeNormal:
		return ne.expandNormal(iface)
	case ScanModeComprehensive:
		return ne.expandComprehensive(iface)
	case ScanModeFirewallTest:
		return ne.expandFirewallTest(iface)
	default:
		return ne.expandNormal(iface)
	}
}

func (ne *NetworkExpansion) expandQuick(iface NetworkInterface) []ScanRange {
	return []ScanRange{
		{
			Network:     iface.Subnet,
			Description: fmt.Sprintf("Interface %s subnet", iface.Name),
			Priority:    1,
			Source:      iface.Name,
		},
	}
}

func (ne *NetworkExpansion) expandNormal(iface NetworkInterface) []ScanRange {
	ranges := []ScanRange{
		{
			Network:     iface.Subnet,
			Description: fmt.Sprintf("Interface %s subnet", iface.Name),
			Priority:    1,
			Source:      iface.Name,
		},
	}

	// If we're on an RFC1918 network, intelligently expand
	if rfc1918 := ne.findRFC1918Parent(iface.IP); rfc1918 != nil {
		expanded := ne.generateIntelligentRanges(iface, rfc1918)
		ranges = append(ranges, expanded...)
	}

	return ranges
}

func (ne *NetworkExpansion) expandComprehensive(iface NetworkInterface) []ScanRange {
	ranges := ne.expandNormal(iface)

	// Add full RFC1918 ranges if we're on a private network
	if ne.isRFC1918(iface.IP) {
		for _, rfc1918 := range RFC1918Networks {
			if rfc1918.Contains(iface.IP) {
				ranges = append(ranges, ScanRange{
					Network:     rfc1918,
					Description: fmt.Sprintf("Full RFC1918 range %s", rfc1918.String()),
					Priority:    10,
					Source:      iface.Name,
				})
				break
			}
		}
	}

	return ranges
}

func (ne *NetworkExpansion) expandFirewallTest(iface NetworkInterface) []ScanRange {
	ranges := ne.expandNormal(iface)

	// Add specific ranges for firewall testing
	if ne.isRFC1918(iface.IP) {
		// Add common internal ranges that might be segregated
		testRanges := []struct {
			cidr        string
			description string
		}{
			{"10.0.0.0/16", "Common 10.x admin range"},
			{"10.1.0.0/16", "Common 10.x DMZ range"},
			{"172.16.0.0/20", "Common 172.16.x range"},
			{"192.168.1.0/24", "Common home router range"},
			{"192.168.0.0/24", "Common default range"},
		}

		for _, tr := range testRanges {
			_, network, err := net.ParseCIDR(tr.cidr)
			if err != nil {
				continue
			}

			// Only add if it's different from our current subnet and within the same RFC1918 space
			if !network.IP.Equal(iface.Subnet.IP) && network.Contains(iface.IP) {
				ranges = append(ranges, ScanRange{
					Network:     network,
					Description: tr.description,
					Priority:    5,
					Source:      iface.Name,
				})
			}
		}
	}

	return ranges
}

func (ne *NetworkExpansion) generateIntelligentRanges(iface NetworkInterface, rfc1918 *net.IPNet) []ScanRange {
	var ranges []ScanRange

	// For 10.x.x.x networks, try common patterns
	if rfc1918.IP.Equal(net.IPv4(10, 0, 0, 0)) {
		ranges = append(ranges, ne.generate10Networks(iface)...)
	}

	// For 172.16.x.x networks, try /20 and /16 expansions
	if rfc1918.IP.Equal(net.IPv4(172, 16, 0, 0)) {
		ranges = append(ranges, ne.generate172Networks(iface)...)
	}

	// For 192.168.x.x networks, try adjacent subnets
	if rfc1918.IP.Equal(net.IPv4(192, 168, 0, 0)) {
		ranges = append(ranges, ne.generate192Networks(iface)...)
	}

	return ranges
}

func (ne *NetworkExpansion) generate10Networks(iface NetworkInterface) []ScanRange {
	var ranges []ScanRange
	ip := iface.IP.To4()

	// Try the broader /16 network (10.X.0.0/16)
	broader := &net.IPNet{
		IP:   net.IPv4(ip[0], ip[1], 0, 0),
		Mask: net.CIDRMask(16, 32),
	}

	if !broader.IP.Equal(iface.Subnet.IP) {
		ranges = append(ranges, ScanRange{
			Network:     broader,
			Description: fmt.Sprintf("Expanded /16 range for %s", iface.Name),
			Priority:    3,
			Source:      iface.Name,
		})
	}

	// Try the even broader /12 network (10.X.0.0/12) if we're in a small subnet
	ones, _ := iface.Subnet.Mask.Size()
	if ones > 16 {
		evenBroader := &net.IPNet{
			IP:   net.IPv4(ip[0], ip[1]&0xF0, 0, 0),
			Mask: net.CIDRMask(12, 32),
		}
		ranges = append(ranges, ScanRange{
			Network:     evenBroader,
			Description: fmt.Sprintf("Expanded /12 range for %s", iface.Name),
			Priority:    7,
			Source:      iface.Name,
		})
	}

	return ranges
}

func (ne *NetworkExpansion) generate172Networks(iface NetworkInterface) []ScanRange {
	var ranges []ScanRange
	ip := iface.IP.To4()

	// Try /20 expansion (172.X.Y.0/20)
	broader := &net.IPNet{
		IP:   net.IPv4(ip[0], ip[1], ip[2]&0xF0, 0),
		Mask: net.CIDRMask(20, 32),
	}

	if !broader.IP.Equal(iface.Subnet.IP) {
		ranges = append(ranges, ScanRange{
			Network:     broader,
			Description: fmt.Sprintf("Expanded /20 range for %s", iface.Name),
			Priority:    3,
			Source:      iface.Name,
		})
	}

	return ranges
}

func (ne *NetworkExpansion) generate192Networks(iface NetworkInterface) []ScanRange {
	var ranges []ScanRange
	ip := iface.IP.To4()

	// Try adjacent /24 networks
	for offset := -2; offset <= 2; offset++ {
		if offset == 0 {
			continue // Skip our own network
		}

		newThirdOctet := int(ip[2]) + offset
		if newThirdOctet < 0 || newThirdOctet > 255 {
			continue
		}

		adjacent := &net.IPNet{
			IP:   net.IPv4(ip[0], ip[1], byte(newThirdOctet), 0),
			Mask: net.CIDRMask(24, 32),
		}

		ranges = append(ranges, ScanRange{
			Network:     adjacent,
			Description: fmt.Sprintf("Adjacent network %s", adjacent.String()),
			Priority:    5,
			Source:      iface.Name,
		})
	}

	return ranges
}

func (ne *NetworkExpansion) findRFC1918Parent(ip net.IP) *net.IPNet {
	for _, rfc1918 := range RFC1918Networks {
		if rfc1918.Contains(ip) {
			return rfc1918
		}
	}
	return nil
}

func (ne *NetworkExpansion) isRFC1918(ip net.IP) bool {
	return ne.findRFC1918Parent(ip) != nil
}

func (ne *NetworkExpansion) mergeRanges(ranges []ScanRange) []ScanRange {
	if len(ranges) <= 1 {
		return ranges
	}

	// Sort by priority and network size
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Priority != ranges[j].Priority {
			return ranges[i].Priority < ranges[j].Priority
		}
		// Prefer smaller networks first
		iOnes, _ := ranges[i].Network.Mask.Size()
		jOnes, _ := ranges[j].Network.Mask.Size()
		return iOnes > jOnes
	})

	var merged []ScanRange
	for _, current := range ranges {
		// Check if this range is already covered by a higher priority range
		covered := false
		for _, existing := range merged {
			if existing.Network.Contains(current.Network.IP) &&
			   existing.Network.Contains(ne.getLastIP(current.Network)) {
				covered = true
				break
			}
		}

		if !covered {
			merged = append(merged, current)
		}
	}

	return merged
}

func (ne *NetworkExpansion) getLastIP(network *net.IPNet) net.IP {
	ip := network.IP.To4()
	mask := network.Mask
	lastIP := make(net.IP, len(ip))

	for i := 0; i < len(ip); i++ {
		lastIP[i] = ip[i] | ^mask[i]
	}

	return lastIP
}

func (ne *NetworkExpansion) GetScanModeDescription(mode ScanMode) string {
	switch mode {
	case ScanModeQuick:
		return "Quick scan (interface subnets only)"
	case ScanModeNormal:
		return "Normal scan (intelligent RFC1918 expansion)"
	case ScanModeComprehensive:
		return "Comprehensive scan (full RFC1918 ranges)"
	case ScanModeFirewallTest:
		return "Firewall test scan (security-focused ranges)"
	default:
		return "Unknown scan mode"
	}
}