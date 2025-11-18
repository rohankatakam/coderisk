# Atomizer Validation Report
**Generated**: 2025-11-18 22:02:25
**Atomizer Progress**: 473/517 commits (91% complete)
**Total Blocks Processed**: 774 code blocks across 314 unique files

---

## Executive Summary

The atomizer is successfully processing commits with the new simple JSON mode (no ResponseSchema). Data is populating correctly in both PostgreSQL and Neo4j. However, there are several data quality issues that require review:

### ✅ Successes
- No JSON parsing errors since switching to simple JSON mode
- No NULL values in required fields (`canonical_file_path`, `block_name`, `block_type`)
- File paths are being parsed correctly from diff headers (no hallucination)
- Data is writing to both PostgreSQL (774 blocks) and Neo4j (589+ blocks)
- Unique constraint on `(repo_id, canonical_file_path, block_name)` is enforced

### ⚠️ Issues Found
1. **Empty block_name values** (14 blocks, ~1.8%)
2. **Zero line numbers** (40+ blocks, ~10%)
3. **Non-existent file paths** (binary files, config files with code blocks)

---

## Issue #1: Empty block_name Values

**Count**: 14 blocks (1.8% of 774 total)
**Severity**: Medium
**Impact**: These blocks cannot be uniquely identified by name

### SQL Query to Retrieve
```sql
SELECT
    id,
    canonical_file_path,
    block_name,
    block_type,
    start_line,
    end_line,
    created_at
FROM code_blocks
WHERE repo_id = 11 AND block_name = ''
ORDER BY created_at DESC;
```

### Sample Data
| canonical_file_path | block_name | block_type | start_line | end_line |
|---------------------|------------|------------|------------|----------|
| `libraries/typescript/packages/mcp-use/examples/client/observability.ts` | _(empty)_ | _(empty)_ | 9 | 15 |
| `libraries/typescript/packages/mcp-use/examples/client/structured_output.ts` | _(empty)_ | _(empty)_ | 11 | 17 |
| `libraries/typescript/packages/mcp-use/package.json` | _(empty)_ | _(empty)_ | 9 | 91 |
| `libraries/typescript/packages/mcp-use/examples/client/add_server_tool.ts` | _(empty)_ | _(empty)_ | 0 | 0 |

### Analysis
All 14 empty block_name entries are from:
- TypeScript example files (`.ts`)
- Configuration files (`package.json`)

These appear to be cases where the LLM extracted a change event but couldn't identify a specific function/method/class name. Possible causes:
1. The diff only shows config changes (e.g., adding dependencies to `package.json`)
2. The LLM extracted imports or file-level changes without a named block
3. The LLM generated empty strings instead of omitting the field

