# Grafana Dashboard Updates for v1.3.0

This document describes the required changes to the Grafana dashboard to support the new `failure_phase` field introduced in v1.3.0.

## Overview

The v1.3.0 release adds network failure phase detection, which requires dashboard updates to visualize:
1. Which network layer failures occur at (DNS, TCP, TLS, HTTP)
2. Specific Chrome error codes (e.g., ERR_NAME_NOT_RESOLVED)
3. Correlation between error types and failure phases

## Required Dashboard Changes

### 1. Update "Recent Failures" Table

**Current columns:**
- @timestamp
- site.name
- error.error_type
- timings.total_duration_ms

**Updated columns:**
- @timestamp
- site.name
- error.error_type (now shows specific Chrome codes like "ERR_NAME_NOT_RESOLVED")
- **error.failure_phase** (NEW - shows "dns", "tcp", "tls", "http", "unknown")
- timings.total_duration_ms

**Elasticsearch query:**
```json
{
  "query": "status.success:false AND site.name.keyword:($site_name)",
  "size": 50,
  "sort": [{"@timestamp": {"order": "desc"}}]
}
```

**Display columns to add:**
- Field: `error.failure_phase`
- Header: "Failure Phase"
- Type: keyword

### 2. Add New Panel: "Failures by Phase" (Pie Chart)

**Position:** Below "Error Distribution" panel
**Type:** Pie chart
**Title:** "Failures by Phase"
**Description:** "Distribution of failures across network layers (DNS, TCP, TLS, HTTP)"

**Elasticsearch query:**
```json
{
  "query": "status.success:false AND site.name.keyword:($site_name)",
  "metrics": [
    {
      "type": "count"
    }
  ],
  "bucketAggs": [
    {
      "type": "terms",
      "field": "error.failure_phase.keyword",
      "settings": {
        "size": 10,
        "order": "desc",
        "orderBy": "_count"
      }
    }
  ]
}
```

**Visualization settings:**
- Legend: Show
- Legend position: Right
- Show percentages: Yes
- Donut mode: Optional (looks cleaner)

**Expected output:**
- dns: 35%
- tcp: 20%
- http: 30%
- tls: 10%
- unknown: 5%

### 3. Add New Panel: "Error Types by Phase" (Table)

**Position:** Replace or augment the current "Error Distribution" panel
**Type:** Table
**Title:** "Error Types by Failure Phase"
**Description:** "Detailed breakdown of Chrome error codes grouped by network layer"

**Elasticsearch query:**
```json
{
  "query": "status.success:false AND site.name.keyword:($site_name)",
  "metrics": [
    {
      "type": "count"
    }
  ],
  "bucketAggs": [
    {
      "id": "2",
      "type": "terms",
      "field": "error.failure_phase.keyword",
      "settings": {
        "size": 10,
        "order": "desc",
        "orderBy": "_count"
      }
    },
    {
      "id": "3",
      "type": "terms",
      "field": "error.error_type.keyword",
      "settings": {
        "size": 20,
        "order": "desc",
        "orderBy": "_count"
      }
    }
  ]
}
```

**Display columns:**
- Failure Phase
- Error Type
- Count
- % of Total

**Expected output:**
| Phase | Error Type | Count | % |
|-------|-----------|-------|---|
| dns | ERR_NAME_NOT_RESOLVED | 150 | 35% |
| tcp | ERR_CONNECTION_REFUSED | 80 | 18% |
| tcp | ERR_CONNECTION_TIMED_OUT | 10 | 2% |
| http | timeout | 120 | 28% |
| tls | ERR_CERT_AUTHORITY_INVALID | 40 | 9% |

### 4. Update "Error Distribution" Panel

**Current:** Shows error_type (generic categories like "timeout", "dns")
**Updated:** Can now show specific Chrome error codes

**Option A - By specific error type:**
```json
{
  "query": "status.success:false AND site.name.keyword:($site_name)",
  "metrics": [{"type": "count"}],
  "bucketAggs": [
    {
      "type": "terms",
      "field": "error.error_type.keyword",
      "size": 10
    }
  ]
}
```

**Option B - By failure phase (high-level view):**
```json
{
  "query": "status.success:false AND site.name.keyword:($site_name)",
  "metrics": [{"type": "count"}],
  "bucketAggs": [
    {
      "type": "terms",
      "field": "error.failure_phase.keyword"
    }
  ]
}
```

**Recommendation:** Keep the existing error_type visualization and add the new "Failures by Phase" as a separate panel.

## Example Dashboard Queries for Troubleshooting

### All DNS failures (any DNS-related error)
```
error.failure_phase: "dns"
```

### Specific Chrome error
```
error.error_type: "ERR_NAME_NOT_RESOLVED"
```

### All timeouts across all phases
```
error.error_type: "timeout" OR error.error_type: "ERR_CONNECTION_TIMED_OUT" OR error.error_type: "ERR_TIMED_OUT"
```

### TCP layer issues (connection problems)
```
error.failure_phase: "tcp"
```

### Failures that made it past TLS but failed at HTTP
```
error.failure_phase: "http" AND timings.tls_handshake_ms: *
```

### Sites with the most DNS failures
```json
{
  "query": "error.failure_phase:dns",
  "bucketAggs": [
    {
      "type": "terms",
      "field": "site.name.keyword",
      "size": 10
    }
  ]
}
```

## Implementation Steps

1. **Export current dashboard:** Download the current dashboard JSON from Grafana
2. **Make updates in Grafana UI:**
   - Update "Recent Failures" table to include `error.failure_phase` column
   - Add "Failures by Phase" pie chart panel
   - Add "Error Types by Phase" table panel
   - Test all queries with real data
3. **Export updated dashboard:** Save the updated dashboard JSON
4. **Replace grafana-dashboard.json:** Update the file in the repository
5. **Test import:** Verify the dashboard can be imported successfully

## Panel Layout Recommendation

```
+------------------+------------------+------------------+------------------+
|  Success Rate    |  Avg Latency     |  P95 Latency     |  Total Tests     |
|     (Stat)       |     (Stat)       |     (Stat)       |     (Stat)       |
+------------------+------------------+------------------+------------------+
|                 Success and Failure Rate Over Time                      |
|                          (Time series)                                  |
+-------------------------------------------------------------------------+
|                    Page Load Times (Avg, P95, P99)                      |
|                          (Time series)                                  |
+-------------------------------------------------------------------------+
| DNS Lookup Time  | TCP Connection   | TLS Handshake    |                |
|  (Time series)   |  (Time series)   |  (Time series)   |                |
+------------------+------------------+------------------+------------------+
|              Success Rate by Site (Bar chart)                           |
+-------------------------------------------------------------------------+
|  Error Distribution | Failures by Phase (NEW)                           |
|    (Pie chart)      |    (Pie chart)                                    |
+--------------------+---------------------------------------------------+
|              Error Types by Phase (NEW - Table)                         |
+-------------------------------------------------------------------------+
|                    Recent Failures (Table)                              |
|         (Updated to include failure_phase column)                       |
+-------------------------------------------------------------------------+
```

## Notes

- The `error.failure_phase` field is only populated for failed requests
- The `error.error_type` field now contains specific Chrome error codes (e.g., "ERR_NAME_NOT_RESOLVED") instead of generic categories
- For chromedp errors without Chrome codes, error_type will be "timeout" or "unknown"
- All timing fields are optional and may be null if the failure occurred early in the connection process
