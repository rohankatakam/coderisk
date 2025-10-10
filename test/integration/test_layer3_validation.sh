#!/bin/bash
# Test Layer 3 (CAUSED_BY) edge creation validation
# 12-Factor Principle - Factor 10: Dev/prod parity
# Automated tests ensure consistent edge creation across environments

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CRISK_BIN="$PROJECT_ROOT/crisk"

echo "==================================================================="
echo "Testing Layer 3 (CAUSED_BY) edge creation..."
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

# Check if PostgreSQL is running
if ! docker ps | grep -q coderisk-postgres; then
    echo "❌ FAIL: PostgreSQL container not running"
    echo "Run: docker compose up -d"
    exit 1
fi

# Create test incident
echo "Creating test incident..."
INCIDENT_OUTPUT=$("$CRISK_BIN" incident create "Test Layer 3 Incident" "Testing CAUSED_BY edge creation" --severity high 2>&1)
echo "$INCIDENT_OUTPUT"

INCIDENT_ID=$(echo "$INCIDENT_OUTPUT" | grep -oP 'ID: \K[a-f0-9-]+' | head -1)

if [ -z "$INCIDENT_ID" ]; then
    echo "❌ FAIL: Could not extract incident ID"
    exit 1
fi

echo "Created incident with ID: $INCIDENT_ID"

# Link to file
echo "Linking incident to file..."
TEST_FILE="cmd/crisk/check.go"
"$CRISK_BIN" incident link "$INCIDENT_ID" "$TEST_FILE" --line 10 --function "main"

# Verify edge in Neo4j
echo "Verifying CAUSED_BY edge in Neo4j..."
COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f:File)
   RETURN count(r) as count" 2>&1 | grep -v "Warning" | grep -E "^[0-9]+$" | head -1)

if [ -z "$COUNT" ]; then
    echo "❌ FAIL: Could not query CAUSED_BY edges"
    exit 1
fi

echo "Found $COUNT CAUSED_BY edges"

if [ "$COUNT" -eq 1 ]; then
    echo "✅ PASS: CAUSED_BY edge created successfully"

    # Verify edge properties
    echo "Verifying edge properties..."
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
      "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f:File)
       RETURN i.title as incident_title, f.path as file_path,
              r.confidence as confidence, r.line_number as line_number,
              r.blamed_function as blamed_function" 2>&1 | grep -v "Warning"

    # Clean up
    echo "Cleaning up test incident..."
    "$CRISK_BIN" incident delete "$INCIDENT_ID" 2>&1 || true

    exit 0
else
    echo "❌ FAIL: Expected 1 CAUSED_BY edge, found $COUNT"
    echo "Incident may not be properly linked in Neo4j"

    # Clean up
    "$CRISK_BIN" incident delete "$INCIDENT_ID" 2>&1 || true

    exit 1
fi
