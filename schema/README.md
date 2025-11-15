# CodeRisk PostgreSQL Schema

## Overview

This directory contains the PostgreSQL schema for the CodeRisk system. The schema is designed as a **"Kitchen"** (source of truth) that stores both raw GitHub API data and expensive LLM computation outputs, enabling the Neo4j **"Restaurant"** (fast query index) to be rebuilt at any time.

## Architecture Principles

### 1. PostgreSQL as Source of Truth
- **Raw Data Storage**: All GitHub API responses stored in JSONB columns
- **LLM Output Persistence**: Expensive computations (atomization, risk scoring) stored permanently
- **Reproducibility**: Neo4j can be deleted and rebuilt from Postgres alone
- **Analytics Layer**: Historical data for backfilling, debugging, auditing

### 2. Multi-Repo Safety
- **Every table** has a `repo_id` foreign key to `github_repositories`
- **No data collision** between repositories
- **Composite unique constraints** prevent duplicates within a repo
- **Cascade deletes** ensure referential integrity

### 3. Pipeline Alignment

The schema is organized to match the three `crisk init` pipelines:

```
Pipeline 1: Link Resolution → 9 tables (github_issues, github_pull_requests, etc.)
Pipeline 2: Code-Block Atomization → 2 tables (code_blocks, code_block_modifications)
Pipeline 3: Risk Indexing → 2 tables (code_block_risk_index, code_block_coupling)
Supporting Data → 7 tables (github_branches, github_contributors, etc.)
```

## Schema Files

### `00_init_schema.sql`
Complete schema creation from scratch. Run this on a fresh database or after dropping all tables.

**Usage:**
```bash
# From project root
PGPASSWORD="your_password" psql -h localhost -p 5433 -U coderisk -d coderisk -f schema/00_init_schema.sql
```

**Or use the helper script:**
```bash
chmod +x scripts/rebuild_postgres_schema.sh
./scripts/rebuild_postgres_schema.sh
```

## Table Reference

### Pipeline 1: Link Resolution (100% Confidence Fact Layer)

| Table | Purpose | Critical Columns |
|-------|---------|------------------|
| `github_repositories` | Root table for all repos | `id`, `owner`, `name`, `full_name` |
| `github_commits` | Commit history with patches | `sha`, `raw_data->files[].patch` ⭐ |
| `github_issues` | All issues (incidents) | `number`, `body`, `closed_at` |
| `github_pull_requests` | All PRs with merge data | `merge_commit_sha` ⭐ |
| `github_issue_timeline` | Lifecycle events | `event_type`, `source_sha` |
| `github_issue_comments` | Discussion threads | `body` (for LLM extraction) |
| `github_issue_commit_refs` | LLM-extracted references | `confidence`, `action` |
| `github_issue_pr_links` | Validated Issue↔PR links | `final_confidence`, `user_verified` ⭐ |
| `github_issue_no_links` | Orphaned issues | `no_links_reason` |
| `github_pr_files` | Individual file changes | `patch` |

**Key for `crisk check`:**
- `github_commits.raw_data->files[].patch`: Contains full diff for atomization
- `github_issue_pr_links.user_verified`: Links approved by user in Linker Service
- `github_pull_requests.merge_commit_sha`: 100% confidence PR→Commit link

### Pipeline 2: Code-Block Atomization (Atomic Unit Layer)

| Table | Purpose | Critical Columns |
|-------|---------|------------------|
| `code_blocks` | Individual functions/methods | `block_name`, `file_path`, `evolved_from_id` ⭐ |
| `code_block_modifications` | Transaction log of changes | `commit_sha`, `raw_llm_output` ⭐ |

**Key for `crisk check`:**
- `code_blocks.evolved_from_id`: Preserves risk history across function renames
- `code_block_modifications.raw_llm_output`: Direct LLM atomizer output
- `code_block_modifications.is_refactor_only`: Filters style-only changes from risk

### Pipeline 3: Risk Indexing (Intelligence Layer)

| Table | Purpose | Critical Columns |
|-------|---------|------------------|
| `code_block_risk_index` | Pre-computed R_temporal, R_ownership | `incident_count`, `temporal_summary` ⭐, `staleness_days` |
| `code_block_coupling` | Pre-computed R_coupling | `co_change_rate`, `reason` ⭐ |

**Key for `crisk check`:**
- `code_block_risk_index.temporal_summary`: LLM-generated incident pattern summary
- `code_block_risk_index.semantic_importance`: LLM-classified business criticality (P0/P1/P2)
- `code_block_coupling.reason`: LLM explanation of why blocks change together

### Supporting Tables

| Table | Purpose |
|-------|---------|
| `github_branches` | Branch tracking |
| `github_contributors` | Contributor metadata |
| `github_developers` | Aggregated developer activity |
| `github_languages` | Language breakdown |
| `github_dora_metrics` | DORA metrics |
| `github_trees` | Full repo snapshots |

## Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  GitHub API → PostgreSQL (Raw Data)                              │
│  (crisk init --days 90)                                          │
└────────────────────┬────────────────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌──────────────────┐
│  Pipeline 1     │    │  Pipeline 2      │
│  LLM Extraction │    │  LLM Atomization │
│  → Postgres     │    │  → Postgres      │
└────────┬────────┘    └────────┬─────────┘
         │                      │
         └──────────┬───────────┘
                    ▼
         ┌──────────────────────┐
         │  Pipeline 3          │
         │  Risk Calculation    │
         │  → Postgres          │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Graph Builder       │
         │  Postgres → Neo4j    │
         └──────────────────────┘
