# 🧪 Testing Instructions: CO_CHANGED Edge Fix

## Overview

This document provides step-by-step instructions to test the fix for CO_CHANGED and CAUSED_BY edge creation in the CodeRisk graph construction pipeline.

### What Was Fixed

1. **CO_CHANGED edges** - Path mismatch between git history (relative paths) and File nodes (absolute paths)
2. **CAUSED_BY edges** - Missing node ID prefixes causing incorrect label detection
3. **Error logging** - Enhanced diagnostics to surface silent edge creation failures

### Files Modified

- `internal/ingestion/processor.go` - Added path conversion for CO_CHANGED edges
- `internal/incidents/linker.go` - Fixed node ID format for CAUSED_BY edges
- `internal/graph/neo4j_backend.go` - Enhanced error logging and verification
- `Makefile` - Added `clean-docker` and improved `clean-all` targets
- `scripts/clean_docker.sh` - Docker cleanup script
- `scripts/validate_graph_edges.sh` - Graph validation script
- `scripts/e2e_clean_test.sh` - End-to-end test automation

---

## Prerequisites

Before testing, ensure you have:

- [x] Docker and Docker Compose installed
- [x] Go 1.21+ installed
- [x] Git installed
- [x] Environment variables set (`.env` file)
- [x] GitHub access token (if testing with private repos)

---

## Quick Test (Automated)

For a fully automated end-to-end test:

```bash
# This will:
# 1. Clean Docker completely
# 2. Rebuild crisk binary
# 3. Start fresh services
# 4. Clone Omnara repository
# 5. Run init-local
# 6. Validate all edge types

./scripts/e2e_clean_test.sh
```

**Expected output:**
- ✅ Docker cleanup successful
- ✅ Binary built successfully
- ✅ Services started
- ✅ Repository cloned (400+ files, 90+ commits)
- ✅ Graph construction completed
- ✅ All edge types validated (CONTAINS, IMPORTS, CO_CHANGED, CAUSED_BY)

**Success criteria:**
- `CO_CHANGED edge count: > 0` (not zero!)
- Sample edges show `frequency`, `co_changes`, `window_days` properties
- Bidirectional verification passes

---

## Manual Test (Step-by-Step)

If you prefer manual control or want to debug:

### Step 1: Clean Everything

```bash
# Option A: Use Makefile (recommended)
make clean-all

# Option B: Use script directly
./scripts/clean_docker.sh

# Verify cleanup
docker ps -a | grep coderisk  # Should return nothing
docker volume ls | grep coderisk  # Should return nothing
```

### Step 2: Build CodeRisk

```bash
# Build the binary
make build-cli

# Verify build
./bin/crisk --version
# Expected: "CodeRisk dev" with build info
```

### Step 3: Start Docker Services

```bash
# Start services
docker compose up -d

# Wait for services to initialize
sleep 15

# Check service health
docker compose ps
# Expected: All services "healthy" or "running"

# Test Neo4j connection
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 "RETURN 1"
# Expected: Returns "1"
```

### Step 4: Clone Test Repository

```bash
# Create test directory
mkdir -p /tmp/coderisk-test
cd /tmp/coderisk-test

# Clone Omnara
git clone https://github.com/omnara-ai/omnara
cd omnara

# Verify repository stats
git log --oneline --since="90 days ago" | wc -l
# Expected: >50 commits (enough for co-change analysis)

find . -name "*.ts" -o -name "*.tsx" -o -name "*.py" | wc -l
# Expected: >400 files
```

### Step 5: Run init-local

```bash
# Run from the test repository directory
/path/to/coderisk-go/bin/crisk init-local

# Or if you built in current directory
~/Documents/brain/coderisk-go/bin/crisk init-local
```

**Watch for these log messages:**

✅ Good signs:
```
INFO starting repository processing
INFO git history parsed commits=XXX window_days=90
INFO co-changes calculated total_pairs=XXX min_frequency=0.3
INFO converting co-change paths before_conversion=XXX
INFO sample co-change after conversion fileA=/full/path/... fileB=/full/path/...
INFO storing CO_CHANGED edges count=XXX
DEBUG: Creating XXX edges. First edge: CO_CHANGED (File:...) -> CO_CHANGED (File:...)
INFO edge verification passed count=XXX
```

❌ Bad signs:
```
WARN no commits found in git history
ERROR failed to store CO_CHANGED edges
WARN edge count mismatch expected=XXX actual=0
```

### Step 6: Validate Graph Construction

```bash
# Return to CodeRisk directory
cd ~/Documents/brain/coderisk-go

# Run validation script
./scripts/validate_graph_edges.sh
```

**Expected output:**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔍 CodeRisk Graph Edge Validation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 Layer 1: Code Structure (CONTAINS, IMPORTS)
  CONTAINS edges: 3014
  IMPORTS edges:  2089
  ✅ CONTAINS edges created successfully
  ✅ IMPORTS edges created successfully

⏱️  Layer 2: Temporal Analysis (CO_CHANGED)
  🔗 CO_CHANGED Edge Count: 336980
  ✅ CO_CHANGED edges created successfully!

  🔍 Sample CO_CHANGED edges:
  | a.name        | b.name        | frequency | co_changes | window_days |
  | page.tsx      | layout.tsx    | 0.85      | 17         | 90          |
  | auth.ts       | session.ts    | 0.92      | 23         | 90          |

  🔄 Bidirectional verification:
  | a.name        | b.name        | frequencies_match |
  | page.tsx      | layout.tsx    | true              |

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ SUCCESS: All edge types created correctly!
```

### Step 7: Test Risk Calculation (Optional)

```bash
cd /tmp/coderisk-test/omnara

# Test basic risk check
~/Documents/brain/coderisk-go/bin/crisk check apps/web/src/app/page.tsx

