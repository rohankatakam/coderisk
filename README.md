# CodeRisk

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org)

AI-powered code risk assessment that catches risky changes before they reach production.

## For Users: Cloud Platform (Recommended)

Get started in under 60 seconds with zero local setup:

```bash
# Install CLI
curl -fsSL https://coderisk.dev/install.sh | bash

# Authenticate with CodeRisk Cloud
crisk login

# Initialize repository (fetches from cloud)
crisk init owner/repo

# Run risk analysis
crisk check
```

**What you get:**
- ✅ Zero setup - no Docker, no databases
- ✅ Instant analysis - pre-built knowledge graphs
- ✅ Team features - shared insights, webhooks
- ✅ Always up-to-date - automatic graph updates

**Visit [coderisk.dev](https://coderisk.dev) to sign up**

---

## For Developers: Local Development

Want to contribute or run locally? Follow this guide.

### Prerequisites

- **Go 1.21+** with CGO enabled
- **Docker Desktop** (for Neo4j, PostgreSQL, Redis)
- **C compiler** (Xcode CLI tools on macOS, `build-essential` on Linux)

### Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/rohankatakam/coderisk-go.git
cd coderisk-go

# 2. Copy and configure environment
cp .env.example .env
# Edit .env and add your tokens:
#   GITHUB_TOKEN=ghp_...          (required)
#   OPENAI_API_KEY=sk-...         (optional, for Phase 2)

# 3. Build, start services, and install
make dev

# 4. Verify installation
crisk --version
```

### Development Workflow

```bash
# Start Docker services
make start

# Build and install globally
make build install-global

# Make code changes...

# Quick rebuild (faster iteration)
make rebuild

# Run tests
make test

# View service logs
make logs

# Stop services
make stop
```

### Testing CodeRisk Locally

```bash
# Initialize a repository (builds local knowledge graph)
crisk init owner/repo

# Check files for risk
crisk check path/to/file.go

# Check with explanation
crisk check --explain path/to/file.go
```

### Available Commands

| Command | Description |
|---------|-------------|
| `make dev` | Full setup: build + start services + install globally |
| `make build` | Build CLI binary to `./bin/crisk` |
| `make rebuild` | Quick rebuild and reinstall (fast iteration) |
| `make start` | Start Docker services (Neo4j, PostgreSQL, Redis) |
| `make stop` | Stop Docker services |
| `make status` | Check service status |
| `make logs` | View service logs |
| `make test` | Run unit tests |
| `make clean-db` | Reset databases (keeps containers) |
| `make clean-all` | Complete cleanup |
| `make help` | Show all available commands |

### Environment Variables (Local Development Only)

Create a `.env` file with these required variables:

```bash
# Required for repository ingestion
GITHUB_TOKEN=ghp_your_token_here

# Optional: Enable Phase 2 LLM-powered analysis
OPENAI_API_KEY=sk_your_key_here

# Database passwords (change for production)
NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
POSTGRES_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
```

**Note:** When using the cloud platform (`crisk login`), credentials are automatically managed - no manual token setup required.

### Project Structure

```
coderisk-go/
├── cmd/crisk/          # CLI entry point
├── internal/           # Core packages
│   ├── agent/         # Phase 2 LLM investigation
│   ├── analysis/      # Risk assessment logic
│   ├── auth/          # Cloud authentication
│   ├── graph/         # Neo4j graph operations
│   ├── ingestion/     # Repository parsing
│   └── metrics/       # Risk metrics calculation
├── Makefile           # Development commands
├── docker-compose.yml # Local infrastructure
└── .env.example       # Configuration template
```

## How It Works

**Phase 0: Pre-Filter** (<50ms)
- Security keyword detection
- Documentation-only change detection

**Phase 1: Baseline Assessment** (1-2s)
- Structural coupling analysis
- Temporal co-change patterns
- Test coverage ratio

**Phase 2: Deep Investigation** (2-5s, optional)
- LLM-guided graph navigation
- Evidence synthesis from code, git history, and incidents
- Confidence-driven investigation

## Architecture

CodeRisk uses a three-layer knowledge graph:

- **Layer 1 (Structure):** Functions, classes, imports from tree-sitter AST
- **Layer 2 (Temporal):** Git commits, co-change patterns, ownership
- **Layer 3 (Incidents):** GitHub issues, PRs, incident correlation

**Local Services:**
- Neo4j (Graph DB): Port 7475 (browser), 7688 (bolt)
- PostgreSQL (Metadata): Port 5433
- Redis (Cache): Port 6380

## Contributing

We welcome contributions!

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `make test`
5. Format code: `make fmt`
6. Commit: `git commit -m 'Add amazing feature'`
7. Push: `git push origin feature/amazing-feature`
8. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Documentation

- [Development Docs](dev_docs/) - Architecture and implementation details
- [Makefile Guide](MAKEFILE_GUIDE.md) - Complete command reference

## License

CodeRisk is open source software licensed under the [MIT License](LICENSE).

**Open Source Components:**
- ✅ CLI tool (full source code)
- ✅ Local mode with Docker Compose
- ✅ Core risk assessment engine
- ✅ Graph ingestion and analysis

**Commercial Cloud Platform:**
- ⚡ Zero-setup hosted infrastructure
- 🚀 Pre-built knowledge graphs
- 👥 Team collaboration features
- 🔒 Enterprise SSO and audit logs

Learn more at [coderisk.dev](https://coderisk.dev)

---

**Made for developers building with AI**

*CodeRisk: Ship fast, ship safely.*
