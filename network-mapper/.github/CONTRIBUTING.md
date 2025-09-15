# Contributing to Network Mapper

Thank you for your interest in contributing to Network Mapper! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker (optional, for container builds)
- Git

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-username/network-mapper.git
   cd network-mapper
   ```

2. **Using DevContainer (Recommended)**:
   ```bash
   # Start the devcontainer
   devcontainer up --workspace-folder .

   # Execute commands in the container
   devcontainer exec --workspace-folder . go run .
   ```

3. **Manual setup**:
   ```bash
   # Install dependencies
   go mod download

   # Run the application
   go run .

   # Run tests
   go test ./...
   ```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

## Code Quality

### Before Submitting

Ensure your code passes all checks:

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...
staticcheck ./...

# Run tests
go test -v ./...

# Check security
gosec ./...
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions and types
- Keep functions focused and small
- Use meaningful variable and function names

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for good test coverage

## CI/CD Pipeline

### Automated Workflows

The project uses GitHub Actions for CI/CD:

#### Build Workflow (`.github/workflows/build.yml`)

Triggered on:
- Push to `main` or `develop` branches
- Pull requests to `main`

Performs:
- Code quality checks (formatting, linting, security)
- Unit tests
- Cross-platform builds
- Artifact uploads

#### Release Workflow (`.github/workflows/release.yml`)

Triggered on:
- Git tags matching `v*` pattern

Performs:
- Cross-platform binary builds
- Docker image builds
- GitHub release creation
- Asset uploads with checksums

### Supported Platforms

Binaries are built for:
- Linux (x64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (x64)

### Docker Support

```bash
# Build Docker image
docker build -t network-mapper .

# Run in container
docker run --rm --network host network-mapper

# Using docker-compose
docker-compose up --build
```

## Release Process

### Creating a Release

1. **Update version** in relevant files
2. **Create and push a tag**:
   ```bash
   git tag -a v1.0.1 -m "Release v1.0.1"
   git push origin v1.0.1
   ```
3. **GitHub Actions** will automatically:
   - Build binaries for all platforms
   - Create GitHub release
   - Upload assets and checksums
   - Build and push Docker images

### Version Numbering

Follow semantic versioning (SemVer):
- `MAJOR.MINOR.PATCH`
- Major: Breaking changes
- Minor: New features, backward compatible
- Patch: Bug fixes, backward compatible

## Pull Request Guidelines

### Before Creating a PR

1. Create a feature branch from `develop`
2. Make your changes
3. Add tests for new functionality
4. Ensure all tests pass
5. Update documentation if needed

### PR Description

Include:
- Clear description of changes
- Link to related issues
- Testing performed
- Screenshots/examples if UI changes

### Review Process

1. Automated checks must pass
2. Code review by maintainers
3. Manual testing if needed
4. Merge to `develop` branch

## Issue Guidelines

### Bug Reports

Include:
- Operating system and version
- Go version (if building from source)
- Network Mapper version
- Steps to reproduce
- Expected vs actual behavior
- Relevant log output

### Feature Requests

Include:
- Clear description of the feature
- Use case and motivation
- Proposed implementation approach
- Alternatives considered

## Security

### Reporting Security Issues

Please report security vulnerabilities privately by emailing the maintainers. Do not create public issues for security vulnerabilities.

### Security Considerations

- Network Mapper performs network scanning - ensure responsible use
- Follow responsible disclosure for any security findings
- Test security changes thoroughly

## Architecture

### Project Structure

```
network-mapper/
├── main.go              # CLI entry point
├── scanner.go           # Core scanning logic
├── service_discovery.go # mDNS/SSDP discovery
├── mac_vendor.go        # MAC vendor lookup
├── ping.go              # Cross-platform ping
├── gateway.go           # Gateway discovery
├── dhcp_scanner.go      # DHCP lease scanning
├── visualizer.go        # CLI visualization
├── .github/             # CI/CD workflows
├── .devcontainer/       # Development container
└── build/               # Build artifacts
```

### Adding New Features

1. **Service Discovery**: Add new protocols in `service_discovery.go`
2. **Device Types**: Extend detection in `ping.go`
3. **Visualization**: Enhance display in `visualizer.go`
4. **Platform Support**: Add OS-specific code with build tags

## Community

### Communication

- GitHub Issues for bugs and features
- GitHub Discussions for questions and ideas
- Pull Requests for code contributions

### Code of Conduct

Be respectful, inclusive, and constructive in all interactions. This project follows the Contributor Covenant Code of Conduct.

## License

By contributing to Network Mapper, you agree that your contributions will be licensed under the same license as the project (MIT License).