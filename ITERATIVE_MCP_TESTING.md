# Iterative MCP Server Testing Guide

## Overview

This guide covers the complete workflow for iteratively developing, building, and testing the CodeRisk MCP server with Claude Code.

---

## Prerequisites

### 1. Databases Running

**Neo4j** (port 7688):
```bash
docker ps | grep neo4j
# Should show neo4j container running on port 7688
```

**PostgreSQL** (port 5433):
```bash
docker ps | grep postgres
# Should show postgres container running on port 5433
```

**Verify data exists:**
```bash
# Neo4j - Check CodeBlock nodes
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (b:CodeBlock) RETURN count(b) as count"}]}' | python3 -m json.tool

# PostgreSQL - Check code_blocks table
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  psql -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM code_blocks WHERE repo_id = 4;"
```

### 2. Environment Variables

**Required for MCP server:**
```bash
# In Claude Code MCP settings (claude_desktop_config.json)
NEO4J_URI=bolt://localhost:7688
NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable
# GEMINI_API_KEY is NO LONGER REQUIRED for diff-based analysis
# (Now uses deterministic block detection instead of LLM)
```

**For standalone testing:**
```bash
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
export GEMINI_API_KEY="<your-gemini-api-key>"
```

---

## Development Workflow

### Step 1: Make Code Changes

Edit files in `/Users/rohankatakam/Documents/brain/coderisk/`:
```
cmd/crisk-check-server/main.go          # Entry point, tool registration
internal/mcp/tools/get_risk_summary.go  # Main risk analysis logic
internal/mcp/local_graph_client.go      # Neo4j/PostgreSQL queries
internal/mcp/identity_resolver.go       # File rename tracking
internal/mcp/diff_atomizer.go           # LLM-based diff extraction
```

### Step 2: Build the Binary

```bash
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/crisk-check-server ./cmd/crisk-check-server
```

**Verify build succeeded:**
```bash
ls -lh bin/crisk-check-server
# Should show binary with recent timestamp
```

**Common build errors:**
- **Missing dependencies:** Run `go mod tidy`
- **Import errors:** Check package paths match directory structure
- **Type errors:** Check that Neo4j/PostgreSQL query result types match Go structs

### Step 3: Kill Old MCP Server Process

Claude Code keeps the MCP server running between requests. After rebuilding, you must kill the old process:

```bash
# Find and kill the old process
pkill -f crisk-check-server

# Verify it's killed
ps aux | grep crisk-check-server | grep -v grep
# Should return nothing
```

**Important:** Claude Code will automatically restart the MCP server on next tool call.

### Step 4: Test in Claude Code

#### A. Open Claude Code in Test Repository

```bash
cd /Users/rohankatakam/Documents/brain/mcp-use
code .  # Or open in your IDE
```

Launch Claude Code from this directory.

#### B. Verify MCP Connection

In Claude Code chat:
```
/mcp
```

Expected output:
```
Connected MCP Servers:
- crisk
  Tools: crisk.get_risk_summary
  Status: ✅ Connected
```

**If not connected:**
1. Check `~/Library/Application Support/Claude/claude_desktop_config.json`
2. Verify MCP server configuration:
```json
{
  "mcpServers": {
    "crisk": {
      "command": "/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server",
      "args": [],
      "env": {
        "NEO4J_URI": "bolt://localhost:7688",
        "NEO4J_PASSWORD": "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
        "POSTGRES_DSN": "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable",
        "GEMINI_API_KEY": "<your-gemini-api-key>"
      }
    }
  }
}
```
3. Restart Claude Code entirely (Cmd+Q, then reopen)

#### C. Run Test Queries

**Test 1: Single file analysis (file-based)**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

Expected: Returns ownership, coupling, and incident data for code blocks in that file.

**Test 2: Uncommitted changes analysis (diff-based)**
```
What is the risk of all my uncommitted changes?
```

Expected: Analyzes all uncommitted changes via `git diff HEAD`, extracts modified blocks, returns risk evidence.

**Test 3: Specific file with uncommitted changes**
```
What are the risk factors for libraries/python/mcp_use/agents/managers/tools/search_tools.py?
```

Expected: Returns risk data for all blocks in that file (whether or not they have uncommitted changes).

### Step 5: Check Logs

**MCP Server Logs:**
```bash
tail -100 /tmp/crisk-mcp-server.log
```

**What to look for:**
- ✅ `Connected to Neo4j`
- ✅ `Connected to PostgreSQL`
- ✅ `Diff atomizer created (supports diff-based analysis)`
- ✅ `Tool called: file_path=..., analyze_all_changes=...`
- ✅ `Found X code blocks in graph`
- ❌ `Failed to connect to...` - Database connection issue
- ❌ `LLM extraction failed` - GEMINI_API_KEY missing or invalid
- ❌ `Found 0 code blocks in graph` - Block names don't exist in Neo4j

**Enable verbose logging (optional):**

Edit `cmd/crisk-check-server/main.go`:
```go
log.SetFlags(log.LstdFlags | log.Lshortfile)  // Add line numbers to logs
```

Rebuild and test.

---

