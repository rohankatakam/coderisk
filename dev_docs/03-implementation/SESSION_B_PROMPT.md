# Session B: Incident Database Implementation

**Duration:** Weeks 4-5 (1.5-2 weeks)
**Package:** `internal/incidents/` (you own this entirely)
**Goal:** PostgreSQL schema, BM25 search, manual incident linking

---

## Your Mission

Implement **Layer 3 (Incidents)** of the graph ontology:
1. Create PostgreSQL schema for incidents with full-text search
2. Implement BM25-style similarity search using tsvector + GIN index
3. Build manual incident linking interface (CLI for MVP)
4. Store Incident nodes in Neo4j with CAUSED_BY edges

**Why this matters:** Historical incidents are the #1 predictor of future bugs. If `payment_processor.py` caused 3 prod incidents in the last 90 days, changing it again is HIGH RISK.

---

## What You Own (No Conflicts!)

### Files You Create
- `internal/incidents/database.go` - PostgreSQL schema, CRUD operations
- `internal/incidents/search.go` - BM25 similarity search with tsvector
- `internal/incidents/linker.go` - Manual incident-to-file linking
- `internal/incidents/types.go` - Data structures
- `internal/incidents/database_test.go` - Unit tests
- `test/integration/test_incident_search.sh` - E2E test

### Files You Modify (Small Changes)
- `internal/storage/postgres.go` - Add incident table migration (~40 lines)
- `internal/graph/builder.go` - Add Layer 3 node/edge creation (~50 lines)
- `cmd/crisk/link-incident.go` - New CLI command for incident linking (~100 lines)

### Files You Read (No Modification)
- `internal/graph/neo4j_backend.go` - Use existing CreateNode/CreateEdge
- `dev_docs/01-architecture/decisions/003-postgresql-fulltext-search.md` - ADR-003 spec
- `dev_docs/01-architecture/graph_ontology.md` - Layer 3 spec

---

## Technical Specification

### PostgreSQL Schema

**File:** `internal/storage/postgres.go` (add migration)

```go
// Add this migration to InitDB()
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    occurred_at TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP,
    root_cause TEXT,
    impact TEXT,

    -- Full-text search columns
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(root_cause, '')), 'C')
    ) STORED,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- GIN index for fast full-text search
CREATE INDEX idx_incidents_search ON incidents USING GIN(search_vector);

-- Index for time-based queries
CREATE INDEX idx_incidents_occurred_at ON incidents(occurred_at DESC);

-- Incident-to-file links (many-to-many)
CREATE TABLE incident_files (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    line_number INT,
    blamed_function TEXT,
    confidence FLOAT DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),

    PRIMARY KEY (incident_id, file_path)
);

CREATE INDEX idx_incident_files_path ON incident_files(file_path);
```

### Data Structures to Create

**File:** `internal/incidents/types.go`

```go
package incidents

import (
    "time"
    "github.com/google/uuid"
)

// Incident represents a production incident or bug
type Incident struct {
    ID          uuid.UUID
    Title       string
    Description string
    Severity    Severity
    OccurredAt  time.Time
    ResolvedAt  *time.Time
    RootCause   string
    Impact      string
    CreatedAt   time.Time
    UpdatedAt   time.Time

    // Linked files (populated on query)
    LinkedFiles []IncidentFile
}

// Severity levels
type Severity string

const (
    SeverityCritical Severity = "critical"
    SeverityHigh     Severity = "high"
    SeverityMedium   Severity = "medium"
    SeverityLow      Severity = "low"
)

// IncidentFile represents a file blamed for an incident
type IncidentFile struct {
    IncidentID      uuid.UUID
    FilePath        string
    LineNumber      int       // 0 if entire file
    BlamedFunction  string    // empty if entire file
    Confidence      float64   // 0.0-1.0 (1.0 = manual link, <1.0 = auto-inferred)
}

// SearchResult represents BM25 similarity search result
type SearchResult struct {
    Incident  Incident
    Rank      float64  // BM25 score (higher = more relevant)
    Relevance string   // "high" (>0.5), "medium" (0.2-0.5), "low" (<0.2)
}

// IncidentStats aggregates incident data for risk calculation
type IncidentStats struct {
    FilePath       string
    TotalIncidents int
    Last30Days     int
    Last90Days     int
    CriticalCount  int
    HighCount      int
    LastIncident   *time.Time
    AvgResolution  time.Duration  // Average time to resolve
}
```

