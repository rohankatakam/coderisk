# CodeRisk End-to-End Testing & Gap Analysis

**Purpose:** Comprehensive testing session to validate CodeRisk implementation against specification documents
**Session Type:** Fresh Claude Code session (no prior context)
**Duration:** 2-3 hours
**Approach:** Dry investigation first → Document gaps → Execute tests → Report findings

---

## Your Mission

You are tasked with **thoroughly investigating** the CodeRisk implementation to identify gaps between what exists and what is documented in:

1. `dev_docs/01-architecture/system_overview_layman.md` - The system we claim to have built
2. `dev_docs/00-product/developer_experience.md` - The UX we promise users

**DO NOT SKIP THE DRY INVESTIGATION.** Start by analyzing code, reading files, and documenting gaps before running any tests.

---

## Phase 1: Dry Investigation (60-90 minutes)

### Step 1.1: Read the Specifications

Read these files completely to understand what the system should do:

```bash
# Read these files first (do not skip)
cat dev_docs/01-architecture/system_overview_layman.md
cat dev_docs/00-product/developer_experience.md
cat SESSIONS_ABC_INTEGRATION_COMPLETE.md
cat INTEGRATION_AUDIT_AND_FIXES.md
```

**Your Task:** Take notes on:
- What features are described?
- What performance targets are mentioned?
- What user workflows are documented?
- What outputs are shown in examples?

---

### Step 1.2: Investigate Code Structure

Systematically review the codebase to understand what's actually implemented:

```bash
# Map out the codebase structure
tree -L 3 -I 'node_modules|.git|vendor'

# Check what commands exist
./crisk --help
./crisk init-local --help
./crisk check --help

# Review key packages
ls -la internal/
ls -la cmd/crisk/

# Check integration status
cat SESSIONS_ABC_INTEGRATION_COMPLETE.md | grep -A 5 "Success Criteria"
```

**Your Task:** Document:
- What CLI commands are implemented?
- What internal packages exist?
- What's the current integration status (from docs)?

---

### Step 1.3: Analyze Session A (Temporal Analysis)

Investigate the temporal analysis implementation:

```bash
# Review Session A files
ls -la internal/temporal/
cat internal/temporal/types.go
head -50 internal/temporal/co_change.go
head -50 internal/temporal/developer.go

# Check test coverage
go test ./internal/temporal/... -v -cover

# Look for TODOs or unimplemented functions
grep -r "TODO\|FIXME\|not implemented" internal/temporal/
```

**Your Task:** Answer:
- ✅ Are `GetCoChangedFiles()` and `GetOwnershipHistory()` implemented?
- ✅ Do they accept git commits as input (as described in integration docs)?
- ❓ Are CO_CHANGED edges actually created in Neo4j during `init-local`?
- ❓ Can these functions query Neo4j, or do they only calculate from git?

**Expected per system_overview_layman.md:**
- CO_CHANGED edges with frequency property (Layer 2)
- Query: "Files that change together 85% of the time"
- Performance: <20ms for co-change lookup
- Example: `payment_processor.py` co-changes with `transactions.py` at 0.85 frequency

**Document gaps:**
```
Gap ID: A1
Component: Session A - CO_CHANGED edge creation
Documented behavior: CO_CHANGED edges created in Neo4j during init-local
Actual behavior: [INVESTIGATE - check internal/graph/builder.go]
Status: [MISSING/PARTIAL/COMPLETE]
```

---

### Step 1.4: Analyze Session B (Incidents Database)

Investigate incident database implementation:

```bash
# Review Session B files
ls -la internal/incidents/
cat internal/incidents/types.go
head -50 internal/incidents/database.go
head -50 internal/incidents/search.go

# Check PostgreSQL schema
cat scripts/init_postgres.sql | grep -A 30 "CREATE TABLE incidents"

# Check CLI commands
./crisk incident --help

# Look for test coverage
go test ./internal/incidents/... -v -cover
```

**Your Task:** Answer:
- ✅ Is PostgreSQL schema created with BM25 full-text search (tsvector + GIN index)?
- ✅ Does `GetIncidentStats()` work?
- ✅ Does `SearchIncidents()` use PostgreSQL FTS?
- ❓ Are CAUSED_BY edges created in Neo4j when linking incidents?

