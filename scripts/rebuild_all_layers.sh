#!/bin/bash
# Rebuild all 3 layers of the knowledge graph from Postgres staging data
# Layer 1: TreeSitter (code structure)
# Layer 2: Git/GitHub (commits, PRs, developers)
# Layer 3: Issue linking (FIXES_ISSUE edges)

set -e

REPO=${1:-"omnara"}
NEO4J_PASSWORD=${NEO4J_PASSWORD:-"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"}
NEO4J_URI=${NEO4J_URI:-"bolt://localhost:7688"}
NEO4J_USERNAME=${NEO4J_USERNAME:-"neo4j"}

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  REBUILD ALL GRAPH LAYERS                                    ║"
echo "║  Repository: $(printf '%-46s' "$REPO") ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Step 0: Verify Neo4j connection
echo "🔌 Step 0: Verifying Neo4j connection..."
if ! curl -s -u "$NEO4J_USERNAME:$NEO4J_PASSWORD" -H "Content-Type: application/json" \
    -X POST "http://localhost:7475/db/neo4j/tx/commit" \
    -d '{"statements":[{"statement":"RETURN 1"}]}' > /dev/null 2>&1; then
    echo "❌ Failed to connect to Neo4j at http://localhost:7475"
    echo "   Make sure Neo4j is running: docker compose up -d"
    exit 1
fi
echo "✅ Neo4j connection verified"
echo ""

# Step 1: Wipe existing graph for this repo
echo "🧹 Step 1: Wiping existing Neo4j data for repo: $REPO..."

curl -s -u "$NEO4J_USERNAME:$NEO4J_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "http://localhost:7475/db/neo4j/tx/commit" \
  -d "{
    \"statements\": [{
      \"statement\": \"MATCH (n) WHERE n.repo_name = '$REPO' DETACH DELETE n\"
    }]
  }" > /dev/null

echo "✅ Wipe complete"
echo ""

# Step 2: Rebuild Layer 1 (TreeSitter - Code Structure)
echo "🌲 Step 2: Rebuilding Layer 1 (TreeSitter Code Structure)..."
echo "   Expected time: 10-30 seconds"
echo ""

# TODO: Implement Layer 1 rebuild
# For now, this is a placeholder - actual implementation will be in Go
# go run cmd/rebuild_graph/main.go --repo "$REPO" --layer 1

echo "⏭️  Layer 1 rebuild not yet implemented (placeholder)"
echo ""

# Step 3: Rebuild Layer 2 (Git/GitHub - Commits, PRs, Developers)
echo "📝 Step 3: Rebuilding Layer 2 (Git/GitHub Temporal Data)..."
echo "   Expected time: 30-60 seconds"
echo ""

# TODO: Implement Layer 2 rebuild
# go run cmd/rebuild_graph/main.go --repo "$REPO" --layer 2

echo "⏭️  Layer 2 rebuild not yet implemented (placeholder)"
echo ""

# Step 4: Rebuild Layer 3 (Issue Linking with LLM)
echo "🔗 Step 4: Rebuilding Layer 3 (Issue Linking)..."
echo "   Expected time: 2-5 minutes (LLM calls)"
echo ""

# TODO: Implement Layer 3 rebuild
# This will use the new comment analyzer and temporal correlator
# go run cmd/rebuild_graph/main.go --repo "$REPO" --layer 3

echo "⏭️  Layer 3 rebuild not yet implemented (placeholder)"
echo ""

# Step 5: Verify graph construction
echo "✅ Step 5: Verifying graph construction..."

# Query Neo4j to count nodes and edges
RESULT=$(curl -s -u "$NEO4J_USERNAME:$NEO4J_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "http://localhost:7475/db/neo4j/tx/commit" \
  -d "{
    \"statements\": [{
      \"statement\": \"MATCH (n) WHERE n.repo_name = '$REPO' RETURN count(n) as node_count\"
    }]
  }")

NODE_COUNT=$(echo "$RESULT" | jq -r '.results[0].data[0].row[0]')

echo "  Nodes created: $NODE_COUNT"

# Query for edges
EDGE_RESULT=$(curl -s -u "$NEO4J_USERNAME:$NEO4J_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "http://localhost:7475/db/neo4j/tx/commit" \
  -d "{
    \"statements\": [{
      \"statement\": \"MATCH ()-[r]->() WHERE r.repo_name = '$REPO' OR startNode(r).repo_name = '$REPO' RETURN count(r) as edge_count\"
    }]
  }")

EDGE_COUNT=$(echo "$EDGE_RESULT" | jq -r '.results[0].data[0].row[0]')

echo "  Edges created: $EDGE_COUNT"
echo ""

if [ "$NODE_COUNT" -eq 0 ]; then
    echo "⚠️  Warning: No nodes created (implementation pending)"
else
    echo "✅ Graph rebuild complete!"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  REBUILD SUMMARY                                             ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║ Repository:  $(printf '%-48s' "$REPO") ║"
echo "║ Nodes:       $(printf '%-48s' "$NODE_COUNT") ║"
echo "║ Edges:       $(printf '%-48s' "$EDGE_COUNT") ║"
echo "╚══════════════════════════════════════════════════════════════╝"
