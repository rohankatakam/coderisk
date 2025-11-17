# CodeRisk MCP Server - Ready to Test

## Build Status: ✅ Complete

The MCP server has been successfully rebuilt with all context-aware optimizations.

### Build Information
- **Binary Location**: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server`
- **Binary Type**: Mach-O 64-bit executable arm64
- **Build Date**: November 16, 2025
- **Go SDK Version**: github.com/modelcontextprotocol/go-sdk/mcp v1.1.0

## System Status

### ✅ Databases Connected
- **Neo4j**: Running at `bolt://localhost:7688` (healthy)
- **PostgreSQL**: Running at `localhost:5433` (healthy)

### ✅ Repository Data Available
Two repositories are ingested and ready:
1. **omnara-ai/omnara** (repo_id: 3)
2. **mcp-use/mcp-use** (repo_id: 4) ← Currently hardcoded in MCP server

### ✅ Claude Code Configuration
MCP server registered at **user scope** with:
```json
{
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
}
```

## New Features Implemented

### 1. Context-Aware Optimizations
Three new parameters to maximize signal density:

#### `min_risk_score` (float, default: 0.0)
Filter blocks by minimum risk threshold
```json
{"min_risk_score": 15.0}  // Only blocks with score >= 15.0
```

#### `include_risk_score` (bool, default: false)
Show calculated risk scores in output for verification
```json
{"include_risk_score": true}  // Shows "risk_score": 45.2 in output
```

#### `prioritize_recent` (bool, default: false)
Boost scores for recently changed code (< 30 days)
```json
{"prioritize_recent": true}  // Fresh code gets +0 to +5 points boost
```

### 2. Complete Parameter List

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `file_path` | string | required | Path to file to analyze |
| `diff_content` | string | optional | Uncommitted changes |
| `max_blocks` | int | 10 | Max code blocks to return |
| `max_coupled_blocks` | int | 1 | Max coupled blocks per block |
| `max_incidents` | int | 1 | Max incidents per block |
| `block_types` | []string | all | Filter by type (class, method, function) |
| `summary_only` | bool | false | Return stats only, no details |
| `min_staleness` | int | 0 | Min days since last change |
| `min_incidents` | int | 0 | Min historical incidents |
| **`min_risk_score`** | **float** | **0.0** | **Min calculated risk score** |
| **`include_risk_score`** | **bool** | **false** | **Show risk scores** |
| **`prioritize_recent`** | **bool** | **false** | **Boost fresh code** |

## Testing Instructions

### Step 1: Restart Claude Code
Since we rebuilt the binary, restart Claude Code to ensure it loads the new version.

### Step 2: Navigate to Test Repository
```bash
cd /Users/rohankatakam/Documents/brain/mcp-use
```

### Step 3: Check MCP Server Status
In Claude Code, type:
```
/mcp
```

**Expected Status**:
- Server name: `coderisk` or `Coderisk MCP Server`
- Status: `✔ connected` (green checkmark)

**If showing "◯ connecting..."**:
1. Click "Reconnect"
2. Wait 5-10 seconds
3. Check status again

**If showing "✘ failed"**:
1. Check logs (see below)
2. Verify databases are running: `docker ps | grep coderisk`
3. Try clicking "Reconnect"

### Step 4: Test Basic Functionality

#### Test 1: Get Risk Summary
Ask Claude:
```
"What are the risk factors for libraries/python/mcp_use/client.py?"
```

**Expected Behavior**:
- Claude calls `crisk.get_risk_summary` tool
- Returns risk analysis with blocks, coupling, incidents
- Response should be < 25,000 tokens

#### Test 2: Test with Risk Scores Visible
Ask Claude:
```
"Show me the risk factors for client.py with the calculated risk scores visible"
```

**Expected Behavior**:
- Claude calls tool with `include_risk_score: true`
- Output shows `"risk_score": <number>` for each block
- Claude can explain scores (e.g., "authenticate has score 45.2 due to 3 incidents...")

#### Test 3: Filter by Risk Threshold
Ask Claude:
```
"Show me only high-risk code (score > 15) in client.py"
```

**Expected Behavior**:
- Claude calls tool with `min_risk_score: 15.0`
- Only blocks with score >= 15.0 returned
- Fewer blocks than Test 1

#### Test 4: Focus on Recent Changes
Ask Claude:
```
"What's risky in recently changed code in client.py?"
```

**Expected Behavior**:
- Claude calls tool with `prioritize_recent: true`
- Fresh code (< 30 days) gets boosted scores
- Results favor recently modified blocks

