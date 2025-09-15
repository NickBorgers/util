package main

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

type Service struct {
	Name        string
	Type        string
	Domain      string
	Port        int
	Protocol    string
	TXT         map[string]string
	Source      string // "mdns", "ssdp", "upnp", etc.
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
	}
}

func (ns *NetworkScanner) SetOptions(disableServices bool, disableDNS bool, timeout int, verbose bool) {
	ns.disableServiceDiscovery = disableServices
	ns.disableDNSLookup = disableDNS
	ns.scanTimeout = time.Duration(timeout) * time.Second
	ns.verbose = verbose
	ns.dnsResolver = NewDNSResolver(10*time.Second, verbose)
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

		sd := NewServiceDiscovery()
		sd.DiscoverServices(ns.interfaces, ns.scanTimeout)

		fmt.Println("\n🔗 Merging service information...")
		ns.devices = sd.MergeWithDevices(ns.devices)
		ns.enhanceDevicesWithVendorInfo(ns.devices, mvl)

		fmt.Println("🧠 Enhancing device identification...")
		for i := range ns.devices {
			ns.enhanceDeviceTypeWithServices(&ns.devices[i])
		}

		fmt.Println("💾 Scanning DHCP lease information...")
		dhcpLeases := ns.scanDHCPLeases()
		ns.enhanceDevicesWithDHCPInfo(ns.devices, dhcpLeases)

		fmt.Printf("✅ Enhanced %d devices with service information\n", len(ns.devices))
	} else {
		fmt.Println("⏭️  Skipping service discovery (disabled)")
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
	var wg sync.WaitGroup

	for _, iface := range ns.interfaces {
		wg.Add(1)
		go func(netInterface NetworkInterface) {
			defer wg.Done()
			ns.scanSubnet(netInterface)
		}(iface)
	}

	wg.Wait()
}

func (ns *NetworkScanner) scanSubnet(netInterface NetworkInterface) {
	ip := netInterface.Subnet.IP

	for i := 1; i < 255; i++ {
		testIP := make(net.IP, len(ip))
		copy(testIP, ip)
		testIP[3] = byte(i)

		if !netInterface.Subnet.Contains(testIP) {
			continue
		}

		if ns.pingHost(testIP.String()) {
			ports := ns.scanCommonPorts(testIP)
			device := Device{
				IP:         testIP,
				MAC:        ns.getMACAddress(testIP),
				MACVendor:  "",
				Hostname:   "", // Will be populated in bulk DNS lookup
				DeviceType: ns.identifyDeviceType(testIP, ports),
				IsGateway:  testIP.Equal(netInterface.Gateway),
				Ports:      ports,
				Services:   make([]Service, 0),
				UPnPInfo:   make(map[string]string),
			}

			ns.mu.Lock()
			ns.devices = append(ns.devices, device)
			ns.mu.Unlock()
		}
	}
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

