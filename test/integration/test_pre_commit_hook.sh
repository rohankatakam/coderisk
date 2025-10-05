#!/bin/bash
# Integration test for pre-commit hook
# Reference: ux_pre_commit_hook.md - Testing strategy

set -e  # Exit on error

echo "=== Pre-commit Hook Integration Test ==="

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Build crisk binary
echo "Building crisk binary..."
go build -o ./crisk ./cmd/crisk
if [ ! -f "./crisk" ]; then
    echo "❌ FAIL: Failed to build crisk binary"
    exit 1
fi
echo "✅ Binary built successfully"

# Setup test repo in temp directory
TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"

cleanup() {
    echo "Cleaning up..."
    cd "$PROJECT_ROOT"
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

cd "$TEST_DIR"
git init
git config user.email "test@example.com"
git config user.name "Test User"

echo "✅ Test repository initialized"

# Copy crisk binary to test directory
cp "$PROJECT_ROOT/crisk" ./crisk
export PATH="$TEST_DIR:$PATH"

# Test 1: Install hook
echo ""
echo "Test 1: Installing hook..."
./crisk hook install

if [ ! -f .git/hooks/pre-commit ]; then
    echo "❌ FAIL: Hook not installed"
    exit 1
fi

if [ ! -x .git/hooks/pre-commit ]; then
    echo "❌ FAIL: Hook is not executable"
    exit 1
fi

echo "✅ PASS: Hook installed and executable"

# Test 2: Hook exists - should fail to install again
echo ""
echo "Test 2: Attempting to install hook again (should fail)..."
if ./crisk hook install 2>&1 | grep -q "already exists"; then
    echo "✅ PASS: Correctly prevented duplicate installation"
else
    echo "❌ FAIL: Should have prevented duplicate installation"
    exit 1
fi

# Test 3: Uninstall hook
echo ""
echo "Test 3: Uninstalling hook..."
./crisk hook uninstall

if [ -f .git/hooks/pre-commit ]; then
    echo "❌ FAIL: Hook still exists after uninstall"
    exit 1
fi

echo "✅ PASS: Hook uninstalled successfully"

# Test 4: Uninstall non-existent hook - should fail
echo ""
echo "Test 4: Attempting to uninstall non-existent hook (should fail)..."
if ./crisk hook uninstall 2>&1 | grep -q "no pre-commit hook installed"; then
    echo "✅ PASS: Correctly reported no hook installed"
else
    echo "❌ FAIL: Should have reported no hook installed"
    exit 1
fi

# Test 5: Hook workflow - install and verify help text
echo ""
echo "Test 5: Verifying hook command help..."
if ./crisk hook --help | grep -q "Install or uninstall"; then
    echo "✅ PASS: Hook help text works"
else
    echo "❌ FAIL: Hook help text missing"
    exit 1
fi

# Test 6: Pre-commit flag on check command
echo ""
echo "Test 6: Testing --pre-commit flag..."

# Create and stage a test file
echo "package main" > test.go
git add test.go

# Note: This test will fail if infrastructure (Neo4j, Redis, Postgres) is not running
# So we just verify the flag is accepted and command runs
if ./crisk check --pre-commit --help 2>&1 | grep -q "pre-commit"; then
    echo "✅ PASS: --pre-commit flag is recognized"
else
    echo "❌ FAIL: --pre-commit flag not recognized"
    exit 1
fi

# Cleanup
cd "$PROJECT_ROOT"

echo ""
echo "=== All tests passed! ==="
echo ""
echo "Summary:"
echo "  ✅ Hook installation works"
echo "  ✅ Hook uninstallation works"
echo "  ✅ Duplicate installation prevented"
echo "  ✅ Uninstall non-existent hook handled"
echo "  ✅ Hook commands available"
echo "  ✅ --pre-commit flag recognized"
echo ""
echo "Note: Full end-to-end hook execution requires running infrastructure"
echo "      (Neo4j, Redis, Postgres). Run 'docker compose up -d' to test fully."
