# Schema Compliance Report

**Generated:** 2025-11-18
**Repository:** mcp-use (repo_id=11)
**Reference:** /Users/rohankatakam/Documents/brain/docs/DATA_SCHEMA_REFERENCE.md

## Executive Summary

Schema validation reveals **CRITICAL compliance issues** in the `code_block_incidents` table and **MISSING risk signals** in the `code_blocks` table. The database schema has undergone a migration (old→new column names) but the application code has NOT been updated to use the new schema.

### Status Overview
- ✅ **code_blocks table structure:** COMPLIANT (all required columns exist)
- ❌ **code_block_incidents table:** NON-COMPLIANT (data written to deprecated columns)
- ❌ **ownership signals:** MISSING (0 blocks have ownership data)
- ✅ **coupling signals:** COMPLIANT (864 blocks have coupling data)
- ⚠️  **risk_score:** MISSING (not calculated yet - expected after crisk-index-coupling)

---

## CRITICAL ISSUE 1: code_block_incidents Schema Mismatch

### Problem

The `code_block_incidents` table has **BOTH old and new columns**, but the application code (`internal/risk/temporal.go:126-131`) writes data to the **OLD columns only**.

### Evidence

**Database Schema:**
```sql
-- OLD columns (currently populated by code)
code_block_id   | bigint        -- FK to code_blocks
fix_commit_sha  | varchar(40)   -- Commit that fixed the issue
fixed_at        | timestamp     -- When issue was fixed

-- NEW columns (per DATA_SCHEMA_REFERENCE.md, currently EMPTY)
block_id        | bigint        -- FK to code_blocks (NULL for all 112 rows)
commit_sha      | text          -- Commit SHA (NULL for all 112 rows)
incident_date   | timestamp     -- Issue creation date (NULL for all 112 rows)
resolution_date | timestamp     -- Issue closure date (NULL for all 112 rows)
incident_type   | text          -- Issue labels (NULL for all 112 rows)
```

**Query Results:**
```sql
SELECT
  COUNT(*) as total_links,
  COUNT(block_id) as with_block_id_new,
  COUNT(code_block_id) as with_code_block_id_old
FROM code_block_incidents WHERE repo_id = 11;

 total_links | with_block_id_new | with_code_block_id_old
-------------+-------------------+------------------------
         112 |                 0 |                    112
```

**DATA_SCHEMA_REFERENCE.md Expectation (lines 330-344):**
```markdown
#### `code_block_incidents`

| Column | Type | Description | Populated By |
|--------|------|-------------|--------------|
| `id` | BIGSERIAL | Primary key | crisk-index-incident |
| `repo_id` | BIGINT | FK to github_repositories | crisk-index-incident |
| `block_id` | BIGINT | FK to code_blocks | crisk-index-incident |
| `issue_id` | BIGINT | FK to github_issues | crisk-index-incident |
| `commit_sha` | TEXT | Commit that closed the issue | crisk-index-incident |
| `incident_date` | TIMESTAMP | When issue was created | crisk-index-incident |
| `resolution_date` | TIMESTAMP | When issue was closed | crisk-index-incident |
| `incident_type` | TEXT | Issue labels (e.g., "bug", "security") | crisk-index-incident |
| `created_at` | TIMESTAMP | Record insertion time | crisk-index-incident |
```

