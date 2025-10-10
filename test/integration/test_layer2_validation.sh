#!/bin/bash
# Test Layer 2 (CO_CHANGED) edge creation validation
# 12-Factor Principle - Factor 10: Dev/prod parity
# Automated tests ensure consistent behavior across environments

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CRISK_BIN="$PROJECT_ROOT/crisk"

echo "==================================================================="
echo "Testing Layer 2 (CO_CHANGED) edge creation..."
echo "==================================================================="

# Build binary
echo "Building crisk binary..."
cd "$PROJECT_ROOT"
go build -o crisk ./cmd/crisk

# Check if Neo4j is running
if ! docker ps | grep -q coderisk-neo4j; then
    echo "❌ FAIL: Neo4j container not running"
    echo "Run: docker compose up -d"
    exit 1
fi

# Clean existing CO_CHANGED edges for fresh test
echo "Cleaning existing CO_CHANGED edges..."
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() DELETE r" 2>&1 | grep -v "Warning" || true

# Run init-local on test repo
echo "Running init-local on test repository..."
cd /tmp

# Use current project as test repo (it has git history)
if [ ! -d "test-repo" ]; then
    cp -r "$PROJECT_ROOT" test-repo
fi

cd test-repo
"$CRISK_BIN" init-local 2>&1 | grep -E "temporal analysis|CO_CHANGED|verification"

# Query edges
echo "Querying CO_CHANGED edges..."
COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" 2>&1 | grep -v "Warning" | grep -E "^[0-9]+$" | head -1)

if [ -z "$COUNT" ]; then
    echo "❌ FAIL: Could not query CO_CHANGED edges"
    exit 1
fi

echo "Found $COUNT CO_CHANGED edges"

if [ "$COUNT" -gt 0 ]; then
    echo "✅ PASS: $COUNT CO_CHANGED edges created"

    # Test specific query for high-frequency co-changes
    echo "Testing high-frequency co-changes query..."
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
      "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
       WHERE r.frequency >= 0.5
       RETURN f.path as file_a, other.path as file_b, r.frequency
       ORDER BY r.frequency DESC
       LIMIT 5" 2>&1 | grep -v "Warning" | head -20

    exit 0
else
    echo "❌ FAIL: No CO_CHANGED edges found"
    echo "Check logs for temporal analysis errors"
    exit 1
fi
