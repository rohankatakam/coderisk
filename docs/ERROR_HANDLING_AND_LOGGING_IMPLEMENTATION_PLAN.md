# Error Handling and Logging Implementation Plan

**Project:** CodeRisk
**Date:** October 2025
**Status:** 🚧 IN PROGRESS
**Priority:** 🔴 CRITICAL

---

## Executive Summary

This document outlines the comprehensive plan to replace all error handling anti-patterns, fallbacks, mocks, and hardcoded values with proper error handling and strategic logging throughout the CodeRisk codebase. This is critical for production readiness and real-time debugging.

### What We've Built (Infrastructure)

1. **Centralized Logging System** - [internal/logging/logger.go](internal/logging/logger.go)
   - Debug/Production mode support
   - File logging with automatic rotation
   - Structured logging (JSON in production, human-readable in debug)
   - Log levels: DEBUG, INFO, WARN, ERROR, FATAL

2. **Error Handling Framework** - [internal/errors/errors.go](internal/errors/errors.go)
   - Typed errors with context and stack traces
   - Severity levels (Low, Medium, High, Critical)
   - No silent failures - every error either returns or logs
   - Fatal errors stop execution

3. **Configuration Validation** - [internal/config/validator.go](internal/config/validator.go)
   - Required environment variables enforced
   - No hardcoded defaults for sensitive values
   - Context-aware validation (init, check, incident, parse)
   - Clear error messages for missing config

---

## Critical Issues Found

### 🔴 CRITICAL SECURITY ISSUES (Must Fix Immediately)

| Priority | File | Line | Issue | Impact |
|----------|------|------|-------|--------|
| **P0** | [cmd/crisk/incident.go](cmd/crisk/incident.go) | 389 | Hardcoded DSN with password | **EXPOSED CREDENTIALS** |
| **P0** | [cmd/crisk/incident.go](cmd/crisk/incident.go) | 403 | Hardcoded password | **EXPOSED CREDENTIALS** |
| **P0** | [cmd/crisk/init.go](cmd/crisk/init.go) | 247 | Password printed to console | **EXPOSED CREDENTIALS** |
| **P0** | [cmd/crisk/check.go](cmd/crisk/check.go) | 367 | Default password in code | **SECURITY RISK** |
| **P0** | [cmd/crisk/parse.go](cmd/crisk/parse.go) | 175 | Hardcoded password default | **SECURITY RISK** |

### 🟡 HIGH PRIORITY ERROR HANDLING ISSUES

| Priority | File | Issue | Count | Impact |
|----------|------|-------|-------|--------|
| **P1** | [cmd/crisk/configure.go](cmd/crisk/configure.go) | Ignored `ReadString` errors | 6+ | Silent failures on stdin errors |
| **P1** | [cmd/crisk/check.go](cmd/crisk/check.go) | Silent continue patterns | 4 | Files skipped without escalation |
| **P1** | Multiple files | TODO/FIXME comments | 26 | Incomplete implementations |
| **P1** | [internal/metrics/](internal/metrics/) | Cache errors marked "non-fatal" | 4+ | Degraded without warning |

---

## Implementation Phases

### Phase 1: Fix Critical Security Issues ⏱️ 1 hour

**Priority:** 🔴 CRITICAL
**Must Complete Before:** Any production deployment

#### Tasks:

