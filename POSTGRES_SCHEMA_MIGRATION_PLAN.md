# PostgreSQL Schema Migration Plan

**Date:** 2025-11-14
**Status:** 📋 Ready for Execution

---

## Current State Analysis

### What We Have Now

**Schema Status:** Old schema (30 tables) with omnara data
- ✅ 30 tables total
- ✅ Omnara repo data: 3 commits, 41 issues, 16 PRs
- ❌ Missing new schema tables: `code_blocks`, `code_block_modifications`, `code_block_risk_index`, `code_block_coupling`
- ⚠️ Schema is hybrid: Some tables match new schema, some don't

**Data Breakdown:**
```sql
github_repositories: 1 repo (omnara-ai/omnara, repo_id=6)
github_commits: 3 commits with raw_data (patches included)
github_issues: 41 issues
github_pull_requests: 16 PRs
github_issue_timeline: Unknown count (table exists but no repo_id column - old structure)
```

**Key Finding:** We have a **hybrid schema** - some tables were updated with helper views, but the core schema is still the old structure.

---

## Migration Strategy Analysis

### Option 1: Fresh Schema Rebuild (RECOMMENDED)

**Approach:** Drop all tables, apply new schema, re-ingest omnara data

**Pros:**
- ✅ Clean slate - guaranteed schema consistency
- ✅ Fast (omnara ingestion ~5 min with new optimized fetcher)
- ✅ No complex data transformation needed
- ✅ Tests new schema + new fetcher optimizations
- ✅ All 20 tables created correctly

**Cons:**
- ❌ Loses existing data temporarily (3 commits, 41 issues, 16 PRs)
- ⚠️ Requires re-fetch from GitHub API (~100 API calls)

**Time Required:** ~10 minutes total
- Schema rebuild: 10 seconds
- GitHub API re-ingestion: ~5 minutes (optimized fetcher)
- Graph reconstruction: 30 seconds

---

### Option 2: In-Place Migration (NOT RECOMMENDED)

**Approach:** Alter existing tables, transform data in place

**Pros:**
- ✅ Preserves existing data without re-fetch

**Cons:**
- ❌ Complex migration scripts needed (20+ tables)
- ❌ High risk of data corruption
- ❌ Time-consuming (~2 hours to write migrations)
- ❌ Difficult to validate correctness
- ❌ github_issue_timeline has no repo_id (would need manual backfill)
- ❌ May miss subtle schema differences

**Time Required:** ~3 hours total
- Write migration scripts: 2 hours
- Test migrations: 30 minutes
- Debug issues: 30+ minutes

---

## Recommended Approach: Fresh Schema Rebuild

Given that:
1. We only have **60 total records** (3 commits + 41 issues + 16 PRs)
2. GitHub API re-fetch takes **~5 minutes** with optimized fetcher
3. Fresh rebuild guarantees **100% schema correctness**
4. We're still in **development/testing phase** (not production)

**Verdict: Fresh rebuild is the clear winner.**

---

## Migration Steps (Fresh Rebuild)

### Step 1: Backup Current Data (Optional but Recommended)

```bash
# Create backup of current database
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" pg_dump \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -f /Users/rohankatakam/Documents/brain/coderisk/backups/omnara_backup_$(date +%Y%m%d_%H%M%S).sql

# Expected file size: ~500KB (minimal data)
```

**Note:** This is optional since we can always re-fetch from GitHub, but provides safety net.

---

### Step 2: Drop Existing Schema

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Use the rebuild script (recommended)
./scripts/rebuild_postgres_schema.sh --force

# This will:
# 1. Drop all 30 existing tables
# 2. Drop all functions and triggers
# 3. Create fresh schema with 20 tables
# 4. Validate all tables exist
```

**Expected Output:**
```
✅ Schema rebuild complete!
   • 20 tables created
   • Multi-repo safe (all tables have repo_id)
   • Ready for GitHub API ingestion
```

---

### Step 3: Verify New Schema

```bash
# Check table count (should be 20, not 30)
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';"

# Expected: 20

# Verify critical tables exist
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT table_name FROM information_schema.tables
      WHERE table_schema='public'
      AND table_name IN ('github_repositories', 'github_commits', 'code_blocks', 'code_block_risk_index')
      ORDER BY table_name;"

# Expected: All 4 tables present
```

---

### Step 4: Re-Ingest Omnara Data

```bash
# Clone fresh omnara repository
cd /tmp
rm -rf omnara
git clone https://github.com/omnara-ai/omnara
cd omnara

# Run crisk init with new optimized fetcher
# Using --days 365 to get full history (omnara is relatively new)
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365
```

**Expected Output:**
```
🔍 Fetching GitHub data for omnara-ai/omnara...
  ✓ Repository ID: 1
  ℹ️  Rate limit: 5000/5000 remaining
  ✓ Fetched 3 commits
  ✓ Fetched 41 issues
  ✓ Fetched 16 pull requests
  ✓ Fetched timeline events
  ✓ Fetched issue comments

