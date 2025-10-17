# Session 5: Init Flow Orchestration

**Created:** October 4, 2025
**Purpose:** Implement complete `crisk init` end-to-end orchestration
**Estimated Duration:** 1-2 days
**Dependencies:** ⚠️ **WAIT for Session 4 Checkpoint 1** before using git functions

---

## Overview

You are implementing the **complete initialization flow** for CodeRisk, orchestrating all layers:

1. **Layer 0:** Git repository detection and validation
2. **Layer 1:** Repository cloning and AST parsing (Tree-sitter)
3. **Layer 2:** GitHub API data fetching
4. **Layer 3:** Graph construction (Neo4j)
5. **Reporting:** Statistics and completion summary

**Your Goal:** Create a seamless `crisk init` command that transforms a GitHub repository URL into a fully-analyzed dependency graph, ready for risk assessment.

---

## Critical: Coordination with Session 4

### Dependency Protocol

**Session 4** is implementing git utility functions at `internal/git/repo.go`. You **MUST** wait for their checkpoint before using these functions.

**Checkpoint 1: Git Functions Ready**

**Wait for Session 4 to notify you when this is complete:**
- `internal/git/repo.go` exists with all 5 functions implemented
- All tests pass: `go test ./internal/git/... -v`
- Functions are ready to import

**What to do while waiting:**
1. Read all integration guides and implementation docs (Step 1)
2. Plan the orchestration flow (Step 2)
3. Implement progress reporting infrastructure (Step 3, partial)
4. Create test repository fixtures

**Once Checkpoint 1 is reached:**
- Import `internal/git` package
- Wire git functions into `cmd/crisk/init.go`
- Complete orchestration implementation

---

## File Ownership

### You Own (Create/Modify Fully)

**Primary Implementation:**
- `cmd/crisk/init.go` - Complete init command orchestration (~300-400 lines)

**Optional Orchestrator:**
- `internal/ingestion/orchestrator.go` - Orchestration logic extracted from cmd (optional refactor)

**Tests:**
- `test/integration/test_init_e2e.sh` - End-to-end init flow test

### You May Modify (Minor Changes Only)

**Progress Reporting:**
- `internal/ingestion/processor.go` - Add progress callbacks (~20 lines)
- `internal/ingestion/walker.go` - Add file count reporting (~10 lines)

### You Read Only (No Modifications)

**Existing Infrastructure:**
- `internal/ingestion/clone.go` - Layer 1 implementation (from Session 3)
- `internal/github/client.go` - Layer 2 GitHub API client
- `internal/github/language_detector.go` - Language detection
- `internal/graph/builder.go` - Layer 3 graph construction
- `internal/models/models.go` - Data models

**Reference Documentation:**
- `dev_docs/03-implementation/integration_guides/cli_integration.md` - Your implementation guide
- `dev_docs/03-implementation/integration_guides/graph_integration.md` - Graph construction
- `dev_docs/01-architecture/system_design.md` - Architecture overview
- `dev_docs/03-implementation/NEXT_STEPS.md` - Task 2 details

**Git Functions (from Session 4):**
- `internal/git/repo.go` - ⚠️ Wait for Checkpoint 1

---

## Implementation Plan

### Step 1: Read and Understand (30 minutes)

**Read these files in order:**

1. `dev_docs/03-implementation/integration_guides/cli_integration.md`
   - Focus on: "Init Command Implementation" section
   - Understand: Orchestration flow, error handling, progress reporting

2. `dev_docs/03-implementation/integration_guides/graph_integration.md`
   - Focus on: Graph construction API and data flow
   - Understand: How to call `graph.Builder.BuildGraph()`

3. `internal/ingestion/clone.go`
   - Understand: `CloneRepository()` function signature and return values
   - Understand: Error handling patterns

4. `internal/github/language_detector.go`
   - Understand: `DetectLanguages()` function and language priority

5. `internal/graph/builder.go`
   - Understand: `Builder.BuildGraph()` orchestration
   - Understand: Statistics returned

