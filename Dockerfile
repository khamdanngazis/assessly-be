# Multi-stage Dockerfile for Assessly API Server
# Stage 1: Build
FROM golang:1.25-alpine AS builder

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

# Build API binary
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux target
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /app/bin/api \
    ./cmd/api

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 assessly && \
    adduser -D -u 1000 -G assessly assessly

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/api /app/api

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Change ownership to non-root user
RUN chown -R assessly:assessly /app

# Switch to non-root user
USER assessly

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the API server
CMD ["/app/api"]
