#!/bin/bash
# Scenario 5: Structural Refactoring (Type 1)
# Tests multi-file refactoring with import updates

set -e

SCENARIO_NAME="Scenario 5: Structural Refactoring (Type 1)"
TEST_DIR="test_sandbox/omnara"
FILES=(
    "src/backend/auth/utils.py"
    "src/backend/auth/routes.py"
)
OUTPUT_FILE="test/integration/modification_type_tests/output_scenario_5.txt"

echo "========================================"
echo "$SCENARIO_NAME"
echo "========================================"
echo ""

# 1. Find crisk binary
if [ -f "./bin/crisk" ]; then
    CRISK_BIN="./bin/crisk"
elif [ -f "./crisk" ]; then
    CRISK_BIN="./crisk"
else
    echo "ERROR: crisk binary not found"
    echo "Expected at: ./bin/crisk or ./crisk"
    echo "Please run: make build"
    exit 1
fi
echo "Using binary: $CRISK_BIN"

# 2. Navigate to test directory
cd "$TEST_DIR" || exit 1

# 3. Ensure clean git state
echo "Checking git status..."
if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: Git working directory not clean"
    git status --short
    exit 1
fi
echo "✅ Git working directory clean"
echo ""

# 4. Make structural changes
echo "Making structural refactoring changes..."

# Change 1: Comment out function in utils.py (simulate move)
FILE1="${FILES[0]}"
echo "  Modifying $FILE1..."
sed -i.bak '/def update_user_profile/i\
# TESTING: Function moved to jwt_utils.py
' "$FILE1"

# Change 2: Update import in routes.py
FILE2="${FILES[1]}"
echo "  Modifying $FILE2..."
sed -i.bak 's/from .utils import update_user_profile/# from .utils import update_user_profile  # TESTING: Now in jwt_utils/' "$FILE2"

echo "✅ Changes applied successfully"
echo ""

# 5. Verify changes
echo "Verifying git changes..."
git diff --stat
echo ""
echo "Files changed:"
for file in "${FILES[@]}"; do
    if git diff --quiet "$file"; then
        echo "  ⚠️  $file: No changes detected"
    else
        echo "  ✅ $file: Modified"
    fi
done
echo ""

# 6. Run crisk check on all affected files
echo "Running crisk check on affected files..."
cd ../..  # Back to coderisk-go root

FILE_PATHS=""
for file in "${FILES[@]}"; do
    FILE_PATHS="$FILE_PATHS $TEST_DIR/$file"
done

"$CRISK_BIN" check $FILE_PATHS > "$OUTPUT_FILE" 2>&1 || true

echo "✅ crisk check completed"
echo ""

# 7. Display output
echo "========================================"
echo "ACTUAL OUTPUT:"
echo "========================================"
cat "$OUTPUT_FILE"
echo ""

# 8. Validation
echo "========================================"
echo "VALIDATION CHECKS:"
echo "========================================"

if grep -q "Risk Level:" "$OUTPUT_FILE"; then
    RISK_LEVEL=$(grep "Risk Level:" "$OUTPUT_FILE" | head -1)
    echo "✅ Risk level found: $RISK_LEVEL"

    if echo "$RISK_LEVEL" | grep -qi "HIGH"; then
        echo "✅ HIGH risk detected (expected for multi-file refactoring)"
    elif echo "$RISK_LEVEL" | grep -qi "MEDIUM"; then
        echo "ℹ️  MEDIUM risk detected (acceptable for 2-file change)"
    else
        echo "⚠️  Expected HIGH or MEDIUM risk for structural refactoring"
    fi
else
    echo "❌ Risk level missing"
fi

if grep -qi "structural\|refactor\|import\|dependency\|type 1" "$OUTPUT_FILE"; then
    echo "✅ Structural change indicators found"
else
    echo "ℹ️  Structural type not explicitly mentioned"
fi

if grep -q "files" "$OUTPUT_FILE" | grep -qE "[2-9]|[0-9]{2,}"; then
    echo "✅ Multi-file impact detected"
else
    echo "ℹ️  Multi-file context not emphasized"
fi

echo ""

# 9. Reset changes
echo "Resetting changes..."
cd "$TEST_DIR"
for file in "${FILES[@]}"; do
    git restore "$file"
    rm -f "${file}.bak"
done

if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  Git not fully restored"
    git status --short
else
    echo "✅ Git restored to clean state"
fi
echo ""

echo "========================================"
echo "✅ Test completed"
echo "Output saved to: $OUTPUT_FILE"
echo "========================================"
echo ""
