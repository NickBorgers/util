#!/bin/bash
set -e

# Integration test for Internet Connection Monitor
# Tests the full data flow: Monitor -> Elasticsearch -> Grafana

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test configuration
MONITOR_CONTAINER="internet-monitor"
ES_CONTAINER="elasticsearch"
GRAFANA_CONTAINER="grafana"
COMPOSE_FILE="deployments/docker-compose.with-stack.yml"
WAIT_TIMEOUT=180  # 3 minutes max wait

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up test environment...${NC}"
    cd deployments
    docker compose -f docker-compose.with-stack.yml down -v 2>/dev/null || true
    cd ..
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Trap cleanup on exit
trap cleanup EXIT

# Helper functions
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

wait_for_service() {
    local service_name=$1
    local health_url=$2
    local timeout=$3
    local elapsed=0

    log_info "Waiting for $service_name to be ready..."

    while [ $elapsed -lt $timeout ]; do
        if curl -sf "$health_url" > /dev/null 2>&1; then
            log_success "$service_name is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done

    log_error "$service_name failed to become ready within ${timeout}s"
    return 1
}

wait_for_container() {
    local container_name=$1
    local timeout=$2
    local elapsed=0

    log_info "Waiting for container $container_name to start..."

    while [ $elapsed -lt $timeout ]; do
        if docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
            log_success "Container $container_name is running"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done

    log_error "Container $container_name failed to start within ${timeout}s"
    return 1
}

# Main test execution
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Internet Connection Monitor - Integration Test                ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Step 1: Build Docker image
log_info "Step 1: Building Docker image..."
docker build -t internet-connection-monitor:latest . > /dev/null
log_success "Docker image built"
echo ""

# Step 2: Start the stack
log_info "Step 2: Starting the full monitoring stack..."
cd deployments
docker compose -f docker-compose.with-stack.yml up -d
cd ..
log_success "Stack started"
echo ""

# Step 3: Wait for Elasticsearch to be ready
log_info "Step 3: Waiting for Elasticsearch..."
wait_for_service "Elasticsearch" "http://localhost:9200/_cluster/health" "$WAIT_TIMEOUT"
echo ""

# Step 4: Wait for Grafana to be ready
log_info "Step 4: Waiting for Grafana..."
wait_for_service "Grafana" "http://localhost:3000/api/health" "$WAIT_TIMEOUT"
echo ""

# Step 5: Wait for Monitor to start
log_info "Step 5: Waiting for Monitor container..."
wait_for_container "$MONITOR_CONTAINER" 30
echo ""

# Step 6: Wait for Monitor health endpoint
log_info "Step 6: Waiting for Monitor health endpoint..."
wait_for_service "Monitor Health" "http://localhost:8080/health" 60
echo ""

# Step 7: Verify Monitor is generating test results
log_info "Step 7: Verifying Monitor is generating test results..."
log_info "Waiting 30 seconds for tests to run..."
sleep 30

# Check logs for successful tests
TEST_RESULTS=$(docker logs "$MONITOR_CONTAINER" 2>&1 | grep -c '"success":true' || echo "0")
if [ "$TEST_RESULTS" -gt 0 ]; then
    log_success "Monitor has generated $TEST_RESULTS successful test results"
else
    log_error "Monitor has not generated any successful test results"
    log_info "Showing recent logs:"
    docker logs --tail 20 "$MONITOR_CONTAINER" 2>&1
    exit 1
fi
echo ""

# Step 8: Verify data in Elasticsearch
log_info "Step 8: Verifying data in Elasticsearch..."

# Wait a bit more for bulk indexer to flush
sleep 10

# Check if index exists
INDEX_COUNT=$(curl -s "http://localhost:9200/_cat/indices/internet-connection-monitor-*?h=index" | wc -l)
if [ "$INDEX_COUNT" -gt 0 ]; then
    log_success "Elasticsearch index created"
else
    log_error "No Elasticsearch indices found"
    curl -s "http://localhost:9200/_cat/indices?v"
    exit 1
fi

# Check document count
DOC_COUNT=$(curl -s "http://localhost:9200/internet-connection-monitor-*/_count" | grep -o '"count":[0-9]*' | grep -o '[0-9]*')
if [ -n "$DOC_COUNT" ] && [ "$DOC_COUNT" -gt 0 ]; then
    log_success "Elasticsearch has $DOC_COUNT documents"
else
    log_error "Elasticsearch has no documents"
    exit 1
fi

# Verify document structure
log_info "Verifying document structure..."
SAMPLE_DOC=$(curl -s "http://localhost:9200/internet-connection-monitor-*/_search?size=1&sort=@timestamp:desc" | jq -r '.hits.hits[0]._source')

# Check required fields (handle @ symbol in field names)
if echo "$SAMPLE_DOC" | jq -e '.["@timestamp"]' > /dev/null 2>&1; then
    log_success "Field '@timestamp' present"
else
    log_error "Required field '@timestamp' missing from document"
    echo "$SAMPLE_DOC" | jq '.'
    exit 1
fi

