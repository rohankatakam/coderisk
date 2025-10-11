#!/bin/bash
# Scenario 10: Documentation-Only Change (Type 6)
# Tests fast-path logic for zero-runtime-impact changes

set -e  # Exit on error

SCENARIO_NAME="Scenario 10: Documentation-Only Change (Type 6)"
TEST_DIR="test_sandbox/omnara"
TARGET_FILE="README.md"
OUTPUT_FILE="test/integration/modification_type_tests/output_scenario_10.txt"

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
    echo "ERROR: Git working directory not clean. Please commit or stash changes."
    git status --short
    exit 1
fi
echo "✅ Git working directory clean"
echo ""

# 4. Make documentation-only changes
echo "Making documentation changes to $TARGET_FILE..."

# Add a new development setup section
cat >> "$TARGET_FILE" << 'EOF'

## Development Setup (Testing)

1. Install dependencies: `pip install -r requirements-dev.txt`
2. Set up environment: Copy `.env.example` to `.env`
3. Run tests: `pytest tests/`
4. Start development server: `./dev-start.sh`

This section added for testing documentation-only changes.
EOF

echo "✅ Changes applied successfully"
echo ""

# 5. Verify changes with git
echo "Verifying git diff..."
git diff --stat "$TARGET_FILE"
echo ""
echo "Git diff preview:"
git diff "$TARGET_FILE" | tail -15
echo ""

# 6. Measure start time
START_TIME=$(date +%s%3N)  # milliseconds

# 7. Run crisk check
echo "Running crisk check..."
cd ../..  # Back to coderisk-go root

# Run crisk check and capture output
"$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true

# Calculate execution time
END_TIME=$(date +%s%3N)
DURATION=$((END_TIME - START_TIME))

echo "✅ crisk check completed in ${DURATION}ms"
echo ""

# 8. Display output
echo "========================================"
echo "ACTUAL OUTPUT:"
echo "========================================"
cat "$OUTPUT_FILE"
echo ""

# 9. Validation checks
echo "========================================"
echo "VALIDATION CHECKS:"
echo "========================================"

# Check for LOW risk (expected for docs-only)
if grep -q "Risk Level:" "$OUTPUT_FILE"; then
    RISK_LEVEL=$(grep "Risk Level:" "$OUTPUT_FILE" | head -1)
    echo "✅ Risk level found: $RISK_LEVEL"

    if echo "$RISK_LEVEL" | grep -qi "LOW"; then
        echo "✅ LOW risk detected (expected for documentation-only)"
    else
        echo "⚠️  Expected LOW risk for documentation-only change"
    fi
else
    echo "❌ Risk level missing from output"
fi

# Check for documentation type detection (if Phase 0 implemented)
if grep -qi "documentation.*only\|type 6\|docs.*only" "$OUTPUT_FILE"; then
    echo "✅ Documentation-only change detected"
else
    echo "ℹ️  Documentation-only detection not in output (Phase 0 not yet implemented)"
fi

# Check for fast execution (should be <50ms with Phase 0 skip logic)
if [ "$DURATION" -lt 50 ]; then
    echo "✅ Very fast execution (<50ms) - possible Phase 0 skip logic"
elif [ "$DURATION" -lt 200 ]; then
    echo "ℹ️  Fast execution (<200ms) - normal Phase 1 performance"
else
    echo "⚠️  Slower than expected ($DURATION ms)"
fi

# Check that Phase 2 was not triggered
if grep -qi "phase 2\|escalating\|investigation" "$OUTPUT_FILE"; then
    echo "⚠️  Phase 2 triggered for documentation-only (unexpected)"
else
    echo "✅ Phase 2 not triggered (correct for documentation-only)"
fi

echo ""

# 10. Reset changes
echo "Resetting changes..."
cd "$TEST_DIR"
git restore "$TARGET_FILE"

# 11. Verify clean state
if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  WARNING: Git working directory not fully restored"
    git status --short
else
    echo "✅ Git working directory restored to clean state"
fi
echo ""

echo "========================================"
echo "✅ Test completed"
echo "Output saved to: $OUTPUT_FILE"
echo "Execution time: ${DURATION}ms"
echo "========================================"
echo ""
