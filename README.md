# CodeRisk

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org)
[![Open Source](https://img.shields.io/badge/Open%20Source-Yes-brightgreen)](LICENSE)

AI-powered code risk assessment that catches risky changes before they reach production. Sub-5-second analysis using graph-based knowledge and LLM intelligence.

## Installation

CodeRisk supports multiple installation methods for maximum flexibility.

### Homebrew (Recommended for macOS/Linux)

```bash
# Install via Homebrew
brew tap rohankatakam/coderisk
brew install crisk

# Verify installation
crisk --version
```

### Install Script (Universal One-Liner)

Works on macOS, Linux, and Windows (WSL):

```bash
curl -fsSL https://coderisk.dev/install.sh | bash
```

The script will:
- Auto-detect your platform (OS + architecture)
- Download the latest release
- Verify checksums
- Install to `~/.local/bin/crisk`
- Prompt for OpenAI API key setup (interactive)

### Docker

For containerized workflows and CI/CD:

```bash
# Pull the image
docker pull coderisk/crisk:latest

# Run a check
docker run --rm -v $(pwd):/repo coderisk/crisk:latest check

# With API key
docker run --rm -v $(pwd):/repo \
  -e OPENAI_API_KEY="sk-..." \
  coderisk/crisk:latest check --explain
```

### Direct Download

Download pre-built binaries from [GitHub Releases](https://github.com/rohankatakam/coderisk-go/releases):

1. Download the appropriate archive for your platform
2. Extract the binary: `tar -xzf crisk_*.tar.gz`
3. Move to PATH: `mv crisk ~/.local/bin/` (or `/usr/local/bin` with sudo)
4. Make executable: `chmod +x ~/.local/bin/crisk`

### Build from Source

If you prefer to build from source:

```bash
git clone https://github.com/rohankatakam/coderisk-go.git
cd coderisk-go
go build -o crisk ./cmd/crisk
mv crisk ~/.local/bin/
```

## Quick Start

### Prerequisites

- **OpenAI API key** (REQUIRED): Get one at https://platform.openai.com/api-keys
- **Docker Desktop** (REQUIRED): For graph database (Neo4j)
- Git repository to analyze

### Setup (17 minutes one-time per repo)

```bash
# 1. Configure API key (if not done during installation)
export OPENAI_API_KEY="sk-..."
# Add to ~/.zshrc or ~/.bashrc for persistence

# 2. Start infrastructure (2 minutes)
docker compose up -d

# 3. Initialize repository (10-15 minutes)
cd /path/to/your/repo
crisk init-local
# Builds graph: Tree-sitter AST + Git history

# 4. Check for risks (2-5 seconds)
crisk check
```

**Setup time:** ~17 minutes (one-time per repo)
**Check time:** 2-5 seconds (after setup)
**Cost:** $0.03-0.05/check (BYOK)

## Usage

### Basic Commands

```bash
# Check changed files for risk
crisk check

# Check specific files with detailed explanation
crisk check --explain path/to/file.go

# Install pre-commit hook for automatic checks
crisk hook install

# View repository status
crisk status
```

### AI-Powered Analysis

CodeRisk uses LLM-guided agentic investigation for risk assessment:

```bash
# Baseline check with detailed explanation
crisk check --explain path/to/file.go

# JSON output for AI tools (Claude Code, Cursor, etc.)
crisk check --ai-mode path/to/file.go
```

> **IMPORTANT:** Both OpenAI API key AND graph database (Docker + init-local) are **REQUIRED** for CodeRisk to function.
>
> ```bash
> # Set API key
> export OPENAI_API_KEY="sk-..."  # Add to ~/.zshrc or ~/.bashrc
>
> # Start graph database
> docker compose up -d
>
> # Initialize repository
> crisk init-local
> ```

### View Graph Data

Open Neo4j Browser at http://localhost:7475

- **Username:** `neo4j`
- **Password:** `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123`

Try these queries:

```cypher
-- See all nodes by type
MATCH (n) RETURN labels(n)[0] as type, count(n) ORDER BY count DESC

-- Find high-coupling files
MATCH (f:File)-[:IMPORTS]->()
WITH f, count(*) as imports
WHERE imports > 10
RETURN f.name, imports ORDER BY imports DESC LIMIT 10
```

## How It Works

**Phase 0: Pre-Filter (<50ms)**
- Security keyword detection (auto-escalate)
- Docs-only changes (skip expensive analysis)
- Configuration file detection

**Phase 1: Baseline Assessment (1-2s)**
- Graph queries for 1-hop neighbors
- Structural coupling analysis
- Initial risk score calculation

**Phase 2: Deep Investigation (2-5s)**
- LLM-guided agentic graph navigation (3-5 hops)
- Temporal coupling detection (CO_CHANGED patterns)
- Evidence synthesis from code structure, git history, and past incidents
- Confidence-driven stopping (stops at 85%)

**Result:** <3% false positive rate (vs 10-20% industry standard)

## Configuration

### Required Environment Variables

```bash
# REQUIRED for CodeRisk to function
export OPENAI_API_KEY="sk-..."
```

Add to your shell config for persistence:
```bash
# For zsh
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.zshrc
source ~/.zshrc

# For bash
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.bashrc
source ~/.bashrc
```

### Optional Configuration

For advanced configuration:

```bash
cp .env.example .env
# Edit .env for custom ports, memory limits, etc.
```

Optional settings:
- `NEO4J_PASSWORD` - Change default database password
- Port mappings if defaults conflict (7687, 7474, 5432, 6379)
- Memory limits for Docker containers

## Development

```bash
# Build
go build -o crisk ./cmd/crisk

# Run tests
go test ./...

# Format code
go fmt ./...
```

See [dev_docs/](dev_docs/) for architecture and detailed documentation.

## License

CodeRisk is **open source** software licensed under the [MIT License](LICENSE).

### Open Source Components

The following components are freely available under the MIT License:

- ✅ **CLI tool** (`crisk` binary) - Full source code
- ✅ **Local mode** - Run CodeRisk entirely on your machine
- ✅ **Core graph engine** - AST parsing, graph ingestion
- ✅ **Phase 1 metrics** - Coupling, co-change, test coverage analysis
- ✅ **Pre-commit hooks** - Automated risk checking
- ✅ **Docker Compose stack** - Local Neo4j, PostgreSQL, Redis setup

**Use cases:**
- Individual developers and small teams
- Privacy-sensitive environments (air-gapped, on-premise)
- Learning and education
- Contributing to the core engine

### Commercial Cloud Platform

For teams wanting hosted infrastructure, advanced features, and zero setup, we offer a **commercial cloud platform**:

- ⚡ **Zero setup** - No Docker, no database management
- 🚀 **Instant access** - Pre-built graphs for popular repos (React, Next.js, etc.)
- 👥 **Team collaboration** - Shared graphs, webhooks, branch analysis
- 🔒 **Enterprise features** - SSO, audit logs, SLA guarantees
- 🎯 **ARC Database** - Access to 100+ architectural risk patterns from 10,000 real incidents
- 🤖 **Phase 2 LLM** - Advanced agentic investigation

**Pricing:** $10-50/user/month • [Learn more at coderisk.dev](https://coderisk.dev)

> **Open Core Model:** We believe in open source for developer tools. The CLI and local mode will always be free and open source. Cloud infrastructure and enterprise features are commercial to sustain development.

See [LICENSE](LICENSE) for full licensing details.

## Contributing

We welcome contributions! CodeRisk is built by developers, for developers.

**How to contribute:**
- 🐛 Report bugs via [GitHub Issues](https://github.com/rohankatakam/coderisk-go/issues)
- 💡 Suggest features or improvements
- 🔧 Submit PRs for new metrics, parsers, or bug fixes
- 📚 Improve documentation

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Community & Support

- **Documentation:** [dev_docs/](dev_docs/)
- **Issues:** [GitHub Issues](https://github.com/rohankatakam/coderisk-go/issues)
- **Discussions:** [GitHub Discussions](https://github.com/rohankatakam/coderisk-go/discussions)
- **Website:** [coderisk.dev](https://coderisk.dev)

---

**Made for developers building with AI**

*CodeRisk: Ship fast, ship safely.*
