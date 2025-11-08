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
- ❌ SNMP agent (implemented but not validated)
- ❌ Output module initialization for SNMP
- ❌ SNMP data export functionality

### ❌ Unit Tests (Not Implemented)

**Status**: No unit tests exist

The following components have **zero test coverage**:

1. **Timing Extraction** (`internal/browser/controller_impl.go:extractTimings()`)
   - Complex logic for calculating DNS, TCP, TLS, TTFB timings
   - 70 lines of conditional logic
   - No validation of accuracy

2. **Error Categorization** (`internal/browser/controller_impl.go:categorizeError()`)
   - String-based error classification
   - No tests for edge cases

3. **Configuration Loading** (`internal/config/`)
   - Environment variable parsing
   - YAML config merging
   - No validation tests

4. **Site Iterator** (`internal/testloop/iterator.go`)
   - Round-robin logic
   - No tests for iteration correctness

5. **Output Modules** (`internal/outputs/`)
   - Prometheus metrics registration (integration tested but no unit tests)
   - Elasticsearch bulk indexing (integration tested but no unit tests)
   - SNMP agent (NOT tested at all - no unit or integration tests)
   - JSON logger (integration tested but no unit tests)

### ⚠️ Known Testing Gaps

#### Timing Metrics - Partially Broken

**Issue**: DNS, TCP, and TLS timing fields always return 0, even for successful tests.

**What Works:**
- ✅ Total duration - accurate
- ✅ Time to first byte (TTFB) - working
- ✅ DOM content loaded - working
- ✅ Full page load - working
- ✅ Network idle - working

**What's Broken:**
- ❌ DNS lookup timing - always 0
- ❌ TCP connection timing - always 0
- ❌ TLS handshake timing - always 0

**Root Cause**: The extraction logic exists (`controller_impl.go:210-230`) and looks correct, but the Performance Navigation Timing API isn't populating these fields. This could be due to:
- Headless Chrome not tracking these timings
- Connection reuse/caching (though caching is disabled)
- Timing data not available when queried
- Need to query a different API endpoint

**Why This Matters**: The README advertises timing metrics as a feature, but 3 of 8 timing fields are broken.

**Recommendation**:
- Investigate why Performance.timing API doesn't populate DNS/TCP/TLS
- Add unit tests that validate extraction logic with mock data
- Add integration tests that verify these fields are non-zero for actual tests
- Consider alternative APIs (Resource Timing, Chrome DevTools Protocol)

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

### Integration Tests

```bash
# Full integration test suite
./test-integration.sh

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

**Current**: No coverage tracking (no unit tests)
**Target**: 70%+ for critical paths

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

1. **Timing Zeros**: Some timing fields (DNS, TCP, TLS) may show 0ms values. This is not validated and may indicate either very fast connections or issues with the extraction logic.

2. **No Unit Tests**: Critical logic (especially timing extraction) has zero test coverage.

## Roadmap

### Short Term
- [ ] Add unit tests for `extractTimings()`
- [ ] Add unit tests for `categorizeError()`
- [ ] Add configuration validation tests
- [ ] Set up code coverage reporting

### Medium Term
- [ ] Timing accuracy validation tests
- [ ] Error simulation tests
- [ ] Performance benchmarks
- [ ] Load testing

### Long Term
- [ ] Contract tests for output formats
- [ ] Chaos testing (network failures, service outages)
- [ ] Multi-environment testing (different browsers, OS)

---

**Last Updated**: 2025-11-08
**Test Coverage**: Integration tests only, no unit tests
**Status**: Integration tests passing, unit test coverage needed