**Ask yourself:**
- What are the inputs and outputs of each layer?
- How do errors propagate through the flow?
- What statistics should be reported to the user?

---

### Step 2: Plan the Orchestration Flow (1 hour)

**Design the `runInit()` function structure:**

```go
func runInit(cmd *cobra.Command, args []string) error {
    // Phase 1: Git Repository Detection (Session 4 functions)
    // - DetectGitRepo() or parse --url flag
    // - ParseRepoURL() to extract org/repo

    // Phase 2: Layer 1 - Clone & Parse
    // - Call ingestion.CloneRepository()
    // - Parse AST with Tree-sitter
    // - Report: Files parsed, languages detected

    // Phase 3: Layer 2 - GitHub Data Fetching
    // - Call github.FetchRepoMetadata()
    // - Fetch commits, PRs, contributors
    // - Report: API calls made, data retrieved

    // Phase 4: Layer 3 - Graph Construction
    // - Call graph.BuildGraph()
    // - Store nodes and edges in Neo4j
    // - Report: Nodes created, edges created

    // Phase 5: Completion Summary
    // - Display statistics
    // - Suggest next steps (crisk check)

    return nil
}
```

**Key Design Questions:**
1. Should we fail-fast on errors or continue with partial data?
2. How do we show progress for long-running operations (>10s)?
3. Should we parallelize Layer 2 fetches (commits, PRs, contributors)?

**Recommended Approach:**
- Fail-fast: If Layer 1 fails, don't proceed to Layer 2
- Progress reporting: Use spinner + status messages
- Sequential fetches: Simpler for initial implementation

---

### Step 3: Implement Progress Reporting (2-3 hours)

**Goal:** Show users what's happening during long operations.

**Example Output:**
```
✅ Repository detected: github.com/anthropics/coderisk-go
⏳ Cloning repository...
✅ Cloned 1,234 files (Go, Python detected)
⏳ Parsing source code...
✅ Parsed 856 functions, 124 structs
⏳ Fetching GitHub data...
✅ Retrieved 456 commits, 23 PRs, 12 contributors
⏳ Building dependency graph...
✅ Graph complete: 1,052 nodes, 3,421 edges

🎉 Initialization complete!

📊 Statistics:
   Files:        1,234
   Functions:    856
   Dependencies: 3,421
   Contributors: 12

🚀 Next steps:
   • Run 'crisk check <file>' to analyze risk
   • Install pre-commit hook: 'crisk hook install'
```

**Implementation:**

**Option A: Simple (recommended for initial version)**
```go
func reportProgress(stage string, status string) {
    if status == "start" {
        fmt.Printf("⏳ %s...\n", stage)
    } else if status == "done" {
        fmt.Printf("✅ %s\n", stage)
    }
}

// Usage:
reportProgress("Cloning repository", "start")
result, err := ingestion.CloneRepository(ctx, org, repo)
if err != nil {
    return fmt.Errorf("❌ Clone failed: %w", err)
}
reportProgress(fmt.Sprintf("Cloned %d files", result.FileCount), "done")
```

**Option B: Advanced (spinner + live updates)**
- Use `github.com/briandowns/spinner` package
- Show live progress percentages
- (Optional - implement if time allows)

---

### Step 4: Implement Orchestration (4-6 hours)

**File:** `cmd/crisk/init.go`

**Implementation Checklist:**

#### 4.1: Command Structure

```go
var initCmd = &cobra.Command{
    Use:   "init [url]",
    Short: "Initialize CodeRisk for a repository",
    Long: `Initialize CodeRisk by cloning a repository, analyzing its structure,
fetching GitHub metadata, and building the dependency graph.

Examples:
  crisk init                           # Auto-detect from current git repo
  crisk init github.com/owner/repo     # Initialize specific repo
  crisk init --url https://github.com/owner/repo.git
`,
    RunE: runInit,
}

func init() {
    rootCmd.AddCommand(initCmd)
    initCmd.Flags().String("url", "", "Repository URL (overrides auto-detection)")
    initCmd.Flags().Bool("skip-github", false, "Skip GitHub API fetching (Layer 2)")
    initCmd.Flags().Bool("skip-graph", false, "Skip graph construction (Layer 3)")
}
```