**Expected per system_overview_layman.md:**
- Incident nodes in Neo4j (Layer 3)
- CAUSED_BY edges: (Incident)-[:CAUSED_BY]->(File)
- PostgreSQL FTS with BM25 ranking (<50ms)
- Example: Search "Stripe timeout" → finds INC-892 with 0.89 similarity

**Document gaps:**
```
Gap ID: B1
Component: Session B - Incident linking to Neo4j
Documented behavior: CAUSED_BY edges created in Neo4j
Actual behavior: [INVESTIGATE - check internal/incidents/linker.go]
Status: [MISSING/PARTIAL/COMPLETE]
```

---

### Step 1.5: Analyze Session C (LLM Investigation)

Investigate agent implementation:

```bash
# Review Session C files
ls -la internal/agent/
cat internal/agent/types.go
head -50 internal/agent/investigator.go
head -50 internal/agent/evidence.go
head -50 internal/agent/hop_navigator.go

# Check if adapters exist (real vs mock clients)
cat internal/agent/adapters.go

# Check Phase 2 integration in check command
grep -A 30 "Phase 2" cmd/crisk/check.go

# Look for OpenAI client usage
grep -r "openai\|anthropic" internal/agent/
```

**Your Task:** Answer:
- ✅ Is `NewLLMClient()` OpenAI-only (Anthropic removed)?
- ✅ Do `RealTemporalClient` and `RealIncidentsClient` adapters exist?
- ❓ Is Phase 2 escalation integrated into `cmd/crisk/check.go`?
- ❓ Does `Investigate()` actually call OpenAI API?
- ❓ Does evidence collection pull from Sessions A & B?

**Expected per system_overview_layman.md:**
- Phase 2 triggers when Phase 1 detects risk (coupling >0.5 or incidents >0)
- LLM performs 1-3 hops (max 3-hop limit enforced)
- Evidence from: co-change (Session A), incidents (Session B), graph structure
- Synthesis: Final risk assessment with recommendations
- Example output shown in system_overview_layman.md lines 315-357

**Document gaps:**
```
Gap ID: C1
Component: Session C - Phase 2 integration in check.go
Documented behavior: Phase 2 escalates when baseline metrics exceed threshold
Actual behavior: [INVESTIGATE - check cmd/crisk/check.go around line 153]
Status: [MISSING/PARTIAL/COMPLETE]
```

---

### Step 1.6: Analyze CLI & User Experience

Check what CLI commands actually work:

```bash
# Test help commands
./crisk --help
./crisk init-local --help
./crisk check --help
./crisk hook --help  # Should exist per developer_experience.md

# Check verbosity levels (per developer_experience.md)
./crisk check --quiet 2>&1 | head -5
./crisk check --verbose 2>&1 | head -10
./crisk check --explain 2>&1 | head -20
./crisk check --ai-mode 2>&1 | head -10  # Should output JSON

# Check incident commands (per Session B)
./crisk incident --help
./crisk incident create --help
./crisk incident link --help
```

**Expected per developer_experience.md:**
- Pre-commit hook installation: `crisk hook install`
- Verbosity levels: `--quiet`, standard, `--explain`, `--ai-mode`
- AI Mode: JSON output with `ai_assistant_actions[]`
- Incident CLI: `create`, `link`, `search`, `stats`, `list`, `unlink`

**Document gaps:**
```
Gap ID: UX1
Component: CLI verbosity levels
Documented behavior: 4 levels (quiet, standard, explain, ai-mode)
Actual behavior: [INVESTIGATE - test each flag]
Status: [MISSING/PARTIAL/COMPLETE]
```

---

### Step 1.7: Compile Gap Report (Before Testing)

Create a structured gap report:

```markdown
# CodeRisk Implementation Gap Analysis
Date: [TODAY]
Analyzer: Claude Code (Fresh Session)

## Summary
- Total Gaps Found: [COUNT]
- Critical (blocks core workflow): [COUNT]
- High (missing advertised feature): [COUNT]
- Medium (partial implementation): [COUNT]
- Low (nice-to-have): [COUNT]

## Critical Gaps

### Gap C1: Phase 2 Escalation Not Integrated
**Component:** cmd/crisk/check.go
**Documented:** system_overview_layman.md lines 41-44, 180-274
**Expected:** When Phase 1 detects coupling >0.5 or incidents >0, escalate to LLM investigation
**Actual:** [YOUR FINDING from check.go inspection]
**Impact:** Phase 2 never runs, core value prop broken
**Priority:** P0 (CRITICAL)

### Gap A1: CO_CHANGED Edges Not Created
**Component:** internal/graph/builder.go
**Documented:** system_overview_layman.md lines 91-94, Layer 2 specification
**Expected:** CO_CHANGED edges created during init-local with frequency property
**Actual:** [YOUR FINDING from builder.go inspection]
**Impact:** Temporal coupling queries fail, Phase 2 evidence collection incomplete
**Priority:** P0 (CRITICAL)

[Continue for all gaps...]

## High Priority Gaps

[List gaps that break documented features]

## Medium Priority Gaps

[List partial implementations]

## Low Priority Gaps

[List nice-to-haves]

## Testing Recommendations

Based on gaps found, prioritize testing in this order:
1. [Test X] - Validates Gap C1
2. [Test Y] - Validates Gap A1
...
```

---

## Phase 2: Execute Tests (60-90 minutes)

### Test Suite 1: Basic Functionality

**Test 1.1: Build & Install**
```bash
# Build the project
go build -o crisk ./cmd/crisk

# Check version
./crisk --version

# Check help
./crisk --help
```

**Expected:** Builds successfully, shows version and help

**Document result:**
```
Test 1.1: Build & Install
Status: [PASS/FAIL]
Notes: [Any errors or warnings]
```

---

**Test 1.2: Init Local (Layer 1 - Structure)**
```bash
# Clone a test repository
cd /tmp
git clone https://github.com/omnara-ai/omnara.git test-repo
cd test-repo

# Run init-local
time /path/to/crisk init-local

# Verify results
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (n) RETURN labels(n)[0] as type, count(n) as count ORDER BY count DESC"

# Expected output per system_overview_layman.md:
# File nodes: ~421
# Function nodes: ~2,563
# Class nodes: ~454
# Import nodes: ~2,089
```

**Expected per system_overview_layman.md (lines 86-90):**
- Layer 1 (Structure): File, Function, Class, Import nodes
- CONTAINS edges: File → Function, File → Class
- IMPORTS edges: File → Import
- Total entities: 5,527 for omnara repository

**Document result:**
```
Test 1.2: Init Local (Layer 1)
Status: [PASS/FAIL]
Node counts: File=[X], Function=[Y], Class=[Z], Import=[W]
Expected: File=421, Function=2563, Class=454, Import=2089
Gap: [If counts don't match, note the difference]
```

---

**Test 1.3: Layer 2 (Temporal) - CO_CHANGED Edges**
```bash
# After init-local, check for CO_CHANGED edges
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"

# Expected per system_overview_layman.md line 93:
# CO_CHANGED edges should exist with frequency property
# Example query should work:
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
   WHERE r.frequency >= 0.7
   RETURN f.unique_id, other.unique_id, r.frequency
   ORDER BY r.frequency DESC
   LIMIT 5"
```

**Expected:** CO_CHANGED edges created, frequency property exists, query returns results

**Document result:**
```
Test 1.3: Layer 2 - CO_CHANGED Edges
Status: [PASS/FAIL]
CO_CHANGED edge count: [X]
Expected: >0 (hundreds for omnara repo)
Sample query result: [Show top 3 pairs]
Gap: [If zero edges, this is Gap A1 - CRITICAL]
```

---

### Test Suite 2: Incident Database (Session B)

**Test 2.1: PostgreSQL Schema**
```bash
# Check if incidents table exists
docker exec coderisk-postgres psql -U coderisk -d coderisk -c "\dt"

# Verify GIN index
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
  "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'incidents'"

# Check tsvector column
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
  "SELECT column_name, data_type FROM information_schema.columns
   WHERE table_name = 'incidents' AND column_name = 'search_vector'"
```

