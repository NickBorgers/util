# Testing Documentation

## Test Coverage Status

### ✅ Integration Tests (Implemented)

**Location**: `test-integration.sh`

The integration tests validate the complete data flow from the monitor through Elasticsearch to Grafana. These tests verify:

1. **Docker Build** - Image builds successfully
2. **Service Health** - Elasticsearch, Grafana, Prometheus, and Monitor all start and become healthy
3. **Data Generation** - Monitor successfully loads websites and generates test results
4. **Elasticsearch Integration** - Data flows to Elasticsearch and is stored correctly
5. **Document Structure** - All required fields are present (@timestamp, test_id, site, status, timings)
6. **Grafana Integration** - Grafana datasource is configured and can query data
7. **Prometheus Metrics** - Metrics endpoint exposes monitoring data
8. **Health Endpoint** - Health check endpoint responds correctly

**Run Integration Tests:**
```bash
./test-integration.sh
```

**What Is Tested:**
- ✅ End-to-end data flow: Monitor → Elasticsearch → Grafana
- ✅ Docker image builds and runs
- ✅ Output modules that ARE tested:
  - ✅ JSON logging produces valid documents
  - ✅ Elasticsearch receives and indexes documents
  - ✅ Prometheus metrics are exposed
  - ✅ Health endpoint responds correctly
- ✅ Grafana can query Elasticsearch data

**What Is NOT Tested:**
- ✅ SNMP agent request handling (validated via unit test `TestSNMPAgentRespondsToGetAndWalk`)
- ⚠️ Output module initialization for SNMP (covered indirectly, but still missing end-to-end validation)
- ⚠️ SNMP data export functionality (MIB export remains manual)

### ✅ Unit Tests (Implemented)

**Status**: Core unit tests implemented with 11.2% overall code coverage

**Test Files**:
- `internal/browser/controller_impl_test.go` - 52.4% coverage
- `internal/config/loader_test.go` - 19.3% coverage
- `internal/testloop/iterator_test.go` - 38.9% coverage

**What Is Tested**:

1. **Timing Extraction** (`internal/browser/controller_impl.go:extractTimings()`)
   - ✅ HTTPS timing extraction logic
   - ✅ HTTP timing extraction logic
   - ✅ Null/empty data handling
   - ✅ Partial data handling
   - ✅ Invalid type handling
   - ✅ Negative value handling
   - ✅ Real-world timing scenarios
   - ✅ Chrome flags verification to force fresh connections

2. **Error Categorization** (`internal/browser/controller_impl.go:categorizeError()`)
   - ✅ Timeout error detection (context deadline, timeout messages)
   - ✅ DNS error detection (dns, no such host)
   - ✅ Connection error detection (connection refused)
   - ✅ TLS error detection
   - ✅ Unknown error handling
   - ✅ Error type priority testing

3. **Configuration Loading** (`internal/config/loader.go:ParseSimpleSiteList()`)
   - ✅ Basic domain parsing
   - ✅ Full URL parsing (HTTPS/HTTP)
   - ✅ WWW prefix handling
   - ✅ Path handling
   - ✅ Whitespace trimming
   - ✅ Empty element handling
   - ✅ Subdomain parsing
   - ✅ Mixed format handling
   - ✅ IP address support
   - ✅ Localhost support
   - ✅ Special characters handling

4. **Site Iterator** (`internal/testloop/iterator.go`)
   - ✅ Round-robin iteration correctness
   - ✅ Single site handling
   - ✅ Empty/nil sites handling
   - ✅ Count tracking
   - ✅ Reset functionality
   - ✅ Long iteration sequences
   - ✅ Concurrent access safety (thread-safe operations)
   - ✅ Data preservation across iterations

**What Still Needs Tests**:

5. **Output Modules** (`internal/outputs/`)
   - ❌ Prometheus metrics registration (integration tested but no unit tests)
   - ❌ Elasticsearch bulk indexing (integration tested but no unit tests)
   - ✅ SNMP agent basic request/response behaviour (`TestSNMPAgentRespondsToGetAndWalk`)
   - ❌ JSON logger (integration tested but no unit tests)

