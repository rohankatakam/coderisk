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

Returns risk evidence for a file including ownership, coupling, and temporal incident data. **Automatically detects and analyzes uncommitted changes** when a file_path is provided.

**Key Features**:
- **Auto-Detection**: When you provide a `file_path`, the tool automatically checks for uncommitted changes using `git diff`
- **Smart Analysis**: If uncommitted changes are found, uses diff-based LLM analysis; otherwise falls back to file-based analysis
- **Simple API**: Just provide a file path - no need to manually call `git diff` or pass diff content

**Parameters**:
- `file_path` (optional): Path to the file to analyze (relative or absolute) - tool will auto-detect uncommitted changes
- `repo_root` (optional): Repository root path for resolving absolute paths and git commands
- `diff_content` (optional): Git diff content for uncommitted change analysis (auto-detected if not provided)
- `max_coupled_blocks` (optional): Maximum coupled blocks per code block (default: 1)
- `max_incidents` (optional): Maximum incidents per code block (default: 1)
- `max_blocks` (optional): Maximum total blocks to return (default: 10, 0 = all)
- `block_types` (optional): Filter by block types (e.g., `["class", "function"]`)
- `summary_only` (optional): Return only aggregated statistics (default: false)
- `min_staleness` (optional): Only return blocks with staleness >= N days
- `min_incidents` (optional): Only return blocks with at least N incidents
- `min_risk_score` (optional): Only return blocks with risk score >= threshold
- `include_risk_score` (optional): Include calculated risk score in output
- `prioritize_recent` (optional): Boost risk score for recently changed code

**Example Usage in Claude Code**:

**Simple Query (Auto-Detection)**:
```
"What is the risk of my uncommitted changes in libraries/python/mcp_use/client/connectors/base.py?"
```

This will automatically:
1. Detect uncommitted changes using `git diff`
2. Use LLM to extract modified code blocks
3. Query the graph for risk evidence
4. Return comprehensive risk analysis

**Using Relative Path** (auto-detects uncommitted changes):
```json
{
  "tool": "crisk.get_risk_summary",
  "arguments": {
    "file_path": "docs/docs.json"
  }
}
```
- If uncommitted changes exist → diff-based analysis (LLM extraction)
- If no changes → file-based analysis (all blocks in file)

**Using Absolute Path** (requires `repo_root`):
```json
{
  "tool": "crisk.get_risk_summary",
  "arguments": {
    "file_path": "/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/auth/oauth.py",
    "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
  }
}
```
- Auto-detects uncommitted changes in the specified file
- Uses `repo_root` for both path normalization and git commands

**Manual Diff (Advanced)**:
```json
{
  "tool": "crisk.get_risk_summary",
  "arguments": {
    "diff_content": "diff --git a/src/auth.py b/src/auth.py\n...",
    "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
  }
}
```
- Bypasses auto-detection and uses provided diff directly

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

#### File-Based Analysis (Traditional)
1. **Path Resolution**: If absolute path provided, normalizes to relative using `repo_root`
2. **File Identity Resolution**: Uses `git log --follow` to track renamed files (cached in bbolt)
3. **Code Block Query**: Queries Neo4j for code blocks in the file (including all historical paths)
4. **Ownership Data**: Retrieved from Neo4j CodeBlock node properties
5. **Coupling Data**: Queries Neo4j CO_CHANGES_WITH edges (rate >= 0.5)
6. **Temporal Data**: Queries PostgreSQL code_block_incidents table
7. **Risk Scoring**: Calculates risk score based on incidents, coupling, staleness, and block type
8. **Response Assembly**: Combines all data into structured BlockEvidence objects

#### Diff-Based Analysis (Meta-Ingestion)
1. **LLM Extraction**: Uses Gemini Flash to extract modified code blocks from diff (reuses atomizer infrastructure)
2. **Path Resolution**: Normalizes file paths using `repo_root` if provided
3. **File Identity Resolution**: Uses `git log --follow` for each modified file
4. **Code Block Query**: Queries Neo4j for extracted blocks by name across all historical paths
5. **Risk Analysis**: Same as file-based (steps 4-8 above)