**Actual Code (internal/risk/temporal.go:126-131):**
```go
insertQuery := `
    INSERT INTO code_block_incidents (
        repo_id, code_block_id, issue_id,    // ❌ using OLD column name
        confidence, evidence_source, evidence_text,
        fix_commit_sha, fixed_at,             // ❌ using OLD column names
        created_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
```

### Impact

1. **Data in wrong columns:** All 112 incident links are written to deprecated columns (`code_block_id`, `fix_commit_sha`, `fixed_at`)
2. **Schema reference queries broken:** Any queries using the new column names (`block_id`, `commit_sha`, `incident_date`) will return NULL
3. **Missing required fields:** `incident_date`, `resolution_date`, `incident_type` are never populated
4. **Foreign key constraint confusion:** Table has TWO FK constraints to code_blocks:
   - `code_block_incidents_code_block_id_fkey` (old, working)
   - `code_block_incidents_block_id_fkey` (new, unused)

### Recommendation

**DECISION REQUIRED:** Choose one of two paths:

**Option A: Migrate to New Schema (Recommended)**
1. Update `internal/risk/temporal.go:126-131` to use new column names:
   - `code_block_id` → `block_id`
   - `fix_commit_sha` → `commit_sha`
   - Add population of `incident_date` (from github_issues.created_at)
   - Add population of `resolution_date` (from github_issues.closed_at)
   - Add population of `incident_type` (from github_issues.labels)
2. Run migration script to copy data from old→new columns
3. Drop old columns once migration is verified

**Option B: Update Schema Reference Document**
1. Update DATA_SCHEMA_REFERENCE.md to reflect the ACTUAL column names in use
2. Mark new columns as deprecated/unused
3. Risk: Schema reference becomes out of sync with intended design

---

## CRITICAL ISSUE 2: Missing Ownership Signals

### Problem

All 864 code blocks have **NULL ownership properties**. The `crisk-index-ownership` command ran but updated 0 blocks.

### Evidence

**Query Results:**
```sql
SELECT
  COUNT(*) as total_blocks,
  COUNT(original_author_email) as with_original_author,
  COUNT(last_modifier_email) as with_last_modifier,
  COUNT(staleness_days) as with_staleness
FROM code_blocks WHERE repo_id = 11;

 total_blocks | with_original_author | with_last_modifier | with_staleness
--------------+----------------------+--------------------+----------------
          864 |                    0 |                  0 |              0
```

**Expected Behavior (per DATA_SCHEMA_REFERENCE.md lines 822-842):**

After `crisk-index-ownership` runs, the following columns should be populated:
- `original_author_email` (who created the block)
- `last_modifier_email` (who last touched it)
- `staleness_days` (days since last modification)
- `familiarity_map` (JSONB: developer → modification count)

### Root Cause Investigation Needed

The ownership indexing failure suggests one of:
1. **Missing MODIFIED_BLOCK relationships in Neo4j:** The ownership calculator traverses `(Commit)-[:MODIFIED_BLOCK]->(CodeBlock)` edges. If these don't exist, ownership cannot be calculated.
2. **Graph query failure:** The Neo4j query may be failing silently
3. **Block ID mismatch:** PostgreSQL block IDs may not match Neo4j block IDs

**Verification Query (Neo4j):**
```cypher
MATCH (c:Commit)-[r:MODIFIED_BLOCK]->(cb:CodeBlock)
WHERE cb.repo_id = 11
RETURN COUNT(r) as modified_block_edges
```

If this returns 0, the graph construction (crisk-atomize) failed to create MODIFIED_BLOCK edges.

### Recommendation

1. Verify Neo4j has MODIFIED_BLOCK edges for repo_id=11
2. Check crisk-atomize logs for edge creation failures
3. If edges are missing, re-run crisk-atomize with debug logging
4. Once edges exist, re-run crisk-index-ownership

---

## Issue 3: Missing Risk Scores

### Problem

All 864 code blocks have **NULL risk_score**. This is expected behavior if `crisk-index-coupling` has not completed risk calculation yet.

### Evidence

```sql
SELECT
  COUNT(*) as total_blocks,
  COUNT(risk_score) as with_risk_score,
  COUNT(co_change_count) as with_coupling
FROM code_blocks WHERE repo_id = 11;

 total_blocks | with_risk_score | with_coupling
--------------+-----------------+---------------
          864 |               0 |           864
```

### Expected Behavior

Per DATA_SCHEMA_REFERENCE.md (lines 977-999), the `risk_score` is calculated by `crisk-index-coupling` as the FINAL step using:

**Risk Score Formula (weighted):**
- **Temporal Component (40% weight):** `incident_count` × recency_multiplier
- **Ownership Component (30% weight):** `staleness_days` / 365 × complexity_estimate
- **Coupling Component (30% weight):** `co_change_count` × `avg_coupling_rate`

**Current Blocker:** The ownership component is 0 because `staleness_days` is NULL. This means risk scores CANNOT be calculated until ownership indexing succeeds.

### Recommendation

1. Fix ownership indexing first (Issue 2)
2. Re-run `crisk-index-coupling` to calculate risk_score
3. Validate risk scores are in 0-100 range

---

## Issue 4: Incident Count Confusion

### Problem (RESOLVED)

The incident indexing report showed "Max incidents/block: 1" which seemed wrong when Issue #120 affects 52 blocks. However, investigation proved this is **CORRECT**.

### Explanation

The data model is:
- **One issue** can affect **many blocks** (e.g., Issue #120 → 52 blocks)
- Each block is linked to **ONE issue per commit** (each link in `code_block_incidents` represents one issue fixing one block)
- Therefore, `incident_count` per block = number of UNIQUE issues that fixed that block

**Query Verification:**
```sql
-- How many blocks does each issue affect?
SELECT i.number, COUNT(cbi.code_block_id) as affected_blocks
FROM github_issues i
JOIN code_block_incidents cbi ON i.id = cbi.issue_id
GROUP BY i.number
ORDER BY affected_blocks DESC;

 number | affected_blocks
--------+-----------------
    120 |              52  -- Issue #120 affects 52 blocks
    138 |              18  -- Issue #138 affects 18 blocks
    ...
```

```sql
-- How many issues fixed each block?
SELECT code_block_id, COUNT(*) as issues_that_fixed_this_block
FROM code_block_incidents
GROUP BY code_block_id
HAVING COUNT(*) > 1;

-- Result: 0 rows (no block has been fixed by more than 1 issue)
```

**Conclusion:** The incident counting is working correctly. Each block has `incident_count = 1` because each block was fixed by exactly ONE issue. The fact that one issue affects many blocks is tracked in the `code_block_incidents` table as many separate rows.

---

## Summary of Required Actions

### Priority 1 (CRITICAL - Blocking Risk Calculation)

1. **Fix ownership indexing** (Issue 2)
   - Verify Neo4j MODIFIED_BLOCK edges exist
   - Debug ownership calculator if edges exist
   - Re-run crisk-index-ownership

### Priority 2 (Schema Consistency)

2. **Resolve code_block_incidents schema mismatch** (Issue 1)
   - DECISION: Migrate code to new schema OR update schema reference
   - If migrating: Update temporal.go + run migration script
   - If not migrating: Update DATA_SCHEMA_REFERENCE.md

### Priority 3 (Final Risk Calculation)

3. **Calculate risk scores** (Issue 3)
   - After ownership indexing succeeds
   - Re-run crisk-index-coupling
   - Validate risk scores are populated

---

## Compliance Checklist

### code_blocks Table (Per Schema Reference Lines 239-269)

| Column | Required | Status | Notes |
|--------|----------|--------|-------|
| `id` | ✅ | ✅ COMPLIANT | 864 blocks |
| `repo_id` | ✅ | ✅ COMPLIANT | All set to 11 |
| `canonical_file_path` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `path_at_creation` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `block_type` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `block_name` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `signature` | ❓ | ⚠️ NOT CHECKED | Optional field |
| `start_line` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `end_line` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `language` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `complexity_estimate` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `first_seen_commit` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `last_modified_commit` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `incident_count` | ✅ | ✅ COMPLIANT | 864/864 populated (752 with 0, 112 with 1) |
| `last_incident_date` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `temporal_summary` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `original_author_email` | ✅ | ❌ NON-COMPLIANT | 0/864 populated |
| `last_modifier_email` | ✅ | ❌ NON-COMPLIANT | 0/864 populated |
| `staleness_days` | ✅ | ❌ NON-COMPLIANT | 0/864 populated |
| `familiarity_map` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `co_change_count` | ✅ | ✅ COMPLIANT | 864/864 populated |
| `avg_coupling_rate` | ❓ | ⚠️ NOT CHECKED | Not validated |
| `risk_score` | ✅ | ❌ NON-COMPLIANT | 0/864 populated (blocked by ownership) |

### code_block_incidents Table (Per Schema Reference Lines 330-350)

| Column | Required | Status | Notes |
|--------|----------|--------|-------|
| `id` | ✅ | ✅ COMPLIANT | 112 incident links |
| `repo_id` | ✅ | ✅ COMPLIANT | All set to 11 |
| `block_id` | ✅ | ❌ NON-COMPLIANT | 0/112 populated (data in code_block_id instead) |
| `issue_id` | ✅ | ✅ COMPLIANT | 112/112 populated |
| `commit_sha` | ✅ | ❌ NON-COMPLIANT | 0/112 populated (data in fix_commit_sha instead) |
| `incident_date` | ✅ | ❌ NON-COMPLIANT | 0/112 populated |
| `resolution_date` | ✅ | ❌ NON-COMPLIANT | 0/112 populated |
| `incident_type` | ✅ | ❌ NON-COMPLIANT | 0/112 populated |
| `created_at` | ✅ | ✅ COMPLIANT | 112/112 populated |

---

## References

1. **Schema Reference:** /Users/rohankatakam/Documents/brain/docs/DATA_SCHEMA_REFERENCE.md
2. **Incident Indexing Code:** /Users/rohankatakam/Documents/brain/coderisk/internal/risk/temporal.go (lines 126-131)
3. **Database:** PostgreSQL (localhost:5433, database=coderisk)
4. **Repository:** mcp-use (repo_id=11, 517 commits, 864 code blocks, 360 files)