## Testing Scenarios

### Scenario 1: Test File-Based Analysis (No Diff)

**Setup:**
```bash
cd /Users/rohankatakam/Documents/brain/mcp-use
# Ensure working directory is clean
git status
```

**Query:**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

**Expected behavior:**
1. MCP server receives `file_path` parameter
2. Resolves historical paths via `git log --follow`
3. Queries Neo4j for ALL blocks in that file
4. Returns coupling/ownership/temporal data

**Success criteria:**
- Returns `total_blocks` > 0
- Returns `risk_evidence` array with block data
- Each block has `coupled_blocks`, `incident_count`, `staleness_days`

### Scenario 2: Test Uncommitted Changes Analysis (Diff-Based)

**Setup:**
```bash
cd /Users/rohankatakam/Documents/brain/mcp-use
# Make a small change
echo "# test comment" >> libraries/python/mcp_use/client/session.py
git diff HEAD  # Verify uncommitted changes exist
```

**Query:**
```
What is the risk of all my uncommitted changes?
```

**Expected behavior:**
1. MCP server receives `analyze_all_changes=true`
2. Runs `git diff HEAD` to get full repo diff
3. Calls LLM to extract modified block names
4. Resolves file identities for each modified file
5. Queries Neo4j for extracted blocks
6. Returns risk evidence

**Success criteria:**
- Returns `analysis_type: "diff-based"`
- Returns `scope: "all uncommitted changes"`
- Returns blocks that were actually modified

**Known issue:** LLM may extract incorrect block names if method signatures are outside diff context.

### Scenario 3: Test File Rename Handling

**Setup:**
```bash
# Find a file that was renamed
cd /Users/rohankatakam/Documents/brain/mcp-use
git log --follow --name-only --pretty=format:"%H" -- libraries/python/mcp_use/client.py | head -20
```

**Query:**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

**Expected behavior:**
1. Identity resolver finds historical paths
2. Neo4j query searches across ALL historical paths
3. Returns blocks even if they were created under old file name

**Success criteria:**
- Identity resolver log shows multiple paths found
- Blocks from old file path are returned

### Scenario 4: Test Coupling Data (Neo4j CO_CHANGES_WITH Edges)

**Setup:**
```bash
# Verify coupling edges exist in Neo4j
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH ()-[r:CO_CHANGES_WITH]->() RETURN count(r) as count"}]}' | python3 -m json.tool
# Should return count > 0
```

**Query:**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

**Expected behavior:**
1. MCP server queries Neo4j CodeBlock nodes
2. For each block, queries Neo4j CO_CHANGES_WITH edges
3. Returns `coupled_blocks` array with co-change rates

**Success criteria:**
- At least some blocks have `coupled_blocks` length > 0
- Each coupled block has `rate`, `co_change_count`, `path`

**Known issue (FIXED 2025-11-17):** Type mismatch where `blockID` was passed as string instead of int64 to Neo4j query.

### Scenario 5: Test Temporal Data (PostgreSQL Incidents)

**Setup:**
```bash
# Verify incident data exists
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  psql -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM code_block_incidents WHERE repo_id = 4;"
```

**Query:**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

**Expected behavior:**
1. MCP server gets CodeBlock nodes from Neo4j (with `db_id`)
2. For each block, queries PostgreSQL `code_block_incidents` table
3. Returns `incidents` array with issue titles

**Success criteria:**
- Blocks with historical incidents show `incident_count` > 0
- `incidents` array contains `issue_id`, `issue_title`, `confidence`

---

## Edge Cases and Troubleshooting

### Edge Case 1: File Doesn't Exist in Graph

**Scenario:** Query a file that hasn't been analyzed by `crisk init`.

**Expected behavior:**
```json
{
  "file_path": "new_file.py",
  "total_blocks": 0,
  "risk_evidence": [],
  "warning": "No code blocks found for this file"
}
```

**Solution:** Run `crisk init` to analyze the repository.

### Edge Case 2: GEMINI_API_KEY Not Set

**Scenario:** `analyze_all_changes=true` without GEMINI_API_KEY.

**Expected error:**
```
⚠️  GEMINI_API_KEY not set - diff-based analysis disabled
```

**Behavior:** Tool falls back to file-based analysis (if `file_path` provided).

**Solution:** Add GEMINI_API_KEY to `claude_desktop_config.json`.

### Edge Case 3: Git Not Available

**Scenario:** MCP server runs in directory without `.git`.

**Expected error:**
```
failed to get uncommitted changes: git diff failed
```

**Solution:** Ensure Claude Code is opened in the repository root (`mcp-use`).

### Edge Case 4: Neo4j Connection Failed

**Logs:**
```
Failed to connect to Neo4j at bolt://localhost:7688
```

**Checks:**
```bash
# 1. Verify Neo4j is running
docker ps | grep neo4j

# 2. Verify port is correct (7688, not 7687)
docker port <neo4j-container-id>

# 3. Test connection manually
curl -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 http://localhost:7475/
```

**Solution:** Restart Neo4j or fix port mapping.

### Edge Case 5: PostgreSQL Connection Failed

