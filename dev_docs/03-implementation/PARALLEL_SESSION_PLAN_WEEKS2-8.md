# Parallel Session Plan: Weeks 2-8 (Temporal Analysis, Incidents, LLM Investigation)

**Created:** October 5, 2025
**Purpose:** Coordinate 3 parallel Claude Code sessions for MVP completion
**Reference:** [status.md](status.md), [system_overview_layman.md](../01-architecture/system_overview_layman.md)

---

## Overview

This plan splits Weeks 2-8 implementation into 3 **independent, non-overlapping** sessions that can run in parallel:

- **Session A:** Temporal Analysis (Weeks 2-3) - Git history parsing, CO_CHANGED edges
- **Session B:** Incident Database (Weeks 4-5) - PostgreSQL FTS, BM25 similarity
- **Session C:** LLM Investigation (Weeks 6-8) - OpenAI/Anthropic, agentic search

**Key Design:**
- Each session owns **separate packages and files** (no collisions)
- Sessions communicate via **well-defined interfaces**
- Can run **100% in parallel** (no blocking dependencies)

**Estimated Duration:** 3-4 weeks parallel (vs 7-8 weeks sequential)

---

## File Ownership Map

### Session A: Temporal Analysis (Weeks 2-3)

**Package:** `internal/temporal/` (NEW - Session A owns entirely)

**Owns:**
- `internal/temporal/git_history.go` (NEW - parse git log)
- `internal/temporal/co_change.go` (NEW - calculate CO_CHANGED edges)
- `internal/temporal/developer.go` (NEW - Developer & Commit nodes)
- `internal/temporal/types.go` (NEW - temporal data structures)
- `internal/temporal/git_history_test.go` (NEW - unit tests)
- `test/integration/test_temporal_analysis.sh` (NEW - e2e test)

**Modifies:**
- `internal/ingestion/processor.go` (add temporal ingestion step - ~30 lines)
- `internal/graph/builder.go` (add Layer 2 node/edge creation - ~50 lines)

**Reads (no modification):**
- `internal/graph/` (existing graph backend)
- `dev_docs/01-architecture/graph_ontology.md` (Layer 2 spec)

**Dependencies:** None (can start immediately)

---

### Session B: Incident Database (Weeks 4-5)

**Package:** `internal/incidents/` (NEW - Session B owns entirely)

**Owns:**
- `internal/incidents/database.go` (NEW - PostgreSQL schema)
- `internal/incidents/search.go` (NEW - full-text search with tsvector)
- `internal/incidents/linker.go` (NEW - manual incident linking)
- `internal/incidents/types.go` (NEW - incident data structures)
- `internal/incidents/database_test.go` (NEW - unit tests)
- `migrations/002_incidents_schema.sql` (NEW - database migration)
- `test/integration/test_incidents_db.sh` (NEW - e2e test)

**Modifies:**
- `internal/storage/postgres.go` (add incident queries - ~40 lines)
- `internal/graph/builder.go` (add Layer 3 node/edge creation - ~50 lines)

**Reads (no modification):**
- `internal/storage/` (existing PostgreSQL client)
- `dev_docs/01-architecture/decisions/003-postgresql-fulltext-search.md` (FTS spec)

**Dependencies:** None (can start immediately)

---

### Session C: LLM Investigation (Weeks 6-8)

**Package:** `internal/agent/` (NEW - Session C owns entirely)

**Owns:**
- `internal/agent/investigator.go` (NEW - main investigation loop)
- `internal/agent/hop_navigator.go` (NEW - 1-hop, 2-hop, 3-hop logic)
- `internal/agent/evidence.go` (NEW - evidence accumulation)
- `internal/agent/synthesis.go` (NEW - LLM synthesis prompts)
- `internal/agent/llm_client.go` (NEW - OpenAI/Anthropic wrapper)
- `internal/agent/types.go` (NEW - investigation data structures)
- `internal/agent/investigator_test.go` (NEW - unit tests)
- `test/integration/test_agentic_search.sh` (NEW - e2e test)

**Modifies:**
- `cmd/crisk/check.go` (add Phase 2 escalation - ~60 lines)
- `internal/metrics/baseline.go` (add Tier 2 metric triggers - ~40 lines)

**Reads (no modification):**
- `internal/graph/` (graph queries)
- `internal/temporal/` (co-change data from Session A)
- `internal/incidents/` (incident search from Session B)
- `dev_docs/01-architecture/agentic_design.md` (investigation spec)

