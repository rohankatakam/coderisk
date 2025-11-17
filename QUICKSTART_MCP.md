# MCP Server Quick Start Guide

## 1. Test Locally (2 minutes)

```bash
cd /Users/rohankatakam/Documents/brain/coderisk
./test_mcp_interactive.sh
```

This will automatically:
- ✅ Check database connections
- ✅ Show available data
- ✅ Test the MCP server
- ✅ Display sample risk evidence

## 2. Integrate with Claude Code (5 minutes)

### Step 1: Create MCP settings file

```bash
mkdir -p ~/.config/Claude
cat > ~/.config/Claude/mcp_settings.json << 'EOF'
{
  "mcpServers": {
    "crisk": {
      "command": "/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server",
      "args": [],
      "env": {
        "NEO4J_URI": "bolt://localhost:7688",
        "NEO4J_PASSWORD": "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
        "POSTGRES_DSN": "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
      }
    }
  }
}
EOF
```

### Step 2: Verify Docker containers are running

```bash
docker ps | grep -E "(neo4j|postgres)"
```

Should show both containers in "Up" status.

### Step 3: Open mcp-use repository in Claude Code

```bash
# Clone if you don't have it
cd /tmp
git clone https://github.com/mcp-use/mcp-use.git
cd mcp-use

# Open in Claude Code
code .
```

### Step 4: Test with Claude Code

In Claude Code, ask:
- "What are the risk factors for libraries/python/mcp_use/client.py?"
- "Which code blocks in this file have ownership issues?"
- "Show me coupling relationships in this file"

Claude will automatically call the `crisk.get_risk_summary` tool and provide analysis.

## 3. What You'll Get

For each code block in a file, you'll see:

**Ownership**:
- Original author
- Last modifier
- Days since last modification (staleness)
- Developer familiarity map

**Coupling**:
- Which other blocks change together with this one
- Coupling rates (0.0-1.0)
- Co-change counts

**Temporal**:
- Historical incidents (issues/PRs) linked to this block
- Confidence scores
- Issue titles and states

## 4. Example Output

```
Block: MCPClient.__init__
- Owner: pietro.zullo@gmail.com (17 days stale)
- Coupled with: MCPSession.__init__ (rate: 1.0)
- Incidents: 0

Block: MCPClient.create_session
- Owner: pietro.zullo@gmail.com (17 days stale)
- Coupled with: MCPSession.__init__, create_session_from_config
- Incidents: 0
```

## 5. Current Data

- **Repository**: mcp-use (repo_id: 4)
- **Code Blocks**: 994
- **Temporal Incidents**: 98 (after timeline fix)
- **Files Covered**: ~200 Python/TypeScript files

## 6. Troubleshooting

**No tool response in Claude Code?**
```bash
# Restart Claude Code
# Check logs: ~/.claude/logs/

# Verify server works standalone:
NEO4J_URI=bolt://localhost:7688 \
NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server
```

**Databases not running?**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
docker-compose up -d
```

**Want to test a different file?**
```bash
# Check which files have the most blocks
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT file_path, COUNT(*) FROM code_blocks WHERE repo_id = 4 GROUP BY file_path ORDER BY count DESC LIMIT 10;"
```

## 7. Next Steps

- ✅ **Committed & Pushed**: All code is in git
- ✅ **MCP Server Tested**: Working with JSON-RPC
- ⏭️ **Claude Code Testing**: Test in real Claude Code session
- ⏭️ **Demo**: Record demo showing risk analysis in action
- ⏭️ **More Repos**: Ingest additional repositories

## 8. Files Reference

- **MCP Server**: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server`
- **Test Script**: `/Users/rohankatakam/Documents/brain/coderisk/test_mcp_interactive.sh`
- **Documentation**: `/Users/rohankatakam/Documents/brain/coderisk/MCP_SERVER_README.md`
- **Test Results**: `/Users/rohankatakam/Documents/brain/coderisk/MCP_SERVER_TEST_RESULTS.md`
