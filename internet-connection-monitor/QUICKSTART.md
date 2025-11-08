# Internet Connection Monitor - Quick Start

## One-Command Demo

```bash
make grafana-dashboard-demo
```

That's it! This single command will:
- ✅ Start Elasticsearch
- ✅ Start Grafana
- ✅ Start Prometheus
- ✅ Start the Internet Connection Monitor
- ✅ Configure everything automatically
- ✅ Begin testing websites immediately

## Access the Dashboard

After 30-60 seconds, visit:

**Grafana Dashboard**: http://localhost:3000
- Username: `admin`
- Password: `admin`

The dashboard will be automatically imported and ready to view!

## What You'll See

The dashboard shows:
1. **Success Rate** - Is the Internet working?
2. **Latency Metrics** - How fast are page loads?
3. **Time-Series Charts** - Trends over time
4. **Per-Site Comparison** - Which sites are slow?
5. **Error Analysis** - What's failing and why?
6. **Recent Failures** - Detailed error logs

## Watch It Work

### View Test Results in Real-Time
```bash
make monitor-logs
```

You'll see JSON output like:
```json
{
  "@timestamp": "2025-01-08T15:23:45.123Z",
  "site": {"name": "google", "url": "https://www.google.com"},
  "status": {"success": true, "http_status": 200},
  "timings": {
    "dns_lookup_ms": 12,
    "tcp_connection_ms": 45,
    "tls_handshake_ms": 89,
    "total_duration_ms": 1456
  }
}
```

### Check Service Health
```bash
make demo-status
```

### View Elasticsearch Data
```bash
make es-indices      # List indices
make es-count        # Count test results
make es-sample       # View a sample result
```

## Common Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make demo-status` | Check if services are running |
| `make demo-logs` | View logs from all services |
| `make demo-stop` | Stop the demo (keeps data) |
| `make demo-restart` | Restart everything |
| `make demo-clean` | Stop and delete all data |

## Customize Sites to Monitor

Edit the docker-compose file or create a `.env` file:

```bash
# deployments/.env
SITES=google.com,github.com,cloudflare.com,yoursite.com
```

Then restart:
```bash
make demo-restart
```

## Troubleshooting

### Services not starting?
```bash
# Check status
make demo-status

# View logs for errors
make demo-logs
```

### Port conflicts?

The monitor uses **host networking** for accurate DNS measurements, which means it binds directly to ports on your host:
- Port 161/udp (SNMP)
- Port 9090 (Prometheus)
- Port 8080 (health check)

If these ports are already in use:
1. **Change the monitor's ports**: Edit environment variables `SNMP_PORT` and `PROM_PORT` in the docker-compose file
2. **Or use bridge networking**: See [deployments/README.md](deployments/README.md#alternative-bridge-networking) (note: reduces DNS measurement accuracy)

For Grafana/Elasticsearch/Prometheus port conflicts (3000, 9091, 9200), edit the port mappings in `deployments/docker-compose.with-stack.yml`.

### No data in Grafana?
```bash
# Verify data is being collected
make es-count

# Wait a few minutes for initial data
# The monitor tests continuously, so data accumulates quickly
```

### Why host networking?

The monitor measures DNS resolution time as a key metric. Host networking ensures these measurements reflect real user experience, not Docker's cached DNS responses. See [deployments/README.md](deployments/README.md#important-host-networking-for-accurate-dns-measurements) for full details.

## Stop the Demo

```bash
# Keep data for next time
make demo-stop

# Remove everything (clean slate)
make demo-clean
```

## Next Steps

- **Production Deployment**: See [deployments/README.md](deployments/README.md)
- **Architecture Details**: See [DESIGN.md](DESIGN.md)
- **Elasticsearch Integration**: See [ELASTICSEARCH_AND_GRAFANA.md](ELASTICSEARCH_AND_GRAFANA.md)

## Advanced Usage

### Run with Your Own Elasticsearch

```bash
# Edit deployments/.env
ES_ENABLED=true
ES_ENDPOINT=http://your-elasticsearch:9200
ES_USERNAME=your-user
ES_PASSWORD=your-password

# Use standalone mode
make run
```

### Monitor Different Sites

```bash
# Override environment variable
SITES="facebook.com,twitter.com,reddit.com" make grafana-dashboard-demo
```

### Export Dashboard

Once running, export your customized dashboard from Grafana:
1. Go to Dashboard Settings (gear icon)
2. Click "JSON Model"
3. Copy the JSON
4. Save to a file for version control

## Screenshots

### Success Rate Over Time
Shows if Internet connectivity is stable or has dips.

### Latency Percentiles (P50, P95, P99)
Identifies performance degradation and outliers.

### Error Distribution
Categorizes failures: timeouts, DNS issues, connection refused, etc.

### Per-Site Comparison
Quickly identify which sites are problematic.

## Learn More

- **Makefile Commands**: Run `make help`
- **Design Philosophy**: Read [DESIGN.md](DESIGN.md)
- **Full Documentation**: Read [README.md](README.md)

---

**That's it! One command gets you a full monitoring dashboard.** 🚀
