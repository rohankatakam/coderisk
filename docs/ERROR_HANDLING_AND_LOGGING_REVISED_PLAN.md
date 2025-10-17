# Error Handling and Logging Implementation Plan (REVISED)

**Project:** CodeRisk
**Date:** October 2025
**Status:** 🚧 READY TO IMPLEMENT
**Priority:** 🔴 CRITICAL

---

## Executive Summary

This document outlines the comprehensive plan to replace all error handling anti-patterns with proper error handling and strategic logging throughout the CodeRisk codebase - **aligned with our developer experience philosophy**.

### Key Principle: Seamless Developer Experience

**What developers SHOULD configure:**
- ✅ `OPENAI_API_KEY` (or via keychain) - Required for Phase 2
- ✅ `GITHUB_TOKEN` - Required for repo fetching

**What we SHOULD auto-configure:**
- ✅ Neo4j password (generated per-deployment or use secure defaults from `.env`)
- ✅ PostgreSQL password (generated per-deployment or use secure defaults from `.env`)
- ✅ Redis (no auth in local dev)
- ✅ Ports, memory limits, connection pools

### What We've Built (Infrastructure)

1. **Centralized Logging System** - [internal/logging/logger.go](internal/logging/logger.go)
2. **Error Handling Framework** - [internal/errors/errors.go](internal/errors/errors.go)
3. **Configuration Validation** - [internal/config/validator.go](internal/config/validator.go)

---

## Revised Security Strategy

### Context-Aware Password Handling

**Local Development (`make dev`, `crisk init-local`):**
- Passwords read from `.env` file (auto-configured by Docker Compose)
- These are **development defaults only** - fine for local containers
- No user interaction required
- No keychain needed

**Production/Cloud (`crisk init <repo-url>`):**
- Passwords MUST be in environment variables or keychain
- No hardcoded defaults allowed
- Clear error messages if missing

**CI/CD:**
- All credentials from environment variables
- Validation enforced at startup

### Implementation Strategy

```go
// internal/config/config_mode.go
type DeploymentMode string

const (
    ModeLocal      DeploymentMode = "local"       // Docker Compose local dev
    ModeProduction DeploymentMode = "production"  // Cloud/production
    ModeCI         DeploymentMode = "ci"          // CI/CD pipelines
)

func DetectMode() DeploymentMode {
    if os.Getenv("CI") != "" {
        return ModeCI
    }
    if os.Getenv("CODERISK_MODE") == "production" {
        return ModeProduction
    }
    return ModeLocal  // Default to local dev
}

// Password validation is mode-aware
func (c *Config) validateNeo4jPassword(result *ValidationResult, mode DeploymentMode) {
    if c.Neo4j.Password == "" {
        result.AddError("NEO4J_PASSWORD is required")
        return
    }

    // In local mode, .env defaults are acceptable
    if mode == ModeLocal {
        // Allow .env defaults for local dev
        return
    }

    // In production/CI, reject insecure defaults
    insecureDefaults := []string{
        "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
        "coderisk123",
        "password",
        "neo4j",
    }
    for _, insecure := range insecureDefaults {
        if c.Neo4j.Password == insecure {
            result.AddError("NEO4J_PASSWORD uses insecure default. Set a secure password for production.")
        }
    }
}
```

---

## Revised Implementation Phases

### Phase 1: Context-Aware Configuration ⏱️ 2 hours

**Goal:** Support seamless local dev while enforcing security in production

#### 1.1 Create Deployment Mode Detection

**New file:** `internal/config/mode.go`

```go
package config

import "os"

type DeploymentMode string

const (
    ModeLocal      DeploymentMode = "local"
    ModeProduction DeploymentMode = "production"
    ModeCI         DeploymentMode = "ci"
)

// DetectMode determines the deployment context
func DetectMode() DeploymentMode {
    // CI environment
    if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
        return ModeCI
    }

    // Explicit production mode
    if mode := os.Getenv("CODERISK_MODE"); mode == "production" {
        return ModeProduction
    }

    // Check if using cloud Neo4j (Aura)
    neo4jURI := os.Getenv("NEO4J_URI")
    if strings.Contains(neo4jURI, "neo4j.io") || strings.Contains(neo4jURI, "amazonaws.com") {
        return ModeProduction
    }

    // Default to local development
    return ModeLocal
}

// IsLocal returns true if running in local development mode
func IsLocal() bool {
    return DetectMode() == ModeLocal
}

// IsProduction returns true if running in production/cloud
func IsProduction() bool {
    mode := DetectMode()
    return mode == ModeProduction || mode == ModeCI
}
```

