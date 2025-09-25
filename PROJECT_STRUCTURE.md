# CodeRisk Go Project Structure

## Overview

This is the complete Go implementation of CodeRisk, designed for high-performance risk assessment with sub-5-second response times. The architecture supports enterprise, team, and OSS deployment models with shared ingestion capabilities.

## Directory Structure

```
coderisk-go/
├── cmd/                           # Application entrypoints
│   ├── crisk/                     # CLI application
│   │   ├── main.go               # Main CLI entry and root command
│   │   ├── init.go               # Repository initialization
│   │   ├── check.go              # Risk assessment command
│   │   ├── pull.go               # Cache synchronization
│   │   ├── status.go             # Status and health check
│   │   └── config.go             # Configuration management
│   └── crisk-server/             # API server (future)
├── internal/                      # Private application packages
│   ├── models/                   # Domain models and data structures
│   │   └── models.go             # Core data models
│   ├── github/                   # GitHub API integration
│   │   ├── client.go             # GitHub API client with rate limiting
│   │   └── extractor.go          # Repository data extraction
│   ├── risk/                     # Risk calculation engine
│   │   └── calculator.go         # Risk assessment algorithms
│   ├── storage/                  # Database abstraction layer
│   │   ├── interface.go          # Storage interface definition
│   │   ├── postgres.go           # PostgreSQL implementation
│   │   └── sqlite.go             # SQLite implementation (local)
│   ├── cache/                    # Caching and sync management
│   │   └── manager.go            # Cache operations and sync logic
│   ├── config/                   # Configuration management
│   │   └── config.go             # Configuration loading and validation
│   ├── ingestion/                # Data ingestion orchestration
│   │   └── orchestrator.go       # Ingestion workflow coordination
│   ├── api/                      # REST API handlers (future)
│   ├── webhook/                  # GitHub webhook handlers (future)
│   └── analytics/                # Usage analytics (future)
├── pkg/                          # Public reusable packages
│   ├── errors/                   # Error handling utilities
│   ├── logger/                   # Structured logging
│   └── utils/                    # General utilities
├── test/                         # Test files and utilities
│   ├── integration/              # Integration tests
│   └── fixtures/                 # Test data and fixtures
├── migrations/                   # Database migrations
├── docs/                         # Documentation
├── scripts/                      # Build and deployment scripts
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums (auto-generated)
├── Makefile                      # Build automation
├── README.md                     # Project documentation
├── .gitignore                    # Git ignore rules
└── PROJECT_STRUCTURE.md         # This file
```

## Key Components

### CLI Commands (`cmd/crisk/`)

- **main.go**: Root command setup with global flags and configuration
- **init.go**: Repository initialization with auto-discovery and connection to shared caches
- **check.go**: Fast risk assessment using cached risk sketches
- **pull.go**: Cache synchronization with remote shared caches
- **status.go**: Status reporting and health checks
- **config.go**: Configuration management and settings

### Core Packages (`internal/`)

#### Models (`internal/models/`)
- Defines all domain models: Repository, Commit, File, RiskAssessment, etc.
- Shared data structures across all components

#### GitHub Integration (`internal/github/`)
- **client.go**: Rate-limited GitHub API client with concurrent processing
- **extractor.go**: Orchestrates repository data extraction with parallel API calls

#### Risk Engine (`internal/risk/`)
- **calculator.go**: Core risk calculation algorithms
- Implements weighted scoring for blast radius, test coverage, ownership, etc.
- Generates actionable suggestions and explanations

#### Storage Layer (`internal/storage/`)
- **interface.go**: Abstract storage interface for pluggable backends
- **postgres.go**: Production PostgreSQL implementation with connection pooling
- **sqlite.go**: Local/development SQLite implementation with WAL mode

#### Cache Management (`internal/cache/`)
- **manager.go**: Handles local cache operations and remote sync
- Implements smart sync logic with configurable thresholds
- Memory and disk cache management

#### Configuration (`internal/config/`)
- **config.go**: Unified configuration using Viper
- Environment variable overrides
- Multiple deployment mode support

#### Ingestion (`internal/ingestion/`)
- **orchestrator.go**: Coordinates full repository ingestion
- Generates risk sketches from raw GitHub data
- Handles both full and incremental ingestion

## Architecture Principles

### 1. Clean Architecture
- Clear separation between domains
- Dependency injection through interfaces
- Testable components with minimal coupling

### 2. Performance First
- Concurrent processing throughout
- Smart caching with configurable TTLs
- Minimal memory allocation in hot paths

### 3. Deployment Flexibility
- Supports enterprise, team, and OSS deployment models
- Pluggable storage backends
- Environment-based configuration

### 4. Developer Experience
- Fast feedback loops (<5 seconds for most operations)
- Smart auto-discovery and zero-config setup
- Rich CLI with progress indicators and explanations

## Key Features Implemented

### 🚀 Fast Risk Assessment
- Sub-5-second risk checks using pre-computed sketches
- Level 1 (local), Level 2 (hybrid), Level 3 (cloud) analysis modes
- Smart sync with configurable freshness thresholds

### 🔗 GitHub Integration
- Complete repository ingestion with parallel processing
- Rate-limited API calls with automatic retry
- Incremental updates for efficient cache maintenance

### 💾 Flexible Storage
- Abstract interface supporting multiple backends
- PostgreSQL for production with connection pooling
- SQLite for local development and air-gapped deployments

### ⚙️ Smart Configuration
- Environment-aware configuration with sensible defaults
- Support for custom endpoints (enterprise)
- Budget controls and cost management

### 🔄 Cache Synchronization
- Git-inspired sync model (`crisk pull`)
- Shared team caches with automatic discovery
- Conflict resolution and version management

## Build and Development

### Prerequisites
- Go 1.21+
- Make (for build automation)
- Git (for development)

### Quick Start
```bash
# Clone and build
git clone <repository>
cd coderisk-go
make build

# Install locally
make install

# Run tests
make test

# Format and lint
make fmt lint

# Full CI pipeline
make ci
```

### Development Workflow
```bash
# Setup development environment
make setup

# Run with hot reload during development
make watch

# Run integration tests
make test-integration

# Build Docker image
make docker-build
```

## Next Steps for Implementation

1. **Complete CLI Implementation**: Add remaining commands and polish UX
2. **Add Go-Git Integration**: Replace placeholder git detection with actual implementation
3. **Implement Webhook Server**: Add GitHub App webhook handling
4. **Add Vector Database**: Integrate Qdrant for embedding storage
5. **Enhanced Risk Algorithms**: Implement sophisticated PageRank and temporal analysis
6. **Monitoring and Analytics**: Add usage tracking and performance metrics
7. **Enterprise Features**: SAML/OIDC, audit logging, custom endpoints

This structure provides a solid foundation for building the complete CodeRisk system while maintaining clean architecture principles and supporting all planned deployment scenarios.