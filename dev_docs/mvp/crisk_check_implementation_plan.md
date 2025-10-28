# crisk check Implementation Plan

**Date:** 2025-10-28
**Purpose:** Design implementation strategy for crisk check with file resolution and agent-based query execution
**Status:** DRAFT - For Review

---

## Executive Summary

This document proposes the implementation approach for `crisk check` addressing two critical challenges:

1. **File Resolution Problem**: Bridging historical GitHub data (filenames from 90-day commits) with current local changes
2. **Query Selection Problem**: Choosing which graph queries to execute for risk assessment

### Recommended Approach

**File Resolution:** Git-native solution using `git log --follow` (Level 1-2 from file_resolution_strategy.md)
**Query Execution:** Stateless agent with structured tool calls for dynamic graph exploration

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [File Resolution Strategy](#file-resolution-strategy)
3. [Agent-Based Query Execution](#agent-based-query-execution)
4. [Implementation Phases](#implementation-phases)
5. [Integration with Existing Code](#integration-with-existing-code)
6. [Performance Targets](#performance-targets)
7. [Testing Strategy](#testing-strategy)

---

## Problem Statement

### Challenge 1: File Resolution

**Context:**
- `crisk check` receives: `src/shared/config/settings.py` (current path)
- Neo4j contains: `shared/config/settings.py` (historical path from 90-day commits)
- Need: Connect current files to historical graph data

**Why this matters:**
Without file resolution, we cannot:
- Find who owns the code (requires historical commits)
- Identify co-change patterns (requires historical file relationships)
- Discover incident history (requires linking to issues via commits)

### Challenge 2: Query Selection

**Context:**
- We have 10+ predefined Cypher queries (ownership, blast radius, co-change, incidents, etc.)
- Different files need different queries depending on risk signals
- Current approach: Execute all queries for every file (slow, expensive)

**Why this matters:**
- Need intelligent query selection based on context
- Should explore the graph dynamically, not follow rigid query patterns
- Want flexibility to add new queries without hardcoding logic

---

## File Resolution Strategy

### Approach: Git-Native (Levels 1-2)

Based on [file_resolution_strategy.md](./file_resolution/file_resolution_strategy.md), we use a cascading resolution approach:

#### Level 1: Exact Match (100% confidence)

```go
func (r *FileResolver) ExactMatch(currentPath string) ([]string, error) {
    // Check if current path exists in graph
    results, err := r.graphClient.QueryWithParams(ctx, `
        MATCH (f:File {path: $path})
        RETURN f.path as path
    `, map[string]interface{}{
        "path": currentPath,
    })

    if len(results) > 0 {
        return []string{currentPath}, nil
    }

    return nil, nil // No exact match, continue to Level 2
}
```

**Use case:** File hasn't moved in 90 days
**Coverage:** ~20% of files

---

#### Level 2: Git Log --follow (95% confidence)

```go
func (r *FileResolver) GitFollowMatch(currentPath string) ([]string, error) {
    // Execute git log --follow to get all historical paths
    cmd := exec.Command("git", "log", "--follow", "--name-only",
                        "--pretty=format:", "--", currentPath)
    cmd.Dir = r.repoPath

    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("git log --follow failed: %w", err)
    }

    // Parse unique file paths from output
    historicalPaths := parseUniquePaths(output)

    // Check which historical paths exist in our graph
    results, err := r.graphClient.QueryWithParams(ctx, `
        MATCH (f:File)
        WHERE f.path IN $paths
        RETURN f.path as path
    `, map[string]interface{}{
        "paths": historicalPaths,
    })

    var matchedPaths []string
    for _, row := range results {
        if path, ok := row["path"].(string); ok {
            matchedPaths = append(matchedPaths, path)
        }
    }

    return matchedPaths, nil
}
```

**Pros:**
- Git's native rename detection (very accurate)
- Handles rename chains automatically
- No external dependencies beyond git
- Free and fast (<50ms per file)

**Use case:** File has been renamed/moved
**Coverage:** ~70% of files (cumulative with Level 1: 90%)

---

#### Implementation: FileResolver Component

```go
package git

import (
    "context"
    "os/exec"
    "strings"
)

// FileResolver bridges current file paths to historical graph data
type FileResolver struct {
    repoPath    string
    graphClient GraphQueryer
}

type GraphQueryer interface {
    QueryWithParams(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
}

type FileMatch struct {
    HistoricalPath string
    Confidence     float64
    Method         string // "exact", "git-follow"
}

// Resolve finds historical paths for a current file path
func (r *FileResolver) Resolve(ctx context.Context, currentPath string) ([]FileMatch, error) {
    // Level 1: Exact match
    if matches, err := r.ExactMatch(ctx, currentPath); err == nil && len(matches) > 0 {
        return []FileMatch{{
            HistoricalPath: matches[0],
            Confidence:     1.0,
            Method:         "exact",
        }}, nil
    }

    // Level 2: Git log --follow
    historicalPaths, err := r.GitFollowMatch(currentPath)
    if err != nil {
        return nil, err
    }

    var matches []FileMatch
    for _, path := range historicalPaths {
        matches = append(matches, FileMatch{
            HistoricalPath: path,
            Confidence:     0.95,
            Method:         "git-follow",
        })
    }

    return matches, nil
}

// BatchResolve resolves multiple files in parallel
func (r *FileResolver) BatchResolve(ctx context.Context, currentPaths []string) (map[string][]FileMatch, error) {
    results := make(map[string][]FileMatch)

    // Execute git log --follow for all files in parallel
    // (git is fast enough to run concurrently)
    var wg sync.WaitGroup
    resultsChan := make(chan struct {
        path    string
        matches []FileMatch
        err     error
    }, len(currentPaths))

    for _, path := range currentPaths {
        wg.Add(1)
        go func(p string) {
            defer wg.Done()
            matches, err := r.Resolve(ctx, p)
            resultsChan <- struct {
                path    string
                matches []FileMatch
                err     error
            }{p, matches, err}
        }(path)
    }

    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    for result := range resultsChan {
        if result.err == nil {
            results[result.path] = result.matches
        }
    }

    return results, nil
}

func parseUniquePaths(output []byte) []string {
    lines := strings.Split(string(output), "\n")
    seen := make(map[string]bool)
    var unique []string

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !seen[line] {
            seen[line] = true
            unique = append(unique, line)
        }
    }

    return unique
}
```

---

### Why NOT Level 3+ (Fuzzy/LLM)

**Decision:** Skip fuzzy matching and LLM-based resolution for MVP

**Rationale:**
1. **Coverage sufficient:** Level 1-2 covers 90% of files
2. **Complexity:** Fuzzy matching adds heuristics that may introduce false positives
3. **Cost:** LLM resolution adds latency (2-3s) and cost ($0.01-0.02 per call)
4. **Fallback strategy:** For the 10% we can't resolve, simply skip historical analysis

**Fallback behavior:**
```go
if len(matches) == 0 {
    // File has no historical data - either new or can't resolve
    return &RiskAssessment{
        RiskLevel:  LOW,
        Confidence: 0.8,
        Message:    "New file or no historical data available",
        Recommendations: []string{
            "Request code review from team lead",
            "Ensure adequate test coverage",
        },
    }
}
```

---

## Agent-Based Query Execution

### Approach: Stateless Agent with Tool Calls

Inspired by [12-factor-agents](../12-factor-agents-main/), we use:
- **Factor 1:** Natural language to tool calls (agent decides what to query)
- **Factor 4:** Tools are structured outputs (Cypher queries as tools)
- **Factor 8:** Own your control flow (we orchestrate the agent loop)
- **Factor 12:** Stateless reducer (agent is pure function)

---

### Architecture: RiskInvestigator Agent

```go
package agent

import (
    "context"
    "encoding/json"
)

// RiskInvestigator uses an LLM to dynamically explore the graph
type RiskInvestigator struct {
    llmClient   LLMClient
    graphClient GraphClient
    pgClient    PostgresClient
}

// Tool represents an action the agent can take
type Tool struct {
    Name        string
    Description string
    Parameters  map[string]ToolParameter
}

type ToolParameter struct {
    Type        string
    Description string
    Required    bool
}

// Available tools for the agent
var RiskInvestigationTools = []Tool{
    {
        Name: "query_ownership",
        Description: "Find who owns a file based on commit history. Returns top 3 contributors with commit counts.",
        Parameters: map[string]ToolParameter{
            "file_paths": {
                Type:        "array",
                Description: "Array of file paths to query (current + historical)",
                Required:    true,
            },
        },
    },
    {
        Name: "query_cochange_partners",
        Description: "Find files that frequently change together with the target file. Useful for detecting incomplete changes.",
        Parameters: map[string]ToolParameter{
            "file_paths": {
                Type:        "array",
                Description: "Array of file paths to query",
                Required:    true,
            },
            "frequency_threshold": {
                Type:        "number",
                Description: "Minimum co-change frequency (0.0-1.0). Default: 0.5",
                Required:    false,
            },
        },
    },
    {
        Name: "query_incident_history",
        Description: "Find issues and commits that fixed bugs in this file. Returns incident patterns.",
        Parameters: map[string]ToolParameter{
            "file_paths": {
                Type:        "array",
                Description: "Array of file paths to query",
                Required:    true,
            },
            "days_back": {
                Type:        "number",
                Description: "How many days back to search. Default: 180",
                Required:    false,
            },
        },
    },
    {
        Name: "query_blast_radius",
        Description: "Find files that depend on the target file. Shows potential impact of changes.",
        Parameters: map[string]ToolParameter{
            "file_path": {
                Type:        "string",
                Description: "File path to check dependencies for",
                Required:    true,
            },
        },
    },
    {
        Name: "get_commit_patch",
        Description: "Retrieve the actual code diff from a specific commit. Useful for understanding what changed.",
        Parameters: map[string]ToolParameter{
            "commit_sha": {
                Type:        "string",
                Description: "Git commit SHA",
                Required:    true,
            },
        },
    },
    {
        Name: "query_recent_commits",
        Description: "Get the last N commits that modified the file. Shows recent activity.",
        Parameters: map[string]ToolParameter{
            "file_paths": {
                Type:        "array",
                Description: "Array of file paths to query",
                Required:    true,
            },
            "limit": {
                Type:        "number",
                Description: "Number of commits to return. Default: 5",
                Required:    false,
            },
        },
    },
    {
        Name: "finish_investigation",
        Description: "Complete the investigation and return final risk assessment.",
        Parameters: map[string]ToolParameter{
            "risk_level": {
                Type:        "string",
                Description: "LOW, MEDIUM, HIGH, or CRITICAL",
                Required:    true,
            },
            "confidence": {
                Type:        "number",
                Description: "Confidence score 0.0-1.0",
                Required:    true,
            },
            "reasoning": {
                Type:        "string",
                Description: "Detailed explanation of the risk assessment",
                Required:    true,
            },
            "recommendations": {
                Type:        "array",
                Description: "List of specific actions to mitigate risk",
                Required:    true,
            },
        },
    },
}
```

---

### Agent Loop: Stateless Reducer Pattern

```go
type InvestigationState struct {
    FilePath          string
    HistoricalPaths   []string
    GitDiff           string
    ConversationHistory []Message
    ToolCallHistory   []ToolCall
    Phase1Data        *Phase1Data // Baseline metrics
}

type Message struct {
    Role    string // "system", "user", "assistant", "tool"
    Content string
    ToolCallID string // For tool responses
}

type ToolCall struct {
    ID         string
    ToolName   string
    Parameters map[string]interface{}
    Result     interface{}
    Error      error
}

// Investigate performs risk analysis using agent loop
func (inv *RiskInvestigator) Investigate(ctx context.Context, state *InvestigationState) (*RiskAssessment, error) {
    maxIterations := 5 // Safety limit

    for iteration := 0; iteration < maxIterations; iteration++ {
        // Build prompt with conversation history + available tools
        prompt := inv.buildPrompt(state)

        // Ask LLM for next action (tool call)
        response, err := inv.llmClient.ChatCompletion(ctx, ChatRequest{
            Messages: append(state.ConversationHistory, Message{
                Role:    "user",
                Content: prompt,
            }),
            Tools: RiskInvestigationTools,
        })
        if err != nil {
            return nil, fmt.Errorf("LLM request failed: %w", err)
        }

        // Append assistant message to history
        state.ConversationHistory = append(state.ConversationHistory, Message{
            Role:    "assistant",
            Content: response.Content,
        })

        // Check if agent wants to finish
        if response.ToolCall != nil && response.ToolCall.Name == "finish_investigation" {
            // Agent has completed investigation
            return inv.buildFinalAssessment(response.ToolCall.Parameters)
        }

        // Execute tool call
        if response.ToolCall != nil {
            result, err := inv.executeTool(ctx, response.ToolCall, state)

            // Record tool call
            toolCall := ToolCall{
                ID:         response.ToolCall.ID,
                ToolName:   response.ToolCall.Name,
                Parameters: response.ToolCall.Parameters,
                Result:     result,
                Error:      err,
            }
            state.ToolCallHistory = append(state.ToolCallHistory, toolCall)

            // Add tool result to conversation
            state.ConversationHistory = append(state.ConversationHistory, Message{
                Role:       "tool",
                Content:    formatToolResult(result, err),
                ToolCallID: response.ToolCall.ID,
            })

            // Continue loop with updated context
            continue
        }

        // No tool call - agent might be confused
        return nil, fmt.Errorf("agent did not call a tool or finish investigation")
    }

    // Safety limit hit - force completion
    return inv.emergencyAssessment(state), nil
}
```

---

### System Prompt: Investigation Guide

```go
func (inv *RiskInvestigator) buildPrompt(state *InvestigationState) string {
    return fmt.Sprintf(`You are a risk assessment investigator analyzing code changes.

**Your Task:**
Determine the risk level (LOW, MEDIUM, HIGH, CRITICAL) for changes to: %s

**Context You Have:**
1. File path (current): %s
2. Historical paths (from git): %s
3. Git diff:
%s

4. Baseline Metrics (Phase 1):
   - Coupling: %d dependencies
   - Co-change frequency: %.2f
   - Recent incidents: %d

**Your Tools:**
- query_ownership: Find who owns the code
- query_cochange_partners: Find files that usually change together
- query_incident_history: Find past bugs/incidents
- query_blast_radius: Find dependent files
- get_commit_patch: Read actual code changes
- query_recent_commits: See recent activity
- finish_investigation: Return final assessment

**Investigation Strategy:**
1. Start by understanding ownership (who knows this code?)
2. Check incident history (has this file caused problems before?)
3. If high co-change detected, verify all partners are updated
4. If high blast radius, assess downstream impact
5. Use commit patches to understand similar past changes

**When to escalate risk:**
- HIGH: Incident history + large change + no recent owner activity
- CRITICAL: Multiple incidents + incomplete co-change + high blast radius

**Output:**
Call tools to gather evidence, then call finish_investigation with your assessment.
Be concise in your reasoning. Focus on actionable recommendations.`,
        state.FilePath,
        state.FilePath,
        strings.Join(state.HistoricalPaths, ", "),
        truncateDiff(state.GitDiff, 500),
        state.Phase1Data.BlastRadius,
        state.Phase1Data.CoChangePartners[0].Frequency,
        state.Phase1Data.IncidentCount,
    )
}
```

---

### Tool Execution: Graph Query Adapters

```go
func (inv *RiskInvestigator) executeTool(ctx context.Context, toolCall *ToolCall, state *InvestigationState) (interface{}, error) {
    switch toolCall.Name {
    case "query_ownership":
        filePaths := toolCall.Parameters["file_paths"].([]interface{})
        paths := make([]string, len(filePaths))
        for i, p := range filePaths {
            paths[i] = p.(string)
        }

        return inv.graphClient.QueryOwnership(ctx, paths)

    case "query_cochange_partners":
        filePaths := toolCall.Parameters["file_paths"].([]interface{})
        threshold := 0.5
        if t, ok := toolCall.Parameters["frequency_threshold"].(float64); ok {
            threshold = t
        }

        paths := make([]string, len(filePaths))
        for i, p := range filePaths {
            paths[i] = p.(string)
        }

        return inv.graphClient.QueryCoChangePartners(ctx, paths, threshold)

    case "query_incident_history":
        filePaths := toolCall.Parameters["file_paths"].([]interface{})
        daysBack := 180
        if d, ok := toolCall.Parameters["days_back"].(float64); ok {
            daysBack = int(d)
        }

        paths := make([]string, len(filePaths))
        for i, p := range filePaths {
            paths[i] = p.(string)
        }

        return inv.graphClient.QueryIncidentHistory(ctx, paths, daysBack)

    case "query_blast_radius":
        filePath := toolCall.Parameters["file_path"].(string)
        return inv.graphClient.QueryBlastRadius(ctx, filePath)

    case "get_commit_patch":
        commitSHA := toolCall.Parameters["commit_sha"].(string)
        return inv.pgClient.GetCommitPatch(ctx, commitSHA)

    case "query_recent_commits":
        filePaths := toolCall.Parameters["file_paths"].([]interface{})
        limit := 5
        if l, ok := toolCall.Parameters["limit"].(float64); ok {
            limit = int(l)
        }

        paths := make([]string, len(filePaths))
        for i, p := range filePaths {
            paths[i] = p.(string)
        }

        return inv.graphClient.QueryRecentCommits(ctx, paths, limit)

    default:
        return nil, fmt.Errorf("unknown tool: %s", toolCall.Name)
    }
}
```

---

## Implementation Phases

### Phase 1: File Resolution (Week 1)

**Goal:** Resolve current file paths to historical graph paths

**Tasks:**
1. Implement `FileResolver` component in `internal/git/`
   - ExactMatch() method
   - GitFollowMatch() method
   - BatchResolve() for multiple files

2. Add unit tests
   - Test with renamed files
   - Test with unchanged files
   - Test with deleted files

3. Integration test with real Neo4j graph
   - Use omnara test repository
   - Verify historical path discovery

**Success Criteria:**
- ✅ 90%+ files resolve to historical paths
- ✅ Resolution time < 50ms per file (batched)
- ✅ Tests pass with various rename scenarios

---

### Phase 2: Agent Infrastructure (Week 2)

**Goal:** Build agent framework without LLM integration

**Tasks:**
1. Create `internal/agent/investigator.go`
   - Define Tool types
   - Implement InvestigationState
   - Build prompt templates

2. Implement tool execution layer
   - Graph query adapters
   - PostgreSQL patch retrieval
   - Tool result formatting

3. Add mock LLM for testing
   - Simulate tool call responses
   - Test agent loop without API costs

**Success Criteria:**
- ✅ Agent loop executes without LLM
- ✅ All 7 tools executable
- ✅ State tracks conversation history

---

### Phase 3: LLM Integration (Week 3)

**Goal:** Connect agent to OpenAI API

**Tasks:**
1. Implement LLM client in `internal/llm/client.go`
   - Support tool calling (function calling API)
   - Handle streaming responses (optional)
   - Retry logic for rate limits

2. Connect agent to LLM
   - Build proper message format for tool calls
   - Parse tool call responses
   - Handle errors gracefully

3. End-to-end testing
   - Test with real files from omnara
   - Measure accuracy of risk assessments
   - Tune system prompt

**Success Criteria:**
- ✅ Agent completes investigation in 3-5 tool calls
- ✅ Risk assessment matches human judgment
- ✅ Total time < 5 seconds (including LLM latency)

---

### Phase 4: Integration with crisk check (Week 4)

**Goal:** Replace current check logic with file resolution + agent

**Tasks:**
1. Update `cmd/crisk/check.go`
   - Call FileResolver for each file
   - Pass resolved paths to agent
   - Format agent output for CLI

2. Maintain backward compatibility
   - Keep Phase 1 baseline metrics
   - Only invoke agent for HIGH risk files
   - Graceful fallback if LLM unavailable

3. Performance optimization
   - Parallel file resolution
   - Concurrent agent investigations
   - Cache tool call results

**Success Criteria:**
- ✅ `crisk check` works with file resolution
- ✅ Agent investigation only for high-risk files
- ✅ Total check time < 3 seconds per file

---

## Integration with Existing Code

### Current Check Flow

```
cmd/crisk/check.go:
  1. Get changed files (git status/diff)
  2. Resolve to absolute paths (resolveFilePaths)
  3. Run Phase 1 (metrics.CalculatePhase1WithConfig)
  4. If high risk → escalate to Phase 2
     - Create SimpleInvestigator (existing)
     - Run LLM analysis
  5. Format and display results
```

### New Check Flow with File Resolution

```
cmd/crisk/check.go:
  1. Get changed files (git status/diff)

  2. FILE RESOLUTION (NEW)
     - FileResolver.BatchResolve(currentPaths)
     - Returns map[currentPath][]historicalPaths

  3. Run Phase 1 with RESOLVED paths
     - Use historicalPaths in graph queries
     - Metrics now include full 90-day history

  4. If high risk → escalate to Phase 2
     - Create RiskInvestigator (NEW agent)
     - Pass both currentPath AND historicalPaths
     - Agent explores graph dynamically

  5. Format and display results
     - Show which historical paths were used
     - Indicate confidence from resolution
```

---

### Code Changes Required

#### 1. Update Collector to Accept Multiple Paths

**File:** `internal/risk/collector.go`

**Change:** Queries should accept `[]string` instead of `string` for file paths

```go
// OLD
func (c *Collector) queryOwnership(ctx context.Context, filePath string, data *Phase1Data) error {
    results, err := c.graphBackend.QueryWithParams(ctx, QueryOwnership, map[string]interface{}{
        "filePath": filePath,
    })
    // ...
}

// NEW
func (c *Collector) queryOwnership(ctx context.Context, filePaths []string, data *Phase1Data) error {
    results, err := c.graphBackend.QueryWithParams(ctx, QueryOwnership, map[string]interface{}{
        "filePaths": filePaths, // Query now uses IN clause
    })
    // ...
}
```

#### 2. Update Cypher Queries

**File:** `internal/risk/queries.go`

```go
// OLD
const QueryOwnership = `
    MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $filePath})
    ...
`

// NEW
const QueryOwnership = `
    MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File)
    WHERE f.path IN $filePaths
    ...
`
```

#### 3. Add FileResolver to Check Command

**File:** `cmd/crisk/check.go`

```go
// After getting changed files
files, err := git.GetChangedFiles()

// NEW: Resolve files to historical paths
resolver := git.NewFileResolver(repoPath, neo4jClient)
resolvedFilesMap, err := resolver.BatchResolve(ctx, files)
if err != nil {
    return fmt.Errorf("file resolution failed: %w", err)
}

// For each file
for _, file := range files {
    matches := resolvedFilesMap[file]

    if len(matches) == 0 {
        // New file - no historical data
        fmt.Printf("ℹ️  %s: New file (no historical data)\n", file)
        continue
    }

    // Extract historical paths for queries
    historicalPaths := make([]string, len(matches))
    for i, match := range matches {
        historicalPaths[i] = match.HistoricalPath
    }

    // Run Phase 1 with ALL paths (current + historical)
    allPaths := append([]string{file}, historicalPaths...)
    adaptiveResult, err := metrics.CalculatePhase1WithPaths(ctx, neo4jClient, repoID, allPaths, riskConfig)

    // ...
}
```

---

## Performance Targets

| Operation | Target | Current | Notes |
|-----------|--------|---------|-------|
| File resolution (per file) | <50ms | N/A | Using git log --follow |
| File resolution (batch 10 files) | <200ms | N/A | Parallel execution |
| Agent investigation (total) | <5s | ~3s | Including LLM latency |
| Agent tool calls (per call) | <500ms | N/A | Graph query + format |
| Total crisk check (low risk) | <500ms | ~300ms | No agent, just Phase 1 |
| Total crisk check (high risk) | <6s | ~3s | Phase 1 + agent |

**Performance assumptions:**
- Neo4j graph queries: 20-100ms each
- OpenAI API latency: 1-2s per call
- Agent makes 3-5 tool calls on average
- Git operations: <50ms per file

---

## Testing Strategy

### Unit Tests

#### FileResolver Tests
```go
func TestFileResolver_ExactMatch(t *testing.T) {
    // Test case: File exists in graph with exact path
}

func TestFileResolver_GitFollowMatch(t *testing.T) {
    // Test case: File was renamed, git log --follow finds it
}

func TestFileResolver_NoMatch(t *testing.T) {
    // Test case: New file, no historical data
}

func TestFileResolver_MultipleRenames(t *testing.T) {
    // Test case: File renamed multiple times
}
```

#### Agent Tests
```go
func TestRiskInvestigator_OwnershipQuery(t *testing.T) {
    // Test: Agent calls query_ownership tool
}

func TestRiskInvestigator_CompleteInvestigation(t *testing.T) {
    // Test: Agent completes within 5 iterations
}

func TestRiskInvestigator_EmergencyFallback(t *testing.T) {
    // Test: Agent hits max iterations, returns emergency assessment
}
```

### Integration Tests

#### End-to-End Check Test
```bash
# Setup: Use omnara repository with known file history
$ cd ~/.coderisk/repos/omnara

# Test 1: File with no renames
$ echo "// test change" >> apps/web/src/app/page.tsx
$ crisk check apps/web/src/app/page.tsx

Expected:
- Exact match resolution
- Phase 1 shows ownership history
- Risk assessment completes

# Test 2: Renamed file
$ git mv apps/web/old.tsx apps/web/new.tsx
$ crisk check apps/web/new.tsx

Expected:
- Git follow finds old.tsx
- Historical commits included
- Ownership shows original authors

# Test 3: New file
$ touch apps/web/brand-new.tsx
$ crisk check apps/web/brand-new.tsx

Expected:
- No resolution needed
- Warning: "New file (no historical data)"
- Risk: LOW (default for new files)
```

---

## Open Questions & Decisions Needed

### Q1: Should we cache file resolution results?

**Proposal:** Yes, cache in Redis with repo commit SHA as cache key

**Rationale:**
- File paths don't change frequently within a branch
- Cache hit rate would be ~80% for developers working on same branch
- Invalidate cache on branch switch or git pull

**Implementation:**
```go
func (r *FileResolver) ResolveWithCache(ctx context.Context, currentPath string) ([]FileMatch, error) {
    // Check cache
    cacheKey := fmt.Sprintf("file_resolution:%s:%s", r.repoPath, currentPath)
    if cached, err := r.redis.Get(ctx, cacheKey).Result(); err == nil {
        var matches []FileMatch
        json.Unmarshal([]byte(cached), &matches)
        return matches, nil
    }

    // Cache miss - resolve and cache
    matches, err := r.Resolve(ctx, currentPath)
    if err == nil && len(matches) > 0 {
        data, _ := json.Marshal(matches)
        r.redis.Set(ctx, cacheKey, data, 1*time.Hour)
    }

    return matches, err
}
```

**Decision:** DEFER to post-MVP. Profile first to see if resolution is actually a bottleneck.

---

### Q2: Should we support fuzzy matching (Level 3)?

**Current stance:** NO for MVP

**Revisit if:**
- Level 1-2 coverage drops below 80%
- Users report many "no historical data" warnings
- We see patterns of files that should match but don't

---

### Q3: How many LLM iterations should we allow?

**Current:** Max 5 iterations (hard limit)

**Considerations:**
- Most investigations complete in 3-4 tool calls
- 5 iterations = ~10-15 seconds total
- Need emergency fallback if agent gets stuck

**Alternative:** Adaptive limit based on file complexity?
- Simple files: 3 iterations max
- Complex files: 7 iterations max

**Decision:** Start with fixed 5, adjust based on data.

---

### Q4: Should we show agent's thought process to user?

**Options:**

1. **Hidden (default):** Show only final risk assessment
2. **Summary:** Show tools called: "Checked ownership → incidents → blast radius"
3. **Full trace (--explain):** Show all tool calls with results

**Proposal:** Implement all 3 modes:
```bash
# Default: Just final assessment
$ crisk check file.py
⚠️  MEDIUM risk detected
Reason: High incident density (3 bugs in last 90 days)
Recommendations: [...]

# Summary mode
$ crisk check --verbose file.py
🔍 Investigation: file.py
  ✓ Checked ownership (2 developers)
  ✓ Checked incidents (3 found)
  ✓ Checked blast radius (12 dependents)
⚠️  MEDIUM risk: [...]

# Explain mode
$ crisk check --explain file.py
🔍 Investigation trace:
  [1] query_ownership(file.py)
      → 2 owners: alice (60%), bob (40%)
  [2] query_incident_history(file.py)
      → 3 incidents: #53, #115, #122
  [3] query_blast_radius(file.py)
      → 12 dependents
  [4] finish_investigation(risk=MEDIUM, ...)
⚠️  MEDIUM risk: [...]
```

**Decision:** APPROVED - Implement all 3 modes.

---

## Success Criteria

### MVP Launch Criteria

- [x] File resolution works for 90%+ of files
- [ ] Agent completes investigation in <5 seconds
- [ ] Risk assessments match human judgment (80%+ agreement)
- [ ] No regressions in existing check functionality
- [ ] Documentation updated with new architecture

### Post-MVP Enhancements

- [ ] Add file resolution caching (Redis)
- [ ] Support fuzzy matching (Level 3)
- [ ] Add more tools (e.g., query_similar_files, query_test_coverage)
- [ ] Multi-file investigation (analyze changeset as a whole)
- [ ] User feedback loop (collect correctness ratings)

---

## Appendix: Example Agent Conversation

### Investigation: `src/auth/login.py`

**User Input:**
```bash
$ crisk check src/auth/login.py
```

**File Resolution:**
```
✓ Resolved paths:
  - src/auth/login.py (current)
  - auth/login.py (renamed 45 days ago)
  - backend/login.py (original, 89 days ago)
```

**Agent Conversation (Internal):**

```
[System Prompt]
You are investigating risk for: src/auth/login.py
Historical paths: auth/login.py, backend/login.py
Git diff: +50 lines, -20 lines
Baseline: 8 dependencies, 2 co-change partners, 1 incident

[Agent, iteration 1]
Tool: query_incident_history
Parameters: {file_paths: ["src/auth/login.py", "auth/login.py", "backend/login.py"]}

[Tool Result]
Found 3 incidents:
- Issue #122 (60 days ago): "Login bypass vulnerability"
  Fixed by commit abc123 (alice@example.com)
- Issue #53 (80 days ago): "Session timeout not working"
  Fixed by commit def456 (bob@example.com)
- Issue #89 (85 days ago): "Password reset token leak"
  Fixed by commit ghi789 (alice@example.com)

[Agent, iteration 2]
Tool: get_commit_patch
Parameters: {commit_sha: "abc123"}

[Tool Result]
diff --git a/auth/login.py b/auth/login.py
@@ validate_password(user, password):
-    return user.password == password
+    return bcrypt.checkpw(password, user.password_hash)

[Agent, iteration 3]
Tool: query_ownership
Parameters: {file_paths: ["src/auth/login.py", "auth/login.py", "backend/login.py"]}

[Tool Result]
Owners:
- alice@example.com: 15 commits (65%)
- bob@example.com: 8 commits (35%)
Last modified: 2 months ago by alice

[Agent, iteration 4]
Tool: query_cochange_partners
Parameters: {file_paths: [...], frequency_threshold: 0.5}

[Tool Result]
Co-change partners:
- src/auth/session.py (frequency: 0.8, 12 co-changes)
- src/auth/middleware.py (frequency: 0.6, 9 co-changes)

[Agent, iteration 5]
Tool: finish_investigation
Parameters: {
  risk_level: "HIGH",
  confidence: 0.85,
  reasoning: "This file has a history of security vulnerabilities (3 incidents in 90 days), including a recent login bypass. The file hasn't been modified in 2 months, suggesting the current owner may not be actively maintaining it. Additionally, there are 2 co-change partners (session.py, middleware.py) that should be reviewed together.",
  recommendations: [
    "Request security review from alice@example.com (primary owner)",
    "Verify session.py and middleware.py are also updated if logic changes",
    "Add integration tests for authentication flow",
    "Consider security audit for authentication module"
  ]
}
```

**Output to User:**
```
⚠️  HIGH RISK: src/auth/login.py

Confidence: 85%

Risk Factors:
  • 3 security incidents in last 90 days (#122, #53, #89)
  • Last modified 2 months ago (may be stale)
  • High co-change with session.py and middleware.py

Recommendations:
  1. Request security review from alice@example.com (primary owner)
  2. Verify session.py and middleware.py are also updated
  3. Add integration tests for authentication flow
  4. Consider security audit for authentication module

Investigation completed in 4.2 seconds (5 queries)
```

---

**Document Status:** DRAFT - Ready for review
**Next Steps:**
1. Review and approve approach
2. Estimate effort for Phase 1
3. Begin implementation

