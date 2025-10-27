# Util Repository - Comprehensive Guide

## Repository Overview

This is a personal utilities repository designed to maintain portable, self-contained tools that persist across job changes and environment setups. The repository follows a philosophy of Docker-based, security-conscious utilities that work consistently across platforms with minimal dependency management.

## Core Design Philosophy

### Containerized Everything
- **Docker-first approach**: Every utility runs in an isolated Docker container
- **Zero local dependencies**: No need to install ffmpeg, imagemagick, rclone, or other tools locally
- **Network isolation**: Older utilities use `--network=none` flag for security (no network access to prevent data exfiltration)
- **Platform independence**: Works identically on macOS, Linux, and Windows wherever Docker runs

### Profile-Based Installation
The `profile` file is the **primary installation mechanism** for older utilities. Users simply add the contents of this file to their shell profile (`.bashrc`, `.zshrc`, etc.) to gain access to all utilities as shell functions. This approach means:
- No complex installation scripts needed
- No PATH modifications required
- Functions are immediately available in new terminal sessions
- Complete portability - just copy the profile file

### Evolution of Approach
The repository shows a clear evolution in utility design:

#### **Early Generation** (Shell Functions + Simple Docker Images)
Utilities like `mov-to-gif`, `update-pdf`, `reduce_framerate`, `heic_to_jpeg`, `unrar`, and `stabilize_video` follow a simple pattern:
- Minimal Alpine Linux base images
- Single-purpose tools (ffmpeg, imagemagick, ghostscript)
- Shell functions in profile file that wrap Docker commands
- Inline execution with `docker run --rm`
- Security-focused with `--network=none`

#### **Middle Generation** (Container Services)
`onedrive-backup`, `unraid-util`, and `stress` represent a transition:
- Long-running container services rather than one-shot commands
- More sophisticated scripts (bash with loops, error handling)
- Environment variable configuration
- Purpose-built for specific infrastructure needs
- Still Dockerized but meant to run continuously

#### **Modern Generation** (Full-Featured Applications)
`network-mapper` and `backup-photos-to-gdrive` represent the current state:
- **network-mapper**: Full Go application with proper releases, CI/CD, multiple install methods (Homebrew, Chocolatey, Docker), extensive documentation, devcontainer development environment
- **backup-photos-to-gdrive**: Sophisticated bash script with retry logic, configuration validation, comprehensive environment variables, detailed documentation

## Utility Inventory

### 🎬 Media Conversion Tools (Early Generation)

#### mov_to_gif
**Location**: `mov-to-gif/`
**Purpose**: Convert QuickTime .mov files to animated GIFs for documentation
**Implementation**: Alpine + ffmpeg + imagemagick
- Uses ffmpeg to extract frames at 5fps with intelligent scaling
- Uses ImageMagick convert to optimize the GIF
- Credits Stack Overflow answers in source code
- Shell script: `mov-to-gif.sh`

#### reduce_framerate
**Purpose**: Reduce video file size by lowering framerate to 15fps
**Profile function**: Wraps ffmpeg in mov-to-gif container
**Use case**: Screen recordings for colleagues where high framerate is unnecessary

#### heic_to_jpeg
**Purpose**: Convert Apple HEIC photos to JPEG for compatibility
**Profile function**: Uses ImageMagick convert with 90% quality
**Use case**: Sharing Apple photos with non-Apple users/tools

#### stabilize_video
**Purpose**: Video stabilization using ffmpeg vidstab
**Profile function**: Two-pass stabilization (detect then transform)
**Parameters**: Takes zoom percentage (default 5%)
**Technical**: Detects transforms then applies with 30-smoothing, CRF 19, slow preset

#### update_pdf
**Location**: `update-pdf/`
**Purpose**: Upgrade PDF files to version 1.4 for compatibility
**Implementation**: Alpine + ghostscript
**Use case**: Some upload sites require specific PDF versions

### 🔧 System & Infrastructure Tools (Middle Generation)

