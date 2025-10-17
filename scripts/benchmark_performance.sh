#!/bin/bash
# Performance benchmarking script for CodeRisk
# Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 8

set -e

echo "=== CodeRisk Performance Benchmark ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Neo4j is running
if ! docker ps | grep -q coderisk-neo4j; then
    echo -e "${RED}❌ Neo4j container not running${NC}"
    echo "Start with: docker compose up -d neo4j"
    exit 1
fi

echo -e "${GREEN}✓ Neo4j running${NC}"
echo ""

# Run Go tests with benchmarking
echo "=== Running Performance Tests ==="
go test -v -run TestPerformance ./internal/graph || true

echo ""
echo "=== Running Benchmarks ==="
go test -bench=. -benchmem -benchtime=10s ./internal/graph || true

echo ""
echo "=== Performance Profile Summary ==="

# Run a simple query performance test
echo "Testing Tier 1 query performance..."
echo ""

# Create a test file for performance validation
cat > /tmp/perf_test.cypher <<'EOF'
// Test query performance
MATCH (f:File) RETURN count(f) as file_count;
MATCH (fn:Function) RETURN count(fn) as function_count;
MATCH (c:Commit) RETURN count(c) as commit_count;
EOF

echo "Graph Statistics:"
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

NEO4J_PASSWORD=${NEO4J_PASSWORD:-"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"}

docker exec coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" < /tmp/perf_test.cypher 2>/dev/null || echo "Query failed"

rm /tmp/perf_test.cypher

echo ""
echo "=== Performance Targets (from optimization guide) ==="
echo ""
echo "Tier 1 Metrics:"
echo "  QueryCoupling: < 150ms ⏱️"
echo "  QueryCoChange: < 150ms ⏱️"
echo ""
echo "Tier 2 Metrics:"
echo "  Ownership Query: < 1s ⏱️"
echo ""
echo "Ingestion:"
echo "  Layer 1 (5K files): < 5 min ⏱️"
echo "  Layer 2 (git history): < 10 min ⏱️"
echo "  Total (5K files): < 15 min ⏱️"
echo ""
echo "Memory:"
echo "  Medium repos (5K files): < 2GB 💾"
echo ""

# Check if performance regression
echo "=== Regression Check ==="
echo ""
echo "Run full regression suite with:"
echo "  go test -v -run TestPerformanceBaselines ./internal/graph"
echo ""
echo "Profile ingestion performance with:"
echo "  time ./crisk init omnara-ai/omnara"
echo ""

echo -e "${GREEN}✓ Benchmark complete${NC}"
