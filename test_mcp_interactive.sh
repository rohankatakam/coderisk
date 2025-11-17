#!/bin/bash

echo "=== MCP Server Interactive Test ==="
echo ""

# Set environment variables
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

echo "Data available:"
echo "- Code blocks: $(PGPASSWORD='CHANGE_THIS_PASSWORD_IN_PRODUCTION_123' psql -h localhost -p 5433 -U coderisk -d coderisk -t -c 'SELECT COUNT(*) FROM code_blocks WHERE repo_id = 4;' | xargs)"
echo "- Temporal incidents: $(PGPASSWORD='CHANGE_THIS_PASSWORD_IN_PRODUCTION_123' psql -h localhost -p 5433 -U coderisk -d coderisk -t -c 'SELECT COUNT(*) FROM code_block_incidents;' | xargs)"
echo ""
echo "Sample files with high block counts:"
PGPASSWORD='CHANGE_THIS_PASSWORD_IN_PRODUCTION_123' psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT file_path, COUNT(*) as blocks
FROM code_blocks
WHERE repo_id = 4
GROUP BY file_path
ORDER BY blocks DESC
LIMIT 3;" | head -3
echo ""

# Create test request file
cat > /tmp/mcp-test-requests.json << 'EOF'
{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"tools/list","id":2}
{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"libraries/python/mcp_use/client.py"}}}
EOF

echo "Starting MCP server and sending test requests..."
echo "================================================"
echo ""

# Run server with test requests
cat /tmp/mcp-test-requests.json | /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server 2>&1 | head -100

echo ""
echo "================================================"
echo ""
echo "To test manually, run:"
echo ""
echo "  NEO4J_URI=bolt://localhost:7688 \\"
echo "  NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \\"
echo "  POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \\"
echo "  /Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server"
echo ""
echo "Then paste JSON-RPC requests (one per line):"
echo '  {"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
echo '  {"jsonrpc":"2.0","method":"tools/list","id":2}'
echo '  {"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"libraries/python/mcp_use/client.py"}}}'
