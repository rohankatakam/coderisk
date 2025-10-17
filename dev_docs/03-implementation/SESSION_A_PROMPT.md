# Session A: Temporal Analysis Implementation

**Duration:** Weeks 2-3 (1.5-2 weeks)
**Package:** `internal/temporal/` (you own this entirely)
**Goal:** Parse git history, calculate CO_CHANGED edges, track ownership

---

## Your Mission

Implement **Layer 2 (Temporal)** of the graph ontology:
1. Parse git commit history (last 90 days)
2. Extract Developer and Commit nodes
3. Calculate CO_CHANGED edge weights (files that change together)
4. Track ownership transitions (who owns which files)

**Why this matters:** Temporal coupling (files changing together) is a leading indicator of risk. If `auth.ts` and `database.ts` change together 85% of the time, changing one without the other is risky.

---

## What You Own (No Conflicts!)

### Files You Create
- `internal/temporal/git_history.go` - Parse `git log` output
- `internal/temporal/co_change.go` - Calculate CO_CHANGED frequencies
- `internal/temporal/developer.go` - Track Developer/Commit nodes
- `internal/temporal/types.go` - Data structures
- `internal/temporal/git_history_test.go` - Unit tests
- `test/integration/test_temporal_analysis.sh` - E2E test

### Files You Modify (Small Changes)
- `internal/ingestion/processor.go` - Add temporal ingestion step (~30 lines)
- `internal/graph/builder.go` - Add Layer 2 node/edge creation (~50 lines)

### Files You Read (No Modification)
- `internal/graph/neo4j_backend.go` - Use existing CreateNode/CreateEdge
- `dev_docs/01-architecture/graph_ontology.md` - Layer 2 spec

---

## Technical Specification

### Input
```bash
# Your code will execute this:
git log --since="90 days ago" --numstat --pretty=format:"%H|%an|%ae|%ad|%s"

# Output format:
abc123|John Doe|john@example.com|2025-09-15|Fix auth bug
10      5       src/auth.ts
3       1       src/database.ts

def456|Jane Smith|jane@example.com|2025-09-16|Add caching
25      0       src/cache.ts
5       2       src/database.ts
```

### Data Structures to Create

**File:** `internal/temporal/types.go`
```go
package temporal

import "time"

// Commit represents a git commit
type Commit struct {
    SHA         string
    Author      string
    Email       string
    Timestamp   time.Time
    Message     string
    FilesChanged []FileChange
}

// FileChange represents file modifications in a commit
type FileChange struct {
    Path      string
    Additions int
    Deletions int
}

// Developer represents a code contributor
type Developer struct {
    Email      string
    Name       string
    FirstCommit time.Time
    LastCommit  time.Time
    TotalCommits int
}

// CoChangeResult represents files that change together
type CoChangeResult struct {
    FileA      string
    FileB      string
    Frequency  float64  // 0.0 to 1.0 (how often they change together)
    CoChanges  int      // absolute count
    WindowDays int      // 90
}

// OwnershipHistory tracks who owns a file
type OwnershipHistory struct {
    FilePath       string
    CurrentOwner   string  // email with most commits
    PreviousOwner  string  // previous primary contributor
    TransitionDate time.Time
    DaysSince      int
}
```

### Key Functions to Implement

**File:** `internal/temporal/git_history.go`
```go
// ParseGitHistory executes git log and parses output
func ParseGitHistory(repoPath string, days int) ([]Commit, error) {
    // 1. Execute: git log --since="X days ago" --numstat --pretty=format:"%H|%an|%ae|%ad|%s"
    // 2. Parse output line by line
    // 3. Build Commit structs with FilesChanged
    // 4. Return commits
}

// ExtractDevelopers builds Developer nodes from commits
func ExtractDevelopers(commits []Commit) []Developer {
    // Group commits by email
    // Calculate first/last commit dates
    // Return unique developers
}
```

**File:** `internal/temporal/co_change.go`
```go
// CalculateCoChanges finds files that frequently change together
func CalculateCoChanges(commits []Commit, minFrequency float64) []CoChangeResult {
    // Algorithm:
    // 1. For each commit, get pairs of files changed
    // 2. Count how many times each pair appears
    // 3. Calculate frequency = pair_count / total_commits_with_either_file
    // 4. Filter by minFrequency (e.g., 0.3 = 30%)
    // 5. Return sorted by frequency
}

// GetCoChangedFiles returns files that change with target (PUBLIC - Session C uses this)
func GetCoChangedFiles(filePath string, minFrequency float64) ([]CoChangeResult, error) {
    // Query Neo4j for CO_CHANGED edges from filePath
    // Filter by frequency >= minFrequency
    // Return results
}
```

