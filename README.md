# CodeRisk

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://golang.org)

AI-powered code risk assessment that catches risky changes before they reach production.

> **Note:** This project is in active development. We're building robust core features before a public release.

---

## Development Workflow

### Prerequisites

- **Go 1.23+** with CGO enabled
- **Docker** (for Neo4j, PostgreSQL, Redis)
- **C compiler** (Xcode CLI tools on macOS, `build-essential` on Linux)

### Quick Start

```bash
# Clone repository
git clone https://github.com/rohankatakam/coderisk.git
cd coderisk

# Configure environment
cp .env.example .env
# Edit .env and set GITHUB_TOKEN

# Build + start services + install
make dev

# Verify installation
crisk --version
```

### Daily Development

```bash
# Start services
make start

# Make code changes...

# Quick rebuild
make rebuild

# Run tests
make test

# View logs
make logs
```

### Available Commands

```bash
make help      # Show all commands
make dev       # Full development setup
make build     # Build CLI binary
make rebuild   # Fast rebuild + reinstall
make start     # Start Docker services
make stop      # Stop services
make test      # Run tests
make lint      # Run linters
make clean-db  # Reset databases
```

### Environment Setup

Required variables in [.env](.env):

```bash
# Required
GITHUB_TOKEN=ghp_your_token_here

# Optional (for Phase 2 features)
OPENAI_API_KEY=sk_your_key_here
```

Defaults are provided for database passwords and ports.

---

## Project Structure

```
coderisk/
├── cmd/crisk/          # CLI entry point
├── internal/           # Core packages
│   ├── graph/         # Neo4j operations
│   ├── ingestion/     # Repository parsing
│   ├── analysis/      # Risk assessment
│   └── auth/          # Authentication
├── dev_docs/          # Architecture & design docs
├── Makefile           # Development commands
└── docker-compose.yml # Local services
```

## Architecture

**Three-layer knowledge graph:**
- **Layer 1:** Code structure (AST, dependencies)
- **Layer 2:** Git history (commits, co-changes)
- **Layer 3:** Incidents (issues, PRs)

**Local services:**
- Neo4j: `localhost:7475` (browser), `localhost:7688` (bolt)
- PostgreSQL: `localhost:5433`
- Redis: `localhost:6380`

---

## Documentation

- [dev_docs/](dev_docs/) - Architecture and implementation details
- [DEVELOPMENT_WORKFLOW.md](dev_docs/DEVELOPMENT_WORKFLOW.md) - Development guidelines
- [DEPLOYMENT_STRATEGY.md](dev_docs/03-implementation/DEPLOYMENT_STRATEGY.md) - Release process

## License

MIT License - see [LICENSE](LICENSE)
