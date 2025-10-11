# CodeRisk

AI-powered code risk assessment that catches risky changes before they reach production. Sub-5-second analysis using graph-based knowledge and LLM intelligence.

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Git repository to analyze

### Setup

```bash
# Clone and install
git clone https://github.com/rohankatakam/coderisk-go.git
cd coderisk-go
./install.sh  # Will prompt for OpenAI API key (optional but recommended)

# Start infrastructure (Neo4j, PostgreSQL, Redis)
docker compose up -d

# Initialize on your repository
cd /path/to/your/repo
crisk init-local
```

> **Note:** The installer will ask if you want to set up your OpenAI API key. This enables Phase 2 deep risk investigation. You can skip this and add it later if needed.

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

For deep LLM investigation of high-risk changes:

```bash
# Phase 2 with detailed explanation
crisk check --explain path/to/file.go

# JSON output for AI tools (Claude Code, Cursor, etc.)
crisk check --ai-mode path/to/file.go
```

> **Note:** Requires `OPENAI_API_KEY` environment variable. Set during installation or add manually:
> ```bash
> export OPENAI_API_KEY="sk-..."  # Add to ~/.zshrc or ~/.bashrc
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

**Phase 1: Fast Baseline (200ms)**
- Structural coupling analysis
- Temporal co-change detection
- Test coverage checks

**Phase 2: LLM Investigation (3-5s)**
- Agentic graph navigation
- Evidence synthesis from code structure, git history, and past incidents
- Actionable recommendations

Result: <3% false positive rate, intelligent risk detection

## Configuration

Environment variables are optional. For advanced configuration:

```bash
cp .env.example .env
# Edit .env for custom ports, memory limits, or API keys
```

Key options:
- `OPENAI_API_KEY` - For Phase 2 LLM analysis
- `NEO4J_PASSWORD` - Change default database password
- Port mappings if defaults conflict

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

MIT License - see [LICENSE](LICENSE) for details.

---

**Made for developers building with AI**

*CodeRisk: Ship fast, ship safely.*