### Step 5: Verify Accuracy

When Claude provides analysis, check:
1. **Risk scores match data**: If `include_risk_score: true`, verify numbers make sense
2. **No hallucination**: Analysis should reference actual incidents, coupling, staleness
3. **Ranking is correct**: If multiple blocks, highest risk should be first

## Troubleshooting

### Issue: Server Status Shows "◯ connecting..."

**Possible Causes**:
1. Binary is starting but hanging on database connection
2. Databases not running
3. Wrong environment variables

**Debug Steps**:
```bash
# 1. Check databases
docker ps | grep coderisk

# 2. Test binary directly
cd /Users/rohankatakam/Documents/brain/coderisk
NEO4J_URI="bolt://localhost:7688" \
NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
./bin/crisk-check-server < /dev/null

# Expected: "✅ Connected to Neo4j" and "✅ Connected to PostgreSQL"
```

### Issue: Server Status Shows "✘ failed"

**Check Logs**:
```bash
# Find most recent log
ls -lt ~/Library/Logs/Claude/*.log | head -1

# View log
tail -100 ~/Library/Logs/Claude/mcp.log
```

**Common Errors**:
- `Connection refused` → Databases not running
- `Protocol version not supported` → Outdated SDK (should not happen after rebuild)
- `Tool schema validation failed` → Schema issue (should not happen after rebuild)

### Issue: Claude Doesn't Call the Tool

**Possible Causes**:
1. Server not connected
2. Tool not registered
3. Prompt doesn't trigger tool use

**Debug Steps**:
1. Verify `/mcp` shows `✔ connected`
2. Try explicit phrasing: "Use the crisk.get_risk_summary tool to analyze..."
3. Check Claude Code logs for errors

### Issue: Response Too Large

**Symptoms**: Error like "MCP tool response (X tokens) exceeds maximum allowed tokens (25000)"

**Solutions**:
1. Use stricter filters: `min_risk_score: 20.0`
2. Reduce limits: `max_blocks: 5`, `max_coupled_blocks: 1`, `max_incidents: 1`
3. Use `summary_only: true` for initial assessment

## Test Questions to Try

### Basic Risk Analysis
```
"What are the main risk factors for libraries/python/mcp_use/client.py?"
```

### Transparent Scoring
```
"Analyze client.py and show me the risk scores for each code block"
```

### High-Risk Focus
```
"Show me only the truly risky code in client.py (risk score > 20)"
```

### Recent Changes Focus
```
"What's risky in code we've recently changed in client.py?"
```

### Combined Optimization
```
"Show me high-risk, recently changed code with scores visible"
```

### Summary Mode (for large files)
```
"Give me a summary of the risk profile for client.py"
```

## Expected Performance

### Token Usage by Configuration

| Configuration | Est. Blocks | Est. Tokens | Signal Density |
|--------------|-------------|-------------|----------------|
| Default (max_blocks: 10) | 10 | ~15k | Medium |
| High-risk only (min_risk_score: 15.0) | 3-5 | ~6-8k | High |
| Summary mode | 0 | ~1k | N/A (stats only) |
| Full detail (max_blocks: 1, max_coupled: 10, max_incidents: 10) | 1 | ~10-15k | Very High |

## Known Limitations

1. **Hardcoded Repository**: Currently uses repo_id=4 (mcp-use). To test omnara repo, need to manually change code.
2. **No Multi-Repo Support**: Can only query one repository at a time.
3. **Staleness Calculation**: Based on last commit to file, not individual blocks.

## Documentation

Comprehensive guides created:
1. **[CONTEXT_AWARE_OPTIMIZATIONS.md](CONTEXT_AWARE_OPTIMIZATIONS.md)** - Detailed explanation of new parameters
2. **[CONTEXT_OPTIMIZATIONS_COMPLETE.md](CONTEXT_OPTIMIZATIONS_COMPLETE.md)** - Implementation summary
3. **[AGENTIC_WORKFLOW_GUIDE.md](AGENTIC_WORKFLOW_GUIDE.md)** - How to use in agentic workflows (updated)
4. **[RISK_SCORING_ALGORITHM.md](RISK_SCORING_ALGORITHM.md)** - Risk calculation details (updated)

## Ready to Test! 🚀

Your MCP server is fully operational and ready for testing. The new context-aware optimizations will maximize signal density and ensure you always see the most relevant, highest-risk code first.

**Next Step**: Open Claude Code, navigate to `/Users/rohankatakam/Documents/brain/mcp-use`, and try the test questions above!
