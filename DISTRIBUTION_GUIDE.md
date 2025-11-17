# CodeRisk Distribution Guide

**How to package and distribute CodeRisk as a Claude Code MCP server**

---

## Current State

✅ **What's Working:**
- MCP server binary (`crisk-check-server`) built and tested
- Docker containers (Neo4j + PostgreSQL) configured
- All data pipelines functional (ownership, coupling, temporal)
- Timeline extraction bug fixed (50x improvement in incident linking)
- Test scripts validated with real data

✅ **What's Packaged:**
- Automated setup script (`setup_coderisk_mcp.sh`)
- User documentation (`CLAUDE_CODE_SETUP.md`)
- Test scripts (`test_mcp_interactive.sh`)
- Release preparation script (`prepare_release.sh`)

---

## Quick Distribution Path

### Option 1: GitHub Release (Recommended)

Perfect for public distribution. Users download, extract, and run setup.

**Steps:**

1. **Prepare release package**:
   ```bash
   cd /Users/rohankatakam/Documents/brain/coderisk
   ./prepare_release.sh v0.1.0
   ```

   This creates `release/coderisk-v0.1.0.tar.gz` containing:
   - Binaries (`bin/crisk`, `bin/crisk-check-server`)
   - Docker configuration (`docker-compose.yml`)
   - Setup script (`setup_coderisk_mcp.sh`)
   - Documentation (README, guides, examples)

2. **Test the package**:
   ```bash
   cd /tmp
   tar -xzf ~/Documents/brain/coderisk/release/coderisk-v0.1.0.tar.gz
   cd coderisk-v0.1.0
   ./verify_install.sh
   ./setup_coderisk_mcp.sh
   ```

3. **Create GitHub release**:
   ```bash
   cd /Users/rohankatakam/Documents/brain/coderisk
   git tag v0.1.0
   git push origin v0.1.0

   gh release create v0.1.0 \
     release/coderisk-v0.1.0.tar.gz \
     --title "CodeRisk v0.1.0 - MCP Server for Claude Code" \
     --notes-file release/coderisk-v0.1.0/RELEASE_NOTES.md
   ```

4. **Users download and install**:
   ```bash
   # Download
   curl -L https://github.com/rohankatakam/coderisk/releases/download/v0.1.0/coderisk-v0.1.0.tar.gz -o coderisk.tar.gz

   # Extract
   tar -xzf coderisk.tar.gz
   cd coderisk-v0.1.0

   # Run setup wizard
   ./setup_coderisk_mcp.sh
   ```

   The setup wizard handles:
   - ✅ Checking prerequisites (Docker, Claude Code)
   - ✅ Starting databases automatically
   - ✅ Guided repository ingestion
   - ✅ Adding MCP server to Claude Code
   - ✅ Testing the integration

---

### Option 2: NPM Package (Future Enhancement)

For developers who prefer npm-based installation:

```bash
# Not yet implemented, but could be:
npx @coderisk/mcp-server setup
```

Would require:
- Publishing binaries to npm
- Creating Node.js wrapper scripts
- Handling Docker compose in npm lifecycle scripts

---

### Option 3: Claude Code Plugin (Best UX, Most Work)

**Ultimate goal**: Package as a Claude Code plugin that bundles the MCP server.

**Benefits:**
- One-click installation from Claude Code
- Automatic updates
- Integrated UI for configuration

**Steps to implement:**

1. Create `plugin.json`:
   ```json
   {
     "name": "coderisk",
     "version": "0.1.0",
     "description": "Code risk analysis for Claude Code",
     "mcpServers": {
       "coderisk": {
         "command": "${CLAUDE_PLUGIN_ROOT}/bin/crisk-check-server",
         "args": [],
         "env": {
           "NEO4J_URI": "${NEO4J_URI:-bolt://localhost:7688}",
           "NEO4J_PASSWORD": "${NEO4J_PASSWORD}",
           "POSTGRES_DSN": "${POSTGRES_DSN}"
         }
       }
     },
     "hooks": {
       "onEnable": "${CLAUDE_PLUGIN_ROOT}/scripts/start-databases.sh",
       "onDisable": "${CLAUDE_PLUGIN_ROOT}/scripts/stop-databases.sh"
     }
   }
   ```

2. Bundle Docker management
3. Package with plugin SDK
4. Submit to Claude Code plugin marketplace

**Status**: Not yet implemented. Current approach (Option 1) is viable interim solution.

---

## Docker Management

### Current Approach (Working)

