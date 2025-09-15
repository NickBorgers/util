package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (ns *NetworkScanner) pingHost(host string) bool {
	switch runtime.GOOS {
	case "windows":
		return ns.pingWindows(host)
	default:
		return ns.pingUnix(host)
	}
}

func (ns *NetworkScanner) pingWindows(host string) bool {
	cmd := exec.Command("ping", "-n", "1", "-w", "1000", host)
	err := cmd.Run()
	return err == nil
}

func (ns *NetworkScanner) pingUnix(host string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", host)
	err := cmd.Run()
	return err == nil
}

func (ns *NetworkScanner) getMACAddress(ip net.IP) string {
	switch runtime.GOOS {
	case "windows":
		return ns.getMACWindows(ip)
	default:
		return ns.getMACUnix(ip)
	}
}

func (ns *NetworkScanner) getMACWindows(ip net.IP) string {
	cmd := exec.Command("arp", "-a", ip.String())
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip.String()) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				mac := parts[1]
				if mac != "00-00-00-00-00-00" && len(mac) == 17 {
					return strings.ReplaceAll(mac, "-", ":")
				}
			}
		}
	}
	return "unknown"
}

func (ns *NetworkScanner) getMACUnix(ip net.IP) string {
	cmd := exec.Command("arp", "-n", ip.String())
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip.String()) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				mac := parts[2]
				if mac != "(incomplete)" && len(mac) == 17 {
					return mac
				}
			}
		}
	}
	return "unknown"
}

func (ns *NetworkScanner) scanCommonPorts(ip net.IP) []int {
	commonPorts := []int{22, 23, 25, 53, 80, 110, 111, 135, 139, 143, 443, 993, 995, 1723, 3389, 5900, 8080}
	var openPorts []int

	for _, port := range commonPorts {
		address := fmt.Sprintf("%s:%d", ip.String(), port)
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			openPorts = append(openPorts, port)
		}
	}

	return openPorts
}

func (ns *NetworkScanner) identifyDeviceType(ip net.IP, ports []int) string {
	if len(ports) == 0 {
		return "Unknown"
	}

	hasSSH := contains(ports, 22)
	hasHTTP := contains(ports, 80) || contains(ports, 443) || contains(ports, 8080)
	hasRDP := contains(ports, 3389)
	hasSMB := contains(ports, 135) || contains(ports, 139)
	hasVNC := contains(ports, 5900)
	hasPrinter := contains(ports, 631) || contains(ports, 9100)
	hasDNS := contains(ports, 53)
	hasDHCP := contains(ports, 67)
	hasSNMP := contains(ports, 161)

	if hasPrinter {
		return "Printer"
	}
	if hasDNS && hasDHCP {
		return "Router/Gateway"
	}
	if hasDNS || hasDHCP || hasSNMP {
		return "Network Device"
	}
	if hasRDP && hasSMB {
		return "Windows PC"
	}
	if hasSSH && hasHTTP {
		return "Linux Server"
	}
	if hasSSH {
		return "Linux/Unix"
	}
	if hasHTTP {
		return "Web Server"
	}
	if hasVNC {
		return "Remote Desktop"
	}
	if hasSMB {
		return "File Server"
	}

	return "Network Device"
}

func (ns *NetworkScanner) enhanceDeviceTypeWithServices(device *Device) {
	if len(device.Services) == 0 {
		return
	}

	hasAirPlay := false
	hasGooglecast := false
	hasHomeKit := false
	hasPrinting := false
	hasMediaServer := false
	hasAppleServices := false

	for _, service := range device.Services {
		serviceType := strings.ToLower(service.Type)
		serviceName := strings.ToLower(service.Name)

		if strings.Contains(serviceType, "airplay") || strings.Contains(serviceName, "airplay") {
			hasAirPlay = true
		}
		if strings.Contains(serviceType, "googlecast") || strings.Contains(serviceName, "googlecast") {
			hasGooglecast = true
		}
		if strings.Contains(serviceType, "homekit") || strings.Contains(serviceType, "hap") {
			hasHomeKit = true
		}
		if strings.Contains(serviceType, "ipp") || strings.Contains(serviceType, "printer") {
			hasPrinting = true
		}
		if strings.Contains(serviceType, "mediaserver") || strings.Contains(serviceType, "mediarenderer") {
			hasMediaServer = true
		}
		if strings.Contains(serviceName, "apple") || strings.Contains(serviceType, "daap") ||
			strings.Contains(serviceType, "timemachine") || strings.Contains(serviceType, "adisk") {
			hasAppleServices = true
		}
	}

	originalType := device.DeviceType

	if hasAirPlay && hasAppleServices {
		device.DeviceType = "Apple TV/AirPlay"
	} else if hasGooglecast {
		device.DeviceType = "Chromecast/Google Device"
	} else if hasHomeKit {
		device.DeviceType = "HomeKit Device"
	} else if hasPrinting {
		device.DeviceType = "Network Printer"
	} else if hasMediaServer {
		device.DeviceType = "Media Server"
	} else if hasAppleServices && device.MACVendor != "" && strings.Contains(device.MACVendor, "Apple") {
		device.DeviceType = "Apple Device"
	}

	if device.DeviceType == originalType && device.MACVendor != "" {
		vendor := strings.ToLower(device.MACVendor)
		if strings.Contains(vendor, "apple") && device.DeviceType == "Network Device" {
			device.DeviceType = "Apple Device"
		} else if strings.Contains(vendor, "raspberry") {
			device.DeviceType = "Raspberry Pi"
		} else if strings.Contains(vendor, "intel") && contains(device.Ports, 22) {
			device.DeviceType = "Intel NUC/Server"
		}
	}
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
