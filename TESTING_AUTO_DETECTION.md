# Testing Auto-Detection Feature

## What Changed

The MCP server now **automatically detects uncommitted changes** when you query about files. This means:

1. **No manual git diff needed** - The tool calls `git diff` internally
2. **Smart routing** - If uncommitted changes exist, uses diff-based LLM analysis; otherwise uses file-based analysis
3. **Simple queries** - Just ask about the file(s), no need to specify you want diff analysis
4. **Analyze all changes** - NEW: Set `analyze_all_changes=true` to analyze ALL uncommitted changes across the repository

## How to Test

### Prerequisites
✅ MCP server binary rebuilt: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk-check-server`
✅ Databases running: Neo4j (7688), PostgreSQL (5433)
✅ Claude Code configuration updated with GEMINI_API_KEY
✅ Repository: mcp-use (repo_id: 4)

### Test Scenarios

#### Test 1: Uncommitted Changes (Auto-Detection)
1. Navigate to the mcp-use repository in Claude Code
2. Make a small change to a file (e.g., `libraries/python/mcp_use/client/connectors/base.py`)
3. Ask Claude Code: **"What is the risk of my uncommitted changes in libraries/python/mcp_use/client/connectors/base.py?"**

**Expected Behavior:**
- Claude Code should route to `coderisk.get_risk_summary` tool
- Tool internally calls `git diff HEAD -- <file_path>`
- Detects uncommitted changes → uses diff-based LLM analysis
- Returns risk evidence for the modified code blocks only

#### Test 2: No Uncommitted Changes (Fallback)
1. Commit or discard changes from Test 1
2. Ask Claude Code: **"What are the risk factors for libraries/python/mcp_use/client/session.py?"**

**Expected Behavior:**
- Claude Code routes to `coderisk.get_risk_summary` tool
- Tool calls `git diff` → no uncommitted changes found
- Falls back to file-based analysis
- Returns risk evidence for all code blocks in the file

#### Test 3: All Uncommitted Changes (NEW)
1. Make changes to multiple files in the repository
2. Ask Claude Code: **"What is the risk of all my uncommitted changes?"**

**Expected Behavior:**
- Claude Code routes to `coderisk.get_risk_summary` with `analyze_all_changes=true`
- Tool calls `git diff HEAD` (no file specified) to get all changes
- Detects all uncommitted changes → uses diff-based LLM analysis
- Returns risk evidence for ALL modified code blocks across ALL changed files

#### Test 4: Absolute Path with repo_root
1. Make a small change to a file
2. Ask Claude Code from the mcp-use directory: **"What is the risk of my changes in /Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/auth/oauth.py?"**

**Expected Behavior:**
- Tool normalizes absolute path to relative using session context
- Auto-detects uncommitted changes
- Uses diff-based analysis if changes exist

## How to Verify It's Working

### Check the Tool Description
In Claude Code, you can verify the tool is registered correctly:
```
"What MCP tools are available?"
```

Look for:
```
crisk.get_risk_summary - Get risk evidence for a file including ownership,
coupling, and temporal incident data. Automatically detects and analyzes
uncommitted changes when a file_path is provided. Use this tool when the user
asks about 'my changes', 'uncommitted changes', or 'risk of changes' in a file.
```

### Check Server Logs
Monitor `/tmp/crisk-mcp-server.log` for:
- ✅ Diff atomizer created (supports diff-based analysis)
- Tool execution logs showing diff-based vs file-based flow

### Verify Tool Parameters
The tool should accept:
- `file_path` (optional) - will auto-detect uncommitted changes
- `diff_content` (optional) - manual override
- `repo_root` (optional) - for absolute path resolution

## Expected Tool Behavior

### Query Keywords That Should Trigger the Tool:
- "What is the risk of my **uncommitted changes** in X?"
- "What is the risk of my **changes** in X?"
- "Analyze the risk of **my edits** to X"
- "What are the **risk factors** for X?" (generic, works both ways)

### Tool Routing Logic:
1. If `file_path` provided → call `git diff HEAD -- <file_path>`
2. If uncommitted changes found → `diffContent = <auto-detected diff>`
3. If `diffContent` is set → diff-based analysis (LLM extraction)
4. Else → file-based analysis (all blocks in file)

## Known Issues to Watch For

1. **Cache file locking** - If multiple server instances start, bbolt will fail (single writer)
   - Solution: Kill all processes with `pkill -9 -f crisk-check-server`
   - Clear cache: `rm /tmp/crisk-mcp-cache.db`

2. **GEMINI_API_KEY not set** - Diff-based analysis will be disabled
   - Check logs for: "⚠️ GEMINI_API_KEY not set"
   - Verify ~/.claude.json has the API key in env

3. **repo_id hardcoded to 4** - Only works for mcp-use repository
   - Change working directory to mcp-use before testing

## Success Criteria

✅ Claude Code routes diff-based queries to coderisk tool (not analyzing itself)
✅ Tool auto-detects uncommitted changes without manual git diff
✅ Diff-based analysis returns only modified code blocks
✅ File-based analysis works when no uncommitted changes exist
✅ Absolute paths work with repo_root parameter

---

**Status**: Ready for Testing
**Date**: 2025-11-16 22:35
