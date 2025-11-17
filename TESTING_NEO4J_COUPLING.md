# Testing Neo4j Coupling Migration

## What Was Done

**Migration:** Moved coupling data storage from PostgreSQL to Neo4j

**Date:** 2025-11-17

**Changes:**
1. Created sync script (`cmd/sync-coupling-to-neo4j/main.go`) to migrate existing PostgreSQL coupling data to Neo4j
2. Successfully synced **8,016 coupling relationships** to Neo4j as `CO_CHANGES_WITH` edges
3. Updated `GetCouplingData` in [local_graph_client.go](internal/mcp/local_graph_client.go#L120-L185) to query Neo4j instead of PostgreSQL
4. Rebuilt MCP server with Neo4j coupling backend

**Neo4j Edge Schema:**
```cypher
(b1:CodeBlock)-[:CO_CHANGES_WITH {
  rate: 0.95,
  co_change_count: 12,
  last_co_changed_at: datetime
}]->(b2:CodeBlock)
```

## How to Test

### Prerequisites
✅ Neo4j running on port 7688
✅ MCP server rebuilt: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server`
✅ Coupling data synced to Neo4j (8,016 edges)
✅ Repository: mcp-use (repo_id: 4)

### Test Query for Claude Code

**Navigate to:** `/Users/rohankatakam/Documents/brain/mcp-use`

**Ask Claude Code:**
```
What are the risk factors for libraries/python/mcp_use/client.py?
```

### Expected Behavior

**Before Migration (PostgreSQL):**
- Queried PostgreSQL `code_block_coupling` table
- Required Neo4j → PostgreSQL round trip for each code block

**After Migration (Neo4j):**
- Single Neo4j query fetches coupling data
- Coupling relationships returned in response
- Example from previous test:
  ```json
  {
    "coupled_blocks": [
      {
        "name": "documentation files",
        "path": "README.md",
        "rate": 1.0,
        "co_change_count": 15
      }
    ]
  }
  ```

### What to Look For

1. **In Claude's Response:**
   - Should mention co-change relationships
   - Should reference files that frequently change together
   - Should note co-change rates (e.g., "changes together 100% of the time")

2. **In Tool Output (if visible):**
   ```json
   {
     "risk_evidence": [
       {
         "block_name": "MCPClient",
         "coupling_data": {
           "coupled_blocks": [
             {
               "name": "related_block",
               "path": "path/to/file.py",
               "rate": 0.95,
               "co_change_count": 12
             }
           ]
         }
       }
     ]
   }
   ```

3. **Performance:**
   - Response time should be similar or faster than PostgreSQL implementation
   - No errors related to database queries

## Verification Commands

### 1. Check Neo4j has coupling edges:
```bash
NEO4J_URI="bolt://localhost:7688" NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:CO_CHANGES_WITH]->() RETURN count(r) as edge_count;"
```
**Expected:** `edge_count: 8016`

### 2. Sample coupling edges:
```bash
NEO4J_URI="bolt://localhost:7688" NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (a:CodeBlock)-[r:CO_CHANGES_WITH]->(b:CodeBlock)
   WHERE r.rate >= 0.9
   RETURN a.name, b.name, r.rate, r.co_change_count
   LIMIT 5;"
```

### 3. Find coupling for specific block:
```bash
# Find coupling for MCPClient class (db_id would be from PostgreSQL)
NEO4J_URI="bolt://localhost:7688" NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (b:CodeBlock {name: 'MCPClient'})-[r:CO_CHANGES_WITH]-(coupled:CodeBlock)
   RETURN b.name, coupled.name, r.rate, r.co_change_count
   ORDER BY r.rate DESC
   LIMIT 10;"
```

## Success Criteria

✅ Claude mentions co-change relationships in risk assessment
✅ Coupling data appears with accurate rates and counts
✅ No database errors in MCP server logs
✅ Response time is acceptable (<10 seconds)
✅ Neo4j has 8,016 `CO_CHANGES_WITH` edges

## Comparison: PostgreSQL vs Neo4j

| Aspect | PostgreSQL (Old) | Neo4j (New) |
|--------|------------------|-------------|
| **Query Count** | 2 queries (Neo4j for blocks + PostgreSQL for coupling) | 1 query (Neo4j only) |
| **Data Locality** | Split across databases | All in graph database |
| **Performance** | Extra round trip | Single query |
| **Graph Traversal** | Limited (can't traverse coupling in same query) | Rich (can combine with ownership, incidents) |
| **Maintainability** | Dual-database complexity | Single source of truth |

## Next Steps

### Phase 3: Update Ingestion Pipeline
Currently, the ingestion pipeline (`internal/risk/coupling.go`) still writes coupling data to PostgreSQL during `crisk init`. To complete the migration:

1. Update `CouplingCalculator.CalculateCoChanges` to write to Neo4j:
   ```go
   // Create CO_CHANGES_WITH edge in Neo4j
   query := `
       MATCH (a:CodeBlock {db_id: $blockAID})
       MATCH (b:CodeBlock {db_id: $blockBID})
       MERGE (a)-[r:CO_CHANGES_WITH]->(b)
       SET r.rate = $rate,
           r.co_change_count = $count,
           r.last_co_changed_at = datetime()
   `
   ```

2. Keep PostgreSQL write for backward compatibility (dual-write pattern)

3. After verifying Neo4j queries work in production, remove PostgreSQL coupling table

## Troubleshooting

### If coupling data returns empty:

1. **Check Neo4j edges exist:**
   ```bash
   # Should return 8016
   cypher-shell "MATCH ()-[r:CO_CHANGES_WITH]->() RETURN count(r);"
   ```

2. **Check blockID format:**
   - GetCouplingData expects string version of integer PostgreSQL ID
   - Neo4j `db_id` property should match PostgreSQL `id` column

3. **Check MCP server logs:**
   ```bash
   tail -100 /tmp/crisk-mcp-server.log
   # Look for "GetCouplingData" calls and Neo4j query execution
   ```

4. **Verify db_id property exists:**
   ```bash
   cypher-shell "MATCH (b:CodeBlock) RETURN b.db_id LIMIT 5;"
   # Should return integer IDs
   ```

5. **Re-run sync script if needed:**
   ```bash
   NEO4J_URI="bolt://localhost:7688" \
   NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
   POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" \
   ./bin/sync-coupling-to-neo4j
   ```
