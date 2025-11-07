#!/bin/bash
# Quick setup and test script for issue-PR linking system

set -e

echo "========================================="
echo "Issue-PR Linking System - Setup & Test"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check environment variables
echo "Step 1: Checking environment variables..."
if [ -z "$OPENAI_API_KEY" ]; then
    echo -e "${RED}❌ OPENAI_API_KEY not set${NC}"
    echo "   Set it with: export OPENAI_API_KEY=\"your-key\""
    exit 1
fi

if [ -z "$POSTGRES_PASSWORD" ]; then
    echo -e "${YELLOW}⚠️  POSTGRES_PASSWORD not set, using default${NC}"
    export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
fi

echo -e "${GREEN}✓ Environment variables OK${NC}"
echo ""

# Step 2: Build binaries
echo "Step 2: Building binaries..."
cd "$(dirname "$0")/.."

if [ ! -d "bin" ]; then
    mkdir bin
fi

echo "  Building issue-pr-linker..."
cd cmd/issue-pr-linker
go build -o ../../bin/issue-pr-linker .
cd ../..

echo "  Building test-linker..."
cd cmd/test-linker
go build -o ../../bin/test-linker .
cd ../..

echo -e "${GREEN}✓ Binaries built${NC}"
echo ""

# Step 3: Check database connection
echo "Step 3: Checking database connection..."
psql -h localhost -p 5433 -U coderisk_user -d coderisk -c "SELECT 1;" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Cannot connect to PostgreSQL${NC}"
    echo "   Make sure Docker containers are running"
    exit 1
fi

echo -e "${GREEN}✓ Database connection OK${NC}"
echo ""

# Step 4: Create schema (if not exists)
echo "Step 4: Creating database schema..."
psql -h localhost -p 5433 -U coderisk_user -d coderisk \
    -f scripts/schema/linking_tables.sql 2>&1 | grep -v "already exists" || true

echo -e "${GREEN}✓ Schema ready${NC}"
echo ""

# Step 5: Check if omnara data exists
echo "Step 5: Checking for omnara repository data..."
ISSUE_COUNT=$(psql -h localhost -p 5433 -U coderisk_user -d coderisk -t -c \
    "SELECT COUNT(*) FROM github_issues WHERE repo_id = (SELECT id FROM github_repositories WHERE full_name = 'omnara-ai/omnara');" 2>/dev/null | xargs)

if [ "$ISSUE_COUNT" = "0" ] || [ -z "$ISSUE_COUNT" ]; then
    echo -e "${RED}❌ No omnara data found${NC}"
    echo "   Run 'crisk init' first to ingest omnara repository"
    exit 1
fi

echo -e "${GREEN}✓ Found $ISSUE_COUNT issues in omnara repository${NC}"
echo ""

# Step 6: Run the linker
echo "Step 6: Running issue-PR linker..."
echo "========================================="
./bin/issue-pr-linker --repo omnara-ai/omnara --days 90
LINKER_EXIT=$?

if [ $LINKER_EXIT -ne 0 ]; then
    echo ""
    echo -e "${RED}❌ Linker failed with exit code $LINKER_EXIT${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Linking complete${NC}"
echo ""

# Step 7: Run test validation
echo "Step 7: Running test validation..."
echo "========================================="
./bin/test-linker \
    --repo omnara-ai/omnara \
    --ground-truth test_data/omnara_ground_truth_expanded.json \
    --output omnara_test_report.txt

TEST_EXIT=$?

echo ""
if [ $TEST_EXIT -eq 0 ]; then
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}✅ ALL TESTS PASSED!${NC}"
    echo -e "${GREEN}=========================================${NC}"
else
    echo -e "${RED}=========================================${NC}"
    echo -e "${RED}❌ Tests failed${NC}"
    echo -e "${RED}=========================================${NC}"
fi

echo ""
echo "Test report saved to: omnara_test_report.txt"
echo ""
echo "To view results:"
echo "  cat omnara_test_report.txt"
echo ""
echo "To query database:"
echo "  psql -h localhost -p 5433 -U coderisk_user -d coderisk"
echo "  SELECT * FROM github_issue_pr_links WHERE repo_id = (SELECT id FROM github_repositories WHERE full_name = 'omnara-ai/omnara');"

exit $TEST_EXIT
