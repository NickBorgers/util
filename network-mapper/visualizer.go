package main

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

func (ns *NetworkScanner) visualizeNetwork() {
	fmt.Println("\n📊 Network Topology Map")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for i, iface := range ns.interfaces {
		ns.drawNetworkSegment(iface, i+1)
	}

	ns.drawSummary()
}

func (ns *NetworkScanner) drawNetworkSegment(iface NetworkInterface, segmentNum int) {
	fmt.Printf("\n🌐 Network Segment %d: %s\n", segmentNum, iface.Name)
	fmt.Printf("   IP: %s | Subnet: %s\n", iface.IP.String(), iface.Subnet.String())

	// Check for broadcast services
	broadcastServicesByHost := ns.getBroadcastServicesByHost(iface)
	if len(broadcastServicesByHost) > 0 {
		fmt.Println("   📡 Broadcast Services:")
		for hostIP, services := range broadcastServicesByHost {
			for _, service := range services {
				protocol := service.Protocol
				if protocol == "" {
					protocol = "tcp"
				}
				fmt.Printf("   │   • %s:%d (%s) via %s from %s\n", service.Name, service.Port, protocol, service.Source, hostIP)
			}
		}
		fmt.Println("   │")
	}

	devices := ns.getDevicesForInterface(iface)
	routableSubnets := ns.getRoutableSubnetsForInterface(iface)

	// Count total items: direct devices + routable subnet headers + routable subnet devices
	totalItems := len(devices) + len(routableSubnets)
	for _, subnetDevices := range routableSubnets {
		totalItems += len(subnetDevices)
	}

	if totalItems == 0 {
		if len(broadcastServicesByHost) == 0 {
			fmt.Println("   └── (No devices discovered)")
		} else {
			fmt.Println("   └── (No individual devices discovered)")
		}
		return
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].IP.String() < devices[j].IP.String()
	})

	fmt.Println("   │")
	if iface.Gateway != nil {
		fmt.Printf("   ├─🏠 Gateway: %s\n", iface.Gateway.String())
		fmt.Println("   │  │")
	}

	currentItem := 0

	// Show direct devices first
	for _, device := range devices {
		currentItem++
		isLast := currentItem == totalItems
		connector := "├─"
		continuation := "│  "
		if isLast {
			connector = "└─"
			continuation = "   "
		}

		icon := ns.getDeviceIcon(device)

		// Use hostname as primary identifier if available, with IP as secondary
		primaryName := device.IP.String()
		secondaryInfo := ""

		if device.Hostname != "" && device.Hostname != "unknown" {
			primaryName = device.Hostname
			secondaryInfo = fmt.Sprintf("(%s)", device.IP.String())
		}

		fmt.Printf("   %s%s %s", connector, icon, primaryName)

		if secondaryInfo != "" {
			fmt.Printf(" %s", secondaryInfo)
		}

		details := ns.getDeviceDetails(device)
		if details != "" {
			fmt.Printf(" %s", details)
		}
		fmt.Println()

		if len(device.Ports) > 0 {
			portStr := ns.formatPorts(device.Ports)
			fmt.Printf("   %s   🔌 Ports: %s\n", continuation, portStr)
		}

		if len(device.Services) > 0 {
			serviceStr := ns.formatServices(device.Services)
			fmt.Printf("   %s   📡 Services: %s\n", continuation, serviceStr)
		}

		if device.MAC != "unknown" && device.MAC != "" {
			macStr := device.MAC
			if device.MACVendor != "" {
				macStr = fmt.Sprintf("%s (%s)", device.MAC, device.MACVendor)
			}
			fmt.Printf("   %s   🏷️  MAC: %s\n", continuation, macStr)
		}

		if len(device.UPnPInfo) > 0 {
			if server, ok := device.UPnPInfo["server"]; ok && server != "" {
				fmt.Printf("   %s   🌐 UPnP: %s\n", continuation, server)
			}
		}
	}

	// Show routable subnets
	if len(routableSubnets) > 0 {
		// Sort subnet keys for consistent output
		var subnetKeys []string
		for subnet := range routableSubnets {
			subnetKeys = append(subnetKeys, subnet)
		}
		sort.Strings(subnetKeys)

		for _, subnet := range subnetKeys {
			subnetDevices := routableSubnets[subnet]
			if len(subnetDevices) == 0 {
				continue
			}

			// Sort devices in subnet
			sort.Slice(subnetDevices, func(i, j int) bool {
				return subnetDevices[i].IP.String() < subnetDevices[j].IP.String()
			})

			// Show routable subnet header
			currentItem++
			isLastSubnet := (currentItem + len(subnetDevices)) == totalItems
			subnetConnector := "├─"
			if isLastSubnet {
				subnetConnector = "└─"
			}

			fmt.Printf("   %s🔍 DISCOVERED Subnet: %s (%d device%s) [via routing table]\n",
				subnetConnector, subnet, len(subnetDevices),
				map[bool]string{true: "", false: "s"}[len(subnetDevices) == 1])

			// Show devices in this routable subnet
			for i, device := range subnetDevices {
				currentItem++
				isLast := currentItem == totalItems
				deviceConnector := "│  ├─"
				deviceContinuation := "│  │  "

				if i == len(subnetDevices)-1 {
					deviceConnector = "│  └─"
					deviceContinuation = "│     "
				}
				if isLast {
					if i == len(subnetDevices)-1 {
						deviceConnector = "   └─"
						deviceContinuation = "      "
					} else {
						deviceConnector = "   ├─"
						deviceContinuation = "   │  "
					}
				}

				icon := ns.getDeviceIcon(device)

				// Use hostname as primary identifier if available, with IP as secondary
				primaryName := device.IP.String()
				secondaryInfo := ""

				if device.Hostname != "" && device.Hostname != "unknown" {
					primaryName = device.Hostname
					secondaryInfo = fmt.Sprintf("(%s)", device.IP.String())
				}

				fmt.Printf("   %s%s %s", deviceConnector, icon, primaryName)

				if secondaryInfo != "" {
					fmt.Printf(" %s", secondaryInfo)
				}

				details := ns.getDeviceDetails(device)
				if details != "" {
					fmt.Printf(" %s", details)
				}
				fmt.Println()

				if len(device.Ports) > 0 {
					portStr := ns.formatPorts(device.Ports)
					fmt.Printf("   %s   🔌 Ports: %s\n", deviceContinuation, portStr)
				}

				if len(device.Services) > 0 {
					serviceStr := ns.formatServices(device.Services)
					fmt.Printf("   %s   📡 Services: %s\n", deviceContinuation, serviceStr)
				}

				if device.MAC != "unknown" && device.MAC != "" {
					macStr := device.MAC
					if device.MACVendor != "" {
						macStr = fmt.Sprintf("%s (%s)", device.MAC, device.MACVendor)
					}
					fmt.Printf("   %s   🏷️  MAC: %s\n", deviceContinuation, macStr)
				}

				if len(device.UPnPInfo) > 0 {
					if server, ok := device.UPnPInfo["server"]; ok && server != "" {
						fmt.Printf("   %s   🌐 UPnP: %s\n", deviceContinuation, server)
					}
				}
			}
		}
	}
}

