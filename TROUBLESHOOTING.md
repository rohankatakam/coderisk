# CodeRisk MCP Server - Troubleshooting Guide

## Issue: Server Hangs at "Restarting MCP server process"

### Symptoms
- Claude Code `/mcp` shows status: "◯ connecting..."
- Clicking "Reconnect" shows "✢ Restarting MCP server process"
- Process hangs indefinitely at this message

### Root Cause
The MCP server logs were going to stderr, which can interfere with the stdio-based MCP protocol communication in some environments.

### Fix Applied (Nov 16, 2025)
**Version**: Latest build redirects all logs to `/tmp/crisk-mcp-server.log`

**Changes Made**:
- Added log file redirect in [cmd/crisk-check-server/main.go](cmd/crisk-check-server/main.go#L17-L22)
- All `log.Println()` calls now go to file instead of stderr
- stdout/stdin remain clean for MCP JSON-RPC protocol

### Solution Steps

#### 1. Ensure You Have Latest Binary
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/crisk-check-server ./cmd/crisk-check-server
ls -lh bin/crisk-check-server  # Should show Nov 16 21:19 or later
```

#### 2. Check Log File Location
The server now logs to: `/tmp/crisk-mcp-server.log`

```bash
# Monitor logs in real-time
tail -f /tmp/crisk-mcp-server.log
```

#### 3. Restart Claude Code
After rebuilding the binary:
1. **Quit Claude Code completely** (Cmd+Q)
2. Wait 5 seconds
3. Reopen Claude Code
4. Navigate to `/Users/rohankatakam/Documents/brain/mcp-use`
5. Type `/mcp` to check status

#### 4. If Still Hanging
If reconnection still hangs, try manual connection test:

```bash
# Kill any hung processes
pkill -f crisk-check-server

# Clear log file
rm -f /tmp/crisk-mcp-server.log

# Test server startup manually
cd /Users/rohankatakam/Documents/brain/coderisk
NEO4J_URI="bolt://localhost:7688" \
NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
./bin/crisk-check-server < /dev/null &

# Wait 3 seconds
sleep 3

# Check log file (should show connections)
cat /tmp/crisk-mcp-server.log

# Kill test process
pkill -f crisk-check-server
```

**Expected log output**:
```
2025/11/16 21:19:25 ✅ Connected to Neo4j
2025/11/16 21:19:25 ✅ Connected to PostgreSQL
2025/11/16 21:19:25 ✅ Registered tool: crisk.get_risk_summary
2025/11/16 21:19:25 🚀 MCP server started on stdio
```

---

## Issue: "Failed to reconnect to coderisk"

### Possible Causes

#### 1. Database Not Running
```bash
# Check database status
docker ps | grep coderisk

# Should show:
# coderisk-neo4j      Up X hours (healthy)
# coderisk-postgres   Up X hours (healthy)
```

**Fix**:
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
docker compose up -d
```

#### 2. Binary Path Incorrect
Check Claude Code configuration:
```bash
cat ~/.claude.json | grep -A 10 '"coderisk"'
```

**Should show**:
```json
"coderisk": {
  "type": "stdio",
  "command": "/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server",
  "args": [],
  "env": {
    "NEO4J_URI": "bolt://localhost:7688",
    "NEO4J_PASSWORD": "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
    "POSTGRES_DSN": "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
  }
}
```

#### 3. Binary Not Executable
```bash
ls -l /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server

# Should show: -rwxr-xr-x (executable bit set)
```

**Fix**:
```bash
chmod +x /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server
```

#### 4. Environment Variables Missing
The server requires three environment variables. Check they're in config:
- `NEO4J_URI`
- `NEO4J_PASSWORD`
- `POSTGRES_DSN`

---

## Issue: Server Connects But Tool Not Called

### Symptoms
- `/mcp` shows "✔ connected"
- Claude reads files directly instead of calling `crisk.get_risk_summary`

### Possible Causes

#### 1. Tool Not Registered
Check log file for registration message:
```bash
grep "Registered tool" /tmp/crisk-mcp-server.log
```

**Expected**:
```
2025/11/16 21:19:25 ✅ Registered tool: crisk.get_risk_summary
```

#### 2. Ambiguous Prompt
Claude might not recognize when to use the tool.

**Try explicit phrasing**:
```
Use the crisk.get_risk_summary tool to analyze libraries/python/mcp_use/client.py
```

#### 3. File Path Issues
Ensure file paths are relative to repository root:
- ✅ `libraries/python/mcp_use/client.py`
- ❌ `/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/client.py`

---

## Issue: "MCP tool response exceeds maximum allowed tokens"

### Symptoms
Error message: "MCP tool response (X tokens) exceeds maximum allowed tokens (25000)"

### Solution
Use stricter filtering parameters:

```
Show me only high-risk code (score > 15) in client.py, limit to 5 blocks
```

**Parameters Claude should use**:
```json
{
  "file_path": "libraries/python/mcp_use/client.py",
  "min_risk_score": 15.0,
  "max_blocks": 5,
  "max_coupled_blocks": 1,
  "max_incidents": 1
}
```

---

## Diagnostic Commands

### Check Server Binary
```bash
# Version check
ls -lh /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server

# Test execution
cd /Users/rohankatakam/Documents/brain/coderisk
./bin/crisk-check-server --help 2>&1 | head -5
```

### Check Databases
```bash
# Neo4j connectivity
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 "MATCH (n) RETURN count(n) LIMIT 1;"

# PostgreSQL connectivity
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT COUNT(*) FROM github_repositories;"
```

### Check Claude Code Configuration
```bash
# View full MCP config
cat ~/.claude.json | python3 -m json.tool | grep -A 15 '"mcpServers"'

# Check if coderisk is registered
cat ~/.claude.json | grep -c '"coderisk"'  # Should be > 0
```

### Monitor Server Logs
```bash
# Clear old logs
rm -f /tmp/crisk-mcp-server.log

# Start monitoring (in separate terminal)
tail -f /tmp/crisk-mcp-server.log

# Then restart Claude Code and check for connection messages
```

---

## Common Error Messages

### "Failed to connect to Neo4j"
**Cause**: Neo4j container not running or wrong credentials
**Fix**:
```bash
docker start coderisk-neo4j
# Wait 10 seconds for startup
docker ps | grep neo4j  # Should show "healthy"
```

### "PostgreSQL ping failed"
**Cause**: PostgreSQL container not running
**Fix**:
```bash
docker start coderisk-postgres
```

### "Failed to open cache"
**Cause**: Cannot write to `/tmp/crisk-mcp-cache.db`
**Fix**:
```bash
rm -f /tmp/crisk-mcp-cache.db
chmod 777 /tmp  # Ensure /tmp is writable
```

### "Protocol version not supported"
**Cause**: Outdated MCP SDK (should not occur after latest build)
**Fix**:
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
go get github.com/modelcontextprotocol/go-sdk/mcp@latest
go build -o bin/crisk-check-server ./cmd/crisk-check-server
```

---

## Reset Everything

If all else fails, complete reset:

```bash
# 1. Kill all processes
pkill -f crisk-check-server

# 2. Clear cache and logs
rm -f /tmp/crisk-mcp-cache.db
rm -f /tmp/crisk-mcp-server.log

# 3. Restart databases
cd /Users/rohankatakam/Documents/brain/coderisk
docker compose restart

# 4. Rebuild binary
go build -o bin/crisk-check-server ./cmd/crisk-check-server

# 5. Quit Claude Code completely (Cmd+Q)

# 6. Reopen Claude Code

# 7. Navigate to mcp-use directory
cd /Users/rohankatakam/Documents/brain/mcp-use

# 8. Check status
# Type /mcp in Claude Code
```

---

## Getting Help

If issues persist:

1. **Collect diagnostic info**:
```bash
{
  echo "=== Binary Info ==="
  ls -lh /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server
  echo ""

  echo "=== Database Status ==="
  docker ps | grep coderisk
  echo ""

  echo "=== Server Logs ==="
  cat /tmp/crisk-mcp-server.log
  echo ""

  echo "=== Claude Config ==="
  cat ~/.claude.json | grep -A 10 '"coderisk"'
} > /tmp/coderisk-diagnostic.txt

cat /tmp/coderisk-diagnostic.txt
```

2. **Share diagnostic output** when reporting issues

3. **Check documentation**:
- [READY_TO_TEST.md](READY_TO_TEST.md) - Testing instructions
- [CONTEXT_AWARE_OPTIMIZATIONS.md](CONTEXT_AWARE_OPTIMIZATIONS.md) - Parameter documentation
- [MCP_TEST_QUESTIONS.md](MCP_TEST_QUESTIONS.md) - Test questions

---

## Success Indicators

When everything is working:

1. ✅ `/mcp` shows "✔ connected" next to "coderisk"
2. ✅ `/tmp/crisk-mcp-server.log` shows database connections
3. ✅ Claude calls `crisk.get_risk_summary` when asked about file risks
4. ✅ Responses include risk data (blocks, coupling, incidents)
5. ✅ No token overflow errors
6. ✅ Risk scores visible when requested

**Test question that should work**:
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

**Expected behavior**:
- Claude calls MCP tool
- Returns risk analysis with code blocks
- Shows ownership, coupling, and incident data
- Response is < 25,000 tokens