### Key Functions to Implement

**File:** `internal/incidents/database.go`

```go
package incidents

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
)

// Database handles PostgreSQL operations for incidents
type Database struct {
    db *sql.DB
}

// NewDatabase creates a new incident database client
func NewDatabase(db *sql.DB) *Database {
    return &Database{db: db}
}

// CreateIncident inserts a new incident
func (d *Database) CreateIncident(ctx context.Context, inc *Incident) error {
    // INSERT INTO incidents (...) VALUES (...) RETURNING id
    // Set inc.ID from returned value
}

// GetIncident retrieves incident by ID with linked files
func (d *Database) GetIncident(ctx context.Context, id uuid.UUID) (*Incident, error) {
    // Query incident
    // Query incident_files
    // Populate LinkedFiles
}

// LinkIncidentToFile creates manual link between incident and file
func (d *Database) LinkIncidentToFile(ctx context.Context, link *IncidentFile) error {
    // INSERT INTO incident_files (...) VALUES (...)
    // Also create CAUSED_BY edge in Neo4j
}

// GetIncidentsByFile returns all incidents linked to a file
func (d *Database) GetIncidentsByFile(ctx context.Context, filePath string) ([]Incident, error) {
    // SELECT i.* FROM incidents i
    // JOIN incident_files if ON i.id = if.incident_id
    // WHERE if.file_path = $1
    // ORDER BY i.occurred_at DESC
}

// GetIncidentStats calculates aggregated stats for a file (PUBLIC - Session C uses this)
func (d *Database) GetIncidentStats(ctx context.Context, filePath string) (*IncidentStats, error) {
    // Count incidents by time window (30d, 90d, all-time)
    // Count by severity
    // Calculate avg resolution time
    // Return IncidentStats
}
```

**File:** `internal/incidents/search.go`

```go
package incidents

import (
    "context"
    "database/sql"
)

// SearchIncidents performs BM25-style full-text search
func (d *Database) SearchIncidents(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    // Algorithm:
    // 1. Use ts_rank_cd() for BM25-style ranking
    // 2. Query: SELECT *, ts_rank_cd(search_vector, query) AS rank
    //           FROM incidents, to_tsquery('english', $1) query
    //           WHERE search_vector @@ query
    //           ORDER BY rank DESC
    //           LIMIT $2
    // 3. Map rank to relevance: >0.5 = high, 0.2-0.5 = medium, <0.2 = low
    // 4. Return []SearchResult
}

// FindSimilarIncidents finds incidents similar to a given incident
func (d *Database) FindSimilarIncidents(ctx context.Context, incidentID uuid.UUID, limit int) ([]SearchResult, error) {
    // Get source incident
    // Build search query from title + description
    // Call SearchIncidents with query
    // Filter out source incident from results
}

// SearchByFile finds incidents mentioning a specific file path
func (d *Database) SearchByFile(ctx context.Context, filePath string) ([]SearchResult, error) {
    // Build query from file path (e.g., "payment_processor.py")
    // Call SearchIncidents
    // Return results
}
```

**File:** `internal/incidents/linker.go`