func (ns *NetworkScanner) getDevicesForInterface(iface NetworkInterface) []Device {
	var devices []Device
	for _, device := range ns.devices {
		if iface.Subnet.Contains(device.IP) && !ns.isBroadcastIP(device.IP, iface.Subnet) {
			devices = append(devices, device)
		}
	}
	return devices
}

// getRoutableSubnetsForInterface returns devices that are accessible through this interface
// but not directly in its subnet, grouped by their network prefix
func (ns *NetworkScanner) getRoutableSubnetsForInterface(iface NetworkInterface) map[string][]Device {
	routableSubnets := make(map[string][]Device)

	// Find devices that aren't in any direct interface subnet
	for _, device := range ns.devices {
		if device.IP == nil {
			continue
		}

		// Skip if this device is in any direct interface subnet
		inDirectSubnet := false
		for _, checkIface := range ns.interfaces {
			if checkIface.Subnet != nil && checkIface.Subnet.Contains(device.IP) {
				inDirectSubnet = true
				break
			}
		}

		if inDirectSubnet {
			continue
		}

		// For devices not in direct subnets, determine if they're routable through this interface
		// For home networks, typically the interface with a gateway can reach other subnets
		if iface.Gateway != nil && ns.isRoutableThrough(device.IP, iface) {
			// Group by /24 network to identify subnet patterns
			networkPrefix := ns.getNetworkPrefix(device.IP, 24)
			if networkPrefix != "" {
				routableSubnets[networkPrefix] = append(routableSubnets[networkPrefix], device)
			}
		}
	}

	return routableSubnets
}

