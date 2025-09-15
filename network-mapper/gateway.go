package main

import (
	"bufio"
	"net"
	"os/exec"
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