package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func (ns *NetworkScanner) findGatewayLinux(subnet *net.IPNet) net.IP {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return ns.guessGateway(subnet)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "default via") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "via" && i+1 < len(parts) {
					if ip := net.ParseIP(parts[i+1]); ip != nil && subnet.Contains(ip) {
						return ip
					}
				}
			}
		}
	}

	return ns.guessGateway(subnet)
}

func (ns *NetworkScanner) findGatewayDarwin(subnet *net.IPNet) net.IP {
	cmd := exec.Command("route", "-n", "get", "default")
	output, err := cmd.Output()
	if err != nil {
		return ns.guessGateway(subnet)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "gateway:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ip := net.ParseIP(parts[1]); ip != nil && subnet.Contains(ip) {
					return ip
				}
			}
		}
	}

	return ns.guessGateway(subnet)
}

func (ns *NetworkScanner) findGatewayWindows(subnet *net.IPNet) net.IP {
	cmd := exec.Command("route", "print", "0.0.0.0")
	output, err := cmd.Output()
	if err != nil {
		return ns.guessGateway(subnet)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "0.0.0.0") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if ip := net.ParseIP(parts[2]); ip != nil && subnet.Contains(ip) {
					return ip
				}
			}
		}
	}

	return ns.guessGateway(subnet)
}

func (ns *NetworkScanner) guessGateway(subnet *net.IPNet) net.IP {
	ip := subnet.IP.To4()
	if ip != nil {
		gateway := make(net.IP, len(ip))
		copy(gateway, ip)
		gateway[3] = 1
		return gateway
	}
	return nil
}

// RouteEntry represents a single route table entry
type RouteEntry struct {
	Destination *net.IPNet
	Gateway     net.IP
	Interface   string
	Metric      int
}