[2/4] Building knowledge graph (100% Confidence)...
  ✓ Graph construction complete!
    Nodes: 105
    Edges: 50

✅ CodeRisk initialized for omnara-ai/omnara (100% Confidence Graph)
```

**Time:** ~5 minutes (with new 1.18 req/sec rate limit)

---

### Step 5: Verify Data Integrity

```bash
# Check data was ingested correctly
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT
        (SELECT COUNT(*) FROM github_repositories) as repos,
        (SELECT COUNT(*) FROM github_commits WHERE repo_id = 1) as commits,
        (SELECT COUNT(*) FROM github_issues WHERE repo_id = 1) as issues,
        (SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = 1) as prs,
        (SELECT COUNT(*) FROM github_issue_timeline) as timeline_events,
        (SELECT COUNT(*) FROM github_issue_pr_links WHERE repo_id = 1) as issue_pr_links;"
```

**Expected:**
```
 repos | commits | issues | prs | timeline_events | issue_pr_links
-------+---------+--------+-----+-----------------+----------------
     1 |       3 |     41 |  16 |            ~50  |           ~15
```

**Note:** Timeline events and issue_pr_links counts may vary as these are derived from API responses.

---

### Step 6: Verify Patch Data (Critical for Pipeline 2)

```bash
# Verify commits have patch data for atomization
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT
        sha,
        jsonb_array_length(raw_data->'files') as file_count,
        (raw_data->'files'->0->>'patch') IS NOT NULL as has_patch
      FROM github_commits
      WHERE repo_id = 1
      LIMIT 3;"
```

**Expected:**
```
    sha     | file_count | has_patch
------------+------------+-----------
 abc123...  |          5 | t
 def456...  |          3 | t
 ghi789...  |          7 | t
```

All rows should have `has_patch = t` (true).

---

### Step 7: Test Graph Query

```bash
# Test that crisk check works with new schema
cd /tmp/omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check apps/web/src/components/dashboard/SidebarDashboardLayout.tsx
```

**Expected Output:**
```
📊 Risk Analysis: apps/web/src/components/dashboard/SidebarDashboardLayout.tsx

File Risk Summary:
  • Temporal Risk (R_temporal): Medium
  • Ownership Risk (R_ownership): Low
  • Coupling Risk (R_coupling): Medium

Incident History:
  • 2 past incidents affecting this file
  • Last incident: 2024-10-15

Developer Familiarity:
  • alice@omnara.ai: 5 edits (high familiarity)
  • bob@omnara.ai: 1 edit (low familiarity)
```

---

### Step 8: Validate Multi-Repo Safety (Optional)

```bash
# Ingest a second repository to test multi-repo safety
cd /tmp
rm -rf supabase
git clone https://github.com/supabase/supabase
cd supabase
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 30

# Verify no data collision
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT r.id, r.owner, r.name,
             COUNT(DISTINCT c.sha) as commits,
             COUNT(DISTINCT i.number) as issues
      FROM github_repositories r
      LEFT JOIN github_commits c ON r.id = c.repo_id
      LEFT JOIN github_issues i ON r.id = i.repo_id
      GROUP BY r.id, r.owner, r.name
      ORDER BY r.id;"
```

**Expected:**
```
 id |   owner   |   name   | commits | issues
----+-----------+----------+---------+--------
  1 | omnara-ai | omnara   |       3 |     41
  2 | supabase  | supabase |     ~50 |    ~20
