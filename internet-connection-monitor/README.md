# Internet Connection Monitor

> Real-world Internet connectivity monitoring from a user's perspective

## User Story
As an engineer who runs an arguably over-complicated home network, I have device-level monitoring but occasionally we see perceptible network problems that do not show up in the monitoring. The ultimate signal of "is Internet working as expected" is how it behaves for us meatsacks. The monitoring approaches I'm already using:
* Suricata netflows -> Elasticsearch -> Grafana dashboard
* Network devices -- SNMP --> Zabbix --> ntfy.sh
* Uptime Kuma

Are good things, and useful to diagnose many problems. However they are not the test I ultimately run if someone tells me there's "a problem with the wifi": I will open a browser and try to load sites and see how it behaves.

Let's make that the test.

## Design
This utility is a container (e.g. Docker) you can deploy into your environment which will run a web browser and navigate to pages. It will make itself monitorable in three ways:
* Logs
    * JSON documents will describe what succeeded and failed and how long it took, for comparison against previous and future tests
* API push
    * Those same JSON documents will be pushed to ElasticSearch, where they can later be rendered such as with Grafana dashboard or even alerted upon if they drift
* Polling
    * Zabbix has some solid alerting features and is good at SNMP polling, so let's offer an SNMP interface which is easy to setup for alerting in Zabbix. I already get alerts for an Ethernet port's link going down on my switches from Zabbix, why not this Internet monitor?
    * The data offered in JSON documents should be readily scrapable, a sort of Internet connection version of Prometheus for a host

## Design Axioms
* No-dependency container; you should be able to spin this up basically anywhere quickly
    * With no configuration it should start test a configured list of web sites and generating logs, and offering last report for scrape/poll
    * If you add Elasticsearch configuration, it pushes to Elasticsearch

## 🚀 Quick Start

**Run and watch live results:**

```bash
make quick-start    # Build & run
# Ctrl+C to stop viewing
make quick-stop     # Stop the monitor
```

**Full Grafana dashboard demo:**

```bash
make grafana-dashboard-demo
```