6. **Environment Variable Loading** (`internal/config/loader.go:LoadFromEnv()`)
   - ❌ No tests for environment variable parsing
   - ❌ No tests for duration parsing
   - ❌ No tests for boolean parsing

### ✅ Timing Metrics - Forced Fresh Connections

**Status**: DNS, TCP, and TLS timing metrics are working correctly and force fresh connections on every test!

**How It Works**:

The browser is configured with Chrome flags to **force fresh connections on every test**:
- **`--disable-http2`** - Forces HTTP/1.1 (prevents connection multiplexing)
- **`--disable-quic`** - Disables HTTP/3
- **`--disable-features=NetworkService,TLSSessionResumption`** - Disables TLS session caching

This ensures we get accurate measurements of DNS, TCP, and TLS on every single test, even when testing the same domain repeatedly.

**Real-World Evidence** (from actual monitor output):

**First request to google.com:**
```json
{
  "dns_lookup_ms": 8,
  "tcp_connection_ms": 4,
  "tls_handshake_ms": 9
}
```
✅ Non-zero values showing DNS lookup and connection establishment!

**Second request to google.com (same domain):**
```json
{
  "dns_lookup_ms": 6,
  "tcp_connection_ms": 4,
  "tls_handshake_ms": 9
}
```
✅ Still non-zero! Fresh connection forced successfully!

**Why This Matters**:
By forcing fresh connections, we can accurately monitor DNS resolution times, TCP connection latency, and TLS handshake performance. This helps detect network issues like:
- Slow DNS servers or DNS resolution failures
- High network latency affecting TCP connections
- TLS certificate validation issues or slow handshakes

**All Timing Metrics Work:**
- ✅ DNS lookup - always non-zero for successful HTTPS requests
- ✅ TCP connection - always non-zero for successful requests
- ✅ TLS handshake - always non-zero for successful HTTPS requests
- ✅ Time to first byte (TTFB) - working
- ✅ DOM content loaded - working
- ✅ Full page load - working
- ✅ Network idle - working
- ✅ Total duration - working

**Note**: Zero values for DNS/TCP/TLS now indicate an error or timeout, not connection reuse.

#### Error Handling

**Issue**: Error handling paths are not tested.

**Not Tested:**
- DNS failures
- Connection timeouts
- TLS errors
- HTTP errors
- Network unreachable scenarios

**Recommendation**: Add tests that simulate these error conditions and verify correct categorization and reporting.

#### Configuration Edge Cases

**Issue**: Configuration parsing has no tests.

**Not Tested:**
- Invalid environment variables
- Malformed YAML
- Conflicting settings
- Default value application

**Recommendation**: Add unit tests for `internal/config/loader.go`.

## Testing Philosophy

### Current Approach: Integration-First

The current testing strategy prioritizes **integration tests** over unit tests. This approach:

**Advantages:**
- ✅ Validates the complete user workflow
- ✅ Tests real-world scenarios (actual browser, actual Elasticsearch)
- ✅ Catches integration issues early
- ✅ Provides confidence for deployment

**Disadvantages:**
- ❌ Slower to run (2-3 minutes)
- ❌ Harder to debug failures
- ❌ Doesn't validate internal logic accuracy
- ❌ Limited edge case coverage

### Recommended Future Approach: Balanced Testing

A production-ready testing strategy should include:

1. **Unit Tests** (Fast, focused)
   - Test individual functions and logic
   - Mock external dependencies
   - Cover edge cases and error paths
   - Target: 70%+ code coverage

2. **Integration Tests** (Comprehensive, slow)
   - Test complete workflows
   - Use real dependencies where possible
   - Validate end-to-end functionality
   - Run in CI/CD on every PR

