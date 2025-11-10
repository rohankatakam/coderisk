# Fresh Init & Testing Guide

**Purpose**: Clean database setup → Full ingestion → Comprehensive testing

---

## Prerequisites

- CodeRisk built (`make build` completed)
- Docker services running (`make start`)
- GEMINI_API_KEY set in environment
- GITHUB_TOKEN set in environment

---

## Step 1: Clean All Databases

**Goal**: Start with completely fresh state

```bash
# Stop all services
cd /Users/rohankatakam/Documents/brain/coderisk
make stop

# Remove all data volumes (DESTRUCTIVE - deletes all ingested data)
docker volume rm coderisk_neo4j_data || true
docker volume rm coderisk_postgres_data || true
docker volume rm coderisk_redis_data || true

# Restart services with clean volumes
make start

# Wait for services to be ready (30-60 seconds)
sleep 60
```

**Verify databases are empty**:
```bash
# Neo4j should have 0 nodes
curl -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (n) RETURN count(n) as count"}]}' | grep -o '"count":[0-9]*'

# PostgreSQL should have empty tables
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM github_commits;"
```

Expected: Both should return 0

---

## Step 2: Clone Fresh Omnara Repo

**Goal**: Clean test repository with no artifacts

```bash
# Remove any existing clone
rm -rf /tmp/omnara

# Clone fresh
cd /tmp
git clone https://github.com/omnara-ai/omnara.git
cd omnara

# Verify we're on main branch with latest code
git branch
git log --oneline -5
```

---

## Step 3: Run Fresh crisk init

**Goal**: Full ingestion of omnara-ai/omnara repository

```bash
# Set required environment variables
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
export GITHUB_TOKEN="YOUR_GITHUB_TOKEN_HERE"
export PHASE2_ENABLED="true"
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

# Navigate to omnara repo
cd /tmp/omnara

# Run init (this will take 10-30 minutes for omnara)
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 180

# Monitor progress
# You should see:
# - Git graph construction
# - GitHub API fetching (issues, PRs, commits)
# - Timeline event processing
# - Issue-PR linking
# - DORA metrics computation
```

**Expected output checkpoints**:
1. ✓ Git repository detected
2. ✓ Connecting to Neo4j
3. ✓ Connecting to PostgreSQL
4. ✓ Fetching GitHub data...
5. ✓ Processing commits...
6. ✓ Linking issues to PRs...
7. ✓ Computing DORA metrics...
8. ✓ Ingestion complete

---

## Step 4: Verify Ingestion Success

**Goal**: Confirm data is properly loaded

### Check Neo4j Graph
```bash
# Count nodes by type
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (n) RETURN labels(n)[0] as type, count(*) as count ORDER BY count DESC"}]}' | \
  python3 -c "import sys, json; data=json.load(sys.stdin); [print(f\"{row['row'][0]}: {row['row'][1]}\") for row in data['results'][0]['data']]"
```

**Expected counts** (approximate):
- Commit: 1000-3000
- File: 500-2000
- Developer: 20-50
- PR: 100-300
- Issue: 100-300

### Check PostgreSQL Tables
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "
    SELECT 'commits' as table, COUNT(*) FROM github_commits UNION ALL
    SELECT 'issues', COUNT(*) FROM github_issues UNION ALL
    SELECT 'prs', COUNT(*) FROM github_pull_requests UNION ALL
    SELECT 'timeline', COUNT(*) FROM github_issue_timeline;
  "
```

**Expected**: All tables should have data

### Check DORA Metrics
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -t -c "SELECT sample_size, ROUND(median_lead_time_hours::numeric, 2) FROM github_dora_metrics WHERE repo_id = 1 ORDER BY computed_at DESC LIMIT 1;"
```

**Expected**: `sample_size > 0` (should show actual PR count, not 0)

---

## Step 5: Run Test Suite

**Goal**: Execute all test cases from [TEST_CASES.md](TEST_CASES.md)

### Setup Test Environment
```bash
# Ensure we're in omnara repo
cd /tmp/omnara

# Set environment (if not already set)
export GEMINI_API_KEY="AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU"
export PHASE2_ENABLED="true"
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

# Create results directory
mkdir -p /tmp/test_results
```

### Test Execution Script

