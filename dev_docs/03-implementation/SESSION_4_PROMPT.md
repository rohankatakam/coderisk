# Session 4 Prompt: Git Integration Functions

**Duration:** 1-2 days
**Owner:** Claude Code Session 4
**Dependencies:** None (can start immediately)

---

## Context

You are implementing **Git Integration Functions** for CodeRisk's core functionality. This is **Session 4 of 3 parallel sessions (Week 1)** working on different parts of the codebase simultaneously.

**Your role:** Implement robust git repository utilities that other sessions will depend on.

**Other sessions (DO NOT MODIFY THEIR FILES):**
- Session 5: Building init flow orchestration in `cmd/crisk/init.go` (WAITS for your git functions)
- Session 6: Validating risk calculation in `internal/risk/` (independent work)

**CRITICAL:** Session 5 needs your git functions. Implement them **correctly and quickly** so Session 5 can proceed.

---

## High-Level Goal

Implement 5 core git functions:
1. **DetectGitRepo()** - Check if directory is a git repository
2. **ParseRepoURL()** - Extract org/repo from git remote URL (supports HTTPS + SSH)
3. **GetChangedFiles()** - Get list of modified files in working directory
4. **GetRemoteURL()** - Get git remote URL
5. **GetCurrentBranch()** - Get current branch name

**Why this matters:** These are foundational utilities. Every `crisk` command depends on detecting the git repo and understanding its state.

---

## Your File Ownership

### Files YOU CREATE (your responsibility):
- `internal/git/repo.go` - Git utility functions (your main deliverable)
- `internal/git/repo_test.go` - Unit tests (80%+ coverage required)
- `test/integration/test_git_integration.sh` - Integration tests

### Files YOU MODIFY (minimal changes):
- `cmd/crisk/init.go` - Replace `detectGitRepo()` and `parseRepoURL()` stubs (~20 lines)
- `cmd/crisk/check.go` - Replace `getChangedFiles()` stub (~10 lines)

### Files YOU READ ONLY (do not modify):
- `internal/git/staged.go` - Already exists from Session 1
- All other files

---

## Reading List (READ THESE FIRST)

**MUST READ before coding:**
1. `dev_docs/03-implementation/NEXT_STEPS.md` - Task 1 (your implementation guide)
2. `dev_docs/03-implementation/PARALLEL_SESSION_PLAN_WEEK1.md` - Coordination plan
3. `internal/git/staged.go` - Existing git utilities (understand the pattern)

**Reference as needed:**
4. `dev_docs/DEVELOPMENT_WORKFLOW.md` - Go development guardrails
5. `cmd/crisk/init.go` - Where you'll integrate (see the stubs)
6. `cmd/crisk/check.go` - Where you'll integrate (see the stubs)

---

## Step-by-Step Implementation Plan

### Step 1: Read Documentation (15 min)
- [ ] Read all files in "Reading List" section above
- [ ] Understand git function requirements from NEXT_STEPS.md
- [ ] Review existing `internal/git/staged.go` for patterns
- [ ] **ASK USER:** "I've read the documentation. Should I proceed with implementation?"

---

### Step 2: Implement Core Git Functions (2-4 hours)

**File: `internal/git/repo.go`** (create new file)

```go
package git

import (
    "fmt"
    "os/exec"
    "regexp"
    "strings"
)

// DetectGitRepo checks if current directory is a git repository
func DetectGitRepo() error {
    cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("not a git repository: %w", err)
    }
    return nil
}

// GetRemoteURL returns the remote URL for the repository
func GetRemoteURL() (string, error) {
    cmd := exec.Command("git", "config", "--get", "remote.origin.url")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get remote URL: %w", err)
    }
    return strings.TrimSpace(string(output)), nil
}

// ParseRepoURL extracts org and repo name from git remote URL
// Supports both HTTPS and SSH formats:
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
//   - git://github.com/owner/repo.git
func ParseRepoURL(remoteURL string) (org, repo string, err error) {
    // TODO: Implement this function
    // Handle multiple URL formats
    // Use regex to extract org/repo
    // Strip .git suffix
    // Return error if URL format not recognized

    // YOUR IMPLEMENTATION HERE
}

// GetChangedFiles returns list of files changed in working directory
func GetChangedFiles() ([]string, error) {
    cmd := exec.Command("git", "diff", "--name-only", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to get changed files: %w", err)
    }

    files := strings.Split(strings.TrimSpace(string(output)), "\n")
    var result []string
    for _, f := range files {
        if f != "" {
            result = append(result, f)
        }
    }
    return result, nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get current branch: %w", err)
    }
    return strings.TrimSpace(string(output)), nil
}

// FindGitRoot returns the root directory of the git repository
// (Already exists in staged.go, but duplicating here for completeness)
func FindGitRoot() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to find git root: %w", err)
    }
    return strings.TrimSpace(string(output)), nil
}
```

