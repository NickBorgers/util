package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type ServiceDiscovery struct {
	devices map[string]*Device
	mu      sync.RWMutex
	verbose bool
}

func NewServiceDiscovery() *ServiceDiscovery {
	return NewServiceDiscoveryWithVerbose(false)
}

func NewServiceDiscoveryWithVerbose(verbose bool) *ServiceDiscovery {
	return &ServiceDiscovery{
		devices: make(map[string]*Device),
		verbose: verbose,
	}
}

func (sd *ServiceDiscovery) DiscoverServices(interfaces []NetworkInterface, timeout time.Duration) {
	var wg sync.WaitGroup

	fmt.Println("🔍 Discovering network services...")

	wg.Add(4)
	go func() {
		defer wg.Done()
		sd.discoverMDNS(timeout)
	}()

	go func() {
		defer wg.Done()
		sd.discoverSSDP(interfaces, timeout)
	}()

	go func() {
		defer wg.Done()
		sd.discoverMulticastGroups(interfaces)
	}()

	go func() {
		defer wg.Done()
		sd.discoverCommonServices(interfaces)
	}()

	wg.Wait()
}

func (sd *ServiceDiscovery) discoverMDNS(timeout time.Duration) {
	// Suppress mDNS library logging unless in verbose mode
	originalLogger := log.Writer()
	if !sd.verbose {
		log.SetOutput(io.Discard) // Suppress mDNS library logs
	}
	defer func() {
		log.SetOutput(originalLogger) // Restore original logger
	}()

	// Reduced output - only show in verbose mode
	if sd.verbose {
		fmt.Println("   📡 Scanning mDNS/Bonjour services...")
	}

	serviceTypes := []string{
		"_http._tcp",
		"_https._tcp",
		"_ssh._tcp",
		"_ftp._tcp",
		"_smb._tcp",
		"_afpovertcp._tcp",
		"_printer._tcp",
		"_ipp._tcp",
		"_airplay._tcp",
		"_googlecast._tcp",
		"_spotify-connect._tcp",
		"_homekit._tcp",
		"_hap._tcp",
		"_companion-link._tcp",
		"_raop._tcp",
		"_rdp._tcp",
		"_vnc._tcp",
		"_rfb._tcp",
		"_workstation._tcp",
		"_device-info._tcp",
		"_sleep-proxy._udp",
		"_airport._tcp",
		"_apple-mobdev2._tcp",
		"_daap._tcp",
		"_dpap._tcp",
		"_eppc._tcp",
		"_touch-able._tcp",
		"_adisk._tcp",
		"_timemachine._tcp",
	}

	for _, serviceType := range serviceTypes {
		sd.queryMDNSService(serviceType, timeout)
	}
}

func (sd *ServiceDiscovery) queryMDNSService(serviceType string, timeout time.Duration) {
	entriesCh := make(chan *mdns.ServiceEntry, 32)
	defer close(entriesCh)

	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		for entry := range entriesCh {
			sd.processMDNSEntry(entry, serviceType)
		}
	}()

	params := &mdns.QueryParam{
		Service: serviceType,
		Domain:  "local",
		Timeout: timeout,
		Entries: entriesCh,
	}

	mdns.Query(params)
}

func (sd *ServiceDiscovery) processMDNSEntry(entry *mdns.ServiceEntry, serviceType string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	ip := entry.AddrV4.String()
	if ip == "<nil>" && entry.AddrV6 != nil {
		ip = entry.AddrV6.String()
	}

	if ip == "<nil>" {
		return
	}

	device, exists := sd.devices[ip]
	if !exists {
		device = &Device{
			IP:       entry.AddrV4,
			Hostname: entry.Host,
			Services: make([]Service, 0),
			UPnPInfo: make(map[string]string),
		}
		sd.devices[ip] = device
	}

	txtMap := make(map[string]string)
	for _, txt := range entry.InfoFields {
		parts := strings.SplitN(txt, "=", 2)
		if len(parts) == 2 {
			txtMap[parts[0]] = parts[1]
		} else {
			txtMap[parts[0]] = ""
		}
	}

	service := Service{
		Name:     entry.Name,
		Type:     serviceType,
		Domain:   "local",
		Port:     entry.Port,
		Protocol: "tcp",
		TXT:      txtMap,
		Source:   "mdns",
	}

	device.Services = append(device.Services, service)

	if entry.Host != "" && device.Hostname == "" {
		device.Hostname = strings.TrimSuffix(entry.Host, ".local.")
	}
}

