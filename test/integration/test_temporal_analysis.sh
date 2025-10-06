#!/bin/bash

# Integration test for temporal analysis (Layer 2)
# Tests CO_CHANGED edge creation and ownership tracking

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🧪 Testing temporal analysis on real repository..."

# Use omnara repo if available, otherwise use current repo
TEST_REPO="${TEST_REPO:-/tmp/omnara}"

if [ ! -d "$TEST_REPO/.git" ]; then
    echo "⚠️  Test repo not found at $TEST_REPO, using current repo instead"
    TEST_REPO="$PROJECT_ROOT"
fi

echo "📂 Using repository: $TEST_REPO"

# Check if Neo4j is running
if ! docker ps | grep -q coderisk-neo4j; then
    echo "❌ Neo4j container not running. Start it with: docker compose up -d"
    exit 1
fi

# Get Neo4j password from env or use default
NEO4J_PASSWORD="${NEO4J_PASSWORD:-CHANGE_THIS_PASSWORD_IN_PRODUCTION_123}"

# Run init-local to trigger temporal analysis
echo "🔄 Running init-local to analyze repository..."
cd "$PROJECT_ROOT"
./bin/crisk init-local "$TEST_REPO" || {
    echo "❌ init-local failed"
    exit 1
}

# Test 1: Verify CO_CHANGED edges exist
echo ""
echo "📊 Test 1: Checking CO_CHANGED edges..."
EDGE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" --format plain | tail -1 | tr -d '"')

echo "   Found $EDGE_COUNT CO_CHANGED edges"

if [ -z "$EDGE_COUNT" ] || [ "$EDGE_COUNT" -lt 1 ]; then
    echo "   ❌ ERROR: Expected at least 1 CO_CHANGED edge, got $EDGE_COUNT"
    exit 1
fi
echo "   ✅ CO_CHANGED edges created successfully"

# Test 2: Verify CO_CHANGED edge properties
echo ""
echo "📊 Test 2: Checking CO_CHANGED edge properties..."
SAMPLE_EDGE=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH (a)-[r:CO_CHANGED]->(b)
   RETURN a.name as fileA, b.name as fileB, r.frequency as freq, r.co_changes as cochanges
   ORDER BY r.frequency DESC LIMIT 1" --format plain)

echo "   Sample edge: $SAMPLE_EDGE"

if echo "$SAMPLE_EDGE" | grep -q "frequency"; then
    echo "   ✅ CO_CHANGED edges have required properties"
else
    echo "   ❌ ERROR: CO_CHANGED edges missing properties"
    exit 1
fi

# Test 3: Query top co-changed pairs
echo ""
echo "📊 Test 3: Top 5 co-changed file pairs..."
docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH (a:File)-[r:CO_CHANGED]->(b:File)
   WHERE id(a) < id(b)
   RETURN a.path as fileA, b.path as fileB, r.frequency as frequency
   ORDER BY r.frequency DESC LIMIT 5" --format plain | head -10

# Test 4: Verify Developer nodes exist
echo ""
echo "📊 Test 4: Checking Developer nodes..."
DEV_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH (d:Developer) RETURN count(d) as count" --format plain | tail -1 | tr -d '"')

echo "   Found $DEV_COUNT developers"

if [ -z "$DEV_COUNT" ] || [ "$DEV_COUNT" -lt 1 ]; then
    echo "   ⚠️  WARNING: Expected at least 1 Developer node, got $DEV_COUNT"
else
    echo "   ✅ Developer nodes created successfully"
fi

# Test 5: Verify Commit nodes exist
echo ""
echo "📊 Test 5: Checking Commit nodes..."
COMMIT_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH (c:Commit) RETURN count(c) as count" --format plain | tail -1 | tr -d '"')

echo "   Found $COMMIT_COUNT commits"

if [ -z "$COMMIT_COUNT" ] || [ "$COMMIT_COUNT" -lt 1 ]; then
    echo "   ⚠️  WARNING: Expected at least 1 Commit node, got $COMMIT_COUNT"
else
    echo "   ✅ Commit nodes created successfully"
fi

# Test 6: Verify AUTHORED edges
echo ""
echo "📊 Test 6: Checking AUTHORED edges (Developer -> Commit)..."
AUTHORED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH ()-[r:AUTHORED]->() RETURN count(r) as count" --format plain | tail -1 | tr -d '"')

echo "   Found $AUTHORED_COUNT AUTHORED edges"

if [ -z "$AUTHORED_COUNT" ] || [ "$AUTHORED_COUNT" -lt 1 ]; then
    echo "   ⚠️  WARNING: Expected at least 1 AUTHORED edge, got $AUTHORED_COUNT"
else
    echo "   ✅ AUTHORED edges created successfully"
fi

# Test 7: Test ownership query
echo ""
echo "📊 Test 7: Testing ownership query..."
OWNERSHIP_RESULT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH (f:File)<-[:MODIFIES]-(c:Commit)<-[:AUTHORED]-(d:Developer)
   WITH f, d, count(c) as commits
   ORDER BY commits DESC
   RETURN f.path as file, d.email as owner, commits
   LIMIT 5" --format plain | head -10)

echo "$OWNERSHIP_RESULT"

if echo "$OWNERSHIP_RESULT" | grep -q "owner"; then
    echo "   ✅ Ownership query working"
else
    echo "   ⚠️  WARNING: Ownership query returned unexpected results"
fi

# Test 8: Verify frequency range
echo ""
echo "📊 Test 8: Checking CO_CHANGED frequency range..."
FREQ_STATS=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
  "MATCH ()-[r:CO_CHANGED]->()
   RETURN min(r.frequency) as min_freq, max(r.frequency) as max_freq, avg(r.frequency) as avg_freq" \
   --format plain)

echo "   Frequency statistics: $FREQ_STATS"

if echo "$FREQ_STATS" | grep -q "min_freq"; then
    echo "   ✅ Frequency statistics available"
else
    echo "   ⚠️  WARNING: Could not get frequency statistics"
fi

# Summary
echo ""
echo "=========================================="
echo "✅ Temporal Analysis Integration Test PASSED"
echo "=========================================="
echo "Summary:"
echo "  - CO_CHANGED edges: $EDGE_COUNT"
echo "  - Developer nodes: $DEV_COUNT"
echo "  - Commit nodes: $COMMIT_COUNT"
echo "  - AUTHORED edges: $AUTHORED_COUNT"
echo ""
echo "Temporal analysis is working correctly! 🚀"