#### 1.2 Update Configuration Validator

**Modify:** `internal/config/validator.go`

- Add mode parameter to validation methods
- Allow `.env` defaults in local mode
- Reject insecure defaults in production mode

#### 1.3 Update Commands to Use Mode-Aware Validation

**Files to update:**
- `cmd/crisk/init.go` - Use `ModeProduction` (cloud init)
- `cmd/crisk/init_local.go` - Use `ModeLocal` (local dev)
- `cmd/crisk/check.go` - Auto-detect mode
- `cmd/crisk/parse.go` - Auto-detect mode
- `cmd/crisk/incident.go` - Auto-detect mode

**Example:**
```go
// cmd/crisk/init_local.go
func runInitLocal(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load("")
    if err != nil {
        logging.Fatal("failed to load config", "error", err)
    }

    // Local mode: .env defaults are fine
    cfg.ValidateOrFatal(config.ValidationContextInit, config.ModeLocal)

    // ... rest of init-local
}
```

**Acceptance Criteria:**
- ✅ Local dev: `make dev` works without manual password config
- ✅ Production: Cloud deployments require secure passwords
- ✅ CI/CD: Environment variables required
- ✅ Clear error messages for each mode

---

### Phase 2: Remove Hardcoded Credentials (Production Only) ⏱️ 1 hour

**Goal:** Remove hardcoded credentials from production code paths only

#### 2.1 Fix Production Code Paths

**Files to fix:**

1. **cmd/crisk/incident.go:389** - Remove hardcoded DSN
```go
// BEFORE (SECURITY RISK)
dsn := "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

// AFTER
dsn := buildPostgresDSN(cfg.Storage.PostgresDSN)
if dsn == "" {
    return errors.ConfigError("POSTGRES_DSN is required. Set it in .env or environment")
}
```

2. **cmd/crisk/incident.go:401,403** - Remove hardcoded Neo4j credentials
```go
// BEFORE (SECURITY RISK)
uri := "bolt://localhost:7688"
password := "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

// AFTER
uri := cfg.Neo4j.URI
password := cfg.Neo4j.Password
// Validation already ensures these are set appropriately for the mode
```

3. **cmd/crisk/init.go:247** - Don't print password
```go
// BEFORE (SECURITY RISK)
fmt.Printf("   • Credentials: neo4j / CHANGE_THIS_PASSWORD_IN_PRODUCTION_123\n")

// AFTER
fmt.Printf("   • Credentials: %s / <from .env>\n", cfg.Neo4j.User)
fmt.Printf("   • Browse: http://localhost:%s (Neo4j Browser)\n", neo4jHTTPPort)
```

4. **cmd/crisk/check.go, parse.go** - Use config instead of getEnvOrDefault
```go
// BEFORE
neo4jPassword := getEnvOrDefault("NEO4J_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")

// AFTER
neo4jPassword := cfg.Neo4j.Password  // Already validated
```

#### 2.2 Remove Helper Functions

**Delete deprecated functions in files using them:**
```go
// DEPRECATED - Remove entirely
func getEnvOrDefault(key, defaultValue string) string { ... }
func GetString(key, defaultValue string) string { ... }
func GetInt(key, defaultValue int) int { ... }
```

**Acceptance Criteria:**
- ✅ No passwords in code (verified by `grep -r "CHANGE_THIS_PASSWORD" cmd/`)
- ✅ All credentials from config
- ✅ Build succeeds
- ✅ `make dev` still works seamlessly

---

### Phase 3: Replace Fallback Patterns ⏱️ 2 hours

**Goal:** Log and track errors instead of silently continuing

**Files:** `cmd/crisk/check.go`

#### 3.1 Track Errors in CheckResult

**Add error tracking structure:**
```go
type CheckResult struct {
    Files       []FileResult
    Errors      []CheckError  // NEW: Track all errors
    TotalFiles  int
    HighRisk    int
    MediumRisk  int
    LowRisk     int
}

type CheckError struct {
    File    string
    Stage   string  // "parse", "phase1", "phase2"
    Error   error
    Skipped bool    // Was file skipped due to this error?
}
```

#### 3.2 Log and Track Instead of Silent Continue