**Dependencies:**
- **CAN start immediately** (reads interfaces, not implementations)
- Session C creates mock data for testing while A & B work

---

## Shared Interface Definitions

### Interface 1: Temporal Data (Session A provides, Session C uses)

**File:** `internal/temporal/types.go`

```go
package temporal

// CoChangeResult represents files that change together
type CoChangeResult struct {
    FileA      string
    FileB      string
    Frequency  float64  // 0.0 to 1.0
    LastCommit time.Time
    Window     int      // days (90)
}

// GetCoChangedFiles returns files that frequently change with target file
func GetCoChangedFiles(filePath string, minFrequency float64) ([]CoChangeResult, error)

// GetOwnershipHistory returns developer ownership transitions
func GetOwnershipHistory(filePath string) (*OwnershipHistory, error)
```

**Session C uses these functions** to calculate temporal coupling metric.

---

### Interface 2: Incident Search (Session B provides, Session C uses)

**File:** `internal/incidents/types.go`

```go
package incidents

// IncidentMatch represents a similar past incident
type IncidentMatch struct {
    ID          int
    Title       string
    Description string
    Similarity  float64  // BM25 score
    CreatedAt   time.Time
    AffectedFiles []string
}

// SearchSimilarIncidents performs PostgreSQL FTS search
func SearchSimilarIncidents(query string, limit int) ([]IncidentMatch, error)

// LinkIncidentToCommit manually links incident to git commit
func LinkIncidentToCommit(incidentID int, commitSHA string) error
```

**Session C uses these functions** for incident similarity metric.

---

### Interface 3: LLM Investigation (Session C provides, Sessions A & B don't need)

**File:** `internal/agent/types.go`

```go
package agent

// InvestigationResult is the final output from Phase 2
type InvestigationResult struct {
    RiskLevel       string // LOW, MEDIUM, HIGH
    Confidence      float64
    EvidenceChain   []Evidence
    Recommendations []Recommendation
    TotalHops       int
    Duration        time.Duration
}

// Investigate performs agentic search starting from changed files
func Investigate(ctx context.Context, changedFiles []string) (*InvestigationResult, error)
```

---

## Critical Checkpoints

### Checkpoint A1: Git History Parsing Works (Session A)

**Trigger:** Session A implements `internal/temporal/git_history.go`

**Verification:**
```bash
# Test git log parsing
go test ./internal/temporal/... -v -run TestParseGitHistory

# Verify 90-day window
cd /tmp/omnara
go run scripts/test_temporal.go
# Expected: Shows commits from last 90 days
```

**YOU ASK:** "✅ Git history parsing complete. Parsed X commits from last 90 days. Developer nodes: Y. Ready for co-change calculation?"

---

### Checkpoint A2: CO_CHANGED Edges Created (Session A)

**Trigger:** Session A calculates and stores CO_CHANGED edges

**Verification:**
```bash
# Query Neo4j for co-change edges
docker exec coderisk-neo4j cypher-shell -u neo4j -p "..." \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"

# Expected: 500-2000 CO_CHANGED edges

# Check edge properties
docker exec coderisk-neo4j cypher-shell -u neo4j -p "..." \
  "MATCH (a)-[r:CO_CHANGED]->(b) RETURN a.path, b.path, r.frequency LIMIT 10"
```

**YOU ASK:** "✅ CO_CHANGED edges created. Total: X edges. Frequency range: 0.05-0.95. Sample queries work. Complete?"

---

### Checkpoint B1: PostgreSQL FTS Schema Created (Session B)

**Trigger:** Session B creates incidents table with tsvector

**Verification:**
```bash
# Run migration
psql -h localhost -p 5433 -U coderisk -d coderisk \
  -f migrations/002_incidents_schema.sql

# Verify schema
psql -h localhost -p 5433 -U coderisk -d coderisk \
  -c "\d incidents"

# Expected: search_vector tsvector column with GIN index
```

**YOU ASK:** "✅ Incidents schema created. Table: incidents. GIN index: incidents_search_idx. Ready for data import?"

---

### Checkpoint B2: BM25 Search Works (Session B)

**Trigger:** Session B implements full-text search

**Verification:**
```bash
# Insert test incident
psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
INSERT INTO incidents (title, description) VALUES
  ('Auth timeout', 'Authentication service timeout after 30s causing login failures');
EOF

# Test search
go run scripts/test_incident_search.go "auth timeout login"

# Expected: Returns incident with similarity score > 0.5
```