Then visit [http://localhost:3000](http://127.0.0.1:3000/d/internet-conn-mon/internet-connection-monitor?orgId=1&from=now-15m&to=now&timezone=browser&var-site_name=$__all&refresh=1m) (admin/admin) to see the dashboard!

> **🔒 Security Note**: All demo services (Grafana, Elasticsearch, Prometheus) bind to `127.0.0.1` only, preventing network exposure when running for extended periods.

See [QUICKSTART.md](QUICKSTART.md) for details or run `make help` for all commands.

## Dashboard Preview

![Grafana Dashboard](dashboard.png)

The dashboard provides real-time monitoring with:
- Success/failure rates over time (stacked area chart)
- Average and P95 latency metrics
- Clear visualization of monitoring gaps vs actual failures
- Site-by-site success stats

## How It Works

The monitor runs **continuously**, testing sites one at a time (like a real person browsing):

1. Load google.com in headless browser → Measure timings → Emit results
2. Small delay (1-5 seconds)
3. Load github.com in headless browser → Measure timings → Emit results
4. Small delay
5. Load cloudflare.com...
6. Repeat forever

**No traffic spikes** - just steady, natural browsing patterns. Results are emitted to:
- **JSON logs** (stdout) - Always on, zero config
- **Elasticsearch** - Optional, for Grafana dashboards and long-term analysis
- **Prometheus** - Optional, for metrics scraping
- **SNMP** - Optional, for Zabbix polling

## Features

* ✅ **Real browser testing** - Uses headless Chrome via CDP
* ✅ **Comprehensive timing metrics** - DNS, TCP, TLS, TTFB, DOM loaded, page load, network idle
* ✅ **Network failure phase detection** - Pinpoints which layer failed (DNS, TCP, TLS, HTTP) using timing analysis
* ✅ **Chrome error code capture** - Reports specific errors (ERR_NAME_NOT_RESOLVED, ERR_CONNECTION_REFUSED, etc.)
* ✅ **Accurate DNS measurements** - Uses host networking to measure real DNS resolution times (not Docker's cached DNS)
* ✅ **Continuous monitoring** - Serial testing, natural traffic patterns
* ✅ **Multiple outputs** - Logs, Elasticsearch, Prometheus, SNMP (SNMP not tested)
* ✅ **Stateless design** - Restart-safe, no data loss
* ✅ **Beautiful Grafana dashboards** - Pre-built, ready to import
* ✅ **Zero configuration start** - Works out of the box
* ✅ **Integration tested** - Core data flow validated: monitor → Elasticsearch → Grafana, Prometheus

**Networking Architecture**: The monitor uses **host networking mode** to ensure DNS resolution times match real user experience. With Docker's default bridge networking, Docker's embedded DNS proxy can cache responses and skew measurements. Host networking provides direct access to your network's DNS servers for accurate timing data. See [deployments/README.md](deployments/README.md) for details and alternatives.

## Documentation

- **[QUICKSTART.md](QUICKSTART.md)** - Get started in 30 seconds
- **[DESIGN.md](DESIGN.md)** - Architecture, technology stack, implementation details
- **[ELASTICSEARCH_AND_GRAFANA.md](ELASTICSEARCH_AND_GRAFANA.md)** - JSON schema, Grafana queries, dashboard guide
- **[TESTING.md](TESTING.md)** - Test coverage, known gaps, and testing philosophy
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Common issues and solutions
- **[deployments/README.md](deployments/README.md)** - Deployment guide, configuration options
- **[Makefile](Makefile)** - Run `make help` to see all commands

## Common Commands

```bash
# Quick test (30 seconds)
make quick-test

# Run tests
make test                   # Unit tests
make test-coverage          # Unit tests with coverage
make test-integration       # Full end-to-end test suite
make test-all               # All tests (unit + integration)

# Run and watch live
make quick-start            # Build & run
make quick-stop             # Stop

# Full demo stack
make grafana-dashboard-demo # Start everything
make demo-status            # Check health
make monitor-logs           # View test results
make demo-stop              # Stop (keeps data)

# Local development
make build-binary           # Build Go binary
make dev                    # Build & run locally

# Watch formatted output
make watch-json             # Pretty-print results
```

See `make help` for all 50+ commands and workflows.

## Configuration

Default sites tested: google.com, github.com, cloudflare.com, wikipedia.org, example.com

To customize:
```bash
# Edit deployments/.env
SITES=yoursite.com,google.com,github.com
```

See [deployments/.env.example](deployments/.env.example) for all options.

## Example Output

**Successful test:**
```json
{
  "@timestamp": "2025-01-08T15:23:45.123Z",
  "test_id": "550e8400-e29b-41d4-a716-446655440000",
  "site": {
    "url": "https://www.google.com",
    "name": "google"
  },
  "status": {
    "success": true,
    "http_status": 200,
    "message": "Page loaded successfully"
  },
  "timings": {
    "dns_lookup_ms": 12,
    "tcp_connection_ms": 45,
    "tls_handshake_ms": 89,
    "time_to_first_byte_ms": 156,
    "dom_content_loaded_ms": 432,
    "network_idle_ms": 1234,
    "total_duration_ms": 1456
  },
  "metadata": {
    "hostname": "monitor-01",
    "version": "1.3.0"
  }
}
```

**Failed test with phase detection:**
```json
{
  "@timestamp": "2025-11-11T14:28:40.859Z",
  "test_id": "660e8400-e29b-41d4-a716-446655440001",
  "site": {
    "url": "https://this-does-not-exist-12345.com",
    "name": "test-site"
  },
  "status": {
    "success": false,
    "message": "Failed to load page"
  },
  "error": {
    "error_type": "ERR_NAME_NOT_RESOLVED",
    "error_message": "page load error net::ERR_NAME_NOT_RESOLVED",
    "failure_phase": "dns"
  },
  "timings": {
    "dns_lookup_ms": null,
    "tcp_connection_ms": null,
    "tls_handshake_ms": null,
    "total_duration_ms": 10393
  },
  "metadata": {
    "hostname": "monitor-01",
    "version": "1.3.0"
  }
}
```

The `failure_phase` field indicates which network layer failed:
- **dns** - DNS resolution failed (hostname couldn't be resolved)
- **tcp** - TCP connection failed (DNS succeeded, but couldn't connect)
- **tls** - TLS handshake failed (connection established, but TLS negotiation failed)
- **http** - HTTP request failed (all connections succeeded, but HTTP request/response failed)
- **unknown** - Failure phase couldn't be determined

## Installation

### Option 1: Use Pre-built Docker Image (Recommended)

Pull the latest release from GitHub Container Registry:

```bash
docker pull ghcr.io/nickborgers/internet-connection-monitor:latest

# Or pull a specific version
docker pull ghcr.io/nickborgers/internet-connection-monitor:1.0.0
```

Then update your `docker-compose.yml` to use the GHCR image:

```yaml
services:
  internet-monitor:
    image: ghcr.io/nickborgers/internet-connection-monitor:latest
    # ... rest of your configuration
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/nickborgers/monorepo.git
cd monorepo/internet-connection-monitor

# Build the Docker image
docker build -t internet-connection-monitor:latest .

# Or use make
make build
```

## Requirements

- Docker
- Docker Compose
- (Optional) Make for convenient commands

## CI/CD

The project uses GitHub Actions for continuous integration:

- **Integration tests** run on every push and pull request
- **Docker images** are published to GitHub Container Registry on release
- **Release tags** follow the pattern: `internet-connection-monitor-v1.0.0`

See [.github/workflows/internet-connection-monitor-ci.yml](../.github/workflows/internet-connection-monitor-ci.yml)

## Testing

See [TESTING.md](TESTING.md) for detailed information about:
- Integration test coverage
- Known testing gaps
- How to run tests
- Contributing tests

## License

See [LICENSE](../LICENSE) in the repository root.