#### 4.2: Phase 1 - Git Detection (uses Session 4 functions)

⚠️ **WAIT for Session 4 Checkpoint 1 before implementing this section**

```go
func runInit(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get repository URL
    var repoURL string
    if urlFlag := cmd.Flags().Lookup("url").Value.String(); urlFlag != "" {
        repoURL = urlFlag
    } else if len(args) > 0 {
        repoURL = args[0]
    } else {
        // Auto-detect from current directory
        if err := git.DetectGitRepo(); err != nil {
            return fmt.Errorf("❌ Not a git repository. Use: crisk init <url>")
        }
        var err error
        repoURL, err = git.GetRemoteURL()
        if err != nil {
            return fmt.Errorf("❌ Could not detect repository URL: %w", err)
        }
    }

    // Parse org and repo name
    org, repo, err := git.ParseRepoURL(repoURL)
    if err != nil {
        return fmt.Errorf("❌ Invalid repository URL: %w", err)
    }

    fmt.Printf("✅ Repository detected: %s/%s\n", org, repo)

    // Continue to Phase 2...
}
```

**Human Checkpoint: Git Detection**
- **When:** After implementing Phase 1
- **You Ask:** "✅ Phase 1 (Git Detection) implemented. Should I test with a sample repo URL?"
- **Test Command:** `go run ./cmd/crisk init github.com/anthropics/coderisk-go`
- **Expected:** Should print "✅ Repository detected: anthropics/coderisk-go"

#### 4.3: Phase 2 - Layer 1 (Clone & Parse)

```go
// Inside runInit(), after Phase 1
reportProgress("Cloning repository", "start")

cloneResult, err := ingestion.CloneRepository(ctx, org, repo)
if err != nil {
    return fmt.Errorf("❌ Clone failed: %w", err)
}

reportProgress(fmt.Sprintf("Cloned %d files (%s detected)",
    cloneResult.FileCount,
    strings.Join(cloneResult.Languages, ", ")), "done")

// Tree-sitter parsing happens inside CloneRepository
// Statistics are in cloneResult
```

**Implementation Note:**
- `ingestion.CloneRepository()` already exists from Session 3
- It returns `CloneResult` with file counts and detected languages
- No changes needed to clone.go unless adding progress callbacks

**Optional Enhancement:**
If `ingestion.CloneRepository()` takes too long (>10s), add progress reporting:

```go
// In internal/ingestion/processor.go
type ProgressCallback func(stage string, current, total int)

// Modify CloneRepository signature:
func CloneRepository(ctx context.Context, org, repo string, onProgress ProgressCallback) (*CloneResult, error) {
    // Inside file walking loop:
    if onProgress != nil {
        onProgress("parsing", currentFileIndex, totalFiles)
    }
}
```

#### 4.4: Phase 3 - Layer 2 (GitHub Data)

```go
// Inside runInit(), after Phase 2
if !cmd.Flags().Lookup("skip-github").Changed {
    reportProgress("Fetching GitHub data", "start")

    // Initialize GitHub client
    githubClient := github.NewClient(ctx, os.Getenv("GITHUB_TOKEN"))

    // Fetch repository metadata
    repoData, err := githubClient.FetchRepoMetadata(org, repo)
    if err != nil {
        return fmt.Errorf("❌ GitHub API failed: %w", err)
    }

    // Fetch commits (last 100)
    commits, err := githubClient.FetchCommits(org, repo, 100)
    if err != nil {
        return fmt.Errorf("❌ Commit fetch failed: %w", err)
    }

    // Fetch PRs (last 50)
    prs, err := githubClient.FetchPullRequests(org, repo, 50)
    if err != nil {
        return fmt.Errorf("❌ PR fetch failed: %w", err)
    }

    reportProgress(fmt.Sprintf("Retrieved %d commits, %d PRs, %d contributors",
        len(commits), len(prs), repoData.ContributorCount), "done")
}
```

