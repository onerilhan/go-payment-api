# Build stage
FROM golang:1.24.5-alpine AS builder

# Build argument for environment (default: production)
ARG BUILD_ENV=production

# Install wget for healthcheck + build dependencies
RUN apk add --no-cache git ca-certificates tzdata wget && update-ca-certificates

# Create appuser (for security)
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy go mod files first (better caching)
COPY go.mod go.sum ./

# Download dependencies (cached layer if go.mod/go.sum unchanged)
RUN go mod download && go mod verify

# Copy source code in layers (better cache invalidation)
COPY cmd/ cmd/
COPY internal/ internal/
COPY migrations/ migrations/
COPY .env* ./

# Build the binary with conditional optimizations
RUN if [ "$BUILD_ENV" = "production" ]; then \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -ldflags='-w -s -extldflags "-static"' \
            -a -installsuffix cgo \
            -o main cmd/main.go; \
    else \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -gcflags="all=-N -l" \
            -o main cmd/main.go; \
    fi

# Final stage - using alpine instead of scratch for wget
FROM alpine:latest

# Install wget for healthcheck
RUN apk --no-cache add wget ca-certificates tzdata

# Import user from builder  
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /app/main /app/main

# Copy .env files (if exist)
COPY --from=builder /app/.env* /app/

# Use non-root user
USER appuser

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Health check - HTTP endpoint kullan
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/main"]