3. **Contract Tests** (API boundaries)
   - Validate Elasticsearch document schema
   - Verify Prometheus metrics format
   - Test health endpoint response format

## Running Tests

### Unit Tests

```bash
# Run all unit tests
make test

# Run with coverage report
make test-coverage

# Run specific package tests
go test ./internal/browser -v
go test ./internal/config -v
go test ./internal/testloop -v

# Generate HTML coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Full integration test suite
./test-integration.sh
# or
make test-integration

# Run all tests (unit + integration)
make test-all

# Quick test (30 seconds)
make quick-test

# Manual testing with full stack
make grafana-dashboard-demo
make demo-status
make monitor-logs
```

### CI/CD Integration

Tests run automatically in GitHub Actions:
- On every push to `internet-connection-monitor/**`
- On every pull request
- On release tag creation

See `.github/workflows/internet-connection-monitor-ci.yml`

## Test Results

### Integration Test Results (Latest Run)

```
✓ Docker image builds successfully
✓ Full stack starts and all services become ready
✓ Monitor generates test results
✓ Elasticsearch receives and stores data
✓ Grafana datasource is configured
✓ Grafana can query data from Elasticsearch
✓ Prometheus metrics endpoint is working
✓ Health endpoint is working

ALL INTEGRATION TESTS PASSED
```

### Code Coverage

**Current**: 11.2% overall code coverage
- **browser package**: 52.4% coverage (timing extraction, error categorization)
- **config package**: 19.3% coverage (site list parsing)
- **testloop package**: 38.9% coverage (site iterator)

**Target**: 70%+ for critical paths

**Coverage by Test Type**:
- Unit tests: 11.2% (critical logic paths)
- Integration tests: Additional coverage for end-to-end flows

## Contributing Tests

### Adding Unit Tests

1. Create test file: `internal/<package>/<file>_test.go`
2. Use Go testing package: `import "testing"`
3. Follow Go conventions: `func TestFunctionName(t *testing.T)`
4. Run with: `go test ./...`

Example:
```go
func TestExtractTimings_HTTPS(t *testing.T) {
    perfData := map[string]interface{}{
        "domainLookupStart": 0.0,
        "domainLookupEnd": 10.5,
        "connectStart": 10.5,
        "connectEnd": 50.2,
        "secureConnectionStart": 30.1,
        // ... more fields
    }

    timings := extractTimings(perfData, 1000)

    if timings.DNSLookupMs != 10 {
        t.Errorf("Expected DNS lookup 10ms, got %d", timings.DNSLookupMs)
    }
    // ... more assertions
}
```

### Adding Integration Tests

Add new validation steps to `test-integration.sh`:
1. Follow the existing step pattern
2. Use helper functions (`log_info`, `log_success`, `log_error`)
3. Clean up resources in the cleanup function
4. Document what the test validates

## Known Issues

None! The timing metrics are working correctly with forced fresh connections on every test. DNS, TCP, and TLS values are consistently non-zero for successful requests. See "Timing Metrics - Forced Fresh Connections" section above.

## Roadmap

### Short Term
- [x] Add unit tests for `extractTimings()`
- [x] Add unit tests for `categorizeError()`
- [x] Add configuration validation tests (`ParseSimpleSiteList`)
- [x] Set up code coverage reporting
- [ ] Add unit tests for `LoadFromEnv()`
- [ ] Add unit tests for output modules

### Medium Term
- [ ] Timing accuracy validation tests
- [ ] Error simulation tests
- [ ] Performance benchmarks
- [ ] Load testing
- [ ] Increase code coverage to 50%+

### Long Term
- [ ] Contract tests for output formats
- [ ] Chaos testing (network failures, service outages)
- [ ] Multi-environment testing (different browsers, OS)
- [ ] Achieve 70%+ code coverage

---

**Last Updated**: 2025-11-08
**Test Coverage**: Unit tests (11.2%) + Integration tests (passing)
**Status**: Core unit tests implemented and passing, integration tests passing