**Error Handling:**
- If GitHub token is missing: Warn but continue (use public API limits)
- If rate limit exceeded: Suggest waiting or using `--skip-github`
- If repo is private: Require valid token with repo access

#### 4.5: Phase 4 - Layer 3 (Graph Construction)

```go
// Inside runInit(), after Phase 3
if !cmd.Flags().Lookup("skip-graph").Changed {
    reportProgress("Building dependency graph", "start")

    // Initialize graph builder
    graphBuilder, err := graph.NewBuilder(ctx)
    if err != nil {
        return fmt.Errorf("❌ Graph initialization failed: %w", err)
    }
    defer graphBuilder.Close()

    // Build graph from all collected data
    stats, err := graphBuilder.BuildGraph(ctx, &graph.BuildInput{
        Org:          org,
        Repo:         repo,
        CloneResult:  cloneResult,
        RepoMetadata: repoData,
        Commits:      commits,
        PullRequests: prs,
    })
    if err != nil {
        return fmt.Errorf("❌ Graph construction failed: %w", err)
    }

    reportProgress(fmt.Sprintf("Graph complete: %d nodes, %d edges",
        stats.NodeCount, stats.EdgeCount), "done")
}
```

**Human Checkpoint: Graph Construction**
- **When:** After implementing Phase 4
- **You Ask:** "✅ Phase 4 (Graph Construction) implemented. Should I test the full init flow?"
- **Test Command:** `go run ./cmd/crisk init --url github.com/anthropics/coderisk-go`
- **Expected:** Full flow completes, Neo4j contains nodes/edges
- **Verification:** Run `docker exec -it coderisk-neo4j cypher-shell -u neo4j -p password "MATCH (n) RETURN count(n)"`

#### 4.6: Phase 5 - Completion Summary

```go
// Inside runInit(), at the end
fmt.Println("\n🎉 Initialization complete!\n")
fmt.Println("📊 Statistics:")
fmt.Printf("   Files:        %d\n", cloneResult.FileCount)
fmt.Printf("   Functions:    %d\n", cloneResult.FunctionCount)
fmt.Printf("   Dependencies: %d\n", stats.EdgeCount)
fmt.Printf("   Contributors: %d\n", repoData.ContributorCount)
fmt.Println("\n🚀 Next steps:")
fmt.Println("   • Run 'crisk check <file>' to analyze risk")
fmt.Println("   • Install pre-commit hook: 'crisk hook install'")

return nil
```

---

### Step 5: Create Integration Test (2-3 hours)

**File:** `test/integration/test_init_e2e.sh`

**Template:**

```bash
#!/bin/bash
set -e

echo "=== CodeRisk Init Flow E2E Test ==="

# Test 1: Init with URL
echo "Test 1: Init with explicit URL"
./bin/crisk init --url https://github.com/anthropics/coderisk-go.git

# Verify: Check Neo4j for nodes
NODE_COUNT=$(docker exec -it coderisk-neo4j cypher-shell -u neo4j -p password \
  "MATCH (n) RETURN count(n)" --format plain | tail -1)

if [ "$NODE_COUNT" -gt 0 ]; then
    echo "✅ PASS: Graph contains $NODE_COUNT nodes"
else
    echo "❌ FAIL: Graph is empty"
    exit 1
fi

# Test 2: Init with auto-detection (in git repo)
echo "Test 2: Init with auto-detection"
cd /tmp/test-repo
git init
git remote add origin https://github.com/test/repo.git
/path/to/bin/crisk init

# Expected: Should detect repo URL from git remote

# Test 3: Init outside git repo (should fail)
echo "Test 3: Init without git repo (should fail)"
cd /tmp/non-git-dir
if ./bin/crisk init 2>&1 | grep -q "Not a git repository"; then
    echo "✅ PASS: Correctly rejected non-git directory"
else
    echo "❌ FAIL: Should reject non-git directory"
    exit 1
fi

echo "=== All tests passed ==="
```

**Run Test:**
```bash
chmod +x test/integration/test_init_e2e.sh
./test/integration/test_init_e2e.sh
```

