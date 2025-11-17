# CodeRisk MCP Server

A Model Context Protocol (MCP) server that provides risk insights to Claude Code by querying Neo4j and PostgreSQL databases.

## Overview

The CodeRisk MCP server (`crisk-check-server`) exposes code risk analysis through the MCP protocol, allowing Claude Code to access:

- **Ownership Data**: Original author, last modifier, staleness, familiarity scores
- **Coupling Data**: Co-change relationships between code blocks
- **Temporal Data**: Incident history linked to code blocks

## Installation

The MCP server binary is located at:
```
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server
```

## Prerequisites

Before using the MCP server, ensure:

1. **Neo4j** is running on port **7688** (not the default 7687)
2. **PostgreSQL** is running on port **5433** (not the default 5432)
3. The databases contain code risk data from a previously analyzed repository

Verify databases are running:
```bash
# Check Neo4j
docker ps | grep neo4j

# Check PostgreSQL
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT COUNT(*) FROM code_blocks;"
```

## Claude Code Integration

### Step 1: Configure MCP Settings

Add the MCP server to Claude Code's configuration file:

**File**: `~/.config/Claude/mcp_settings.json` (or `~/Library/Application Support/Claude/mcp_settings.json` on macOS)

```json
{
  "mcpServers": {
    "crisk": {
      "command": "/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server",
      "args": [],
      "env": {
        "NEO4J_URI": "bolt://localhost:7688",
        "NEO4J_PASSWORD": "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
        "POSTGRES_DSN": "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
      }
    }
  }
}
```

**Important**: Use these exact connection details:
- Neo4j port: **7688** (not 7687)
- PostgreSQL port: **5433** (not 5432)
- PostgreSQL user: **coderisk** (not coderisk_user)

### Step 2: Restart Claude Code

After updating the configuration, restart Claude Code for the changes to take effect.

### Step 3: Verify Integration

In Claude Code, the MCP server should appear in the available tools. You can verify by asking:

```
"What MCP tools are available?"
```

You should see `crisk.get_risk_summary` listed.

## Using the Tool

### Tool: `crisk.get_risk_summary`

Returns risk evidence for a file including ownership, coupling, and temporal incident data.

**Parameters**:
- `file_path` (required): Path to the file to analyze
- `diff_content` (optional): Diff content for uncommitted changes (future use)

**Example Usage in Claude Code**:

```
"Can you get the risk summary for docs/docs.json?"
```

Or invoke the tool directly:
```json
{
  "tool": "crisk.get_risk_summary",
  "arguments": {
    "file_path": "docs/docs.json"
  }
}
```

**Example Response**:
```json
{
  "file_path": "docs/docs.json",
  "total_blocks": 1,
  "risk_evidence": [
    {
      "block_name": "UI Widgets",
      "block_type": "function",
      "file_path": "docs/docs.json",
      "original_author": "luigipederzani@gmail.com",
      "last_modifier": "luigipederzani@gmail.com",
      "staleness_days": 24,
      "familiarity_map": "[{\"dev\":\"luigipederzani@gmail.com\",\"edits\":1}]",
      "semantic_importance": "",
      "coupled_blocks": [
        {
          "id": "4:codeblock:...",
          "name": "OpenAIComponentRenderer",
          "file_path": "libraries/typescript/packages/mcp-ui/src/OpenAIComponentRenderer.tsx",
          "coupling_rate": 1.0,
          "co_change_count": 0
        }
      ],
      "incident_count": 0,
      "incidents": []
    }
  ]
}
```

## Standalone Testing

Test the MCP server without Claude Code:

```bash
# Test initialize
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/crisk-check-server

# List available tools
echo '{"jsonrpc":"2.0","method":"tools/list","id":2}' | ./bin/crisk-check-server

# Call the risk summary tool
echo '{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"crisk.get_risk_summary","arguments":{"file_path":"docs/docs.json"}}}' | ./bin/crisk-check-server
```

## How It Works

