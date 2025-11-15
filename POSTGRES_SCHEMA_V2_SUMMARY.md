# PostgreSQL Schema V2 - Implementation Summary

**Date:** 2025-11-14
**Status:** ✅ Ready for Testing

---

## What Was Done

### 1. Created Clean Schema from Scratch
Built a new PostgreSQL schema (`schema/00_init_schema.sql`) that is:
- **Aligned** with the three-pipeline `crisk init` architecture
- **Multi-repo safe** (every table has `repo_id` foreign key)
- **Non-redundant** (removed duplicate/legacy tables)
- **LLM-output preserving** (stores expensive computations)
- **Reproducible** (Neo4j can be rebuilt from Postgres)

### 2. Schema Organization

**Total: 20 Tables** organized by purpose:

#### Pipeline 1: Link Resolution (9 tables)
```
github_repositories          ← Root table (multi-repo master list)
github_commits              ← Contains raw_data->files[].patch for atomization
github_issues               ← Incidents/bugs
github_pull_requests        ← PRs with merge_commit_sha
github_issue_timeline       ← Timeline events for REFERENCES/CLOSED_BY edges
github_issue_comments       ← Comments for LLM extraction
github_issue_commit_refs    ← LLM-extracted references
github_issue_pr_links       ← Validated Issue↔PR links (user_verified flag)
github_issue_no_links       ← Orphaned issues
github_pr_files             ← Individual file changes
```

#### Pipeline 2: Code-Block Atomization (2 tables)
```
code_blocks                 ← Individual functions/methods
code_block_modifications    ← Transaction log (LLM output stored here)
```

#### Pipeline 3: Risk Indexing (2 tables)
```
code_block_risk_index       ← Pre-computed R_temporal, R_ownership
code_block_coupling         ← Pre-computed R_coupling
```

#### Supporting Tables (7 tables)
```
github_branches
github_contributors
github_developers
github_languages
github_dora_metrics
github_trees
```

### 3. Key Improvements Over Old Schema

| Old Schema Issue | New Schema Solution |
|------------------|---------------------|
| ❌ Redundant tables (`repositories` + `github_repositories`) | ✅ Single `github_repositories` table |
| ❌ Missing code-block tables | ✅ Added `code_blocks`, `code_block_modifications` |
| ❌ No LLM output storage | ✅ JSONB columns: `raw_llm_output`, `temporal_summary`, `reason` |
| ❌ No rename tracking | ✅ `evolved_from_id` preserves risk history |
| ❌ No multi-repo safety validation | ✅ All tables have `repo_id` FK with CASCADE |
| ❌ No refactor filtering | ✅ `is_refactor_only` flag in modifications |

### 4. Critical JSONB Structures

#### `github_commits.raw_data` (Input to Pipeline 2)
```json
{
  "files": [{
    "filename": "src/TableEditor.tsx",
    "patch": "@@ -42,7 +42,10 @@\n- old\n+ new"
  }]
}
```

#### `code_block_modifications.raw_llm_output` (Output from Pipeline 2)
```json
{
  "blocks_modified": [{
    "block_name": "updateTableEditor",
    "change_type": "modified"
  }],
  "is_refactor": false
}
```

#### `code_block_risk_index.familiarity_map` (Output from Pipeline 3)
```json
{
  "alice@co.com": 15,
  "bob@co.com": 3
}
```

### 5. Files Created

```
coderisk/
├── schema/
│   ├── 00_init_schema.sql          ← Complete schema (570 lines)
│   └── README.md                   ← Documentation
├── scripts/
│   └── rebuild_postgres_schema.sh  ← Safe rebuild helper
└── POSTGRES_SCHEMA_V2_SUMMARY.md   ← This file
```

---

## How to Use

### Step 1: Rebuild PostgreSQL Schema

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Option A: Interactive (asks for confirmation)
./scripts/rebuild_postgres_schema.sh

# Option B: Force mode (no confirmation)
./scripts/rebuild_postgres_schema.sh --force
```

**Output:**
```
✅ Schema initialization complete!
   Total tables created: 20

📊 Pipeline Table Mapping:
   Pipeline 1 (Link Resolution): 9 tables
   Pipeline 2 (Atomization): 2 tables
   Pipeline 3 (Risk Indexing): 2 tables
   Supporting tables: 7 tables

✅ Ready for multi-repo ingestion!
```

### Step 2: Test GitHub API Ingestion

```bash
# Test on omnara repository
cd /tmp
rm -rf omnara  # Clean start
git clone https://github.com/omnara-ai/omnara
cd omnara
crisk init --days 30  # Without --llm flag (100% confidence graph only)
```

### Step 3: Verify Data in PostgreSQL

```bash
# Check commits were ingested
psql -c "SELECT repo_id, COUNT(*) as commits,
         COUNT(*) FILTER (WHERE raw_data->'files' IS NOT NULL) as with_patches
         FROM github_commits GROUP BY repo_id;"

# Check issues
psql -c "SELECT repo_id, COUNT(*) as issues,
         COUNT(*) FILTER (WHERE body IS NOT NULL) as with_description
         FROM github_issues GROUP BY repo_id;"

# Verify multi-repo safety
psql -c "SELECT id, owner, name, full_name FROM github_repositories ORDER BY id;"
```

### Step 4: Test Multi-Repo Support

```bash
# Ingest a second repository
cd /tmp
rm -rf supabase
git clone https://github.com/supabase/supabase
cd supabase
crisk init --days 30

# Verify no data collision
psql -c "SELECT r.owner, r.name,
         COUNT(DISTINCT c.sha) as commits,
         COUNT(DISTINCT i.number) as issues
         FROM github_repositories r
         LEFT JOIN github_commits c ON r.id = c.repo_id
         LEFT JOIN github_issues i ON r.id = i.repo_id
         GROUP BY r.owner, r.name;"
