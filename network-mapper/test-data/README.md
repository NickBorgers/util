# Integration Test Data

This directory contains test fixtures for the network-mapper integration tests.

## Files

- `nginx-home.conf` - Nginx configuration for home network test service (192.168.1.10)
- `nginx-corp.conf` - Nginx configuration for corporate network test service (10.10.0.50)
- `run-integration-test.sh` - Helper script for running integration tests

## Usage

These files are used by the Docker Compose integration test setup in `docker-compose.test.yml`.

### Quick Start

To run the integration tests with the helper script:

```bash
cd /workspaces/monorepo/network-mapper
./test-data/run-integration-test.sh
```

This script will:
1. Build the network-mapper Docker image
2. Start test networks and services
3. Verify all services are responding
4. Run network-mapper discovery
5. Clean up test environment

### Manual Testing

Alternatively, run Docker Compose directly:

```bash
docker-compose -f docker-compose.test.yml up --build
```

### What It Tests

The test creates three isolated networks and verifies that network-mapper can discover cross-network subnets using intelligent discovery mode:
- **Home network** (192.168.1.0/24) - with HTTP and SSDP services
- **Corporate network** (10.10.0.0/24) - with HTTP API service
- **Alternative network** (192.168.0.0/24) - with HTTP service