Create `/tmp/run_all_tests.sh`:
```bash
#!/bin/bash
cd /tmp/omnara

RESULTS_DIR="/tmp/test_results"
CRISK="/Users/rohankatakam/Documents/brain/coderisk/bin/crisk"

# Test 1: Configuration Files
echo "=== TEST 1.1: pyproject.toml ===" | tee $RESULTS_DIR/test_1.1.txt
$CRISK check pyproject.toml --explain 2>&1 | tee -a $RESULTS_DIR/test_1.1.txt

echo "=== TEST 1.2: package.json ===" | tee $RESULTS_DIR/test_1.2.txt
$CRISK check apps/web/package.json --explain 2>&1 | tee -a $RESULTS_DIR/test_1.2.txt

echo "=== TEST 1.3: .env.example ===" | tee $RESULTS_DIR/test_1.3.txt
$CRISK check .env.example --explain 2>&1 | tee -a $RESULTS_DIR/test_1.3.txt

# Test 2: Core Logic
echo "=== TEST 2.1: auth.ts ===" | tee $RESULTS_DIR/test_2.1.txt
$CRISK check apps/web/src/lib/auth.ts --explain 2>&1 | tee -a $RESULTS_DIR/test_2.1.txt

echo "=== TEST 2.2: schema.prisma ===" | tee $RESULTS_DIR/test_2.2.txt
$CRISK check packages/database/prisma/schema.prisma --explain 2>&1 | tee -a $RESULTS_DIR/test_2.2.txt

# Test 3: UI Components
echo "=== TEST 3.1: Dashboard Layout ===" | tee $RESULTS_DIR/test_3.1.txt
$CRISK check apps/web/src/components/dashboard/SidebarDashboardLayout.tsx --explain 2>&1 | tee -a $RESULTS_DIR/test_3.1.txt

echo "=== TEST 3.2: Chat Message ===" | tee $RESULTS_DIR/test_3.2.txt
$CRISK check apps/web/src/components/dashboard/chat/ChatMessage.tsx --explain 2>&1 | tee -a $RESULTS_DIR/test_3.2.txt

# Test 4: Infrastructure
echo "=== TEST 4.1: docker-compose.yml ===" | tee $RESULTS_DIR/test_4.1.txt
$CRISK check docker-compose.yml --explain 2>&1 | tee -a $RESULTS_DIR/test_4.1.txt

# Test 5: Edge Cases
echo "=== TEST 5.1: README (Low Risk) ===" | tee $RESULTS_DIR/test_5.1.txt
$CRISK check README.md 2>&1 | tee -a $RESULTS_DIR/test_5.1.txt

# Test 6: Batch Test
echo "=== TEST 6: Multi-file batch ===" | tee $RESULTS_DIR/test_6.txt
$CRISK check pyproject.toml apps/web/package.json apps/web/src/lib/auth.ts --explain 2>&1 | tee -a $RESULTS_DIR/test_6.txt

echo ""
echo "========================================"
echo "All tests complete!"
echo "Results saved to: $RESULTS_DIR"
echo "========================================"
```

Run tests:
```bash
chmod +x /tmp/run_all_tests.sh
/tmp/run_all_tests.sh
```

---

## Step 6: Analyze Results

### Automated Checks
```bash
# Check for errors across all tests
grep -i "error\|failed\|panic" /tmp/test_results/*.txt

# Verify Phase 2 ran
grep -i "phase 2\|hop\|llm" /tmp/test_results/*.txt | head -20

# Check risk levels
grep "Risk Level:" /tmp/test_results/*.txt

# Verify no OpenAI errors
grep -i "openai" /tmp/test_results/*.txt
```

### Manual Review Checklist

For each test result file in `/tmp/test_results/`:

1. **Risk Level**: Appropriate for file type
2. **Confidence**: >60% for complex files
3. **Agent Hops**: 3-5 hops completed
4. **Incidents**: Sample incidents displayed
5. **Co-change**: Partner files identified
6. **Ownership**: Developers listed
7. **Response Time**: <30s for Phase 2
8. **No Errors**: Clean execution

---

## Step 7: Database Validation Queries

After tests complete, verify data integrity:

### Neo4j Validation
```bash
# Check for orphaned nodes (nodes with no relationships)
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (n) WHERE NOT (n)--() RETURN labels(n)[0] as type, count(*) as count"}]}'

# Verify commit->file relationships
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (c:Commit)-[:MODIFIED]->(f:File) RETURN count(*) as relationship_count"}]}'
```

### PostgreSQL Validation
```bash
# Check for NULL values in critical columns
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT 'commits_with_null_sha' as check, COUNT(*) FROM github_commits WHERE sha IS NULL;"

# Verify issue-PR links
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM github_issue_timeline WHERE event_type = 'cross-referenced';"
```

---

## Troubleshooting

### Issue: `crisk init` hangs or times out
**Solution**: Check GitHub API rate limits
```bash
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/rate_limit
```

### Issue: Neo4j queries return no data
**Solution**: Verify Neo4j is accessible
```bash
curl -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 http://localhost:7475
```

### Issue: DORA metrics show sample_size=0
**Solution**: Check that issue-pr-linker ran successfully during init
```bash
# Should see cross-referenced events
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql \
  -h localhost -p 5433 -U coderisk -d coderisk \
  -c "SELECT COUNT(*) FROM github_issue_timeline;"
```

### Issue: Phase 2 not running
**Solution**: Verify environment variables
```bash
echo "PHASE2_ENABLED: $PHASE2_ENABLED"
echo "GEMINI_API_KEY: ${GEMINI_API_KEY:0:20}..."
```

---

## Success Metrics

After completing this guide, you should have:

- ✅ Clean databases with fresh omnara data
- ✅ Neo4j graph with 1000+ commits, 500+ files
- ✅ PostgreSQL with issues, PRs, timeline events
- ✅ DORA metrics computed (sample_size > 0)
- ✅ All test cases passing
- ✅ Phase 2 agent running (5 hops, >60% confidence)
- ✅ No Gemini/OpenAI errors
- ✅ Actionable risk assessments for all file types

---

## Next Steps

1. Review test results in `/tmp/test_results/`
2. Compare outputs against expected results in [TEST_CASES.md](TEST_CASES.md)
3. Document any anomalies or unexpected behaviors
4. Run additional targeted tests for edge cases
5. Validate DORA metrics correlation with actual PR data

---

## Quick Reference Commands

```bash
# Re-run a single test
cd /tmp/omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check <file> --explain

# Check database stats
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 "MATCH (n) RETURN count(n);"

# View recent logs
docker logs coderisk-neo4j --tail 50
docker logs coderisk-postgres --tail 50

# Reset and start over
make stop && docker volume prune -f && make start
```