```go
package incidents

import (
    "context"
    "fmt"
)

// Linker handles manual incident-to-file linking
type Linker struct {
    db    *Database
    graph GraphClient  // Interface for Neo4j operations
}

// NewLinker creates a new incident linker
func NewLinker(db *Database, graph GraphClient) *Linker {
    return &Linker{db: db, graph: graph}
}

// LinkIncident creates link between incident and file (CLI command)
func (l *Linker) LinkIncident(ctx context.Context, incidentID string, filePath string, lineNumber int, function string) error {
    // 1. Validate incident exists
    // 2. Validate file exists in Neo4j
    // 3. Create link in PostgreSQL (incident_files table)
    // 4. Create CAUSED_BY edge in Neo4j: (Incident)-[:CAUSED_BY]->(File)
    // 5. Log success
}

// UnlinkIncident removes link between incident and file
func (l *Linker) UnlinkIncident(ctx context.Context, incidentID string, filePath string) error {
    // Delete from incident_files
    // Delete CAUSED_BY edge from Neo4j
}

// SuggestLinks uses BM25 search to suggest file links for an incident
func (l *Linker) SuggestLinks(ctx context.Context, incidentID string, threshold float64) ([]string, error) {
    // Get incident
    // Search for files in description/root_cause using SearchByFile()
    // Parse file paths from search results
    // Return unique file paths with confidence > threshold
}

// GraphClient interface (implement in internal/graph/builder.go)
type GraphClient interface {
    CreateIncidentNode(ctx context.Context, incident *Incident) error
    CreateCausedByEdge(ctx context.Context, incidentID, filePath string) error
}
```

### Neo4j Graph Nodes & Edges to Create

**Nodes:**
```cypher
(:Incident {
    id,            -- UUID
    title,         -- String
    severity,      -- "critical" | "high" | "medium" | "low"
    occurred_at,   -- Timestamp
    resolved_at,   -- Timestamp (nullable)
    root_cause     -- String
})
```

**Edges:**
```cypher
(Incident)-[:CAUSED_BY {confidence, line_number, blamed_function}]->(File)
```

**Your code in `internal/graph/builder.go` addition:**

```go
// AddLayer3Incidents adds incident nodes and CAUSED_BY edges to graph
func (b *Builder) AddLayer3Incidents(incidents []incidents.Incident, links []incidents.IncidentFile) error {
    // Create Incident nodes
    for _, inc := range incidents {
        properties := map[string]interface{}{
            "id":          inc.ID.String(),
            "title":       inc.Title,
            "severity":    string(inc.Severity),
            "occurred_at": inc.OccurredAt.Unix(),
        }
        if inc.ResolvedAt != nil {
            properties["resolved_at"] = inc.ResolvedAt.Unix()
        }
        if inc.RootCause != "" {
            properties["root_cause"] = inc.RootCause
        }

        if err := b.backend.CreateNode(ctx, "Incident", inc.ID.String(), properties); err != nil {
            return err
        }
    }

    // Create CAUSED_BY edges
    for _, link := range links {
        edgeProps := map[string]interface{}{
            "confidence": link.Confidence,
        }
        if link.LineNumber > 0 {
            edgeProps["line_number"] = link.LineNumber
        }
        if link.BlamedFunction != "" {
            edgeProps["blamed_function"] = link.BlamedFunction
        }

        edge := graph.GraphEdge{
            From:       link.IncidentID.String(),
            To:         link.FilePath,
            Type:       "CAUSED_BY",
            Properties: edgeProps,
        }
        if err := b.backend.CreateEdge(ctx, edge); err != nil {
            return err
        }
    }

    return nil
}
```

---

## Integration Points

### 1. CLI Command (New File: `cmd/crisk/link-incident.go`)

```go
package main

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
    "your-module/internal/incidents"
)

var linkIncidentCmd = &cobra.Command{
    Use:   "link-incident [incident-id] [file-path]",
    Short: "Link an incident to a file that caused it",
    Args:  cobra.ExactArgs(2),
    RunE:  runLinkIncident,
}

func init() {
    rootCmd.AddCommand(linkIncidentCmd)
    linkIncidentCmd.Flags().Int("line", 0, "Specific line number (0 = entire file)")
    linkIncidentCmd.Flags().String("function", "", "Function name that caused the incident")
}

func runLinkIncident(cmd *cobra.Command, args []string) error {
    incidentID := args[0]
    filePath := args[1]
    lineNumber, _ := cmd.Flags().GetInt("line")
    function, _ := cmd.Flags().GetString("function")

    // Initialize database and linker
    db := getPostgresDB()  // Helper to get connection
    incDB := incidents.NewDatabase(db)
    graphClient := getGraphClient()  // Helper to get Neo4j client
    linker := incidents.NewLinker(incDB, graphClient)

    // Create link
    ctx := context.Background()
    if err := linker.LinkIncident(ctx, incidentID, filePath, lineNumber, function); err != nil {
        return fmt.Errorf("failed to link incident: %w", err)
    }

    fmt.Printf("✅ Linked incident %s to %s\n", incidentID, filePath)
    return nil
}
```

