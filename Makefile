# CodeRisk Development Makefile

# Build configuration
BINARY_NAME=crisk
BUILD_DIR=./bin
CMD_DIR=./cmd

# Microservice binaries
SERVICES=crisk-stage crisk-ingest crisk-atomize crisk-index-incident crisk-index-ownership crisk-index-coupling crisk-init

# Version info
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0-dev")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
VERSION_FLAGS=-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(GIT_TAG) -X main.BuildTime=$(BUILD_DATE)

.PHONY: help build clean test dev start stop logs status rebuild test-cli coverage lint clean-db clean-all restart init-db

## help: Show available commands
help:
	@echo 'CodeRisk Development Commands:'
	@echo ''
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/  /'

## dev: Build + start services (full development setup)
dev: clean build start init-db
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
	@echo "💡 Quick checks:"
	@echo "   ./bin/crisk --version          # Check version"
	@echo "   ./bin/crisk help               # See all commands"
	@echo ""
	@echo "📌 Binary location: ./bin/crisk (local build)"
	@echo ""

## build: Build all binaries (CLI + microservices)
build:
	@echo "🔨 Building CodeRisk binaries..."
	@mkdir -p $(BUILD_DIR)
	@echo "   Building main CLI (crisk)..."
	@CGO_ENABLED=1 go build -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/crisk
	@echo "   ✓ $(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "   Building microservices..."
	@for service in $(SERVICES); do \
		echo "   Building $$service..."; \
		CGO_ENABLED=1 go build -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$$service $(CMD_DIR)/$$service 2>&1 | grep -v "no Go files" || true; \
		if [ -f $(BUILD_DIR)/$$service ]; then \
			echo "   ✓ $(BUILD_DIR)/$$service"; \
		else \
			echo "   ✗ Failed to build $$service"; \
			exit 1; \
		fi; \
	done
	@echo ""
	@echo "✅ All binaries built successfully!"
	@echo "📌 Main CLI: $(shell pwd)/bin/crisk"
	@echo "📌 Services: $(shell pwd)/bin/{crisk-stage,crisk-ingest,crisk-atomize,...}"

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
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 8
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

## init-db: Initialize database schemas
init-db:
	@echo "🗄️  Initializing database schemas..."
	@echo "   Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "   Applying base schema (init_postgres.sql)..."
	@PGPASSWORD="$${POSTGRES_PASSWORD:-CHANGE_THIS_PASSWORD_IN_PRODUCTION_123}" \
		psql -h localhost -p $${POSTGRES_PORT_EXTERNAL:-5433} \
		-U $${POSTGRES_USER:-coderisk} -d $${POSTGRES_DB:-coderisk} \
		-f scripts/init_postgres.sql -q 2>&1 | grep -v "NOTICE" || true
	@echo "   Applying GitHub staging schema (postgresql_staging.sql)..."
	@PGPASSWORD="$${POSTGRES_PASSWORD:-CHANGE_THIS_PASSWORD_IN_PRODUCTION_123}" \
		psql -h localhost -p $${POSTGRES_PORT_EXTERNAL:-5433} \
		-U $${POSTGRES_USER:-coderisk} -d $${POSTGRES_DB:-coderisk} \
		-f scripts/schema/postgresql_staging.sql -q 2>&1 | grep -v "NOTICE" || true
	@echo "✅ Database schemas initialized"
	@echo ""

## clean-db: Reset databases (WARNING: deletes all data)
clean-db:
	@echo "⚠️  Resetting databases..."
	@echo "   Stopping containers..."
	@docker compose down
	@echo "   Removing volumes..."
	@docker volume rm coderisk_neo4j_data coderisk_postgres_data coderisk_redis_data 2>/dev/null || true
	@echo "✅ Databases reset (all data cleared)"

## clean-all: Complete cleanup (removes everything)
clean-all: clean clean-db
	@echo "🧹 Pruning Docker system..."
	@docker system prune -f --volumes
	@echo "✅ Complete cleanup done"
