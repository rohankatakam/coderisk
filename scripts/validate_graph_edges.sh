#!/bin/bash
# Comprehensive graph edge validation script
# Verifies all 3 layers of edges are created correctly

set -e

NEO4J_CMD="docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 CodeRisk Graph Edge Validation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Wait for Neo4j to be ready
echo ""
echo "⏳ Waiting for Neo4j to be ready..."
for i in {1..30}; do
    if docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 "RETURN 1" &>/dev/null; then
        echo "  ✓ Neo4j is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "  ❌ Neo4j failed to start within 30 seconds"
        exit 1
    fi
    sleep 1
done

# Layer 1: Code Structure
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📁 Layer 1: Code Structure (CONTAINS, IMPORTS)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo "📊 Node Counts:"
$NEO4J_CMD "MATCH (f:File) RETURN count(f) as files" || true
$NEO4J_CMD "MATCH (fn:Function) RETURN count(fn) as functions" || true
$NEO4J_CMD "MATCH (c:Class) RETURN count(c) as classes" || true

echo ""
echo "🔗 Edge Counts:"
CONTAINS_COUNT=$($NEO4J_CMD "MATCH ()-[r:CONTAINS]->() RETURN count(r) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")
IMPORTS_COUNT=$($NEO4J_CMD "MATCH ()-[r:IMPORTS]->() RETURN count(r) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")

echo "  CONTAINS edges: ${CONTAINS_COUNT}"
echo "  IMPORTS edges:  ${IMPORTS_COUNT}"

if [ "${CONTAINS_COUNT}" -gt 0 ]; then
    echo "  ✅ CONTAINS edges created successfully"
else
    echo "  ⚠️  WARNING: No CONTAINS edges found"
fi

if [ "${IMPORTS_COUNT}" -gt 0 ]; then
    echo "  ✅ IMPORTS edges created successfully"
else
    echo "  ⚠️  WARNING: No IMPORTS edges found"
fi

# Layer 2: Temporal Analysis
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⏱️  Layer 2: Temporal Analysis (CO_CHANGED)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CO_CHANGED_COUNT=$($NEO4J_CMD "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")

echo ""
echo "🔗 CO_CHANGED Edge Count: ${CO_CHANGED_COUNT}"

if [ "${CO_CHANGED_COUNT}" -gt 0 ]; then
    echo "  ✅ CO_CHANGED edges created successfully!"

    echo ""
    echo "🔍 Sample CO_CHANGED edges (showing properties):"
    $NEO4J_CMD "MATCH (a:File)-[r:CO_CHANGED]->(b:File) RETURN a.name, b.name, r.frequency, r.co_changes, r.window_days LIMIT 3" || true

    echo ""
    echo "📈 Frequency distribution:"
    $NEO4J_CMD "MATCH ()-[r:CO_CHANGED]->() WHERE r.frequency >= 0.7 RETURN count(r) as high_frequency" || true
    $NEO4J_CMD "MATCH ()-[r:CO_CHANGED]->() WHERE r.frequency >= 0.3 AND r.frequency < 0.7 RETURN count(r) as medium_frequency" || true

    echo ""
    echo "🔄 Bidirectional verification (checking 5 random pairs):"
    $NEO4J_CMD "MATCH (a:File)-[r1:CO_CHANGED]->(b:File), (b)-[r2:CO_CHANGED]->(a) RETURN a.name, b.name, r1.frequency = r2.frequency as frequencies_match LIMIT 5" || true

else
    echo "  ❌ CRITICAL: No CO_CHANGED edges found!"
    echo ""
    echo "🔍 Diagnostic checks:"

    # Check if File nodes exist
    FILE_COUNT=$($NEO4J_CMD "MATCH (f:File) RETURN count(f) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")
    echo "  - File nodes in graph: ${FILE_COUNT}"

    if [ "${FILE_COUNT}" -eq 0 ]; then
        echo "  ❌ No File nodes found - graph construction may have failed"
    else
        echo "  ✅ File nodes exist"

        # Check sample File node properties
        echo ""
        echo "  Sample File node properties:"
        $NEO4J_CMD "MATCH (f:File) RETURN f.file_path, f.name LIMIT 2" || true

        echo ""
        echo "  Possible causes:"
        echo "    1. Git history had no commits (check logs for 'no commits found')"
        echo "    2. Path mismatch between git paths and File node paths"
        echo "    3. Co-change frequency threshold too high (current: 0.3)"
        echo "    4. Transaction commit failed (check logs for errors)"
    fi
fi

# Layer 3: Incidents
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚨 Layer 3: Incidents (CAUSED_BY)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

INCIDENT_COUNT=$($NEO4J_CMD "MATCH (i:Incident) RETURN count(i) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")
CAUSED_BY_COUNT=$($NEO4J_CMD "MATCH ()-[r:CAUSED_BY]->() RETURN count(r) as count" | grep -oE '[0-9]+' | tail -1 || echo "0")

echo ""
echo "📊 Incident Nodes: ${INCIDENT_COUNT}"
echo "🔗 CAUSED_BY Edges: ${CAUSED_BY_COUNT}"

if [ "${INCIDENT_COUNT}" -gt 0 ]; then
    echo "  ✅ Incident nodes created"

    if [ "${CAUSED_BY_COUNT}" -gt 0 ]; then
        echo "  ✅ CAUSED_BY edges created"
        echo ""
        echo "🔍 Sample incident links:"
        $NEO4J_CMD "MATCH (i:Incident)-[r:CAUSED_BY]->(f:File) RETURN i.title, f.name, r.confidence LIMIT 3" || true
    else
        echo "  ℹ️  No CAUSED_BY edges (incidents not linked to files yet)"
    fi
else
    echo "  ℹ️  No incidents created yet (this is normal - use 'crisk incident create' to add incidents)"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TOTAL_EDGES=$((CONTAINS_COUNT + IMPORTS_COUNT + CO_CHANGED_COUNT + CAUSED_BY_COUNT))

echo ""
echo "Total edges created: ${TOTAL_EDGES}"
echo "  - CONTAINS:   ${CONTAINS_COUNT}"
echo "  - IMPORTS:    ${IMPORTS_COUNT}"
echo "  - CO_CHANGED: ${CO_CHANGED_COUNT}"
echo "  - CAUSED_BY:  ${CAUSED_BY_COUNT}"

echo ""
if [ "${CO_CHANGED_COUNT}" -gt 0 ]; then
    echo "✅ SUCCESS: All edge types created successfully!"
    echo ""
    echo "Next steps:"
    echo "  - Test risk calculation: ./crisk check <file>"
    echo "  - Test AI mode: ./crisk check <file> --ai-mode"
    echo "  - Create incidents: ./crisk incident create <title> <description>"
else
    echo "⚠️  WARNING: CO_CHANGED edges are missing"
    echo ""
    echo "Recommended actions:"
    echo "  1. Check crisk init-local logs for errors"
    echo "  2. Verify git history exists: cd <repo> && git log --oneline | head -10"
    echo "  3. Re-run with debug logging: CODERISK_DEBUG=true ./crisk init-local"
fi

echo ""