```

Expected output:
```
   owner    |   name   | commits | issues
------------+----------+---------+--------
 omnara-ai  | omnara   |      50 |     41
 supabase   | supabase |     200 |    150
```

### Step 5: Verify Neo4j Graph Construction

```bash
# Rebuild graph from PostgreSQL data
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/test-graph-construction ./cmd/test-graph-construction
./bin/test-graph-construction

# Expected output:
# ✅ Graph construction complete!
#   Nodes created: 300
#   Edges created: 150
#   ✅ No duplicate edges found
```

---

## Data Flow Verification

```
┌──────────────────────────────────────────────┐
│ 1. GitHub API → PostgreSQL                   │
│    crisk init --days 30                      │
│    ✓ github_commits (with patches)          │
│    ✓ github_issues (with descriptions)      │
│    ✓ github_pull_requests (with merge SHAs) │
└────────────────┬─────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────┐
│ 2. PostgreSQL → Neo4j (100% confidence)      │
│    graph.Builder.BuildGraph()                │
│    ✓ (Developer)-[:AUTHORED]->(Commit)      │
│    ✓ (PR)-[:MERGED_AS]->(Commit)            │
│    ✓ (Commit)-[:MODIFIED]->(File)           │
└────────────────┬─────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────┐
│ 3. crisk check <file>                        │
│    Queries Neo4j graph                       │
│    ✓ Fast response (<1s)                    │
│    ✓ File-level risk (for now)              │
└──────────────────────────────────────────────┘
```

---

## Next Steps (After Verification)

Once the schema is verified working:

### 1. Implement Pipeline 2 (Code-Block Atomization)
- [ ] Create LLM service to parse `github_commits.raw_data->files[].patch`
- [ ] Extract functions/methods and populate `code_blocks`
- [ ] Store LLM output in `code_block_modifications.raw_llm_output`
- [ ] Handle function renames via `evolved_from_id`

### 2. Implement Pipeline 3 (Risk Indexing)
- [ ] Calculate incident counts from `github_issue_pr_links`
- [ ] Generate LLM summaries for `temporal_summary`
- [ ] Compute staleness and familiarity metrics
- [ ] Calculate co-change patterns for coupling risk

### 3. Update Graph Builder
- [ ] Extend `graph.Builder` to create `CodeBlock` nodes
- [ ] Add `(CodeBlock)-[:WAS_ROOT_CAUSE_IN]->(Issue)` edges
- [ ] Add `(CodeBlock)-[:CO_CHANGES_WITH]->(CodeBlock)` edges

### 4. Update `crisk check`
- [ ] Query code-block level risk instead of file-level
- [ ] Display function-specific incident history
- [ ] Show co-change warnings

---

## Troubleshooting

### Issue: "relation does not exist"
```bash
# Rebuild schema
./scripts/rebuild_postgres_schema.sh --force
```

### Issue: "duplicate key value violates unique constraint"
```bash
# Check for existing repos with same full_name
psql -c "SELECT * FROM github_repositories WHERE full_name = 'owner/repo';"

# Delete if needed
psql -c "DELETE FROM github_repositories WHERE full_name = 'owner/repo';"
```

### Issue: "cannot drop table because other objects depend on it"
```bash
# Use CASCADE delete
psql -c "DROP TABLE table_name CASCADE;"

# Or rebuild entire schema
./scripts/rebuild_postgres_schema.sh --force
```

### Issue: Missing patch data in commits
```bash
# Verify GitHub API is returning file data
psql -c "SELECT sha,
         jsonb_array_length(raw_data->'files') as file_count,
         (raw_data->'files'->0->>'patch') IS NOT NULL as has_patch
         FROM github_commits LIMIT 5;"

# Expected: file_count > 0 and has_patch = true
```

---

## Migration Notes

### From Old Schema to New Schema

**⚠️ BREAKING CHANGE:** This is a complete schema redesign. Data migration is not automated.

**Recommended Approach:**
1. ✅ Rebuild schema from scratch (use rebuild script)
2. ✅ Re-run `crisk init` on all repositories
3. ✅ Faster and cleaner than migration

**Alternative (Preserve Data):**
1. Backup old database: `pg_dump coderisk > backup.sql`
2. Create custom migration scripts (per-table basis)
3. Map old columns to new columns
4. Re-run Pipeline 2 and 3 to generate new tables

**Not Recommended:** The new schema is significantly cleaner. Fresh ingestion is the best path.

---

## Validation Checklist

Before proceeding to Pipeline 2/3 implementation:

- [x] Schema created successfully (20 tables)
- [ ] GitHub API ingestion works (`crisk init`)
- [ ] Multi-repo support verified (2+ repos ingested)
- [ ] No data collision between repos
- [ ] Neo4j graph builds from PostgreSQL data
- [ ] `crisk check` works with existing graph
- [ ] All foreign keys enforce correctly
- [ ] Cascade deletes work as expected

---

## Performance Benchmarks (Target)

| Operation | Target Time | Notes |
|-----------|-------------|-------|
| `crisk init` (90 days) | < 5 min | GitHub API + Postgres insert |
| Schema rebuild | < 10 sec | Drop + create all tables |
| Multi-repo ingestion (2 repos) | < 10 min | No data collision |
| Graph rebuild from Postgres | < 30 sec | 1000 commits, 500 issues |
| `crisk check <file>` | < 1 sec | Neo4j query |

---

**Schema Version:** 2.0
**Status:** ✅ Ready for Testing
**Next Milestone:** Verify GitHub API → PostgreSQL ingestion works
