#!/bin/bash

# Generate release notes for GitHub releases
# Usage: ./release-notes.sh v1.0.0

set -e

TAG_NAME=${1:-${GITHUB_REF#refs/tags/}}

if [ -z "$TAG_NAME" ]; then
    echo "Usage: $0 <tag-name>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

# Get previous tag for changelog
PREVIOUS_TAG=$(git describe --abbrev=0 --tags ${TAG_NAME}^ 2>/dev/null || echo "")

echo "Generating release notes for $TAG_NAME"
if [ -n "$PREVIOUS_TAG" ]; then
    echo "Changes since $PREVIOUS_TAG"
else
    echo "Initial release"
fi

# Generate changelog
cat > release_notes.md << EOF
# Network Mapper ${TAG_NAME}

A powerful cross-platform network discovery and visualization tool.

## 🚀 What's New

EOF

if [ -n "$PREVIOUS_TAG" ]; then
    # Get commits since last tag
    git log --pretty=format:"- %s (%h)" ${PREVIOUS_TAG}..${TAG_NAME} | grep -v "Merge pull request" >> release_notes.md
else
    # Initial release
    echo "- Initial release of Network Mapper" >> release_notes.md
    echo "- Cross-platform network discovery" >> release_notes.md
    echo "- Advanced service discovery (mDNS, SSDP, UPnP)" >> release_notes.md
    echo "- MAC vendor identification" >> release_notes.md
    echo "- Beautiful CLI visualization" >> release_notes.md
fi

cat >> release_notes.md << EOF

## 📦 Installation

### Quick Install

Download the appropriate binary for your platform:

| Platform | Architecture | Download |
|----------|-------------|----------|
| Linux | x64 | \`network-mapper-linux-amd64\` |
| Linux | ARM64 | \`network-mapper-linux-arm64\` |
| macOS | Intel | \`network-mapper-darwin-amd64\` |
| macOS | Apple Silicon | \`network-mapper-darwin-arm64\` |
| Windows | x64 | \`network-mapper-windows-amd64.exe\` |

### Docker

\`\`\`bash
docker pull ghcr.io/$(echo $GITHUB_REPOSITORY | tr '[:upper:]' '[:lower:]'):${TAG_NAME}
docker run --rm --network host ghcr.io/$(echo $GITHUB_REPOSITORY | tr '[:upper:]' '[:lower:]'):${TAG_NAME}
\`\`\`

### Usage

\`\`\`bash
# Make executable (Linux/macOS)
chmod +x network-mapper-*

# Run full scan
./network-mapper-*

# Quick scan without service discovery
./network-mapper-* --no-services

# Custom timeout
./network-mapper-* --timeout 10

# Show help
./network-mapper-* --help
\`\`\`

## ✨ Features

### Core Network Discovery
- 🖥️ **Multi-platform support**: Works on Linux, macOS, and Windows
- 🔍 **Automatic network discovery**: Finds all network interfaces and subnets
- 📡 **Gateway detection**: Automatically discovers network gateways
- 🎯 **Device identification**: Identifies device types based on ports and services

### Advanced Service Discovery
- 📡 **mDNS/Bonjour discovery**: Discovers Apple devices, AirPlay, HomeKit services
- 🌐 **SSDP/UPnP scanning**: Finds media servers, smart TVs, routers, IoT devices
- 💾 **DHCP lease scanning**: Extracts device information from network infrastructure
- 🏭 **MAC vendor identification**: Looks up device manufacturers using IEEE OUI database

### Visual Features
- 🎨 **Beautiful CLI visualization**: ASCII art representation of network topology
- 📊 **Service mapping**: Shows discovered services with source identification
- 🏷️ **Vendor information**: Displays manufacturer info for network devices

## 🔒 Security & Verification

Verify your download with the provided checksums:

\`\`\`bash
# Download checksums.txt and verify
sha256sum -c checksums.txt
\`\`\`

## 🐛 Known Issues

- Some advanced features may require elevated privileges
- Windows Defender may flag the binary (false positive)
- Large networks may take longer to scan completely

## 📝 Full Changelog

View the complete changelog at: https://github.com/$GITHUB_REPOSITORY/compare/${PREVIOUS_TAG}...${TAG_NAME}

## 🤝 Contributing

Contributions are welcome! Please see our [Contributing Guide](.github/CONTRIBUTING.md) for details.

---

**Built with ❤️ using GitHub Actions**

EOF

echo "Release notes generated in release_notes.md"