# Deployment Configurations

This directory contains deployment configurations for the Internet Connection Monitor.

## Important: Host Networking for Accurate DNS Measurements

**The Internet Connection Monitor uses host networking mode for maximum measurement accuracy.**

### Why Host Networking?

The monitor measures DNS resolution time as a key metric. With Docker's default bridge networking:
- Docker's embedded DNS proxy (at 127.0.0.11) sits between the container and your actual DNS servers
- This DNS proxy can cache responses, making measurements artificially fast
- The cached timings don't reflect what users actually experience on the network

With host networking:
- The container uses the host's DNS servers directly
- DNS resolution times match real user experience
- TCP/IP stack behavior is identical to the host system
- Measurements are realistic and actionable

### Implications of Host Networking

**Port binding:**
- The monitor binds directly to host ports (no port mapping needed)
- Default ports: 161/udp (SNMP), 9090 (Prometheus), 8080 (health check)
- Ensure these ports are available on your host

**Service connectivity:**
- When connecting to Elasticsearch/Prometheus on the same host, use `localhost:port`
- Example: `ES_ENDPOINT=http://localhost:9200` (not `http://elasticsearch:9200`)

**Security:**
- The container shares the host's network namespace (less isolation)
- The monitor runs as a non-root user for security
- Only uses outbound connections to test sites

**Compatibility:**
- Works on Linux, macOS, and Windows with Docker Desktop
- Some cloud platforms may restrict host networking (use bridge mode as fallback)

### Alternative: Bridge Networking

If you need bridge networking (e.g., cloud platform restrictions), you can configure custom DNS servers in `docker-compose.yml`:

```yaml
services:
  internet-monitor:
    # Remove network_mode: host
    dns:
      - 8.8.8.8      # Google DNS
      - 1.1.1.1      # Cloudflare DNS
    networks:
      - default
    ports:
      - "161:161/udp"
      - "9090:9090"
```

**Note:** This maintains some isolation but still has Docker's DNS proxy layer. For most home/enterprise deployments, host networking is recommended.

## Quick Start

### Option 1: Standalone Monitor (You Have Existing Infrastructure)

If you already have Elasticsearch, Grafana, and/or Prometheus running:

```bash
# Copy and customize environment file
cp .env.example .env
nano .env  # Edit to match your infrastructure

# Start just the monitor
docker-compose up -d

# View logs
docker-compose logs -f internet-monitor
```

The monitor will:
- Write JSON logs to stdout (viewable via Docker logs)
- Expose SNMP on port 161 (if enabled)
- Expose Prometheus metrics on port 9090 (if enabled)
- Push to Elasticsearch (if configured and enabled)

### Option 2: Full Stack (Everything Included)

For testing or if you don't have monitoring infrastructure:

```bash
# Start the full stack
docker-compose -f docker-compose.with-stack.yml up -d

# Wait for services to be ready (30-60 seconds)
docker-compose -f docker-compose.with-stack.yml ps

# Access Grafana
# URL: http://localhost:3000
# Username: admin
# Password: admin
```

This includes:
- Internet Connection Monitor
- Elasticsearch (single-node)
- Grafana with pre-configured dashboard
- Prometheus (optional)

## Configuration

### Environment Variables

See `.env.example` for all available options. Key settings:

#### Sites to Monitor
```bash
SITES=google.com,github.com,cloudflare.com,wikipedia.org
```

#### Elasticsearch
```bash
ES_ENABLED=true
ES_ENDPOINT=http://your-elasticsearch:9200
ES_USERNAME=monitor_user
ES_PASSWORD=secret_password
```

#### SNMP
```bash
SNMP_ENABLED=true
SNMP_COMMUNITY=public
SNMP_HOST_PORT=161  # Change if port 161 is already in use
```

#### Prometheus
```bash
PROM_ENABLED=true
PROM_HOST_PORT=9090  # Change if port 9090 is already in use
```

## Grafana Dashboard Import

### Automatic (Full Stack)

When using `docker-compose.with-stack.yml`, the dashboard is automatically provisioned.

### Manual Import