**Logs:**
```
Failed to connect to PostgreSQL
```

**Checks:**
```bash
# 1. Verify PostgreSQL is running
docker ps | grep postgres

# 2. Verify port is correct (5433, not 5432)
docker port <postgres-container-id>

# 3. Test connection manually
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT 1;"
```

**Solution:** Restart PostgreSQL or fix DSN in config.

### Edge Case 6: LLM Extracts Wrong Block Names

**Scenario:** Diff-based analysis returns 0 blocks even though changes exist.

**Logs:**
```
✅ LLM extracted 3 block references
✅ Found 2 historical paths for file.py
✅ Found 0 code blocks in graph  ← PROBLEM
```

**Root cause:** LLM extracted block names that don't exist in Neo4j.

**Debug:**
```bash
# 1. Check what blocks actually exist
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  psql -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT block_name FROM code_blocks WHERE file_path = 'path/to/file.py' LIMIT 10;"

# 2. Check what LLM extracted (in logs)
tail -50 /tmp/crisk-mcp-server.log | grep "LLM extracted"
```

**Current status:** Known issue. LLM sometimes extracts incorrect method names from limited diff context.

**Workaround:** Use file-based analysis instead:
```
What are the risk factors for libraries/python/mcp_use/agents/managers/tools/search_tools.py?
```

### Edge Case 7: Absolute vs Relative Paths

**Scenario:** Claude Code passes absolute path, but Neo4j stores relative paths.

**Solution:** MCP server normalizes absolute → relative using `repo_root` parameter.

**Expected behavior:**
```
Input:  /Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/client.py
Normalized: libraries/python/mcp_use/client.py
```

**Verify in logs:**
```bash
tail -50 /tmp/crisk-mcp-server.log | grep "repo_root"
```

---

## Standalone Testing (Without Claude Code)

For faster iteration, test the MCP server directly via stdio:

### Test 1: Initialize

```bash
cd /Users/rohankatakam/Documents/brain/coderisk
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | \
  NEO4J_URI="bolt://localhost:7688" \
  NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
  ./bin/crisk-check-server
```

Expected: JSON response with server capabilities.

### Test 2: List Tools

```bash
echo '{"jsonrpc":"2.0","method":"tools/list","id":2}' | \
  NEO4J_URI="bolt://localhost:7688" \
  NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
  ./bin/crisk-check-server
```

Expected: JSON array with `crisk.get_risk_summary` tool.

### Test 3: Call Tool

```bash
echo '{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"libraries/python/mcp_use/client.py"}}}' | \
  NEO4J_URI="bolt://localhost:7688" \
  NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
  ./bin/crisk-check-server
```

Expected: JSON response with risk evidence.

---

## Performance Benchmarks

**Cold cache (first call):** ~2-3 seconds
- Neo4j query: ~500ms
- PostgreSQL query: ~200ms
- File identity resolution (git): ~1-2s

**Warm cache (subsequent calls):** ~500ms
- Cached file paths: <10ms
- Neo4j query: ~300ms
- PostgreSQL query: ~200ms

**Diff-based analysis:** ~5-7 seconds
- Git diff: ~100ms
- LLM extraction: ~3-5s (depends on Gemini API latency)
- Graph queries: ~1-2s

---

## Quick Reference: Common Commands

```bash
# Rebuild MCP server
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/crisk-check-server ./cmd/crisk-check-server

# Kill old process
pkill -f crisk-check-server

# Check logs
tail -100 /tmp/crisk-mcp-server.log

# Clear cache
rm /tmp/crisk-mcp-cache.db

# Verify databases
docker ps | grep -E "neo4j|postgres"

# Test Neo4j connection
curl -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 http://localhost:7475/

# Test PostgreSQL connection
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT 1;"

# Restart Claude Code
# Cmd+Q, then reopen
```

---

## Best Practices

1. **Always rebuild before testing** - Go doesn't hot-reload
2. **Always kill old process** - Claude Code caches the binary
3. **Check logs after every test** - Logs reveal actual behavior
4. **Test incrementally** - Make small changes, test immediately
5. **Use file-based queries first** - More reliable than diff-based
6. **Verify data exists** - Check Neo4j/PostgreSQL before debugging MCP server
7. **Clear cache when debugging renames** - `rm /tmp/crisk-mcp-cache.db`

---

## Checklist: Fresh Start

Use this when nothing works:

```bash
# 1. Verify databases
docker ps | grep neo4j
docker ps | grep postgres

# 2. Kill all MCP processes
pkill -f crisk-check-server

# 3. Clear cache
rm /tmp/crisk-mcp-cache.db

# 4. Rebuild
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/crisk-check-server ./cmd/crisk-check-server

# 5. Verify config
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

# 6. Restart Claude Code
# Cmd+Q, then reopen

# 7. Verify connection
# In Claude Code: /mcp

# 8. Test simple query
# "What are the risk factors for libraries/python/mcp_use/client.py?"

# 9. Check logs
tail -50 /tmp/crisk-mcp-server.log
```

---

**Last Updated:** 2025-11-17

**Status:** ✅ Iterative testing workflow validated and documented
