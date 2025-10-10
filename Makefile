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

.PHONY: all build clean test coverage lint fmt run help install docker migrate

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## all: Build everything
all: clean fmt lint test build

## build: Build the CLI and server binaries
build: build-cli build-server

## build-cli: Build the CLI binary with version info
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(VERSION_FLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/crisk

## build-server: Build the server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "$(CMD_DIR)/crisk-server/main.go" ]; then \
		$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_NAME) $(CMD_DIR)/crisk-server; \
	else \
		echo "⚠️  Server not implemented yet - skipping"; \
	fi

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)

## clean-all: Deep clean for fresh repository state (OSS friendly)
clean-all:
	@echo "🧹 Deep cleaning repository..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	@rm -rf vendor/
	@rm -f go.sum
	@rm -rf tmp/ temp/ local/ .cache/ .scratch/
	@rm -f *.log *.tmp *.test *.prof *.pprof
	@rm -rf ~/.coderisk/ # Remove local development cache
	@rm -f cpu.prof mem.prof profile.out
	@echo "✨ Repository cleaned for fresh state - ready for OSS contribution!"

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

## install: Install binaries to GOPATH/bin
install: build
	@echo "Installing..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## run-cli: Run the CLI directly
run-cli:
	@echo "Running CLI..."
	$(GO) run $(CMD_DIR)/crisk/... $(ARGS)

## run-server: Run the server directly
run-server:
	@echo "Running server..."
	$(GO) run $(CMD_DIR)/crisk-server/...

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

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t coderisk:$(GIT_TAG) -t coderisk:latest .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -it --rm \
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