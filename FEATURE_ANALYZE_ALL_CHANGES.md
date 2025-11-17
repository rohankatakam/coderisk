# Feature: Analyze All Uncommitted Changes

## Overview

The MCP server now supports analyzing **ALL uncommitted changes** across an entire repository in a single query. This eliminates the need to specify individual files and provides comprehensive risk analysis for your entire working diff.

## Implementation

### Core Mechanism

When `analyze_all_changes=true` is set:

1. **Git Diff Retrieval**: `git diff HEAD` (no file path) gets all uncommitted changes
2. **LLM Extraction**: Gemini Flash extracts modified code blocks from the complete diff
3. **Multi-File Resolution**: For each extracted block, resolves file identity (handles renames)
4. **Graph Queries**: Queries Neo4j for each modified block across all files
5. **Risk Assembly**: Aggregates risk evidence (ownership, coupling, incidents) for all blocks

### Parameters

```json
{
  "analyze_all_changes": true,  // Analyze ALL uncommitted changes
  "repo_root": "/path/to/repo"  // Optional: working directory for git commands
}
```

- `analyze_all_changes` (boolean): When `true`, analyzes all uncommitted changes
- `repo_root` (string): Repository root for git commands and path resolution
- All other filtering parameters still apply (min_risk_score, block_types, etc.)

## Usage Examples

### Simple Query
```
"What is the risk of all my uncommitted changes?"
```

Claude Code will route this to:
```json
{
  "tool": "crisk.get_risk_summary",
  "arguments": {
    "analyze_all_changes": true,
    "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
  }
}
```

### Advanced Filtering
```json
{
  "analyze_all_changes": true,
  "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use",
  "min_risk_score": 5.0,
  "max_blocks": 20,
  "include_risk_score": true
}
```

## Response Format

```json
{
  "analysis_type": "diff-based",
  "scope": "all uncommitted changes",
  "diff_provided": true,
  "total_blocks": 5,
  "risk_evidence": [
    {
      "block_name": "BaseConnector._initialize_capability_list",
      "block_type": "method",
      "file_path": "libraries/python/mcp_use/client/connectors/base.py",
      "original_author": "luigi@example.com",
      "staleness_days": 2,
      "coupled_blocks": [...],
      "incident_count": 0,
      "incidents": []
    },
    {
      "block_name": "ConfigManager",
      "block_type": "class",
      "file_path": ".mcp.json",
      "original_author": "rohan@example.com",
      "staleness_days": 0,
      ...
    }
  ]
}
```

## Key Differences from Single-File Analysis

### Single File (`file_path`)
- Analyzes **all blocks in one file** (file-based)
- Does NOT auto-detect uncommitted changes
- Returns ownership/coupling/incidents for entire file

### All Changes (`analyze_all_changes=true`)
- Analyzes **only modified blocks across all files** (diff-based)
- Automatically runs `git diff HEAD`
- Returns risk evidence for changed code only

## Technical Flow

```
User Query: "What is the risk of all my uncommitted changes?"
    ↓
Claude Code Routes → crisk.get_risk_summary(analyze_all_changes=true)
    ↓
Tool Execution:
    1. getUncommittedDiff("", repo_root) → Full git diff
    2. DiffAtomizer.ExtractBlocksFromDiff(diff) → Block references
    3. For each block:
        - IdentityResolver.ResolveHistoricalPaths(file)
        - GraphClient.GetCodeBlocksByNames(blocks, historical_paths)
        - GraphClient.GetCouplingData(block_id)
        - GraphClient.GetTemporalData(block_id)
    4. Aggregate and return risk evidence
```

## Advantages

1. **Comprehensive**: Analyzes all changes in one query
2. **Efficient**: Single git diff, single LLM call
3. **Accurate**: Uses same LLM extraction as ingestion (meta-ingestion)
4. **Context-Aware**: Resolves file renames, tracks coupled blocks
5. **Flexible**: Supports all filtering and sorting options

## Limitations

1. **repo_id Hardcoded**: Currently set to 4 (mcp-use repository)
2. **Requires GEMINI_API_KEY**: Diff-based analysis needs LLM
3. **Large Diffs**: Very large diffs may hit LLM context limits

## Testing

### Prerequisites
- Neo4j running on port 7688
- PostgreSQL running on port 5433
- GEMINI_API_KEY configured in `.mcp.json`
- Repository analyzed with `crisk init`

### Test Commands

```bash
# Make multiple changes
cd /Users/rohankatakam/Documents/brain/mcp-use
echo "// test change" >> libraries/python/mcp_use/client/session.py
echo "// test change" >> libraries/python/mcp_use/client/connectors/base.py

# In Claude Code
"What is the risk of all my uncommitted changes?"
```

### Expected Output
- `analysis_type`: "diff-based"
- `scope`: "all uncommitted changes"
- `risk_evidence`: Array of blocks from multiple files

## Migration from Old Auto-Detection

**Old Approach** (removed):
- Auto-detected uncommitted changes when `file_path` was provided
- Ambiguous: "analyze file" vs "analyze uncommitted changes in file"

**New Approach** (current):
- Explicit parameter: `analyze_all_changes=true`
- Clear separation:
  - `file_path` = analyze entire file (all blocks)
  - `analyze_all_changes` = analyze uncommitted changes (all files)

## Future Enhancements

- [ ] Auto-detect repo_id from git remote
- [ ] Support staged changes only (`git diff --cached`)
- [ ] Support specific commit ranges (`git diff commit1..commit2`)
- [ ] Streaming results for very large diffs

---

**Status**: ✅ Fully Implemented

**Last Updated**: 2025-11-16 22:50

**Key Changes**:
- Added `analyze_all_changes` boolean parameter
- Removed conflicting single-file auto-detection
- Updated tool description to clarify usage
- Enhanced response format with `scope` field
