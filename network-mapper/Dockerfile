# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o network-mapper .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    iputils \
    net-tools \
    bind-tools \
    iproute2 \
    arp-scan \
    nmap-nping \
    curl

# Create non-root user
RUN addgroup -g 1001 -S netmap && \
    adduser -u 1001 -S netmap -G netmap

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/network-mapper .

# Change ownership
RUN chown netmap:netmap /app/network-mapper

# Switch to non-root user
USER netmap

# Set entrypoint
ENTRYPOINT ["./network-mapper"]

# Default command
CMD ["--help"]

# Add labels
LABEL org.opencontainers.image.title="Network Mapper"
LABEL org.opencontainers.image.description="Cross-platform network discovery and visualization tool"
LABEL org.opencontainers.image.source="https://github.com/NickBorgers/util"
LABEL org.opencontainers.image.licenses="MIT"