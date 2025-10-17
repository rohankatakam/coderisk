#!/bin/bash
# Apply Neo4j indexes for CodeRisk
# Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 1

set -e

echo "🔧 Applying Neo4j indexes for CodeRisk..."
echo ""

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

NEO4J_PASSWORD=${NEO4J_PASSWORD:-"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"}

# Check if Neo4j is running
if ! docker ps | grep -q coderisk-neo4j; then
    echo "❌ Neo4j container not running. Start with: docker compose up -d neo4j"
    exit 1
fi

echo "📊 Current indexes:"
echo "SHOW INDEXES;" | docker exec -i coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" 2>/dev/null || echo "No indexes yet"

echo ""
echo "🔨 Creating indexes..."
cat scripts/schema/neo4j_indexes.cypher | docker exec -i coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD"

echo ""
echo "✅ Indexes applied successfully"
echo ""
echo "📊 Updated indexes:"
echo "SHOW INDEXES;" | docker exec -i coderisk-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD"

echo ""
echo "⏱️  Note: Indexes may take a few seconds to populate for large graphs"
