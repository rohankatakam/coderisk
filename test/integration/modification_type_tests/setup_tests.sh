#!/bin/bash
# Setup script for modification type tests
# Prepares environment and validates prerequisites

set -e

echo "========================================"
echo "Modification Type Tests - Setup"
echo "========================================"
echo ""

# 1. Check we're in coderisk-go root
if [ ! -f "Makefile" ] || [ ! -d "cmd/crisk" ]; then
    echo "❌ ERROR: Must run from coderisk-go root directory"
    echo "Current directory: $(pwd)"
    exit 1
fi
echo "✅ Running from coderisk-go root"

# 2. Check crisk binary exists
if [ ! -f "./bin/crisk" ] && [ ! -f "./crisk" ]; then
    echo "❌ crisk binary not found. Building..."
    make build
    if [ ! -f "./bin/crisk" ] && [ ! -f "./crisk" ]; then
        echo "❌ Build failed"
        exit 1
    fi
fi

if [ -f "./bin/crisk" ]; then
    echo "✅ crisk binary exists at ./bin/crisk"
    CRISK_BIN="./bin/crisk"
elif [ -f "./crisk" ]; then
    echo "✅ crisk binary exists at ./crisk"
    CRISK_BIN="./crisk"
fi

# 3. Check Docker services
echo ""
echo "Checking Docker services..."
if ! docker ps | grep -q "neo4j\|postgres\|redis"; then
    echo "⚠️  WARNING: Docker services may not be running"
    echo "Starting services..."
    docker compose up -d
    sleep 5  # Give services time to start
else
    echo "✅ Docker services running"
fi

# 4. Check omnara repository
echo ""
echo "Checking omnara test repository..."
if [ ! -d "test_sandbox/omnara" ]; then
    echo "⚠️  omnara repository not found. Cloning..."
    mkdir -p test_sandbox
    cd test_sandbox
    git clone https://github.com/omnara-ai/omnara.git
    cd omnara
    echo "✅ omnara repository cloned"
else
    echo "✅ omnara repository exists"
fi

# 5. Verify omnara git state
cd test_sandbox/omnara
if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  omnara working directory not clean"
    echo "Modified files:"
    git status --short
    echo ""
    echo "Cleaning working directory..."
    git restore .
    git clean -fd
fi

if [ -n "$(git status --porcelain)" ]; then
    echo "❌ Failed to clean working directory"
    exit 1
fi
echo "✅ omnara working directory clean"
cd ../..

# 6. Make test scripts executable
echo ""
echo "Making test scripts executable..."
chmod +x test/integration/modification_type_tests/*.sh
echo "✅ Test scripts are executable"

# 7. Create output directory
echo ""
echo "Preparing output directory..."
mkdir -p test/integration/modification_type_tests
echo "✅ Output directory ready"

# 8. Summary
echo ""
echo "========================================"
echo "✅ Setup Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo "  1. Run individual test:"
echo "     ./test/integration/modification_type_tests/scenario_7_security.sh"
echo ""
echo "  2. Run all tests:"
echo "     ./test/integration/modification_type_tests/run_all_tests.sh"
echo ""
echo "  3. View test documentation:"
echo "     cat test/integration/modification_type_tests/README.md"
echo ""

# Optional: Initialize omnara graph
echo "Optional: Initialize omnara graph for realistic testing"
echo "  ./crisk init-local test_sandbox/omnara"
echo ""
echo "(This enables coupling analysis, co-change detection, etc.)"
echo "Run time: ~30-60 seconds"
echo ""

read -p "Initialize omnara graph now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Initializing graph..."
    ./crisk init-local test_sandbox/omnara || echo "⚠️  Graph initialization failed (non-critical)"
    echo "✅ Graph initialization complete"
fi

echo ""
echo "========================================"
echo "Ready to run tests!"
echo "========================================"
