# Network Mapper 🌐

[![Build Status](https://github.com/NickBorgers/util/workflows/Build%20and%20Test/badge.svg)](https://github.com/NickBorgers/util/actions)
[![Release](https://github.com/NickBorgers/util/workflows/Release/badge.svg)](https://github.com/NickBorgers/util/releases)
[![Docker](https://img.shields.io/badge/docker-available-blue)](https://github.com/NickBorgers/util/pkgs/container/network-mapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/NickBorgers/util)](https://goreportcard.com/report/github.com/NickBorgers/util)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A cross-platform CLI tool that scans your local network, discovers devices, and presents a beautiful pictographic representation of the network topology.

## Features

### Core Network Discovery
- 🖥️ **Multi-platform support**: Works on Linux, macOS, and Windows
- 🔍 **Automatic network discovery**: Finds all network interfaces and their subnets
- 📡 **Gateway detection**: Automatically discovers network gateways using platform-specific commands
- 🎯 **Device identification**: Identifies device types based on open ports and services
- 🔌 **Port scanning**: Scans common ports to fingerprint devices and services
- 🏷️ **MAC address resolution**: Discovers MAC addresses using ARP tables
- 🌐 **Enhanced DNS resolution**: Bulk reverse DNS lookups with intelligent caching and validation
- 🏠 **Smart hostname display**: Prioritizes DNS names over IP addresses in output

### Advanced Service Discovery
- 📡 **mDNS/Bonjour discovery**: Discovers Apple devices, AirPlay, HomeKit, and other Bonjour services
- 🌐 **SSDP/UPnP scanning**: Finds media servers, smart TVs, routers, and IoT devices
- 📻 **Multicast group detection**: Monitors IGMP traffic and common multicast addresses
- 💾 **DHCP lease scanning**: Extracts device information from DHCP server lease tables
- 🏭 **MAC vendor identification**: Looks up device manufacturers using IEEE OUI database
- 🧠 **Smart device classification**: Enhanced device type detection using service signatures

### Network Services Detected
- **Apple ecosystem**: AirPlay, HomeKit, Time Machine, DAAP, Apple TV
- **Google services**: Chromecast, Google Cast devices
- **Media services**: Plex, DLNA, media servers and renderers
- **Network infrastructure**: Routers, switches, access points, printers
- **Smart home devices**: IoT devices, security cameras, smart speakers
- **Development tools**: SSH servers, web servers, databases

### Visual Features
- 🎨 **Beautiful CLI visualization**: ASCII art representation of network topology
- 📊 **Service mapping**: Shows discovered services with source identification
- 🏷️ **Vendor information**: Displays manufacturer info for network devices
- 📈 **Comprehensive reporting**: Detailed statistics and device summaries

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [GitHub Releases](https://github.com/NickBorgers/util/releases) page:

- **Linux (x64)**: `network-mapper-linux-amd64`
- **Linux (ARM64)**: `network-mapper-linux-arm64`
- **macOS (Intel)**: `network-mapper-darwin-amd64`
- **macOS (Apple Silicon)**: `network-mapper-darwin-arm64`
- **Windows (x64)**: `network-mapper-windows-amd64.exe`

Make the binary executable and run:
```bash
chmod +x network-mapper-*
./network-mapper-* --help
```

### Docker

Run directly with Docker:
```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/nickborgers/network-mapper:latest

# Run with host network access
docker run --rm --network host ghcr.io/nickborgers/network-mapper:latest

# Quick scan without service discovery
docker run --rm --network host ghcr.io/nickborgers/network-mapper:latest --no-services
```

### Build from Source

#### Prerequisites

- Go 1.21 or later
- Network tools available on your system:
  - Linux/macOS: `ping`, `arp`, `ip`/`route`
  - Windows: `ping`, `arp`, `route`

#### Using DevContainer (Recommended)

This project includes a DevContainer configuration for consistent development:

```bash
# Clone and start devcontainer
git clone https://github.com/NickBorgers/util.git
cd util/network-mapper
devcontainer up --workspace-folder .

# Build all platforms
devcontainer exec --workspace-folder . make build-all

# Run in container
devcontainer exec --workspace-folder . go run .
```

#### Manual Build

```bash
git clone https://github.com/NickBorgers/util.git
cd util/network-mapper
go mod tidy
go build -o network-mapper
```

### Package Managers

#### Homebrew (macOS/Linux)

```bash
# Coming soon
brew install nickborgers/tap/network-mapper
```

#### Chocolatey (Windows)

```bash
# Coming soon
choco install network-mapper
```

#### APT/YUM Packages

```bash
# Coming soon - .deb and .rpm packages
```

## Usage

```bash
./network-mapper
```

The tool will automatically:
1. Discover all network interfaces on your system
2. Scan each subnet for active devices
3. Identify device types and services
4. Display a beautiful network topology map

## Example Output

```
🌐 Network Mapper v1.0 - linux
==========================================
📡 Discovering network interfaces...
✅ Found 2 network interfaces
  [1] eth0: 192.168.1.100/255.255.255.0 (Gateway: 192.168.1.1)
  [2] wlan0: 10.0.0.50/255.255.255.0 (Gateway: 10.0.0.1)

🔍 Scanning for devices...
✅ Found 8 devices

🌐 Performing reverse DNS lookups...
✅ DNS lookups complete: 6/8 successful

🔍 Initializing service discovery...
   📋 Loaded MAC vendor database from cache
🔍 Discovering network services...
   📡 Scanning mDNS/Bonjour services...
   🔌 Scanning SSDP/UPnP devices...
   📻 Scanning multicast groups...
   🎯 Probing common service ports...

🔗 Merging service information...
🧠 Enhancing device identification...
💾 Scanning DHCP lease information...
   💾 Scanning DHCP lease tables...
   ✅ Found 12 DHCP lease entries
✅ Enhanced 8 devices with service information

📊 Network Topology Map
═══════════════════════════════════════════════════════════════

🌐 Network Segment 1: eth0
   IP: 192.168.1.100 | Subnet: 192.168.1.0/24
   │
   ├─🏠 Gateway: 192.168.1.1
   │  │
   ├─🏠 router.local (192.168.1.1) (Router/Gateway)
   │     🔌 Ports: 53(DNS), 80(HTTP), 443(HTTPS)
   │     📡 Services: HTTP:80(probe), UPnP Device(ssdp), DNS:53(probe)
   │     🌐 UPnP: Linux/3.14 UPnP/1.0 Linksys E8450/1.0
   │     🏷️  MAC: aa:bb:cc:dd:ee:ff (Linksys LLC)
   ├─📺 apple-tv.local (192.168.1.15) (Apple TV/AirPlay)
   │     🔌 Ports: 7000, 62078
   │     📡 Services: AirPlay:7000(mdns), Remote Audio:7000(mdns)
   │     🏷️  MAC: 11:22:33:44:55:66 (Apple, Inc.)
   ├─🖥️  desktop-pc.local (192.168.1.10) (Windows PC)
   │     🔌 Ports: 135(RPC), 3389(RDP)
   │     🏷️  MAC: 22:33:44:55:66:77 (Intel Corporation)
   ├─🥧 homeassistant.local (192.168.1.25) (Raspberry Pi)
   │     🔌 Ports: 22(SSH), 8123
   │     📡 Services: SSH:22(probe), HomeAssistant:8123(mdns)
   │     🏷️  MAC: dc:a6:32:11:22:33 (Raspberry Pi Foundation)
   └─🐧 webserver.local (192.168.1.20) (Linux Server)
         🔌 Ports: 22(SSH), 80(HTTP), 443(HTTPS)
         📡 Services: SSH:22(mdns), HTTP:80(mdns), Plex:32400(ssdp)
         🏷️  MAC: 77:88:99:aa:bb:cc (Intel Corporation)

📈 Discovery Summary
────────────────────────────────────────────────────────────────
🌐 Network Interfaces: 2
📱 Total Devices Found: 8
🔍 Device Types Detected:
   • Router/Gateway: 1
   • Apple TV/AirPlay: 1
   • Windows PC: 2
   • Raspberry Pi: 1
   • Linux Server: 2
   • Apple Device: 1
🔌 Total Open Ports: 28
📡 Active Network Segments: 2/2
✨ Scan Complete!
```

## Device Types & Service Discovery

The tool can identify various device types using multiple detection methods:

### Traditional Port-Based Detection
- **Windows PC**: Devices with RDP (3389) and SMB (135/139) ports
- **Linux Server**: Devices with SSH (22) and HTTP/HTTPS (80/443) ports
- **Linux/Unix**: Devices with SSH (22) port only
- **Web Server**: Devices with HTTP/HTTPS ports only
- **File Server**: Devices with SMB ports
- **Network Printer**: Devices with IPP (631) or raw printing (9100) ports
- **Router/Gateway**: Devices with DNS (53) and DHCP (67) services

### Enhanced Service-Based Detection
- **Apple TV/AirPlay**: Detected via mDNS AirPlay services and Apple vendor MAC
- **Chromecast/Google Device**: Identified through Google Cast mDNS services
- **HomeKit Device**: Found via HomeKit Accessory Protocol (HAP) services
- **Media Server**: Detected through DLNA/UPnP media server announcements
- **Apple Device**: Identified via Apple services (DAAP, Time Machine, etc.) and MAC vendor
- **Raspberry Pi**: Detected through Raspberry Pi Foundation MAC addresses
- **Smart Home Devices**: IoT devices found via various mDNS service types

### Service Discovery Protocols Used
- **mDNS/Bonjour**: Discovers Apple ecosystem services, IoT devices, printers
- **SSDP/UPnP**: Finds media devices, routers, smart TVs, NAS devices
- **DHCP Lease Tables**: Extracts hostnames and MAC addresses from network infrastructure
- **IEEE OUI Database**: Identifies device manufacturers from MAC addresses
- **Multicast Monitoring**: Detects devices participating in multicast groups

## Security Considerations

- The tool uses non-intrusive scanning methods
- Only scans common ports (22, 80, 443, etc.)
- Uses standard network tools available on the system
- Respects network timeouts to avoid being too aggressive
- Only scans local network subnets

## Development & Contributing

### Development Setup

See the [Contributing Guide](.github/CONTRIBUTING.md) for detailed development setup instructions.

### Quick Start

```bash
# Using DevContainer (recommended)
git clone https://github.com/NickBorgers/util.git
cd util/network-mapper
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . go run .

# Manual setup
go mod download
go run .
```

### CI/CD Pipeline

The project uses GitHub Actions for automated building, testing, and releases:

#### Automated Builds
- **Triggered on**: Push to `main`/`develop`, Pull Requests
- **Platforms**: Linux (x64/ARM64), macOS (Intel/Apple Silicon), Windows (x64)
- **Quality Gates**: Code formatting, linting, security scanning, unit tests
- **Artifacts**: Cross-platform binaries with checksums

#### Automated Releases
- **Triggered on**: Git tags (`v*` pattern)
- **Deliverables**:
  - GitHub Release with binaries and checksums
  - Docker images published to GitHub Container Registry
  - Multi-platform support with compressed archives

#### Security & Quality
- **Security Scanning**: Gosec for vulnerability detection
- **Dependency Scanning**: Nancy for known vulnerabilities
- **Code Quality**: Static analysis with staticcheck
- **SARIF Reports**: Integrated with GitHub Security tab

### Contributing

1. Fork the repository
2. Create a feature branch from `develop`
3. Make your changes
4. Add tests for new functionality
5. Ensure all CI checks pass
6. Submit a pull request

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for detailed guidelines.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Troubleshooting

### Permission Issues
Some operations may require elevated privileges:
- Linux/macOS: Run with `sudo` if needed for network interface access
- Windows: Run as Administrator if needed

### No Devices Found
- Check your network connection
- Ensure you're on a network with other devices
- Try running with elevated privileges
- Check firewall settings that might block scanning

### Gateway Not Detected
- The tool will fall back to guessing the gateway (.1 address)
- Ensure routing commands are available on your system