**Pattern:**
```go
// BEFORE
if err != nil {
    fmt.Printf("Error: %v\n", err)
    continue  // Silent skip
}

// AFTER
if err != nil {
    logging.Error("failed to process file",
        "file", filePath,
        "stage", "parse",
        "error", err)
    result.Errors = append(result.Errors, CheckError{
        File:    filePath,
        Stage:   "parse",
        Error:   err,
        Skipped: true,
    })
    continue  // Now logged and tracked
}
```

**Locations to fix:**
- Line 226: File parsing errors
- Line 269: Missing API key for high-risk files
- Line 298: LLM client creation errors
- Line 331: Investigation failures

#### 3.3 Fail Fast on Critical Errors

**For infrastructure errors (database, config), fail immediately:**
```go
// BEFORE
if err != nil {
    slog.Warn("temporal client creation failed", "error", err)
    temporalClient = nil  // Continue without temporal
}

// AFTER
temporalClient, err := temporal.NewClient(ctx, cfg.Neo4j)
if err != nil {
    logging.Fatal("failed to create temporal client - graph database required",
        "error", err,
        "neo4j_uri", cfg.Neo4j.URI)
    // Process stops here
}
```

**Acceptance Criteria:**
- ✅ All file-level errors logged and tracked
- ✅ Infrastructure errors stop execution
- ✅ CheckResult includes error summary
- ✅ No silent failures

---

### Phase 4: Fix Ignored Errors ⏱️ 1 hour

**Goal:** Handle all ignored errors properly

#### 4.1 cmd/crisk/configure.go - ReadString Errors (6 instances)

**Pattern:**
```go
// BEFORE
response, _ := reader.ReadString('\n')

// AFTER
response, err := reader.ReadString('\n')
if err != nil {
    if err == io.EOF {
        logging.Warn("stdin closed during input")
        return errors.FileSystemError(err, "stdin closed - please provide input")
    }
    return errors.FileSystemErrorf(err, "failed to read input")
}
response = strings.TrimSpace(response)
```

**Locations:** Lines 75, 89, 106, 148, 171, 200

#### 4.2 cmd/crisk/check.go - strconv.Atoi Errors (3 instances)

**Pattern:**
```go
// BEFORE
port, _ := strconv.Atoi(portStr)  // Defaults to 0 on error

// AFTER
port, err := strconv.Atoi(portStr)
if err != nil {
    logging.Warn("invalid port number, using default",
        "port_string", portStr,
        "default", defaultPort)
    port = defaultPort
}
```

**Locations:** Lines 374, 385, 406

**Acceptance Criteria:**
- ✅ All errors checked
- ✅ Proper fallbacks with logging
- ✅ No underscore error ignores

---

### Phase 5: Dynamic Metadata Extraction ⏱️ 2 hours

**Goal:** Replace hardcoded metadata with actual git queries

#### 5.1 Create Git Metadata Helper

**New file:** `internal/git/metadata.go`

```go
package git

import (
    "os/exec"
    "strings"
)

// GetCurrentBranch returns the current git branch name
func GetCurrentBranch(repoPath string) (string, error) {
    cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

// GetRepoName returns the repository name from git config
func GetRepoName(repoPath string) (string, error) {
    cmd := exec.Command("git", "-C", repoPath, "config", "--get", "remote.origin.url")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }

    // Parse repo name from URL
    url := strings.TrimSpace(string(output))
    // Handle both HTTPS and SSH URLs
    // https://github.com/user/repo.git -> repo
    // git@github.com:user/repo.git -> repo
    parts := strings.Split(url, "/")
    if len(parts) > 0 {
        name := parts[len(parts)-1]
        name = strings.TrimSuffix(name, ".git")
        return name, nil
    }

    return "unknown", nil
}

// GetLinesChanged returns the number of lines changed in a file
func GetLinesChanged(repoPath, filePath string) (int, error) {
    cmd := exec.Command("git", "-C", repoPath, "diff", "--numstat", "HEAD", "--", filePath)
    output, err := cmd.Output()
    if err != nil {
        return 0, err
    }

    // Parse: "added deleted filename"
    parts := strings.Fields(string(output))
    if len(parts) >= 2 {
        added, _ := strconv.Atoi(parts[0])
        deleted, _ := strconv.Atoi(parts[1])
        return added + deleted, nil
    }

    return 0, nil
}
```

#### 5.2 Create Language Detector