```

Each repo should have separate `id` and isolated data.

---

## What Gets Preserved vs. Lost

### ✅ Preserved (Re-fetched from GitHub API)
- Repository metadata (owner, name, description, stars, etc.)
- All commits with full patch data
- All issues with descriptions, labels, state
- All pull requests with merge data
- Issue timeline events (cross-references, closed-by)
- Issue comments
- Pull request files
- Branches, contributors, languages

### ❌ Lost (Not in GitHub API, but not critical)
- PostgreSQL internal IDs (auto-generated anyway)
- Old `processed_at` timestamps (fresh ingestion starts clean)
- Any manual database edits (we shouldn't have any)
- Old tables: `repositories`, `incidents`, `incident_files`, `investigations`, `clqs_scores`, etc.

### ⚠️ Not Yet Populated (Require Pipeline 2/3)
- `code_blocks` table (requires LLM atomization - Pipeline 2)
- `code_block_modifications` (requires LLM atomization - Pipeline 2)
- `code_block_risk_index` (requires risk calculation - Pipeline 3)
- `code_block_coupling` (requires co-change analysis - Pipeline 3)

**Note:** These tables are empty even after migration because they require Pipelines 2 and 3 implementation.

---

## Rollback Plan (If Something Goes Wrong)

### Option 1: Restore from Backup (If Created)
```bash
# Drop corrupted schema
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Restore backup
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -f /Users/rohankatakam/Documents/brain/coderisk/backups/omnara_backup_YYYYMMDD_HHMMSS.sql
```

### Option 2: Fresh Rebuild (If No Backup)
```bash
# Just re-run the migration steps from Step 2
./scripts/rebuild_postgres_schema.sh --force
cd /tmp/omnara
crisk init --days 365
```

**Recovery Time:** 5-10 minutes

---

## Post-Migration Validation Checklist

- [ ] Schema has exactly 20 tables (not 30)
- [ ] `github_repositories` has 1 row (omnara)
- [ ] `github_commits` has 3 commits with `raw_data->files[].patch` populated
- [ ] `github_issues` has 41 issues
- [ ] `github_pull_requests` has 16 PRs
- [ ] `github_issue_timeline` has events with proper structure
- [ ] `github_issue_pr_links` exists and has validated links
- [ ] `code_blocks` table exists (empty until Pipeline 2)
- [ ] `code_block_risk_index` table exists (empty until Pipeline 3)
- [ ] Neo4j graph built successfully (~105 nodes, ~50 edges)
- [ ] `crisk check <file>` returns risk analysis
- [ ] All foreign keys work (CASCADE deletes)
- [ ] Multi-repo safety verified (if second repo ingested)

---

## Timeline

| Step | Duration | Description |
|------|----------|-------------|
| 1. Backup | 30 sec | Optional pg_dump |
| 2. Schema rebuild | 10 sec | Drop + create tables |
| 3. Verify schema | 10 sec | Check table count |
| 4. Re-ingest omnara | ~5 min | GitHub API fetch with optimized rate |
| 5. Verify data | 30 sec | Query validation |
| 6. Verify patches | 10 sec | Check raw_data |
| 7. Test graph query | 5 sec | Run crisk check |
| 8. Multi-repo test | ~3 min | Optional supabase test |
| **Total** | **~10 min** | **End-to-end migration** |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Data loss | Low | Low | Backup created, GitHub API is source of truth |
| Schema errors | Very Low | Medium | rebuild_postgres_schema.sh has validation |
| GitHub API rate limit | Very Low | Low | Optimized fetcher (1.18 req/sec), only ~100 API calls |
| Graph build failure | Very Low | Medium | Test with test-graph-construction first |
| Rollback needed | Very Low | Low | Simple: re-run rebuild script + init |

**Overall Risk:** ✅ **LOW** - Safe to proceed

---

## Decision Matrix

| Criterion | Fresh Rebuild | In-Place Migration |
|-----------|--------------|-------------------|
| Time required | 10 minutes | 3 hours |
| Complexity | Simple | Complex |
| Risk of errors | Very low | High |
| Data preservation | Re-fetched | Preserved |
| Schema correctness | Guaranteed | Uncertain |
| Testing effort | Minimal | Extensive |
| **Recommendation** | ✅ **DO THIS** | ❌ **Avoid** |

---

## Execution Command

**One-line migration** (if you're confident):
```bash
cd /Users/rohankatakam/Documents/brain/coderisk && \
./scripts/rebuild_postgres_schema.sh --force && \
cd /tmp && rm -rf omnara && \
git clone https://github.com/omnara-ai/omnara && \
cd omnara && \
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365
```

**Or step-by-step** (recommended for first time):
```bash
# Step 1: Create backup directory
mkdir -p /Users/rohankatakam/Documents/brain/coderisk/backups

# Step 2: Backup current database (optional)
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" pg_dump \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -f /Users/rohankatakam/Documents/brain/coderisk/backups/omnara_backup_$(date +%Y%m%d_%H%M%S).sql

# Step 3: Rebuild schema
cd /Users/rohankatakam/Documents/brain/coderisk
./scripts/rebuild_postgres_schema.sh --force

# Step 4: Re-ingest omnara
cd /tmp && rm -rf omnara
git clone https://github.com/omnara-ai/omnara
cd omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365

# Step 5: Validate
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';"
# Should show: 20
```

---

## Next Steps After Migration

Once migration is complete and validated:

1. **Commit schema changes** to git
   ```bash
   cd /Users/rohankatakam/Documents/brain/coderisk
   git add schema/ scripts/ internal/database/staging.go internal/github/fetcher.go
   git commit -m "Apply optimized schema V2 with 16% faster GitHub API fetcher"
   git push
   ```

2. **Test multi-repo ingestion** (supabase, or another repo)

3. **Begin Pipeline 2 implementation** (code-block atomization with LLM)

4. **Begin Pipeline 3 implementation** (risk indexing calculations)

---

**Migration Status:** 📋 Ready for Execution
**Recommended Approach:** Fresh Schema Rebuild
**Estimated Time:** 10 minutes
**Risk Level:** ✅ Low
