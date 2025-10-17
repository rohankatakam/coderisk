# Production Readiness Guide

## CGO & Build Requirements

### Critical Understanding: Why CGO is Required

**Tree-sitter requires CGO** - not SQLite as initially thought.

- **Purpose**: AST parsing for 14 language parsers (JavaScript, Python, TypeScript, etc.)
- **Requirement**: C bindings for performance
- **Impact**: `CGO_ENABLED=1` required on all platforms
- **Status**: ✅ Correctly configured in Makefile and GoReleaser

### SQLite Usage (Test-Only)

- **Library**: `github.com/mattn/go-sqlite3`
- **Usage**: Only in `internal/incidents/database_test.go` for in-memory testing
- **Impact**: Zero overhead since tree-sitter already requires CGO
- **Status**: ✅ Harmless, intentionally kept

### Build Requirements by Platform

**macOS:**
```bash
# Xcode Command Line Tools (includes gcc)
xcode-select --install
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install build-essential
```

**Linux (Alpine/Docker):**
```bash
apk add gcc musl-dev
```

### Docker Build Configuration

**Current Dockerfile** (✅ Correct):
```dockerfile
# Build stage - ENABLE CGO for tree-sitter
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Install build dependencies INCLUDING gcc for CGO
RUN apk add --no-cache git ca-certificates gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build with CGO ENABLED
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath \
    -ldflags="-s -w" -o crisk ./cmd/crisk

# Runtime stage - ADD libgcc for tree-sitter
FROM alpine:latest
RUN apk --no-cache add ca-certificates git libgcc
COPY --from=builder /app/crisk /usr/local/bin/crisk
```

---

## Release Automation

### Current Workflow (Tag-Based)

**Trigger**: Git tags matching `v*` (e.g., `v0.2.0`)

**Process**:
1. GitHub Actions runs [release.yml](../../.github/workflows/release.yml)
2. GoReleaser builds binaries with CGO
3. Pushes to Docker Hub: `rohankatakam/coderisk:latest`
4. Updates Homebrew tap: `rohankatakam/homebrew-coderisk`
5. Creates GitHub Release with binaries

**Supported Platforms**:
- Linux amd64 ✅
- macOS (Homebrew builds locally with Xcode)

### Simplified Release Workflow

The release workflow has been optimized to remove redundancy:

**Removed**:
- ❌ Duplicate test job (GoReleaser runs tests)
- ❌ Empty integration test placeholder
- ❌ Announce job (no real announcements)
- ❌ Redundant artifact upload

**Result**: ~2 minutes faster releases, cleaner workflow

---

## API Key Management

### Multi-Source Priority

CodeRisk supports multiple credential sources with this priority:

1. **Environment variables** (highest priority - CI/CD)
2. **OS keychain** (recommended for local dev)
3. **Config file** (fallback)
4. **`.env` file** (development/CI)

### Keychain Support (Local Development)

**Implementation**: `zalando/go-keyring`

**Supported Platforms**:
- macOS: Keychain Access
- Windows: Credential Manager
- Linux: libsecret

**Usage**:
```bash
crisk configure  # Interactive wizard stores in keychain
```

### Environment Variables (Production/CI)

**Required Variables**:
```bash
export OPENAI_API_KEY="sk-..."
export GITHUB_TOKEN="ghp_..."  # Optional, for Layer 2/3
```

### Docker Credential Handling

⚠️ **Important**: OS keychain not available in containers.

**Docker usage requires environment variables**:

```bash
# Option 1: Direct env vars
docker run --rm \
  -e OPENAI_API_KEY \
  -e GITHUB_TOKEN \
  -v $(pwd):/repo \
  rohankatakam/coderisk:latest check

# Option 2: .env file
docker run --rm \
  --env-file .env \
  -v $(pwd):/repo \
  rohankatakam/coderisk:latest check
```

---

## Dependency Audit

### Core Dependencies (Essential)

