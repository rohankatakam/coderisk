# CodeRisk-Go Makefile Guide

## Overview

The improved Makefile provides a comprehensive set of commands for local development, testing, and deployment of the CodeRisk CLI tool. This guide covers all available commands and workflows.

## Quick Start

### First Time Setup

```bash
# 1. Verify your .env file has required API keys
make verify-env

# 2. Build, start services, and install crisk globally
make dev

# 3. Verify installation
crisk --version
```

### Daily Development Workflow

```bash
# Make your code changes...

# Quick rebuild and reinstall (no cleanup)
make rebuild

# Test your changes
crisk check <file-path>
```

---

## Core Commands

### Building

| Command | Description | When to Use |
|---------|-------------|-------------|
| `make build` | Build CLI binary with CGO enabled | Standard build for testing |
| `make build-all` | Build both CLI and API server | When you need both components |
| `make rebuild` | Quick rebuild and reinstall | Fast iteration during development |

**Key Features:**
- CGO_ENABLED=1 for SQLite support
- Version info embedded (git commit, tag, build time)
- Executable permissions set automatically

### Installing

| Command | Description | Install Location |
|---------|-------------|------------------|
| `make install-global` | Install crisk globally | `/usr/local/bin` or `~/.local/bin` |
| `make install` | Install to GOPATH/bin | `$GOPATH/bin` |
| `make uninstall` | Remove all crisk installations | System-wide cleanup |

**Recommendation:** Use `make install-global` for easy global access to `crisk` command.

---

## Cleanup Commands

### Database Management

```bash
# Clean database volumes only (keeps containers)
make clean-db

# Clean Docker containers and volumes
make clean-docker

# Complete cleanup (binaries + Docker + temp files)
make clean-all

# Fresh clone state (removes ~/.coderisk/ too!)
make clean-fresh
```

### Cleanup Levels

| Command | Binaries | Docker | Volumes | User Data | Use Case |
|---------|----------|--------|---------|-----------|----------|
| `make clean` | ✅ | ❌ | ❌ | ❌ | Clean rebuild |
| `make clean-db` | ❌ | ❌ | ✅ | ❌ | Database issues |
| `make clean-docker` | ❌ | ✅ | ✅ | ❌ | Docker problems |
| `make clean-all` | ✅ | ✅ | ✅ | ❌ | Complete reset |
| `make clean-fresh` | ✅ | ✅ | ✅ | ✅ | Simulate fresh clone |

**Warning:** `make clean-fresh` removes `~/.coderisk/` - use with caution!

---

## Docker Services

### Service Management

```bash
# Start all services (Neo4j, PostgreSQL, Redis, API)
make start

# Stop all services
make stop

# Restart services
make restart

# Check service status
make status

# View logs (all services)
make logs

# View specific service logs
make logs-neo4j
make logs-postgres
```

### Service Details

When you run `make start`, the following services are started:

| Service | Port (External → Internal) | Purpose |
|---------|---------------------------|---------|
| Neo4j Browser | 7475 → 7474 | Graph database UI |
| Neo4j Bolt | 7688 → 7687 | Graph queries |
| PostgreSQL | 5433 → 5432 | Metadata storage |
| Redis | 6380 → 6379 | Caching |
| API Server | 8080 → 8080 | Health checks |

**Note:** External ports are mapped differently to avoid conflicts with existing services.

---

## Development Workflows

### Workflow 1: Quick Development (Recommended)

```bash
# One-time setup
make dev

# Make code changes...

# Quick rebuild and test
make rebuild
crisk check <file>
```

### Workflow 2: Clean Development

```bash
# Clean everything and start fresh
make clean-all
make build
make start
make install-global

# Test
crisk --version
crisk init https://github.com/user/repo
```

### Workflow 3: Database Reset

```bash
# If database has stale data
make clean-db
make start

# Re-initialize repository
crisk init https://github.com/user/repo
```

### Workflow 4: Fresh Clone Simulation

```bash
# Completely reset to fresh state
make clean-fresh

# Verify .env configuration
make verify-env

# Start development
make dev
```

---

## Testing Commands

### Unit Tests

```bash
# Run all tests
make test

# Run short tests only
make test-short

# Run tests with coverage
make coverage
```

### Integration Tests

```bash
# Run all integration tests
make test-integration

# Run specific layer tests
make test-layer2       # CO_CHANGED edge validation
make test-layer3       # CAUSED_BY edge validation

# Run performance benchmarks
make test-performance
```

---

## Configuration & Environment

### Verify Configuration

