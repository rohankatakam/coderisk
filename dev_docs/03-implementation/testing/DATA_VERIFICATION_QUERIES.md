# Data Verification Queries: Neo4j & PostgreSQL

**Purpose:** Complete reference of validation queries for verifying CodeRisk data storage
**Last Updated:** October 9, 2025
**Audience:** Developers, QA engineers, Integration testers
**Context:** Use during Phase 1 testing after `crisk init-local` completes

> **Cross-reference:** Part of the integration testing strategy outlined in [INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md)

---

## The Big Picture

After running `crisk init-local`, you need to verify that all three layers of the knowledge graph are correctly stored in Neo4j and PostgreSQL. This document provides ready-to-run queries organized by layer and use case.

**What gets stored where:**
- **Neo4j**: Graph structure (nodes and edges for all 3 layers)
- **PostgreSQL**: Incident metadata and BM25 full-text search

---

## Quick Start

### Prerequisites

1. **Neo4j running:** `docker ps | grep neo4j` shows healthy
2. **PostgreSQL running:** `docker ps | grep postgres` shows healthy
3. **Data loaded:** `crisk init-local` completed successfully

### Query Execution

**Neo4j queries:**
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "YOUR_CYPHER_QUERY_HERE"
```

**PostgreSQL queries:**
```bash
docker exec coderisk-postgres psql -U coderisk -d coderisk -c "YOUR_SQL_QUERY_HERE"
```

---

## Layer 1: Code Structure Validation (Tree-sitter)

### Overview

Layer 1 contains entities extracted from source code via Tree-sitter parsing:
- **File nodes:** Source files (TypeScript, Python, JavaScript, Go)
- **Function nodes:** Function/method definitions
- **Class nodes:** Class/interface definitions
- **Import nodes:** Import statements
- **CONTAINS edges:** File → Function/Class relationships
- **IMPORTS edges:** File → File import dependencies

### 1.1 Node Count Verification

**Check total entity counts:**
```cypher
// Get counts for all Layer 1 node types
MATCH (f:File) WITH count(f) as files
MATCH (fn:Function) WITH files, count(fn) as functions
MATCH (c:Class) WITH files, functions, count(c) as classes
MATCH (i:Import) WITH files, functions, classes, count(i) as imports
RETURN
  files,
  functions,
  classes,
  imports,
  files + functions + classes + imports as total_entities
```

**Expected output (omnara repository):**
```
files     | functions | classes | imports | total_entities
421-422   | ~2,563    | ~454    | ~2,089  | ~5,527
```

**Pass criteria:** ✅ Counts within ±1% of expected values

---

### 1.2 File Node Structure

**Check File node properties:**
```cypher
MATCH (f:File)
RETURN
  f.file_path as path,
  f.name as filename,
  f.language as language,
  f.unique_id as unique_id
LIMIT 3
```

**Expected properties:**
- `file_path`: Absolute path (e.g., `/Users/.../apps/web/src/app/page.tsx`)
- `name`: Filename only (e.g., `page.tsx`)
- `language`: Language type (e.g., `typescript`, `python`, `javascript`)
- `unique_id`: Same as `file_path` (used for matching)

**Pass criteria:** ✅ All 4 properties present on every File node

---

### 1.3 Function Node Structure

**Check Function node properties:**
```cypher
MATCH (fn:Function)
RETURN
  fn.name as function_name,
  fn.file_path as file,
  fn.line_number as line,
  fn.unique_id as unique_id
LIMIT 5
```

**Expected properties:**
- `name`: Function name (e.g., `Home`, `processData`)
- `file_path`: File containing the function
- `line_number`: Starting line number
- `unique_id`: Format: `filepath:name:line` (e.g., `/path/to/file.ts:Home:10`)

**Pass criteria:** ✅ All Function nodes have name, file_path, line_number, unique_id

---

### 1.4 CONTAINS Relationship Verification

**Count CONTAINS edges (File → Function/Class):**
```cypher
MATCH ()-[r:CONTAINS]->()
RETURN count(r) as contains_edges
```

**Expected:** >3,000 edges (functions + classes)

**Verify CONTAINS relationship structure:**
```cypher
MATCH (f:File)-[r:CONTAINS]->(fn:Function)
RETURN
  f.name as file,
  fn.name as function,
  fn.line_number as line
