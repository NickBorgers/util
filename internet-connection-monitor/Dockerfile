# Multi-stage Dockerfile for Internet Connection Monitor
# Based on chromedp/headless-shell for headless Chrome support

# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY configs/ ./configs/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static" -s -w' \
    -o internet-monitor \
    ./cmd/monitor

# Stage 2: Runtime image with headless Chrome
FROM chromedp/headless-shell:stable

# Install ca-certificates for HTTPS
USER root
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for running the application
RUN useradd -m -u 1000 monitor

# Create working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/internet-monitor /app/internet-monitor

# Copy default configurations (optional, can be overridden)
COPY --from=builder /build/configs /app/configs

# Change ownership
RUN chown -R monitor:monitor /app

# Switch to non-root user
USER monitor

# Expose ports
# 9090 - Prometheus metrics
# 161/udp - SNMP
# 8080 - Health check
EXPOSE 9090 161/udp 8080

# Set default environment variables
ENV INTER_TEST_DELAY=2s \
    GLOBAL_TIMEOUT=30s \
    CACHE_SIZE=100 \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    PROM_ENABLED=true \
    PROM_PORT=9090 \
    SNMP_ENABLED=true \
    SNMP_PORT=161 \
    ES_ENABLED=false

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the monitor
ENTRYPOINT ["/app/internet-monitor"]