**Expected per system_overview_layman.md (line 437):**
- incidents table with search_vector (tsvector) column
- GIN index: idx_incidents_search
- Performance target: <50ms for incident similarity search

**Document result:**
```
Test 2.1: PostgreSQL Schema
Status: [PASS/FAIL]
Tables found: [incidents, incident_files, ...]
GIN index: [idx_incidents_search] [EXISTS/MISSING]
search_vector column: [tsvector] [EXISTS/MISSING]
```

---

**Test 2.2: Create and Link Incident**
```bash
# Create an incident
INCIDENT_ID=$(./crisk incident create "Payment timeout" \
  "Stripe API timeout causing checkout failures" \
  --severity critical \
  --root-cause "Missing connection pooling in payment_processor.py" \
  | grep -oP 'ID: \K[a-f0-9-]+')

echo "Created incident: $INCIDENT_ID"

# Link to a file (assuming src/payments/payment_processor.py exists)
# If not, use any file from the test repo
./crisk incident link "$INCIDENT_ID" "src/server.ts" --line 45 --function "handleRequest"

# Verify in PostgreSQL
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
  "SELECT * FROM incident_files WHERE incident_id = '$INCIDENT_ID'"

# Verify CAUSED_BY edge in Neo4j
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f:File)
   RETURN i.title, f.unique_id, r.confidence"
```

**Expected per system_overview_layman.md (lines 97-99):**
- Incident created in PostgreSQL
- Link created in incident_files table
- CAUSED_BY edge created in Neo4j: (Incident)-[:CAUSED_BY]->(File)

**Document result:**
```
Test 2.2: Create and Link Incident
Status: [PASS/FAIL]
Incident created: [YES/NO] ID=[...]
PostgreSQL link: [EXISTS/MISSING]
Neo4j CAUSED_BY edge: [EXISTS/MISSING]
Gap: [If edge missing, this is Gap B1]
```

---

**Test 2.3: Incident Search (BM25)**
```bash
# Create a few more incidents for search testing
./crisk incident create "Database deadlock" "Deadlock in user table during peak traffic" \
  --severity high

./crisk incident create "API timeout" "External API call timeout" \
  --severity medium

# Test BM25 search
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
  "SELECT title, ts_rank_cd(search_vector, query) AS rank
   FROM incidents, to_tsquery('english', 'timeout') query
   WHERE search_vector @@ query
   ORDER BY rank DESC"
```

**Expected:** BM25 search returns ranked results, <50ms query time

**Document result:**
```
Test 2.3: Incident BM25 Search
Status: [PASS/FAIL]
Query: "timeout"
Results count: [X]
Top result: [title] (rank=[Y])
Performance: [Z]ms
```

---

### Test Suite 3: LLM Investigation (Session C)

**Test 3.1: Evidence Collection (Without LLM)**
```bash
# Test evidence collector directly
# Create a Go test file to call the evidence collector

cat > test_evidence.go <<'EOF'
package main

import (
    "context"
    "fmt"
    "github.com/coderisk/coderisk-go/internal/agent"
    "github.com/coderisk/coderisk-go/internal/temporal"
    "github.com/coderisk/coderisk-go/internal/incidents"
)

func main() {
    ctx := context.Background()

    // Create real temporal client
    temporalClient, err := agent.NewRealTemporalClient("/tmp/test-repo")
    if err != nil {
        fmt.Printf("Error creating temporal client: %v\n", err)
        return
    }

    // Create mock incidents client (or real if DB is set up)
    // incidentsClient := agent.NewRealIncidentsClient(...)

    // For now, test that adapters exist and compile
    fmt.Printf("Temporal client created: %+v\n", temporalClient)
}
EOF

go run test_evidence.go
rm test_evidence.go
```

**Expected:**
- RealTemporalClient adapter exists (from SESSIONS_ABC_INTEGRATION_COMPLETE.md)
- Parses git history and provides GetCoChangedFiles() / GetOwnershipHistory()

**Document result:**
```
Test 3.1: Evidence Collection Adapters
Status: [PASS/FAIL]
RealTemporalClient: [EXISTS/MISSING]
RealIncidentsClient: [EXISTS/MISSING]
Compiles: [YES/NO]
```

---

