# Dockerfile for CodeRisk Development
# Note: This is for LOCAL DEVELOPMENT ONLY
# Production builds use GoReleaser with .goreleaser.yml

FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO for tree-sitter
RUN CGO_ENABLED=1 go build -trimpath \
    -ldflags="-s -w" \
    -o crisk ./cmd/crisk

# Runtime stage
FROM alpine:latest

# Runtime dependencies
RUN apk --no-cache add ca-certificates git

# Copy binary
COPY --from=builder /app/crisk /usr/local/bin/crisk
RUN chmod +x /usr/local/bin/crisk

WORKDIR /repo

ENTRYPOINT ["crisk"]
CMD ["--help"]
