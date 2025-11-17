# Testing MCP Server with Your Existing Data

Since you already have mcp-use ingested (repo_id: 4), here's how to test immediately:

## Quick Test (2 minutes)

### Option 1: Clone mcp-use fresh (Recommended)

```bash
# 1. Clone mcp-use to a test location
cd /tmp
git clone https://github.com/mcp-use/mcp-use.git
cd mcp-use

# 2. Open in VS Code/Claude Code
code .
```

### Option 2: Use existing omnara repo

Since you also have omnara-ai/omnara ingested (repo_id: 3), you need to:

**Either:**
1. Update the MCP server to use repo_id: 3 instead of 4
2. Or just use omnara directly

**To use omnara:**

```bash
cd ~/.coderisk/repos/a1ee33a52509d445-full
code .
```

Then in Claude Code, ask:
```
What are the risk factors for apps/web/src/utils/statusUtils.ts?
```

## Fix for Multi-Repository Support

The MCP server currently hardcodes `repo_id=4`. Here's the quick fix:

### Edit the MCP server code:

**File**: `internal/mcp/tools/get_risk_summary.go` (line 142)

**Change from:**
```go
repoID := 4 // mcp-use repo_id from actual implementation
```

**Change to** (quick fix):
```go
// Use omnara repo
repoID := 3
```

**Or better** (auto-detect):
```go
// Auto-detect from database based on file path
repoID, err := t.identityResolver.GetRepoIDFromPath(ctx, filePath)
if err != nil {
    return nil, fmt.Errorf("failed to determine repo: %w", err)
}
```

### Rebuild and test:

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Quick edit to use repo_id 3
sed -i.bak 's/repoID := 4/repoID := 3/' internal/mcp/tools/get_risk_summary.go

# Rebuild
go build -o bin/crisk-check-server cmd/crisk-check-server/main.go

# Restart Claude Code to pick up new binary
```

## Verify Your Data

Check what data you have:

```bash
# Check repositories
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT id, full_name FROM github_repositories;"

# Check code blocks for each repo
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "
SELECT
  r.full_name,
  COUNT(DISTINCT cb.id) as blocks,
  COUNT(DISTINCT cb.file_path) as files
FROM github_repositories r
LEFT JOIN code_blocks cb ON cb.repo_id = r.id
GROUP BY r.id, r.full_name
ORDER BY r.id;
"
```

**Expected output:**
```
     full_name     | blocks | files
-------------------+--------+-------
 omnara-ai/omnara  |    XXX |   XXX
 mcp-use/mcp-use   |    994 |   ~200
```

## Test the MCP Server Directly

Without Claude Code, you can test the server:

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Start server in one terminal
NEO4J_URI=bolt://localhost:7688 \
NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \
./bin/crisk-check-server
```

Then in another terminal, send a test request:

```bash
# For mcp-use (repo_id 4)
echo '{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"libraries/python/mcp_use/client.py"}}}' | nc localhost 12345

# For omnara (repo_id 3) - need to fix repo_id first
echo '{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"apps/web/src/utils/statusUtils.ts"}}}' | nc localhost 12345
```

## Immediate Action Plan

**To test RIGHT NOW with existing data:**

1. **Pick a repository**:
   - mcp-use: Clone fresh to `/tmp/mcp-use` (server already configured for this)
   - omnara: Use existing at `~/.coderisk/repos/a1ee33a52509d445-full` (need to change repo_id)

2. **For mcp-use** (easiest):
   ```bash
   cd /tmp
   git clone https://github.com/mcp-use/mcp-use.git
   cd mcp-use
   code .
   ```

   Then ask Claude:
   ```
   What are the risk factors for libraries/python/mcp_use/client.py?
   ```

3. **For omnara** (need quick edit):
   ```bash
   # Change repo_id to 3
   cd /Users/rohankatakam/Documents/brain/coderisk
   sed -i.bak 's/repoID := 4/repoID := 3/' internal/mcp/tools/get_risk_summary.go

   # Rebuild
   go build -o bin/crisk-check-server cmd/crisk-check-server/main.go

   # Test with omnara
   cd ~/.coderisk/repos/a1ee33a52509d445-full
   code .
   ```

   Then ask Claude:
   ```
   What are the risk factors for apps/web/src/components/dashboard/SidebarDashboardLayout.tsx?
   ```

## Why the MCP Server Needs the Right Repository

The MCP server:
1. Receives a file path from Claude Code (e.g., `src/main.py`)
2. Queries the database with that file path + repo_id
3. Returns risk data for code blocks in that file

**If repo_id is wrong**, it won't find any blocks because the file paths are scoped to repositories.

## Next Steps

1. **Immediate**: Clone mcp-use fresh or edit code for omnara
2. **Short-term**: Implement auto-detection of repo_id from git remote
3. **Long-term**: Support multiple repositories simultaneously

---

**Bottom line**: The MCP server is working, you just need to either:
- Use it with mcp-use (clone fresh)
- OR edit the repo_id to 3 and rebuild for omnara
