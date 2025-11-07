FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with build tags for locking support
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -tags "locking persistence" -o webdav-gateway ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /root/

# Create directories for data and logs
RUN mkdir -p /var/lib/webdav/locks /var/log/webdav-gateway

# Copy the binary from builder
COPY --from=builder /app/webdav-gateway .

# Copy additional scripts
COPY deployments/docker/scripts/ /usr/local/bin/

# Set permissions
RUN chmod +x /usr/local/bin/*

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the application with locking support
CMD ["./webdav-gateway"]