#!/bin/bash
set -e

echo "=== Testing MCP Server ==="
echo ""

# Set environment variables
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Step 1: Verify databases are accessible"
echo "----------------------------------------"

# Test PostgreSQL
echo -n "Testing PostgreSQL... "
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT COUNT(*) FROM code_blocks WHERE repo_id = 4;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
    echo "PostgreSQL is not accessible. Check docker containers."
    exit 1
fi

# Test Neo4j
echo -n "Testing Neo4j... "
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (n:CodeBlock {repo_id: 4}) RETURN count(n) LIMIT 1"}]}' > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
    echo "Neo4j is not accessible. Check docker containers."
    exit 1
fi

echo ""
echo "Step 2: Check data availability"
echo "--------------------------------"

# Count code blocks
BLOCK_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "SELECT COUNT(*) FROM code_blocks WHERE repo_id = 4;" | xargs)
echo "Code blocks in PostgreSQL: ${BLOCK_COUNT}"

# Count incidents
INCIDENT_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "SELECT COUNT(*) FROM code_block_incidents;" | xargs)
echo "Temporal incidents: ${INCIDENT_COUNT}"

# Sample file paths
echo ""
echo "Sample file paths with blocks:"
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT DISTINCT file_path, COUNT(*) as block_count
FROM code_blocks
WHERE repo_id = 4
GROUP BY file_path
ORDER BY block_count DESC
LIMIT 5;"

echo ""
echo "Step 3: Test MCP server standalone"
echo "-----------------------------------"
echo ""
echo "Starting MCP server in background..."

# Start server in background
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server > /tmp/mcp-server-test.log 2>&1 &
SERVER_PID=$!
echo "Server PID: ${SERVER_PID}"

# Give it a moment to start
sleep 2

# Check if server is still running
if ps -p $SERVER_PID > /dev/null; then
    echo -e "${GREEN}✓ Server started${NC}"
else
    echo -e "${RED}✗ Server failed to start${NC}"
    echo "Check logs at /tmp/mcp-server-test.log"
    cat /tmp/mcp-server-test.log
    exit 1
fi

echo ""
echo "Step 4: Send test requests"
echo "--------------------------"

# Test 1: Initialize
echo ""
echo "Test 1: Initialize request"
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | nc localhost 12345 > /tmp/mcp-init-response.json 2>&1 || true

# Since it's stdio, we need to test differently
# Kill the background server
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "Step 5: Interactive test (stdio mode)"
echo "--------------------------------------"
echo ""
echo "The MCP server uses stdio transport, so we'll test with echo/pipe:"
echo ""

# Test initialize
echo -e "${YELLOW}Testing initialize...${NC}"
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server &
INIT_PID=$!
sleep 3
kill $INIT_PID 2>/dev/null || true

echo ""
echo "Step 6: Manual testing instructions"
echo "------------------------------------"
echo ""
echo "To test the MCP server manually:"
echo ""
echo "1. Start the server:"
echo "   NEO4J_URI=bolt://localhost:7688 \\"
echo "   NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \\"
echo "   POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \\"
echo "   /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server"
echo ""
echo "2. In another terminal, send JSON-RPC requests:"
echo ""
echo "   # Initialize"
echo '   echo '"'"'{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'"'"
echo ""
echo "   # List tools"
echo '   echo '"'"'{"jsonrpc":"2.0","method":"tools/list","id":2}'"'"
echo ""
echo "   # Call get_risk_summary"
echo '   echo '"'"'{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"src/index.ts"}}}'"'"
echo ""
echo "Step 7: Claude Code integration"
echo "--------------------------------"
echo ""
echo "Add to Claude Code MCP settings (~/.config/Claude/mcp_settings.json):"
echo ""
echo '{'
echo '  "mcpServers": {'
echo '    "crisk": {'
echo '      "command": "/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server",'
echo '      "args": [],'
echo '      "env": {'
echo '        "NEO4J_URI": "bolt://localhost:7688",'
echo '        "NEO4J_PASSWORD": "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",'
echo '        "POSTGRES_DSN": "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"'
echo '      }'
echo '    }'
echo '  }'
echo '}'
echo ""
echo "Then restart Claude Code and test in the mcp-use repository."
echo ""
echo -e "${GREEN}=== Test script complete ===${NC}"
