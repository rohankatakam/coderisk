# Dockerfile for CodeRisk CLI
# Multi-stage build for minimal image size

# Build stage (not used by GoReleaser, but useful for local builds)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.Version=dev -X main.GitCommit=local -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o crisk ./cmd/crisk

# Runtime stage
FROM alpine:latest

LABEL org.opencontainers.image.title="CodeRisk CLI" \
      org.opencontainers.image.description="Lightning-fast AI-powered code risk assessment" \
      org.opencontainers.image.vendor="CodeRisk" \
      org.opencontainers.image.url="https://coderisk.dev" \
      org.opencontainers.image.documentation="https://docs.coderisk.dev" \
      org.opencontainers.image.source="https://github.com/rohankatakam/coderisk-go" \
      org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    git \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 crisk && \
    adduser -D -u 1000 -G crisk crisk

# Copy binary from builder (or from GoReleaser)
COPY crisk /usr/local/bin/crisk

# Make binary executable
RUN chmod +x /usr/local/bin/crisk

# Switch to non-root user
USER crisk

# Set working directory
WORKDIR /repo

# Entrypoint
ENTRYPOINT ["crisk"]

# Default command (show help)
CMD ["--help"]

# Usage examples:
# docker build -t coderisk/crisk:local .
# docker run --rm coderisk/crisk:local --version
# docker run --rm -v $(pwd):/repo coderisk/crisk:local check
# docker run --rm -v $(pwd):/repo -e OPENAI_API_KEY="sk-..." coderisk/crisk:local check --explain
