# Network Mapper 🌐

[![Build Status](https://github.com/NickBorgers/util/workflows/Build%20and%20Test/badge.svg)](https://github.com/NickBorgers/util/actions)
[![Release](https://github.com/NickBorgers/util/workflows/Release/badge.svg)](https://github.com/NickBorgers/util/releases)
[![Docker](https://img.shields.io/badge/docker-available-blue)](https://github.com/NickBorgers/util/pkgs/container/network-mapper)
[![Homebrew](https://img.shields.io/badge/homebrew-available-orange)](https://github.com/NickBorgers/homebrew-tap)
[![Chocolatey](https://img.shields.io/badge/chocolatey-available-brown)](https://chocolatey.org/packages/network-mapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/NickBorgers/util)](https://goreportcard.com/report/github.com/NickBorgers/util)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Quick Install:** `brew install nickborgers/tap/network-mapper` | `choco install network-mapper` | `docker run --rm --network host ghcr.io/nickborgers/network-mapper:latest`

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

Choose your preferred installation method:

### 🍺 Homebrew (macOS/Linux)

The easiest way to install on macOS and Linux:

```bash
# Add the tap and install
brew tap nickborgers/tap
brew install network-mapper

# Or install directly
brew install nickborgers/tap/network-mapper

# Upgrade to latest version
brew upgrade network-mapper
```

### 🍫 Chocolatey (Windows)

For Windows users with Chocolatey:

```powershell
# Install
choco install network-mapper

# Upgrade to latest version
choco upgrade network-mapper
```

### 🐳 Docker

Run directly with Docker (cross-platform):

```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/nickborgers/network-mapper:latest

# Run with host network access
docker run --rm --network host ghcr.io/nickborgers/network-mapper:latest

# Quick scan without service discovery
docker run --rm --network host ghcr.io/nickborgers/network-mapper:latest --scan-mode quick
```

### 📦 Pre-built Binaries

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

See [Installation](#installation) section above for complete package manager setup.

## Usage

### Basic Usage

```bash
./network-mapper
```

### Scan Modes & Intelligent Discovery

Network-mapper uses sophisticated subnet discovery to find devices beyond your immediate network. The tool supports multiple scan modes:

```bash
# Intelligent scan (default) - smart subnet discovery
./network-mapper --scan-mode intelligent --thoroughness 3

# Quick scan - interface subnets only
./network-mapper --scan-mode quick

# Brute force scans - comprehensive but slower
./network-mapper --scan-mode brute-expanded      # Common private ranges
./network-mapper --scan-mode brute-comprehensive # Full RFC1918 ranges
./network-mapper --scan-mode brute-firewall      # Security testing ranges
```

#### Scan Mode Details

- **`intelligent`** (default): 4-phase intelligent subnet discovery using network heuristics
- **`quick`**: Only scans interface subnets - fastest option
- **`brute-expanded`**: Scans common private network ranges systematically
- **`brute-comprehensive`**: Exhaustive scan of all RFC1918 private ranges
- **`brute-firewall`**: Security-focused scan targeting common network segments

#### Thoroughness Levels (Intelligent Mode Only)

Control the breadth of intelligent discovery with `--thoroughness 1-5`:

```bash
# Minimal discovery - only most common subnets
./network-mapper --thoroughness 1

# Light discovery - few common ranges
./network-mapper --thoroughness 2

# Default discovery - balanced approach
./network-mapper --thoroughness 3

# Thorough discovery - extensive subnet candidates
./network-mapper --thoroughness 4

# Exhaustive discovery - maximum subnet coverage
./network-mapper --thoroughness 5
```

**Thoroughness Level Details:**
- **Level 1**: Tests `.0.0` and `.1.0` subnets only (2-3 candidates)
- **Level 2**: Adds `.10.0` and `.100.0` subnets (4-6 candidates)
- **Level 3**: Common admin subnets: `.0`, `.1`, `.10`, `.20`, `.30`, `.50`, `.100`, `.168`, `.254` (9-12 candidates)
- **Level 4**: Extended ranges including `.2`, `.5`, `.11`, `.15`, `.25`, `.40`, etc. (20-30 candidates)
- **Level 5**: Every 10th subnet + all level 4 ranges (40-60 candidates)

### Intelligent Discovery Process

When you see "🔍 Probing 19 subnet candidates for activity...", here's what's happening:

#### Phase 1: Interface Subnets (Priority 100)
- Adds all network interface subnets as high-priority candidates
- For large subnets (>/24), creates targeted /24 around interface IP
- Example: Interface on `192.168.1.5/24` → candidate `192.168.1.0/24`

#### Phase 2: Adjacent Subnets (Priority 80)
- Generates numerically adjacent subnets to interface networks
- For `/24` networks: checks previous/next third octet
- Example: From `192.168.1.0/24` → tests `192.168.0.0/24` and `192.168.2.0/24`

#### Phase 3: Common Subnet Patterns (Priority 60)
- Creates candidates based on common network administration patterns
- Uses thoroughness level to determine which third octets to test
- Example: In `192.168.x.0/24` range, tests common `x` values like `0`, `1`, `10`, `50`, `100`, `254`

#### Phase 4: Gateway Probing
- Tests common gateway IPs in each candidate subnet: `.1`, `.254`, `.10`, `.100`, `.50`
- Uses ICMP ping to verify gateway accessibility
- Marks subnets as active when gateway responds
- Example output: `✅ Found active gateway 192.168.254.1 in subnet 192.168.254.0/24`

### Route Table Integration

Network-mapper consults your system's routing table to accurately determine which interface should handle discovered subnets:

- **Linux**: Uses `ip route show` command
- **macOS**: Uses `netstat -rn -f inet` command
- **Windows**: Uses `route print` command

Discovered subnets are displayed with the correct interface based on actual routing rules, not just heuristics.

### Practical Examples & Usage Scenarios

#### Home Network Security Audit
```bash
# Quick overview of your immediate network
./network-mapper --scan-mode quick

# Thorough discovery of all network segments
./network-mapper --scan-mode intelligent --thoroughness 4 --verbose

# Example output when multiple VLANs are discovered:
🔍 Probing 19 subnet candidates for activity...
✅ Found active gateway 192.168.1.1 in subnet 192.168.1.0/24      # Main network
✅ Found active gateway 192.168.10.1 in subnet 192.168.10.0/24   # IoT VLAN
✅ Found active gateway 192.168.50.1 in subnet 192.168.50.0/24   # Guest network
✅ Found active gateway 10.10.0.1 in subnet 10.10.0.0/24         # Work VLAN
✅ Found active gateway 172.16.0.1 in subnet 172.16.0.0/24       # Admin network
```

#### Enterprise Network Discovery
```bash
# Conservative scan for large corporate networks
./network-mapper --scan-mode intelligent --thoroughness 2

# Security assessment with firewall testing
./network-mapper --scan-mode brute-firewall --verbose
```

#### Troubleshooting Network Issues
```bash
# Fast scan to identify connectivity issues
./network-mapper --scan-mode quick --no-services

# Detailed scan with verbose output for debugging
./network-mapper --verbose --timeout 10
```

#### Understanding Your Network Topology
When network-mapper finds multiple subnets, it means:

- **🏠 Home Networks**: You likely have VLANs for IoT devices, guest access, or network segmentation
- **🏢 Corporate**: Different departments, security zones, or VLAN configurations
- **🔍 Security**: Previously unknown network segments that need investigation

**Common subnet patterns discovered:**
- `.0.0/24` - Main/default network
- `.1.0/24` - Management network
- `.10.0/24` - IoT/smart home devices
- `.50.0/24` - Guest network
- `.100.0/24` - Work/office network
- `.254.0/24` - Administrative/infrastructure

### Other Options

```bash
# Fast scanning without DNS lookups or service discovery
./network-mapper --no-dns --no-services --scan-mode quick

# Custom timeout for slower networks
./network-mapper --timeout 10 --scan-mode intelligent
```

The tool will automatically:
1. Discover all network interfaces on your system
2. Expand scan ranges based on RFC1918 networks and selected scan mode
3. Scan for active devices across the determined ranges
4. Identify device types and services
5. Display a beautiful network topology map

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

## Build Transparency & Antivirus Compatibility

Network-mapper is built with **complete transparency** to help users understand exactly how the binary works and to minimize antivirus false positives.

### Key Transparency Features

- **🔍 Full debug symbols**: Windows binaries preserve all debugging information for complete transparency
- **📖 Open source**: Every network operation is visible in the source code
- **🛠️ Standard libraries**: Uses Go's standard `net` package in documented, legitimate ways
- **🚫 Zero obfuscation**: No packing, compression, or symbol stripping that looks suspicious

### Windows Defender Compatibility

Our Windows builds specifically avoid the `-s -w` build flags that are known triggers for `Trojan:Script/Wacatac.B!ml` false positives. This approach:

- Preserves full debug information for security analysis
- Makes network operations completely transparent to ML algorithms
- Allows developers to step through and verify network discovery logic

For detailed debugging instructions and transparency documentation, see [`BUILD_TRANSPARENCY.md`](BUILD_TRANSPARENCY.md).

## Security Considerations & Firewall Testing

### Safe Scanning Practices
- The tool uses non-intrusive scanning methods
- Only scans common ports (22, 80, 443, etc.)
- Uses standard network tools available on the system
- Respects network timeouts to avoid being too aggressive
- Only scans RFC1918 private address spaces

### Firewall Configuration Testing

The `--scan-mode firewall-test` option is specifically designed for network security testing:

```bash
# Test your firewall configuration across network segments
./network-mapper --scan-mode firewall-test --verbose
```

This mode helps identify:
- **Network segmentation gaps**: Devices accessible across supposed network boundaries
- **Firewall rule effectiveness**: Whether inter-VLAN blocking is working properly
- **Hidden network ranges**: Additional subnets that may not be immediately visible
- **Security policy compliance**: Verification that network isolation is functioning

### Example Use Cases

1. **Home Network Security**: If your home network is `192.168.1.0/24` but your WiFi guest network is `192.168.50.0/24`, firewall-test mode will scan both ranges to verify isolation.

2. **Corporate Network Audit**: Test whether development networks can access production subnets, or if guest WiFi is properly segmented.

3. **Network Discovery**: Find additional network ranges that may not be obvious from your current interface configuration.

### RFC1918 Network Expansion

The tool intelligently expands scan ranges based on detected RFC1918 networks:

- **10.x.x.x networks**: Expands to /16 and /12 ranges as appropriate
- **172.16-31.x.x networks**: Expands to /20 and broader ranges
- **192.168.x.x networks**: Scans adjacent /24 networks

This expansion only occurs within RFC1918 private address spaces for safety.

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

# Install git hooks for quality checks (recommended)
./.githooks/install.sh

# Manual setup
go mod download
go run .
```

### Git Hooks

This project includes git hooks that run the same quality checks as the CI pipeline locally:

```bash
# Install git hooks
./.githooks/install.sh

# The pre-commit hook will now run automatically before each commit:
# - go mod download & verify
# - go test -v ./...
# - go vet ./...
# - staticcheck ./...
# - gofmt formatting check
```

The hooks use the devcontainer to ensure consistency with CI. See [.githooks/README.md](.githooks/README.md) for details.

### CI/CD Pipeline

The project uses GitHub Actions for automated building, testing, and releases:

#### Automated Builds
- **Triggered on**: Push to `main`/`develop`, Pull Requests
- **Platforms**: Linux (x64/ARM64), macOS (Intel/Apple Silicon), Windows (x64)
- **Quality Gates**: Code formatting, linting, security scanning, unit tests
- **Artifacts**: Cross-platform binaries with checksums

#### Automated Releases
- **Triggered on**: Git tags (`network-mapper-v*` or legacy `v*` pattern)
- **Deliverables**:
  - GitHub Release with binaries and checksums
  - Docker images published to GitHub Container Registry
  - Multi-platform support with compressed archives

#### Package Manager Automation
- **Homebrew**: Automatically updates [homebrew-tap](https://github.com/NickBorgers/homebrew-tap) formula
- **Chocolatey**: Publishes to [Chocolatey Community Repository](https://community.chocolatey.org/packages/network-mapper)
- **Triggered on**: Release publication (fully automated)
- **Updates**: Checksums, download URLs, and version numbers across all package managers
- **Secrets Required**:
  - `HOMEBREW_TAP_TOKEN`: GitHub token with write access to homebrew-tap repository
  - `CHOCOLATEY_API_KEY`: API key for publishing to Chocolatey community repository

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