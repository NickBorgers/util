# Release Validation System

This document describes the automated validation system that ensures new releases meet end-user expectations.

## Overview

The release validation system consists of multiple workflows that test different aspects of the release:

1. **Smoke Tests** - Run on every commit to catch issues early
2. **Release Validation** - Comprehensive testing triggered on release publication
3. **Package Manager Validation** - Ensures distribution channels work correctly

## Validation Coverage

### 1. Binary Validation (`validate-binaries`)
- **Cross-platform testing**: Linux (x64/ARM64), macOS (Intel/Apple Silicon), Windows (x64)
- **Checksum verification**: Ensures download integrity
- **Basic functionality**: Version output, help text, core scanning features
- **Command-line interface**: Validates essential flags and options

### 2. Docker Validation (`validate-docker`)
- **Container functionality**: Pulls and tests Docker images
- **Version consistency**: Ensures Docker image matches release tag
- **Basic operations**: Help and version commands work in containerized environment

### 3. Output Format Validation (`validate-output-format`)
Tests that core user-facing output remains consistent:

**Expected Output Patterns:**
```
✅ Network Mapper v2.x.x
✅ Discovering network interfaces
✅ Scan mode: [mode description]
✅ Target ranges: [CIDR ranges]  ← New in v2.5.0
✅ Network Topology Map
✅ Discovery Summary
✅ Scan Complete
```

**Critical Error Detection:**
- Detects panics, fatal errors, or crashes
- Ensures scan completes successfully
- Validates new CIDR range output feature

### 4. Package Manager Validation (`validate-package-managers`)
- **Homebrew**: Verifies tap repository updates
- **Chocolatey**: Checks community gallery updates (with review delay tolerance)
- **Automation verification**: Ensures package manager workflows succeeded

### 5. Integration Testing (`integration-test`)
- **Multi-mode testing**: Tests all scan modes (quick, intelligent, brute-expanded)
- **End-to-end flows**: Complete scan operations with expected output
- **Cross-feature validation**: Ensures features work together correctly

## Smoke Testing (Development)

Runs on every commit to `main`/`develop`:

### Core Functionality Tests
- Build verification across platforms
- Basic command execution (version, help)
- Quick scan mode operation
- Output format validation

### Device Rules Validation
- YAML export functionality
- Syntax validation of exported rules
- Rules loading verification

### Performance Benchmarking
- Measures scan completion time
- Validates performance metrics reporting
- Detects potential performance regressions

## End-User Expectations

The validation system specifically tests these user expectations:

### ✅ **Installation Works**
- Downloads complete successfully
- Checksums verify correctly
- Binaries are executable
- Package managers provide current version

### ✅ **Basic Usage Works**
- `--version` shows correct version
- `--help` displays comprehensive help
- Quick scans complete successfully
- No crashes or panics

### ✅ **Output is Consistent**
- Familiar header and branding
- Clear scan progress indication
- Network topology visualization
- Summary statistics
- **NEW**: CIDR range transparency

### ✅ **Cross-Platform Compatibility**
- Works on Linux, macOS, Windows
- ARM64 and x64 architectures supported
- Docker images function correctly

### ✅ **Performance is Reasonable**
- Quick scans complete in under 45 seconds
- Performance metrics are reported
- No significant regressions

## Running Validation

### Automatic Validation
```yaml
# Triggers automatically on release
on:
  release:
    types: [published]
```

### Manual Validation
```bash
# Validate specific release
gh workflow run release-validation.yml -f release_tag=v2.5.0
```

### Development Testing
```bash
# Run smoke tests locally
cd network-mapper
go build -o network-mapper .
./network-mapper --version
./network-mapper --scan-mode quick --no-dns --no-services
```

## Validation Report

After each release, the system generates a validation report:

```markdown
# Release Validation Report for v2.5.0

## Validation Results
- Binary Validation: ✅ success
- Docker Validation: ✅ success
- Output Format: ✅ success
- Package Managers: ✅ success
- Integration Test: ✅ success

## ✅ Overall Status: PASSED
Release v2.5.0 meets all end-user expectations.
```

## Adding New Validations

When adding new features, update the validation system:

1. **Add output pattern checks** in `validate-output-format`
2. **Add functionality tests** in `integration-test`
3. **Update smoke tests** for development feedback
4. **Document user expectations** in this file

## Failure Response

If validation fails:

1. **Review the validation report** to identify specific failures
2. **Check GitHub Actions logs** for detailed error information
3. **Fix the issue** and create a new release if necessary
4. **Update validation tests** if user expectations have legitimately changed

This system ensures that every release maintains the quality and functionality that users expect from Network Mapper.