LIMIT 5
```

**Pass criteria:**
- ✅ Edge count matches (functions + classes) count
- ✅ Every Function/Class has exactly 1 incoming CONTAINS edge from a File

---

### 1.5 IMPORTS Relationship Verification

**Count IMPORTS edges (File → File):**
```cypher
MATCH ()-[r:IMPORTS]->()
RETURN count(r) as import_edges
```

**Expected:** ~2,089 edges (one per import statement)

**Verify IMPORTS relationship structure:**
```cypher
MATCH (from:File)-[r:IMPORTS]->(to:File)
RETURN
  from.name as importing_file,
  to.name as imported_file
LIMIT 10
```

**Example:**
```
importing_file     | imported_file
page.tsx           | layout.tsx
api/route.ts       | lib/db.ts
```

**Pass criteria:**
- ✅ IMPORTS edges connect File nodes to other File nodes
- ✅ Import count matches expected value

---

### 1.6 Language Distribution

**Check file distribution by language:**
```cypher
MATCH (f:File)
RETURN
  f.language as language,
  count(f) as file_count
ORDER BY file_count DESC
```

**Expected output (omnara):**
```
language    | file_count
typescript  | 286
python      | 129
javascript  | 6
```

**Pass criteria:** ✅ Matches repository composition

---

## Layer 2: Temporal Analysis Validation (Git History)

### Overview

Layer 2 contains co-change patterns from git history analysis:
- **CO_CHANGED edges:** Bidirectional edges between files that change together
- **Properties:** `frequency` (0.0-1.0), `co_changes` (count), `window_days` (90)

### 2.1 CO_CHANGED Edge Count

**Count all CO_CHANGED edges:**
```cypher
MATCH ()-[r:CO_CHANGED]->()
RETURN count(r) as co_change_edges
```

**Expected:** >0 (typically tens of thousands for active repositories)

**Note:** The exact count depends on git history depth and co-change threshold (0.3 frequency minimum)

**Pass criteria:** ✅ Count > 0 (edges were created)

---

### 2.2 CO_CHANGED Edge Properties

**Verify edge properties are set:**
```cypher
MATCH ()-[r:CO_CHANGED]->()
RETURN
  r.frequency as frequency,
  r.co_changes as co_changes,
  r.window_days as window_days
LIMIT 5
```

**Expected properties:**
- `frequency`: Float between 0.3 and 1.0
- `co_changes`: Integer > 0 (number of times files changed together)
- `window_days`: 90 (analysis window)

**Pass criteria:**
- ✅ All CO_CHANGED edges have these 3 properties
- ✅ `frequency` >= 0.3 (minimum threshold)
- ✅ `window_days` = 90

---

### 2.3 High-Frequency Co-Changes

**Find files with strongest co-change patterns:**
```cypher
MATCH (a:File)-[r:CO_CHANGED]-(b:File)
WHERE r.frequency >= 0.7
RETURN
  a.name as file_a,
  b.name as file_b,
  r.frequency as frequency,
  r.co_changes as times_changed_together
ORDER BY r.frequency DESC
LIMIT 10
```

**Example output:**
```
file_a              | file_b              | frequency | times_changed_together
auth/session.ts     | auth/route.ts       | 0.95      | 19
layout.tsx          | page.tsx            | 0.87      | 13
schema.ts           | migrations/001.sql  | 0.82      | 12
```

**Pass criteria:**
- ✅ Returned pairs make logical sense (related files)
- ✅ High frequency files are actually related in the codebase

---

### 2.4 Co-Change for Specific File

**Check co-changes for a specific file (e.g., auth route):**
```cypher
MATCH (f:File)-[r:CO_CHANGED]-(other:File)
WHERE f.name CONTAINS 'auth' OR f.file_path CONTAINS 'auth'
RETURN
  f.name as auth_file,
  other.name as co_changed_with,
  r.frequency as frequency