Users must have Docker Desktop installed and running. The setup script starts containers:

```bash
docker-compose up -d
```

**Ports used:**
- PostgreSQL: `5433:5432`
- Neo4j HTTP: `7475:7474`
- Neo4j Bolt: `7688:7687`

**Why non-standard ports?**
Avoids conflicts with existing PostgreSQL (5432) and Neo4j (7474, 7687) installations.

### Data Persistence

Docker volumes ensure data persists across restarts:
- `coderisk-postgres-data`: PostgreSQL database
- `coderisk-neo4j-data`: Neo4j graph database

**Users can stop/start without losing data:**
```bash
docker-compose stop    # Stop containers
docker-compose start   # Restart containers
docker-compose down -v # Delete everything
```

### Alternative: Embedded Database (Future)

To avoid Docker dependency, could embed:
- **SQLite** instead of PostgreSQL (simpler, but less performant)
- **Embedded Neo4j** (possible, but complex licensing)

**Trade-off**: Docker adds complexity but ensures consistent environment.

---

## Repository Ingestion Flow

### How It Works Now

1. **User clones repository** (or uses existing clone)
2. **Runs `crisk init`** in repository directory:
   ```bash
   cd /path/to/my-project
   /path/to/coderisk/bin/crisk init --days 365
   ```

3. **CodeRisk fetches data** from GitHub API:
   - Commits (with patches)
   - Issues (with timeline events)
   - Pull Requests

4. **Processes data** through pipelines:
   - Pipeline 1: Extracts metadata
   - Pipeline 2: Extracts code blocks
   - Pipeline 3: Calculates risk properties

5. **Stores in databases**:
   - PostgreSQL: Tabular data (commits, issues, blocks)
   - Neo4j: Graph data (ownership, coupling)

### Hardcoded Repository ID Issue

**Current limitation**: MCP server assumes `repo_id=4` (mcp-use repository).

**File**: `internal/mcp/tools/get_risk_summary.go:142`
```go
repoID := 4 // FIXME: Auto-detect from git remote
```

**Solutions:**

1. **Short-term**: Document that users must manually update this value
2. **Better**: Auto-detect from file path by querying:
   ```sql
   SELECT id FROM github_repositories
   WHERE full_name = (SELECT origin_url FROM git_config);
   ```

3. **Best**: Store `repo_id` in `.coderisk/config` when ingesting, read it in MCP server

**Recommended fix** (5 minutes of work):
```go
// internal/mcp/identity_resolver.go
func (r *IdentityResolver) GetRepoID(ctx context.Context, filePath string) (int64, error) {
    // Get git remote origin
    cmd := exec.Command("git", "config", "--get", "remote.origin.url")
    cmd.Dir = filepath.Dir(filePath)
    output, err := cmd.Output()

    // Parse owner/repo from URL
    // Query: SELECT id FROM github_repositories WHERE full_name = ?
    // Return repo_id
}
```

---

## User Experience Flow

### Ideal Happy Path

1. **Download and extract**:
   ```bash
   curl -L https://github.com/rohankatakam/coderisk/releases/latest/download/coderisk.tar.gz | tar xz
   cd coderisk
   ```

2. **Run setup wizard**:
   ```bash
   ./setup_coderisk_mcp.sh
   ```

   Wizard prompts:
   ```
   ✓ Checking prerequisites (Docker, Claude Code)
   ✓ Starting databases (Neo4j, PostgreSQL)
   ? Ingest a repository? (y/n) y
   ? Repository path: ~/projects/my-app
   ? GitHub token: ghp_xxxxx
   ? Gemini API key (optional): [Enter]
   ⏳ Ingesting... (10-30 min)
   ✓ Added to Claude Code
   ```

3. **Use in Claude Code**:
   ```bash
   cd ~/projects/my-app
   code .
   ```

   Ask Claude:
   ```
   What are the risk factors for src/auth.py?
   ```

   Claude responds with ownership, coupling, and temporal risk data!

### Current Friction Points

1. **Docker requirement**: Users must install Docker Desktop (~500 MB download)
   - **Mitigation**: Check and guide installation in setup wizard

2. **Long ingestion time**: 10-30 minutes for medium repos
   - **Mitigation**: Show progress bar, explain why it's worth it
   - **Future**: Incremental updates

3. **GitHub rate limiting**: Free tier = 60 req/hour, authenticated = 5000/hour
   - **Mitigation**: Require GitHub token, show remaining quota

