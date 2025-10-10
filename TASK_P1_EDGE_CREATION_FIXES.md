# Task: Fix Layer 2 & 3 Edge Creation (P1 - HIGH)

**Priority:** P1 - MISSING ADVERTISED FEATURES
**Estimated Time:** 3-5 hours
**Gap References:**
- Gap A1 (CO_CHANGED edges) - [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md#finding-2-co_changed-edges-not-created-in-neo4j-gap-a1-confirmed--high)
- Gap B1 (CAUSED_BY edges) - [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md#finding-3-caused_by-edges-not-created-gap-b1---new-discovery--high)

---

## Context & Problem Statement

**Two Related Issues:**

### Issue 1: CO_CHANGED Edges Not Persisted (Gap A1)
- **Evidence:** Test 1.3 - Neo4j query returns 0 CO_CHANGED edges
- **Root Cause:** Init-local timeout during git history parsing OR silent Neo4j transaction failure
- **Impact:** Layer 2 queries fail, co-change metric always 0, Phase 2 missing temporal data

### Issue 2: CAUSED_BY Edges Not Created (Gap B1)
- **Evidence:** Test 2.2 - Incident node exists, but no CAUSED_BY edge to File
- **Root Cause:** Edge creation code may not be calling Neo4j properly
- **Impact:** Incident-based risk assessment incomplete, blast radius queries fail

**What Works:**
- ✅ Code exists for both edge types
- ✅ PostgreSQL incident linking works
- ✅ Neo4j Incident nodes created
- ✅ Temporal analysis calculates co-changes

**What's Broken:**
- ❌ CO_CHANGED edges: 0 in Neo4j (expected: hundreds)
- ❌ CAUSED_BY edges: 0 in Neo4j (expected: 1 per incident link)

---

## Before You Start

### 1. Read Documentation (REQUIRED)

```bash
# Architecture & Graph Schema
cat dev_docs/01-architecture/graph_ontology.md | grep -A 20 "CO_CHANGED\|CAUSED_BY"
cat dev_docs/01-architecture/system_overview_layman.md | grep -A 30 "Layer 2\|Layer 3"

# Existing Implementation
cat internal/graph/builder.go | grep -B 5 -A 30 "AddLayer2CoChangedEdges"
cat internal/incidents/linker.go | grep -B 5 -A 30 "CreateEdge"
cat internal/ingestion/processor.go | grep -B 10 -A 20 "temporal analysis"
```

### 2. Verify Current State

```bash
# Check what's in Neo4j now
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"
# Expected: 0

docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CAUSED_BY]->() RETURN count(r)"
# Expected: 0

# Check if code exists
grep -r "AddLayer2CoChangedEdges" internal/
grep -r "CAUSED_BY" internal/incidents/
```

---

## Part 1: Fix CO_CHANGED Edge Creation (Gap A1)

### Task 1.1: Add Timeout & Better Logging to Temporal Analysis

**File:** `internal/ingestion/processor.go`
**Location:** Lines 143-161 (temporal analysis section)

**Problem:** Git history parsing may timeout (>2 min), or Neo4j transaction fails silently

**Solution:**

```go
// Replace lines 143-161 with improved version:

// Step 6: Add Layer 2 (Temporal Analysis)
slog.Info("starting temporal analysis", "window_days", 90)

// Add timeout for git history parsing
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
defer cancel()

// Parse git history with timeout
type parseResult struct {
    commits []temporal.Commit
    err     error
}

resultChan := make(chan parseResult, 1)
go func() {
    commits, err := temporal.ParseGitHistory(repoPath, 90)
    resultChan <- parseResult{commits: commits, err: err}
}()

var commits []temporal.Commit
var err error

select {
case <-ctx.Done():
    slog.Warn("temporal analysis timeout",
        "timeout", "3 minutes",
        "recommendation", "reduce window_days or optimize git parsing")
    return result, nil  // Don't fail entire init-local
case res := <-resultChan:
    commits = res.commits
    err = res.err
}

if err != nil {
    slog.Error("temporal analysis failed", "error", err)
    return result, nil  // Don't fail entire init-local
}

if len(commits) == 0 {
    slog.Warn("no commits found in git history", "window_days", 90)
    return result, nil
}

slog.Info("git history parsed",
    "commits", len(commits),
    "window_days", 90)

// Calculate co-changes and ownership
developers := temporal.ExtractDevelopers(commits)
coChanges := temporal.CalculateCoChanges(commits, 0.3)  // 30% frequency threshold

slog.Info("co-changes calculated",
    "total_pairs", len(coChanges),
    "min_frequency", 0.3)

// Store in Neo4j
if p.graphBuilder != nil && len(coChanges) > 0 {
    slog.Info("storing CO_CHANGED edges", "count", len(coChanges))

    stats, err := p.graphBuilder.AddLayer2CoChangedEdges(ctx, coChanges)
    if err != nil {
        slog.Error("failed to store CO_CHANGED edges", "error", err, "count", len(coChanges))
        // Continue - don't fail entire init-local
    } else {
        slog.Info("temporal analysis complete",
            "commits", len(commits),
            "developers", len(developers),
            "co_change_edges", stats.Edges)

        // Verify edges were actually created
        verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer verifyCancel()

        verifyCount, verifyErr := p.verifyCoChangedEdges(verifyCtx)
        if verifyErr != nil {
            slog.Error("edge verification failed", "error", verifyErr)
        } else if verifyCount != stats.Edges {
            slog.Warn("edge count mismatch",
                "expected", stats.Edges,
                "actual", verifyCount,
                "possible_transaction_issue", true)
        } else {
            slog.Info("edge verification passed", "count", verifyCount)
        }
    }
} else if len(coChanges) == 0 {
    slog.Warn("no co-changes found", "min_frequency", 0.3)
}
```

---

### Task 1.2: Add Edge Verification Method

**File:** `internal/ingestion/processor.go`
**Location:** Add new method to Processor struct

```go
// verifyCoChangedEdges queries Neo4j to confirm edges were created
func (p *Processor) verifyCoChangedEdges(ctx context.Context) (int, error) {
    if p.graphBuilder == nil || p.graphBuilder.backend == nil {
        return 0, fmt.Errorf("graph backend not available")
    }

    // Query Neo4j directly
    query := "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count"

    // Use backend's query method
    // Note: Adjust based on actual Backend interface
    result, err := p.graphBuilder.backend.Query(ctx, query)
    if err != nil {
        return 0, fmt.Errorf("verification query failed: %w", err)
    }

    // Extract count from result
    if len(result) > 0 {
        if count, ok := result[0]["count"].(int64); ok {
            return int(count), nil
        }
    }

    return 0, fmt.Errorf("unexpected query result format")
}
```

---

### Task 1.3: Check Neo4j Transaction Commit

**File:** `internal/graph/builder.go`
**Location:** Line 429-471 (AddLayer2CoChangedEdges method)

**Verify transaction commits properly:**

```go
// AddLayer2CoChangedEdges creates CO_CHANGED edges from temporal analysis
func (b *Builder) AddLayer2CoChangedEdges(ctx context.Context, coChanges []temporal.CoChangeResult) (*BuildStats, error) {
    stats := &BuildStats{}
    var edges []GraphEdge

    for _, cc := range coChanges {
        // Forward edge
        edge := GraphEdge{
            Label: "CO_CHANGED",
            From:  fmt.Sprintf("file:%s", cc.FileA),
            To:    fmt.Sprintf("file:%s", cc.FileB),
            Properties: map[string]interface{}{
                "frequency":   cc.Frequency,
                "co_changes":  cc.CoChanges,
                "window_days": cc.WindowDays,
            },
        }
        edges = append(edges, edge)

        // Reverse edge (CO_CHANGED is bidirectional)
        reverseEdge := GraphEdge{
            Label: "CO_CHANGED",
            From:  fmt.Sprintf("file:%s", cc.FileB),
            To:    fmt.Sprintf("file:%s", cc.FileA),
            Properties: map[string]interface{}{
                "frequency":   cc.Frequency,
                "co_changes":  cc.CoChanges,
                "window_days": cc.WindowDays,
            },
        }
        edges = append(edges, reverseEdge)
    }

    // Batch create edges
    if len(edges) > 0 {
        log.Printf("Creating %d CO_CHANGED edges in batches...", len(edges))

        // Create in batches of 100 to avoid large transactions
        batchSize := 100
        for i := 0; i < len(edges); i += batchSize {
            end := i + batchSize
            if end > len(edges) {
                end = len(edges)
            }

            batch := edges[i:end]
            if err := b.backend.CreateEdges(batch); err != nil {
                return stats, fmt.Errorf("failed to create CO_CHANGED edges (batch %d-%d): %w", i, end, err)
            }

            log.Printf("  ✓ Created batch %d-%d (%d edges)", i, end, len(batch))
        }

        stats.Edges = len(edges)
        log.Printf("  ✓ Created %d CO_CHANGED edges total", len(edges))
    }

    return stats, nil
}
```

---

## Part 2: Fix CAUSED_BY Edge Creation (Gap B1)

### Task 2.1: Debug and Fix Edge Creation in Linker

**File:** `internal/incidents/linker.go`
**Location:** Lines 108-120 (edge creation code)

**Current Code Review:**
```bash
cat internal/incidents/linker.go | sed -n '108,120p'
```

**Enhanced Implementation:**

```go
// Replace lines 108-120 with improved version:

// Create CAUSED_BY edge in Neo4j: (Incident)-[:CAUSED_BY]->(File)
edgeProps := map[string]interface{}{
    "confidence": link.Confidence,
    "created_at": time.Now().Unix(),
}

if lineNumber > 0 {
    edgeProps["line_number"] = lineNumber
}
if function != "" {
    edgeProps["blamed_function"] = function
}

edge := GraphEdge{
    Label:      "CAUSED_BY",
    From:       incident.ID.String(),  // Incident node ID
    To:         filePath,               // File node path
    Properties: edgeProps,
}

// Create edge in Neo4j with error handling
log.Printf("Creating CAUSED_BY edge: incident=%s -> file=%s", incident.ID, filePath)

if err := l.graph.CreateEdge(edge); err != nil {
    return fmt.Errorf("create CAUSED_BY edge: incident=%s file=%s: %w",
        incident.ID, filePath, err)
}

// Verify edge was created (development/debug only)
if os.Getenv("CODERISK_DEBUG") == "true" {
    verifyQuery := fmt.Sprintf(
        "MATCH (i:Incident {id: '%s'})-[r:CAUSED_BY]->(f:File {path: '%s'}) RETURN count(r) as count",
        incident.ID, filePath)

    // Note: Add verification logic here if Backend supports raw queries
    log.Printf("✓ CAUSED_BY edge created: %s -> %s", incident.ID, filePath)
}

return nil
```

---

### Task 2.2: Verify Graph Client Implementation

**File:** `internal/incidents/linker.go`
**Location:** Lines 11-15 (GraphClient interface)

**Current Interface:**
```go
type GraphClient interface {
    CreateNode(node GraphNode) (string, error)
    CreateEdge(edge GraphEdge) error
}
```

**Issue Check:** Ensure the `graph.Backend` actually implements this interface

**Add Adapter if Needed:**

```go
// If graph.Backend doesn't match GraphClient interface, create adapter

// graphBackendAdapter wraps graph.Backend to match GraphClient interface
type graphBackendAdapter struct {
    backend graph.Backend
}

func newGraphBackendAdapter(backend graph.Backend) GraphClient {
    return &graphBackendAdapter{backend: backend}
}

func (a *graphBackendAdapter) CreateNode(node GraphNode) (string, error) {
    // Convert to graph.Backend format
    graphNode := graph.GraphNode{
        Label:      node.Label,
        ID:         node.ID,
        Properties: node.Properties,
    }

    return a.backend.CreateNode(graphNode)
}

func (a *graphBackendAdapter) CreateEdge(edge GraphEdge) error {
    // Convert to graph.Backend format
    graphEdge := graph.GraphEdge{
        Label:      edge.Label,
        From:       edge.From,
        To:         edge.To,
        Properties: edge.Properties,
    }

    return a.backend.CreateEdge(graphEdge)
}
```

**Update NewLinker:**
```go
// Update constructor if adapter needed
func NewLinker(db *Database, backend graph.Backend) *Linker {
    return &Linker{
        db:    db,
        graph: newGraphBackendAdapter(backend),  // Wrap if needed
    }
}
```

---

### Task 2.3: Fix CLI Null Handling Bug

**File:** `internal/incidents/search.go`
**Location:** Search result scanning

**Issue:** NULL values in `impact` column cause scan error

**Fix:**

```go
// SearchIncidents performs BM25 full-text search
func (d *Database) SearchIncidents(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    // ... existing query setup ...

    rows, err := d.db.QueryContext(ctx, sqlQuery, tsQuery, limit)
    if err != nil {
        return nil, fmt.Errorf("search query: %w", err)
    }
    defer rows.Close()

    var results []SearchResult
    for rows.Next() {
        var result SearchResult

        // Use sql.NullString for nullable fields
        var impact, rootCause sql.NullString

        err := rows.Scan(
            &result.ID,
            &result.Title,
            &result.Description,
            &result.Severity,
            &result.OccurredAt,
            &result.ResolvedAt,
            &rootCause,  // ← Changed to sql.NullString
            &impact,     // ← Changed to sql.NullString
            &result.Rank,
        )
        if err != nil {
            return nil, fmt.Errorf("scan search result: %w", err)
        }

        // Convert nullable fields
        if rootCause.Valid {
            result.RootCause = rootCause.String
        }
        if impact.Valid {
            result.Impact = impact.String
        }

        results = append(results, result)
    }

    return results, nil
}
```

---

## Testing Instructions

### Test 1: CO_CHANGED Edge Creation

```bash
# Clean Neo4j (optional - for clean test)
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() DELETE r"

# Run init-local on test repo
cd /tmp/omnara
/path/to/crisk init-local

# Verify edges created
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as total"

# Expected: >0 (hundreds for omnara repo)

# Test specific query
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
   WHERE r.frequency >= 0.7
   RETURN f.path as file_a, other.path as file_b, r.frequency
   ORDER BY r.frequency DESC
   LIMIT 5"

# Expected: Top co-changed file pairs with frequency scores
```

### Test 2: CAUSED_BY Edge Creation

```bash
# Create incident
INCIDENT_ID=$(./crisk incident create "Test incident" "Test description" --severity high | grep -oP 'ID: \K[a-f0-9-]+')

# Link to file
./crisk incident link "$INCIDENT_ID" "apps/web/src/app/page.tsx" --line 10 --function "Home"

# Verify in Neo4j
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f:File)
   RETURN i.title, f.path, r.confidence, r.line_number"

# Expected: 1 row with incident title, file path, confidence=1.0, line=10
```

### Test 3: Incident Search (Null Fix)

```bash
# Test search command
./crisk incident search "test"

# Expected: Results without NULL errors
# Should show:
# Test incident (severity: high)
# [... other results ...]
```

### Test 4: Performance Validation

```bash
# Time co-change query
time docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (f:File {path: 'apps/web/src/app/page.tsx'})-[r:CO_CHANGED]-(other)
   RETURN other.path, r.frequency
   ORDER BY r.frequency DESC
   LIMIT 10"

# Expected: <20ms (target from spec)
```

---

## Validation Criteria

**Success Criteria:**

**CO_CHANGED Edges (Gap A1):**
- [ ] ✅ Init-local completes without timeout
- [ ] ✅ CO_CHANGED edges exist in Neo4j (>0 count)
- [ ] ✅ Edge count matches calculated co-changes
- [ ] ✅ Verification query passes
- [ ] ✅ Query performance <20ms
- [ ] ✅ Structured logging shows edge creation

**CAUSED_BY Edges (Gap B1):**
- [ ] ✅ Incident link creates edge in Neo4j
- [ ] ✅ Edge properties include confidence, line_number
- [ ] ✅ Query returns edge with correct incident → file relationship
- [ ] ✅ Error handling provides clear messages

**Incident Search:**
- [ ] ✅ Search command handles NULL values correctly
- [ ] ✅ Returns results without scan errors
- [ ] ✅ BM25 ranking works (<50ms)

**Code Quality:**
- [ ] ✅ Timeout handling for long-running operations
- [ ] ✅ Structured logging (slog) throughout
- [ ] ✅ Error messages include context
- [ ] ✅ Batching for large edge creation

---

## Integration Tests to Add

Create `test/integration/test_layer2_validation.sh`:

```bash
#!/bin/bash
set -e

echo "Testing Layer 2 (CO_CHANGED) edge creation..."

# Run init-local
./crisk init-local

# Query edges
COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)" | tail -1)

if [ "$COUNT" -gt 0 ]; then
    echo "✅ PASS: $COUNT CO_CHANGED edges created"
    exit 0
else
    echo "❌ FAIL: No CO_CHANGED edges found"
    exit 1
fi
```

Create `test/integration/test_layer3_validation.sh`:

```bash
#!/bin/bash
set -e

echo "Testing Layer 3 (CAUSED_BY) edge creation..."

# Create incident
INCIDENT_ID=$(./crisk incident create "Test" "Description" --severity high | grep -oP 'ID: \K[a-f0-9-]+')

# Link to file
./crisk incident link "$INCIDENT_ID" "src/test.ts" --line 1

# Verify edge
COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f) RETURN count(r)" | tail -1)

if [ "$COUNT" -eq 1 ]; then
    echo "✅ PASS: CAUSED_BY edge created"
    exit 0
else
    echo "❌ FAIL: CAUSED_BY edge not found"
    exit 1
fi
```

---

## Commit Message Template

```
Fix Layer 2 & 3 edge creation in Neo4j

**CO_CHANGED Edges (Gap A1):**
- Add timeout handling for git history parsing (3min limit)
- Add edge verification after creation
- Batch edge creation (100 per transaction)
- Enhanced logging for debugging
- Verify edges persist correctly

**CAUSED_BY Edges (Gap B1):**
- Fix edge creation in incident linker
- Add error handling and logging
- Verify graph client interface compatibility
- Add debug mode verification

**Incident Search:**
- Fix NULL handling in search results
- Use sql.NullString for nullable fields

Fixes: Gap A1, Gap B1
Tests: Added integration tests for both layers
Performance: CO_CHANGED query <20ms, incident search <50ms

- internal/ingestion/processor.go: Add timeout & verification
- internal/graph/builder.go: Batch edge creation
- internal/incidents/linker.go: Fix edge creation & logging
- internal/incidents/search.go: Fix NULL handling
- test/integration/: Add validation tests
```

---

## Success Validation

Re-run failed tests:

```bash
# Test 1.3: Layer 2 CO_CHANGED Edges
docker exec coderisk-neo4j cypher-shell "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"
# Expected: ✅ >0 (was 0)

# Test 2.2: Layer 3 CAUSED_BY Edges
# Create incident, link, verify edge exists
# Expected: ✅ 1 edge (was 0)

# Test 2.3: Incident Search
./crisk incident search "timeout"
# Expected: ✅ Results without NULL error
```

**This task is COMPLETE when:**
- Both edge types persist to Neo4j correctly ✅
- Verification queries confirm edge existence ✅
- Performance targets met (<20ms, <50ms) ✅
- Integration tests pass ✅