#### stress
**Location**: `stress/`
**Purpose**: CPU, memory, and I/O stress testing container
**Implementation**: Sophisticated bash script with GPT-4o attribution
**Features**:
- Auto-detects physical CPU cores (lscpu, /proc/cpuinfo, sysctl)
- Auto-detects available memory
- Three modes: heavy (1x), medium (0.5x), light (0.25x)
- Configurable duration via `DURATION` env var
- Optional I/O thread testing via `IO_THREADS` env var
- Smart worker calculation based on available resources

#### unraid-util
**Location**: `unraid-util/`
**Purpose**: Diagnostic container for Unraid NAS systems
**Implementation**: Ubuntu-based sleep-forever container
**Use case**: Provides tools like tcpdump that aren't available in Unraid base OS
**Usage**: `docker exec -it util bash` to access diagnostic tools
**Philosophy**: Ad-hoc package installation with `apt install`

### 📦 Backup Services (Middle-Modern Generation)

#### onedrive-backup
**Location**: `onedrive-backup/`
**Purpose**: Continuous OneDrive backup to local storage using rclone
**Implementation**: rclone base image + ChatGPT-written bash script
**Features**:
- Hourly check with configurable `BACKUP_INTERVAL`
- Incremental backups (copies from last backup before syncing)
- One-year retention by default
- Requires rclone.conf with OneDrive remote named "onedrive"
**Philosophy**: Defense against human error and cloud provider failures; "Storage is cheap, family memories are not"

#### backup-photos-to-gdrive
**Location**: `backup-photos-to-gdrive/`
**Purpose**: Continuous local photos backup to Google Drive using rclone
**Implementation**: Modern, sophisticated bash script with extensive features
**Features**:
- Safe copy mode (default) vs. destructive sync mode
- Configurable intervals (default 6 hours)
- Built-in retry logic and error handling
- Configuration validation before starting
- Comprehensive logging with timestamps
- Multiple environment variables for flexibility
**Quality**: Production-ready with extensive documentation and proper error handling

### 🔍 Helper Functions (Profile-Only)

#### get_docker_pids
**Purpose**: Map Docker container IDs/names to host PIDs and UIDs
**Implementation**: Parses `docker ps`, inspects containers, reads `/proc/$pid/status`
**Output**: Container name, UID, username, PID
**Use case**: Troubleshooting Docker container processes on host

#### network_blip
**Purpose**: Diagnostic logging for network connectivity issues
**Implementation**: OS-aware (macOS vs Linux) network diagnostics
**Actions**:
- Logs to `/tmp/network_blips.log`
- Captures gateway, interfaces, routing tables, ARP cache
- Pings Google DNS (8.8.8.8) and default gateway
- Works on both macOS (route, ifconfig, netstat) and Linux (ip)
**Use case**: Debugging intermittent network issues with timestamped diagnostics

#### unrar
**Purpose**: Extract RAR archives using Docker
**Implementation**: Uses maxcnunes/unrar Docker image
**Pattern**: `unrar e -r` (extract recursively)

### 🌐 Network Mapper (Modern Generation - Primary Utility)

**Location**: `network-mapper/`
**Language**: Go
**Status**: This is the PRIMARY utility in the repository

**Features**:
- Cross-platform network discovery (Linux, macOS, Windows)
- Automatic subnet and gateway detection
- Port scanning and device fingerprinting
- mDNS/Bonjour discovery (Apple ecosystem)
- SSDP/UPnP discovery (media servers, IoT)
- DHCP lease table scanning
- MAC vendor identification via IEEE OUI database
- Beautiful CLI visualization with ASCII art
- Multiple installation methods (Homebrew, Chocolatey, Docker, binary releases)

**Architecture**:
- Professional Go codebase with proper package structure
- Multiple specialized files: `agent.go`, `device_detector.go`, `dhcp_scanner.go`, `dns_resolver.go`, `gateway.go`, `intelligent_discovery.go`, `mac_vendor.go`, `network_expansion.go`, `ping.go`, `scanner.go`, `service_discovery.go`, `visualizer.go`
- YAML-based device rules: `device_rules.yaml`
- Devcontainer setup for development
- Git hooks for code quality
- Comprehensive CI/CD with GitHub Actions

