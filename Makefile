# CodeRisk Development Makefile

# Build configuration
BINARY_NAME=crisk
BUILD_DIR=./bin
CMD_DIR=./cmd

# Version info
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0-dev")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
VERSION_FLAGS=-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(GIT_TAG) -X main.BuildTime=$(BUILD_DATE)

.PHONY: help build clean test dev start stop logs status rebuild test-cli coverage lint clean-db clean-all restart

## help: Show available commands
help:
	@echo 'CodeRisk Development Commands:'
	@echo ''
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/  /'

## dev: Build + start services (full development setup)
dev: clean build start
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Development environment ready!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@if command -v crisk &> /dev/null; then \
		echo "⚠️  WARNING: Found global 'crisk' at: $$(which crisk)"; \
		echo "   To avoid conflicts, always use the full path: ./bin/crisk"; \
		echo ""; \
	fi
	@echo "🚀 Test the workflow:"
	@echo "   cd /tmp"
	@echo "   git clone https://github.com/hashicorp/terraform-exec"
	@echo "   cd terraform-exec"
	@echo "   $(shell pwd)/bin/crisk init"
	@echo ""
	@echo "💡 Check version:"
	@echo "   $(shell pwd)/bin/crisk --version"
	@echo ""
	@echo "📌 Development: Use ./bin/crisk (local binary)"
	@echo "   Production:  Use crisk (global from Homebrew)"
	@echo ""

## build: Build CLI binary
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=1 go build -v -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/crisk
	@echo "✅ Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "📌 Use: $(shell pwd)/bin/crisk"

## rebuild: Quick rebuild (fast iteration)
rebuild: build
	@echo "✅ Rebuild complete"

## start: Start Docker services
start:
	@echo "🐳 Starting services..."
	@if docker ps -a --format '{{.Names}}' | grep -q 'coderisk-'; then \
		echo "⚠️  Found existing CodeRisk containers. Cleaning up..."; \
		docker rm -f coderisk-neo4j coderisk-postgres coderisk-redis 2>/dev/null || true; \
		echo "   Cleaned up old containers"; \
	fi
	@docker compose up -d
	@echo "⏳ Waiting for services to initialize..."
	@sleep 5
	@docker compose ps
	@echo ""
	@echo "✅ Services running:"
	@echo "   Neo4j:      http://localhost:7475"
	@echo "   PostgreSQL: localhost:5433"
	@echo "   Redis:      localhost:6380"
	@echo ""

## stop: Stop Docker services
stop:
	@echo "🛑 Stopping services..."
	@docker compose down
	@echo "✅ Services stopped"

## restart: Restart services
restart: stop start

## status: Show service status
status:
	@docker compose ps

## logs: Show service logs
logs:
	@docker compose logs -f

## test: Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

## test-cli: Quick test of built CLI binary
test-cli: build
	@echo "🧪 Testing CLI binary..."
	@echo ""
	@./bin/$(BINARY_NAME) --version
	@echo ""
	@./bin/$(BINARY_NAME) init --help | head -15
	@echo ""
	@echo "✅ CLI binary works!"
	@echo ""
	@echo "💡 Test graph construction:"
	@echo "   cd /tmp && git clone https://github.com/hashicorp/terraform-exec"
	@echo "   cd /tmp/terraform-exec && $(shell pwd)/bin/$(BINARY_NAME) init"
	@echo ""

## coverage: Generate test coverage
coverage:
	@echo "📊 Generating coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage: coverage.html"

## lint: Run linters
lint:
	@echo "🔍 Running linters..."
	@go fmt ./...
	@go vet ./...
	@if command -v golangci-lint &> /dev/null; then golangci-lint run; fi


## clean: Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✅ Cleaned"

## clean-db: Reset databases (WARNING: deletes all data)
clean-db:
	@echo "⚠️  Cleaning databases..."
	@docker compose down -v
	@echo "✅ Databases reset"

## clean-all: Complete cleanup
clean-all: clean stop
	@docker system prune -f
	@echo "✅ Complete cleanup done"