// getRouteTable retrieves the system's routing table
func (ns *NetworkScanner) getRouteTable() ([]RouteEntry, error) {
	switch runtime.GOOS {
	case "linux":
		return ns.getRouteTableLinux()
	case "darwin":
		return ns.getRouteTableDarwin()
	case "windows":
		return ns.getRouteTableWindows()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getRouteTableLinux gets routing table on Linux using ip route
func (ns *NetworkScanner) getRouteTableLinux() ([]RouteEntry, error) {
	cmd := exec.Command("ip", "route", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var routes []RouteEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		route, err := ns.parseLinuxRoute(line)
		if err == nil {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseLinuxRoute parses a single Linux route line
func (ns *NetworkScanner) parseLinuxRoute(line string) (RouteEntry, error) {
	parts := strings.Fields(line)
	if len(parts) < 1 {
		return RouteEntry{}, fmt.Errorf("invalid route line: %s", line)
	}

	route := RouteEntry{}

	// Parse destination
	if parts[0] == "default" {
		_, route.Destination, _ = net.ParseCIDR("0.0.0.0/0")
	} else {
		// Check if it has CIDR notation
		if strings.Contains(parts[0], "/") {
			_, dest, err := net.ParseCIDR(parts[0])
			if err != nil {
				return RouteEntry{}, err
			}
			route.Destination = dest
		} else {
			// Single IP, assume /32
			ip := net.ParseIP(parts[0])
			if ip == nil {
				return RouteEntry{}, fmt.Errorf("invalid IP: %s", parts[0])
			}
			if ip.To4() != nil {
				route.Destination = &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
			} else {
				route.Destination = &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
			}
		}
	}

	// Parse the rest of the route
	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "via":
			if i+1 < len(parts) {
				route.Gateway = net.ParseIP(parts[i+1])
				i++ // Skip the next part as we consumed it
			}
		case "dev":
			if i+1 < len(parts) {
				route.Interface = parts[i+1]
				i++ // Skip the next part as we consumed it
			}
		case "metric":
			if i+1 < len(parts) {
				if metric, err := strconv.Atoi(parts[i+1]); err == nil {
					route.Metric = metric
				}
				i++ // Skip the next part as we consumed it
			}
		}
	}

	return route, nil
}

// getRouteTableDarwin gets routing table on macOS using netstat
func (ns *NetworkScanner) getRouteTableDarwin() ([]RouteEntry, error) {
	cmd := exec.Command("netstat", "-rn", "-f", "inet")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var routes []RouteEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header lines
	headerSkipped := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !headerSkipped {
			if strings.Contains(line, "Destination") {
				headerSkipped = true
			}
			continue
		}

		if line == "" {
			continue
		}

		route, err := ns.parseDarwinRoute(line)
		if err == nil {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseDarwinRoute parses a single macOS route line from netstat
func (ns *NetworkScanner) parseDarwinRoute(line string) (RouteEntry, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return RouteEntry{}, fmt.Errorf("invalid route line: %s", line)
	}

	route := RouteEntry{}

	// Parse destination
	if parts[0] == "default" {
		_, route.Destination, _ = net.ParseCIDR("0.0.0.0/0")
	} else {
		// Handle various destination formats
		if strings.Contains(parts[0], "/") {
			_, dest, err := net.ParseCIDR(parts[0])
			if err != nil {
				return RouteEntry{}, err
			}
			route.Destination = dest
		} else {
			ip := net.ParseIP(parts[0])
			if ip == nil {
				return RouteEntry{}, fmt.Errorf("invalid IP: %s", parts[0])
			}
			if ip.To4() != nil {
				route.Destination = &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
			} else {
				route.Destination = &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
			}
		}
	}

	// Gateway is typically the second column
	if parts[1] != "*" {
		route.Gateway = net.ParseIP(parts[1])
	}

	// Interface is typically in one of the later columns
	for i := 2; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "en") || strings.HasPrefix(parts[i], "eth") ||
			strings.HasPrefix(parts[i], "wlan") || strings.HasPrefix(parts[i], "lo") {
			route.Interface = parts[i]
			break
		}
	}

	return route, nil
}

// getRouteTableWindows gets routing table on Windows using route print
func (ns *NetworkScanner) getRouteTableWindows() ([]RouteEntry, error) {
	cmd := exec.Command("route", "print")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var routes []RouteEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	inIPv4Section := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "IPv4 Route Table") {
			inIPv4Section = true
			continue
		}

		if strings.Contains(line, "IPv6 Route Table") {
			inIPv4Section = false
			continue
		}

		if !inIPv4Section || line == "" {
			continue
		}

		// Skip header lines
		if strings.Contains(line, "Network Destination") ||
			strings.Contains(line, "=======") {
			continue
		}

		route, err := ns.parseWindowsRoute(line)
		if err == nil {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseWindowsRoute parses a single Windows route line
func (ns *NetworkScanner) parseWindowsRoute(line string) (RouteEntry, error) {
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return RouteEntry{}, fmt.Errorf("invalid route line: %s", line)
	}

	route := RouteEntry{}

	// Network Destination
	if parts[0] == "0.0.0.0" {
		_, route.Destination, _ = net.ParseCIDR("0.0.0.0/0")
	} else {
		// Parse destination and netmask
		destIP := net.ParseIP(parts[0])
		netmask := net.ParseIP(parts[1])
		if destIP == nil || netmask == nil {
			return RouteEntry{}, fmt.Errorf("invalid destination or netmask: %s %s", parts[0], parts[1])
		}

		// Convert netmask to CIDR
		mask := net.IPMask(netmask.To4())
		route.Destination = &net.IPNet{IP: destIP, Mask: mask}
	}

	// Gateway
	if parts[2] != "0.0.0.0" {
		route.Gateway = net.ParseIP(parts[2])
	}

	// Interface (column 3, but we'll try to resolve this later)
	// Metric
	if len(parts) > 4 {
		if metric, err := strconv.Atoi(parts[4]); err == nil {
			route.Metric = metric
		}
	}

	return route, nil
}

// findBestRouteForSubnet finds the best interface to route to a given subnet
func (ns *NetworkScanner) findBestRouteForSubnet(subnet *net.IPNet) *NetworkInterface {
	routes, err := ns.getRouteTable()
	if err != nil {
		// Fallback to original logic
		return ns.findInterfaceWithGateway()
	}

	var bestRoute *RouteEntry

	// Find the most specific route that can handle this subnet
	for _, route := range routes {
		if route.Destination == nil {
			continue
		}

		// Check if this route can handle the subnet
		if route.Destination.Contains(subnet.IP) {
			// Prefer more specific routes (larger prefix length)
			if bestRoute == nil {
				bestRoute = &route
			} else {
				bestPrefixLen, _ := bestRoute.Destination.Mask.Size()
				currentPrefixLen, _ := route.Destination.Mask.Size()
				if currentPrefixLen > bestPrefixLen {
					bestRoute = &route
				}
			}
		}
	}

	if bestRoute != nil {
		// Find the interface that corresponds to this route
		for i := range ns.interfaces {
			iface := &ns.interfaces[i]
			// Match by interface name if available
			if bestRoute.Interface != "" && iface.Name == bestRoute.Interface {
				return iface
			}
			// Match by gateway
			if bestRoute.Gateway != nil && iface.Gateway != nil &&
				bestRoute.Gateway.Equal(iface.Gateway) {
				return iface
			}
		}
	}

	// Fallback to interface with gateway
	return ns.findInterfaceWithGateway()
}

// findInterfaceWithGateway returns the first interface that has a gateway
func (ns *NetworkScanner) findInterfaceWithGateway() *NetworkInterface {
	for i := range ns.interfaces {
		if ns.interfaces[i].Gateway != nil {
			return &ns.interfaces[i]
		}
	}
	return nil
}