**New file:** `internal/language/detector.go`

```go
package language

import (
    "path/filepath"
    "strings"
)

var extensionMap = map[string]string{
    ".go":   "Go",
    ".js":   "JavaScript",
    ".ts":   "TypeScript",
    ".jsx":  "JavaScript",
    ".tsx":  "TypeScript",
    ".py":   "Python",
    ".rb":   "Ruby",
    ".java": "Java",
    ".kt":   "Kotlin",
    ".rs":   "Rust",
    ".c":    "C",
    ".cpp":  "C++",
    ".cc":   "C++",
    ".h":    "C/C++ Header",
    ".hpp":  "C++ Header",
    ".cs":   "C#",
    ".php":  "PHP",
    ".swift":"Swift",
    ".m":    "Objective-C",
    ".scala":"Scala",
    ".sh":   "Shell",
    ".sql":  "SQL",
    ".r":    "R",
}

// Detect returns the language based on file extension
func Detect(filePath string) string {
    ext := strings.ToLower(filepath.Ext(filePath))
    if lang, ok := extensionMap[ext]; ok {
        return lang
    }
    return "unknown"
}
```

#### 5.3 Update Output Converters

**Files to update:**
- `internal/output/converter.go`
- `internal/output/ai_converter.go`

**Changes:**
```go
// BEFORE
Branch: "main",
Language: "unknown",
LinesChanged: 0,
Repository: "local",

// AFTER
Branch: getBranchName(repoPath),
Language: language.Detect(filePath),
LinesChanged: getLinesChanged(repoPath, filePath),
Repository: getRepoName(repoPath),
```

**Acceptance Criteria:**
- ✅ Actual branch name detected
- ✅ Language detected from extension
- ✅ Lines changed from git diff
- ✅ Repo name from git config
- ✅ Graceful fallback if git unavailable

---

### Phase 6: Add Strategic Logging ⏱️ 3 hours

**Goal:** Enable real-time debugging of `crisk init` pipeline

#### 6.1 Initialize Logging in main()

**File:** `cmd/crisk/main.go`

```go
func main() {
    // Detect debug mode
    debugMode := os.Getenv("DEBUG") == "true" || hasDebugFlag()

    // Initialize logging
    var logConfig logging.Config
    if debugMode {
        // Debug mode: verbose, human-readable, stdout only
        logConfig = logging.DebugConfig()
    } else {
        // Production mode: structured JSON, file logging
        logConfig = logging.DefaultConfig(false)
    }

    if err := logging.Initialize(logConfig); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
        os.Exit(1)
    }
    defer logging.Close()

    logging.Info("crisk starting",
        "version", Version,
        "commit", Commit,
        "build_date", BuildDate,
        "debug", debugMode,
        "logfile", logging.GetLogFilePath())

    // Run Cobra command
    if err := rootCmd.Execute(); err != nil {
        logging.Error("command failed", "error", err)
        os.Exit(1)
    }
}

func hasDebugFlag() bool {
    for _, arg := range os.Args {
        if arg == "--debug" || arg == "-d" {
            return true
        }
    }
    return false
}
```

#### 6.2 crisk init-local Pipeline Logging

**File:** `cmd/crisk/init_local.go`

Add logging at each stage:

```go
func runInitLocal(cmd *cobra.Command, args []string) error {
    logging.Info("=== crisk init-local starting ===", "repo", args[0])

    // Stage 1: Configuration
    logging.Info("stage 1: loading configuration")
    cfg, err := config.Load("")
    if err != nil {
        logging.Fatal("failed to load configuration", "error", err)
    }
    cfg.ValidateOrFatal(config.ValidationContextInit, config.ModeLocal)
    logging.Info("configuration validated", "mode", "local")

    // Stage 2: Docker
    logging.Info("stage 2: starting docker services")
    if err := startDockerServices(); err != nil {
        logging.Fatal("docker services failed", "error", err)
    }
    logging.Info("docker services started")

    // Stage 3: Database connection
    logging.Info("stage 3: connecting to databases")
    client, err := connectToDatabases(cfg)
    if err != nil {
        logging.Fatal("database connection failed", "error", err)
    }
    defer client.Close()
    logging.Info("database connection established")

    // Stage 4: Layer 1 ingestion
    logging.Info("stage 4: layer 1 ingestion starting", "repo", repoPath)
    layer1Start := time.Now()
    stats, err := ingestLayer1(client, repoPath)
    if err != nil {
        logging.Fatal("layer 1 ingestion failed", "error", err)
    }
    logging.Info("layer 1 ingestion complete",
        "duration", time.Since(layer1Start),
        "files", stats.Files,
        "functions", stats.Functions,
        "classes", stats.Classes)

    // ... continue for all stages
}
```