1. Copy the dashboard JSON:
   ```bash
   cp ../grafana-dashboard.json ./grafana-provisioning/dashboards/json/
   ```

2. Or import via Grafana UI:
   - Navigate to Grafana → Dashboards → Import
   - Upload `../grafana-dashboard.json`
   - Select the Elasticsearch datasource
   - Click Import

## Elasticsearch Setup

### Automatic (Full Stack)

When using `docker-compose.with-stack.yml`, the index template and ILM policy are automatically created.

### Manual Setup

If using an existing Elasticsearch cluster:

```bash
# Create ILM policy for data retention (90 days)
curl -X PUT "http://your-elasticsearch:9200/_ilm/policy/internet-monitor-ilm-policy" \
  -H 'Content-Type: application/json' \
  -d @../elasticsearch-ilm-policy.json

# Create index template
curl -X PUT "http://your-elasticsearch:9200/_index_template/internet-connection-monitor" \
  -H 'Content-Type: application/json' \
  -d @../elasticsearch-index-template.json
```

### Verify Setup

```bash
# Check if ILM policy exists
curl "http://localhost:9200/_ilm/policy/internet-monitor-ilm-policy?pretty"

# Check if index template exists
curl "http://localhost:9200/_index_template/internet-connection-monitor?pretty"

# List indices (after monitor has been running)
curl "http://localhost:9200/_cat/indices/internet-connection-monitor-*?v"
```

## Prometheus Configuration

To add the monitor to an existing Prometheus instance, add this scrape config:

```yaml
scrape_configs:
  - job_name: 'internet-monitor'
    static_configs:
      - targets: ['internet-monitor:9090']
        labels:
          service: 'internet-connection-monitor'
```

## SNMP Monitoring with Zabbix

To monitor via Zabbix:

1. Add SNMP host in Zabbix:
   - Hostname: `internet-monitor`
   - SNMP Community: `public` (or your custom value)
   - Port: `161`

2. Create items for:
   - Last test timestamp
   - Success/failure status per site
   - Latency values

3. Configure triggers:
   - Alert if no tests for 5+ minutes
   - Alert if success rate < 95%
   - Alert if latency > threshold

## Port Conflicts

**With host networking (default):**

The monitor binds directly to host ports. If ports 161, 9090, or 8080 are already in use:

**Option 1: Change the application ports** (edit environment variables):
```bash
# In .env file or docker-compose.yml
SNMP_PORT=1161          # Change from default 161
PROM_PORT=9190          # Change from default 9090
# Health check runs on 8080 (less likely to conflict)
```

**Option 2: Switch to bridge networking** (see Alternative: Bridge Networking section above):
```yaml
# Remove network_mode: host
ports:
  - "1161:161/udp"      # Map host port 1161 to container port 161
  - "9190:9090"         # Map host port 9190 to container port 9090
```

**Note:** Option 2 reduces DNS measurement accuracy but provides more port flexibility.

## Verifying It's Working

### Check Logs
```bash
# Should see JSON test results
docker-compose logs -f internet-monitor

# Example output:
# {"@timestamp":"2025-01-08T15:23:45.123Z","site":{"name":"google"},"status":{"success":true},...}
```

### Check Prometheus Metrics
```bash
curl http://localhost:9090/metrics

# Should see metrics like:
# internet_monitor_test_success{site="google"} 1
# internet_monitor_latency_ms{site="google"} 1234
```

### Check Elasticsearch
```bash
# Count documents
curl "http://localhost:9200/internet-connection-monitor-*/_count?pretty"

# View recent test
curl "http://localhost:9200/internet-connection-monitor-*/_search?size=1&sort=@timestamp:desc&pretty"
```

### Check SNMP
```bash
# Requires snmpwalk tool
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1

# Or using Docker
docker run --rm --network host alpine/net-snmp \
  snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1
```

## Troubleshooting

### Monitor not starting
```bash
# Check logs for errors
docker-compose logs internet-monitor

# Common issues:
# - Elasticsearch not reachable (if ES_ENABLED=true)
# - Port conflicts (change host ports in .env)
# - Invalid site URLs in SITES variable
```

