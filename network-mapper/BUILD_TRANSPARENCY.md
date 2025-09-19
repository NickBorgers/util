# Build Transparency & Antivirus Compatibility

## Philosophy: Maximum Transparency

Network-mapper is built with complete transparency to help users understand exactly how the binary works. This approach serves multiple purposes:

1. **Educational Value**: Anyone can debug and understand the network discovery process
2. **Security Assurance**: Full debug symbols allow security researchers to verify functionality
3. **Antivirus Compatibility**: Transparent binaries are less likely to trigger false positives

## Build Configuration

### Windows Builds - Antivirus Friendly
Windows binaries are built with **full debug symbols preserved** to avoid false positives:

```bash
# Windows build command (no symbol stripping)
go build -ldflags "-X main.version=${VERSION}" -o network-mapper-windows-amd64.exe .
```

**Why this approach?**
- The `-s -w` flags (strip symbols/DWARF) are known triggers for `Trojan:Script/Wacatac.B!ml` detection
- Full debug information makes the binary behavior transparent to antivirus ML algorithms
- Allows developers to step through the exact network discovery logic

### Other Platforms - Optimized
Linux/macOS builds use optimized compilation for smaller size:

```bash
# Linux/macOS build command (optimized)
go build -ldflags "-X main.version=${VERSION} -s -w" -o network-mapper .
```

## Debugging the Binary

### Using Go's Built-in Tools

1. **View Build Information**:
   ```bash
   go version -m network-mapper.exe
   ```

2. **Examine Symbol Table**:
   ```bash
   go tool nm network-mapper.exe | head -20
   ```

3. **Disassemble Specific Functions**:
   ```bash
   go tool objdump -s main.main network-mapper.exe
   ```

### Using Delve Debugger

1. **Install Delve**:
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

2. **Debug Network Discovery**:
   ```bash
   dlv exec network-mapper.exe
   (dlv) break main.main
   (dlv) break (*NetworkScanner).DiscoverDevices
   (dlv) continue
   ```

3. **Inspect Network Scanning Logic**:
   ```bash
   (dlv) break (*IntelligentDiscovery).DiscoverActiveSubnets
   (dlv) continue
   (dlv) vars
   ```

## Source Code Inspection

### Key Files for Network Behavior Understanding

1. **`scanner.go`**: Main scanning orchestration
   - `DiscoverDevices()`: Entry point for network discovery
   - `scanNetwork()`: Core scanning logic

2. **`intelligent_discovery.go`**: Smart subnet detection
   - `DiscoverActiveSubnets()`: Heuristic-based discovery
   - `getTrulyCommonSubnets()`: Common network ranges

3. **`service_discovery.go`**: Service detection
   - `discoverMDNS()`: Bonjour/mDNS discovery
   - `discoverSSDP()`: UPnP device discovery

4. **`ping.go`**: Host reachability testing
   - `pingHost()`: Platform-specific ping implementation

### Network Operations

The tool performs these **legitimate network operations**:

```go
// TCP port scanning (clearly visible in source)
conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)

// UDP multicast discovery (documented behavior)
conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: iface.IP, Port: 0})

// System command execution (transparent implementation)
cmd := exec.Command("ping", "-c", "1", "-W", "1", host)
```

## Why This Approach Beats Antivirus False Positives

1. **Behavioral Transparency**: ML algorithms can clearly see legitimate network library usage
2. **No Obfuscation**: Zero packing, compression, or symbol stripping that looks suspicious
3. **Open Source**: Complete source code is available for audit
4. **Standard Library Usage**: Uses Go's standard `net` package in documented ways

## For Security Researchers

Want to verify the network behavior? Here's how:

1. **Static Analysis**:
   ```bash
   strings network-mapper.exe | grep -E "(ping|scan|discover)"
   ```

2. **Network Monitoring**:
   ```bash
   # Monitor actual network calls
   tcpdump -i any host <target_ip>
   ```

3. **System Call Tracing**:
   ```bash
   # Windows
   strace network-mapper.exe

   # Linux
   strace ./network-mapper
   ```

The goal is to make network-mapper the **most transparent network discovery tool** available, where every operation can be understood, debugged, and verified.