---

### Step 6: Error Handling & Edge Cases (2 hours)

**Implement robust error handling:**

#### 6.1: Missing GitHub Token
```go
githubToken := os.Getenv("GITHUB_TOKEN")
if githubToken == "" {
    fmt.Println("⚠️  Warning: GITHUB_TOKEN not set. Using public API (rate limited).")
    fmt.Println("   Set token with: export GITHUB_TOKEN=ghp_...")
}
```

#### 6.2: Network Failures
```go
if err := githubClient.FetchCommits(...); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("❌ GitHub API timeout. Try again or use --skip-github")
    }
    return fmt.Errorf("❌ Network error: %w", err)
}
```

#### 6.3: Neo4j Connection Failure
```go
if err := graphBuilder.BuildGraph(...); err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        return fmt.Errorf("❌ Neo4j not running. Start with: docker-compose up -d")
    }
    return fmt.Errorf("❌ Graph construction failed: %w", err)
}
```

#### 6.4: Large Repository Warning
```go
if cloneResult.FileCount > 10000 {
    fmt.Printf("⚠️  Large repository detected (%d files). This may take several minutes.\n",
        cloneResult.FileCount)
}
```

---

## Testing Strategy

### Unit Tests

**File:** `cmd/crisk/init_test.go`

**Test Coverage (aim for 80%+):**

```go
func TestParseInitArgs(t *testing.T) {
    // Test URL parsing from args
    // Test URL parsing from --url flag
    // Test auto-detection mode
}

func TestProgressReporting(t *testing.T) {
    // Test progress message formatting
    // Test emoji rendering
}

func TestInitErrorHandling(t *testing.T) {
    // Test invalid URL handling
    // Test missing git repo handling
    // Test network error handling
}
```

**Run Tests:**
```bash
go test ./cmd/crisk/... -v -cover
```

**Expected Coverage:** >80%

### Integration Tests

**test/integration/test_init_e2e.sh** (created in Step 5)

**Scenarios to Test:**
1. ✅ Init with explicit URL
2. ✅ Init with auto-detection (inside git repo)
3. ✅ Init without git repo (should fail gracefully)
4. ✅ Init with --skip-github flag
5. ✅ Init with --skip-graph flag
6. ✅ Verify Neo4j contains expected data

**Run All Integration Tests:**
```bash
./test/integration/test_init_e2e.sh
```

---

## Critical Checkpoints

### Checkpoint 1: Wait for Session 4 (Git Functions Ready)

**Trigger:** Session 4 completes git function implementation

**Verification:**
```bash
# Check if internal/git/repo.go exists
ls -l internal/git/repo.go

# Check if tests pass
go test ./internal/git/... -v

# Expected: All tests pass
```

**Action:** Once verified, proceed with Phase 1 (Git Detection) in Step 4.2

**YOU ASK:** "⚠️  Waiting for Session 4 to complete git functions. Is `internal/git/repo.go` ready? (Run: `go test ./internal/git/... -v`)"

---

### Checkpoint 2: Init Flow Complete (End-to-End Test)

**Trigger:** You complete Step 4 (all phases implemented)

**Verification:**
```bash
# Build binary
go build -o bin/crisk ./cmd/crisk

# Test full init flow
./bin/crisk init --url https://github.com/anthropics/coderisk-go.git

# Expected output:
# ✅ Repository detected: anthropics/coderisk-go
# ⏳ Cloning repository...
# ✅ Cloned 1,234 files (Go detected)
# ⏳ Fetching GitHub data...
# ✅ Retrieved 456 commits, 23 PRs, 12 contributors
# ⏳ Building dependency graph...
# ✅ Graph complete: 1,052 nodes, 3,421 edges
# 🎉 Initialization complete!
```

**Verify Neo4j:**
```bash
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p password \
  "MATCH (n) RETURN count(n)"

# Expected: >1000 nodes
```

**YOU ASK:** "✅ Init flow complete. End-to-end test passed. Statistics: [paste output]. Neo4j contains [X] nodes. Should I proceed with integration tests?"