ORDER BY r.frequency DESC
LIMIT 10
```

**Pass criteria:**
- ✅ Auth-related files co-change with session/middleware files
- ✅ Frequencies reflect actual development patterns

---

### 2.5 Bidirectional Edge Verification

**Verify CO_CHANGED edges are bidirectional:**
```cypher
MATCH (a:File)-[r1:CO_CHANGED]->(b:File)
MATCH (b)-[r2:CO_CHANGED]->(a)
WHERE a.name = 'page.tsx' AND b.name = 'layout.tsx'
RETURN
  a.name as file_a,
  b.name as file_b,
  r1.frequency as a_to_b_frequency,
  r2.frequency as b_to_a_frequency
```

**Expected:** Both edges exist with identical frequency values

**Pass criteria:**
- ✅ For every A → B edge, B → A edge exists
- ✅ Frequencies are identical in both directions

---

## Layer 3: Incident Data Validation (PostgreSQL + Neo4j)

### Overview

Layer 3 contains production incident data:
- **PostgreSQL:** Incident metadata, BM25 full-text search
- **Neo4j:** Incident nodes + CAUSED_BY edges to Files

### 3.1 PostgreSQL: Incident Table Schema

**Verify incidents table exists:**
```sql
SELECT
  column_name,
  data_type,
  is_nullable
FROM information_schema.columns
WHERE table_name = 'incidents'
ORDER BY ordinal_position;
```

**Expected columns:**
```
column_name    | data_type                | is_nullable
id             | uuid                     | NO
title          | text                     | NO
description    | text                     | YES
severity       | text                     | YES
occurred_at    | timestamp with time zone | YES
resolved_at    | timestamp with time zone | YES
root_cause     | text                     | YES
impact         | text                     | YES
created_at     | timestamp with time zone | YES
updated_at     | timestamp with time zone | YES
search_vector  | tsvector                 | YES
```

**Pass criteria:** ✅ All 11 columns present with correct data types

---

### 3.2 PostgreSQL: Incident Count

**Count total incidents:**
```sql
SELECT COUNT(*) as total_incidents
FROM incidents;
```

**Expected:** Matches number of `crisk incident create` commands run

**Pass criteria:** ✅ Count >= 0 (no errors)

---

### 3.3 PostgreSQL: BM25 Full-Text Search

**Test BM25 search functionality:**
```sql
SELECT
  id,
  title,
  severity,
  ts_rank(search_vector, to_tsquery('english', 'auth | session')) as rank
FROM incidents
WHERE search_vector @@ to_tsquery('english', 'auth | session')
ORDER BY rank DESC
LIMIT 5;
```

**Expected:** Returns incidents containing "auth" or "session" ranked by relevance

**Pass criteria:**
- ✅ Query executes without errors
- ✅ Results are ranked (higher rank = better match)
- ✅ Matches contain search terms in title/description

---

### 3.4 PostgreSQL: Incident Links Table

**Verify incident_links table:**
```sql
SELECT
  il.id,
  i.title as incident_title,
  il.file_path,
  il.line_number,
  il.function_name,
  il.confidence
FROM incident_links il
JOIN incidents i ON il.incident_id = i.id
LIMIT 5;
```

**Expected columns:**
- `incident_id`: UUID foreign key
- `file_path`: File linked to incident
- `line_number`: Optional line number
- `function_name`: Optional function name
- `confidence`: Confidence score (0.0-1.0)

**Pass criteria:** ✅ Links exist and foreign keys are valid

---

### 3.5 Neo4j: Incident Nodes

**Count Incident nodes in Neo4j:**
```cypher
MATCH (i:Incident)
RETURN count(i) as incident_count
```

**Expected:** Matches PostgreSQL incident count

**Verify Incident node properties:**
```cypher
MATCH (i:Incident)
RETURN
  i.id as incident_id,
  i.title as title,
  i.severity as severity