#### 6.3 Add Debug Logging

**Add debug logs for expensive operations:**

```go
// Layer 1 ingestion
logging.Debug("parsing file with tree-sitter", "file", filePath)
// ... parse
logging.Debug("file parsed", "file", filePath, "duration", parseDuration)

// Graph queries
logging.Debug("executing cypher query", "query", query, "params", params)
// ... execute
logging.Debug("query complete", "rows", rowCount, "duration", queryDuration)
```

**Acceptance Criteria:**
- ✅ Logging initialized in main()
- ✅ All stages log start/complete with timing
- ✅ Errors logged with full context
- ✅ Debug logs for diagnostics
- ✅ Log file created at `logs/crisk_<timestamp>.log`

---

### Phase 7: Remove TODOs and Placeholders ⏱️ 2 hours

**Goal:** Complete or document all incomplete implementations

#### 7.1 Implement or Remove Placeholders

**Test Ratio LOC (internal/metrics/test_ratio.go:43):**
```go
// Option 1: Query from graph (preferred)
query := `MATCH (f:File {file_path: $path}) RETURN f.loc AS loc`
result, err := client.ExecuteQuery(ctx, query, map[string]interface{}{"path": filePath})
if err != nil {
    logging.Warn("failed to query LOC from graph, using fallback", "error", err)
    sourceLOC = countLinesInFile(filePath)  // Fallback
} else {
    sourceLOC = result.Rows[0]["loc"].(int64)
}

// Option 2: Document as future enhancement
// TODO(Phase 2): Query actual LOC from File nodes once available
sourceLOC := 100  // Placeholder - will be replaced with graph query
logging.Debug("using placeholder LOC", "file", filePath, "loc", sourceLOC)
```

#### 7.2 Document Incomplete Features

**For features not yet implemented, add clear documentation:**

```go
// internal/llm/client.go
func (c *Client) completeAnthropic(ctx context.Context, messages []Message) (*Response, error) {
    // TODO: Implement Anthropic SDK integration
    // Tracked in: https://github.com/rohankatakam/coderisk-go/issues/XXX
    // Expected completion: Q1 2026
    logging.Warn("Anthropic integration not yet implemented, using OpenAI fallback")
    return c.completeOpenAI(ctx, messages)
}
```

**Acceptance Criteria:**
- ✅ All critical TODOs resolved
- ✅ Placeholder values replaced or documented
- ✅ Unimplemented features have clear documentation
- ✅ No "FIXME" comments

---

### Phase 8: End-to-End Testing ⏱️ 2 hours

**Goal:** Validate complete pipeline with live logging

#### Test Protocol

```bash
# 1. Clean slate
make clean-fresh

# 2. Enable debug logging
export DEBUG=true

# 3. Set up required credentials (only these!)
export OPENAI_API_KEY="sk-..."
export GITHUB_TOKEN="ghp_..."

# 4. Run init-local with logging
./bin/crisk init-local omnara-ai/omnara

# 5. Monitor logs in real-time (separate terminal)
tail -f logs/crisk_*.log | grep -E "(ERROR|WARN|INFO)"

# 6. Run check command
./bin/crisk check

# 7. Verify results
cat logs/crisk_*.log | grep "ERROR"  # Should be empty
cat logs/crisk_*.log | grep "WARN"   # Review warnings
```

#### What to Verify

1. **No Manual Password Config Required**
   - ✅ `.env` passwords auto-loaded
   - ✅ No prompts for Neo4j/Postgres passwords
   - ✅ Docker Compose uses `.env` automatically

2. **All Stages Log Progress**
   - ✅ Configuration validation
   - ✅ Docker startup
   - ✅ Database connection
   - ✅ Layer 1, 2, 3 ingestion
   - ✅ Index creation
   - ✅ Completion summary

3. **Errors Are Caught**
   - ✅ Clear error messages
   - ✅ Stack traces in debug mode
   - ✅ Context fields populated
   - ✅ Process stops on fatal errors

4. **Performance Acceptable**
   - ✅ Init completes in <15 min
   - ✅ Check completes in <5 sec
   - ✅ Log file size reasonable (<10MB)

