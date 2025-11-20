# Migration 014: Coupling Metadata Columns - Complete Documentation

## Overview

Migration 014 implements the **Static vs Dynamic Property Separation** architecture for the `code_block_coupling` table, aligning with the architectural strategy documented in [microservice_arch.md](../../docs/microservice_arch.md) section 6.

## Purpose

Transform `code_block_coupling` from storing computed risk scores to storing only **static historical facts**, enabling dynamic risk computation at query time.

## Architecture Rationale

**Problem:** Storing dynamic properties (incident weights, recency multipliers, Top-K rankings) in the database creates operational complexity:
- Daily CRON jobs needed to update staleness/recency values
- Data becomes stale immediately after computation
- Top-K rankings change when new incidents are filed

**Solution:** Store ONLY static historical facts, compute dynamic properties at query time:
- **STATIC (Store):** `co_change_count`, `co_change_percentage`, `computed_at`, `window_start`, `window_end`
- **DYNAMIC (Compute):** Incident weights, recency multipliers, coupling scores, Top-K rankings

## Migration Files

### 1. `014_coupling_metadata_columns.sql` (Primary Migration)

**Purpose:** Add metadata columns to track coupling analysis window

**Changes:**
- Added `computed_at` - timestamp when coupling analysis was performed
- Added `window_start` / `window_end` - analysis time window (e.g., 12 months)
- Renamed `coupling_rate` → `co_change_percentage` for clarity (0.0-1.0 range)
- Dropped `total_changes` column (deprecated, can be computed from other fields)
- Backfilled window dates from existing `last_co_change` data
- Updated indexes to reflect new schema
- Added column comments documenting static vs dynamic separation

**Validation:**
- Counts total coupling edges
- Counts edges with complete metadata
- Warns if metadata coverage < 95%

### 2. `014_coupling_metadata_columns_completion.sql` (Completion Script)

**Purpose:** Fix remaining schema issues after primary migration was partially applied

**Created:** 2025-11-19 (after discovering primary migration had errors during backfill)

**Changes:**
- Renamed `last_co_changed_at` → `last_co_change` (spec alignment)
- Added `first_co_change` column (missing from primary migration)
- Dropped `co_change_rate` column (replaced by `co_change_percentage`)
- Dropped `reason` column (not in spec)
- Created index on `block_a_id` for fast lookups
- Validates all 12 required columns are present

**Required Columns (per DATA_SCHEMA_REFERENCE.md):**
1. `id` (BIGSERIAL PRIMARY KEY)
2. `repo_id` (BIGINT FK)
3. `block_a_id` (BIGINT FK)
4. `block_b_id` (BIGINT FK)
5. `co_change_count` (INTEGER - STATIC)
6. `co_change_percentage` (FLOAT - STATIC)
7. `first_co_change` (TIMESTAMP - STATIC)
8. `last_co_change` (TIMESTAMP - STATIC)
9. `computed_at` (TIMESTAMP - metadata)
10. `window_start` (TIMESTAMP - metadata)
11. `window_end` (TIMESTAMP - metadata)
12. `created_at` (TIMESTAMP - record timestamp)

## Backward Compatibility Fix

**Issue:** `coupling.go` code still referenced old column names (`co_change_rate`, `last_co_changed_at`)

**Solution:** Added generated column as compatibility alias:
```sql
ALTER TABLE code_block_coupling
ADD COLUMN IF NOT EXISTS co_change_rate DOUBLE PRECISION
GENERATED ALWAYS AS (co_change_percentage) STORED;
```

**Status:** This is a **temporary compatibility layer**. Future work should update `coupling.go` to use new column names and remove the generated column.

## Ultra-Strict Coupling Strategy

Migration 014 enables the ultra-strict coupling filtering strategy implemented in `crisk-index-coupling`:

**Filtering Criteria:**
- Co-change percentage ≥ 95% (blocks changed together 95%+ of the time)
- Minimum 10 absolute co-changes (statistical significance)
- BOTH blocks must have `incident_count ≥ 1` (only incident-prone blocks)
- 12-month rolling window (exclude stale relationships >365 days)

**Result:** Reduces coupling edges from O(n²) to O(n), typically ~1,500-2,000 edges for repos with 2,844 blocks (99.3% reduction from naive 230K edges).

## Query-Time Dynamic Computation

After migration 014, the MCP server computes dynamic coupling scores at query time:

```sql
SELECT
  cb2.block_name,
  edge.co_change_percentage,
  (cb1.incident_count + cb2.incident_count) as incident_weight,
  CASE
    WHEN GREATEST(cb1.last_incident_date, cb2.last_incident_date) > NOW() - INTERVAL '90 days' THEN 1.5
    WHEN GREATEST(cb1.last_incident_date, cb2.last_incident_date) > NOW() - INTERVAL '180 days' THEN 1.0
    ELSE 0.5
  END as recency_multiplier,
  edge.co_change_percentage * (cb1.incident_count + cb2.incident_count) *
  CASE ... END as coupling_score
FROM code_blocks cb1
JOIN code_block_coupling edge ON edge.block_a_id = cb1.id
JOIN code_blocks cb2 ON edge.block_b_id = cb2.id
WHERE cb1.id = $1 AND cb1.incident_count >= 1 AND cb2.incident_count >= 1
ORDER BY coupling_score DESC
LIMIT 3
```

## Running the Migration

**First Time (Fresh Database):**
```bash
psql -h localhost -p 5433 -U coderisk -d coderisk < migrations/014_coupling_metadata_columns.sql
psql -h localhost -p 5433 -U coderisk -d coderisk < migrations/014_coupling_metadata_columns_completion.sql
```

**Existing Database (Already Ran Primary Migration):**
```bash
# Only run completion script
psql -h localhost -p 5433 -U coderisk -d coderisk < migrations/014_coupling_metadata_columns_completion.sql
```

## Validation

After running both scripts, verify schema:

```sql
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'code_block_coupling'
ORDER BY ordinal_position;
```

Expected: 12 columns matching the required column list above.

## Related Documentation

- [microservice_arch.md](../../docs/microservice_arch.md) - Section 6: `crisk-index-coupling` architecture
- [DATA_SCHEMA_REFERENCE.md](../../docs/DATA_SCHEMA_REFERENCE.md) - Lines 405-474: `code_block_coupling` table spec
- [coupling.go](../internal/risk/coupling.go) - Implementation of ultra-strict filtering strategy

## Future Work

1. **Update coupling.go:** Refactor to use `co_change_percentage` instead of `co_change_rate` generated column
2. **Remove compatibility layer:** Drop `co_change_rate` generated column after code update
3. **Incremental coupling updates:** Add logic to incrementally update coupling edges when new commits arrive (currently full recomputation)
