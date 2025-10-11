#!/bin/bash
# Scenario 7: Security-Sensitive Change (Type 9)
# Tests security keyword detection and forced Phase 2 escalation

set -e  # Exit on error

SCENARIO_NAME="Scenario 7: Security-Sensitive Change (Type 9)"
TEST_DIR="test_sandbox/omnara"
TARGET_FILE="src/backend/auth/routes.py"
OUTPUT_FILE="test/integration/modification_type_tests/output_scenario_7.txt"

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

# 4. Make controlled changes - Add TODO comment with security context
echo "Making security-sensitive changes to $TARGET_FILE..."
CHANGE_MARKER="    # TODO: Add session timeout validation here"

# Find the sync_user function and add our comment after the docstring
sed -i.bak '/"""Sync user from Supabase to our database"""/a\
'"$CHANGE_MARKER"'
' "$TARGET_FILE"

# Verify the change was made
if ! grep -q "$CHANGE_MARKER" "$TARGET_FILE"; then
    echo "ERROR: Failed to apply changes to $TARGET_FILE"
    mv "$TARGET_FILE.bak" "$TARGET_FILE"  # Restore backup
    exit 1
fi

echo "✅ Changes applied successfully"
echo ""

# 5. Verify changes with git
echo "Verifying git diff..."
git diff --stat "$TARGET_FILE"
echo ""
echo "Git diff preview:"
git diff "$TARGET_FILE" | head -20
echo ""

# 6. Run crisk check
echo "Running crisk check..."
cd ../..  # Back to coderisk-go root

# Run crisk check and capture output (allow non-zero exit for HIGH/CRITICAL risk)
"$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true

echo "✅ crisk check completed"
echo ""

# 7. Display output
echo "========================================"
echo "ACTUAL OUTPUT:"
echo "========================================"
cat "$OUTPUT_FILE"
echo ""

# 8. Check for key indicators
echo "========================================"
echo "VALIDATION CHECKS:"
echo "========================================"

# Check for security detection
if grep -qi "security\|auth\|critical" "$OUTPUT_FILE"; then
    echo "✅ Security context detected"
else
    echo "⚠️  Security context not explicitly mentioned"
fi

# Check for risk level
if grep -q "Risk Level:" "$OUTPUT_FILE"; then
    RISK_LEVEL=$(grep "Risk Level:" "$OUTPUT_FILE" | head -1)
    echo "✅ Risk level found: $RISK_LEVEL"
else
    echo "❌ Risk level missing from output"
fi

# Check for modification type (if Phase 0 implemented)
if grep -q "Modification Type:" "$OUTPUT_FILE"; then
    MOD_TYPE=$(grep "Modification Type:" "$OUTPUT_FILE" | head -1)
    echo "✅ Modification type found: $MOD_TYPE"
else
    echo "ℹ️  Modification type not in output (Phase 0 not yet implemented)"
fi

# Check for HIGH/CRITICAL risk
if grep -qi "HIGH\|CRITICAL" "$OUTPUT_FILE"; then
    echo "✅ HIGH or CRITICAL risk level detected (expected for security changes)"
else
    echo "⚠️  Expected HIGH or CRITICAL risk, but not found"
fi

echo ""

# 9. Reset changes
echo "Resetting changes..."
cd "$TEST_DIR"
git restore "$TARGET_FILE"
rm -f "$TARGET_FILE.bak"  # Remove sed backup

# 10. Verify clean state
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
echo "========================================"
echo ""
