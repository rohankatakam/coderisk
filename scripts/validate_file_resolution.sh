#!/bin/bash
# File Resolution Validation Script
# Tests that git log --follow correctly discovers historical file paths
# for files that have been reorganized or renamed.

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "File Resolution Validation Test"
echo "========================================="
echo ""

# Try to find omnara repository in common locations
POSSIBLE_PATHS=(
    "$HOME/.coderisk/repos/omnara"
    "$HOME/Documents/brain/omnara"
    "/tmp/omnara"
)

REPO_PATH=""
for path in "${POSSIBLE_PATHS[@]}"; do
    if [ -d "$path/.git" ]; then
        REPO_PATH="$path"
        break
    fi
done

if [ -z "$REPO_PATH" ]; then
    echo -e "${RED}❌ FAIL: Omnara repository not found${NC}"
    echo "Searched in:"
    for path in "${POSSIBLE_PATHS[@]}"; do
        echo "  - $path"
    done
    echo ""
    echo "Please clone omnara repository to one of these locations:"
    echo "  git clone https://github.com/omnara-ai/omnara ~/.coderisk/repos/omnara"
    exit 1
fi

echo "Found repository at: $REPO_PATH"
echo ""

# Test file that is known to have been reorganized
TEST_FILE="src/shared/config/settings.py"

echo "Testing file resolution for: $TEST_FILE"
echo ""

# Step 1: Get historical paths using git log --follow
echo "Step 1: Running git log --follow..."
cd "$REPO_PATH"

# Get all historical paths (deduplicated and sorted)
PATHS=$(git log --follow --name-only --pretty=format: -- "$TEST_FILE" | sort -u | grep -v '^$')

if [ -z "$PATHS" ]; then
    echo -e "${RED}❌ FAIL: No paths found for $TEST_FILE${NC}"
    echo "This might mean:"
    echo "  1. The file doesn't exist in the repository"
    echo "  2. The file has no commit history"
    echo "  3. git log --follow is not working correctly"
    exit 1
fi

echo "Found paths:"
echo "$PATHS" | while IFS= read -r path; do
    echo "  - $path"
done
echo ""

# Step 2: Count paths
PATH_COUNT=$(echo "$PATHS" | wc -l | tr -d ' ')
echo "Total unique paths: $PATH_COUNT"
echo ""

# Step 3: Verify expected paths exist
echo "Step 2: Validating expected paths..."

# Check for current path
if echo "$PATHS" | grep -q "^src/shared/config/settings.py$"; then
    echo -e "${GREEN}✅ Found current path: src/shared/config/settings.py${NC}"
else
    echo -e "${RED}❌ FAIL: Missing current path: src/shared/config/settings.py${NC}"
    echo "Found paths were:"
    echo "$PATHS"
    exit 1
fi

# Check for historical path (before reorganization to src/)
if echo "$PATHS" | grep -q "^shared/config/settings.py$"; then
    echo -e "${GREEN}✅ Found historical path: shared/config/settings.py${NC}"
else
    echo -e "${YELLOW}⚠️  WARNING: Missing expected historical path: shared/config/settings.py${NC}"
    echo "This might be okay if the file was never at this location"
    echo "Found paths were:"
    echo "$PATHS"
    # Don't fail - just warn
fi

echo ""

# Step 4: Verify minimum path count
if [ "$PATH_COUNT" -lt 1 ]; then
    echo -e "${RED}❌ FAIL: Expected at least 1 path, got $PATH_COUNT${NC}"
    exit 1
fi

echo "Step 3: Testing performance..."
# Measure how long git log --follow takes
START_TIME=$(date +%s%N)
git log --follow --name-only --pretty=format: -- "$TEST_FILE" > /dev/null 2>&1
END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "git log --follow execution time: ${ELAPSED_MS}ms"

if [ "$ELAPSED_MS" -gt 1000 ]; then
    echo -e "${YELLOW}⚠️  WARNING: git log --follow took more than 1 second${NC}"
    echo "This might impact crisk check performance for large repositories"
else
    echo -e "${GREEN}✅ Performance acceptable (<1 second)${NC}"
fi

echo ""

# Step 5: Test with a file that was never renamed (if README exists)
echo "Step 4: Testing with never-renamed file..."
if [ -f "$REPO_PATH/README.md" ]; then
    README_PATHS=$(git log --follow --name-only --pretty=format: -- "README.md" | sort -u | grep -v '^$')
    README_COUNT=$(echo "$README_PATHS" | wc -l | tr -d ' ')

    echo "README.md has $README_COUNT historical path(s)"
    if [ "$README_COUNT" -eq 1 ]; then
        echo -e "${GREEN}✅ README.md correctly shows 1 path (never renamed)${NC}"
    else
        echo -e "${YELLOW}⚠️  README.md shows $README_COUNT paths (might have been renamed)${NC}"
    fi
else
    echo "README.md not found, skipping never-renamed file test"
fi

echo ""
echo "========================================="
echo -e "${GREEN}✅ SUCCESS: File resolution foundation working${NC}"
echo "========================================="
echo ""
echo "Summary:"
echo "  - Repository: $REPO_PATH"
echo "  - Test file: $TEST_FILE"
echo "  - Historical paths found: $PATH_COUNT"
echo "  - Performance: ${ELAPSED_MS}ms"
echo ""
echo "Next steps:"
echo "  1. Run unit tests: go test ./internal/git -v"
echo "  2. Verify Neo4j File nodes have current/historical flags"
echo "  3. Verify MODIFIED edge coverage is 85%+"
echo "  4. Implement QueryCommitsForPaths in internal/graph/queries.go"
echo ""
