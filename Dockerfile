# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Update go.mod and build the service binary with CGO enabled (required for SQLite)
RUN go mod tidy && CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /wasvc ./cmd/wasvc

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    tzdata

# Create non-root user
RUN addgroup -g 1000 wasvc && \
    adduser -u 1000 -G wasvc -s /bin/sh -D wasvc

# Create data directory
RUN mkdir -p /data && chown wasvc:wasvc /data

# Copy binary from builder
COPY --from=builder /wasvc /usr/local/bin/wasvc

# Switch to non-root user
USER wasvc

# Set working directory
WORKDIR /data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default environment variables
ENV WASVC_HOST=0.0.0.0 \
    WASVC_PORT=8080 \
    WASVC_DATA_DIR=/data

# Run the service
ENTRYPOINT ["/usr/local/bin/wasvc"]