---

### Checkpoint 3: Integration Tests Pass

**Trigger:** You complete Step 5 (integration test script)

**Verification:**
```bash
./test/integration/test_init_e2e.sh

# Expected: All tests pass
```

**YOU ASK:** "✅ All integration tests passing (X/X). Ready to mark Session 5 complete?"

---

## Success Criteria

### Functional Requirements
- [ ] `crisk init` works end-to-end with explicit URL
- [ ] `crisk init` auto-detects repository from current git directory
- [ ] `crisk init` fails gracefully when not in a git repo
- [ ] `--skip-github` and `--skip-graph` flags work correctly
- [ ] Progress reporting shows all 5 phases
- [ ] Statistics are accurate (file counts, node counts match reality)
- [ ] Neo4j contains expected nodes and edges after init

### Performance Requirements
- [ ] Init completes in <30s for small repos (~1K files)
- [ ] Init completes in <5 minutes for medium repos (~10K files)
- [ ] Progress updates every 2-5 seconds during long operations

### Quality Requirements
- [ ] 80%+ unit test coverage for init command logic
- [ ] Integration test passes for all scenarios
- [ ] Error messages are actionable (tell user what to do)
- [ ] No build errors, all packages compile

---

## What Could Go Wrong

### Issue: Session 4 git functions not ready
**Prevention:** Explicit checkpoint protocol (wait for notification)
**Recovery:** Create temporary stubs, replace with real functions after Checkpoint 1

### Issue: Init hangs on large repositories
**Prevention:** Add timeouts (5 min max), progress reporting
**Recovery:** Add `--timeout` flag, suggest `--skip-layers` for debugging

### Issue: Neo4j connection fails
**Prevention:** Check connection before starting init
**Recovery:** Print helpful error message: "Start Neo4j with: docker-compose up -d"

### Issue: GitHub rate limit exceeded
**Prevention:** Warn if token not set, show rate limit status
**Recovery:** Suggest waiting or using `--skip-github`

### Issue: Statistics don't match (e.g., file count wrong)
**Prevention:** Double-check counts at each layer
**Recovery:** Add `--debug` flag to show detailed layer statistics

---

## Quick Commands for Verification

**Build:**
```bash
go build -o bin/crisk ./cmd/crisk
```

**Test init flow:**
```bash
./bin/crisk init --url https://github.com/anthropics/coderisk-go.git
```

**Verify Neo4j:**
```bash
docker exec -it coderisk-neo4j cypher-shell -u neo4j -p password \
  "MATCH (n) RETURN labels(n), count(n)"
```

**Run unit tests:**
```bash
go test ./cmd/crisk/... -v -cover
```

**Run integration tests:**
```bash
./test/integration/test_init_e2e.sh
```

---

## References

**Implementation Guides:**
- [cli_integration.md](integration_guides/cli_integration.md) - Your primary reference
- [graph_integration.md](integration_guides/graph_integration.md) - Graph construction API

**Architecture:**
- [system_design.md](../01-architecture/system_design.md) - Overall system architecture
- [data_collection_layers.md](../01-architecture/data_collection_layers.md) - Layer details

**Coordination:**
- [PARALLEL_SESSION_PLAN_WEEK1.md](PARALLEL_SESSION_PLAN_WEEK1.md) - Overall coordination plan
- [SESSION_4_PROMPT.md](SESSION_4_PROMPT.md) - Git functions (dependency)

**Next Steps:**
- [NEXT_STEPS.md](NEXT_STEPS.md) - Week 1 Task 2 (this session)

---

## After Session Complete

**Update documentation:**
- Mark Week 1 Task 2 complete in [NEXT_STEPS.md](NEXT_STEPS.md)
- Add entry to [IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md)
- Update [status.md](status.md) with init flow completion

**Coordinate with other sessions:**
- Session 6 can now use `crisk init` for test setup

---

**Created:** October 4, 2025
**Status:** Ready to execute (after Session 4 Checkpoint 1)
**Estimated Duration:** 1-2 days
**Owner:** Session 5
