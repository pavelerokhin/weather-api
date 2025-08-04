# Stage 1: Dependencies - Separate dependency layer for better caching
FROM golang:1.24.3-alpine AS deps

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached unless go.mod/go.sum changes)
RUN go mod download && \
    go mod verify

# Stage 2: Builder - Compile the application
FROM golang:1.24.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /go/pkg/mod /go/pkg/mod

# Copy dependency files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOAMD64=v1 \
    go build \
        -ldflags="-w -s -extldflags=-static" \
        -trimpath \
        -o weather-api \
        ./cmd/weather-api

# Stage 3: Runtime - Minimal runtime image
FROM scratch AS runtime

# Copy timezone data and certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set timezone
ENV TZ=UTC

# Create non-root user (using numeric IDs for scratch image)
USER 1001:1001

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=1001:1001 /app/weather-api .

# Copy configuration files
COPY --from=builder --chown=1001:1001 /app/config/config.yaml ./config/config.yaml

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/weather-api", "-health"] || exit 1

# Run the application
ENTRYPOINT ["./weather-api"]

# Stage 4: Alpine Runtime (Alternative)
FROM alpine:3.19 AS runtime-alpine

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/weather-api .

# Copy configuration files
COPY --from=builder /app/config/config.yaml ./config/config.yaml

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./weather-api"]