**Documentation**:
- Extensive README with badges for build status, releases, package managers
- `BUILD_TRANSPARENCY.md` for addressing antivirus false positives
- `PACKAGE_SETUP.md` for maintainer documentation
- `example_usage.md` for user guidance
- Project-specific `CLAUDE.md` with AI assistant guidelines

**CI/CD Pipeline**:
- Multi-platform builds (Linux amd64/arm64, macOS, Windows)
- Automated releases with GitHub Releases
- Package manager automation (Homebrew tap, Chocolatey)
- Docker multi-arch images (GHCR)
- Smoke testing and release validation
- Automated package updates

**Philosophy**: "A lightweight home network security audit tool - not a high-power security tool for finding hidden subnets, but helping home users understand their environment"

## Coding Style & Patterns

### Shell Scripting
- **Shebang**: Uses `#!/bin/sh` for POSIX compatibility or `#!/bin/bash` when bash-specific features are needed
- **Error handling**: Modern scripts use `set -e` for fail-fast behavior
- **Configurability**: Environment variables with sensible defaults (e.g., `MODE="${MODE:-heavy}"`)
- **Documentation**: Inline comments crediting sources (Stack Overflow, GPT-4o)
- **Output**: Echo statements for user feedback during execution
- **Safety**: Platform detection (`uname`, conditional logic) for cross-platform scripts

### Docker Patterns

#### Early Generation (Simple Tools)
```dockerfile
FROM alpine:3.16.2
RUN apk add tool-package
COPY script.sh /usr/local/bin/tool-name
```
- Minimal Alpine base (specific version pinning)
- Single package installation
- Script copied to `/usr/local/bin/` for PATH access
- No ENTRYPOINT - invoked explicitly from shell function

#### Profile Function Pattern
```bash
function tool_name() {
    docker run --rm -it \
        --volume=$(pwd):/content/ \
        --workdir=/content/ \
        --network=none \
        container-name command "$1"
}
```
- `--rm` for automatic cleanup
- `-it` for interactive terminal
- Volume mount current directory
- Set workdir to mounted volume
- **`--network=none` for security isolation**
- Pass first argument to container command

#### Modern Generation (Services)
```dockerfile
FROM base-image
RUN apt-get update && apt-get install -y packages
COPY script.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```
- More sophisticated base images (Ubuntu for unraid-util)
- Scripts become entrypoints
- Designed to run continuously with sleep loops
- Environment variable configuration

### Go Programming (network-mapper)
- **Standard practices**: Proper package structure, separated concerns
- **CLI framework**: Likely using cobra or similar
- **Error handling**: Go idiomatic error returns
- **Cross-platform**: Build tags or conditional compilation for OS-specific code
- **Testing**: Test files alongside implementation
- **Dependencies**: Go modules with `go.mod` and `go.sum`

### Documentation Style
- **Markdown everywhere**: README.md for all major components
- **Emoji headers**: Makes documentation more approachable (🌐, 🎬, 🔧, etc.)
- **Clear hierarchy**: Overview → Installation → Usage → Details
- **Why sections**: Explains rationale and philosophy
- **Security notes**: Transparent about Docker image provenance, security measures
- **Attribution**: Credits tools, contributors, and AI assistance

### Security Practices
- **Transparent build process**: GitHub Actions workflows are public
- **Network isolation**: `--network=none` on utilities that don't need network
- **Minimal permissions**: Read-only where possible
- **Image provenance**: Documents Docker Hub publishing pipeline
- **Account security**: Documents use of FIDO2/WebAuthn for GitHub access
- **Dependency pinning**: Specific Alpine versions, Go dependency locking

### GitHub Actions Patterns
- **Multi-stage workflows**: Separate build, test, release, validation
- **Multi-platform builds**: Uses Docker buildx for amd64/arm64
- **Conditional execution**: Release workflows trigger on release publish
- **Secrets management**: Uses GitHub Secrets for Docker Hub, package managers
- **Validation steps**: Smoke tests after releases
- **Automated updates**: Package manager update workflows

## Release and Tagging Strategy