// isRoutableThrough determines if a device IP is likely routable through the given interface
func (ns *NetworkScanner) isRoutableThrough(deviceIP net.IP, iface NetworkInterface) bool {
	if deviceIP == nil {
		return false
	}

	// Create a /32 subnet for the device IP to check routing
	var deviceSubnet *net.IPNet
	if deviceIP.To4() != nil {
		deviceSubnet = &net.IPNet{IP: deviceIP, Mask: net.CIDRMask(32, 32)}
	} else {
		deviceSubnet = &net.IPNet{IP: deviceIP, Mask: net.CIDRMask(128, 128)}
	}

	// Use the new route table consultation
	bestInterface := ns.findBestRouteForSubnet(deviceSubnet)
	return bestInterface != nil && bestInterface.Name == iface.Name
}

// getNetworkPrefix returns the network prefix (e.g., "192.168.1.0/24") for an IP
func (ns *NetworkScanner) getNetworkPrefix(ip net.IP, prefixLength int) string {
	if ip == nil {
		return ""
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}

	// Create a mask for the prefix length
	mask := net.CIDRMask(prefixLength, 32)
	network := ip4.Mask(mask)

	return fmt.Sprintf("%s/%d", network.String(), prefixLength)
}

// isBroadcastIP checks if an IP address is the broadcast address for the given subnet
func (ns *NetworkScanner) isBroadcastIP(ip net.IP, subnet *net.IPNet) bool {
	if subnet == nil {
		return false
	}

	subnetIP := subnet.IP.To4()
	mask := subnet.Mask
	testIP := ip.To4()

	if subnetIP == nil || mask == nil || testIP == nil {
		return false
	}

	// Calculate broadcast address
	broadcast := make(net.IP, len(subnetIP))
	for i := range subnetIP {
		broadcast[i] = subnetIP[i] | ^mask[i]
	}

	return testIP.Equal(broadcast)
}

// getBroadcastServicesByHost returns services discovered grouped by host IP
func (ns *NetworkScanner) getBroadcastServicesByHost(iface NetworkInterface) map[string][]Service {
	servicesByHost := make(map[string][]Service)

	for _, device := range ns.devices {
		// Include all devices in the subnet that have services from broadcast discovery
		if iface.Subnet.Contains(device.IP) {
			var broadcastServices []Service
			for _, service := range device.Services {
				// Include services discovered via broadcast methods
				if service.Source == "probe" || service.Source == "ssdp" || service.Source == "igmp" {
					broadcastServices = append(broadcastServices, service)
				}
			}

			if len(broadcastServices) > 0 {
				hostIP := device.IP.String()
				if device.Hostname != "" && device.Hostname != "unknown" {
					hostIP = fmt.Sprintf("%s (%s)", device.Hostname, device.IP.String())
				}
				servicesByHost[hostIP] = broadcastServices
			}
		}
	}

	return servicesByHost
}

func (ns *NetworkScanner) getDeviceIcon(device Device) string {
	if device.IsGateway {
		return "🌐"
	}

	// Use device detector for icons if available
	if ns.deviceDetector != nil {
		icon := ns.deviceDetector.GetDeviceIcon(device.DeviceType)
		if icon != "❓" {
			return icon
		}
	}

	// Fallback to legacy icon mapping
	switch device.DeviceType {
	case "Windows PC":
		return "🖥️ "
	case "Linux Server":
		return "🐧"
	case "Linux/Unix":
		return "🖥️ "
	case "Web Server":
		return "🌍"
	case "File Server":
		return "📁"
	case "Remote Desktop":
		return "🖥️ "
	case "Network Device":
		return "🔌"
	case "Router/Gateway":
		return "🏠"
	case "Printer", "Network Printer":
		return "🖨️ "
	case "Apple TV/AirPlay":
		return "📺"
	case "Chromecast/Google Device":
		return "📺"
	case "HomeKit Device":
		return "🏡"
	case "Media Server":
		return "🎵"
	case "Apple Device":
		return "🍎"
	case "Raspberry Pi":
		return "🥧"
	case "Intel NUC/Server":
		return "💻"
	default:
		return "❓"
	}
}

func (ns *NetworkScanner) getDeviceDetails(device Device) string {
	var parts []string

	if device.DeviceType != "Unknown" && device.DeviceType != "" {
		parts = append(parts, fmt.Sprintf("(%s)", device.DeviceType))
	}

	// Only show hostname in brackets if it's not already being used as primary name
	// and if it's different from what we'd expect (not just the device type)
	if device.Hostname != "unknown" && device.Hostname != "" {
		hostname := strings.TrimSuffix(device.Hostname, ".")

		// Don't show hostname in details if it's already the primary name
		// or if it's a generic hostname that doesn't add value
		if !ns.isHostnamePrimary(device) && !ns.isGenericHostname(hostname) {
			if len(hostname) > 20 {
				hostname = hostname[:17] + "..."
			}
			parts = append(parts, fmt.Sprintf("[%s]", hostname))
		}
	}

	return strings.Join(parts, " ")
}

