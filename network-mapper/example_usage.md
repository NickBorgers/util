# Network Mapper Usage Examples

## Basic Usage

```bash
# Full scan with all service discovery
./network-mapper

# Quick scan without service discovery
./network-mapper --no-services

# Fast scan without DNS lookups
./network-mapper --no-dns

# Ultra-fast scan without services or DNS
./network-mapper --no-services --no-dns

# Extended scan with longer timeout
./network-mapper --timeout 10

# Verbose output
./network-mapper --verbose
```

## Command Line Options

```bash
# Display help
./network-mapper --help

# Show version
./network-mapper --version

# Disable advanced service discovery for faster scanning
./network-mapper --no-services

# Set service discovery timeout (default: 5 seconds)
./network-mapper --timeout 15

# Enable verbose output for debugging
./network-mapper --verbose
```

## Service Discovery Details

When service discovery is enabled (default), the tool will:

1. **Download IEEE OUI Database**: Fetch MAC vendor information from IEEE
2. **mDNS/Bonjour Scanning**: Query for common service types including:
   - `_http._tcp` - Web servers
   - `_ssh._tcp` - SSH services
   - `_airplay._tcp` - Apple AirPlay devices
   - `_googlecast._tcp` - Google Chromecast devices
   - `_homekit._tcp` - HomeKit accessories
   - `_printer._tcp` - Network printers
   - `_smb._tcp` - File sharing services
   - And many more...

3. **SSDP/UPnP Discovery**: Search for UPnP devices including:
   - Media servers and renderers
   - Internet gateways and routers
   - Smart TVs and streaming devices
   - Network storage devices

4. **Multicast Group Monitoring**: Listen for devices on common multicast addresses:
   - `224.0.0.251` - mDNS
   - `239.255.255.250` - SSDP
   - `224.0.0.1` - All Systems
   - `224.0.0.2` - All Routers

5. **DHCP Lease Scanning**: Parse DHCP lease files from:
   - `/var/lib/dhcp/dhcpd.leases` (Linux)
   - `/var/db/dhcpd_leases` (macOS)
   - Windows DHCP server tables

## Performance Considerations

- **Full scan**: 15-30 seconds depending on network size and timeout
- **Quick scan** (`--no-services`): 5-10 seconds
- **Network impact**: Minimal, uses standard discovery protocols
- **Permissions**: May require elevated privileges for some features

## Detected Device Examples

### Smart Home Devices
- **Philips Hue Bridge**: Detected via mDNS and SSDP
- **Ring Doorbell**: Found through UPnP announcements
- **Nest Thermostat**: Identified via Google Cast services
- **HomeKit Devices**: Discovered through HAP services

### Media Devices
- **Apple TV**: AirPlay services + Apple MAC vendor
- **Chromecast**: Google Cast mDNS services
- **Plex Server**: DLNA/UPnP media server announcements
- **Smart TVs**: Various UPnP device types

### Network Infrastructure
- **Routers**: DNS/DHCP ports + SSDP IGD services
- **Network Printers**: IPP services via mDNS
- **NAS Devices**: SMB services + vendor identification
- **Access Points**: Vendor MAC + network services

### Development Devices
- **Raspberry Pi**: MAC vendor + SSH services
- **Arduino/ESP**: Custom mDNS service announcements
- **Development Servers**: SSH + HTTP services
- **Docker Containers**: Port patterns + service names