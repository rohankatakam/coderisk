# Makefile for CodeRisk Go implementation

# Variables
BINARY_NAME=crisk
SERVER_NAME=crisk-server
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"
BUILD_DIR=./bin
CMD_DIR=./cmd
COVERAGE_FILE=coverage.out

# Git variables
GIT_COMMIT=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "untagged")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags with version info
VERSION_FLAGS=-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(GIT_TAG) -X main.BuildTime=$(BUILD_DATE)

.PHONY: all build clean test coverage lint fmt run help install docker migrate \
	build-cli build-api build-all clean-db clean-docker clean-all clean-fresh \
	install-global uninstall start stop restart status logs dev rebuild verify-env

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## all: Build everything
all: clean fmt lint test build

## build: Build the crisk CLI binary (recommended for testing)
build: build-cli
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Build complete!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "📦 Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "🚀 Next steps:"
	@echo "   1. Start services:  make start"
	@echo "   2. Run crisk:       $(BUILD_DIR)/$(BINARY_NAME) init owner/repo"
	@echo ""
	@echo "📋 Useful commands:"
	@echo "   make start    - Start Docker services (required)"
	@echo "   make stop     - Stop Docker services"
	@echo "   make status   - Check service status"
	@echo "   make logs     - View service logs"
	@echo "   make test     - Run unit tests"
	@echo ""

## build-cli: Build the CLI binary with version info (with CGO enabled for SQLite)
build-cli:
	@echo "🔨 Building crisk CLI..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=1 $(GO) build $(GOFLAGS) -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/crisk
	@echo "   ✓ CLI: $(BUILD_DIR)/$(BINARY_NAME)"
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)

## build-api: Build the API server binary (optional - for health monitoring)
build-api:
	@echo "🔨 Building API server..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/crisk-api $(CMD_DIR)/api
	@echo "   ✓ API: $(BUILD_DIR)/crisk-api"

## build-all: Build both CLI and API server
build-all: build-cli build-api
	@echo ""
	@echo "✅ All binaries built successfully"
	@echo "   - CLI: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "   - API: $(BUILD_DIR)/crisk-api"

## clean: Remove build artifacts and binaries
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME) $(SERVER_NAME)
	@rm -f $(COVERAGE_FILE)
	@echo "✅ Build artifacts cleaned"

## clean-db: Clean database volumes only (preserves containers)
clean-db:
	@echo "🗄️  Cleaning database volumes..."
	@echo "   ⚠️  WARNING: This will delete all stored data!"
	@echo "   Stopping containers..."
	@docker compose down 2>/dev/null || true
	@echo "   Removing volumes..."
	@docker volume rm -f coderisk-go_neo4j_data coderisk-go_postgres_data coderisk-go_redis_data 2>/dev/null || true
	@docker volume rm -f coderisk_neo4j_data coderisk_postgres_data coderisk_redis_data 2>/dev/null || true
	@rm -rf ./volumes/neo4j_data ./volumes/postgres_data ./volumes/redis_data 2>/dev/null || true
	@echo "   Pruning dangling volumes..."
	@docker volume prune -f 2>/dev/null || true
	@echo "✅ Database volumes cleaned"
	@echo ""
	@echo "💡 Run 'make start' to recreate databases with fresh data"

## clean-docker: Stop and remove Docker containers and volumes
clean-docker:
	@echo "🐳 Cleaning Docker environment..."
	@echo "   Stopping containers..."
	@docker compose down 2>/dev/null || true
	@echo "   Removing containers..."
	@docker rm -f coderisk-neo4j coderisk-postgres coderisk-redis coderisk-api 2>/dev/null || true
	@echo "   Removing volumes..."
	@docker volume rm -f coderisk-go_neo4j_data coderisk-go_postgres_data coderisk-go_redis_data 2>/dev/null || true
	@docker volume rm -f coderisk_neo4j_data coderisk_postgres_data coderisk_redis_data 2>/dev/null || true
	@rm -rf ./volumes/ 2>/dev/null || true
	@echo "   Pruning dangling volumes..."
	@docker volume prune -f 2>/dev/null || true
	@echo "   Removing dangling images..."
	@docker image prune -f 2>/dev/null || true
	@echo "✅ Docker environment cleaned"

