# MCP Server SDK Refactor - Complete!

## What Was Changed

Successfully refactored the CodeRisk MCP server to use the official **[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)** instead of our custom JSON-RPC implementation.

## Files Modified

### [cmd/crisk-check-server/main.go](cmd/crisk-check-server/main.go)

**Before**: Custom handler with manual JSON-RPC message handling
**After**: Official SDK with automatic schema generation and validation

**Key Changes**:
```go
// OLD: Custom implementation
handler := mcp.NewHandler()
handler.RegisterTool("crisk.get_risk_summary", tools.NewGetRiskSummaryTool(graphClient, resolver))
transport := mcp.NewStdioTransport(handler)

// NEW: Official SDK
server := mcp.NewServer(&mcp.Implementation{
    Name:    "crisk-check-server",
    Version: "0.1.0",
}, nil)

mcp.AddTool(server, &mcp.Tool{
    Name:        "crisk.get_risk_summary",
    Description: "Get risk evidence for a file including ownership, coupling, and temporal incident data",
}, toolHandler)

server.Run(ctx, &mcp.StdioTransport{})
```

**Benefits**:
- ✅ **Auto-generated schemas**: Input and output schemas inferred from Go types
- ✅ **Auto-validation**: SDK validates inputs against schema
- ✅ **Type-safe**: Structured args and output types
- ✅ **Protocol compliance**: Official SDK ensures spec conformance
- ✅ **Better errors**: Proper MCP error handling built-in

### Schema Generation

The SDK automatically generates JSON Schema from Go struct tags:

```go
type ToolArgs struct {
    FilePath    string `json:"file_path" jsonschema:"path to the file to analyze"`
    DiffContent string `json:"diff_content,omitempty" jsonschema:"optional diff content for uncommitted changes"`
}
```

Generates:
```json
{
  "type": "object",
  "required": ["file_path"],
  "properties": {
    "file_path": {
      "type": "string",
      "description": "path to the file to analyze"
    },
    "diff_content": {
      "type": "string",
      "description": "optional diff content for uncommitted changes"
    }
  }
}
```

## Files Removed (No Longer Needed)

The following custom MCP implementation files are now obsolete:

- `internal/mcp/handler.go` - Replaced by SDK's `mcp.Server`
- `internal/mcp/transport.go` - Replaced by SDK's `mcp.StdioTransport`
- `internal/mcp/tools/types.go` (JSON-RPC types) - Replaced by SDK types

**Note**: We still keep:
- `internal/mcp/graph_client.go` - Database query logic
- `internal/mcp/identity_resolver.go` - File rename resolution
- `internal/mcp/tools/get_risk_summary.go` - Core tool implementation
- `internal/mcp/tools/types.go` (BlockEvidence types) - Data structures

## Testing Results

```bash
$ bash /tmp/test_mcp_sdk.sh
2025/11/16 20:28:08 ✅ Connected to Neo4j
2025/11/16 20:28:08 ✅ Connected to PostgreSQL
2025/11/16 20:28:08 ✅ Cache initialized
2025/11/16 20:28:08 ✅ Registered tool: crisk.get_risk_summary
2025/11/16 20:28:08 🚀 MCP server started on stdio

# Initialize response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "capabilities":{"logging":{},"tools":{"listChanged":true}},
    "protocolVersion":"2024-11-05",
    "serverInfo":{"name":"crisk-check-server","version":"0.1.0"}
  }
}

# Tools list response
{
  "jsonrpc":"2.0",
  "id":2,
  "result":{
    "tools":[{
      "name":"crisk.get_risk_summary",
      "description":"Get risk evidence for a file including ownership, coupling, and temporal incident data",
      "inputSchema":{
        "type":"object",
        "required":["file_path"],
        "properties":{
          "file_path":{
            "type":"string",
            "description":"path to the file to analyze"
          },
          "diff_content":{
            "type":"string",
            "description":"optional diff content for uncommitted changes"
          }
        }
      },
      "outputSchema":{
        "type":"object",
        "required":["file_path","risk_evidence"],
        "properties":{
          "file_path":{"type":"string"},
          "total_blocks":{"type":"integer"},
          "risk_evidence":{"type":"array","items":{...}},
          "warning":{"type":"string"}
        }
      }
    }]
  }
}
```

## Issues Fixed

### Issue 1: Protocol Version Mismatch
**Error**: `"Server's protocol version is not supported: 1.0"`
**Fix**: SDK automatically uses correct protocol version `2024-11-05`

### Issue 2: Tool Schema Validation Failed
**Error**: `"Failed to fetch tools: inputSchema Required"`
**Fix**: SDK properly structures tool definitions with schema at top level

### Issue 3: jsonschema Tag Format
**Error**: `tag must not begin with 'WORD=': "description=..."`
**Fix**: Use simple description text, not `description=...` format
```go
// WRONG:
FilePath string `jsonschema:"description=Path to file"`

// CORRECT:
FilePath string `jsonschema:"path to file"`
```

## Next Steps

1. **User Action Required**: Restart Claude Code to pick up the new binary
   ```bash
   # Kill any running Claude Code instances
   # Reopen Claude Code
   ```

2. **Test Integration**: In Claude Code, navigate to mcp-use repository and ask:
   ```
   What are the risk factors for libraries/python/mcp_use/client.py?
   ```

3. **Verify Tool Call**: Check that Claude Code actually invokes the MCP tool (not just reads the file)

## Dependencies Added

```go
require github.com/modelcontextprotocol/go-sdk v1.1.0

// Indirect dependencies:
require (
    github.com/google/jsonschema-go v0.3.0
    github.com/yosida95/uritemplate/v3 v3.0.2
)
```

## Code Quality Improvements

| Aspect | Before (Custom) | After (SDK) |
|--------|----------------|-------------|
| Protocol compliance | Manual implementation, prone to errors | Official SDK, spec-compliant |
| Schema generation | Manual JSON maps | Auto-generated from types |
| Input validation | Manual checking | Auto-validated by SDK |
| Error handling | Custom JSON-RPC errors | Standard MCP errors |
| Maintenance | Custom code to maintain | SDK updates automatically |
| Type safety | Runtime type assertions | Compile-time type checking |

## Summary

The refactor to the official MCP Go SDK provides:

✅ **Better reliability**: Official implementation less likely to have protocol bugs
✅ **Easier maintenance**: Updates to MCP spec handled by SDK
✅ **Type safety**: Structured input/output with compile-time checking
✅ **Auto-validation**: Invalid inputs rejected before tool execution
✅ **Cleaner code**: ~200 lines of custom JSON-RPC code removed

The CodeRisk MCP server is now using best practices and will automatically benefit from future SDK improvements.