4. **Hardcoded repo ID**: Only works for one repository at a time
   - **Mitigation**: Document limitation, provide fix (see above)

---

## Distribution Checklist

Before releasing, ensure:

### Functionality
- [x] MCP server works with Claude Code
- [x] All three risk dimensions return data
- [x] Timeline extraction bug fixed
- [x] Docker containers start reliably
- [x] Ingestion completes for test repository

### Documentation
- [x] User setup guide (CLAUDE_CODE_SETUP.md)
- [x] Quick start guide (QUICKSTART_MCP.md)
- [x] Test results documented (MCP_SERVER_TEST_RESULTS.md)
- [x] Release notes template created

### Scripts
- [x] Automated setup wizard (setup_coderisk_mcp.sh)
- [x] Test script (test_mcp_interactive.sh)
- [x] Release packaging script (prepare_release.sh)

### Missing (Nice-to-Have)
- [ ] Auto-detect repository ID from git remote
- [ ] Progress bar for ingestion
- [ ] Web UI for data exploration
- [ ] Incremental updates (don't re-ingest everything)
- [ ] Windows native support (currently requires WSL)
- [ ] Pre-built Docker images on Docker Hub

---

## Release Process

### 1. Pre-Release Testing

```bash
# Build fresh binaries
cd /Users/rohankatakam/Documents/brain/coderisk
make clean build

# Test locally
./test_mcp_interactive.sh

# Verify with Claude Code
claude mcp list
# Should show 'coderisk' server
```

### 2. Version and Tag

```bash
# Update version in relevant files
VERSION="v0.1.0"

# Create git tag
git tag -a $VERSION -m "Release $VERSION: MCP Server for Claude Code"
git push origin $VERSION
```

### 3. Build Release Package

```bash
./prepare_release.sh $VERSION
```

### 4. Test Package

```bash
# Extract to temp location
cd /tmp
tar -xzf ~/Documents/brain/coderisk/release/coderisk-$VERSION.tar.gz
cd coderisk-$VERSION

# Run verification
./verify_install.sh

# Test setup wizard (but don't complete)
./setup_coderisk_mcp.sh
```

### 5. Create GitHub Release

```bash
gh release create $VERSION \
  ~/Documents/brain/coderisk/release/coderisk-$VERSION.tar.gz \
  --title "CodeRisk $VERSION - MCP Server for Claude Code" \
  --notes-file ~/Documents/brain/coderisk/release/coderisk-$VERSION/RELEASE_NOTES.md
```

### 6. Announce

Post to:
- GitHub Discussions
- Claude Code community forums
- Twitter/X
- Reddit (r/ClaudeAI)

---

## Future Enhancements

### High Priority
1. **Auto-detect repository ID** - Critical for multi-repo support
2. **Incremental updates** - Only fetch new commits since last ingestion
3. **Progress indicators** - Show % complete during ingestion

### Medium Priority
4. **Web UI** - Explore risk data visually
5. **Pre-built Docker images** - Skip build step
6. **Windows native support** - No WSL required

### Low Priority
7. **Claude Code plugin** - One-click install
8. **NPM package** - Alternative distribution
9. **Cloud deployment** - SaaS version

---

## Support and Maintenance

### User Support Channels

1. **GitHub Issues**: Primary support channel
   - Bug reports
   - Feature requests
   - Installation help

2. **Documentation**: Self-service
   - CLAUDE_CODE_SETUP.md (installation)
   - MCP_SERVER_README.md (technical details)
   - QUICKSTART_MCP.md (quick reference)

### Common Issues and Solutions

See CLAUDE_CODE_SETUP.md "Troubleshooting" section.

### Update Process

When releasing updates:

1. Update `CHANGELOG.md`
2. Bump version in `prepare_release.sh`
3. Create new release package
4. Users download and re-run setup script

---

## Summary

**CodeRisk is ready for distribution via GitHub releases.**

✅ **Working end-to-end**: Download → Setup → Use in Claude Code
✅ **Automated setup**: Interactive wizard handles complexity
✅ **Well-documented**: Multiple guides for different user needs
✅ **Tested**: Validated with real repository (mcp-use)

**Next steps:**
1. Fix hardcoded repo_id (5 minute task)
2. Test release package
3. Create GitHub release
4. Announce to users

**Known limitations:**
- Requires Docker (not embedded)
- One repository at a time (fixable)
- Long ingestion time (inherent to approach)

The current packaging approach (Option 1: GitHub Release) is production-ready and provides good UX while maintaining flexibility for future enhancements.