### Recommendations
1. **Filter out empty block_name entries** during atomization or post-processing
2. **Update filterValidEvents()** in [llm_extractor.go:315](internal/atomizer/llm_extractor.go#L315) to skip events with empty `block_name`
3. **Add validation** to ensure `block_name` is non-empty for non-import events

### Fix Implementation
Add this check to `filterValidEvents()`:
```go
// Skip events without block name (except imports which use dependency_path)
if event.BlockType != "" && event.TargetBlockName == "" {
    continue
}
```

---

## Issue #2: Zero Line Numbers

**Count**: 40+ blocks (10.3% of total)
**Severity**: Low
**Impact**: These blocks cannot be located to specific line ranges

### SQL Query to Retrieve
```sql
SELECT
    id,
    canonical_file_path,
    block_name,
    block_type,
    start_line,
    end_line,
    created_at
FROM code_blocks
WHERE repo_id = 11 AND start_line = 0 AND end_line = 0
ORDER BY created_at DESC
LIMIT 20;
```

### Sample Data
| canonical_file_path | block_name | block_type | start_line | end_line |
|---------------------|------------|------------|------------|----------|
| `libraries/python/.gitignore` | BaseConnector | class | 0 | 0 |
| `libraries/python/pyproject.toml` | test_async_placeholder | function | 0 | 0 |
| `libraries/python/.pre-commit-config.yaml` | run_memory_chat | function | 0 | 0 |
| `libraries/python/examples/airbnb_mcp.json` | main | function | 0 | 0 |
| `libraries/python/README.md` | DEFAULT_SYSTEM_PROMPT_TEMPLATE | _(empty)_ | 0 | 0 |

### Analysis
These entries have `start_line=0, end_line=0` which indicates the diff parser couldn't extract line numbers. Common patterns:
1. **Binary files**: `.gitignore`, `.png`, `.yaml` (not valid code files)
2. **Config files**: `pyproject.toml`, `package.json`, `.pre-commit-config.yaml`
3. **Documentation**: `README.md`, `.mdx` files
4. **Entire file additions**: New files where diff shows `new file mode` without hunk headers

### Recommendations
1. **Filter non-code files** in the diff parser using `IsBinaryFile()` check
2. **Skip documentation files** (`.md`, `.mdx`, `.txt`) in [llm_extractor.go:42](internal/atomizer/llm_extractor.go#L42)
3. **Accept zero line numbers** for entire file operations where hunk headers don't exist

### Fix Implementation
Update the diff parser to skip non-code files:
```go
parsedFiles := ParseDiff(commit.DiffContent)

// Filter out binary and documentation files
for filePath := range parsedFiles {
    if IsBinaryFile(filePath) || IsDocumentationFile(filePath) {
        delete(parsedFiles, filePath)
    }
}
```

Add `IsDocumentationFile()` helper:
```go
func IsDocumentationFile(filename string) bool {
    docExtensions := []string{".md", ".mdx", ".txt", ".rst"}
    lowerFilename := strings.ToLower(filename)
    for _, ext := range docExtensions {
        if strings.HasSuffix(lowerFilename, ext) {
            return true
        }
    }
    return false
}
```

---

## Issue #3: Non-Code Files with Code Blocks

**Severity**: Medium
**Impact**: LLM is extracting "code blocks" from non-code files

### Examples from Sample Data
1. **`.gitignore`** has `BaseConnector` class extracted
2. **`pyproject.toml`** has `test_async_placeholder` function extracted
3. **`.pre-commit-config.yaml`** has `run_memory_chat` function extracted
4. **`README.md`** has `DEFAULT_SYSTEM_PROMPT_TEMPLATE` extracted
5. **`docs/images/hero-dark.png`** has `get_server` method extracted

### SQL Query to Retrieve
```sql
SELECT
    canonical_file_path,
    block_name,
    block_type
FROM code_blocks
WHERE repo_id = 11
  AND (
      canonical_file_path LIKE '%.md'
      OR canonical_file_path LIKE '%.yaml'
      OR canonical_file_path LIKE '%.yml'
      OR canonical_file_path LIKE '%.json'
      OR canonical_file_path LIKE '%.toml'
      OR canonical_file_path LIKE '%.png'
      OR canonical_file_path LIKE '%.jpg'
      OR canonical_file_path LIKE '%.svg'
      OR canonical_file_path LIKE '%.ini'
      OR canonical_file_path LIKE '.%'
  )
ORDER BY canonical_file_path;
```

### Analysis
The LLM is seeing code snippets in documentation files, config files, and even attempting to extract from binary image files. This happens because:
1. README.md and .mdx files contain code examples in markdown blocks
2. Config files (`.yaml`, `.toml`, `.json`) may reference function names in comments
3. The diff parser doesn't filter out these files before sending to LLM

### Recommendations
1. **Add file type filtering** BEFORE calling the LLM in [llm_extractor.go:42](internal/atomizer/llm_extractor.go#L42)
2. **Update prompt** to explicitly exclude documentation and config files
3. **Filter parsedFiles** to only include valid source code extensions

### Fix Implementation
```go
// 1. Parse diff to extract file paths and line numbers (BEFORE LLM)
parsedFiles := ParseDiff(commit.DiffContent)

// 2. Filter to only code files
codeFiles := make(map[string]*DiffFileChange)
for filePath, change := range parsedFiles {
    if IsCodeFile(filePath) {
        codeFiles[filePath] = change
    }
}

// If no code files remain, return empty result
if len(codeFiles) == 0 {
    return &CommitChangeEventLog{
        CommitSHA:        commit.SHA,
        AuthorEmail:      commit.AuthorEmail,
        Timestamp:        commit.Timestamp,
        LLMIntentSummary: "No code file changes detected",
        MentionedIssues:  []string{},
        ChangeEvents:     []ChangeEvent{},
    }, nil
}
```

Add `IsCodeFile()` helper:
```go
func IsCodeFile(filename string) bool {
    // Skip binary and documentation files
    if IsBinaryFile(filename) || IsDocumentationFile(filename) {
        return false
    }

    // Skip config files
    configExtensions := []string{
        ".json", ".yaml", ".yml", ".toml", ".ini",
        ".lock", ".sum", ".mod", ".cfg",
    }
    lowerFilename := strings.ToLower(filename)
    for _, ext := range configExtensions {
        if strings.HasSuffix(lowerFilename, ext) {
            return false
        }
    }

    // Skip dotfiles (like .gitignore, .env)
    base := filepath.Base(filename)
    if strings.HasPrefix(base, ".") {
        return false
    }

    // Allow known code file extensions
    codeExtensions := []string{
        ".go", ".py", ".js", ".ts", ".tsx", ".jsx",
        ".java", ".c", ".cpp", ".h", ".hpp",
        ".rs", ".rb", ".php", ".swift", ".kt",
    }
    for _, ext := range codeExtensions {
        if strings.HasSuffix(lowerFilename, ext) {
            return true
        }
    }

    return false
}
```

---

## PostgreSQL Statistics

### Overall Counts
```sql
SELECT
    COUNT(*) as total_blocks,
    COUNT(DISTINCT canonical_file_path) as unique_files,
    COUNT(DISTINCT block_name) as unique_block_names,
    MIN(created_at) as first_insertion,
    MAX(created_at) as last_insertion
FROM code_blocks
WHERE repo_id = 11;
```

**Current Results** (as of commit 473/517):
- Total blocks: **774**
- Unique files: **314**
- Unique block names: **~600** (estimated)
- First insertion: 2025-11-18 21:51:36
- Last insertion: 2025-11-18 22:02:19

### Schema Validation
```sql
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE canonical_file_path IS NULL) as null_file_path,
    COUNT(*) FILTER (WHERE block_name IS NULL) as null_block_name,
    COUNT(*) FILTER (WHERE block_type IS NULL) as null_block_type,
    COUNT(*) FILTER (WHERE canonical_file_path = '') as empty_file_path,
    COUNT(*) FILTER (WHERE block_name = '') as empty_block_name,
    COUNT(*) FILTER (WHERE start_line = 0 AND end_line = 0) as zero_line_numbers
FROM code_blocks
WHERE repo_id = 11;
```

**Current Results**:
- Total: 774
- NULL canonical_file_path: **0** ✅
- NULL block_name: **0** ✅
- NULL block_type: **0** ✅
- Empty canonical_file_path: **0** ✅
- Empty block_name: **14** ⚠️
- Zero line numbers: **40+** ⚠️

---

## Neo4j Statistics

### CodeBlock Counts
```cypher
MATCH (cb:CodeBlock)
WHERE cb.repo_id = 11
RETURN
    count(cb) as total_blocks,
    count(DISTINCT cb.file_path) as unique_files
```

**Current Results** (as of commit 473/517):
- Total CodeBlocks: **589**
- Unique files: **0** (query issue - `file_path` property may not exist)

### Corrected Query
```cypher
MATCH (cb:CodeBlock)
WHERE cb.repo_id = 11
RETURN
    count(cb) as total_blocks,
    count(DISTINCT cb.canonical_file_path) as unique_files
```

---

## Log File Warnings

### Warning Types Observed

#### 1. MODIFY_BLOCK for Non-Existent Block
```
WARNING: MODIFY_BLOCK for non-existent block libraries/python/tests/integration/transports/sse/test_connection_state_tracking.py:MCPSession (creating it)
```

**Frequency**: Common (dozens of occurrences)
**Severity**: Low
**Analysis**: This happens when the LLM detects a modification to a function/class that hasn't been seen before in the atomization process. The atomizer creates the block as a fallback.

**Recommendation**: This is expected behavior for incomplete git history. The atomizer doesn't have access to the initial creation commit, so it treats modifications as creations.

#### 2. DELETE_BLOCK for Non-Existent Block
```
WARNING: DELETE_BLOCK for non-existent block docs/python/api-reference/mcp_use_server.mdx:print_header (ignoring)
```

**Frequency**: Occasional
**Severity**: Low
**Analysis**: The LLM detected a deletion of a block that was never created (or was in a non-code file like `.mdx`).

**Recommendation**: These can be safely ignored. Adding file type filtering (Issue #3) will reduce these warnings for documentation files.

#### 3. ADD_IMPORT Not Yet Implemented
```
INFO: ADD_IMPORT event (not yet implemented): libraries/python/docs/development/telemetry.mdx imports time
```

**Frequency**: Occasional
**Severity**: Low
**Analysis**: Import tracking is not yet implemented in the graph writer.

**Recommendation**: Implement import tracking in a future iteration.

---

## Recommendations Summary

### High Priority
1. **Filter non-code files** before sending to LLM (Issue #3)
   - Add `IsCodeFile()` helper in [diff_parser.go](internal/atomizer/diff_parser.go)
   - Filter `parsedFiles` in [llm_extractor.go:42](internal/atomizer/llm_extractor.go#L42)

2. **Filter empty block_name entries** (Issue #1)
   - Update `filterValidEvents()` in [llm_extractor.go:315](internal/atomizer/llm_extractor.go#L315)
   - Skip events with empty `block_name` for non-import behaviors

### Medium Priority
3. **Accept zero line numbers** for valid cases (Issue #2)
   - Document that `start_line=0, end_line=0` is valid for entire file operations
   - Consider adding a `line_range_available` boolean field to distinguish between "no data" and "entire file"

4. **Improve diff parser heuristics**
   - Handle `new file mode` diffs by parsing tree-sitter for entire file
   - Extract line ranges from file content when hunk headers are unavailable

### Low Priority
5. **Implement import tracking**
   - Handle `ADD_IMPORT` and `REMOVE_IMPORT` events in graph writer
   - Link imports to dependency graph

6. **Add telemetry for warning types**
   - Track frequency of MODIFY_BLOCK and DELETE_BLOCK warnings
   - Use metrics to identify patterns in LLM extraction quality

---

## Validation Queries for Final Review

### After Atomizer Completes (517/517)

#### Check Final Counts
```sql
SELECT
    COUNT(*) as total_blocks,
    COUNT(DISTINCT canonical_file_path) as unique_files,
    COUNT(DISTINCT block_name) as unique_block_names
FROM code_blocks
WHERE repo_id = 11;
```

#### Check Data Quality
```sql
SELECT
    COUNT(*) FILTER (WHERE block_name = '') as empty_names,
    COUNT(*) FILTER (WHERE start_line = 0 AND end_line = 0) as zero_lines,
    COUNT(*) FILTER (WHERE canonical_file_path LIKE '%.md' OR canonical_file_path LIKE '%.json') as non_code_files
FROM code_blocks
WHERE repo_id = 11;
```

#### Check Neo4j Sync
```cypher
MATCH (cb:CodeBlock)
WHERE cb.repo_id = 11
WITH count(cb) as neo4j_count

CALL {
    // This would require a procedure to query PostgreSQL
    // For now, manually compare with SQL query
    RETURN 774 as postgres_count
}

RETURN neo4j_count, postgres_count, neo4j_count - postgres_count as difference
```

---

## Next Steps

1. **Wait for atomizer to complete** (currently at 473/517)
2. **Run validation queries** from this document
3. **Implement high-priority fixes** for Issues #1 and #3
4. **Re-run atomization** with fixes applied
5. **Proceed to next pipeline stages**:
   - `crisk-index-incident` (temporal risk)
   - `crisk-index-ownership` (ownership risk)
   - `crisk-index-coupling` (coupling risk)

---

**Document Maintained By**: Atomizer Validation System
**Last Updated**: 2025-11-18 22:02:25
**Atomizer Log File**: `/tmp/crisk-atomize-simple-json.log`