---

## Success Criteria

### Developer Experience
- [ ] `make dev` works without manual password configuration
- [ ] Only `OPENAI_API_KEY` and `GITHUB_TOKEN` need to be set
- [ ] Clear error messages guide users
- [ ] Debug mode provides detailed diagnostics

### Code Quality
- [ ] No hardcoded production credentials
- [ ] No silently ignored errors
- [ ] All errors logged with context
- [ ] Mode-aware configuration validation

### Production Security
- [ ] Insecure defaults rejected in production mode
- [ ] Clear distinction between dev and prod
- [ ] Credentials from environment/keychain only
- [ ] Audit-ready logging

### Testing
- [ ] `make dev` completes successfully
- [ ] `crisk init-local` works without config
- [ ] `crisk check` runs with logging
- [ ] Debug mode provides useful diagnostics
- [ ] Log files contain actionable information

---

## Implementation Order

### Week 1: Core Infrastructure
- **Day 1**: Phase 1 - Context-aware configuration (2h)
- **Day 2**: Phase 2 - Remove hardcoded credentials (1h)
- **Day 2**: Phase 3 - Replace fallback patterns (2h)
- **Day 3**: Phase 4 - Fix ignored errors (1h)
- **Day 3**: Phase 5 - Dynamic metadata (2h)

### Week 2: Logging & Validation
- **Day 1**: Phase 6 - Strategic logging (3h)
- **Day 2**: Phase 7 - Remove TODOs (2h)
- **Day 2**: Phase 8 - E2E testing (2h)

**Total: 15 hours over 2 weeks**

---

## File Tracking

### New Files Created
1. ✅ `internal/logging/logger.go` - Logging infrastructure
2. ✅ `internal/errors/errors.go` - Error handling framework
3. ✅ `internal/config/validator.go` - Configuration validation
4. 🔲 `internal/config/mode.go` - Deployment mode detection
5. 🔲 `internal/git/metadata.go` - Git metadata extraction
6. 🔲 `internal/language/detector.go` - Language detection

### Files to Modify

**Phase 1-2 (Config & Security):**
- `internal/config/validator.go` - Add mode-aware validation
- `cmd/crisk/main.go` - Initialize logging
- `cmd/crisk/init.go` - Use production mode validation
- `cmd/crisk/init_local.go` - Use local mode validation
- `cmd/crisk/check.go` - Auto-detect mode, remove hardcoded defaults
- `cmd/crisk/parse.go` - Remove hardcoded defaults
- `cmd/crisk/incident.go` - Remove hardcoded credentials

**Phase 3-4 (Error Handling):**
- `cmd/crisk/check.go` - Add error tracking, fix ignored errors
- `cmd/crisk/configure.go` - Fix ReadString errors

**Phase 5 (Metadata):**
- `internal/output/converter.go` - Dynamic metadata
- `internal/output/ai_converter.go` - Dynamic metadata

**Phase 6 (Logging):**
- `cmd/crisk/init_local.go` - Add stage logging
- All command files - Add logging

**Phase 7 (Cleanup):**
- `internal/metrics/test_ratio.go` - Resolve placeholder
- `internal/metrics/co_change.go` - Resolve placeholder
- `internal/llm/client.go` - Document unimplemented features

---

## Developer Experience Examples

### Before (❌ Poor DX)
```bash
# Manual password setup required
export NEO4J_PASSWORD="my-secure-password-123"
export POSTGRES_PASSWORD="another-password-456"
export OPENAI_API_KEY="sk-..."
export GITHUB_TOKEN="ghp_..."

make dev  # Still might fail with cryptic errors
```

### After (✅ Great DX)
```bash
# Only explicit configs needed
export OPENAI_API_KEY="sk-..."
export GITHUB_TOKEN="ghp_..."

make dev  # Just works! Passwords auto-configured from .env
```

---

## Next Steps

1. ✅ **Review this revised plan** - Confirms alignment with dev workflow
2. 🔲 **Start Phase 1** - Context-aware configuration
3. 🔲 **Test incrementally** - Verify `make dev` after each phase
4. 🔲 **Document changes** - Update README and .env.example
5. 🔲 **Final E2E test** - Complete pipeline validation

---

**Plan Status:** ✅ READY TO IMPLEMENT
**Next Action:** Implement Phase 1 - Context-Aware Configuration