**Test 3.2: Phase 2 Integration**
```bash
# This is the CRITICAL test per system_overview_layman.md example (lines 143-357)

# Make a change to a file
cd /tmp/test-repo
echo "// Test change" >> src/server.ts
git add src/server.ts

# Run check (should trigger Phase 2 if risk detected)
# NOTE: Requires OPENAI_API_KEY
export OPENAI_API_KEY="sk-..."  # Use test key
./crisk check src/server.ts

# Expected output per developer_experience.md (lines 84-104):
# If MEDIUM/HIGH risk:
# - Should show Phase 2 investigation
# - Should show hop-by-hop analysis
# - Should provide recommendations
```

**Expected per system_overview_layman.md:**
- Phase 1 runs baseline metrics (coupling, co-change, incidents)
- If risk thresholds exceeded, escalates to Phase 2
- Phase 2 performs LLM investigation (1-3 hops)
- Output shows investigation trace and recommendations

**Document result:**
```
Test 3.2: Phase 2 Integration
Status: [PASS/FAIL/SKIPPED - no API key]
Phase 1 ran: [YES/NO]
Phase 2 escalated: [YES/NO]
LLM called: [YES/NO]
Output format matches spec: [YES/NO]
Gap: [If Phase 2 doesn't run, this is Gap C1 - CRITICAL]
```

---

### Test Suite 4: Developer Experience

**Test 4.1: Pre-Commit Hook**
```bash
# Install hook per developer_experience.md (lines 56-64)
cd /tmp/test-repo
/path/to/crisk hook install

# Verify hook created
cat .git/hooks/pre-commit

# Test hook (make a change and commit)
echo "// Another change" >> src/server.ts
git add src/server.ts
git commit -m "Test pre-commit hook"

# Expected: Hook runs, shows risk assessment
```

**Expected per developer_experience.md:**
- `crisk hook install` creates `.git/hooks/pre-commit`
- Hook runs automatically on `git commit`
- Shows one-line summary in quiet mode
- Blocks commit on HIGH risk (unless `--no-verify`)

**Document result:**
```
Test 4.1: Pre-Commit Hook
Status: [PASS/FAIL]
Hook installed: [YES/NO]
Hook runs on commit: [YES/NO]
Output format: [One-line summary/Verbose/None]
```

---

**Test 4.2: Verbosity Levels**
```bash
# Test all 4 verbosity levels per developer_experience.md

# Level 1: Quiet
./crisk check --quiet src/server.ts

# Level 2: Standard (default)
./crisk check src/server.ts

# Level 3: Explain
./crisk check --explain src/server.ts

# Level 4: AI Mode (JSON)
./crisk check --ai-mode src/server.ts | jq '.'
```

**Expected per developer_experience.md (lines 157-654):**
- `--quiet`: One-line summary
- Standard: Issues + recommendations
- `--explain`: Full investigation trace
- `--ai-mode`: JSON with `ai_assistant_actions[]`, `risk` object, etc.

**Document result:**
```
Test 4.2: Verbosity Levels
Level 1 (--quiet): [PASS/FAIL] Output=[...]
Level 2 (standard): [PASS/FAIL] Output=[...]
Level 3 (--explain): [PASS/FAIL] Output=[...]
Level 4 (--ai-mode): [PASS/FAIL] Valid JSON=[YES/NO]
```

---

## Phase 3: Final Report (30 minutes)

### Create Comprehensive Test Report

