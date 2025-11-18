#!/bin/bash
# Cleanup and Re-run Ingestion Pipeline for repo_id=11 (mcp-use)
# This script clears all ingestion data while preserving GitHub staging data,
# then re-runs the complete pipeline with fixed schema and file filtering.

set -e  # Exit on error

REPO_ID=11
REPO_NAME="mcp-use"

echo "=================================="
echo "CodeRisk Pipeline Cleanup & Re-run"
echo "Repository: $REPO_NAME (repo_id=$REPO_ID)"
echo "=================================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Verify we're in the right directory
if [ ! -f "./bin/crisk" ]; then
    echo -e "${RED}Error: Must run from coderisk root directory${NC}"
    exit 1
fi

echo -e "${YELLOW}=== Phase 1: Database Cleanup ===${NC}"
echo ""

# Step 2: Clear PostgreSQL ingestion data (preserve staging data)
echo "Clearing PostgreSQL ingestion data for repo_id=$REPO_ID..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
-- Clear atomizer outputs
DELETE FROM code_block_changes WHERE repo_id = $REPO_ID;
DELETE FROM code_block_imports WHERE repo_id = $REPO_ID;
DELETE FROM code_block_modifications WHERE repo_id = $REPO_ID;
DELETE FROM code_blocks WHERE repo_id = $REPO_ID;

-- Clear incident indexing outputs
DELETE FROM code_block_incidents WHERE repo_id = $REPO_ID;

-- Clear coupling indexing outputs
DELETE FROM code_block_coupling WHERE repo_id = $REPO_ID;

-- ✅ CRITICAL FIX: Reset processed_at to allow re-ingestion
UPDATE github_commits SET processed_at = NULL WHERE repo_id = $REPO_ID;
UPDATE github_pull_requests SET processed_at = NULL WHERE repo_id = $REPO_ID;
UPDATE github_issues SET processed_at = NULL WHERE repo_id = $REPO_ID;

-- Verify cleanup
SELECT
  (SELECT COUNT(*) FROM code_blocks WHERE repo_id = $REPO_ID) as blocks,
  (SELECT COUNT(*) FROM code_block_changes WHERE repo_id = $REPO_ID) as changes,
  (SELECT COUNT(*) FROM code_block_incidents WHERE repo_id = $REPO_ID) as incidents,
  (SELECT COUNT(*) FROM code_block_coupling WHERE repo_id = $REPO_ID) as coupling,
  (SELECT COUNT(*) FILTER (WHERE processed_at IS NULL) FROM github_commits WHERE repo_id = $REPO_ID) as unprocessed_commits;
EOF

echo -e "${GREEN}✓ PostgreSQL ingestion data cleared${NC}"
echo ""

# Step 3: Clear Neo4j ingestion data
echo "Clearing Neo4j ingestion data for repo_id=$REPO_ID..."
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d "{\"statements\":[{\"statement\":\"MATCH (n) WHERE n.repo_id = $REPO_ID DETACH DELETE n\"}]}" > /dev/null

echo -e "${GREEN}✓ Neo4j ingestion data cleared${NC}"
echo ""

# Step 4: Verify GitHub staging data is preserved
echo "Verifying GitHub staging data is preserved..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
SELECT
  'GitHub Staging Data (should be preserved):' as check_type;
SELECT
  (SELECT COUNT(*) FROM github_commits WHERE repo_id = $REPO_ID) as commits,
  (SELECT COUNT(*) FROM github_issues WHERE repo_id = $REPO_ID) as issues,
  (SELECT COUNT(*) FROM file_identity_map WHERE repo_id = $REPO_ID) as files;
EOF

echo -e "${GREEN}✓ GitHub staging data verified${NC}"
echo ""
echo ""

echo -e "${YELLOW}=== Phase 2: Pipeline Re-execution ===${NC}"
echo ""

# Step 5: Get repo path from database
echo "Fetching repository path..."
REPO_PATH=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "SELECT absolute_path FROM github_repositories WHERE id = $REPO_ID;" | xargs)

if [ -z "$REPO_PATH" ]; then
    echo -e "${RED}Error: Could not find repository path for repo_id=$REPO_ID${NC}"
    exit 1
fi

echo "Repository path: $REPO_PATH"
echo ""

# Step 6: Run crisk-ingest (populate Neo4j knowledge graph)
echo -e "${YELLOW}Step 1/5: Running crisk-ingest (populating Neo4j graph)...${NC}"
./bin/crisk-ingest --repo-id $REPO_ID