### 2. Add to Root Command (`cmd/crisk/main.go`)

```go
// Add near other command imports
import _ "your-module/cmd/crisk/link-incident.go"
```

---

## Testing Strategy

### Unit Tests (`internal/incidents/database_test.go`)

```go
func TestCreateIncident(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    incDB := incidents.NewDatabase(db)

    inc := &incidents.Incident{
        Title:       "Payment processor timeout",
        Description: "Users unable to complete checkout due to 30s timeout",
        Severity:    incidents.SeverityCritical,
        OccurredAt:  time.Now(),
        RootCause:   "payment_processor.py missing connection pooling",
    }

    err := incDB.CreateIncident(context.Background(), inc)
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, inc.ID)
}

func TestSearchIncidents(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    incDB := incidents.NewDatabase(db)

    // Create test incidents
    incidents := []incidents.Incident{
        {Title: "Payment timeout", Description: "Timeout in payment processor"},
        {Title: "Database deadlock", Description: "Deadlock in user table"},
        {Title: "Payment failed", Description: "Payment gateway returned 500"},
    }

    for _, inc := range incidents {
        incDB.CreateIncident(context.Background(), &inc)
    }

    // Search for "payment"
    results, err := incDB.SearchIncidents(context.Background(), "payment", 10)

    assert.NoError(t, err)
    assert.Len(t, results, 2)  // Only payment-related incidents
    assert.Greater(t, results[0].Rank, results[1].Rank)  // Ranked by relevance
}
```

### Integration Test (`test/integration/test_incident_search.sh`)

```bash
#!/bin/bash

# Test incident database and BM25 search

set -e

echo "Testing Incident Database..."

# 1. Create test incident via CLI (requires crisk CLI to support this)
INCIDENT_ID=$(crisk create-incident \
    --title "Payment processor timeout" \
    --description "Users unable to checkout due to timeout in payment_processor.py" \
    --severity critical \
    --root-cause "Missing connection pooling in database client" | grep "ID:" | awk '{print $2}')

echo "Created incident: $INCIDENT_ID"

# 2. Link incident to file
crisk link-incident "$INCIDENT_ID" "src/payment_processor.py" \
    --line 142 \
    --function "process_payment"

echo "Linked incident to file"

# 3. Verify in PostgreSQL
LINK_COUNT=$(docker exec coderisk-postgres psql -U coderisk -d coderisk -t -c \
    "SELECT COUNT(*) FROM incident_files WHERE incident_id = '$INCIDENT_ID'")

if [ "$LINK_COUNT" -lt 1 ]; then
    echo "ERROR: Incident link not found in PostgreSQL"
    exit 1
fi

# 4. Verify CAUSED_BY edge in Neo4j
EDGE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (i:Incident {id: '$INCIDENT_ID'})-[:CAUSED_BY]->(f:File) RETURN count(f)" | tail -1)

if [ "$EDGE_COUNT" -lt 1 ]; then
    echo "ERROR: CAUSED_BY edge not found in Neo4j"
    exit 1
fi

# 5. Test BM25 search
SEARCH_RESULTS=$(docker exec coderisk-postgres psql -U coderisk -d coderisk -t -c \
    "SELECT COUNT(*) FROM incidents WHERE search_vector @@ to_tsquery('english', 'payment & timeout')")

if [ "$SEARCH_RESULTS" -lt 1 ]; then
    echo "ERROR: BM25 search failed to find incident"
    exit 1
fi

echo "✅ Incident database tests passed!"
echo "   - Incident created: $INCIDENT_ID"
echo "   - PostgreSQL links: $LINK_COUNT"
echo "   - Neo4j edges: $EDGE_COUNT"
echo "   - BM25 search results: $SEARCH_RESULTS"
```

---

## Checkpoints

### Checkpoint B1: PostgreSQL Schema Created ✅

