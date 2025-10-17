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

# Build + start services + install (requires sudo password)
make dev

# Verify installation
crisk-dev --version
```

**Note:** Development uses `crisk-dev` command to avoid conflicts with production `crisk` from Homebrew.

### Daily Development

```bash
# Make code changes...

# Quick rebuild (requires sudo password)
make rebuild

# Test
crisk-dev --version
make test
```

### Testing Graph Construction

```bash
# Clone a test repository
cd /tmp
git clone https://github.com/hashicorp/terraform-exec
cd terraform-exec

# Run crisk-dev init (auto-detects from git remote)
crisk-dev init

# Verify in Neo4j browser
open http://localhost:7475
```

### Available Commands

```bash
make help       # Show all commands
make dev        # Full development setup
make rebuild    # Fast rebuild + install
make build      # Build binary only
make test-cli   # Test built binary
make start      # Start Docker services
make stop       # Stop services
make test       # Run unit tests
make lint       # Run linters
make uninstall  # Remove crisk-dev
make clean-db   # Reset databases
```

### Dev vs Production Separation

| Command | Binary | Use Case |
|---------|--------|----------|
| `crisk-dev` | Development | From `make dev` or `make rebuild` |
| `crisk` | Production | From `brew install rohankatakam/coderisk/crisk` |

**No conflicts:** You can have both installed simultaneously!

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