| Dependency | Purpose | Status |
|------------|---------|--------|
| `github.com/tree-sitter/*` | AST parsing (Layer 1) | ✅ Essential (requires CGO) |
| `github.com/spf13/cobra` | CLI framework | ✅ Keep |
| `github.com/sashabaranov/go-openai` | OpenAI API (primary LLM) | ✅ Keep |
| `github.com/neo4j/neo4j-go-driver/v5` | Neo4j graph DB | ✅ Keep |
| `github.com/jackc/pgx/v5` | PostgreSQL driver | ✅ Keep |
| `github.com/redis/go-redis/v9` | Redis client | ✅ Keep |
| `github.com/google/go-github/v57` | GitHub API | ✅ Keep |
| `github.com/zalando/go-keyring` | OS keychain | ✅ Keep |

### Database Drivers (Both Used)

| Driver | Usage | Status |
|--------|-------|--------|
| `github.com/jackc/pgx/v5` | Primary storage operations | ✅ Keep |
| `github.com/lib/pq` | Staging DB with pq.Array helpers | ✅ Keep |
| `github.com/mattn/go-sqlite3` | Test-only (in-memory) | ✅ Keep |

### Caching Strategy

| Library | Use Case | Status |
|---------|----------|--------|
| `github.com/patrickmn/go-cache` | Short-lived in-memory (5-10min) | ✅ Keep |
| `github.com/redis/go-redis/v9` | Persistent cache | ✅ Keep |

**Rationale**: Different use cases - both needed.

### Optional/Heavy Dependencies

| Dependency | Purpose | Notes |
|------------|---------|-------|
| `github.com/anthropics/anthropic-sdk-go` | Claude API fallback | Heavy (AWS/GCP SDKs) but valuable for flexibility |

---

## Multi-Platform Support

### Current Setup

**Working**:
- Linux amd64 (GitHub Actions with gcc)
- macOS (Homebrew builds locally with Xcode)

**Docker**:
- Linux amd64 only (works)

### Future Expansion

For additional platforms, update `.goreleaser.yml`:

```yaml
builds:
  # Linux arm64 (requires cross-compiler)
  - id: linux-arm64
    env:
      - CGO_ENABLED=1
      - CC=aarch64-linux-gnu-gcc
    goos: [linux]
    goarch: [arm64]

  # macOS builds (requires macOS runner)
  - id: darwin
    env:
      - CGO_ENABLED=1
    goos: [darwin]
    goarch: [amd64, arm64]
```

**Note**: macOS builds require GitHub Actions macOS runners (not currently used).

---

## Development vs Production

| Aspect | Development (Local) | Production (Release) |
|--------|---------------------|----------------------|
| Build | `make build` with CGO | GoReleaser with CGO |
| API Keys | Keychain preferred | ENV vars required |
| Platform | Any OS with gcc | Linux amd64 + macOS (Homebrew) |
| Docker Services | Required (docker-compose) | Optional (can use managed services) |
| Installation | `make install-global` | Homebrew/Docker |

---

## Production Workflows

### Development Iteration

```bash
# 1. One-command setup
make dev  # Builds, starts services, installs globally

# 2. Configure API keys (keychain)
crisk configure  # Interactive wizard

# 3. Development cycle
# ... make code changes ...
make rebuild  # Quick rebuild and reinstall
crisk check test-file.go
```

### Creating a Release

```bash
# 1. Ensure tests pass
make test

# 2. Update CHANGELOG.md
# Document new features, fixes

# 3. Create git tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

# 4. GitHub Actions automatically:
#    - Runs tests
#    - Builds binaries (GoReleaser)
#    - Pushes to Docker Hub
#    - Updates Homebrew tap
#    - Creates GitHub Release
```

### End-User Installation

**Homebrew (recommended)**:
```bash
brew tap rohankatakam/coderisk
brew install coderisk
crisk configure  # Uses keychain
```

**Docker**:
```bash
docker pull rohankatakam/coderisk:latest
# Create .env file with keys
docker run --rm --env-file .env -v $(pwd):/repo rohankatakam/coderisk:latest check
```

