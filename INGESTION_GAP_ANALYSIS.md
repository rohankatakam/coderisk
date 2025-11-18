# CodeRisk Ingestion Pipeline Gap Analysis

**Generated:** 2025-11-18
**Repository:** mcp-use (repo_id=11)
**Analysis Scope:** Full pipeline from crisk-stage → crisk-index-coupling

---

## Executive Summary

The ingestion pipeline for mcp-use (repo_id=11) has **critical gaps** that block risk calculation:

### 🔴 CRITICAL Issues (Blocking)
1. **Missing crisk-ingest execution** - No Commit/Developer/File nodes in Neo4j (0 commits vs 517 in Postgres)
2. **MODIFIED_BLOCK edges cannot be created** - Atomizer created 1167 CodeBlocks but 0 edges (no Commit nodes to link to)
3. **Ownership indexing fails** - Requires MODIFIED_BLOCK edges to calculate ownership signals
4. **Schema mismatch in code_block_incidents** - Code writes to old columns, schema expects new columns

### ⚠️ HIGH Priority Issues
5. **Atomizer data quality problems** - Empty block names (14), zero line numbers (40+), non-code files extracted
6. **Missing risk scores** - Cannot calculate without ownership data (blocked by #3)

### ✅ What's Working
- ✅ crisk-stage completed successfully (517 commits, 360 files, file identity map built)
- ✅ Atomizer created 864 code blocks in Postgres (repo_id=11)
- ✅ Incident linking created 112 links (10 unique issues)
- ✅ Coupling analysis created 4317 co-change edges

---

## Issue 1: Missing crisk-ingest Execution

### Problem
**crisk-ingest was never run for repo_id=11**, causing a complete gap in the Neo4j knowledge graph.

### Evidence

**PostgreSQL (Source of Truth):**
```sql
SELECT COUNT(*) FROM github_commits WHERE repo_id = 11;
-- Result: 517 commits
```

**Neo4j (Knowledge Graph):**
```cypher
MATCH (c:Commit) WHERE c.repo_id = 11 RETURN count(c);
-- Result: 0 commits ❌
```

**Impact:**
- No Developer nodes (cannot calculate ownership)
- No Commit nodes (cannot create MODIFIED_BLOCK edges)
- No File nodes (cannot create CONTAINS edges)
- No Issue/PR nodes (incident linking incomplete)
- No CLOSED_BY edges (temporal analysis incomplete)

### Root Cause
The pipeline was run in this order:
1. ✅ crisk-stage (GitHub data fetched to Postgres)
2. ❌ **crisk-ingest SKIPPED** (should populate Neo4j from Postgres)
3. ✅ crisk-atomize (created CodeBlocks but couldn't link them)
4. ✅ crisk-index-incident (created incident links in Postgres only)
5. ❌ crisk-index-ownership (failed - no MODIFIED_BLOCK edges)
6. ❌ crisk-index-coupling (incomplete - no ownership data)

### Fix Required
**Re-run the pipeline in correct order:**
```bash
# 1. crisk-stage already done ✅
# 2. Run crisk-ingest to populate Neo4j
./bin/crisk-ingest --repo-id 11

# 3. Clear atomizer data and re-run
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
  DELETE FROM code_block_changes WHERE repo_id = 11;
  DELETE FROM code_blocks WHERE repo_id = 11;
"
# Clear Neo4j CodeBlocks
curl -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (cb:CodeBlock {repo_id: 11}) DETACH DELETE cb"}]}'

# Re-run atomizer
./bin/crisk-atomize --repo-id 11 --repo-path ~/.coderisk/repos/<hash>

# 4. Re-run all indexing services
./bin/crisk-index-incident --repo-id 11
./bin/crisk-index-ownership --repo-id 11
./bin/crisk-index-coupling --repo-id 11
```

---

## Issue 2: Zero MODIFIED_BLOCK Edges

### Problem
**Atomizer created 1167 CodeBlock nodes in Neo4j but 0 MODIFIED_BLOCK edges.**

### Evidence
```cypher
MATCH (cb:CodeBlock) WHERE cb.repo_id = 11 RETURN count(cb);
-- Result: 1167 CodeBlocks ✅

MATCH ()-[r:MODIFIED_BLOCK]->() WHERE r.repo_id = 11 RETURN count(r);
-- Result: 0 edges ❌
```

### Root Cause Analysis

**Code Flow (event_processor.go:113-116, 178-180):**
```go
// handleCreateBlock creates CREATED_BLOCK edge
if err := p.graphWriter.CreateCreatedBlockEdge(ctx, commit.SHA, blockID, event, repoID, timestamp); err != nil {
    log.Printf("WARNING: Failed to create CREATED_BLOCK edge: %v", err)
}

// handleModifyBlock creates MODIFIED_BLOCK edge
if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, blockID, repoID, timestamp); err != nil {
    log.Printf("WARNING: Failed to create MODIFIED_BLOCK edge: %v", err)
}
```

**Graph Writer Query (graph_writer.go:110-114):**
```cypher
MATCH (c:Commit {sha: $commit_sha, repo_id: $repo_id})
MATCH (b:CodeBlock {db_id: $block_id, repo_id: $repo_id})
MERGE (c)-[r:MODIFIED_BLOCK]->(b)
```

**The Query Fails Silently Because:**
- Line 110: `MATCH (c:Commit ...)` finds **0 Commit nodes** (crisk-ingest not run)
- Query returns no results (no error thrown)
- Edge creation silently skips
- Warning logged but processing continues

### Impact Chain
```
No Commit nodes
  → MODIFIED_BLOCK edges cannot be created
    → Ownership calculator cannot traverse commit history
      → staleness_days remains NULL
        → Risk score cannot be calculated (ownership is 30% of formula)
```

### Fix Required
1. Run crisk-ingest to create Commit nodes
2. Clear and re-run atomizer to create edges
3. Validate edge counts after atomizer completes

---

## Issue 3: Ownership Indexing Failure

### Problem
**crisk-index-ownership reported "0 blocks updated"** - all ownership fields are NULL.

### Evidence
```sql
SELECT
  COUNT(*) as total_blocks,
  COUNT(original_author_email) as with_author,
  COUNT(staleness_days) as with_staleness
FROM code_blocks WHERE repo_id = 11;

-- Result:
-- total_blocks: 864
-- with_author: 0 ❌
-- with_staleness: 0 ❌
```

### Root Cause
The ownership calculator (crisk-index-ownership) uses this Neo4j query pattern:

```cypher
// Find first author (creator)
MATCH (c:Commit)-[:MODIFIED_BLOCK]->(cb:CodeBlock {id: $block_id})
RETURN c.author_email ORDER BY c.author_date ASC LIMIT 1

// Find last modifier
MATCH (c:Commit)-[:MODIFIED_BLOCK]->(cb:CodeBlock {id: $block_id})
RETURN c.author_email, c.author_date ORDER BY c.author_date DESC LIMIT 1
```

**Since MODIFIED_BLOCK edges = 0, these queries return empty results.**

### Impact
- `original_author_email`: NULL (can't identify creator)
- `last_modifier_email`: NULL (can't identify last toucher)
- `staleness_days`: NULL (can't calculate days since modification)
- `familiarity_map`: NULL (can't build developer familiarity)
- **Risk score CANNOT be calculated** (ownership component is 30% of formula)

### Fix Required
1. Fix Issue #1 (run crisk-ingest)
2. Fix Issue #2 (re-run atomizer to create edges)
3. Re-run crisk-index-ownership
4. Validate ownership fields are populated

---

## Issue 4: code_block_incidents Schema Mismatch

### Problem
**The code writes to OLD column names, but the schema reference defines NEW column names.**

### Evidence

**Database Schema (Actual):**
```sql
\d code_block_incidents

-- OLD columns (populated by code):
code_block_id   | bigint        -- 112/112 rows
fix_commit_sha  | varchar(40)   -- 112/112 rows
fixed_at        | timestamp     -- 112/112 rows

-- NEW columns (per schema reference, empty):
block_id        | bigint        -- 0/112 rows ❌
commit_sha      | text          -- 0/112 rows ❌
incident_date   | timestamp     -- 0/112 rows ❌
resolution_date | timestamp     -- 0/112 rows ❌
incident_type   | text          -- 0/112 rows ❌
```

**Application Code (temporal.go:126-131):**
```go
INSERT INTO code_block_incidents (
    repo_id, code_block_id, issue_id,    // ❌ OLD: code_block_id
    confidence, evidence_source, evidence_text,
    fix_commit_sha, fixed_at,             // ❌ OLD: fix_commit_sha, fixed_at
    created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
```

**Schema Reference (DATA_SCHEMA_REFERENCE.md:336-342):**
```markdown
| Column | Type | Description | Populated By |
| block_id | BIGINT | FK to code_blocks | crisk-index-incident |
| commit_sha | TEXT | Commit that closed the issue | crisk-index-incident |
| incident_date | TIMESTAMP | When issue was created | crisk-index-incident |
| resolution_date | TIMESTAMP | When issue was closed | crisk-index-incident |
| incident_type | TEXT | Issue labels (e.g., "bug", "security") | crisk-index-incident |
```

### Impact
1. **Data in wrong columns** - All 112 incident links use deprecated columns
2. **Missing required fields** - incident_date, resolution_date, incident_type never populated
3. **Schema reference queries broken** - Any code using new column names gets NULL
4. **Dual foreign keys** - Table has TWO FK constraints to code_blocks (old + new)

### Fix Required

**Option A: Migrate Code to New Schema (RECOMMENDED)**

Update `internal/risk/temporal.go:126-131`:
```go
INSERT INTO code_block_incidents (
    repo_id, block_id, issue_id,              // ✅ NEW: block_id
    confidence, evidence_source, evidence_text,
    commit_sha, incident_date, resolution_date, incident_type,  // ✅ NEW columns
    created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
```

Add data population:
```go
// Query to get issue dates and labels
SELECT created_at, closed_at, labels
FROM github_issues
WHERE id = $issue_id
```

Then migrate existing data:
```sql
UPDATE code_block_incidents SET
  block_id = code_block_id,
  commit_sha = fix_commit_sha,
  resolution_date = fixed_at,
  incident_date = (SELECT created_at FROM github_issues WHERE id = issue_id),
  incident_type = (SELECT array_to_string(labels, ',') FROM github_issues WHERE id = issue_id)
WHERE repo_id = 11;
```

**Option B: Update Schema Reference (NOT RECOMMENDED)**
- Marks new columns as deprecated
- Keeps current implementation
- Creates technical debt

---

## Issue 5: Atomizer Data Quality Issues

### Problem
**Atomizer extracted blocks from non-code files and created invalid entries.**

### Evidence

**Empty Block Names (14 blocks, 1.8%):**
```sql
SELECT canonical_file_path, block_type FROM code_blocks
WHERE repo_id = 11 AND block_name = '';

-- Examples:
-- libraries/typescript/packages/mcp-use/package.json (empty, empty)
-- libraries/typescript/packages/mcp-use/examples/client/observability.ts (empty, empty)
```

**Zero Line Numbers (40+ blocks, 10%):**
```sql
SELECT canonical_file_path, block_name FROM code_blocks
WHERE repo_id = 11 AND start_line = 0 AND end_line = 0;

-- Examples:
-- libraries/python/.gitignore | BaseConnector
-- libraries/python/pyproject.toml | test_async_placeholder
-- docs/images/hero-dark.png | get_server
```

**Non-Code Files with Code Blocks:**
- `.md` files (documentation)
- `.yaml`, `.toml`, `.json` (config files)
- `.png`, `.jpg` (binary images)
- `.gitignore`, `.env` (dotfiles)

### Root Cause
The atomizer's LLM extractor (llm_extractor.go:42) processes ALL files in the diff without filtering:

```go
// 1. Parse diff to extract file paths and line numbers
parsedFiles := ParseDiff(commit.DiffContent)

// 2. Send directly to LLM WITHOUT filtering ❌
eventLog, err := e.extractViaLLM(ctx, commit, parsedFiles)
```

**No file type validation exists before LLM call.**

### Impact
1. **Garbage data** - Non-code "blocks" pollute the database
2. **Invalid line numbers** - Cannot locate blocks in source files
3. **Wasted LLM tokens** - Processing docs/config files
4. **Search pollution** - False positives in block searches

### Fix Required

**Add file filtering before LLM extraction:**

```go
// internal/atomizer/llm_extractor.go

func (e *Extractor) ExtractCodeBlocks(ctx context.Context, commit CommitData) (*CommitChangeEventLog, error) {
    // 1. Parse diff
    parsedFiles := ParseDiff(commit.DiffContent)

    // 2. Filter to code files only ✅ NEW
    codeFiles := make(map[string]*DiffFileChange)
    for filePath, change := range parsedFiles {
        if IsCodeFile(filePath) {
            codeFiles[filePath] = change
        }
    }

    // 3. Skip if no code files
    if len(codeFiles) == 0 {
        return &CommitChangeEventLog{
            CommitSHA:        commit.SHA,
            AuthorEmail:      commit.AuthorEmail,
            Timestamp:        commit.Timestamp,
            LLMIntentSummary: "No code file changes detected",
            ChangeEvents:     []ChangeEvent{},
        }, nil
    }

    // 4. Extract from code files only
    eventLog, err := e.extractViaLLM(ctx, commit, codeFiles)
    // ...
}

func IsCodeFile(filename string) bool {
    // Skip binary files
    if IsBinaryFile(filename) {
        return false
    }

    // Skip documentation
    if strings.HasSuffix(strings.ToLower(filename), ".md") ||
       strings.HasSuffix(strings.ToLower(filename), ".mdx") {
        return false
    }

    // Skip config files
    configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".lock"}
    for _, ext := range configExts {
        if strings.HasSuffix(strings.ToLower(filename), ext) {
            return false
        }
    }

    // Skip dotfiles
    base := filepath.Base(filename)
    if strings.HasPrefix(base, ".") {
        return false
    }

    // Allow known code extensions
    codeExts := []string{".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".c", ".cpp"}
    for _, ext := range codeExts {
        if strings.HasSuffix(strings.ToLower(filename), ext) {
            return true
        }
    }

    return false
}
```

**Also filter empty block names:**
```go
// internal/atomizer/llm_extractor.go:315 (filterValidEvents)

func filterValidEvents(events []ChangeEvent) []ChangeEvent {
    filtered := []ChangeEvent{}
    for _, event := range events {
        // Skip events with empty block names ✅ NEW
        if event.BlockType != "" && event.TargetBlockName == "" {
            continue
        }

        // Existing validation...
        filtered = append(filtered, event)
    }
    return filtered
}
```

---

## Issue 6: Missing Risk Scores

### Problem
**All 864 code blocks have NULL risk_score.**

### Evidence
```sql
SELECT COUNT(*) FILTER (WHERE risk_score IS NOT NULL) FROM code_blocks WHERE repo_id = 11;
-- Result: 0 ❌
```

### Root Cause
Risk score calculation (by crisk-index-coupling) requires ALL three components:

**Risk Score Formula:**
```
risk_score =
  (incident_count × recency_multiplier) × 0.40 +     // Temporal: 40%
  (staleness_days / 365 × complexity) × 0.30 +       // Ownership: 30% ❌ BLOCKED
  (co_change_count × avg_coupling_rate) × 0.30       // Coupling: 30%
```

**Since staleness_days = NULL (Issue #3), the ownership component cannot be calculated, blocking the entire risk score.**

### Impact
- Cannot identify high-risk code blocks
- Cannot prioritize technical debt
- MCP server queries return incomplete results
- Core product value proposition broken

### Fix Required
1. Fix Issues #1, #2, #3 (populate ownership data)
2. Re-run crisk-index-coupling to calculate risk scores
3. Validate risk_score is populated for all blocks

---

## Data Validation Summary

### PostgreSQL Data Status

| Table | Expected | Actual | Status |
|-------|----------|--------|--------|
| `github_commits` | 517 | 517 | ✅ |
| `github_issues` | ~100 | ~100 | ✅ |
| `file_identity_map` | 360 | 360 | ✅ |
| `code_blocks` | 864 | 864 | ✅ |
| `code_block_incidents` | 112 | 112 | ⚠️ (wrong columns) |
| `code_blocks.original_author_email` | 864 | 0 | ❌ |
| `code_blocks.staleness_days` | 864 | 0 | ❌ |
| `code_blocks.risk_score` | 864 | 0 | ❌ |

### Neo4j Graph Status

| Entity/Edge | Expected | Actual | Status |
|-------------|----------|--------|--------|
| `Commit` nodes | 517 | 0 | ❌ |
| `Developer` nodes | ~10 | 0 | ❌ |
| `File` nodes | 360 | 0 | ❌ |
| `Issue` nodes | ~100 | 0 | ❌ |
| `CodeBlock` nodes | 864 | 1167 | ⚠️ (includes garbage) |
| `MODIFIED_BLOCK` edges | ~2000 | 0 | ❌ |
| `CLOSED_BY` edges | 112 | 0 | ❌ |

---

## Fix Implementation Plan

### Phase 1: Fix Critical Blockers (1-2 hours)

**Step 1.1: Update code_block_incidents Schema**
```bash
# File: internal/risk/temporal.go
# Update insert query to use new column names
# Add queries to fetch incident_date, resolution_date, incident_type
```

**Step 1.2: Add File Filtering to Atomizer**
```bash
# File: internal/atomizer/llm_extractor.go
# Add IsCodeFile() function
# Filter parsedFiles before LLM call
# Add empty block name filtering
```

**Step 1.3: Rebuild Binary**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
make build
```

### Phase 2: Clear Existing Data (5 minutes)

**Step 2.1: Clear Postgres Ingestion Data**
```sql
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
-- Clear atomizer outputs
DELETE FROM code_block_changes WHERE repo_id = 11;
DELETE FROM code_block_imports WHERE repo_id = 11;
DELETE FROM code_blocks WHERE repo_id = 11;

-- Clear incident indexing outputs
DELETE FROM code_block_incidents WHERE repo_id = 11;

-- Clear coupling indexing outputs
DELETE FROM code_block_coupling WHERE repo_id = 11;

-- Verify cleanup
SELECT
  (SELECT COUNT(*) FROM code_blocks WHERE repo_id = 11) as blocks,
  (SELECT COUNT(*) FROM code_block_changes WHERE repo_id = 11) as changes,
  (SELECT COUNT(*) FROM code_block_incidents WHERE repo_id = 11) as incidents;
-- Should return: 0, 0, 0
EOF
```

**Step 2.2: Clear Neo4j Ingestion Data**
```bash
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (n) WHERE n.repo_id = 11 DETACH DELETE n"}
  ]}'

# Verify cleanup
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (n) WHERE n.repo_id = 11 RETURN count(n) as count"}
  ]}'
# Should return: 0
```

**Step 2.3: Verify GitHub Staging Data is Preserved**
```sql
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  (SELECT COUNT(*) FROM github_commits WHERE repo_id = 11) as commits,
  (SELECT COUNT(*) FROM github_issues WHERE repo_id = 11) as issues,
  (SELECT COUNT(*) FROM file_identity_map WHERE repo_id = 11) as files;
"
# Should return: 517, ~100, 360 (preserved ✅)
```

### Phase 3: Re-run Ingestion Pipeline (30-60 minutes)

**Step 3.1: Run crisk-ingest**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Find repo path
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c \
  "SELECT absolute_path FROM github_repositories WHERE id = 11;"

# Run ingest
./bin/crisk-ingest --repo-id 11

# Validate Neo4j population
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (c:Commit {repo_id: 11}) RETURN count(c) as commits"},
    {"statement":"MATCH (d:Developer {repo_id: 11}) RETURN count(d) as developers"},
    {"statement":"MATCH (f:File {repo_id: 11}) RETURN count(f) as files"}
  ]}'
# Expected: commits=517, developers=~10, files=360
```

**Step 3.2: Run crisk-atomize**
```bash
REPO_PATH=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "SELECT absolute_path FROM github_repositories WHERE id = 11;" | xargs)

./bin/crisk-atomize --repo-id 11 --repo-path "$REPO_PATH"

# Validate CodeBlocks and MODIFIED_BLOCK edges
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (cb:CodeBlock {repo_id: 11}) RETURN count(cb) as blocks"},
    {"statement":"MATCH ()-[r:MODIFIED_BLOCK]->(:CodeBlock {repo_id: 11}) RETURN count(r) as edges"}
  ]}'
# Expected: blocks=~800 (fewer than before due to filtering), edges=~2000
```

**Step 3.3: Run crisk-index-incident**
```bash
./bin/crisk-index-incident --repo-id 11

# Validate incident links
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  COUNT(*) as total_links,
  COUNT(block_id) as with_new_columns,
  COUNT(incident_date) as with_incident_date,
  COUNT(incident_type) as with_incident_type
FROM code_block_incidents WHERE repo_id = 11;
"
# Expected: All columns populated ✅
```

**Step 3.4: Run crisk-index-ownership**
```bash
./bin/crisk-index-ownership --repo-id 11

# Validate ownership fields
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  COUNT(*) as total_blocks,
  COUNT(original_author_email) as with_author,
  COUNT(staleness_days) as with_staleness
FROM code_blocks WHERE repo_id = 11;
"
# Expected: with_author=~800, with_staleness=~800 ✅
```

**Step 3.5: Run crisk-index-coupling**
```bash
./bin/crisk-index-coupling --repo-id 11

# Validate risk scores
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  COUNT(*) as total_blocks,
  COUNT(risk_score) as with_risk_score,
  AVG(risk_score) as avg_risk,
  MAX(risk_score) as max_risk
FROM code_blocks WHERE repo_id = 11;
"
# Expected: with_risk_score=~800, avg_risk=20-40, max_risk=60-90 ✅
```

### Phase 4: Validation (10 minutes)

**Step 4.1: Run Full Data Validation**
```sql
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
-- PostgreSQL Validation
SELECT 'PostgreSQL Validation' as check_type;

SELECT
  'code_blocks' as table_name,
  COUNT(*) as total,
  COUNT(CASE WHEN block_name = '' THEN 1 END) as empty_names,
  COUNT(CASE WHEN start_line = 0 AND end_line = 0 THEN 1 END) as zero_lines,
  COUNT(original_author_email) as with_author,
  COUNT(staleness_days) as with_staleness,
  COUNT(risk_score) as with_risk_score
FROM code_blocks WHERE repo_id = 11;

SELECT
  'code_block_incidents' as table_name,
  COUNT(*) as total,
  COUNT(block_id) as with_new_fk,
  COUNT(incident_date) as with_incident_date,
  COUNT(incident_type) as with_incident_type
FROM code_block_incidents WHERE repo_id = 11;
EOF
```

**Step 4.2: Neo4j Validation**
```bash
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (c:Commit {repo_id: 11}) RETURN count(c) as commits"},
    {"statement":"MATCH (cb:CodeBlock {repo_id: 11}) RETURN count(cb) as blocks"},
    {"statement":"MATCH ()-[r:MODIFIED_BLOCK]->(:CodeBlock {repo_id: 11}) RETURN count(r) as modified_edges"},
    {"statement":"MATCH (i:Issue)-[r:FIXED_BY_BLOCK]->(:CodeBlock {repo_id: 11}) RETURN count(r) as incident_edges"}
  ]}' | python3 -m json.tool
```

**Expected Results:**
- ✅ No empty block names
- ✅ Fewer than 5 blocks with zero line numbers (only valid edge cases)
- ✅ 100% of blocks have ownership data
- ✅ 100% of blocks have risk scores
- ✅ All code_block_incidents use new schema columns
- ✅ Neo4j entity counts ≥ 95% of Postgres counts

---

## Files to Modify

### 1. internal/risk/temporal.go
**Lines 126-140:** Update code_block_incidents insert query
- Change `code_block_id` → `block_id`
- Change `fix_commit_sha` → `commit_sha`
- Add `incident_date`, `resolution_date`, `incident_type` columns
- Query github_issues table to populate new fields

### 2. internal/atomizer/llm_extractor.go
**Line 42:** Add file filtering before LLM call
- Create `IsCodeFile()` function
- Filter `parsedFiles` to code files only
- Skip LLM call if no code files

**Line 315:** Add empty block name filtering
- Skip events with empty `TargetBlockName`

### 3. internal/atomizer/diff_parser.go (optional)
Add helper functions:
- `IsCodeFile(filename string) bool`
- `IsBinaryFile(filename string) bool` (if not exists)

---

## Success Criteria

### Critical Requirements (Must Have)
- ✅ All Commit nodes in Neo4j (517 commits)
- ✅ All MODIFIED_BLOCK edges created (~2000 edges)
- ✅ All ownership fields populated (original_author, staleness_days)
- ✅ All risk scores calculated (0-100 range)
- ✅ code_block_incidents uses new schema columns
- ✅ No empty block names in database
- ✅ No code blocks from non-code files

### Quality Metrics (Should Have)
- ✅ Neo4j entity counts ≥ 95% of Postgres counts
- ✅ Zero line numbers < 5 blocks (only valid edge cases)
- ✅ Average risk score: 20-40 (reasonable distribution)
- ✅ Max risk score: 60-90 (some high-risk blocks exist)
- ✅ Incident links: 100-120 (similar to current)

### Performance Metrics (Nice to Have)
- ✅ Atomizer completes in < 60 minutes
- ✅ Ownership indexing completes in < 5 minutes
- ✅ Coupling indexing completes in < 10 minutes
- ✅ Total pipeline runtime: < 90 minutes

---

## Rollback Plan

If pipeline fails or produces invalid data:

**Step 1: Clear all ingestion data**
```bash
# Use Phase 2 cleanup scripts above
```

**Step 2: Restore to current state (optional)**
```bash
# Re-run atomizer with old code (before fixes)
git stash  # Stash fixes
make build
./bin/crisk-atomize --repo-id 11 --repo-path "$REPO_PATH"
git stash pop  # Restore fixes
```

**Step 3: Debug and iterate**
- Review logs for errors
- Check Neo4j connection issues
- Verify Postgres schema matches expectations
- Test with smaller repo first

---

## Next Steps

1. **Review this analysis** with team
2. **Decide on code_block_incidents migration** (Option A vs B)
3. **Implement fixes** (Phase 1)
4. **Clear data** (Phase 2)
5. **Re-run pipeline** (Phase 3)
6. **Validate results** (Phase 4)
7. **Document lessons learned**

---

**Document Owner:** CodeRisk Engineering
**Last Updated:** 2025-11-18
**Status:** Ready for Implementation