func (sd *ServiceDiscovery) discoverSSDP(interfaces []NetworkInterface, timeout time.Duration) {
	if sd.verbose {
		fmt.Println("   🔌 Scanning SSDP/UPnP devices...")
	}

	ssdpTargets := []string{
		"upnp:rootdevice",
		"urn:schemas-upnp-org:device:MediaServer:1",
		"urn:schemas-upnp-org:device:MediaRenderer:1",
		"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
		"urn:schemas-upnp-org:device:WANDevice:1",
		"urn:schemas-upnp-org:device:PrinterBasic:1",
		"urn:dial-multiscreen-org:service:dial:1",
		"urn:schemas-upnp-org:service:ContentDirectory:1",
	}

	for _, iface := range interfaces {
		for _, target := range ssdpTargets {
			sd.sendSSDP(iface, target, timeout)
		}
	}
}

func (sd *ServiceDiscovery) sendSSDP(iface NetworkInterface, target string, timeout time.Duration) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: iface.IP, Port: 0})
	if err != nil {
		return
	}
	defer conn.Close()

	multicastAddr, _ := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")

	ssdpRequest := fmt.Sprintf(
		"M-SEARCH * HTTP/1.1\r\n"+
			"HOST: 239.255.255.250:1900\r\n"+
			"MAN: \"ssdp:discover\"\r\n"+
			"ST: %s\r\n"+
			"MX: 3\r\n\r\n", target)

	conn.WriteToUDP([]byte(ssdpRequest), multicastAddr)

	conn.SetReadDeadline(time.Now().Add(timeout))
	buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break
		}

		sd.processSSDP(string(buffer[:n]), addr.IP)
	}
}

func (sd *ServiceDiscovery) processSSDP(response string, ip net.IP) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	ipStr := ip.String()
	device, exists := sd.devices[ipStr]
	if !exists {
		device = &Device{
			IP:       ip,
			Services: make([]Service, 0),
			UPnPInfo: make(map[string]string),
		}
		sd.devices[ipStr] = device
	}

	lines := strings.Split(response, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SERVER:") {
			device.UPnPInfo["server"] = strings.TrimSpace(line[7:])
		} else if strings.HasPrefix(line, "ST:") {
			device.UPnPInfo["type"] = strings.TrimSpace(line[3:])
		} else if strings.HasPrefix(line, "LOCATION:") {
			device.UPnPInfo["location"] = strings.TrimSpace(line[9:])
		}
	}

	service := Service{
		Name:     "UPnP Device",
		Type:     device.UPnPInfo["type"],
		Protocol: "http",
		Source:   "ssdp",
	}

	device.Services = append(device.Services, service)
}

func (sd *ServiceDiscovery) discoverMulticastGroups(interfaces []NetworkInterface) {
	if sd.verbose {
		fmt.Println("   📻 Scanning multicast groups...")
	}

	commonGroups := []string{
		"224.0.0.1",       // All Systems
		"224.0.0.2",       // All Routers
		"224.0.0.22",      // IGMP
		"224.0.0.251",     // mDNS
		"224.0.0.252",     // LLMNR
		"239.255.255.250", // SSDP
		"224.0.1.60",      // OSPF Hello
		"224.0.1.129",     // Cisco Auto-RP
		"224.2.127.254",   // SLPv2
	}

	for _, group := range commonGroups {
		sd.checkMulticastGroup(group, interfaces)
	}
}