### CI/CD Integration

```yaml
# .github/workflows/coderisk.yml
name: CodeRisk Check

on: [pull_request]

jobs:
  risk-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker://rohankatakam/coderisk:latest
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Key Learnings & Decisions

### Why CGO Cannot Be Removed

**Original assumption**: SQLite requires CGO, should be removed for static builds

**Testing revealed**: `CGO_ENABLED=0 go build` fails with:
```
github.com/tree-sitter/tree-sitter-javascript/bindings/go: build constraints exclude all Go files
github.com/tree-sitter/tree-sitter-python/bindings/go: build constraints exclude all Go files
```

**Conclusion**: Tree-sitter (core AST parsing feature) requires CGO. Current setup is optimal.

### Why Dependencies Are Not Bloated

All dependencies serve distinct purposes:
- Tree-sitter: AST parsing (requires CGO)
- SQLite: Test-only (no overhead since CGO already enabled)
- lib/pq + pgx: Different PostgreSQL use cases
- go-cache + Redis: Different caching strategies
- Anthropic SDK: User flexibility (OpenAI alternative)

### Release Workflow Optimization

**Removed redundant jobs**:
- Duplicate test execution
- Empty integration test placeholders
- Useless announcement jobs
- Redundant artifact uploads

**Result**: Cleaner, faster releases with zero functionality loss.

---

## Production Checklist

### Before Release

- [ ] All tests passing (`make test`)
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version number incremented
- [ ] Git tag created (`v*`)

### After Release

- [ ] GitHub Release created (automatic)
- [ ] Docker Hub updated (automatic)
- [ ] Homebrew tap updated (automatic)
- [ ] Test installation: `brew upgrade coderisk && crisk --version`
- [ ] Verify Docker image: `docker pull rohankatakam/coderisk:latest`

### User-Facing Validation

- [ ] Homebrew installation works
- [ ] Docker image runs correctly
- [ ] API key configuration (both keychain and env vars)
- [ ] All CLI commands functional
- [ ] Documentation accurate

---

## Troubleshooting

### Build Failures

**Error**: `build constraints exclude all Go files`
- **Cause**: CGO disabled but tree-sitter requires it
- **Fix**: Ensure `CGO_ENABLED=1` in build command

**Error**: `gcc not found`
- **Cause**: C compiler not installed
- **Fix**: Install build tools (see Build Requirements above)

### Docker Issues

**Error**: Container exits immediately
- **Cause**: Missing environment variables
- **Fix**: Pass `-e OPENAI_API_KEY` or `--env-file .env`

**Error**: `cannot connect to Neo4j`
- **Cause**: Docker services not running
- **Fix**: Run `docker compose up -d` first

### Keychain Issues

**Error**: `keyring not available`
- **Cause**: Running in Docker or unsupported environment
- **Fix**: Use environment variables instead

---

## Future Enhancements

### Potential Improvements (Post-Launch)

1. **API Key Validation**: Add health checks for expired keys
2. **`crisk doctor` Command**: Comprehensive system health check
3. **Proactive Warnings**: Alert on key expiration before failure
4. **Multi-Platform Binaries**: Add arm64, Windows support
5. **Automated Key Rotation**: Monitoring and alerts

### Low-Priority Optimizations

If binary size becomes critical (currently acceptable):

1. Replace logrus with stdlib slog (-5MB)
2. Replace sqlx with pgx native (-2MB)
3. Make Anthropic SDK optional (-15MB, 25 fewer deps)
4. Replace viper with simpler config (-3MB)

**Recommendation**: Current setup is production-ready. Optimize only if needed.

---

## References

- [Makefile Guide](./integration_guides/MAKEFILE_GUIDE.md)
- [GitHub Actions Release Workflow](../../.github/workflows/release.yml)
- [GoReleaser Configuration](../../.goreleaser.yml)
- [Docker Compose Setup](../../docker-compose.yml)
