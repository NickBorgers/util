# Internet Connection Monitor

> Real-world Internet connectivity monitoring from a user's perspective

## 🚀 Quick Start

**Try the full Grafana dashboard demo in one command:**

```bash
make grafana-dashboard-demo
```

Then visit http://localhost:3000 (admin/admin) to see the dashboard!

See [QUICKSTART.md](QUICKSTART.md) for details.

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

✅ **Real browser testing** - Uses headless Chrome via CDP
✅ **Detailed timing metrics** - DNS, TCP, TLS, TTFB, DOM loaded, network idle
✅ **Continuous monitoring** - Serial testing, natural traffic patterns
✅ **Multiple outputs** - Logs, Elasticsearch, Prometheus, SNMP
✅ **Stateless design** - Restart-safe, no data loss
✅ **Beautiful Grafana dashboards** - Pre-built, ready to import
✅ **Zero configuration start** - Works out of the box

## Documentation

- **[QUICKSTART.md](QUICKSTART.md)** - Get started in 30 seconds
- **[DESIGN.md](DESIGN.md)** - Architecture, technology stack, implementation details
- **[ELASTICSEARCH_AND_GRAFANA.md](ELASTICSEARCH_AND_GRAFANA.md)** - JSON schema, Grafana queries, dashboard guide
- **[deployments/README.md](deployments/README.md)** - Deployment guide, configuration options, troubleshooting
- **[Makefile](Makefile)** - Run `make help` to see all commands

## Common Commands

```bash
# Start the full demo stack
make grafana-dashboard-demo

# Check status
make demo-status

# View test results
make monitor-logs

# View sample data from Elasticsearch
make es-sample

# Stop demo (keeps data)
make demo-stop

# Clean up everything
make demo-clean
```

See `make help` for all 40+ commands.

## Configuration

Default sites tested: google.com, github.com, cloudflare.com, wikipedia.org, example.com

To customize:
```bash
# Edit deployments/.env
SITES=yoursite.com,google.com,github.com
```

See [deployments/.env.example](deployments/.env.example) for all options.

## Example Output

```json
{
  "@timestamp": "2025-01-08T15:23:45.123Z",
  "site": {
    "url": "https://www.google.com",
    "name": "google"
  },
  "status": {
    "success": true,
    "http_status": 200
  },
  "timings": {
    "dns_lookup_ms": 12,
    "tcp_connection_ms": 45,
    "tls_handshake_ms": 89,
    "time_to_first_byte_ms": 156,
    "dom_content_loaded_ms": 432,
    "network_idle_ms": 1234,
    "total_duration_ms": 1456
  }
}
```

## Requirements

- Docker
- Docker Compose
- (Optional) Make for convenient commands

## License

See [LICENSE](../LICENSE) in the repository root.