func (sd *ServiceDiscovery) checkMulticastGroup(group string, interfaces []NetworkInterface) {
	for _, iface := range interfaces {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   iface.IP,
			Port: 0,
		})
		if err != nil {
			continue
		}

		maddr, err := net.ResolveUDPAddr("udp4", group+":1900")
		if err != nil {
			conn.Close()
			continue
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		testMsg := []byte("test")
		conn.WriteToUDP(testMsg, maddr)

		buffer := make([]byte, 1024)
		_, addr, err := conn.ReadFromUDP(buffer)
		if err == nil && addr != nil {
			sd.mu.Lock()
			ipStr := addr.IP.String()
			device, exists := sd.devices[ipStr]
			if !exists {
				device = &Device{
					IP:       addr.IP,
					Services: make([]Service, 0),
					UPnPInfo: make(map[string]string),
				}
				sd.devices[ipStr] = device
			}

			service := Service{
				Name:     fmt.Sprintf("Multicast Group %s", group),
				Type:     "multicast",
				Protocol: "udp",
				Source:   "igmp",
			}
			device.Services = append(device.Services, service)
			sd.mu.Unlock()
		}

		conn.Close()
	}
}

func (sd *ServiceDiscovery) discoverCommonServices(interfaces []NetworkInterface) {
	if sd.verbose {
		fmt.Println("   🎯 Probing common service ports...")
	}

	serviceProbes := map[int]string{
		21:   "FTP",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		67:   "DHCP Server",
		68:   "DHCP Client",
		80:   "HTTP",
		110:  "POP3",
		123:  "NTP",
		143:  "IMAP",
		161:  "SNMP",
		389:  "LDAP",
		443:  "HTTPS",
		548:  "AFP",
		631:  "IPP/CUPS",
		993:  "IMAPS",
		995:  "POP3S",
		1900: "UPnP",
		3389: "RDP",
		5353: "mDNS",
		5900: "VNC",
		8080: "HTTP Alt",
		9100: "Raw Printing",
	}

	for _, iface := range interfaces {
		for port, serviceName := range serviceProbes {
			sd.probeService(iface, port, serviceName)
		}
	}
}

func (sd *ServiceDiscovery) probeService(iface NetworkInterface, port int, serviceName string) {
	broadcastIP := sd.getBroadcastIP(iface.Subnet)
	if broadcastIP == nil {
		return
	}

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", broadcastIP.String(), port), 500*time.Millisecond)
	if err == nil {
		conn.Close()

		sd.mu.Lock()
		ipStr := broadcastIP.String()
		device, exists := sd.devices[ipStr]
		if !exists {
			device = &Device{
				IP:       broadcastIP,
				Services: make([]Service, 0),
				UPnPInfo: make(map[string]string),
			}
			sd.devices[ipStr] = device
		}

		service := Service{
			Name:     serviceName,
			Port:     port,
			Protocol: "udp",
			Source:   "probe",
		}
		device.Services = append(device.Services, service)
		sd.mu.Unlock()
	}
}

func (sd *ServiceDiscovery) getBroadcastIP(subnet *net.IPNet) net.IP {
	ip := subnet.IP.To4()
	mask := subnet.Mask
	if ip == nil || mask == nil {
		return nil
	}

	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}
	return broadcast
}

func (sd *ServiceDiscovery) MergeWithDevices(devices []Device) []Device {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	deviceMap := make(map[string]*Device)
	for i := range devices {
		deviceMap[devices[i].IP.String()] = &devices[i]
	}

	for ipStr, discoveredDevice := range sd.devices {
		if existingDevice, exists := deviceMap[ipStr]; exists {
			existingDevice.Services = append(existingDevice.Services, discoveredDevice.Services...)
			for k, v := range discoveredDevice.UPnPInfo {
				if existingDevice.UPnPInfo == nil {
					existingDevice.UPnPInfo = make(map[string]string)
				}
				existingDevice.UPnPInfo[k] = v
			}
			if existingDevice.Hostname == "" && discoveredDevice.Hostname != "" {
				existingDevice.Hostname = discoveredDevice.Hostname
			}
		} else {
			deviceMap[ipStr] = discoveredDevice
		}
	}

	result := make([]Device, 0, len(deviceMap))
	for _, device := range deviceMap {
		result = append(result, *device)
	}

	return result
}