```markdown
# CodeRisk End-to-End Test Report
Date: [TODAY]
Tester: Claude Code (Fresh Session)
Repository: coderisk-go
Test Repo: omnara-ai/omnara (421 files)

## Executive Summary
- Tests Executed: [X]/[Y]
- Tests Passed: [X]
- Tests Failed: [Y]
- Critical Gaps: [X]
- System Readiness: [%]

## Critical Findings

### Finding 1: [Gap ID]
**Severity:** CRITICAL
**Component:** [...]
**Test:** [Test X.Y]
**Expected:** [From spec]
**Actual:** [What you found]
**Impact:** [How this breaks the system]
**Recommendation:** [How to fix]

[Continue for all critical findings]

## Gap-to-Test Mapping

| Gap ID | Test ID | Status | Priority |
|--------|---------|--------|----------|
| A1     | 1.3     | FAIL   | P0       |
| C1     | 3.2     | FAIL   | P0       |
| B1     | 2.2     | PASS   | P1       |
| ...    | ...     | ...    | ...      |

## Test Results Detail

### Test Suite 1: Basic Functionality
[Detailed results for each test]

### Test Suite 2: Incident Database
[Detailed results]

### Test Suite 3: LLM Investigation
[Detailed results]

### Test Suite 4: Developer Experience
[Detailed results]

## Implementation Completeness

### Layer 1 (Code Structure) ✅
- Status: COMPLETE
- Confidence: HIGH
- Evidence: [Test 1.2 passed, 5,527 nodes created]

### Layer 2 (Temporal Analysis) ❓
- Status: PARTIAL
- Confidence: MEDIUM
- Evidence: [Functions implemented but edges not created]
- Gap: Gap A1 (CO_CHANGED edges missing from Neo4j)

### Layer 3 (Incidents) ✅
- Status: COMPLETE
- Confidence: HIGH
- Evidence: [Tests 2.1-2.3 passed]

### Phase 2 (LLM Investigation) ❌
- Status: NOT INTEGRATED
- Confidence: HIGH
- Evidence: [Test 3.2 failed, Phase 2 never triggers]
- Gap: Gap C1 (check.go integration missing)

## Recommendations

### Immediate Actions (P0 - Blocks Core Value Prop)
1. **Fix Gap C1:** Integrate Phase 2 into cmd/crisk/check.go
   - Add escalation logic when baseline metrics exceed thresholds
   - Call investigator.Investigate() with OpenAI client
   - Format and display results per developer_experience.md
   - Estimated time: 2-3 hours

2. **Fix Gap A1:** Add CO_CHANGED edge creation to graph builder
   - Modify internal/graph/builder.go
   - Add AddLayer2CoChangedEdges() method
   - Call after commit processing in init-local
   - Estimated time: 1-2 hours

### High Priority (P1 - Missing Advertised Features)
[List P1 items]

### Medium Priority (P2 - Polish)
[List P2 items]

## Next Steps

1. Share this report with the team
2. Prioritize critical gaps (P0)
3. Create GitHub issues for each gap
4. Re-test after fixes applied
5. Update documentation if specs don't match reality

## Appendix

### Environment
- Go version: [X]
- Docker version: [X]
- Neo4j version: [X]
- PostgreSQL version: [X]

### Test Repository
- URL: https://github.com/omnara-ai/omnara
- Files: 421
- Languages: TypeScript (286), Python (129), JavaScript (6)

### Logs & Artifacts
[Attach relevant logs, screenshots, error messages]
```

---

## Success Criteria for This Session

✅ **Dry Investigation Complete:** All code paths inspected, gaps documented before testing
✅ **All Tests Executed:** Minimum 15 tests across 4 suites
✅ **Gap Report Created:** Every discrepancy between spec and reality documented
✅ **Test Report Created:** Comprehensive report with findings and recommendations
✅ **Priority Assigned:** Each gap marked P0/P1/P2 based on impact

---

## Tips for Effective Testing

1. **Don't skip dry investigation** - Understanding gaps before testing saves time
2. **Document everything** - Every test result, even if passing
3. **Be thorough** - Check not just "does it work" but "does it work as documented"
4. **Cross-reference specs** - Every test should cite specific lines from spec docs
5. **Measure performance** - Time every operation, compare to targets
6. **Think like a user** - Does the UX match developer_experience.md?
7. **Flag assumptions** - If you had to guess or infer, note it

---

## What NOT to Do

❌ Don't fix issues during testing (document them instead)
❌ Don't skip tests because "it probably works"
❌ Don't assume documentation is correct (verify everything)
❌ Don't test in isolation (use the full workflow)
❌ Don't ignore performance (speed targets are in the specs)

---

## Deliverables

1. **Gap Analysis Report** (Phase 1 output)
2. **Test Results** (Phase 2 output)
3. **Final Test Report** (Phase 3 output)
4. **Prioritized Fix List** (Ranked by impact)

---

**Ready? Start with Phase 1, Step 1.1. Read the specs completely before writing any code.**