# Validate Neo4j population
echo "Validating Neo4j graph population..."
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (c:Commit {repo_id: 11}) RETURN count(c) as commits"},
    {"statement":"MATCH (d:Developer {repo_id: 11}) RETURN count(d) as developers"},
    {"statement":"MATCH (f:File {repo_id: 11}) RETURN count(f) as files"}
  ]}' | python3 -c "import sys, json; data = json.load(sys.stdin); print(f\"  Commits: {data['results'][0]['data'][0]['row'][0]}\"); print(f\"  Developers: {data['results'][1]['data'][0]['row'][0]}\"); print(f\"  Files: {data['results'][2]['data'][0]['row'][0]}\")"

echo -e "${GREEN}✓ crisk-ingest completed${NC}"
echo ""

# Step 7: Run crisk-atomize (extract code blocks with fixed filtering)
echo -e "${YELLOW}Step 2/5: Running crisk-atomize (extracting code blocks with file filtering)...${NC}"
GEMINI_API_KEY="AIzaSyCIDbK3lKQ2qq7KR8J1hLuXUhhsf6fT6YQ" ./bin/crisk-atomize --repo-id $REPO_ID --repo-path "$REPO_PATH"

# Validate atomizer output
echo "Validating atomizer output..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
SELECT
  COUNT(*) as total_blocks,
  COUNT(*) FILTER (WHERE block_name = '') as empty_names,
  COUNT(*) FILTER (WHERE start_line = 0 AND end_line = 0) as zero_lines
FROM code_blocks WHERE repo_id = $REPO_ID;
EOF

curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[
    {"statement":"MATCH (cb:CodeBlock {repo_id: 11}) RETURN count(cb) as blocks"},
    {"statement":"MATCH ()-[r:MODIFIED_BLOCK]->(:CodeBlock {repo_id: 11}) RETURN count(r) as edges"}
  ]}' | python3 -c "import sys, json; data = json.load(sys.stdin); print(f\"  CodeBlocks in Neo4j: {data['results'][0]['data'][0]['row'][0]}\"); print(f\"  MODIFIED_BLOCK edges: {data['results'][1]['data'][0]['row'][0]}\")"

echo -e "${GREEN}✓ crisk-atomize completed${NC}"
echo ""

# Step 8: Run crisk-index-incident (link incidents with new schema)
echo -e "${YELLOW}Step 3/5: Running crisk-index-incident (linking incidents with new schema)...${NC}"
./bin/crisk-index-incident --repo-id $REPO_ID

# Validate incident links
echo "Validating incident links..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
SELECT
  COUNT(*) as total_links,
  COUNT(block_id) as with_new_fk,
  COUNT(incident_date) as with_incident_date,
  COUNT(incident_type) as with_incident_type
FROM code_block_incidents WHERE repo_id = $REPO_ID;
EOF

echo -e "${GREEN}✓ crisk-index-incident completed${NC}"
echo ""

# Step 9: Run crisk-index-ownership (calculate ownership signals)
echo -e "${YELLOW}Step 4/5: Running crisk-index-ownership (calculating ownership signals)...${NC}"
./bin/crisk-index-ownership --repo-id $REPO_ID

# Validate ownership fields
echo "Validating ownership fields..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
SELECT
  COUNT(*) as total_blocks,
  COUNT(original_author_email) as with_author,
  COUNT(staleness_days) as with_staleness
FROM code_blocks WHERE repo_id = $REPO_ID;
EOF

echo -e "${GREEN}✓ crisk-index-ownership completed${NC}"
echo ""

# Step 10: Run crisk-index-coupling (calculate final risk scores)
echo -e "${YELLOW}Step 5/5: Running crisk-index-coupling (calculating risk scores)...${NC}"
./bin/crisk-index-coupling --repo-id $REPO_ID

# Validate risk scores
echo "Validating risk scores..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk <<EOF
SELECT
  COUNT(*) as total_blocks,
  COUNT(risk_score) as with_risk_score,
  ROUND(AVG(risk_score)::numeric, 2) as avg_risk,
  ROUND(MAX(risk_score)::numeric, 2) as max_risk
FROM code_blocks WHERE repo_id = $REPO_ID;
EOF

echo -e "${GREEN}✓ crisk-index-coupling completed${NC}"
echo ""
echo ""

echo -e "${GREEN}=================================="
echo "✓ Pipeline Re-execution Complete"
echo "=================================="
echo ""
echo "Summary:"
echo "  Repository: $REPO_NAME (repo_id=$REPO_ID)"
echo "  Microservices executed:"
echo "    1. crisk-ingest (Neo4j graph)"
echo "    2. crisk-atomize (code blocks + file filtering)"
echo "    3. crisk-index-incident (new schema)"
echo "    4. crisk-index-ownership (ownership signals)"
echo "    5. crisk-index-coupling (risk scores)"
echo ""
echo "Next steps:"
echo "  - Review output above for any warnings/errors"
echo "  - Check that all blocks have risk scores"
echo "  - Verify no empty block names or non-code files"
echo "  - Test MCP server queries"
echo -e "${NC}"