```

## Critical JSONB Structures

### `github_commits.raw_data` (Pipeline 2 Input)
```json
{
  "sha": "abc123...",
  "files": [
    {
      "filename": "src/TableEditor.tsx",
      "status": "modified",
      "additions": 10,
      "deletions": 3,
      "patch": "@@ -42,7 +42,10 @@\n- old line\n+ new line"
    }
  ]
}
```

### `code_block_modifications.raw_llm_output` (Pipeline 2 Output)
```json
{
  "blocks_modified": [
    {
      "block_name": "updateTableEditor",
      "block_type": "function",
      "change_type": "modified",
      "summary": "Added null check for data parameter"
    }
  ],
  "is_refactor": false,
  "confidence": 0.95
}
```

### `code_block_risk_index.familiarity_map` (Pipeline 3 Output)
```json
{
  "alice@company.com": 15,
  "bob@company.com": 3,
  "charlie@company.com": 1
}
```

## Multi-Repo Ingestion Example

```sql
-- Repo 1: omnara-ai/omnara
INSERT INTO github_repositories (github_id, owner, name, full_name)
VALUES (123456, 'omnara-ai', 'omnara', 'omnara-ai/omnara');
-- Returns repo_id=1

-- Repo 2: supabase/supabase
INSERT INTO github_repositories (github_id, owner, name, full_name)
VALUES (789012, 'supabase', 'supabase', 'supabase/supabase');
-- Returns repo_id=2

-- All child tables use repo_id to keep data separate:
INSERT INTO github_commits (repo_id, sha, ...) VALUES (1, 'abc123', ...);  -- omnara commit
INSERT INTO github_commits (repo_id, sha, ...) VALUES (2, 'def456', ...);  -- supabase commit

-- No collision! Queries are always scoped:
SELECT * FROM github_commits WHERE repo_id = 1;  -- Only omnara commits
SELECT * FROM code_blocks WHERE repo_id = 2;     -- Only supabase code blocks
```

## Schema Validation

The schema includes automatic validation that runs after creation:

```sql
DO $$
BEGIN
    -- Checks that all 20 required tables exist
    -- Raises exception if any are missing
    -- Prints summary on success
END $$;
```

## Rebuilding the Schema

### Option 1: Helper Script (Recommended)
```bash
cd /Users/rohankatakam/Documents/brain/coderisk
./scripts/rebuild_postgres_schema.sh

# Or skip confirmation:
./scripts/rebuild_postgres_schema.sh --force
```

### Option 2: Manual
```bash
# Drop all tables
psql -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Recreate schema
psql -f schema/00_init_schema.sql
```

## Migration from Old Schema

If you have an existing `coderisk` database with data you want to preserve, use the migration approach:

```bash
# 1. Backup existing data
pg_dump -h localhost -p 5433 -U coderisk coderisk > backup.sql

# 2. Rebuild schema
./scripts/rebuild_postgres_schema.sh

# 3. Migrate data (custom migration script needed based on your data)
# Note: The new schema is cleaner, so a full re-ingestion is recommended
```

## Testing the Schema

```bash
# 1. Rebuild schema
./scripts/rebuild_postgres_schema.sh

# 2. Test GitHub API ingestion
cd /tmp
git clone https://github.com/omnara-ai/omnara
cd omnara
crisk init --days 30

# 3. Verify data
psql -c "SELECT repo_id, COUNT(*) FROM github_commits GROUP BY repo_id;"
psql -c "SELECT repo_id, COUNT(*) FROM github_issues GROUP BY repo_id;"
psql -c "SELECT repo_id, COUNT(*) FROM code_blocks GROUP BY repo_id;"

# 4. Test multi-repo
cd /tmp
git clone https://github.com/supabase/supabase
cd supabase
crisk init --days 30

# 5. Verify no collision
psql -c "SELECT owner, name, COUNT(*) as commits FROM github_repositories r
         JOIN github_commits c ON r.id = c.repo_id
         GROUP BY owner, name;"
```

## Performance Considerations

### Indexes
All critical foreign keys and query paths have indexes:
- `repo_id` on every table
- Composite indexes for unique constraints
- GIN indexes for JSONB columns
- Conditional indexes for sparse data

### Triggers
Auto-updating timestamps on:
- `github_repositories.updated_at`
- `code_blocks.updated_at`
- `code_block_risk_index.updated_at`
- `code_block_coupling.updated_at`
- `github_issue_pr_links.updated_at`

### Cleanup
Use CASCADE deletes to maintain referential integrity:
```sql
-- Deleting a repo removes all child data automatically
DELETE FROM github_repositories WHERE id = 1;
-- All commits, issues, PRs, code_blocks, etc. are deleted
```

## Support

For questions or issues:
1. Check the validation output from `00_init_schema.sql`
2. Verify table counts: `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';`
3. Check for foreign key errors in logs
4. Review `CURRENT_POSTGRES_SCHEMA_OVERVIEW.md` for detailed column documentation

---

**Schema Version:** 2.0
**Last Updated:** 2025-11-14
**Aligned With:** `crisk init` three-pipeline architecture
