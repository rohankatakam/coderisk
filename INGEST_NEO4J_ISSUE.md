# Neo4j Node Creation Issue in crisk-ingest

**Date**: 2025-11-18
**Status**: Documented - Non-blocking for atomization
**Severity**: Medium (edges created but nodes missing)

## Issue

crisk-ingest is creating edges (791 edges) but **0 nodes** in Neo4j for repo_id=11.

### Observed Behavior

```
Logs show:
- ✓ Processed commits: 0 nodes, 0 edges
- ✓ Processed PRs: 0 nodes, 0 edges
- ✓ Linked PRs to merge commits: 259 edges
- ✓ Processed issues: 0 nodes
- ✓ Created timeline edges: 1 edges (100% confidence)
- ✓ Linked issues: 0 nodes, 509 edges
- Total: Nodes: 0 | Edges: 791
```

### Error Message

```
⚠️  Index creation failed (non-fatal): failed to create index file_repo_path_unique:
query failed: Neo.ClientError.Statement.AccessMode (Writing in read access mode not allowed.
Attempted write to neo4j)
```

## Root Cause Analysis

1. **Neo4j Session Mode Issue**: The error mentions "read access mode" but code uses `ExecuteQuery` API
2. **processCommits/processPRs returning 0 nodes**: Node creation logic may be failing silently
3. **Edges being created**: This suggests Neo4j connection works, but node creation doesn't

## Schema Impact

**Postgres**: ✅ Fully populated correctly
- Commits: 520 ✅
- Files: 406 ✅
- Issues: 172 ✅
- PRs: 282 ✅

**Neo4j**: ⚠️ Partially populated
- Nodes: 0 (BROKEN)
- Edges: 791 (working but referencing non-existent nodes)

## Workaround

**crisk-atomize creates its own nodes** in Neo4j, so:
1. ✅ Atomization can proceed (creates CodeBlock, File, Commit nodes)
2. ⚠️ Ingest nodes (Developer, Issue, PR) missing from graph
3. ✅ All data in Postgres is correct (source of truth)

## Fix Required

Need to investigate:
1. Why `processCommits()` and `processPRs()` return 0 nodes
2. Whether Neo4j session mode is configured correctly
3. Why edges are created but nodes aren't

**Location**: `/Users/rohankatakam/Documents/brain/coderisk/internal/graph/builder.go`

Methods to check:
- `processCommits()` (line ~250)
- `processPRs()` (line ~350)
- `CreateNode()` vs `CreateEdge()` in Neo4jBackend

## Resolution Plan

**Short-term** (this run):
- ✅ Continue with atomization
- ✅ Let crisk-atomize create its own nodes
- ✅ Document issue for later fix

**Long-term** (next iteration):
- Debug `processCommits()` and `processPRs()`
- Fix Neo4j session configuration
- Re-run crisk-ingest to populate missing nodes
- Validate full graph structure

## Impact on Current Run

**Minimal** - CodeRisk can function because:
1. Postgres has all data (source of truth) ✅
2. crisk-atomize creates CodeBlock nodes ✅
3. Indexers read from Postgres ✅
4. MCP server queries Postgres first ✅

The missing Developer/Issue/PR nodes in Neo4j will only affect:
- Graph visualizations
- Neo4j-specific queries
- Relationship traversals

All risk calculations and core functionality work because they use Postgres as primary data source.