**CRITICAL FUNCTION: ParseRepoURL()**

You need to implement this to handle multiple URL formats. Here are examples:

```go
// Test cases you MUST handle:
// HTTPS: "https://github.com/owner/repo.git" → org="owner", repo="repo"
// SSH:   "git@github.com:owner/repo.git" → org="owner", repo="repo"
// Git:   "git://github.com/owner/repo.git" → org="owner", repo="repo"
// No .git: "https://github.com/owner/repo" → org="owner", repo="repo"
```

**Implementation hints:**
```go
func ParseRepoURL(remoteURL string) (org, repo string, err error) {
    // Remove .git suffix if present
    remoteURL = strings.TrimSuffix(remoteURL, ".git")

    // Try HTTPS format: https://github.com/owner/repo
    httpsRegex := regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)`)
    if matches := httpsRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
        return matches[1], matches[2], nil
    }

    // Try SSH format: git@github.com:owner/repo
    sshRegex := regexp.MustCompile(`git@[^:]+:([^/]+)/([^/]+)`)
    if matches := sshRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
        return matches[1], matches[2], nil
    }

    // Try git protocol: git://github.com/owner/repo
    gitRegex := regexp.MustCompile(`git://[^/]+/([^/]+)/([^/]+)`)
    if matches := gitRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
        return matches[1], matches[2], nil
    }

    return "", "", fmt.Errorf("unrecognized git URL format: %s", remoteURL)
}
```

**Checkpoint:** Build the package
```bash
go build ./internal/git
```

**ASK USER:** "✅ Git functions implemented in internal/git/repo.go. Package compiles. Should I proceed with unit tests?"

---

### Step 3: Write Comprehensive Unit Tests (1-2 hours)

**File: `internal/git/repo_test.go`** (create new file)

```go
package git

import (
    "testing"
)

func TestParseRepoURL(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantOrg string
        wantRepo string
        wantErr bool
    }{
        {
            name:    "HTTPS with .git",
            url:     "https://github.com/coderisk/coderisk-go.git",
            wantOrg: "coderisk",
            wantRepo: "coderisk-go",
            wantErr: false,
        },
        {
            name:    "HTTPS without .git",
            url:     "https://github.com/coderisk/coderisk-go",
            wantOrg: "coderisk",
            wantRepo: "coderisk-go",
            wantErr: false,
        },
        {
            name:    "SSH format",
            url:     "git@github.com:coderisk/coderisk-go.git",
            wantOrg: "coderisk",
            wantRepo: "coderisk-go",
            wantErr: false,
        },
        {
            name:    "Git protocol",
            url:     "git://github.com/coderisk/coderisk-go.git",
            wantOrg: "coderisk",
            wantRepo: "coderisk-go",
            wantErr: false,
        },
        {
            name:    "Invalid URL",
            url:     "not-a-git-url",
            wantOrg: "",
            wantRepo: "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            org, repo, err := ParseRepoURL(tt.url)

            if (err != nil) != tt.wantErr {
                t.Errorf("ParseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if org != tt.wantOrg {
                t.Errorf("ParseRepoURL() org = %v, want %v", org, tt.wantOrg)
            }

            if repo != tt.wantRepo {
                t.Errorf("ParseRepoURL() repo = %v, want %v", repo, tt.wantRepo)
            }
        })
    }
}

// Add similar tests for other functions:
// TestDetectGitRepo()
// TestGetChangedFiles()
// TestGetRemoteURL()
// TestGetCurrentBranch()
// TestFindGitRoot()
```

**Checkpoint:** Run unit tests
```bash
go test ./internal/git/... -v -cover
```

**Target:** 80%+ coverage, all tests passing

**ASK USER:** "✅ Unit tests written. Coverage: [X]%. All tests pass: YES/NO. Should I proceed with integration tests?"

---

### Step 4: Create Integration Tests (1 hour)

**File: `test/integration/test_git_integration.sh`** (create new)

```bash
#!/bin/bash
set -e

echo "=== Git Integration Tests ==="

# Create temporary git repository for testing
TEST_DIR=$(mktemp -d)
cd $TEST_DIR

echo "Test 1: DetectGitRepo (not a repo)"
if go run -C /path/to/project scripts/test_git_detect.go 2>&1 | grep -q "not a git repository"; then
    echo "✅ PASS: Correctly detects non-git directory"
else
    echo "❌ FAIL: Should detect non-git directory"
    exit 1
fi

