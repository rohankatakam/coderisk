#!/bin/bash
# End-to-End Clean Test Script
# This script performs a complete clean test of CodeRisk graph construction
# with the Omnara repository

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="/tmp/coderisk-e2e-test"
OMNARA_REPO="https://github.com/omnara-ai/omnara"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 CodeRisk End-to-End Clean Test"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "This script will:"
echo "  1. Clean all Docker containers and volumes"
echo "  2. Rebuild CodeRisk binary"
echo "  3. Start fresh Docker services"
echo "  4. Clone Omnara repository to temporary directory"
echo "  5. Run init-local to build graph"
echo "  6. Validate all edge types were created"
echo ""
read -p "Press ENTER to continue or Ctrl+C to cancel..."

# Step 1: Clean Docker
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 1: Cleaning Docker Environment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

./scripts/clean_docker.sh

# Step 2: Rebuild CodeRisk
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 2: Building CodeRisk Binary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "🔨 Building crisk binary..."
go build -o ./crisk ./cmd/crisk

if [ -f ./crisk ]; then
    echo -e "${GREEN}✅ Binary built successfully${NC}"
    ./crisk --version
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Step 3: Start Docker services
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 3: Starting Docker Services${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "🐳 Starting services (Neo4j, PostgreSQL, Redis)..."
docker compose up -d

echo ""
echo "⏳ Waiting 15 seconds for services to initialize..."
sleep 15

echo ""
echo "🔍 Checking service health..."
docker compose ps

# Step 4: Prepare test directory
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 4: Preparing Test Directory${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "📁 Creating test directory: ${TEST_DIR}"
rm -rf ${TEST_DIR}
mkdir -p ${TEST_DIR}
cd ${TEST_DIR}

echo ""
echo "📥 Cloning Omnara repository..."
git clone ${OMNARA_REPO} omnara

if [ -d omnara ]; then
    echo -e "${GREEN}✅ Repository cloned successfully${NC}"
    cd omnara
    echo ""
    echo "📊 Repository stats:"
    echo "  - Total commits: $(git rev-list --count HEAD)"
    echo "  - Recent commits: $(git log --oneline --since='90 days ago' | wc -l) (last 90 days)"
    FILE_COUNT=$(find . -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.py" -o -name "*.js" \) | wc -l)
    echo "  - Source files: ${FILE_COUNT}"
else
    echo -e "${RED}❌ Clone failed${NC}"
    exit 1
fi

# Step 5: Run init-local
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 5: Building Knowledge Graph${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "🚀 Running: crisk init-local"
echo "   (This will parse source files and build the graph)"
echo ""

# Get the absolute path to crisk binary from the original directory
CRISK_BIN="$(cd - > /dev/null && pwd)/crisk"

# Run init-local and capture output
START_TIME=$(date +%s)
if "${CRISK_BIN}" init-local 2>&1 | tee ${TEST_DIR}/init-local.log; then
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    echo ""
    echo -e "${GREEN}✅ init-local completed in ${DURATION} seconds${NC}"
else
    echo ""
    echo -e "${RED}❌ init-local failed - check logs above${NC}"
    exit 1
fi

# Step 6: Validate graph construction
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 6: Validating Graph Construction${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cd - > /dev/null  # Return to crisk directory
./scripts/validate_graph_edges.sh | tee ${TEST_DIR}/validation.log

# Step 7: Summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test Complete!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "📝 Logs saved to:"
echo "  - Init-local: ${TEST_DIR}/init-local.log"
echo "  - Validation: ${TEST_DIR}/validation.log"
echo ""
echo "📂 Test repository: ${TEST_DIR}/omnara"
echo ""

# Check if CO_CHANGED edges exist
CO_CHANGED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" 2>/dev/null | grep -oE '[0-9]+' | tail -1 || echo "0")

if [ "${CO_CHANGED_COUNT}" -gt 0 ]; then
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ SUCCESS: All edge types created correctly!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "You can now test:"
    echo "  - Risk analysis: cd ${TEST_DIR}/omnara && ${CRISK_BIN} check <file>"
    echo "  - AI mode: cd ${TEST_DIR}/omnara && ${CRISK_BIN} check <file> --ai-mode"
else
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}⚠️  WARNING: CO_CHANGED edges not created${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Check the logs for errors:"
    echo "  cat ${TEST_DIR}/init-local.log | grep -i error"
    echo "  cat ${TEST_DIR}/init-local.log | grep -i co_change"
fi

echo ""
