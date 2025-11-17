# CodeRisk MCP Server - Claude Code Setup Guide

**Turn any git repository into a queryable code risk database for Claude Code**

This guide will help you set up the CodeRisk MCP server to work with Claude Code, giving Claude contextual awareness of:
- **Ownership**: Who wrote and maintains each code block
- **Coupling**: Which code blocks change together
- **Temporal Risk**: Historical issues/PRs linked to code blocks

---

## Prerequisites

Before starting, ensure you have:

- [x] **Docker Desktop** installed and running
- [x] **Git** installed
- [x] **Claude Code** installed (from [claude.ai/code](https://claude.ai/code))
- [x] A GitHub repository you want to analyze (can be public or private)
- [x] GitHub Personal Access Token with `repo` scope

---

## Quick Start (5 minutes)

### Step 1: Download and Extract

```bash
# Download the latest release
curl -L https://github.com/rohankatakam/coderisk/releases/latest/download/coderisk-macos.tar.gz -o coderisk.tar.gz

# Extract
tar -xzf coderisk.tar.gz
cd coderisk

# Make binaries executable
chmod +x bin/crisk bin/crisk-check-server
```

### Step 2: Start Databases

```bash
# Start Neo4j and PostgreSQL
docker-compose up -d

# Wait for databases to be ready (~30 seconds)
sleep 30

# Verify databases are running
docker ps | grep -E "(neo4j|postgres)"
```

You should see both containers with status "Up" and "(healthy)".

### Step 3: Ingest Your Repository

```bash
# Set your GitHub token
export GITHUB_TOKEN="your_github_pat_here"
export GEMINI_API_KEY="your_gemini_api_key_here"  # Optional, for AI features

# Navigate to the repository you want to analyze
cd /path/to/your/repository

# Run ingestion (this will take 10-30 minutes depending on repo size)
/path/to/coderisk/bin/crisk init --days 365
```

**What happens during ingestion:**
1. Fetches commits, issues, and PRs from GitHub
2. Extracts code blocks from commit patches
3. Calculates ownership, coupling, and temporal risk
4. Stores everything in Neo4j and PostgreSQL

### Step 4: Connect to Claude Code

```bash
# Add the MCP server to Claude Code
claude mcp add --transport stdio coderisk \
  --env NEO4J_URI=bolt://localhost:7688 \
  --env NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  --env POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \
  -- /path/to/coderisk/bin/crisk-check-server
```

**Important**: Replace `/path/to/coderisk` with the actual path where you extracted CodeRisk.

### Step 5: Test in Claude Code

Open your repository in Claude Code:

```bash
cd /path/to/your/repository
code .
```

Then ask Claude:

```
What are the risk factors for src/main.py?
```

Claude will automatically use the `coderisk.get_risk_summary` tool to analyze the file.

---

## Detailed Setup

### Environment Variables

The MCP server requires these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NEO4J_URI` | `bolt://localhost:7688` | Neo4j connection string |
| `NEO4J_PASSWORD` | `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123` | Neo4j password |
| `POSTGRES_DSN` | `postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable` | PostgreSQL connection string |

### Multiple Repositories

To analyze multiple repositories:

1. **Ingest each repository separately**:
   ```bash
   cd /path/to/repo1
   /path/to/coderisk/bin/crisk init --days 365

   cd /path/to/repo2
   /path/to/coderisk/bin/crisk init --days 365
   ```

2. **Query any repository**: The MCP server will automatically detect which repository you're querying based on file paths.

### Custom Docker Configuration

If you need to change database ports or passwords:

1. **Edit `docker-compose.yml`**:
   ```yaml
   services:
     postgres:
       ports:
         - "5433:5432"  # Change left side for custom port
       environment:
         POSTGRES_PASSWORD: your_custom_password

     neo4j:
       ports:
         - "7688:7687"  # Change left side for custom port
       environment:
         NEO4J_AUTH: neo4j/your_custom_password
   ```

2. **Update MCP server environment variables** when adding to Claude Code.

---

## Troubleshooting

### Issue: "Failed to connect to Neo4j"

**Cause**: Neo4j container not running or not healthy

**Solution**:
```bash
# Check Neo4j status
docker ps | grep neo4j

# If not running:
cd /path/to/coderisk
docker-compose up -d neo4j

# Wait for health check
sleep 30

# Check logs if still failing
docker logs coderisk-neo4j
```

### Issue: "Failed to connect to PostgreSQL"

**Cause**: PostgreSQL container not running

**Solution**:
```bash
# Check PostgreSQL status
docker ps | grep postgres

# If not running:
cd /path/to/coderisk
docker-compose up -d postgres

# Test connection
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT version();"
```

### Issue: "No code blocks found for this file"

**Possible Causes**:
1. Repository not ingested yet
2. File not in ingested repository
3. File is new (not in git history)

**Solution**:
```bash
# Verify repository is ingested
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT id, full_name FROM github_repositories;"

# If your repo is missing, ingest it:
cd /path/to/your/repository
/path/to/coderisk/bin/crisk init --days 365
```

### Issue: Claude Code doesn't call the tool

**Solutions**:

1. **Verify MCP server is configured**:
   ```bash
   claude mcp list
   ```

   You should see `coderisk` in the list.

2. **Check MCP server status** in Claude Code:
   ```
   /mcp
   ```

3. **Restart Claude Code**:
   ```bash
   # Close Claude Code completely
   # Then restart it
   code .
   ```

4. **Check server logs**:
   ```bash
   # Server logs go to stderr, check in Claude Code terminal output
   ```

### Issue: Slow ingestion

**Cause**: Large repository or rate limiting

**Solutions**:

1. **Reduce time range**:
   ```bash
   # Ingest only last 90 days instead of 365
   /path/to/coderisk/bin/crisk init --days 90
   ```

2. **Check rate limit**:
   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://api.github.com/rate_limit
   ```

3. **Use authenticated requests**: Make sure `GITHUB_TOKEN` is set.

---

## Uninstalling

### Remove from Claude Code

```bash
claude mcp remove coderisk
```

### Stop and Remove Databases

```bash
cd /path/to/coderisk
docker-compose down -v  # -v removes volumes (deletes all data)
```

### Remove Files

```bash
rm -rf /path/to/coderisk
```

---

## Advanced Usage

### Update Analysis for a Repository

If you've made new commits or want to update the analysis:

```bash
cd /path/to/your/repository
/path/to/coderisk/bin/crisk init --days 365
```

This will incrementally update the database with new data.

### Query Specific Files

In Claude Code, you can ask about specific files:

```
What are the ownership details for src/components/Header.tsx?
```

```
Which code blocks in this file are highly coupled?
```

```
Show me historical incidents for src/api/auth.py
```

### Access Raw Data

You can query the databases directly:

**Neo4j (Graph Data)**:
```bash
# Open Neo4j Browser
open http://localhost:7475

# Login: neo4j / CHANGE_THIS_PASSWORD_IN_PRODUCTION_123

# Run queries:
MATCH (b:CodeBlock) RETURN b.name, b.original_author LIMIT 10;
```

**PostgreSQL (Tabular Data)**:
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk

# Run queries:
SELECT * FROM code_block_incidents LIMIT 10;
```

---

## What Data is Collected

CodeRisk analyzes your repository and stores:

1. **GitHub Metadata**:
   - Commits (SHA, author, message, timestamp)
   - Issues (title, state, labels, comments)
   - Pull Requests (title, state, files changed)
   - Timeline events (issue/PR references, closes)

2. **Code Structure**:
   - Code blocks (functions, classes, methods)
   - File paths and line ranges
   - Modification history

3. **Calculated Metrics**:
   - Ownership (who owns each block)
   - Coupling (which blocks change together)
   - Temporal risk (incidents linked to blocks)

**Privacy**: All data is stored locally on your machine. Nothing is sent to external servers except GitHub API calls to fetch repository data.

---

## Getting Help

- **GitHub Issues**: [https://github.com/rohankatakam/coderisk/issues](https://github.com/rohankatakam/coderisk/issues)
- **Documentation**: See `MCP_SERVER_README.md` for implementation details
- **Test Results**: See `MCP_SERVER_TEST_RESULTS.md` for test output examples

---

## System Requirements

- **OS**: macOS, Linux, or Windows (WSL)
- **Docker**: Desktop 4.0+ with 4GB RAM allocated
- **Disk Space**:
  - Small repos (<1000 commits): ~500 MB
  - Medium repos (1000-10000 commits): ~2 GB
  - Large repos (>10000 commits): ~5+ GB
- **Memory**: 8 GB RAM recommended (4 GB minimum)

---

## License

MIT License - See LICENSE file for details
