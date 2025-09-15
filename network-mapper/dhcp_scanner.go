package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type DHCPLease struct {
	IP       net.IP
	MAC      string
	Hostname string
	Expires  time.Time
}

func (ns *NetworkScanner) scanDHCPLeases() []DHCPLease {
	fmt.Println("   💾 Scanning DHCP lease tables...")

	var leases []DHCPLease

	switch runtime.GOOS {
	case "linux":
		leases = append(leases, ns.scanLinuxDHCPLeases()...)
	case "darwin":
		leases = append(leases, ns.scanDarwinDHCPLeases()...)
	case "windows":
		leases = append(leases, ns.scanWindowsDHCPLeases()...)
	}

	leases = append(leases, ns.scanCommonDHCPFiles()...)

	fmt.Printf("   ✅ Found %d DHCP lease entries\n", len(leases))
	return leases
}

func (ns *NetworkScanner) scanLinuxDHCPLeases() []DHCPLease {
	var leases []DHCPLease

	dhcpFiles := []string{
		"/var/lib/dhcp/dhcpd.leases",
		"/var/lib/dhcpcd5/dhcpcd.leases",
		"/var/db/dhcpcd.leases",
		"/etc/dhcp/dhcpd.leases",
	}

	for _, file := range dhcpFiles {
		if _, err := os.Stat(file); err == nil {
			fileLeases := ns.parseDHCPLeasesFile(file)
			leases = append(leases, fileLeases...)
		}
	}

	return leases
}

func (ns *NetworkScanner) scanDarwinDHCPLeases() []DHCPLease {
	var leases []DHCPLease

	dhcpFiles := []string{
		"/var/db/dhcpd_leases",
		"/usr/local/var/db/dhcpd.leases",
	}

	for _, file := range dhcpFiles {
		if _, err := os.Stat(file); err == nil {
			fileLeases := ns.parseDHCPLeasesFile(file)
			leases = append(leases, fileLeases...)
		}
	}

	return leases
}

func (ns *NetworkScanner) scanWindowsDHCPLeases() []DHCPLease {
	var leases []DHCPLease

	cmd := exec.Command("netsh", "dhcp", "server", "show", "scope")
	output, err := cmd.Output()
	if err == nil {
		leases = append(leases, ns.parseWindowsDHCPOutput(string(output))...)
	}

	return leases
}

func (ns *NetworkScanner) scanCommonDHCPFiles() []DHCPLease {
	var leases []DHCPLease

	commonPaths := []string{
		"/tmp",
		"/var/tmp",
		filepath.Join(os.TempDir(), "dhcp"),
	}

	for _, basePath := range commonPaths {
		pattern := filepath.Join(basePath, "*dhcp*")
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				fileLeases := ns.parseDHCPLeasesFile(match)
				leases = append(leases, fileLeases...)
			}
		}
	}

	return leases
}

func (ns *NetworkScanner) parseDHCPLeasesFile(filename string) []DHCPLease {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var leases []DHCPLease
	scanner := bufio.NewScanner(file)

	var currentLease DHCPLease
	inLease := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "lease ") && strings.HasSuffix(line, " {") {
			inLease = true
			ipStr := strings.TrimSuffix(strings.TrimPrefix(line, "lease "), " {")
			currentLease = DHCPLease{IP: net.ParseIP(ipStr)}
		} else if line == "}" && inLease {
			if currentLease.IP != nil {
				leases = append(leases, currentLease)
			}
			inLease = false
			currentLease = DHCPLease{}
		} else if inLease {
			if strings.HasPrefix(line, "hardware ethernet ") {
				mac := strings.TrimSuffix(strings.TrimPrefix(line, "hardware ethernet "), ";")
				currentLease.MAC = strings.ToUpper(mac)
			} else if strings.HasPrefix(line, "client-hostname ") {
				hostname := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "client-hostname "), ";"), "\"")
				currentLease.Hostname = hostname
			} else if strings.HasPrefix(line, "ends ") {
				timeStr := strings.TrimSuffix(strings.TrimPrefix(line, "ends "), ";")
				if t, err := time.Parse("1 2006/01/02 15:04:05", timeStr); err == nil {
					currentLease.Expires = t
				}
			}
		}
	}

	return leases
}

func (ns *NetworkScanner) parseWindowsDHCPOutput(output string) []DHCPLease {
	var leases []DHCPLease

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Subnet") && strings.Contains(line, "Address") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ip := net.ParseIP(parts[1]); ip != nil {
					lease := DHCPLease{
						IP: ip,
					}
					leases = append(leases, lease)
				}
			}
		}
	}

	return leases
}

func (ns *NetworkScanner) enhanceDevicesWithDHCPInfo(devices []Device, leases []DHCPLease) {
	leaseMap := make(map[string]DHCPLease)
	for _, lease := range leases {
		if lease.IP != nil {
			leaseMap[lease.IP.String()] = lease
		}
	}

	for i := range devices {
		ipStr := devices[i].IP.String()
		if lease, exists := leaseMap[ipStr]; exists {
			if devices[i].MAC == "" || devices[i].MAC == "unknown" {
				devices[i].MAC = lease.MAC
			}
			if devices[i].Hostname == "" || devices[i].Hostname == "unknown" {
				devices[i].Hostname = lease.Hostname
			}
		}
	}
}