**File:** `internal/temporal/developer.go`
```go
// CalculateOwnership determines primary owner of each file
func CalculateOwnership(commits []Commit) map[string]*OwnershipHistory {
    // For each file:
    // 1. Count commits per developer
    // 2. Current owner = developer with most commits
    // 3. Previous owner = second most (if different)
    // 4. Transition date = when current owner overtook previous
    // 5. Return ownership map
}

// GetOwnershipHistory returns ownership for a file (PUBLIC - Session C uses this)
func GetOwnershipHistory(filePath string) (*OwnershipHistory, error) {
    // Query graph or cache for ownership data
    // Return OwnershipHistory
}
```

### Neo4j Graph Nodes & Edges to Create

**Nodes:**
```cypher
(:Commit {sha, author, email, timestamp, message, additions, deletions})
(:Developer {email, name, first_commit, last_commit, total_commits})
```

**Edges:**
```cypher
(Developer)-[:AUTHORED]->(Commit)
(Commit)-[:MODIFIES {additions, deletions}]->(File)
(File)-[: CO_CHANGED {frequency, co_changes, window_days}]-(File)
```

**Your code in `internal/graph/builder.go` addition:**
```go
// AddLayer2Temporal adds temporal nodes and edges to graph
func (b *Builder) AddLayer2Temporal(commits []temporal.Commit, coChanges []temporal.CoChangeResult) error {
    // Create Developer nodes
    // Create Commit nodes
    // Create AUTHORED edges
    // Create MODIFIES edges
    // Create CO_CHANGED edges
}
```

---

## Integration Points

### 1. Ingestion Hook (Modify: `internal/ingestion/processor.go`)

Find the `ProcessRepository` function and add this step **after** Layer 1:

```go
// After Step 5: Build graph (Layer 1)
if p.graphClient != nil {
    if err := p.buildGraph(ctx, allEntities); err != nil {
        return nil, fmt.Errorf("failed to build graph: %w", err)
    }
    slog.Info("graph construction complete")

    // NEW: Add Layer 2 (Temporal Analysis)
    slog.Info("starting temporal analysis", "window_days", 90)
    commits, err := temporal.ParseGitHistory(repoPath, 90)
    if err != nil {
        slog.Warn("temporal analysis failed", "error", err)
    } else {
        developers := temporal.ExtractDevelopers(commits)
        coChanges := temporal.CalculateCoChanges(commits, 0.3) // min 30% frequency

        if err := p.graphClient.AddLayer2Temporal(commits, coChanges); err != nil {
            slog.Warn("failed to store temporal data", "error", err)
        } else {
            slog.Info("temporal analysis complete",
                "commits", len(commits),
                "developers", len(developers),
                "co_change_edges", len(coChanges))
        }
    }
}
```

---

## Testing Strategy

### Unit Tests (`internal/temporal/git_history_test.go`)

```go
func TestParseGitHistory(t *testing.T) {
    // Create test git repo
    // Add commits with file changes
    // Parse history
    // Verify commits extracted correctly
}

func TestCalculateCoChanges(t *testing.T) {
    commits := []Commit{
        {FilesChanged: []FileChange{{Path: "a.ts"}, {Path: "b.ts"}}},
        {FilesChanged: []FileChange{{Path: "a.ts"}, {Path: "b.ts"}}},
        {FilesChanged: []FileChange{{Path: "a.ts"}, {Path: "c.ts"}}},
    }

    coChanges := CalculateCoChanges(commits, 0.5)

    // Expect: a.ts <-> b.ts frequency = 2/3 = 0.67 (above 0.5 threshold)
    //         a.ts <-> c.ts frequency = 1/3 = 0.33 (below threshold)
    // Should return only a.ts <-> b.ts
}
```

### Integration Test (`test/integration/test_temporal_analysis.sh`)

```bash
#!/bin/bash

# Test temporal analysis on real repository
cd /tmp/omnara

# Run init with temporal analysis
/path/to/crisk init-local

# Verify CO_CHANGED edges exist
EDGE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "..." \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)" | tail -1)

if [ "$EDGE_COUNT" -lt 100 ]; then
    echo "ERROR: Expected >100 CO_CHANGED edges, got $EDGE_COUNT"
    exit 1
fi

# Test ownership query
OWNER=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "..." \
  "MATCH (f:File {path: '/tmp/omnara/src/server.ts'})<-[:MODIFIES]-(c:Commit)<-[:AUTHORED]-(d:Developer) \
   RETURN d.email ORDER BY count(c) DESC LIMIT 1" | tail -1)

echo "✅ Temporal analysis working. Edges: $EDGE_COUNT, Top owner: $OWNER"
```

