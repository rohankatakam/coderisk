# Testing Incident Linking Fix

## What Was Fixed

**Root Cause:** ID format mismatch between Neo4j and PostgreSQL queries
- Neo4j CodeBlock nodes have TWO ID properties:
  - `b.id`: Composite string (e.g., `"4:codeblock:path:name"`) - for graph relationships
  - `b.db_id`: PostgreSQL integer (e.g., `584`) - foreign key to PostgreSQL
- **Bug:** MCP queries returned `b.id` (string) but PostgreSQL expected integer
- **Fix:** Changed queries to return `b.db_id` and convert to string format

**Files Changed:**
- `/Users/rohankatakam/Documents/brain/coderisk/internal/mcp/local_graph_client.go`
  - Line 41: Changed `RETURN b.id` → `RETURN b.db_id`
  - Line 255: Changed `RETURN b.id` → `RETURN b.db_id`
  - Lines 70-72 & 284-286: Added int64 → string conversion

**Documentation:**
- `/Users/rohankatakam/Documents/brain/docs/ingestion_aws.md`
  - Added architectural note about future Neo4j-only incident storage optimization

## Test File Details

**File to Test:** `libraries/python/mcp_use/agents/mcpagent.py`

**Expected Incidents:**
1. **MCPAgent** (class) → "Disallowed tools not working" (confidence: 0.80)
2. **MCPAgent.connect** (method) → "Connection State Not Properly Tracked After SSE Disconnection" (confidence: 0.80)
3. **__init__** (method) → "Disallowed tools not working" (confidence: 0.80)
4. **_consume_and_return** (method) → "Disallowed tools not working" (confidence: 0.80)
5. **stream** (method) → "Disallowed tools not working" (confidence: 0.80)

**Summary:** 5 blocks linked to 2 unique incidents

## How to Test

### Prerequisites
✅ MCP server rebuilt: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server`
✅ Databases running: Neo4j (7688), PostgreSQL (5433)
✅ Repository: mcp-use (repo_id: 4)

### Test Query for Claude Code

**Navigate to:** `/Users/rohankatakam/Documents/brain/mcp-use`

**Ask Claude Code:**
```
What are the risk factors for libraries/python/mcp_use/agents/mcpagent.py?
```

### Expected Behavior

**Before Fix:**
- All blocks showed `"incident_count": 0` and `"incidents": []`

**After Fix:**
- `MCPAgent` class should show `"incident_count": 1` with incident titled "Disallowed tools not working"
- `MCPAgent.connect` should show `"incident_count": 1` with incident titled "Connection State Not Properly Tracked After SSE Disconnection"
- `__init__`, `_consume_and_return`, `stream` should each show `"incident_count": 1` with incident "Disallowed tools not working"

### What to Look For

1. **In Claude's Response:**
   - Should mention historical incidents/bugs
   - Should reference specific issue titles like "Disallowed tools not working"
   - Should note confidence scores (0.80)

2. **In Tool Output (if visible):**
   ```json
   {
     "risk_evidence": [
       {
         "block_name": "MCPAgent",
         "temporal_data": {
           "incident_count": 1,
           "incidents": [
             {
               "issue_title": "Disallowed tools not working",
               "confidence_score": 0.80
             }
           ]
         }
       }
     ]
   }
   ```

3. **In Risk Score Calculation:**
   - Blocks with incidents should have HIGHER risk scores
   - Incidents contribute 10x weight to overall risk score

## Verification Commands

If you want to verify the fix at database level:

```bash
# 1. Check PostgreSQL has the incidents
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT cb.id, cb.block_name, COUNT(cbi.issue_id) as incidents
FROM code_blocks cb
LEFT JOIN code_block_incidents cbi ON cb.id = cbi.code_block_id
WHERE cb.file_path = 'libraries/python/mcp_use/agents/mcpagent.py'
GROUP BY cb.id, cb.block_name
ORDER BY cb.block_name;
"

# 2. Check Neo4j has db_id property
# (Would need cypher-shell to verify, but ingestion code confirms this exists)
```

## Success Criteria

✅ Claude mentions specific historical incidents in its risk assessment
✅ Incident titles appear in the analysis (not just counts)
✅ Blocks with incidents show higher risk scores than blocks without
✅ Confidence scores are included (0.80)

## If It Still Doesn't Work

Check these debugging steps:

1. **Verify MCP server is using new binary:**
   ```bash
   ls -lh /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server
   # Should show recent modification time
   ```

2. **Check logs:**
   ```bash
   tail -100 /tmp/crisk-mcp-server.log
   # Look for "GetTemporalData" calls and SQL queries
   ```

3. **Verify Neo4j has db_id:**
   - The ingestion code sets `b.db_id = $db_id` (line 38 in graph_writer.go)
   - If missing, re-run `crisk init` to rebuild graph