**YOU ASK:** "✅ BM25 search working. Test query 'auth timeout' returns incident with score 0.82. Ready for integration?"

---

### Checkpoint C1: LLM Client Works (Session C)

**Trigger:** Session C implements OpenAI/Anthropic wrapper

**Verification:**
```bash
# Test LLM call
export OPENAI_API_KEY=sk-...
go test ./internal/agent/... -v -run TestLLMClient

# Verify prompt/response
go run scripts/test_llm.go "Is this code risky?"
# Expected: Returns JSON decision {action: "CALCULATE_METRIC", reasoning: "..."}
```

**YOU ASK:** "✅ LLM client working. OpenAI: ✓, Anthropic: ✓. Token usage: ~500 tokens/call. Cost: $0.001. Ready for hop navigation?"

---

### Checkpoint C2: Hop Navigation Works (Session C)

**Trigger:** Session C implements 1-hop, 2-hop, 3-hop logic

**Verification:**
```bash
# Test hop navigation
go run scripts/test_hops.go /tmp/omnara/src/server.ts

# Expected output:
# Hop 1: Loaded 15 files (1-hop neighbors)
# Hop 2: LLM requested ownership metric
# Hop 3: LLM requested incident search
# FINALIZE: Evidence: coupling=HIGH, ownership=RECENT, incident=SIMILAR
```

**YOU ASK:** "✅ Hop navigation complete. Max hops: 3. Context size: <2K tokens. Evidence chain working. Ready for synthesis?"

---

### Checkpoint C3: End-to-End Investigation Works (Session C)

**Trigger:** Session C completes full agentic search

**Verification:**
```bash
# Full investigation test
cd /tmp/omnara
echo "// test change" >> src/server.ts
git add src/server.ts

./crisk check src/server.ts --explain

# Expected:
# 🔍 CodeRisk: Analyzing... (3.2s)
# ⚠️ MEDIUM risk detected
# Investigation trace:
#   Hop 1: Coupling: 18 files (HIGH)
#   Hop 2: Ownership: Changed 14 days ago
#   Hop 3: Incident: Similar to INC-892 (89% match)
# Recommendations: [3 actionable items]
```

**YOU ASK:** "✅ End-to-end investigation works! Risk levels accurate. Evidence chain clear. Recommendations actionable. Ready for integration?"

---

### Checkpoint FINAL: All Sessions Integrated

**Trigger:** All 3 sessions complete

**Verification:**
```bash
# Complete Phase 1 + Phase 2 flow
cd /tmp/test-repo
crisk init-local                    # Layer 1 (Week 1)
# Uses Session A for temporal analysis
# Uses Session B for incident import

echo "// risky change" >> core/auth.ts
git add core/auth.ts
crisk check core/auth.ts           # Phase 1 (200ms)
# If escalated → Phase 2 (3-5s)
# Uses Session A (co-change)
# Uses Session B (incident search)
# Uses Session C (LLM investigation)

# Expected: Complete risk report with evidence
```

**Action:** Mark Weeks 2-8 complete, ready for production!

---

## Coordination Protocol

### Start Order (All Parallel)

**All 3 sessions start simultaneously:**

1. **Session A starts** → Implements temporal analysis (independent)
2. **Session B starts** → Implements incident database (independent)
3. **Session C starts** → Implements LLM investigation (independent)

**No blocking dependencies!** Session C creates mock interfaces while A & B implement real ones.

### Mock Data Strategy (Session C Only)

While Sessions A & B work, Session C uses **mock data**:

```go
// internal/agent/mocks.go (temporary, Session C creates)
func MockCoChangeData() []temporal.CoChangeResult {
    return []temporal.CoChangeResult{
        {FileA: "a.ts", FileB: "b.ts", Frequency: 0.85},
    }
}

func MockIncidentData() []incidents.IncidentMatch {
    return []incidents.IncidentMatch{
        {ID: 123, Title: "Auth timeout", Similarity: 0.89},
    }
}
```

**When A & B complete:** Session C swaps mocks for real implementations (2-line change).

### Communication via Interfaces

- Sessions A & B **export interfaces** in `types.go`
- Session C **imports interfaces** and codes against them
- **No file conflicts** - each session owns its package

---

## Success Criteria