1. **Remove all hardcoded passwords** (5 locations)
   - [cmd/crisk/incident.go:389](cmd/crisk/incident.go#L389) - Replace hardcoded DSN
   - [cmd/crisk/incident.go:403](cmd/crisk/incident.go#L403) - Remove hardcoded password
   - [cmd/crisk/init.go:247](cmd/crisk/init.go#L247) - Stop printing credentials
   - [cmd/crisk/check.go:367](cmd/crisk/check.go#L367) - Remove default password
   - [cmd/crisk/parse.go:175](cmd/crisk/parse.go#L175) - Remove default password

2. **Enforce configuration validation**
   - Update all commands to call `config.ValidateOrFatal()` before execution
   - Remove `GetEnvOrDefault()` function entirely
   - Use config validator for all sensitive values

3. **Update .env.example**
   - Add clear documentation that passwords MUST be set
   - Remove any default passwords
   - Add security warnings

**Acceptance Criteria:**
- ✅ No passwords in code (verified by grep)
- ✅ All sensitive config validated before use
- ✅ Build succeeds
- ✅ Security audit passes

---

### Phase 2: Replace Fallback Patterns ⏱️ 2 hours

**Priority:** 🟡 HIGH
**Files to Fix:**

#### 2.1 [cmd/crisk/check.go](cmd/crisk/check.go) - Silent Continue Patterns

**Current Anti-Pattern:**
```go
// Line 226
if err != nil {
    fmt.Printf("Error: %v\n", err)
    continue  // File silently skipped
}
```

**Fix:**
```go
if err != nil {
    logging.Error("failed to parse file",
        "file", absPath,
        "error", err)
    result.Errors = append(result.Errors, CheckError{
        File: absPath,
        Error: errors.FileSystemErrorf(err, "failed to parse file %s", absPath),
    })
    continue  // Now logged and tracked
}
```

**Locations:**
- Line 226: File parsing error → Log and track
- Line 269: High risk without API key → Log and track, don't silently skip
- Line 298: LLM client error → Log and track
- Line 331: Investigation failed → Log and track

#### 2.2 [cmd/crisk/check.go](cmd/crisk/check.go) - Temporal Client Fallback

**Current Anti-Pattern:**
```go
// Line 282-286
if err != nil {
    slog.Warn("temporal client creation failed", "error", err)
    temporalClient = nil // Continue without temporal data
}
```

**Fix:**
```go
if err != nil {
    return errors.DatabaseErrorf(err,
        "failed to create temporal client - database connection required")
}
// Don't continue with nil temporal client - fail fast
```

**Acceptance Criteria:**
- ✅ All continue patterns log errors
- ✅ Errors tracked in CheckResult
- ✅ No silent skips
- ✅ Fatal errors stop execution

---

### Phase 3: Fix Ignored Errors ⏱️ 1 hour

**Priority:** 🟡 HIGH
**File:** [cmd/crisk/configure.go](cmd/crisk/configure.go)

#### 3.1 ReadString Error Handling

**Current Anti-Pattern:**
```go
response, _ := reader.ReadString('\n')  // Error ignored
```

**Fix:**
```go
response, err := reader.ReadString('\n')
if err != nil {
    if err == io.EOF {
        logging.Warn("stdin closed unexpectedly")
        return "", errors.FileSystemError(err, "stdin closed during input")
    }
    return "", errors.FileSystemErrorf(err, "failed to read from stdin")
}
```

**Locations (6 instances):**
- Line 75: Neo4j URI input
- Line 89: Neo4j user input
- Line 106: Neo4j password input
- Line 148: PostgreSQL host input
- Line 171: PostgreSQL password input
- Line 200: GitHub token input

#### 3.2 strconv.Atoi Error Handling

**File:** [cmd/crisk/check.go](cmd/crisk/check.go)

**Current Anti-Pattern:**
```go
port, _ := strconv.Atoi(portStr)  // Error ignored, defaults to 0
```

**Fix:**
```go
port, err := strconv.Atoi(portStr)
if err != nil {
    return errors.ValidationErrorf("invalid port number: %s", portStr)
}
```

**Locations (3 instances):**
- Line 374: Redis port
- Line 385: Postgres port
- Line 406: Neo4j port

**Acceptance Criteria:**
- ✅ All ReadString errors handled
- ✅ All strconv errors handled
- ✅ Proper error messages
- ✅ No underscore error ignores

---

### Phase 4: Replace Hardcoded Values ⏱️ 3 hours

**Priority:** 🟡 HIGH

#### 4.1 Default Localhost Values

**Current Issues:**
- Multiple files default to `localhost` for production services
- Port numbers hardcoded
- SSL disabled by default

**Fix Strategy:**
1. Remove all localhost defaults
2. Require explicit configuration
3. Validate URIs/DSNs at startup

**Files to Update:**
- [cmd/crisk/parse.go:173-176](cmd/crisk/parse.go#L173-L176) - Neo4j defaults
- [cmd/crisk/init.go:81,97](cmd/crisk/init.go#L81) - Postgres/Neo4j defaults
- [cmd/crisk/check.go:366-410](cmd/crisk/check.go#L366-L410) - Multiple service defaults
- [cmd/crisk/incident.go:389,401](cmd/crisk/incident.go#L389) - Hardcoded DSN/URI

#### 4.2 Branch Names and Metadata

**Current Issues:**
- Branch hardcoded to "main"
- Language set to "unknown"
- Repository name hardcoded to "local"

**Files to Fix:**
- [internal/output/converter.go](internal/output/converter.go) - Branch, Language
- [internal/output/ai_converter.go](internal/output/ai_converter.go) - Repository

**Fix:**
```go
// Get actual branch name
branch, err := git.GetCurrentBranch(repoPath)
if err != nil {
    logging.Warn("failed to get branch name", "error", err)
    branch = "unknown"
}

// Detect language from file extension
language := detectLanguage(filePath)

// Get repository name from git config
repoName, err := git.GetRepoName(repoPath)
if err != nil {
    logging.Warn("failed to get repo name", "error", err)
    repoName = "local"
}
```

**Acceptance Criteria:**
- ✅ No hardcoded localhost values
- ✅ All configuration validated
- ✅ Dynamic metadata extraction
- ✅ Proper error handling for missing data

---

### Phase 5: Remove Placeholder Values ⏱️ 2 hours

**Priority:** 🟠 MEDIUM

#### 5.1 Test Ratio LOC Placeholder

**File:** [internal/metrics/test_ratio.go:43](internal/metrics/test_ratio.go#L43)

**Current:**
```go
sourceLOC := 100 // Placeholder
```

**Fix:**
```go
// Query actual LOC from File nodes
query := `
    MATCH (f:File {file_path: $file_path})
    RETURN f.loc AS loc
`
result, err := client.ExecuteQuery(ctx, query, map[string]interface{}{
    "file_path": filePath,
})
if err != nil {
    return 0, errors.DatabaseErrorf(err, "failed to query LOC for %s", filePath)
}
```

#### 5.2 Co-Change Frequency Placeholder

**File:** [internal/metrics/co_change.go:77](internal/metrics/co_change.go#L77)

**Fix:**
```go
// Use actual edge.frequency property from CO_CHANGED relationship
query := `
    MATCH (f1:File {file_path: $file1})-[r:CO_CHANGED]-(f2:File {file_path: $file2})
    RETURN r.frequency AS frequency
`
```

#### 5.3 Incomplete Implementations

**Files:**
- [internal/llm/client.go:167-172](internal/llm/client.go#L167-L172) - Anthropic SDK
- [internal/cache/manager.go:53-83](internal/cache/manager.go#L53-L83) - Cache operations

**Strategy:**
- Complete implementations or remove unused code
- Add proper error messages for unimplemented features
- Document what's not yet implemented

**Acceptance Criteria:**
- ✅ No placeholder LOC values
- ✅ Co-change uses actual edge properties
- ✅ Unimplemented features documented
- ✅ Clear errors for missing features

---

### Phase 6: Add Strategic Logging ⏱️ 4 hours

**Priority:** 🟠 MEDIUM

#### 6.1 Initialize Logging in main()

**File:** [cmd/crisk/main.go](cmd/crisk/main.go)

```go
func main() {
    // Initialize logging based on debug flag
    debugMode := os.Getenv("DEBUG") == "true" || hasFlag("--debug")

    var logConfig logging.Config
    if debugMode {
        logConfig = logging.DebugConfig()
    } else {
        logConfig = logging.DefaultConfig(false)
    }

    if err := logging.Initialize(logConfig); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
        os.Exit(1)
    }
    defer logging.Close()

    logging.Info("crisk starting",
        "version", version,
        "debug", debugMode,
        "logfile", logging.GetLogFilePath())

    // ... rest of main
}
```

#### 6.2 crisk init Pipeline Logging

**Stages to Log:**

1. **Stage 0: Configuration Validation**
```go
logging.Info("validating configuration")
if err := cfg.ValidateOrFatal(config.ValidationContextInit); err != nil {
    logging.Fatal("configuration validation failed", "error", err)
}
logging.Info("configuration validated successfully")
```

2. **Stage 1: Docker Startup**
```go
logging.Info("starting docker services")
// ... docker compose up
logging.Info("docker services started", "services", ["neo4j", "postgres", "redis"])
```

3. **Stage 2: Database Connection**
```go
logging.Info("connecting to neo4j", "uri", cfg.Neo4j.URI)
client, err := graph.NewClient(ctx, cfg.Neo4j)
if err != nil {
    logging.Fatal("failed to connect to neo4j", "error", err)
}
logging.Info("neo4j connection established")
```

4. **Stage 3: Layer 1 Ingestion**
```go
logging.Info("starting layer 1 ingestion", "repo", repoPath)
logging.Debug("parsing files with tree-sitter")
// ... ingestion
logging.Info("layer 1 ingestion complete",
    "files", fileCount,
    "functions", functionCount,
    "duration", duration)
```

5. **Stage 3.5: Index Creation**
```go
logging.Info("creating neo4j indexes")
// ... index creation
logging.Info("indexes created successfully", "count", indexCount)
```

6. **Stage 4: Layer 2 Ingestion**
```go
logging.Info("starting layer 2 ingestion (git history)")
logging.Debug("fetching commits from git")
// ... ingestion
logging.Info("layer 2 ingestion complete",
    "commits", commitCount,
    "developers", devCount,
    "duration", duration)
```

7. **Stage 5: Layer 3 Ingestion**
```go
logging.Info("starting layer 3 ingestion (incidents)")
// ... ingestion
logging.Info("layer 3 ingestion complete",
    "incidents", incidentCount,
    "duration", duration)
```

#### 6.3 Error Path Logging

**Key Principle:** Every error path must log before returning

```go
// Good: Log before returning error
if err := someOperation(); err != nil {
    logging.Error("operation failed",
        "operation", "someOperation",
        "error", err,
        "context", additionalContext)
    return errors.Wrap(err, ...)
}

// Bad: Silent error return
if err := someOperation(); err != nil {
    return err  // ❌ No logging
}
```

**Acceptance Criteria:**
- ✅ Logging initialized in main()
- ✅ All crisk init stages log progress
- ✅ All errors logged before returning
- ✅ Debug logs for detailed diagnostics
- ✅ Log file created and rotated properly

---

### Phase 7: Update Commands to Use New Infrastructure ⏱️ 3 hours

**Priority:** 🟢 LOW (after Phase 1-6)

#### 7.1 Update All Commands

**Pattern for Each Command:**

```go
func runInit(cmd *cobra.Command, args []string) error {
    // 1. Load configuration
    cfg, err := config.Load("")
    if err != nil {
        logging.Fatal("failed to load configuration", "error", err)
    }

    // 2. Validate configuration for this command
    cfg.ValidateOrFatal(config.ValidationContextInit)

    // 3. Execute command with proper logging
    logging.Info("starting init command", "args", args)

    if err := executeInit(cfg, args); err != nil {
        logging.Error("init command failed", "error", err)
        return err
    }

    logging.Info("init command completed successfully")
    return nil
}
```

**Commands to Update:**
- [cmd/crisk/init.go](cmd/crisk/init.go)
- [cmd/crisk/init_local.go](cmd/crisk/init_local.go)
- [cmd/crisk/check.go](cmd/crisk/check.go)
- [cmd/crisk/parse.go](cmd/crisk/parse.go)
- [cmd/crisk/incident.go](cmd/crisk/incident.go)
- [cmd/crisk/configure.go](cmd/crisk/configure.go)

**Acceptance Criteria:**
- ✅ All commands use config validator
- ✅ All commands initialize logging
- ✅ All commands log start/completion
- ✅ All commands handle errors properly

---

## Testing Strategy

### Phase 8: End-to-End Testing ⏱️ 2 hours

**Objective:** Run `crisk init` from start to finish with live logging and fix issues in real-time

#### Test Environment Setup

```bash
# 1. Enable debug mode
export DEBUG=true

# 2. Clear existing logs
rm -rf logs/

# 3. Set up test configuration (NO DEFAULTS)
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="your-actual-password"
export NEO4J_DATABASE="neo4j"

export POSTGRES_DSN="postgres://user:pass@localhost:5432/crisk?sslmode=require"

# Optional
export GITHUB_TOKEN="your-github-token"
export OPENAI_API_KEY="your-openai-key"
```

#### Test Execution

```bash
# Run init with logging
./crisk init omnara-ai/omnara

# In another terminal, tail the log file
tail -f logs/crisk_*.log | grep -E "(ERROR|FATAL|WARN)"
```

#### What to Look For

1. **Configuration Validation**
   - ✅ All required env vars checked
   - ✅ Clear error messages if missing
   - ✅ No hardcoded defaults used

2. **Error Handling**
   - ✅ All errors logged before returning
   - ✅ No silent continues
   - ✅ Fatal errors stop execution
   - ✅ Stack traces in debug mode

3. **Progress Tracking**
   - ✅ Each stage logs start/completion
   - ✅ File counts, durations logged
   - ✅ Clear indication of current operation

4. **Error Recovery**
   - When an error occurs:
     - ✅ Clear error message in log
     - ✅ Stack trace available
     - ✅ Context (file, operation, etc.) logged
     - ✅ Process stops (no silent continuation)

#### Real-Time Debugging Protocol

**When an error occurs:**

1. **Check the log file** - Find the ERROR/FATAL entry
2. **Review context** - Look at context fields (file, operation, etc.)
3. **Check stack trace** - Identify exact location
4. **Decide on fix:**
   - Add fallback? (Only if truly non-fatal)
   - Fix implementation? (If logic is wrong)
   - Improve error message? (If unclear)
   - Add validation? (If missing config)

5. **Apply fix** - Update code
6. **Re-run** - Test again
7. **Verify** - Check log for clean execution

**Document each fix in a session log**

---

## Success Criteria

### Phase 1-7 Completion

- [ ] All 5 hardcoded passwords removed
- [ ] All 6 ReadString errors handled
- [ ] All 3 strconv errors handled
- [ ] All 4 silent continue patterns fixed
- [ ] All 20+ hardcoded config values removed
- [ ] All 8 placeholder values replaced or documented
- [ ] Logging infrastructure integrated in all commands
- [ ] Configuration validation enforced

### End-to-End Testing

- [ ] `crisk init` runs successfully with debug logging
- [ ] All stages log progress
- [ ] All errors are caught and logged
- [ ] No silent failures
- [ ] Log file created and contains useful information
- [ ] Can diagnose and fix issues from log file alone

### Code Quality

- [ ] No TODO/FIXME comments for critical functionality
- [ ] All functions document error returns
- [ ] All errors include context
- [ ] Build succeeds with no warnings
- [ ] All tests pass

---

## File Tracking

### Files Created

1. [internal/logging/logger.go](internal/logging/logger.go) - Centralized logging
2. [internal/errors/errors.go](internal/errors/errors.go) - Error handling framework
3. [internal/config/validator.go](internal/config/validator.go) - Configuration validation
4. This document - Implementation plan

### Files to Modify (Phase 1-7)

**Priority 0 (Security):**
- [cmd/crisk/incident.go](cmd/crisk/incident.go) (lines 389, 403)
- [cmd/crisk/init.go](cmd/crisk/init.go) (line 247)
- [cmd/crisk/check.go](cmd/crisk/check.go) (line 367)
- [cmd/crisk/parse.go](cmd/crisk/parse.go) (line 175)

**Priority 1 (Error Handling):**
- [cmd/crisk/configure.go](cmd/crisk/configure.go) (6 error ignores)
- [cmd/crisk/check.go](cmd/crisk/check.go) (4 silent continues, 3 strconv ignores)
- [internal/metrics/test_ratio.go](internal/metrics/test_ratio.go)
- [internal/metrics/co_change.go](internal/metrics/co_change.go)

**Priority 2 (Hardcoded Values):**
- [cmd/crisk/parse.go](cmd/crisk/parse.go)
- [cmd/crisk/init.go](cmd/crisk/init.go)
- [cmd/crisk/init_local.go](cmd/crisk/init_local.go)
- [internal/output/converter.go](internal/output/converter.go)
- [internal/output/ai_converter.go](internal/output/ai_converter.go)

**Priority 3 (Logging Integration):**
- [cmd/crisk/main.go](cmd/crisk/main.go)
- All command files (init, check, parse, incident, configure)

### Files to Create (Phase 6-7)

- `internal/git/metadata.go` - Git metadata extraction
- `internal/language/detector.go` - Language detection
- Testing log for E2E session

---

## Timeline

| Phase | Duration | Priority | Dependencies |
|-------|----------|----------|--------------|
| **Phase 1: Security** | 1 hour | 🔴 CRITICAL | None |
| **Phase 2: Fallbacks** | 2 hours | 🟡 HIGH | Phase 1 |
| **Phase 3: Ignored Errors** | 1 hour | 🟡 HIGH | Phase 1 |
| **Phase 4: Hardcoded Values** | 3 hours | 🟡 HIGH | Phase 1 |
| **Phase 5: Placeholders** | 2 hours | 🟠 MEDIUM | Phase 1-4 |
| **Phase 6: Logging** | 4 hours | 🟠 MEDIUM | Phase 1-5 |
| **Phase 7: Commands** | 3 hours | 🟢 LOW | Phase 1-6 |
| **Phase 8: E2E Testing** | 2 hours | 🟡 HIGH | Phase 1-7 |
| **Total** | **18 hours** | | |

---

## Next Steps

1. **Review this plan** - Confirm approach and priorities
2. **Start Phase 1** - Fix critical security issues immediately
3. **Continue sequentially** - Complete each phase before moving to next
4. **Test incrementally** - Build and test after each phase
5. **Document fixes** - Keep session log of all changes
6. **Final E2E test** - Run complete pipeline with logging

---

## References

- [NEO4J_IMPLEMENTATION_STATUS_REPORT.md](NEO4J_IMPLEMENTATION_STATUS_REPORT.md) - Recently completed Neo4j modernization
- Anti-pattern analysis (see full report in exploration agent output)
- [internal/logging/logger.go](internal/logging/logger.go) - Logging infrastructure
- [internal/errors/errors.go](internal/errors/errors.go) - Error handling framework
- [internal/config/validator.go](internal/config/validator.go) - Configuration validation

---

**Plan Status:** ✅ COMPLETE - Ready to Execute
**Next Action:** Start Phase 1 - Fix Critical Security Issues