echo "Test 2: DetectGitRepo (is a repo)"
git init
if go run -C /path/to/project scripts/test_git_detect.go; then
    echo "✅ PASS: Correctly detects git repository"
else
    echo "❌ FAIL: Should detect git repository"
    exit 1
fi

echo "Test 3: ParseRepoURL (HTTPS)"
git remote add origin https://github.com/coderisk/coderisk-go.git
ORG_REPO=$(go run -C /path/to/project scripts/test_git_parse.go)
if echo "$ORG_REPO" | grep -q "coderisk coderisk-go"; then
    echo "✅ PASS: Correctly parsed HTTPS URL"
else
    echo "❌ FAIL: Incorrect parsing: $ORG_REPO"
    exit 1
fi

echo "Test 4: ParseRepoURL (SSH)"
git remote set-url origin git@github.com:coderisk/coderisk-go.git
ORG_REPO=$(go run -C /path/to/project scripts/test_git_parse.go)
if echo "$ORG_REPO" | grep -q "coderisk coderisk-go"; then
    echo "✅ PASS: Correctly parsed SSH URL"
else
    echo "❌ FAIL: Incorrect parsing: $ORG_REPO"
    exit 1
fi

echo "Test 5: GetChangedFiles"
echo "test content" > test_file.go
git add test_file.go
git commit -m "Initial commit"
echo "modified" >> test_file.go
FILES=$(go run -C /path/to/project scripts/test_git_changed.go)
if echo "$FILES" | grep -q "test_file.go"; then
    echo "✅ PASS: Correctly detected changed file"
else
    echo "❌ FAIL: Should detect test_file.go as changed"
    exit 1
fi

echo "Test 6: GetCurrentBranch"
BRANCH=$(go run -C /path/to/project scripts/test_git_branch.go)
if [ "$BRANCH" == "main" ] || [ "$BRANCH" == "master" ]; then
    echo "✅ PASS: Correctly detected branch: $BRANCH"
else
    echo "❌ FAIL: Unexpected branch: $BRANCH"
    exit 1
fi

# Cleanup
cd /
rm -rf $TEST_DIR

echo "=== All git integration tests passed! ==="
```

**Make executable:**
```bash
chmod +x test/integration/test_git_integration.sh
```

**ASK USER:** "✅ Integration test created. Should I run it to verify?"

---

### Step 5: Integrate with Check and Init Commands (30 min)

**File: Update `cmd/crisk/check.go`** (minimal changes)

```go
import (
    // ... existing imports ...
    "github.com/coderisk/coderisk-go/internal/git"
)

func runCheck(cmd *cobra.Command, args []string) error {
    var files []string
    var err error

    if len(args) > 0 {
        files = args
    } else {
        // Replace stub with real function
        files, err = git.GetChangedFiles()
        if err != nil {
            return fmt.Errorf("failed to get changed files: %w", err)
        }

        if len(files) == 0 {
            fmt.Println("No changed files to check")
            return nil
        }
    }

    // ... rest of existing check logic ...
}
```

**File: Update `cmd/crisk/init.go`** (minimal changes)

```go
import (
    // ... existing imports ...
    "github.com/coderisk/coderisk-go/internal/git"
)

func runInit(cmd *cobra.Command, args []string) error {
    // Replace stub with real function
    if err := git.DetectGitRepo(); err != nil {
        return err
    }

    // Replace stub with real function
    remoteURL, err := git.GetRemoteURL()
    if err != nil {
        return fmt.Errorf("failed to get git remote: %w", err)
    }

    // Replace stub with real function
    org, repo, err := git.ParseRepoURL(remoteURL)
    if err != nil {
        return fmt.Errorf("failed to parse repository URL: %w", err)
    }

    log.Infof("Detected repository: %s/%s", org, repo)

    // ... rest of existing init logic ...
}
```

**Checkpoint:** Build and test
```bash
go build ./cmd/crisk
./bin/crisk check
# Should detect changed files automatically

