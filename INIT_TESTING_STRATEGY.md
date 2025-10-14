# CodeRisk `init` Command - Testing Strategy

**Purpose:** Verify all 3 layers work end-to-end on omnara-ai/omnara
**Test Repo:** https://github.com/omnara-ai/omnara
**Expected Duration:** 15-20 minutes

---

## Pre-Test Setup (5 min)

### 1. Clean Environment
```bash
# Stop and remove all data
cd ~/Documents/brain/coderisk-go
docker compose down -v

# Start fresh infrastructure
docker compose up -d

# Wait for services to be ready (30 seconds)
sleep 30
```

### 2. Configure Credentials
```bash
# Run configuration wizard
crisk configure

# Enter when prompted:
# - OpenAI API key: sk-proj-...
# - GitHub token: ghp_...
# - Model: 1 (gpt-4o-mini)
# - Storage: 1 (OS Keychain)
```

### 3. Verify Prerequisites
```bash
# Check services are running
docker compose ps
# Should show: neo4j, postgres, redis all "Up"

# Check Neo4j accessible
curl -I http://localhost:7475
# Should return: HTTP/1.1 200 OK

# Check config saved
cat ~/.coderisk/config.yaml
# Should show: use_keychain: true
```

---

## Test Suite

### Test 1: Layer 1 - Structure (Tree-sitter) [CRITICAL]

**What it tests:** Code structure parsing and graph construction

**Commands:**
```bash
crisk init omnara-ai/omnara
```

**Expected Output:**
```
[0/4] Cloning and parsing repository...
  ✓ Repository cloned to /tmp/coderisk/omnara-ai/omnara
  ✓ Found 45 source files: TypeScript (32 files), Python (13 files)
  ✓ Parsed 45 files in 8s (156 functions, 42 classes, 89 imports)
  ✓ Graph construction complete: 287 entities stored
```