```bash
# Check .env file and required variables
make verify-env
```

This command checks:
- ✅ `.env` file exists
- ✅ `GITHUB_TOKEN` is set (required)
- ✅ `OPENAI_API_KEY` is set (optional, for Phase 2)
- ✅ `NEO4J_PASSWORD` is set
- ✅ `POSTGRES_PASSWORD` is set

### Environment Setup

```bash
# Install development tools (linters, formatters, etc.)
make setup
```

Installs:
- `golangci-lint` - Linter
- `goimports` - Import formatter
- `air` - Hot reload for development

---

## Advanced Commands

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Format + lint + test + build
make all
```

### Pre-Push Validation

```bash
# Run full OSS validation before pushing
make pre-push
```

This runs:
1. Complete cleanup (`clean-all`)
2. Download dependencies (`deps`)
3. Format code (`fmt`)
4. Run linters (`lint`)
5. Run tests (`test`)
6. Clean build (`build`)

### Release Management

```bash
# Create a release archive
make release
```

Creates: `bin/crisk-<version>-linux-amd64.tar.gz`

---

## Troubleshooting

### Build Issues

```bash
# Problem: Build fails with CGO errors
# Solution: Ensure build tools are installed
xcode-select --install  # macOS
sudo apt-get install build-essential  # Linux

# Problem: "GOPATH not set" error
# Solution: Use install-global instead
make install-global
```

### Docker Issues

```bash
# Problem: Services won't start
# Solution: Check for port conflicts
make status
netstat -an | grep -E '7475|7688|5433|6380|8080'

# Problem: Database connection errors
# Solution: Clean and restart Docker
make clean-docker
make start
```

### CLI Issues

```bash
# Problem: crisk command not found
# Solution: Ensure .local/bin is in PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Problem: Old version of crisk
# Solution: Rebuild and reinstall
make rebuild
```

---

## Configuration Files

### .env File

Located at: `coderisk-go/.env`

**Required Variables:**
- `GITHUB_TOKEN` - For fetching repository data
- `NEO4J_PASSWORD` - Neo4j authentication
- `POSTGRES_PASSWORD` - PostgreSQL authentication

**Optional Variables:**
- `OPENAI_API_KEY` - For Phase 2 LLM features
- `ANTHROPIC_API_KEY` - Alternative to OpenAI

**Note:** Never commit `.env` to git! Use `.env.example` as template.

### docker-compose.yml

The docker-compose automatically:
- Reads variables from `.env`
- Sets up health checks for all services
- Creates persistent volumes for data
- Maps ports to avoid conflicts

---

## Best Practices

### Local Development

1. **Use `make dev` for initial setup** - Handles everything in one command
2. **Use `make rebuild` for quick iterations** - Faster than full rebuild
3. **Run `make verify-env` before starting** - Catches configuration issues early
4. **Use `make clean-db` for database issues** - Faster than full cleanup

### Before Committing

```bash
# Format and lint
make fmt
make lint

# Run tests
make test

# Verify clean build
make clean build
```

### Before Pushing

```bash
# Run full validation
make pre-push

# If passing, push
git push origin main
```

---

## Global Access to crisk

After running `make install-global` or `make dev`, you can use `crisk` from anywhere:

```bash
# Initialize a repository
crisk init https://github.com/user/repo

# Check a file
crisk check path/to/file.go

# View status
crisk status

# Configure settings
crisk configure
```

---

## Summary of Key Commands

**Quick Reference:**

```bash
make dev           # 🚀 First time setup (build + start + install)
make rebuild       # ⚡ Quick iteration (build + install)
make verify-env    # 🔍 Check configuration
make clean-db      # 🗄️  Reset database
make clean-fresh   # 🔄 Complete reset (like fresh clone)
make status        # 📊 Check service status
make logs          # 📜 View service logs
crisk --version    # ✅ Verify installation
```

---

## Next Steps

After setting up with this Makefile:

1. **Verify everything works:**
   ```bash
   make verify-env
   make status
   crisk --version
   ```

2. **Test with a repository:**
   ```bash
   crisk init https://github.com/<your-repo>
   crisk check <file-path>
   ```

3. **Monitor logs if issues arise:**
   ```bash
   make logs
   ```

4. **For development:**
   - Make code changes
   - Run `make rebuild`
   - Test with `crisk` commands
   - Run `make test` before committing

---

## Support

For issues or questions:
- Check service status: `make status`
- View logs: `make logs`
- Verify configuration: `make verify-env`
- Run full help: `make help`