./bin/crisk init
# Should detect git repo and remote URL
```

**ASK USER:** "✅ Integrated with check and init commands. Binary builds. Should I proceed with final validation?"

---

### Step 6: Final Validation & Notification (30 min)

**Validation checklist:**
- [ ] Run `go build ./internal/git` - Verify package compiles
- [ ] Run `go test ./internal/git/... -v -cover` - Verify >80% coverage, all tests pass
- [ ] Run `./bin/crisk check` in a git repo - Verify detects changed files
- [ ] Run `./bin/crisk init` in a git repo - Verify detects repo and remote
- [ ] Test with multiple git URL formats (HTTPS, SSH)

**Performance check:**
- [ ] Time git function calls: `time ./bin/crisk check`
- [ ] Verify git overhead <100ms
- [ ] **ASK USER:** "Performance: git functions take [X]ms. Acceptable? (<100ms target)"

**Documentation:**
- [ ] Add docstrings to all exported functions
- [ ] Add examples in comments

**CRITICAL CHECKPOINT:**

**ASK USER (NOTIFY SESSIONS):**
"✅ Session 4 complete! Git functions ready for use.

**Deliverables:**
- internal/git/repo.go (5 functions implemented)
- internal/git/repo_test.go (80%+ coverage)
- test/integration/test_git_integration.sh
- Integrated with cmd/crisk/check.go and cmd/crisk/init.go

**Test Results:**
- Unit tests: [X]/[X] passing
- Coverage: [X]%
- Integration tests: [X]/[X] passing
- Performance: [X]ms overhead

**NOTIFY SESSION 5:** Git functions available at `internal/git/repo.go`. Can now proceed with init flow implementation.

Should I update status.md and mark Session 4 complete?"

**WAIT FOR USER CONFIRMATION BEFORE MARKING COMPLETE**

---

## Critical Checkpoints (Human Verification Required)

### Checkpoint 1: After Step 2 (Functions Implemented)
**YOU ASK:** "Git functions implemented. Package compiles. Should I proceed with tests?"
**WAIT FOR:** User confirmation

### Checkpoint 2: After Step 3 (Unit Tests)
**YOU ASK:** "Unit tests complete. Coverage: [X]%. All pass: YES/NO. Should I proceed?"
**WAIT FOR:** User confirmation

### Checkpoint 3: After Step 5 (Integration)
**YOU ASK:** "Integrated with check/init. Binary builds and runs. Should I proceed with validation?"
**WAIT FOR:** User confirmation

### Checkpoint 4: Final (Before notifying Session 5)
**YOU ASK:** "All deliverables complete. Ready to notify Session 5?"
**WAIT FOR:** User confirmation

---

## Coordination with Other Sessions

### YOU CREATE (Session 5 depends on this):
- **`internal/git/repo.go`** - **CREATE THIS CORRECTLY!**
  - Session 5 imports and uses these functions in init.go
  - Quality matters: bugs here will block Session 5

### DO NOT MODIFY (other sessions own these):
- `cmd/crisk/init.go` - Session 5 owns orchestration logic
- `internal/risk/*` - Session 6
- `internal/output/*` - Session 2 (already done)

### NOTIFY SESSION 5 WHEN:
- **After Step 6:** Git functions are tested and ready
  - **ASK USER:** "Should I notify Session 5 that git functions are ready?"

---

## Success Criteria

**Functional:**
- [ ] DetectGitRepo() works in git repos and non-git directories
- [ ] ParseRepoURL() handles HTTPS, SSH, and git:// URLs
- [ ] GetChangedFiles() returns modified files correctly
- [ ] GetRemoteURL() returns remote URL
- [ ] GetCurrentBranch() returns branch name

**Quality:**
- [ ] 80%+ unit test coverage
- [ ] All edge cases tested (invalid URLs, non-git dirs, etc.)
- [ ] Integration tests pass

**Performance:**
- [ ] Git function overhead <100ms

---

## Error Handling

**If you encounter issues:**
1. **Regex not matching URLs:** Test with real git URLs, adjust regex patterns
2. **Git command fails:** Add proper error handling, informative error messages
3. **Test failures:** Debug with verbose output (`go test -v`)
4. **Build errors:** Check imports, ensure `go.mod` is up to date

**Always ask before:**
- Modifying files not in your ownership list
- Making breaking changes to function signatures
- Adding new dependencies

---

## Final Deliverables

When complete, you should have:
1. ✅ `internal/git/repo.go` with 5 working functions
2. ✅ `internal/git/repo_test.go` with 80%+ coverage
3. ✅ `test/integration/test_git_integration.sh` passing
4. ✅ Integration with `cmd/crisk/check.go` and `cmd/crisk/init.go`
5. ✅ All tests passing, binary builds successfully
6. ✅ Notification to Session 5 that git functions are ready

---

## Questions to Ask During Implementation

- "I've read the documentation. Should I proceed?"
- "Git functions implemented. Package compiles. Should I proceed with tests?"
- "Unit tests complete. Coverage: [X]%. Should I proceed with integration tests?"
- "Integration test ready. Should I run it?"
- "Integrated with check/init commands. Should I proceed with validation?"
- "Performance: [X]ms. Acceptable?"
- "Session 4 complete! Should I notify Session 5 and update status.md?"

**Remember:** Session 5 is waiting for your git functions. Quality and correctness are critical!