**Validation (Neo4j Browser: http://localhost:7475):**
```cypher
// 1. Count File nodes
MATCH (f:File) RETURN count(f) as file_count
// Expected: ~45 files

// 2. Count Function nodes
MATCH (fn:Function) RETURN count(fn) as func_count
// Expected: ~156 functions

// 3. Count Class nodes
MATCH (c:Class) RETURN count(c) as class_count
// Expected: ~42 classes

// 4. Verify IMPORTS relationships
MATCH (f1:File)-[r:IMPORTS]->(f2:File) RETURN count(r) as import_count
// Expected: >50 imports

// 5. Verify CALLS relationships
MATCH (fn1:Function)-[r:CALLS]->(fn2:Function) RETURN count(r) as call_count
// Expected: >100 function calls

// 6. Verify CONTAINS relationships
MATCH (f:File)-[r:CONTAINS]->(fn:Function) RETURN count(r) as contains_count
// Expected: ~156 (files contain functions)
```

**Success Criteria:**
- [ ] All source files discovered and counted
- [ ] Tree-sitter parsing completes without errors
- [ ] File, Function, Class nodes exist in Neo4j
- [ ] IMPORTS, CALLS, CONTAINS edges exist
- [ ] No "file failed to parse" warnings (or <5%)

---

### Test 2: Layer 2 - Temporal (GitHub API) [CRITICAL]

**What it tests:** Commit history, co-change patterns, developer tracking

**Expected Output (from same `crisk init` above):**
```
[1/4] Fetching GitHub API data...
  ✓ Fetched in 45s
    Commits: 234 | Issues: 18 | PRs: 42 | Branches: 8

[2/4] Building knowledge graph...
  ✓ Processed commits: 234 nodes, 468 edges
  ...
```

**Validation (Neo4j Browser):**
```cypher
// 1. Count Commit nodes
MATCH (c:Commit) RETURN count(c) as commit_count
// Expected: ~234 commits (90-day window)

// 2. Count Developer nodes
MATCH (d:Developer) RETURN count(d) as dev_count
// Expected: ~5-15 developers

// 3. Verify AUTHORED relationships
MATCH (d:Developer)-[r:AUTHORED]->(c:Commit) RETURN count(r) as authored_count
// Expected: ~234 (each commit has author)

// 4. Verify MODIFIES relationships
MATCH (c:Commit)-[r:MODIFIES]->(f:File) RETURN count(r) as modifies_count
// Expected: >500 (commits modify multiple files)

// 5. Verify CO_CHANGED edges (temporal coupling)
MATCH (f1:File)-[r:CO_CHANGED]-(f2:File)
RETURN count(r) as co_change_count,
       avg(r.frequency) as avg_frequency
// Expected: >20 co-change pairs, frequency 0.3-0.8

// 6. Find high co-change pairs
MATCH (f1:File)-[r:CO_CHANGED]-(f2:File)
WHERE r.frequency > 0.7
RETURN f1.path, f2.path, r.frequency
ORDER BY r.frequency DESC
LIMIT 5
// Should show files that often change together
```

**Success Criteria:**
- [ ] GitHub API fetch completes (may take 30-60s)
- [ ] Commit and Developer nodes exist
- [ ] AUTHORED and MODIFIES edges exist
- [ ] CO_CHANGED edges calculated with frequency scores
- [ ] No rate limit errors from GitHub API

---

### Test 3: Layer 3 - Incidents (GitHub Issues) [CRITICAL]

**What it tests:** Production incident tracking, issue analysis

**Expected Output (from same `crisk init` above):**
```
[2/4] Building knowledge graph...
  ...
  ✓ Processed issues: 18 nodes, 36 edges
  ✓ Processed PRs: 42 nodes, 84 edges
```

**Validation (Neo4j Browser):**
```cypher
// 1. Count Issue nodes
MATCH (i:Issue) RETURN count(i) as issue_count
// Expected: ~18 issues (90-day window)

// 2. Count PullRequest nodes
MATCH (pr:PullRequest) RETURN count(pr) as pr_count
// Expected: ~42 PRs

// 3. Find bug/incident issues
MATCH (i:Issue)
WHERE any(label IN i.labels WHERE label =~ '(?i)bug|incident|hotfix')
RETURN i.number, i.title, i.labels
LIMIT 10
// Should show issues labeled as bugs/incidents

// 4. Verify AFFECTS relationships (if incidents exist)
MATCH (i:Issue)-[r:AFFECTS]->(f:File) RETURN count(r) as affects_count
// Expected: >0 if incidents were linked

// 5. Find PR-commit relationships
MATCH (pr:PullRequest)-[r:INCLUDES]->(c:Commit) RETURN count(r) as pr_commit_count
// Expected: >50 (PRs include commits)
```

**Success Criteria:**
- [ ] Issue and PullRequest nodes exist
- [ ] Issues have labels property populated
- [ ] Bug/incident issues identifiable by labels
- [ ] PR-commit relationships exist
- [ ] Issue-file AFFECTS edges (if incidents manually linked)

---

### Test 4: Cross-Layer Integration [CRITICAL]

**What it tests:** Layers work together for risk analysis

**Validation (Neo4j Browser):**
```cypher
// 1. Find files with most changes (Layer 1 + Layer 2)
MATCH (c:Commit)-[:MODIFIES]->(f:File)
RETURN f.path, count(c) as change_count
ORDER BY change_count DESC
LIMIT 10
// Should show frequently modified files

// 2. Find developers who modified specific file (Layer 1 + Layer 2)
MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIES]->(f:File)
WHERE f.path CONTAINS 'auth'
RETURN d.name, count(c) as commits
ORDER BY commits DESC
LIMIT 5
// Should show ownership patterns

// 3. Find files with incidents + high coupling (Layer 1 + Layer 3)
MATCH (i:Issue)-[:AFFECTS]->(f:File)
MATCH (f)-[r:IMPORTS|CALLS]-(other:File)
RETURN f.path, i.title, count(other) as coupling
ORDER BY coupling DESC
// Should show risky files (incidents + dependencies)

// 4. Co-change + function calls (Layer 1 + Layer 2)
MATCH (f1:File)-[co:CO_CHANGED]-(f2:File)
MATCH (f1)-[:CONTAINS]->(fn1:Function)-[:CALLS]->(fn2:Function)<-[:CONTAINS]-(f2)
WHERE co.frequency > 0.5
RETURN f1.path, f2.path, co.frequency, fn1.name, fn2.name
LIMIT 5
// Should show structural + temporal coupling
```

**Success Criteria:**
- [ ] Can traverse from Layer 1 → Layer 2
- [ ] Can traverse from Layer 2 → Layer 3
- [ ] Can find ownership patterns
- [ ] Can identify high-risk files using multiple layers

---

### Test 5: `crisk check` Command [CRITICAL]

**What it tests:** End-to-end risk analysis using all 3 layers

**Commands:**
```bash
# Create a test change
cd /tmp/coderisk/omnara-ai/omnara
echo "// test change" >> src/core/auth.ts

# Run risk check
crisk check src/core/auth.ts
```

**Expected Output:**
```
🔍 Analyzing risk for src/core/auth.ts...

Phase 0: Pre-analysis
  ✓ Detected modification type: logic_change
  ✓ File type: TypeScript (security-critical pattern)

Phase 1: Baseline Assessment
  ✓ Structural coupling: 12 dependents (HIGH)
  ✓ Temporal co-change: 0.73 with user_service.ts (HIGH)
  ✓ Test coverage: 0.45 (MEDIUM)
  → Risk Level: HIGH (proceeding to Phase 2)

Phase 2: LLM Investigation
  ✓ Loading 1-hop neighbors...
  ✓ LLM analyzing evidence...
  ✓ Confidence: 0.92 (stopping early)

🎯 Risk Assessment: HIGH
   Confidence: 92%

Key Findings:
  • Authentication file with 12 direct dependents
  • Frequently changes with user_service.ts (73% co-change)
  • Modified by 3 developers in last 30 days
  • Similar to past incident #123 (auth timeout)

Recommendations:
  1. Review all dependent services before merge
  2. Ensure comprehensive auth tests exist
  3. Consider pair review with @alice (primary owner)

Evidence Chain (5 items):
  [1] High coupling (12 imports)
  [2] Temporal coupling with critical services
  [3] Recent ownership change (14 days ago)
  [4] Past incident similarity (score: 0.82)
  [5] Security-critical file pattern
```

**Success Criteria:**
- [ ] Check command runs without errors
- [ ] Uses all 3 layers (structure, temporal, incidents)
- [ ] Provides concrete evidence from graph
- [ ] LLM produces coherent risk assessment
- [ ] Recommendations are actionable

---

## Performance Benchmarks

| Stage | Expected Time | Max Acceptable |
|-------|---------------|----------------|
| Clean environment | 30s | 1 min |
| Configure | 30s | 1 min |
| Clone + parse (Layer 1) | 5-10s | 30s |
| GitHub fetch (Layer 2/3) | 30-60s | 2 min |
| Graph build | 10-20s | 1 min |
| Validation | 2-5s | 10s |
| **Total `crisk init`** | **1-2 min** | **5 min** |
| `crisk check` | 2-5s | 15s |

---

## Failure Scenarios

### Scenario 1: GitHub Rate Limit
**Symptom:** "403 rate limit exceeded"
**Fix:** Wait 1 hour or use different token
**Prevention:** Fetcher uses 1 req/sec (conservative)

### Scenario 2: Tree-sitter Parse Failures
**Symptom:** "X files failed to parse"
**Acceptable:** <5% failure rate (syntax errors, exotic code)
**Action:** Check logs, verify tree-sitter bindings

### Scenario 3: Neo4j Connection Failure
**Symptom:** "Neo4j connection failed"
**Fix:** `docker compose restart neo4j`
**Check:** `docker compose logs neo4j`

### Scenario 4: Missing Relationships
**Symptom:** Nodes exist but edges missing
**Cause:** Graph builder not connecting layers
**Action:** Check builder.go transform logic

---

## Quick Validation Script

```bash
#!/bin/bash
# File: test_init_omnara.sh

set -e

echo "🧪 CodeRisk Init Test - omnara-ai/omnara"
echo "=========================================="

# Clean start
echo "[1/5] Cleaning environment..."
docker compose down -v
docker compose up -d
sleep 30

# Run init
echo "[2/5] Running crisk init..."
crisk init omnara-ai/omnara

# Validate Layer 1
echo "[3/5] Validating Layer 1 (Structure)..."
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (f:File) RETURN count(f) as files" | grep -q "45"

# Validate Layer 2
echo "[4/5] Validating Layer 2 (Temporal)..."
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (c:Commit) RETURN count(c) as commits" | grep -q "234"

# Validate Layer 3
echo "[5/5] Validating Layer 3 (Incidents)..."
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (i:Issue) RETURN count(i) as issues" | grep -q "18"

echo "✅ All layers validated successfully!"
```

---

## Sign-Off Checklist

After running all tests, verify:

- [ ] All 3 layers have nodes in Neo4j
- [ ] All relationship types exist
- [ ] `crisk check` uses multi-layer analysis
- [ ] Performance within acceptable ranges
- [ ] No critical errors in logs
- [ ] Documentation matches actual behavior

**Tested By:** _____________
**Date:** _____________
**Version:** v0.1.0-beta.5
**Result:** PASS / FAIL
