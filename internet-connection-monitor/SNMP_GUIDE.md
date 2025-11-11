# SNMP Monitoring Guide

The Internet Connection Monitor includes a full-featured SNMP v2c agent for integration with network monitoring systems like Zabbix, Nagios, and others.

## Features

### 1. SNMP v2c Agent
- **Full protocol support**: Responds to GET, GETNEXT, and GETBULK requests
- **Complete OID tree**: Hierarchical structure following SNMP standards
- **Real-time data**: Queries return current statistics and cached test results
- **Community string authentication**: Simple, widely-supported authentication

### 2. HTTP API
For easier testing and debugging, the monitor exposes SNMP data via HTTP endpoints:
- `http://localhost:162/snmp/data` - JSON representation of all SNMP data
- `http://localhost:162/snmp/mib` - Text format MIB definition
- `http://localhost:162/snmp/oids` - List of all available OIDs

### 3. SNMP Traps
Automatic trap notifications for:
- **Test failures**: Sent when a site test fails
- **Service degradation**: Sent when failure rate exceeds 50%
- **Service recovery**: Sent when a degraded site recovers

### 4. Formal MIB Definition
Complete MIB file (`INTERNET-CONNECTION-MONITOR-MIB.txt`) for import into SNMP managers.

## Configuration

### Environment Variables

```bash
# Enable SNMP agent (default: true)
SNMP_ENABLED=true

# SNMP port (default: 161)
SNMP_PORT=161

# SNMP community string (default: public)
SNMP_COMMUNITY=public

# Listen address (default: 0.0.0.0)
SNMP_LISTEN_ADDRESS=0.0.0.0

# Enterprise OID (default: .1.3.6.1.4.1.99999)
# For production, obtain your own from IANA: https://www.iana.org/assignments/enterprise-numbers
SNMP_ENTERPRISE_OID=.1.3.6.1.4.1.99999

# Trap destinations (comma-separated, format: host:port)
# SNMP_TRAP_DESTINATIONS=192.168.1.100:162,192.168.1.101:162
```

### Docker Compose Example

```yaml
services:
  internet-monitor:
    image: ghcr.io/nickborgers/internet-connection-monitor:latest
    network_mode: host
    environment:
      - SNMP_ENABLED=true
      - SNMP_PORT=161
      - SNMP_COMMUNITY=mySecretCommunity
      - SNMP_TRAP_DESTINATIONS=192.168.1.100:162
```

**Important**: Use `network_mode: host` to bind SNMP port 161 (which requires root/elevated privileges in most systems).

## OID Structure

### Enterprise OID
Base: `.1.3.6.1.4.1.99999`

### Branch 1: General Statistics (`.1.3.6.1.4.1.99999.1`)

| OID | Name | Type | Description |
|-----|------|------|-------------|
| .1.3.6.1.4.1.99999.1.1.0 | cacheSize | Integer | Current cache size |
| .1.3.6.1.4.1.99999.1.2.0 | maxCacheSize | Integer | Maximum cache size |
| .1.3.6.1.4.1.99999.1.3.0 | monitoredSitesCount | Integer | Number of monitored sites |
| .1.3.6.1.4.1.99999.1.4.0 | totalTestsRun | Counter64 | Total tests executed |
| .1.3.6.1.4.1.99999.1.5.0 | totalSuccesses | Counter64 | Total successful tests |
| .1.3.6.1.4.1.99999.1.6.0 | totalFailures | Counter64 | Total failed tests |

### Branch 2: Per-Site Statistics (`.1.3.6.1.4.1.99999.2`)

Table format: `.1.3.6.1.4.1.99999.2.<siteIndex>.<metric>`

| Metric | Type | Description |
|--------|------|-------------|
| .1 | String | Site name |
| .2 | Counter64 | Total tests for site |
| .3 | Counter64 | Successful tests |
| .4 | Counter64 | Failed tests |
| .5 | Gauge32 | Last duration (ms) |
| .6 | Gauge32 | Average duration (ms) |
| .7 | Gauge32 | Min duration (ms) |
| .8 | Gauge32 | Max duration (ms) |
| .9 | Counter64 | Last success time (Unix timestamp) |
| .10 | Counter64 | Last failure time (Unix timestamp) |