### No data in Elasticsearch
```bash
# Verify ES_ENABLED=true in environment
docker-compose exec internet-monitor env | grep ES_

# Check network connectivity
docker-compose exec internet-monitor ping -c 3 elasticsearch

# Check Elasticsearch logs
docker-compose logs elasticsearch
```

### Dashboard shows "No Data"
```bash
# Verify data exists
curl "http://localhost:9200/internet-connection-monitor-*/_count"

# Check datasource in Grafana
# - Go to Configuration → Data Sources
# - Test the Elasticsearch connection
# - Verify index pattern matches: internet-connection-monitor-*
```

### High resource usage
```bash
# Check container stats
docker stats internet-monitor

# If high:
# - Reduce number of sites in SITES
# - Increase INTER_TEST_DELAY
# - Check for DNS issues causing timeouts
```

## Data Retention

### Elasticsearch ILM Policy

Default lifecycle:
- **Hot** (0-7 days): Active writing, high priority
- **Warm** (7-30 days): Read-only, merged segments
- **Cold** (30-90 days): Compressed, low priority
- **Delete** (>90 days): Automatically deleted

To change retention period, edit `../elasticsearch-ilm-policy.json` and update:

```json
{
  "policy": {
    "phases": {
      "delete": {
        "min_age": "180d"  # Change from 90d to 180d for 6 months
      }
    }
  }
}
```

Then reapply the policy:
```bash
curl -X PUT "http://localhost:9200/_ilm/policy/internet-monitor-ilm-policy" \
  -H 'Content-Type: application/json' \
  -d @../elasticsearch-ilm-policy.json
```

### Prometheus Retention

Default: 90 days

To change, edit `docker-compose.with-stack.yml`:
```yaml
prometheus:
  command:
    - '--storage.tsdb.retention.time=180d'  # Change to 180 days
```

## Upgrading

### Monitor Container

**Using pre-built images from GHCR:**

First, update your `docker-compose.yml` to use the GHCR image:
```yaml
services:
  internet-monitor:
    image: ghcr.io/nickborgers/internet-connection-monitor:latest
```

Then upgrade:
```bash
# Pull latest image
docker-compose pull internet-monitor

# Recreate container
docker-compose up -d internet-monitor
```

**Using locally built images:**

```bash
# Rebuild the image
docker build -t internet-connection-monitor:latest ../

# Recreate container
docker-compose up -d internet-monitor
```

The monitor is stateless, so upgrades are safe and don't lose data.

### Elasticsearch/Grafana
```bash
# Update image version in docker-compose.with-stack.yml
# Then recreate:
docker-compose -f docker-compose.with-stack.yml up -d elasticsearch grafana
```

## Production Considerations

### Security
- [ ] Change Grafana admin password
- [ ] Enable Elasticsearch authentication (xpack.security)
- [ ] Use secrets management for passwords (Docker secrets, Vault)
- [ ] Configure SNMP v3 with encryption (instead of v2c community strings)
- [ ] Run on internal network only, expose via reverse proxy
- [ ] Enable TLS for Elasticsearch and Grafana

### Reliability
- [ ] Use managed Elasticsearch (AWS OpenSearch, Elastic Cloud)
- [ ] Configure Grafana with persistent storage
- [ ] Set up multiple monitor instances for redundancy
- [ ] Configure alerting for monitor failures
- [ ] Monitor the monitor (health checks, meta-metrics)

### Scale
- [ ] For large deployments, use Elasticsearch cluster
- [ ] Consider regional monitor instances
- [ ] Use Elasticsearch ILM for automatic index management
- [ ] Configure Grafana with separate database (PostgreSQL)

## Files in This Directory

- `docker-compose.yml` - Standalone monitor only
- `docker-compose.with-stack.yml` - Full stack with ES/Grafana/Prometheus
- `prometheus.yml` - Prometheus scrape configuration
- `.env.example` - Example environment variables
- `grafana-provisioning/` - Auto-provisioning for Grafana datasources and dashboards
- `README.md` - This file