LIMIT 5
```

**Pass criteria:**
- ✅ Incident node count matches PostgreSQL
- ✅ Nodes have id, title, severity properties

---

### 3.6 Neo4j: CAUSED_BY Edges

**Count CAUSED_BY edges (Incident → File):**
```cypher
MATCH ()-[r:CAUSED_BY]->()
RETURN count(r) as caused_by_edges
```

**Expected:** Matches number of `crisk incident link` commands

**Verify CAUSED_BY relationship structure:**
```cypher
MATCH (i:Incident)-[r:CAUSED_BY]->(f:File)
RETURN
  i.title as incident,
  f.name as file,
  r.confidence as confidence,
  r.line_number as line
LIMIT 5
```

**Expected properties:**
- `confidence`: 0.0-1.0 (typically 1.0 for manual links)
- `line_number`: Optional integer
- `blamed_function`: Optional function name

**Pass criteria:**
- ✅ Edge count matches incident links
- ✅ Edges connect Incident nodes to File nodes
- ✅ Properties are populated

---

### 3.7 Cross-Database Consistency Check

**Verify PostgreSQL and Neo4j have same incident data:**

**PostgreSQL:**
```sql
SELECT id, title, severity
FROM incidents
ORDER BY created_at DESC
LIMIT 5;
```

**Neo4j:**
```cypher
MATCH (i:Incident)
RETURN i.id, i.title, i.severity
LIMIT 5
```

**Pass criteria:** ✅ Same incidents appear in both databases with identical properties

---

## Performance Validation

### 4.1 Phase 1 Query Performance (<200ms target)

**Test coupling query (1-hop structural):**
```cypher
PROFILE
MATCH (f:File {name: 'page.tsx'})<-[:IMPORTS]-(other:File)
RETURN count(other) as dependents
```

**Expected:** Query completes in <20ms (dbHits shown in PROFILE output)

**Test co-change query:**
```cypher
PROFILE
MATCH (f:File {name: 'page.tsx'})-[r:CO_CHANGED]-(other:File)
WHERE r.frequency >= 0.5
RETURN other.name, r.frequency
ORDER BY r.frequency DESC
LIMIT 10
```

**Expected:** Query completes in <20ms

**Pass criteria:** ✅ Both queries complete in <50ms total

---

### 4.2 Incident Search Performance (<50ms target)

**Test BM25 search with EXPLAIN:**
```sql
EXPLAIN ANALYZE
SELECT
  id,
  title,
  ts_rank(search_vector, to_tsquery('english', 'timeout')) as rank
FROM incidents
WHERE search_vector @@ to_tsquery('english', 'timeout')
ORDER BY rank DESC
LIMIT 10;
```

**Expected:** Execution time <50ms (shown in EXPLAIN ANALYZE output)

**Pass criteria:** ✅ Search completes in <50ms with index usage

---

## Data Integrity Checks

### 5.1 Orphaned Nodes

**Check for Functions without parent File:**
```cypher
MATCH (fn:Function)
WHERE NOT EXISTS((fn)<-[:CONTAINS]-())
RETURN count(fn) as orphaned_functions
```

**Expected:** 0 orphaned functions

**Check for Classes without parent File:**
```cypher
MATCH (c:Class)
WHERE NOT EXISTS((c)<-[:CONTAINS]-())
RETURN count(c) as orphaned_classes
```

**Expected:** 0 orphaned classes

**Pass criteria:** ✅ All Functions and Classes have exactly 1 parent File

---

### 5.2 Duplicate Nodes

**Check for duplicate File nodes:**
```cypher
MATCH (f:File)
WITH f.file_path as path, count(f) as duplicates
WHERE duplicates > 1
RETURN path, duplicates
```

**Expected:** No duplicates

**Pass criteria:** ✅ Every file_path appears exactly once

---

### 5.3 Missing Properties

**Check for File nodes missing required properties:**
```cypher
MATCH (f:File)
WHERE f.file_path IS NULL
   OR f.name IS NULL
   OR f.language IS NULL