---

## Checkpoints

### Checkpoint A1: Git History Parsing ✅

**What to verify:**
```bash
go test ./internal/temporal/... -v -run TestParseGitHistory
```

**Ask me:**
> ✅ Git history parsing works! Parsed 1,523 commits from last 90 days. 47 developers identified. Sample commit: [show first commit]. Ready to calculate CO_CHANGED?

---

### Checkpoint A2: CO_CHANGED Calculation ✅

**What to verify:**
```bash
# Query Neo4j
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"

# Show sample edges
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (a)-[r:CO_CHANGED]->(b) RETURN a.name, b.name, r.frequency ORDER BY r.frequency DESC LIMIT 10"
```

**Ask me:**
> ✅ CO_CHANGED edges created! Total: 847 edges. Top pair: auth.ts <-> database.ts (frequency: 0.89). Frequency range: 0.30-0.95. Complete?

---

### Checkpoint A3: Ownership Tracking ✅

**What to verify:**
```bash
go run scripts/test_ownership.go /tmp/omnara/src/server.ts

# Should output:
# Current owner: alice@company.com (23 commits)
# Previous owner: bob@company.com (12 commits)
# Transition date: 2025-09-01
# Days since transition: 34
```

**Ask me:**
> ✅ Ownership tracking works! Tested on 10 files. Transition detection accurate. Ready for integration with Phase 2?

---

## Success Criteria

- [ ] `ParseGitHistory()` returns commits with file changes
- [ ] `CalculateCoChanges()` returns pairs with frequency >0.3
- [ ] `GetCoChangedFiles()` queries Neo4j successfully
- [ ] `GetOwnershipHistory()` returns current/previous owners
- [ ] Layer 2 nodes created in Neo4j (Commit, Developer)
- [ ] CO_CHANGED edges created with frequency property
- [ ] Integration test passes on real repository
- [ ] Unit tests: >70% coverage

---

## Performance Targets

- Parse git history (5K commits): <5s
- Calculate CO_CHANGED (500 file pairs): <2s
- Store in Neo4j (2K nodes + 1K edges): <3s
- Total temporal analysis: <10s

---

## References

**Read these first:**
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Layer 2 specification (lines 44-83)
- [PARALLEL_SESSION_PLAN_WEEKS2-8.md](PARALLEL_SESSION_PLAN_WEEKS2-8.md) - Your file ownership
- [GRAPH_INVESTIGATION_REPORT.md](../../GRAPH_INVESTIGATION_REPORT.md) - How graph construction works

**Your interfaces (Session C will use these):**
- `GetCoChangedFiles(filePath, minFreq) -> []CoChangeResult`
- `GetOwnershipHistory(filePath) -> *OwnershipHistory`

---

## What NOT to Do

❌ Don't modify Session B's files (`internal/incidents/`)
❌ Don't modify Session C's files (`internal/agent/`)
❌ Don't change `cmd/crisk/check.go` (Session C owns Phase 2 integration)
❌ Don't create new database tables (use Neo4j only for Layer 2)

---

## Tips for Success

1. **Start with git parsing** - Get commits first, everything else depends on it
2. **Use small batches** - Parse 1000 commits at a time (pagination)
3. **Cache in Redis** - Co-change queries are expensive, cache results (15-min TTL)
4. **Test on real repos** - Use omnara-ai/omnara for realistic data
5. **Ask at checkpoints** - Don't proceed past A1/A2/A3 without confirmation

---

## Getting Started

```bash
# 1. Create your package
mkdir -p internal/temporal
touch internal/temporal/types.go
touch internal/temporal/git_history.go
touch internal/temporal/co_change.go
touch internal/temporal/developer.go

# 2. Define types first
# Edit internal/temporal/types.go (copy from above)

# 3. Implement git parsing
# Edit internal/temporal/git_history.go

# 4. Test as you go
go test ./internal/temporal/... -v

# 5. Integrate with processor
# Edit internal/ingestion/processor.go (add ~30 lines)
```

---

**You are Session A. Your mission: Make temporal analysis work. Go! 🚀**
