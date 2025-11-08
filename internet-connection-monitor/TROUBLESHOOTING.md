# Internet Connection Monitor - Troubleshooting

## Common Issues

## Service Won't Start

### Port conflicts

**Symptoms:**
```
Error starting userland proxy: listen tcp4 0.0.0.0:9090: bind: address already in use
```

**Solution:**
Check which ports are in use:
```bash
# Check what's using port 9090
sudo lsof -i :9090

# Or use netstat
netstat -tulpn | grep 9090
```

Change ports in `deployments/.env`:
```bash
PROM_HOST_PORT=9190   # Instead of 9090
SNMP_HOST_PORT=1161   # Instead of 161
```

### Browser not found (local binary only)

**Symptoms:**
```
exec: "google-chrome": executable file not found in $PATH
```

**Solution:**
This only affects `make run-binary` or `make dev` (local execution).

**Option 1: Use Docker** (recommended)
```bash
make quick-start    # Uses Docker (includes Chrome)
```

**Option 2: Install Chrome**
```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install --cask google-chrome

# Then try again
make run-binary
```

---

## No Data in Grafana

**Symptoms:**
- Grafana dashboard is empty
- "No data" message

**Checks:**

1. **Verify monitor is running:**
```bash
make demo-status
# Should show all services healthy
```

2. **Check Elasticsearch has data:**
```bash
make es-count
# Should show document count > 0
```

3. **Wait a few minutes:**
The monitor tests sites every 2 seconds, but it may take 1-2 minutes for data to accumulate and indices to be created.

4. **Check monitor logs:**
```bash
make monitor-logs
# Should show JSON test results
```

5. **Verify Elasticsearch connection:**
```bash
make es-health
# Should show cluster status: green or yellow
```

---

## Tests Timing Out

**Symptoms:**
```json
{"status":{"success":false},"error":{"error_type":"timeout"}}
```

**Causes:**
- Slow internet connection
- Site is actually down
- Timeout too short for your connection

**Solutions:**

1. **Increase timeout:**
```bash
# In deployments/.env
GLOBAL_TIMEOUT=60s   # Instead of 30s
```

2. **Check actual connectivity:**
```bash
# Test if site is reachable
curl -I https://www.google.com
```

3. **Reduce sites to test:**
```bash
# In deployments/.env
SITES=google.com,example.com   # Test fewer sites
```

---

## High Memory Usage

**Symptoms:**
- Container using > 500MB RAM
- System running slow

**Normal usage:**
- ~50-100MB: Normal operation (1 browser instance)
- ~150-200MB: During active page loads
- > 300MB: May indicate issues

**Solutions:**

1. **Increase test delay:**
```bash
# In deployments/.env
INTER_TEST_DELAY=5s   # Slower testing, less memory pressure
```

2. **Reduce concurrent tests:**
Already set to 1 (serial testing) by default.

3. **Disable images:**
```bash
# In config file
browser:
  disable_images: true
```

4. **Test fewer sites:**
```bash
SITES=google.com,github.com   # Just 2 sites
```

---

## Docker Build Fails

**Symptoms:**
```
error getting credentials - err: exit status 255
```

**Solution:**
This is a Docker credential helper issue, not related to the monitor.

**Fix:**
```bash
# Remove credential helper
rm ~/.docker/config.json

# Or edit ~/.docker/config.json and remove credStore line
```

---

## SNMP Not Working

**Symptoms:**
- snmpwalk returns no data
- Zabbix can't poll SNMP

**Checks:**

1. **Verify SNMP is enabled:**
```bash
# Check container logs for "SNMP agent" message
make monitor-logs
```

2. **Check port binding:**
```bash
docker ps | grep internet-monitor
# Should show 0.0.0.0:161->161/udp
```

3. **Test with snmpwalk:**
```bash
# Install snmpwalk if needed
sudo apt-get install snmp

# Test
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1
```

4. **Check firewall:**
```bash
# Allow UDP port 161
sudo ufw allow 161/udp
```

---

## Prometheus Metrics Empty

**Symptoms:**
- `/metrics` endpoint returns 404 or empty

**Checks:**

1. **Verify Prometheus is enabled:**
```bash
# Should be enabled by default
docker logs internet-monitor | grep "Prometheus"
```

2. **Check endpoint:**
```bash
curl http://localhost:9090/metrics
# Should return Prometheus text format
```

3. **Verify container port:**
```bash
docker ps | grep internet-monitor
# Should show 0.0.0.0:9090->9090/tcp
```

---

## Quick Debugging Commands

```bash
# Check everything at once
make demo-status

# View all logs (filtered)
make demo-logs

# Check Elasticsearch
make es-health
make es-count
make es-sample

# Test monitor directly
make quick-test

# Shell into container
make shell-monitor
```

---

## Getting Help

1. **Check existing issues:** https://github.com/nickborgers/monorepo/issues
2. **Run `make help`** for all commands
3. **Read documentation:**
   - [DESIGN.md](DESIGN.md) - Architecture
   - [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) - Current status
   - [README.md](README.md) - Overview

---

## Known Limitations

1. **Timing granularity** - Some detailed timing fields (DNS, TCP, TLS) show 0 (performance.timing API limitations)
2. **Go version** - Requires Go 1.25+
3. **Single browser** - Tests serially (by design, mimics real user)

---

**Last Updated:** 2025-11-08