RETURN count(f) as files_with_missing_props
```

**Expected:** 0 files with missing properties

**Pass criteria:** ✅ All File nodes have file_path, name, language

---

## Complete Test Suite Script

### Quick Validation Script

Save this as `test/integration/validate_graph_data.sh`:

```bash
#!/bin/bash
# Complete data validation for CodeRisk graph construction

set -e

NEO4J_CMD="docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
PG_CMD="docker exec coderisk-postgres psql -U coderisk -d coderisk"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "CodeRisk Data Verification Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Layer 1: Code Structure
echo -e "\n📁 Layer 1: Code Structure"
$NEO4J_CMD "MATCH (f:File) RETURN count(f) as files"
$NEO4J_CMD "MATCH ()-[r:CONTAINS]->() RETURN count(r) as contains_edges"
$NEO4J_CMD "MATCH ()-[r:IMPORTS]->() RETURN count(r) as import_edges"

# Layer 2: Temporal Analysis
echo -e "\n⏱️  Layer 2: Temporal Analysis"
$NEO4J_CMD "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as co_change_edges"

# Layer 3: Incidents
echo -e "\n🚨 Layer 3: Incidents"
$NEO4J_CMD "MATCH (i:Incident) RETURN count(i) as incident_nodes"
$NEO4J_CMD "MATCH ()-[r:CAUSED_BY]->() RETURN count(r) as caused_by_edges"
$PG_CMD -c "SELECT COUNT(*) as pg_incidents FROM incidents;"

# Data Integrity
echo -e "\n✅ Data Integrity Checks"
$NEO4J_CMD "MATCH (fn:Function) WHERE NOT EXISTS((fn)<-[:CONTAINS]-()) RETURN count(fn) as orphaned"
$NEO4J_CMD "MATCH (f:File) WITH f.file_path as path, count(f) as dups WHERE dups > 1 RETURN count(*) as duplicates"

echo -e "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Validation complete!"
```

**Usage:**
```bash
chmod +x test/integration/validate_graph_data.sh
./test/integration/validate_graph_data.sh
```

---

## Troubleshooting

### No CO_CHANGED Edges (count = 0)

**Symptoms:** Layer 2 query returns 0 edges

**Possible causes:**
1. ❌ **Field mismatch:** Code using `path` instead of `file_path` for File node matching
2. ❌ **Transaction not committed:** Edges created but not persisted
3. ❌ **Git history too shallow:** Repository has <2 commits

**Diagnosis:**
```cypher
// Check if File nodes have 'path' or 'file_path' property
MATCH (f:File) RETURN keys(f) LIMIT 1
```

**Fix:** Update `internal/graph/neo4j_backend.go` to use correct property name

---

### Incident Nodes in Neo4j but no CAUSED_BY Edges

**Symptoms:** Incident count > 0, but CAUSED_BY edge count = 0

**Possible causes:**
1. ❌ **File path mismatch:** Incident linked to path not in graph
2. ❌ **Label case sensitivity:** Looking for `file:` but stored as `File:`

**Diagnosis:**
```cypher
// Check Incident node structure
MATCH (i:Incident) RETURN i LIMIT 1

// Check if File exists with the linked path
MATCH (i:Incident)
// Manually check if linked file exists in graph
RETURN i.linked_file_path
```

**Fix:** Ensure incident link uses exact file_path from File nodes

---

### Slow Query Performance

**Symptoms:** Queries taking >200ms

**Possible causes:**
1. ❌ **Missing indexes:** File nodes not indexed on file_path
2. ❌ **Large result sets:** Returning too many nodes without LIMIT

**Diagnosis:**
```cypher
// Check indexes
SHOW INDEXES
```

**Fix:** Ensure indexes exist on frequently-queried properties

---

## See Also

- **Integration Test Strategy:** [INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md)
- **E2E Test Results:** [E2E_TEST_SUMMARY.md](E2E_TEST_SUMMARY.md)
- **Graph Schema:** [../../01-architecture/graph_ontology.md](../../01-architecture/graph_ontology.md)

---

**Last Updated:** October 9, 2025
**Status:** Ready for use in Phase 1 testing
**Owner:** QA + Engineering team
