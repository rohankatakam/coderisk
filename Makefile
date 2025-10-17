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

.PHONY: help build clean test dev start stop logs status install

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

## install: Install binary globally (optional, for testing prod experience)
install: build
	@echo "📦 Installing to /usr/local/bin..."
	@echo "⚠️  This requires sudo password"
	@echo "⚠️  WARNING: This will overwrite any Homebrew-installed crisk"
	@echo ""
	@read -p "Continue? (y/N) " confirm && [ "$$confirm" = "y" ] || exit 1
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed globally"

## rebuild: Quick rebuild (fast iteration, no install needed)
rebuild: build
	@echo "✅ Rebuild complete - use ./bin/crisk"

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
	@echo "   cd /tmp/terraform-exec && $(BINARY_NAME) init"
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

## uninstall: Remove any globally installed crisk (including old dev versions)
uninstall:
	@echo "🗑️  Removing globally installed crisk..."
	@sudo rm -f /usr/local/bin/crisk /usr/local/bin/crisk-dev
	@rm -f ~/.local/bin/crisk ~/.local/bin/crisk-dev 2>/dev/null || true
	@echo "✅ Removed all global installations"
	@echo ""
	@echo "Note: Use 'make dev' to build local binary at ./bin/crisk"

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