# Test AI mode (requires OpenAI API key)
~/Documents/brain/coderisk-go/bin/crisk check apps/web/src/app/page.tsx --ai-mode
```

### Step 8: Test Incident Linking (Optional)

```bash
# Create a test incident
~/Documents/brain/coderisk-go/bin/crisk incident create \
  "Auth timeout" \
  "Session timeout causing login failures" \
  --severity critical

# Link incident to file (use incident ID from output above)
~/Documents/brain/coderisk-go/bin/crisk incident link \
  <INCIDENT_ID> \
  "apps/web/src/auth/session.ts" \
  --line 42 \
  --function validateSession

# Verify CAUSED_BY edge
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (i:Incident)-[r:CAUSED_BY]->(f:File) RETURN i.title, f.name, r.confidence LIMIT 5"
```

---

## Troubleshooting

### Problem: No CO_CHANGED edges created

**Symptom:** Validation shows `CO_CHANGED Edge Count: 0`

**Diagnosis:**
```bash
# Check if git history exists
cd /tmp/coderisk-test/omnara
git log --oneline --since="90 days ago" | wc -l
# Should be >0

# Check init-local logs for errors
grep -i "co_change\|error" /tmp/coderisk-e2e-test/init-local.log
```

**Possible causes:**
1. Repository has no git history (not a git repo or shallow clone)
2. No commits in last 90 days
3. Path conversion failed (check DEBUG logs)

**Solution:**
```bash
# Re-run with debug logging
cd /tmp/coderisk-test/omnara
CODERISK_DEBUG=true ~/Documents/brain/coderisk-go/bin/crisk init-local 2>&1 | tee debug.log

# Look for path conversion
grep "sample co-change" debug.log
# Should show BEFORE (relative) and AFTER (absolute) paths
```

### Problem: CAUSED_BY edges not working

**Symptom:** Error when linking incident to file

**Diagnosis:**
```bash
# Check if File nodes exist
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (f:File {name: 'session.ts'}) RETURN f.file_path LIMIT 1"
```

**Solution:**
```bash
# Use exact file path from File node
# Get list of files:
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (f:File) RETURN f.file_path LIMIT 10"

# Use the exact file_path when linking
```

### Problem: Docker services won't start

**Symptom:** `docker compose up -d` fails

**Diagnosis:**
```bash
docker compose logs neo4j
docker compose logs postgres
```

**Solution:**
```bash
# Complete cleanup and restart
make clean-all
docker compose up -d
sleep 20  # Give more time for initialization
```

---

## Verification Queries

### Manual Neo4j Queries

Connect to Neo4j:
```bash
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
```

**Check File node structure:**
```cypher
MATCH (f:File)
RETURN f.file_path, f.name, f.language
LIMIT 3;
```

**Check CO_CHANGED edges:**
```cypher
MATCH (a:File)-[r:CO_CHANGED]->(b:File)
RETURN a.name, b.name, r.frequency, r.co_changes
ORDER BY r.frequency DESC
LIMIT 10;
```

**Check bidirectional CO_CHANGED:**
```cypher
MATCH (a:File)-[r1:CO_CHANGED]->(b:File),
      (b)-[r2:CO_CHANGED]->(a)
WHERE r1.frequency <> r2.frequency
RETURN count(*) as mismatched_pairs;
// Expected: 0
```

**Check CAUSED_BY edges:**
```cypher
MATCH (i:Incident)-[r:CAUSED_BY]->(f:File)
RETURN i.title, f.name, r.confidence, r.line_number
LIMIT 5;
```

---

## Success Criteria Checklist

After running tests, verify:

- [ ] Docker cleanup removed all containers and volumes
- [ ] Binary builds without errors
- [ ] Services start and are healthy
- [ ] Repository clones successfully (400+ files)
- [ ] `init-local` completes without errors
- [ ] File nodes created (count > 400)
- [ ] CONTAINS edges created (count > 3000)
- [ ] IMPORTS edges created (count > 2000)
- [ ] **CO_CHANGED edges created (count > 0)** ← **CRITICAL FIX**
- [ ] CO_CHANGED edges have properties (frequency, co_changes, window_days)
- [ ] CO_CHANGED edges are bidirectional (A→B implies B→A)
- [ ] Incident nodes can be created
- [ ] CAUSED_BY edges can be created
- [ ] `crisk check` command works
- [ ] AI mode works (if API key configured)

---

## Reporting Issues

If tests fail, please provide:

1. **Init-local logs:**
   ```bash
   cat /tmp/coderisk-e2e-test/init-local.log
   ```

2. **Validation output:**
   ```bash
   cat /tmp/coderisk-e2e-test/validation.log
   ```

3. **Debug logs:**
   ```bash
   grep "DEBUG\|ERROR\|WARN" /tmp/coderisk-e2e-test/init-local.log
   ```

4. **Neo4j query results:**
   ```cypher
   MATCH ()-[r:CO_CHANGED]->() RETURN count(r);
   MATCH (f:File) RETURN f.file_path LIMIT 3;
   ```

5. **Environment info:**
   ```bash
   docker --version
   go version
   uname -a
   ```

---

## Next Steps After Successful Test

Once all tests pass:

1. **Commit changes:**
   ```bash
   git add -A
   git commit -m "fix: Resolve CO_CHANGED and CAUSED_BY edge creation issues"
   ```

2. **Run pre-push validation:**
   ```bash
   make pre-push
   ```

3. **Test on production repository** (if available)

4. **Update documentation** with findings

5. **Create PR** with test results

---

**Last Updated:** $(date)
**Test Environment:** macOS/Linux with Docker
**Expected Duration:** 5-10 minutes (automated), 15-20 minutes (manual)