func (ns *NetworkScanner) isHostnamePrimary(device Device) bool {
	return device.Hostname != "" && device.Hostname != "unknown"
}

func (ns *NetworkScanner) isGenericHostname(hostname string) bool {
	genericPatterns := []string{
		"localhost",
		"unknown",
		"router",
		"gateway",
		"modem",
	}

	hostname = strings.ToLower(hostname)
	for _, pattern := range genericPatterns {
		if strings.Contains(hostname, pattern) {
			return true
		}
	}

	return false
}

func (ns *NetworkScanner) formatPorts(ports []int) string {
	if len(ports) == 0 {
		return "none"
	}

	portStrs := make([]string, 0, len(ports))
	for _, port := range ports {
		service := ns.getServiceName(port)
		if service != "" {
			portStrs = append(portStrs, fmt.Sprintf("%d(%s)", port, service))
		} else {
			portStrs = append(portStrs, fmt.Sprintf("%d", port))
		}
	}

	if len(portStrs) > 5 {
		return fmt.Sprintf("%s, ... (+%d more)", strings.Join(portStrs[:5], ", "), len(portStrs)-5)
	}

	return strings.Join(portStrs, ", ")
}

func (ns *NetworkScanner) formatServices(services []Service) string {
	if len(services) == 0 {
		return "none"
	}

	serviceStrs := make([]string, 0, len(services))
	seen := make(map[string]bool)

	for _, service := range services {
		var serviceStr string

		if service.Port > 0 {
			if service.Name != "" {
				serviceStr = fmt.Sprintf("%s:%d", service.Name, service.Port)
			} else {
				serviceStr = fmt.Sprintf("%s:%d", service.Type, service.Port)
			}
		} else {
			if service.Name != "" {
				serviceStr = service.Name
			} else {
				serviceStr = service.Type
			}
		}

		if service.Source != "" {
			serviceStr = fmt.Sprintf("%s(%s)", serviceStr, service.Source)
		}

		if !seen[serviceStr] {
			serviceStrs = append(serviceStrs, serviceStr)
			seen[serviceStr] = true
		}
	}

	if len(serviceStrs) > 3 {
		return fmt.Sprintf("%s, ... (+%d more)", strings.Join(serviceStrs[:3], ", "), len(serviceStrs)-3)
	}

	return strings.Join(serviceStrs, ", ")
}

func (ns *NetworkScanner) getServiceName(port int) string {
	services := map[int]string{
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		111:  "RPC",
		135:  "RPC",
		139:  "SMB",
		143:  "IMAP",
		443:  "HTTPS",
		993:  "IMAPS",
		995:  "POP3S",
		1723: "PPTP",
		3389: "RDP",
		5900: "VNC",
		8080: "HTTP-Alt",
	}

	if service, ok := services[port]; ok {
		return service
	}
	return ""
}

func (ns *NetworkScanner) drawSummary() {
	fmt.Println("\n📈 Discovery Summary")
	fmt.Println("────────────────────────────────────────────────────────────────")

	fmt.Printf("🌐 Network Interfaces: %d\n", len(ns.interfaces))
	fmt.Printf("📱 Total Devices Found: %d\n", len(ns.devices))

	deviceTypes := make(map[string]int)
	totalPorts := 0

	for _, device := range ns.devices {
		if device.DeviceType != "" && device.DeviceType != "Unknown" {
			deviceTypes[device.DeviceType]++
		}
		totalPorts += len(device.Ports)
	}

	if len(deviceTypes) > 0 {
		fmt.Println("\n🔍 Device Types Detected:")
		for deviceType, count := range deviceTypes {
			fmt.Printf("   • %s: %d\n", deviceType, count)
		}
	}

	fmt.Printf("\n🔌 Total Open Ports: %d\n", totalPorts)

	activeInterfaces := 0
	for _, iface := range ns.interfaces {
		if len(ns.getDevicesForInterface(iface)) > 0 {
			activeInterfaces++
		}
	}

	fmt.Printf("📡 Active Network Segments: %d/%d\n", activeInterfaces, len(ns.interfaces))

	fmt.Println("\n✨ Scan Complete!")
}
