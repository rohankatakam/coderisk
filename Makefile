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

## dev: Build + start services + install (full development setup)
dev: clean build start install
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Development environment ready!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Run: crisk --version"
	@echo ""

## build: Build CLI binary
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=1 go build -v -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/crisk
	@echo "✅ Binary: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Install binary globally
install: build
	@echo "📦 Installing globally..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

## rebuild: Quick rebuild and install (fast iteration)
rebuild: build install
	@echo "✅ Rebuild complete"

## start: Start Docker services
start:
	@echo "🐳 Starting services..."
	@docker compose up -d
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