> **Meta-Ingestion**: The diff-based flow applies the same LLM extraction process used during ingestion to real-time diffs, enabling "What is the risk of my changes?" queries without tree-sitter parsing.

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
- **Absolute path without repo_root**: If using absolute path, ensure `repo_root` is provided

Check what files exist:
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT DISTINCT file_path FROM code_blocks WHERE repo_id = 4 LIMIT 10;"
```

**Solution for absolute paths**: Always pass `repo_root` when using absolute file paths:
```json
{
  "file_path": "/full/path/to/file.py",
  "repo_root": "/full/path/to/repo"
}
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

- `cmd/crisk-check-server/main.go` - Entry point, tool registration
- `internal/mcp/local_graph_client.go` - Neo4j/PostgreSQL queries
- `internal/mcp/identity_resolver.go` - git log --follow wrapper with dynamic repo_root
- `internal/mcp/diff_atomizer.go` - LLM-based diff extraction (meta-ingestion)
- `internal/mcp/path_utils.go` - Path normalization utilities
- `internal/mcp/tools/get_risk_summary.go` - Main risk tool with branching logic
- `internal/mcp/tools/types.go` - Data type definitions
- `internal/llm/client.go` - Multi-provider LLM client (OpenAI/Gemini)
- `internal/llm/gemini_client.go` - Gemini-specific client with retry logic

## Path Resolution Strategies

The MCP server supports three path resolution approaches:

### 1. Relative Paths (Simplest)
```json
{"file_path": "libraries/python/mcp_use/client/session.py"}
```
- Works if Claude Code is in the repository root
- No additional configuration needed

### 2. Absolute Paths (Dynamic)
```json
{
  "file_path": "/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/auth/oauth.py",
  "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
}
```
- Pass `repo_root` from Claude Code's working directory context
- MCP server normalizes absolute → relative for graph queries
- Enables cross-repository analysis

### 3. Diff-Based (Uncommitted Changes)
```json
{
  "diff_content": "diff --git a/src/file.py ...",
  "repo_root": "/Users/rohankatakam/Documents/brain/project"
}
```
- Analyzes uncommitted changes
- Uses LLM to extract modified code blocks
- Queries graph for historical risk data

**Key Insight**: The `repo_root` parameter is passed from the Claude Code session context, making path resolution dynamic without database storage or filesystem probing.

## Current Limitations

1. **Hardcoded repo_id**: Currently set to 4 (mcp-use repository)
2. **Single repo support**: Cannot analyze multiple repos simultaneously
3. **GEMINI_API_KEY required**: Diff-based analysis requires Gemini API key in environment

## Recent Enhancements

- [x] **Auto-detection of uncommitted changes** (NEW) - Tool automatically calls `git diff` when analyzing files
- [x] Support diff_content for uncommitted change analysis (LLM-based meta-ingestion)
- [x] Dynamic absolute path resolution via `repo_root` parameter
- [x] Risk scoring with filtering and prioritization options
- [x] Git history tracking with bbolt caching

## Future Enhancements

- [ ] Auto-detect repo_id from git remote
- [ ] Add resource endpoints for browsing all files
- [ ] Multi-repo support
- [ ] Incremental cache invalidation
- [ ] GraphQL query interface

---

**Status**: ✅ Fully Implemented with Auto-Detection and Diff-Based Analysis

**Last Updated**: 2025-11-16 22:30

**Recent Changes**:
- **Added automatic git diff detection** - Tool now internally checks for uncommitted changes when file_path is provided
- Made `file_path` optional in tool schema to support auto-detection
- Updated tool description to help Claude Code route diff-based queries correctly
- Simplified user experience - no need to manually call `git diff` or pass diff_content
- Enhanced documentation with auto-detection examples and workflow