## clean-all: Complete cleanup (binaries + Docker + temp files)
clean-all: clean clean-docker
	@echo "🧹 Deep cleaning repository..."
	@rm -f $(COVERAGE_FILE) coverage.html
	@rm -rf vendor/
	@rm -rf tmp/ temp/ local/ .cache/ .scratch/
	@rm -f *.log *.tmp *.test *.prof *.pprof
	@rm -f cpu.prof mem.prof profile.out
	@echo "✨ Complete cleanup finished!"
	@echo ""
	@echo "📋 What was cleaned:"
	@echo "   ✓ Build artifacts and binaries"
	@echo "   ✓ Docker containers and volumes"
	@echo "   ✓ Test coverage files"
	@echo "   ✓ Temporary and cache directories"
	@echo "   ✓ Log and profile files"
	@echo ""
	@echo "📌 Preserved:"
	@echo "   ✓ .env (configuration)"
	@echo "   ✓ go.mod and go.sum (dependencies)"
	@echo "   ✓ ~/.coderisk/ (user data)"
	@echo ""
	@echo "🚀 Ready for fresh start!"

## clean-fresh: Complete reset (as if freshly cloned) - removes user data too
clean-fresh: clean-all
	@echo "🔄 Fresh clone simulation - removing ALL local state..."
	@echo "   ⚠️  WARNING: This includes ~/.coderisk/ user data!"
	@rm -rf ~/.coderisk/ 2>/dev/null || true
	@rm -rf test_sandbox/ 2>/dev/null || true
	@echo ""
	@echo "✨ Repository reset to fresh clone state!"
	@echo ""
	@echo "📋 Next steps:"
	@echo "   1. Verify .env has your API keys"
	@echo "   2. Run 'make dev' to build and start services"
	@echo "   3. Run 'bin/crisk init owner/repo' to initialize a repository"

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## test-short: Run short tests only
test-short:
	@echo "Running short tests..."
	$(GO) test -short -v ./...

## test-integration: Run all integration tests (Layer 2, Layer 3, Performance)
test-integration: test-layer2 test-layer3 test-performance
	@echo "✅ All integration tests complete"

## test-layer2: Validate CO_CHANGED edge creation (Layer 2)
test-layer2:
	@echo "Running Layer 2 (CO_CHANGED) validation..."
	@./test/integration/test_layer2_validation.sh

## test-layer3: Validate CAUSED_BY edge creation (Layer 3)
test-layer3:
	@echo "Running Layer 3 (CAUSED_BY) validation..."
	@./test/integration/test_layer3_validation.sh

## test-performance: Run performance benchmarks
test-performance:
	@echo "Running performance benchmarks..."
	@./test/integration/test_performance_benchmarks.sh

## coverage: Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint &> /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		$(GO) vet ./...; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports &> /dev/null; then \
		goimports -w .; \
	fi

## install: Install binaries to GOPATH/bin (requires GOPATH)
install: build
	@echo "Installing to GOPATH/bin..."
	@if [ -z "$(GOPATH)" ]; then \
		echo "❌ GOPATH not set. Use 'make install-global' instead."; \
		exit 1; \
	fi
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"
	@echo ""
	@echo "💡 Make sure $(GOPATH)/bin is in your PATH"