### Functional
- [ ] Session A: CO_CHANGED edges in Neo4j (500-2000 edges)
- [ ] Session A: Ownership history tracking works
- [ ] Session B: PostgreSQL FTS returns incidents (BM25 ranking)
- [ ] Session B: Manual incident linking working
- [ ] Session C: LLM client calls OpenAI/Anthropic successfully
- [ ] Session C: Hop navigation explores graph (1-3 hops)
- [ ] Session C: Evidence synthesis produces actionable recommendations
- [ ] **Integration:** `crisk check` escalates to Phase 2 and returns accurate risk

### Performance
- [ ] Temporal analysis: <10s for 90-day history (5K commits)
- [ ] Incident search: <50ms for BM25 query (PostgreSQL)
- [ ] LLM investigation: 3-5s total (including LLM calls)
- [ ] Phase 2 total: <8s (p95)

### Quality
- [ ] 70%+ unit test coverage for new packages
- [ ] Integration tests pass for each session
- [ ] No file conflicts during merge
- [ ] All builds succeed: `go build ./...`

---

## What Could Go Wrong

### Issue: Session C starts before A & B define interfaces
**Prevention:** Session C creates temporary interface stubs based on spec
**Recovery:** Replace stubs with real imports after Checkpoint A2/B2 (2-line change per interface)

### Issue: Git history parsing hangs on large repos
**Prevention:** Session A adds timeout and pagination (1000 commits/batch)
**Recovery:** Add `--max-commits` flag to limit history depth

### Issue: PostgreSQL FTS returns irrelevant results
**Prevention:** Session B tunes BM25 weights and tests with known incidents
**Recovery:** Add manual relevance scoring layer on top of FTS

### Issue: LLM calls exceed token limits
**Prevention:** Session C limits context to 2K tokens per hop
**Recovery:** Implement context compression (summarize evidence before LLM call)

### Issue: Phase 2 cost too high ($0.10+ per check)
**Prevention:** Session C caches LLM responses (Redis, 15-min TTL)
**Recovery:** Use cheaper models (gpt-4o-mini instead of gpt-4) or reduce hops to 2 max

---

## Timeline Estimate

**Session A (Temporal):** 1.5-2 weeks
- Week 2: Git history parsing, commit extraction
- Week 3: CO_CHANGED calculation, ownership tracking

**Session B (Incidents):** 1.5-2 weeks
- Week 4: PostgreSQL schema, FTS setup
- Week 5: BM25 search, manual linking UI

**Session C (LLM Investigation):** 2-3 weeks
- Week 6: LLM client, basic prompts
- Week 7: Hop navigation, evidence accumulation
- Week 8: Synthesis, integration with Phase 1

**Total (parallel):** 3-4 weeks (vs 7-8 weeks sequential)

**Speedup:** ~2x faster

---

## Session Prompts

See separate files:
- `SESSION_A_PROMPT.md` - Temporal Analysis (Git History, CO_CHANGED)
- `SESSION_B_PROMPT.md` - Incident Database (PostgreSQL FTS, BM25)
- `SESSION_C_PROMPT.md` - LLM Investigation (Agentic Search, Evidence Synthesis)

---

## After All Sessions Complete

**Update documentation:**
- [ ] Mark Weeks 2-8 complete in [status.md](status.md)
- [ ] Update [system_overview_layman.md](../01-architecture/system_overview_layman.md) with "100% working"
- [ ] Create performance report with actual metrics
- [ ] Document API key setup in README.md

**Celebrate!** 🎉
- MVP complete in 3-4 weeks (vs 7-8 weeks)
- Full agentic search working end-to-end
- <3% false positive rate achieved
- Ready for beta users

---

## Quick Commands for Verification

**Temporal analysis (Checkpoint A2):**
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p "..." \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"
```

**Incident search (Checkpoint B2):**
```bash
go run scripts/test_incident_search.go "auth timeout"
```

**LLM investigation (Checkpoint C3):**
```bash
./crisk check src/server.ts --explain
```

**Full integration:**
```bash
./test/integration/test_mvp_complete.sh
```

---

## References

- [system_overview_layman.md](../01-architecture/system_overview_layman.md) - Full system design
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Layer 2 & 3 specs
- [agentic_design.md](../01-architecture/agentic_design.md) - Investigation algorithm
- [ADR-003](../01-architecture/decisions/003-postgresql-fulltext-search.md) - PostgreSQL FTS decision
- [status.md](status.md) - Current implementation status

---

**Created:** October 5, 2025
**Status:** Ready to execute
**Next Review:** After each checkpoint
