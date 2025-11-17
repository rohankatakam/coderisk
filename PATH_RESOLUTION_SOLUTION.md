# Path Resolution Solution: Dynamic repo_root from Claude Code Session

## Problem Statement

The MCP server was unable to resolve absolute file paths because:
1. Neo4j stores relative paths (e.g., `libraries/python/mcp_use/client/session.py`)
2. Claude Code passes absolute paths (e.g., `/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/client/session.py`)
3. The MCP server runs as a separate process without access to the user's repository clones

**Failed Approach**: Using `os.Stat()` to walk up the filesystem from the absolute path looking for `.git` directory
- ❌ Checks the **server's** filesystem, not the user's
- ❌ Repository clones don't exist on the server
- ❌ Fundamentally flawed architecture

## Solution: Pass repo_root from Claude Code Session

Instead of trying to detect the repository root server-side, we accept it as a **parameter** from Claude Code, which knows the working directory context.

### Implementation

#### 1. Updated Tool Schema (main.go)
```go
type ToolArgs struct {
    FilePath         string   `json:"file_path"`
    RepoRoot         string   `json:"repo_root,omitempty"` // NEW: from Claude Code session
    DiffContent      string   `json:"diff_content,omitempty"`
    // ... other parameters
}
```

#### 2. Updated IdentityResolver Interface (tools/get_risk_summary.go)
```go
type IdentityResolver interface {
    ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error)
    ResolveHistoricalPathsWithRoot(ctx context.Context, currentPath string, repoRoot string) ([]string, error)
}
```

#### 3. Path Normalization Logic (identity_resolver.go)
```go
func (r *IdentityResolver) ResolveHistoricalPathsWithRoot(ctx context.Context, currentPath string, repoRoot string) ([]string, error) {
    normalizedPath := currentPath

    // Normalize absolute → relative if repo_root provided
    if repoRoot != "" && filepath.IsAbs(currentPath) {
        normalizedPath = NormalizeToRelativePath(currentPath, repoRoot)
    }

    // Use repoRoot for git command working directory
    cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--name-only", "--format=%H", normalizedPath)
    if repoRoot != "" {
        cmd.Dir = repoRoot
    }
    // ... execute git command
}
```

#### 4. Execution Flow (get_risk_summary.go)
```go
func (t *GetRiskSummaryTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    filePath, _ := args["file_path"].(string)
    repoRoot, _ := args["repo_root"].(string) // Extract from args

    var historicalPaths []string
    if repoRoot != "" {
        historicalPaths, err = t.identityResolver.ResolveHistoricalPathsWithRoot(ctx, filePath, repoRoot)
    } else {
        historicalPaths, err = t.identityResolver.ResolveHistoricalPaths(ctx, filePath)
    }
    // ... query graph with normalized paths
}
```

## Usage Examples

### Relative Path (No repo_root needed)
```json
{
  "file_path": "libraries/python/mcp_use/client/session.py"
}
```
- Works if Claude Code is in repository root
- No path normalization needed

### Absolute Path (repo_root required)
```json
{
  "file_path": "/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/auth/oauth.py",
  "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
}
```
- Claude Code provides working directory from session
- MCP server normalizes: `/Users/.../oauth.py` → `libraries/python/mcp_use/auth/oauth.py`
- Git command runs in correct directory: `cmd.Dir = /Users/.../mcp-use`

### Diff-Based Analysis
```json
{
  "diff_content": "diff --git a/src/auth.py ...",
  "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
}
```
- LLM extracts block references from diff
- Paths normalized using repo_root
- Git log --follow runs in correct repository

## Benefits

✅ **Dynamic**: No database storage, no filesystem probing
✅ **Session-Aware**: Leverages Claude Code's working directory knowledge
✅ **Cross-Repository**: Can analyze multiple repos in same session
✅ **Backward Compatible**: Works without repo_root for relative paths
✅ **Git-Aware**: git log --follow runs in correct repository context

## Key Insight

The solution leverages the **separation of concerns**:
- **Claude Code Session**: Knows where repositories are cloned, what directory user is in
- **MCP Server**: Knows how to query graph, normalize paths, run git commands
- **Contract**: `repo_root` parameter bridges the two contexts

This is superior to:
- ❌ Storing paths in database (not dynamic)
- ❌ Detecting via filesystem (wrong process context)
- ❌ Hardcoding paths (not portable)

## Implementation Files

1. [cmd/crisk-check-server/main.go:118](cmd/crisk-check-server/main.go#L118) - Tool args schema
2. [internal/mcp/tools/get_risk_summary.go:19](internal/mcp/tools/get_risk_summary.go#L19) - Interface definition
3. [internal/mcp/tools/get_risk_summary.go:56](internal/mcp/tools/get_risk_summary.go#L56) - Parameter extraction
4. [internal/mcp/identity_resolver.go:87](internal/mcp/identity_resolver.go#L87) - Path resolution with repo_root
5. [internal/mcp/path_utils.go:11](internal/mcp/path_utils.go#L11) - Path normalization utilities

## Testing

The solution can be tested with:

```bash
# Test with relative path (no repo_root)
echo '{"file_path": "libraries/python/mcp_use/client/session.py"}' | crisk-check-server

# Test with absolute path (with repo_root)
echo '{
  "file_path": "/Users/rohankatakam/Documents/brain/mcp-use/libraries/python/mcp_use/auth/oauth.py",
  "repo_root": "/Users/rohankatakam/Documents/brain/mcp-use"
}' | crisk-check-server
```

Expected: Both queries should find the same code blocks after path normalization.

---

**Status**: ✅ Implemented and Documented
**Date**: 2025-11-16 22:06