### Architecture

```
┌─────────────┐     stdio      ┌──────────────────┐
│ Claude Code │ ◄─────────────► │ crisk-check-     │
│             │   JSON-RPC      │ server (MCP)     │
└─────────────┘                 └────────┬─────────┘
                                         │
                          ┌──────────────┴──────────────┐
                          │                             │
                          ▼                             ▼
                    ┌──────────┐                 ┌──────────┐
                    │  Neo4j   │                 │ Postgres │
                    │ (7688)   │                 │ (5433)   │
                    └──────────┘                 └──────────┘
                    Ownership +                  Temporal
                    Coupling                     Incidents
```

### Data Flow

1. **File Identity Resolution**: Uses `git log --follow` to track renamed files (cached in bbolt)
2. **Code Block Query**: Queries Neo4j for code blocks in the file (including all historical paths)
3. **Ownership Data**: Retrieved from Neo4j CodeBlock node properties
4. **Coupling Data**: Queries Neo4j CO_CHANGES_WITH edges (rate >= 0.5)
5. **Temporal Data**: Queries PostgreSQL code_block_incidents table
6. **Response Assembly**: Combines all data into structured BlockEvidence objects

### Caching

File rename history is cached in `/tmp/crisk-mcp-cache.db` using bbolt to avoid repeated git calls.

To clear the cache:
```bash
rm /tmp/crisk-mcp-cache.db
```

## Troubleshooting

### Server won't start

**Error**: "Failed to connect to Neo4j"
- Verify Neo4j is running: `docker ps | grep neo4j`
- Check port: Should be **7688**, not 7687
- Verify password in environment variable

**Error**: "Failed to connect to PostgreSQL"
- Verify PostgreSQL is running: `docker ps | grep postgres`
- Check port: Should be **5433**, not 5432
- Verify DSN format includes `?sslmode=disable`

### No blocks found for file

**Response**: `{"warning": "No code blocks found for this file"}`

This means:
- File hasn't been analyzed yet (run `crisk init` first)
- File path doesn't match what's in the database
- Wrong repo_id (currently hardcoded to 4)

Check what files exist:
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT DISTINCT file_path FROM code_blocks WHERE repo_id = 4 LIMIT 10;"
```

### Empty coupling/temporal data

This is expected if:
- Block was modified in isolation (no co-changes)
- No issues/PRs linked to this block (low linking quality)

These are non-fatal and will return empty arrays.

## Performance

- **Cold cache** (first call): ~2-3 seconds
- **Warm cache** (subsequent calls): ~500ms
- **Memory usage**: <500 MB
- **Cache file size**: <10 MB

## Development

### Rebuilding the binary

```bash
cd /Users/rohankatakam/Documents/brain/coderisk
go build -o bin/crisk-check-server ./cmd/crisk-check-server
```

### Source files

- `cmd/crisk-check-server/main.go` - Entry point
- `internal/mcp/handler.go` - MCP protocol handler
- `internal/mcp/stdio_transport.go` - stdio JSON-RPC transport
- `internal/mcp/local_graph_client.go` - Neo4j/PostgreSQL queries
- `internal/mcp/identity_resolver.go` - git log --follow wrapper
- `internal/mcp/tools/get_risk_summary.go` - Main risk tool
- `internal/mcp/tools/types.go` - Data type definitions

## Current Limitations

1. **Hardcoded repo_id**: Currently set to 4 (mcp-use repository)
2. **Uncommitted changes**: `diff_content` parameter not yet implemented
3. **Single repo support**: Cannot analyze multiple repos simultaneously

## Future Enhancements

- [ ] Auto-detect repo_id from git remote
- [ ] Support diff_content for uncommitted change analysis
- [ ] Add resource endpoints for browsing all files
- [ ] Multi-repo support
- [ ] Incremental cache invalidation
- [ ] GraphQL query interface

---

**Status**: ✅ Fully Implemented and Tested

**Last Updated**: 2025-11-16