**What to verify:**
```bash
# Check tables exist
docker exec coderisk-postgres psql -U coderisk -d coderisk -c "\dt"

# Verify GIN index
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
    "SELECT indexname FROM pg_indexes WHERE tablename = 'incidents'"

# Test tsvector generation
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
    "INSERT INTO incidents (title, description, severity, occurred_at) VALUES ('Test', 'Description', 'low', NOW()); SELECT search_vector FROM incidents WHERE title = 'Test'"
```

**Ask me:**
> ✅ PostgreSQL schema created! Tables: incidents (8 columns), incident_files (5 columns). GIN index on search_vector working. tsvector auto-generates correctly. Ready for BM25 search?

---

### Checkpoint B2: BM25 Search Working ✅

**What to verify:**
```bash
# Run unit tests
go test ./internal/incidents/... -v -run TestSearchIncidents

# Test real search query
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
    "SELECT title, ts_rank_cd(search_vector, query) AS rank
     FROM incidents, to_tsquery('english', 'payment & timeout') query
     WHERE search_vector @@ query
     ORDER BY rank DESC
     LIMIT 5"
```

**Ask me:**
> ✅ BM25 search working! Unit tests pass. Query "payment & timeout" returns 12 results ranked by relevance. Top result has rank 0.87. Integration with Neo4j complete (247 CAUSED_BY edges). Complete?

---

## Success Criteria

- [ ] PostgreSQL schema created with tsvector + GIN index
- [ ] `CreateIncident()` inserts with auto-generated search_vector
- [ ] `SearchIncidents()` returns BM25-ranked results in <50ms
- [ ] `LinkIncidentToFile()` creates both PostgreSQL row + Neo4j edge
- [ ] `GetIncidentStats()` aggregates counts by time window and severity
- [ ] CLI command `crisk link-incident` works end-to-end
- [ ] Layer 3 nodes created in Neo4j (Incident)
- [ ] CAUSED_BY edges created with confidence property
- [ ] Integration test passes on real repository
- [ ] Unit tests: >70% coverage

---

## Performance Targets

- PostgreSQL incident insert: <10ms
- BM25 search (1000 incidents): <50ms
- Link creation (PostgreSQL + Neo4j): <20ms
- GetIncidentStats query: <30ms
- Total incident linking flow: <100ms

---

## References

**Read these first:**
- [003-postgresql-fulltext-search.md](../01-architecture/decisions/003-postgresql-fulltext-search.md) - ADR-003 spec
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Layer 3 specification (lines 84-121)
- [PARALLEL_SESSION_PLAN_WEEKS2-8.md](PARALLEL_SESSION_PLAN_WEEKS2-8.md) - Your file ownership

**Your interfaces (Session C will use these):**
- `GetIncidentStats(filePath) -> *IncidentStats`
- `SearchIncidents(query, limit) -> []SearchResult`

---

## What NOT to Do

❌ Don't modify Session A's files (`internal/temporal/`)
❌ Don't modify Session C's files (`internal/agent/`)
❌ Don't change `cmd/crisk/check.go` (Session C owns Phase 2 integration)
❌ Don't use LanceDB or vector embeddings (use PostgreSQL FTS per ADR-003)

---

## Tips for Success

1. **Start with schema** - Get PostgreSQL tables + indexes working first
2. **Test tsvector** - Ensure search_vector auto-generates correctly
3. **Use psql for debugging** - Docker exec into PostgreSQL to test queries
4. **Manual linking first** - CLI command is simpler than auto-inference
5. **Ask at checkpoints** - Don't proceed past B1/B2 without confirmation

---

## Getting Started

```bash
# 1. Create your package
mkdir -p internal/incidents
touch internal/incidents/types.go
touch internal/incidents/database.go
touch internal/incidents/search.go
touch internal/incidents/linker.go

# 2. Define types first
# Edit internal/incidents/types.go (copy from above)

# 3. Add schema migration
# Edit internal/storage/postgres.go (add CREATE TABLE statements)

# 4. Test schema creation
docker-compose up -d postgres
docker exec coderisk-postgres psql -U coderisk -d coderisk -c "\dt"

# 5. Implement database operations
# Edit internal/incidents/database.go

# 6. Test as you go
go test ./internal/incidents/... -v
```

---

**You are Session B. Your mission: Make incident tracking work. Go! 🚀**