## install-global: Install crisk globally for easy access
install-global: build
	@echo "🌐 Installing crisk globally..."
	@if [ -d "/usr/local/bin" ]; then \
		sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME); \
		sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
		echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"; \
	elif [ -d "$$HOME/.local/bin" ]; then \
		mkdir -p $$HOME/.local/bin; \
		cp $(BUILD_DIR)/$(BINARY_NAME) $$HOME/.local/bin/$(BINARY_NAME); \
		chmod +x $$HOME/.local/bin/$(BINARY_NAME); \
		echo "✅ Installed to $$HOME/.local/bin/$(BINARY_NAME)"; \
		echo ""; \
		echo "💡 Make sure $$HOME/.local/bin is in your PATH:"; \
		echo "   export PATH=\"$$HOME/.local/bin:$$PATH\""; \
	else \
		echo "❌ Neither /usr/local/bin nor ~/.local/bin exists"; \
		echo "   Create one of these directories first"; \
		exit 1; \
	fi
	@echo ""
	@echo "✅ Run 'crisk --version' to verify installation"

## uninstall: Remove globally installed crisk
uninstall:
	@echo "🗑️  Uninstalling crisk..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	@rm -f $$HOME/.local/bin/$(BINARY_NAME) 2>/dev/null || true
	@rm -f $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || true
	@echo "✅ crisk uninstalled"

## run-cli: Run the CLI directly (without building)
run-cli:
	@echo "🚀 Running CLI..."
	$(GO) run $(CMD_DIR)/crisk/... $(ARGS)

## serve: Start the API health check server
serve: build-api
	@echo ""
	@echo "🌐 Starting API server on http://localhost:8080"
	@echo ""
	@echo "   Health check: curl http://localhost:8080/health"
	@echo "   Stop server: Ctrl+C"
	@echo ""
	@$(BUILD_DIR)/crisk-api

## serve-dev: Run API server in development mode (hot reload)
serve-dev:
	@echo "🌐 Starting API server in dev mode..."
	$(GO) run $(CMD_DIR)/api/...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## update: Update dependencies
update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

## migrate: Run database migrations
migrate:
	@echo "Running migrations..."
	$(GO) run ./migrations/migrate.go up

## migrate-down: Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	$(GO) run ./migrations/migrate.go down

## start: Start Docker services (Neo4j, PostgreSQL, Redis) - builds containers if needed
start:
	@echo "🐳 Starting CodeRisk services..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found!"; \
		echo "   Copy .env.example to .env and configure your API keys:"; \
		echo "   cp .env.example .env"; \
		exit 1; \
	fi
	@docker compose up -d --build
	@echo ""
	@echo "⏳ Waiting for services to initialize..."
	@sleep 8
	@echo ""
	@docker compose ps
	@echo ""
	@echo "✅ Services started!"
	@echo ""
	@echo "📋 Service URLs:"
	@echo "   Neo4j Browser: http://localhost:7475 (user: neo4j, pass: check .env)"
	@echo "   Neo4j Bolt:    bolt://localhost:7688"
	@echo "   PostgreSQL:    localhost:5433"
	@echo "   Redis:         localhost:6380"
	@echo "   API Server:    http://localhost:8080/health"
	@echo ""
	@echo "🚀 Ready to use crisk:"
	@echo "   $(BUILD_DIR)/$(BINARY_NAME) init owner/repo"
	@echo ""

## stop: Stop Docker services
stop:
	@echo "🛑 Stopping CodeRisk services..."
	@docker compose down
	@echo "✅ Services stopped"

## restart: Restart Docker services
restart: stop start

## status: Show status of Docker services
status:
	@echo "📊 CodeRisk Service Status:"
	@echo ""
	@docker compose ps
	@echo ""
	@echo "💾 Docker volumes:"
	@docker volume ls | grep -E "coderisk|VOLUME" || echo "   No CodeRisk volumes found"

## logs: Show logs from Docker services
logs:
	@echo "📜 Service logs (press Ctrl+C to exit):"
	@docker compose logs -f

## logs-neo4j: Show Neo4j logs only
logs-neo4j:
	@docker compose logs -f neo4j