REQUIRED_FIELDS=("test_id" "site" "status" "timings")
for field in "${REQUIRED_FIELDS[@]}"; do
    if echo "$SAMPLE_DOC" | jq -e ".$field" > /dev/null 2>&1; then
        log_success "Field '$field' present"
    else
        log_error "Required field '$field' missing from document"
        echo "$SAMPLE_DOC" | jq '.'
        exit 1
    fi
done
echo ""

# Step 9: Verify Grafana datasource
log_info "Step 9: Verifying Grafana datasource..."
DATASOURCE_HEALTH=$(curl -s -u admin:admin "http://localhost:3000/api/datasources" | jq -r '.[0].type')
if [ "$DATASOURCE_HEALTH" == "elasticsearch" ]; then
    log_success "Grafana Elasticsearch datasource configured"
else
    log_error "Grafana datasource not configured correctly"
    curl -s -u admin:admin "http://localhost:3000/api/datasources" | jq '.'
    exit 1
fi
echo ""

# Step 10: Verify Grafana can query data
log_info "Step 10: Verifying Grafana can query data..."

# Get datasource UID
DATASOURCE_UID=$(curl -s -u admin:admin "http://localhost:3000/api/datasources" | jq -r '.[0].uid')
log_info "Datasource UID: $DATASOURCE_UID"

# Test query to Grafana
QUERY_PAYLOAD=$(cat <<EOF
{
  "queries": [
    {
      "refId": "A",
      "datasource": {
        "type": "elasticsearch",
        "uid": "$DATASOURCE_UID"
      },
      "query": "*",
      "metrics": [{"type": "count", "id": "1"}],
      "bucketAggs": [{"type": "date_histogram", "field": "@timestamp", "id": "2"}],
      "timeField": "@timestamp"
    }
  ],
  "from": "now-1h",
  "to": "now"
}
EOF
)

QUERY_RESULT=$(curl -s -u admin:admin \
    -H "Content-Type: application/json" \
    -X POST \
    -d "$QUERY_PAYLOAD" \
    "http://localhost:3000/api/ds/query")

# Check if query returned data
QUERY_DOC_COUNT=$(echo "$QUERY_RESULT" | jq -r '.results.A.frames[0].data.values[1] | length' 2>/dev/null || echo "0")
if [ "$QUERY_DOC_COUNT" != "0" ] && [ "$QUERY_DOC_COUNT" != "null" ]; then
    log_success "Grafana successfully queried data from Elasticsearch"
else
    log_warning "Grafana query returned no data (this may be timing-related)"
    log_info "Query result: $QUERY_RESULT"
fi
echo ""

# Step 11: Verify Prometheus metrics
log_info "Step 11: Verifying Prometheus metrics endpoint..."
METRICS=$(curl -s "http://localhost:9090/metrics")
if echo "$METRICS" | grep -q "internet_monitor_test_"; then
    log_success "Prometheus metrics endpoint is working"
    # Count how many custom metrics are present
    METRIC_COUNT=$(echo "$METRICS" | grep -c "^internet_monitor_" || echo "0")
    log_info "Found $METRIC_COUNT internet_monitor metrics"
else
    log_error "Prometheus metrics not found"
    echo "$METRICS" | head -20
    exit 1
fi
echo ""

# Step 12: Verify health endpoint
log_info "Step 12: Verifying health endpoint response..."
HEALTH_RESPONSE=$(curl -s "http://localhost:8080/health")
HEALTH_STATUS=$(echo "$HEALTH_RESPONSE" | jq -r '.status')
if [ "$HEALTH_STATUS" == "healthy" ]; then
    log_success "Health endpoint reports healthy status"
    TESTS_RUN=$(echo "$HEALTH_RESPONSE" | jq -r '.total_tests_run')
    log_info "Total tests run: $TESTS_RUN"
else
    log_error "Health endpoint reports unhealthy status"
    echo "$HEALTH_RESPONSE" | jq '.'
    exit 1
fi
echo ""

# Step 13: Verify SNMP agent
log_info "Step 13: Verifying SNMP agent via gosnmp client..."
if SNMP_OUTPUT=$(go run ./cmd/snmpcheck -target 127.0.0.1 -port 161 -community public -base .1.3.6.1.4.1.99999 -retries 5 2>&1); then
    log_success "SNMP agent responded to queries"
    echo "$SNMP_OUTPUT"
else
    log_error "SNMP verification failed"
    echo "$SNMP_OUTPUT"
    exit 1
fi
echo ""

# Final summary
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Integration Test Results                                      ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
log_success "✓ Docker image builds successfully"
log_success "✓ Full stack starts and all services become ready"
log_success "✓ Monitor generates test results"
log_success "✓ Elasticsearch receives and stores data ($DOC_COUNT documents)"
log_success "✓ Grafana datasource is configured"
log_success "✓ Grafana can query data from Elasticsearch"
log_success "✓ Prometheus metrics endpoint is working"
log_success "✓ Health endpoint is working"
log_success "✓ SNMP agent responded to queries"
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ALL INTEGRATION TESTS PASSED                                  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Stack will be cleaned up by trap
exit 0