### The Monorepo Challenge
This repository is a **monorepo** containing multiple independent utilities. Each utility has its own version lifecycle, release cadence, and potentially different maintainers. This creates a challenge for tagging and releases that traditional single-project repositories don't face.

### Historical Context
**All existing tags from v1.0 through v2.8.3 refer exclusively to network-mapper releases.** This includes:
- Early tags without 'v' prefix: `1.0`, `1.1`, `1.1.1`, `1.2.1`-`1.2.6`
- Modern tags with 'v' prefix: `v1.2.1`, `v1.3.0`, `v1.4.0`, `v1.5.0`, `v1.5.1`, `v2.0.0`-`v2.8.3`

The tag `v1.0.0-smart-crop` was an attempt to version the smart-crop-video utility, but this approach has problems:
- It still triggered network-mapper build workflows (matching the `v*` pattern)
- The suffix approach is ambiguous and doesn't follow standard conventions
- Version numbers would conflict across utilities (both utilities could want v1.0.0)

### Official Tagging Convention

**Going forward, ALL releases MUST use utility-prefixed tags:**

```
<utility-name>-v<version>
```

#### Examples:
```
network-mapper-v2.9.0
smart-crop-video-v1.0.0
backup-photos-to-gdrive-v1.0.0
onedrive-backup-v1.0.0
stress-v1.0.0
```

#### Rules:
1. **Utility name prefix**: Use the exact directory name from the repository
2. **Dash separator**: Always use `-` between utility name and version
3. **Version format**: `v<major>.<minor>.<patch>` following semantic versioning
4. **No exceptions**: Even if a utility is the "primary" utility, it must use the prefix

### Creating a Release

When you're ready to release a utility, follow these steps:

#### 1. Update Version References
Update version numbers in relevant files for the utility (e.g., `go.mod`, documentation, package files)

#### 2. Create the Tag Locally
```bash
# Example for network-mapper v2.9.0
git tag network-mapper-v2.9.0
git push origin network-mapper-v2.9.0
```

#### 3. The Tag Triggers CI/CD
The `.github/workflows/release.yml` workflow will:
- Detect which utility is being released based on the tag prefix
- Build binaries for that specific utility
- Create a GitHub Release
- Upload release artifacts
- Trigger package manager updates (if applicable)

#### 4. Verify the Release
After the tag is pushed:
- Check GitHub Actions to ensure workflows completed successfully
- Verify the GitHub Release was created with correct artifacts
- For network-mapper: confirm Homebrew and Chocolatey packages were updated
- For Docker-based utilities: confirm images were published to Docker Hub

### Workflow Implications

#### Current Workflows That Need Updates
As of this writing, the following workflows need to be updated to properly handle utility-prefixed tags:

1. **`.github/workflows/release.yml`**:
   - Currently triggers on `v*` tags (matches ALL version tags)
   - Should parse the tag prefix to determine which utility to build
   - Example: `network-mapper-v2.9.0` should only build network-mapper

2. **`.github/workflows/publish.yml`**:
   - Currently builds ALL utilities' Docker images for ANY release
   - Should be split into utility-specific workflows, or add conditional logic based on tag prefix
   - Example: `smart-crop-video-v1.0.0` should only build smart-crop-video Docker image

3. **`.github/workflows/update-packages.yml`**:
   - Already has detection logic to check for network-mapper assets
   - Should additionally check the tag prefix for clarity
   - Only network-mapper should trigger Homebrew/Chocolatey updates

#### Recommended Workflow Structure
For maximum clarity, consider creating utility-specific workflows:
- `.github/workflows/release-network-mapper.yml` (triggers on `network-mapper-v*`)
- `.github/workflows/release-smart-crop-video.yml` (triggers on `smart-crop-video-v*`)
- `.github/workflows/release-backup-photos.yml` (triggers on `backup-photos-to-gdrive-v*`)

Alternatively, use a single workflow with conditional steps based on tag prefix parsing.

### Handling the Historical network-mapper Tags

**Do NOT rename or delete existing tags.** They are part of the release history and users may depend on them. Instead:

1. **Document the transition**: This section serves as that documentation
2. **Continue from current version**: The next network-mapper release should be `network-mapper-v2.9.0` (or whatever follows v2.8.3)
3. **Update documentation**: README files should reference the new tag format
4. **Maintain compatibility**: Keep existing GitHub releases and their download URLs intact

### For Contributors and AI Assistants

When creating a release:
1. **Always use the utility-prefixed format**: Never create tags like `v1.0.0` without a utility prefix
2. **Check existing versions**: Look at existing tags for the specific utility to determine the next version number
3. **Update relevant workflows**: Ensure CI/CD workflows will properly handle the new tag
4. **Test before tagging**: Verify builds work locally before creating the tag
5. **Verify after release**: Always check that workflows completed successfully
6. **Don't batch releases**: Release one utility at a time to avoid confusion

### Querying Tags by Utility

To see all tags for a specific utility:
```bash
# List all network-mapper releases
git tag -l 'network-mapper-v*'

# List all smart-crop-video releases
git tag -l 'smart-crop-video-v*'

# List all tags (sorted by version)
git tag --list --sort=-version:refname
```

### Version Number Independence

Each utility maintains its own version numbers independently:
- `network-mapper-v2.9.0` can coexist with `smart-crop-video-v1.0.0`
- Version numbers have no relation between utilities
- Breaking changes in one utility don't affect others
- Each utility follows semantic versioning for its own scope

## Installation Philosophy

### For Early Generation Utilities (Profile-Based)
The `profile` file is a **self-contained installation artifact**. Users can:
1. Clone the repo and `cat profile >> ~/.zshrc`
2. Or just copy-paste the profile contents manually
3. Reload shell or source the profile
4. All functions immediately available

**No additional files required** - the profile file references Docker Hub images that are pre-built by CI/CD.

### For Modern Utilities (Package Managers)
`network-mapper` uses **professional distribution channels**:
- **Homebrew**: `brew install nickborgers/tap/network-mapper`
- **Chocolatey**: `choco install network-mapper`
- **Docker**: `docker run ghcr.io/nickborgers/network-mapper:latest`
- **Binary**: Download from GitHub Releases

## Key Insights

### Repository Philosophy
1. **Portability over complexity**: Tools should follow you across jobs/machines
2. **Security by isolation**: Docker + network-none prevents data leaks
3. **Zero dependency installation**: Just Docker + shell profile
4. **Self-documenting**: README files explain why and how

### Evolution Pattern
The repository shows **increasing sophistication**:
- **2020s early**: Simple ffmpeg/imagemagick wrappers
- **Mid**: Service containers with loop-based monitoring
- **Recent**: Full CI/CD, package managers, professional Go applications

### Technical Debt Awareness
- Early utilities lack versioning beyond Docker tags
- Shell functions in profile don't have `--help` flags
- No automated testing for shell functions
- Docker image security scanning not visible (if present)

### Modern Best Practices (network-mapper)
- Proper semantic versioning
- Automated releases
- Multiple install methods
- Comprehensive documentation
- Devcontainer for contributor onboarding
- Build transparency documentation
- CI/CD with validation

## Project-Specific Guidance

### network-mapper
- Build and test using devcontainer configuration in its folder
- After cutting a release, **always confirm pipelines worked and release was successful**
- This is a home network tool, not a high-power security scanner
- Goal: help users understand their environment, not find deliberately hidden subnets

### General Development
- Docker must be installed and accessible
- For shell utilities: test on both Linux and macOS (Windows less critical for profile-based tools)
- Always consider security implications of network access
- Document the "why" not just the "how"

## Attribution & Credits
- mov-to-gif credits Stack Overflow users alexey-kozhevnikov and pleasestand
- stress script credits GPT-4o
- onedrive-backup script from ChatGPT
- Michael Jarvis credited for the original inspiration of portable shell environments

## Future Considerations
- Could add `--help` to shell functions
- Consider versioning for Docker images beyond just `latest`
- Automated security scanning in CI/CD
- Testing framework for shell functions
- Consider migrating remaining utilities to modern approach with proper releases