## logs-postgres: Show PostgreSQL logs only
logs-postgres:
	@docker compose logs -f postgres

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t coderisk:$(GIT_TAG) -t coderisk:latest .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -it --rm \
		-v ~/.coderisk:/root/.coderisk \
		-e GITHUB_TOKEN \
		-e OPENAI_API_KEY \
		coderisk:latest

## proto: Generate protobuf code (if using gRPC)
proto:
	@echo "Generating protobuf code..."
	@if [ -d "./proto" ]; then \
		protoc --go_out=. --go-grpc_out=. proto/*.proto; \
	else \
		echo "No proto directory found"; \
	fi

## vendor: Create vendor directory
vendor:
	@echo "Creating vendor directory..."
	$(GO) mod vendor

## ci: Run CI pipeline locally
ci: deps fmt lint test build

## pre-push: Validate code before pushing (OSS workflow)
pre-push: clean-all deps fmt lint test build
	@echo "🔍 Pre-push validation completed successfully!"
	@echo "✅ Code formatted and linted"
	@echo "✅ All tests passed"
	@echo "✅ Clean build successful"
	@echo "✅ Repository cleaned for OSS contribution"
	@echo ""
	@echo "🚀 Ready to push! Your code follows OSS best practices."

## release: Create a new release
release: clean test build
	@echo "Creating release..."
	@echo "Version: $(GIT_TAG)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Date: $(BUILD_DATE)"
	@tar -czf $(BUILD_DIR)/$(BINARY_NAME)-$(GIT_TAG)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)
	@echo "Release archive created: $(BUILD_DIR)/$(BINARY_NAME)-$(GIT_TAG)-linux-amd64.tar.gz"

# Development helpers

## dev: Quick development workflow (clean build + start services + install globally)
dev: clean build start install-global
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🎉 Development environment ready!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "✅ crisk CLI built and installed globally"
	@echo "✅ Docker services running"
	@echo ""
	@echo "🚀 Quick start:"
	@echo "   crisk --version           # Verify installation"
	@echo "   crisk init owner/repo     # Initialize a repository"
	@echo "   crisk check <file>        # Check a file for risks"
	@echo ""
	@echo "📋 Useful commands:"
	@echo "   make logs                 # View service logs"
	@echo "   make status               # Check service status"
	@echo "   make stop                 # Stop services"
	@echo "   make clean-db             # Clean database (keep containers)"
	@echo ""

## rebuild: Quick rebuild and reinstall (no clean, faster for iteration)
rebuild: build install-global
	@echo ""
	@echo "✅ Rebuild complete - crisk updated globally"
	@echo "   Run: crisk --version"

## watch: Watch for changes and rebuild
watch:
	@if command -v air &> /dev/null; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
	fi

## setup: Setup development environment
setup:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/cosmtrek/air@latest
	@echo "Development tools installed"

## verify-env: Verify .env configuration
verify-env:
	@echo "🔍 Verifying .env configuration..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found"; \
		echo "   Run: cp .env.example .env"; \
		exit 1; \
	fi
	@echo "✅ .env file exists"
	@echo ""
	@echo "📋 Checking required variables..."
	@grep -q "GITHUB_TOKEN=.*[a-zA-Z0-9]" .env && echo "   ✅ GITHUB_TOKEN set" || echo "   ❌ GITHUB_TOKEN missing or empty"
	@grep -q "OPENAI_API_KEY=.*[a-zA-Z0-9]" .env && echo "   ✅ OPENAI_API_KEY set" || echo "   ⚠️  OPENAI_API_KEY empty (Phase 2 features disabled)"
	@grep -q "NEO4J_PASSWORD=" .env && echo "   ✅ NEO4J_PASSWORD set" || echo "   ❌ NEO4J_PASSWORD missing"
	@grep -q "POSTGRES_PASSWORD=" .env && echo "   ✅ POSTGRES_PASSWORD set" || echo "   ❌ POSTGRES_PASSWORD missing"
	@echo ""