### Branch 3: Recent Test Results (`.1.3.6.1.4.1.99999.3`)

Table format: `.1.3.6.1.4.1.99999.3.<testIndex>.<metric>`

| Metric | Type | Description |
|--------|------|-------------|
| .1 | String | Site name |
| .2 | Counter64 | Timestamp (Unix) |
| .3 | Integer | Success (1) or failure (0) |
| .4 | Gauge32 | Total duration (ms) |
| .5 | Integer | HTTP status code |

## Usage Examples

### Using snmpget

Query total tests run:
```bash
snmpget -v2c -c public localhost .1.3.6.1.4.1.99999.1.4.0
```

Query first site's name:
```bash
snmpget -v2c -c public localhost .1.3.6.1.4.1.99999.2.1.1
```

### Using snmpwalk

Walk all general statistics:
```bash
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.99999.1
```

Walk all site statistics:
```bash
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.99999.2
```

Walk entire MIB:
```bash
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.99999
```

### Using HTTP API

Get all SNMP data as JSON:
```bash
curl http://localhost:162/snmp/data | jq .
```

Get MIB definition:
```bash
curl http://localhost:162/snmp/mib
```

Get list of available OIDs:
```bash
curl http://localhost:162/snmp/oids | jq .
```

## Zabbix Integration

### 1. Import MIB (Optional)

Copy `INTERNET-CONNECTION-MONITOR-MIB.txt` to Zabbix server:
```bash
sudo cp INTERNET-CONNECTION-MONITOR-MIB.txt /usr/share/snmp/mibs/
```

### 2. Create Host

1. Go to **Configuration → Hosts**
2. Click **Create host**
3. Set **Host name**: `internet-monitor`
4. Set **Groups**: Create/select `Monitoring`
5. Set **Interfaces**:
   - Type: SNMP
   - IP address: `<monitor-ip>`
   - Port: `161`
   - SNMP version: `SNMPv2`
   - Community: `{$SNMP_COMMUNITY}` (define as macro)

### 3. Create Items

Example items to monitor:

**Total Tests Run**:
- Name: `Internet Monitor: Total Tests`
- Type: `SNMP agent`
- Key: `internet.monitor.total.tests`
- SNMP OID: `.1.3.6.1.4.1.99999.1.4.0`
- Type of information: `Numeric (unsigned)`

**Success Rate (Calculated)**:
- Name: `Internet Monitor: Success Rate`
- Type: `Calculated`
- Key: `internet.monitor.success.rate`
- Formula: `last(//internet.monitor.total.successes) / last(//internet.monitor.total.tests) * 100`

**Per-Site Items** (Discovery rule recommended):
- Name: `Internet Monitor: {#SITENAME} Average Duration`
- SNMP OID: `.1.3.6.1.4.1.99999.2.{#SITEINDEX}.6`

### 4. Create Triggers

**High Failure Rate**:
- Name: `Internet Monitor: High failure rate on {HOST.NAME}`
- Expression: `last(/internet-monitor/internet.monitor.success.rate)<90`
- Severity: `Warning`

**Site Degraded**:
- Name: `Internet Monitor: {#SITENAME} is degraded`
- Expression: `Site-specific failure rate > 50%`
- Severity: `Average`

### 5. Configure Trap Reception

Edit `/etc/zabbix/zabbix_server.conf`:
```conf
StartSNMPTrapper=1
SNMPTrapperFile=/var/log/snmptrap/snmptrap.log
```

Configure snmptrapd (`/etc/snmp/snmptrapd.conf`):
```conf
authCommunity log,execute,net public
perl do "/usr/share/zabbix-server/snmptt_zabbix_traps.pl";
```

## Troubleshooting

### Port 161 Permission Denied

SNMP port 161 requires root privileges. Options:

1. **Run container as root** (not recommended):
   ```yaml
   user: root
   ```

2. **Use alternative port** (recommended for testing):
   ```yaml
   environment:
     - SNMP_PORT=1161
   ```

3. **Use host networking** (recommended for production):
   ```yaml
   network_mode: host
   ```

### No SNMP Response

Check if SNMP agent is running:
```bash
# Check HTTP API (easier to debug)
curl http://localhost:162/snmp/data

# Check SNMP port is listening
netstat -tulpn | grep 161
```

Check community string:
```bash
# Try with correct community
snmpget -v2c -c public localhost .1.3.6.1.4.1.99999.1.1.0

# Wrong community will return timeout/no response
```

### Empty Statistics

SNMP statistics are ephemeral and reset on restart. Wait for tests to run:
```bash
# Check if tests are running
docker logs internet-monitor | grep "success"

# Verify via HTTP API
curl http://localhost:162/snmp/data | jq '.monitored_sites'
```

### HTTP API Not Working

HTTP API runs on SNMP_PORT + 1:
```bash
# Default: Port 162 (SNMP 161 + 1)
curl http://localhost:162/snmp/data

# Custom port: If SNMP_PORT=1161, HTTP API is on 1162
curl http://localhost:1162/snmp/data
```

## Performance Considerations

### Resource Usage
- **CPU**: Minimal (<1% when idle, ~2-5% during query processing)
- **Memory**: ~5-10 MB for SNMP agent + cache
- **Network**: Negligible (SNMP queries are small, typically <1 KB)

### Scalability
- **Cache size**: Default 100 results, configurable
- **Query rate**: Can handle hundreds of SNMP queries per second
- **Site count**: Tested with up to 50 monitored sites without issues

### Security Best Practices

1. **Change default community**: Don't use `public` in production
2. **Use strong community strings**: Treat like passwords (long, random)
3. **Firewall rules**: Only allow SNMP from monitoring servers
4. **Read-only access**: The agent is read-only by design
5. **No SNMPv3 yet**: SNMPv2c only, consider network isolation for sensitive environments
6. **Register your OID**: Obtain official enterprise number from IANA for production use

## Advanced Usage

### Custom OID Queries

Query specific metrics programmatically:
```python
from pysnmp.hlapi import *

# Query total tests
iterator = getCmd(
    SnmpEngine(),
    CommunityData('public'),
    UdpTransportTarget(('localhost', 161)),
    ContextData(),
    ObjectType(ObjectIdentity('.1.3.6.1.4.1.99999.1.4.0'))
)

errorIndication, errorStatus, errorIndex, varBinds = next(iterator)

if not errorIndication and not errorStatus:
    for varBind in varBinds:
        print(f'{varBind[0]} = {varBind[1]}')
```

### Monitoring with Prometheus

The SNMP HTTP API can be scraped by Prometheus using the JSON exporter:
```yaml
scrape_configs:
  - job_name: 'internet-monitor-snmp'
    static_configs:
      - targets: ['localhost:162']
    metrics_path: '/snmp/data'
    metric_relabel_configs:
      - source_labels: [__name__]
        target_label: __name__
        regex: 'snmp_(.*)'
        replacement: 'internet_monitor_snmp_$1'
```

### Grafana Dashboard for SNMP

Query SNMP data directly from Grafana using the JSON datasource:
1. Add JSON datasource pointing to `http://monitor:162/snmp/data`
2. Create panels querying `$.cache_size`, `$.monitored_sites`, etc.
3. Use transformations to extract site-specific metrics

## See Also

- [INTERNET-CONNECTION-MONITOR-MIB.txt](INTERNET-CONNECTION-MONITOR-MIB.txt) - Full MIB definition
- [DESIGN.md](DESIGN.md) - Architecture and implementation details
- [TESTING.md](TESTING.md) - Testing documentation including SNMP tests
- [IANA Enterprise Numbers](https://www.iana.org/assignments/enterprise-numbers) - Register your OID
