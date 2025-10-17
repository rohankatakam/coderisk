# Development Workflow for AI-Assisted Implementation

**Purpose:** Guide for Claude Code agents implementing code changes intelligently and safely
**Last Updated:** October 2, 2025
**Design Philosophy:** Based on [12-factor agents](12-factor-agents-main/README.md) principles and defensive security practices

> This workflow implements guardrails for safe code development: providing AI agents with decision trees, security constraints, architecture compliance checks, and quality gates to maximize code safety while minimizing technical debt and security risks.

---

## Quick Start for AI Agents

**When user asks to implement code changes:**

1. **Read this file first** (you are here)
2. **Check 12-factor principles** (see [Consult 12-Factor Principles](#consult-12-factor-principles) below)
3. **Determine implementation type** (see Decision Tree below)
4. **Read relevant architecture docs** (see Reading Strategy)
5. **Verify security constraints** (see Security Guardrails)
6. **Implement changes** (see Implementation Guidelines)
7. **Run safety checks** (see Quality Gates)
8. **Cite 12-factor principles** where applied (see Citation Guidelines)

---

## Decision Tree: Where Does This Code Go?

```
START: New code or feature to implement
│
├─ Is it a SECURITY-SENSITIVE change?
│  │  (auth, permissions, crypto, data access, API keys)
│  YES → STOP: Get explicit user approval BEFORE implementing
│  │     Ask: "This change affects security. Please confirm implementation approach."
│  │     Then: Follow extra security review (see Security Guardrails)
│  NO  → Continue
│
├─ Does it require ARCHITECTURE CHANGES?
│  │  (new database, new service, graph schema change, API breaking change)
│  YES → STOP: Read architecture docs first
│  │     Read: spec.md + relevant architecture doc
│  │     Verify: Change aligns with constraints
│  │     Create: ADR in 01-architecture/decisions/ if major decision
│  │     Then: Update spec.md if requirements change
│  NO  → Continue
│
├─ Is it CLI COMMAND implementation?
│  │  (crisk init, check, pull, status, config)
│  YES → cmd/crisk/
│  │     ├─ New command? → Create cmd/crisk/{command}.go
│  │     ├─ Update existing? → Edit cmd/crisk/{command}.go
│  │     └─ Update main.go to register command
│  NO  → Continue
│
├─ Is it CORE BUSINESS LOGIC?
│  │  (risk calculation, graph traversal, metric computation, investigation logic)
│  YES → internal/{domain}/
│  │     ├─ Investigation logic? → internal/investigation/
│  │     ├─ Graph operations? → internal/graph/
│  │     ├─ Metric calculation? → internal/metrics/
│  │     ├─ Ingestion/parsing? → internal/ingestion/
│  │     └─ Models/DTOs? → internal/models/
│  NO  → Continue
│
├─ Is it INFRASTRUCTURE/UTILITY code?
│  │  (cache, config, logging, API clients)
│  YES → internal/{utility}/
│  │     ├─ Caching? → internal/cache/
│  │     ├─ Configuration? → internal/config/
│  │     ├─ Logging? → internal/logging/
│  │     └─ API clients? → internal/api/
│  NO  → Continue
│
├─ Is it a TEST?
│  │  (unit test, integration test, test helper)
│  YES → Same directory as code being tested
│  │     ├─ Unit test? → {package}_test.go
│  │     ├─ Integration test? → {package}_integration_test.go
│  │     └─ Test fixtures? → testdata/ subdirectory
│  NO  → Continue
│
├─ Is it a SCRIPT or TOOL?
│  │  (setup script, migration, development helper)
│  YES → scripts/
│  │     ├─ Go script? → scripts/{name}.go
│  │     ├─ Shell script? → scripts/{name}.sh
│  │     └─ Make it executable: chmod +x
│  NO  → Continue
│
└─ Is it DEPLOYMENT configuration?
   │  (Docker, k8s, docker-compose)
   YES → deploy/
         ├─ Docker? → Dockerfile or docker-compose.yml
         ├─ Kubernetes? → deploy/k8s/
         └─ Helm? → deploy/helm/
```

---

## Consult 12-Factor Principles

Before proceeding with development, check if any 12-factor principles apply to your specific implementation. This ensures our code aligns with proven AI agent development practices.

### How to Use 12-Factor Principles

1. **Start here:** Read [12-factor-agents-main/README.md](12-factor-agents-main/README.md) to see all 12 factors
2. **Identify relevant factors** based on your implementation task (see mapping below)
3. **Read applicable factor documents** from [12-factor-agents-main/content/](12-factor-agents-main/content/)
4. **Apply principles** to your code design
5. **Cite the factor** in your code comments or commit message (see Citation Guidelines below)

### Factor Relevance Mapping

Use this guide to determine which factors to consult based on your implementation task:

| Implementation Topic | Relevant 12-Factor Principles | Files to Read |
|---------------------|-------------------------------|---------------|
| **CLI Design** (command parsing, tool calls) | Factor 1: Natural Language to Tool Calls | [factor-01-natural-language-to-tool-calls.md](12-factor-agents-main/content/factor-01-natural-language-to-tool-calls.md) |
| **Prompt Engineering** (LLM prompts, templates) | Factor 2: Own your prompts | [factor-02-own-your-prompts.md](12-factor-agents-main/content/factor-02-own-your-prompts.md) |
| **Context Management** (graph loading, memory) | Factor 3: Own your context window | [factor-03-own-your-context-window.md](12-factor-agents-main/content/factor-03-own-your-context-window.md) |
| **API Design** (structured outputs, DTOs) | Factor 4: Tools are structured outputs | [factor-04-tools-are-structured-outputs.md](12-factor-agents-main/content/factor-04-tools-are-structured-outputs.md) |
| **State Management** (graph vs cache, persistence) | Factor 5: Unify execution state and business state | [factor-05-unify-execution-state.md](12-factor-agents-main/content/factor-05-unify-execution-state.md) |
| **Investigation Logic** (agent loops, decisions) | Factor 8: Own your control flow | [factor-08-own-your-control-flow.md](12-factor-agents-main/content/factor-08-own-your-control-flow.md) |
| **Error Handling** (LLM errors, retry logic) | Factor 9: Compact Errors into Context Window | [factor-09-compact-errors.md](12-factor-agents-main/content/factor-09-compact-errors.md) |
| **Agent Design** (investigation phases, specialization) | Factor 10: Small, Focused Agents | [factor-10-small-focused-agents.md](12-factor-agents-main/content/factor-10-small-focused-agents.md) |
| **API Service** (stateless design, scaling) | Factor 12: Stateless Reducer | [factor-12-stateless-reducer.md](12-factor-agents-main/content/factor-12-stateless-reducer.md) |

**Note:** You don't need to follow every principle rigidly - use them as **guidance and structure** for making informed decisions.

### Citation Guidelines

When you apply a 12-factor principle in code, add a citation like this:

**In code comments:**
```go
// Investigation uses hop-limited graph traversal to control LLM costs
// 12-factor: Factor 3 - Own your context window
func (a *Agent) Investigate(ctx context.Context, file string) (*Risk, error) {
    // ...
}
```

**In commit messages:**
```
Add stateless investigation API

Implements REST API for risk checks with no server-side session state.
All context loaded per-request from graph/cache.

12-factor: Factor 12 - Stateless Reducer
```

---

## Security Guardrails

### CRITICAL: Security Constraints

**NEVER implement code that:**
- ❌ Harvests credentials (SSH keys, API tokens, browser cookies, wallet keys)
- ❌ Executes arbitrary code from user input without sandboxing
- ❌ Bypasses authentication or authorization checks
- ❌ Exposes sensitive data in logs or error messages
- ❌ Implements cryptographic algorithms (use standard libraries)
- ❌ Disables security features (TLS verification, CORS, CSP)
- ❌ Creates backdoors or hidden functionality

**ALWAYS require explicit user approval for:**
- 🔐 Authentication/authorization logic changes
- 🔐 Data access control modifications
- 🔐 API key handling or storage
- 🔐 Encryption/decryption operations
- 🔐 Network security configurations
- 🔐 Third-party service integrations

### Security Review Checklist

Before implementing security-sensitive changes:

- [ ] **Explicit user approval obtained**
- [ ] **Read relevant security section in spec.md**
- [ ] **Verify against OWASP Top 10**
- [ ] **Check for input validation requirements**
- [ ] **Review error handling (no sensitive data leaks)**
- [ ] **Confirm using standard libraries (no custom crypto)**
- [ ] **Test with malicious inputs**

### Input Validation Rules

**For all external inputs (CLI args, API requests, file paths):**

```go
// ✅ GOOD: Validate and sanitize
func ValidateFilePath(path string) error {
    // Check for path traversal
    if strings.Contains(path, "..") {
        return fmt.Errorf("invalid path: contains '..'")
    }

    // Ensure within allowed directory
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    if !strings.HasPrefix(absPath, allowedDir) {
        return fmt.Errorf("path outside allowed directory")
    }

    return nil
}

// ❌ BAD: Direct use of user input
func ReadFile(userPath string) ([]byte, error) {
    return os.ReadFile(userPath) // Path traversal vulnerability!
}
```

---

## Reading Strategy

### Before Making ANY Changes

**ALWAYS read these first:**
1. **[spec.md](spec.md)** - Check requirements and constraints
2. **[dev_docs/README.md](README.md)** - Understand documentation structure
3. **DEVELOPMENT_WORKFLOW.md** - This file (ensures you follow process)

### For Specific Implementation Types

**CLI command implementation:**
```
Read: spec.md section 1.5 (CLI scope)
Read: cmd/crisk/main.go (existing commands)
Read: internal/models/models.go (data structures)
Create: cmd/crisk/{command}.go
Update: cmd/crisk/main.go (register command)
Test: Write cmd/crisk/{command}_test.go
```

**Investigation logic:**
```
Read: spec.md (requirements)
Read: 01-architecture/agentic_design.md (investigation flow)
Read: 01-architecture/graph_ontology.md (graph schema)
Read: internal/investigation/*.go (existing logic)
Update or create: internal/investigation/{feature}.go
Test: internal/investigation/{feature}_test.go
```

**Graph operations:**
```
Read: 01-architecture/graph_ontology.md (schema)
Read: 01-architecture/cloud_deployment.md (Neptune/Neo4j)
Read: internal/graph/*.go (existing operations)
Verify: Graph indexes exist for new queries
Update: internal/graph/{operation}.go
Test: internal/graph/{operation}_test.go
```

**Metric calculation:**
```
Read: 01-architecture/agentic_design.md (metric tiers)
Read: internal/metrics/*.go (existing metrics)
Verify: Metric has FP rate tracking
Create: internal/metrics/{metric}.go
Test: internal/metrics/{metric}_test.go with FP scenarios
```

**API changes:**
```
Read: spec.md section 1.5 (BYOK model)
Read: 01-architecture/cloud_deployment.md (infrastructure)
Read: internal/api/*.go (existing endpoints)
Verify: Backward compatibility or versioning strategy
Update: internal/api/{endpoint}.go
Update: API documentation
Test: Integration tests for API contract
```

**Deployment changes:**
```
Read: 01-architecture/cloud_deployment.md (infrastructure)
Read: 03-implementation/integration_guides/local_deployment.md (Docker)
Read: Existing docker-compose.yml or Dockerfile
Update: Deployment configs
Test: Local deployment validation
Document: Update local_deployment.md if needed
```

---

## Implementation Guidelines

### Rule 1: Architecture Compliance

**CRITICAL:** If implementation affects architecture, MUST align with spec.md constraints.

**Architecture decision checklist:**
- [ ] Read spec.md relevant sections
- [ ] Verify against architectural constraints
- [ ] Check if ADR needed (major technology/pattern decision)
- [ ] If ADR created, update 01-architecture/decisions/README.md
- [ ] If requirements change, update spec.md FIRST

**Example:**
```
User: "Add Redis caching for investigation results"
Agent:
1. Read spec.md section 1.5 (caching in scope)
2. Read 01-architecture/cloud_deployment.md (Redis config)
3. Read 01-architecture/agentic_design.md (cache strategy)
4. Verify: 15-min TTL requirement from agentic_design.md
5. Implement: internal/cache/investigation.go
6. Test: Verify cache invalidation on graph updates
7. Document: Update agentic_design.md if behavior changes
```

### Rule 2: Go Project Structure

**Follow standard Go project layout:**

```
coderisk-go/
├── cmd/
│   └── crisk/              # Main CLI application
│       ├── main.go         # Entry point, command registration
│       ├── init.go         # crisk init command
│       ├── check.go        # crisk check command
│       └── *_test.go       # Command tests
│
├── internal/               # Private application code
│   ├── investigation/      # Risk investigation logic
│   ├── graph/             # Graph database operations
│   ├── metrics/           # Metric calculation
│   ├── ingestion/         # Code parsing, git extraction
│   ├── cache/             # Redis caching
│   ├── config/            # Configuration management
│   ├── api/               # API clients (OpenAI, GitHub)
│   └── models/            # Data models, DTOs
│
├── scripts/               # Build, setup, migration scripts
├── deploy/               # Deployment configs (Docker, k8s)
└── dev_docs/            # Architecture and design docs
```

**Naming conventions:**
- Files: `snake_case.go`
- Packages: `lowercase` (no underscores)
- Types: `PascalCase`
- Functions: `PascalCase` (exported) or `camelCase` (private)
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`

### Rule 3: Code Quality Standards

**Every implementation MUST include:**

1. **Error Handling:**
```go
// ✅ GOOD: Descriptive errors with context
func LoadGraph(ctx context.Context, repoID string) (*Graph, error) {
    graph, err := db.Query(ctx, repoID)
    if err != nil {
        return nil, fmt.Errorf("failed to load graph for repo %s: %w", repoID, err)
    }
    return graph, nil
}

// ❌ BAD: Silent failures or generic errors
func LoadGraph(ctx context.Context, repoID string) *Graph {
    graph, _ := db.Query(ctx, repoID)
    return graph
}
```

2. **Context Propagation:**
```go
// ✅ GOOD: Pass context for cancellation/timeout
func Investigate(ctx context.Context, file string) (*Risk, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // ...
    }
}

// ❌ BAD: No context
func Investigate(file string) (*Risk, error) {
    // Can't be cancelled or timeout
}
```

3. **Logging (structured):**
```go
import "log/slog"

// ✅ GOOD: Structured logging
slog.Info("investigation complete",
    "file", filePath,
    "risk_level", risk.Level,
    "duration_ms", elapsed.Milliseconds(),
)

// ❌ BAD: Unstructured logging
fmt.Printf("Investigation done: %s is %s\n", filePath, risk.Level)
```

4. **Testing:**
```go
// Every .go file MUST have corresponding _test.go
// internal/metrics/coupling.go → internal/metrics/coupling_test.go

func TestCouplingMetric(t *testing.T) {
    tests := []struct {
        name     string
        file     string
        expected int
        wantErr  bool
    }{
        {"no dependencies", "isolated.go", 0, false},
        {"high coupling", "core.go", 15, false},
        {"invalid file", "", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CalculateCoupling(tt.file)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if got != tt.expected {
                t.Errorf("got %d, want %d", got, tt.expected)
            }
        })
    }
}
```

### Rule 4: Dependency Management

**Before adding new dependencies:**

- [ ] Check if functionality exists in standard library
- [ ] Verify dependency is well-maintained (recent commits, issue response)
- [ ] Check license compatibility (Apache 2.0, MIT, BSD)
- [ ] Evaluate bundle size impact
- [ ] Add to go.mod using `go get`
- [ ] Document why dependency is needed (code comment or ADR)

```bash
# ✅ GOOD: Minimal, well-vetted dependencies
go get github.com/neo4j/neo4j-go-driver/v5

# ❌ BAD: Unnecessary dependency for simple task
go get github.com/some-random-pkg/string-utils
# (Use strings package from stdlib instead)
```

### Rule 5: Performance Considerations

**For graph-heavy operations:**

```go
// ✅ GOOD: Batch operations, limit hops
func Load1HopNeighbors(ctx context.Context, fileID string) ([]*Node, error) {
    query := `
        MATCH (f:File {id: $fileId})-[r]-(n)
        RETURN n LIMIT 1000
    ` // Explicit limit prevents unbounded queries

    return db.Query(ctx, query, map[string]any{"fileId": fileID})
}

// ❌ BAD: Unbounded graph traversal
func LoadAllRelated(ctx context.Context, fileID string) ([]*Node, error) {
    query := `
        MATCH (f:File {id: $fileId})-[*]-(n)
        RETURN n
    ` // Could return millions of nodes!
}
```

**For LLM calls:**

```go
// ✅ GOOD: Cache results, limit context
func InvestigateWithLLM(ctx context.Context, evidence []Evidence) (*Assessment, error) {
    cacheKey := fmt.Sprintf("investigation:%s", hashEvidence(evidence))

    // Check cache first
    if cached, ok := cache.Get(cacheKey); ok {
        return cached.(*Assessment), nil
    }

    // Limit context size (12-factor: Factor 3)
    contextTokens := len(evidence) * 100 // rough estimate
    if contextTokens > 4000 {
        evidence = evidence[:40] // truncate
    }

    assessment, err := llm.Call(ctx, evidence)
    if err != nil {
        return nil, err
    }

    cache.Set(cacheKey, assessment, 15*time.Minute)
    return assessment, nil
}
```

---

## Quality Gates

### Before Committing Code

**Run through this checklist:**

- [ ] **Code compiles:** `go build ./...`
- [ ] **Tests pass:** `go test ./...`
- [ ] **No race conditions:** `go test -race ./...`
- [ ] **Code formatted:** `go fmt ./...`
- [ ] **No lint errors:** `golangci-lint run` (if available)
- [ ] **Security check:** `gosec ./...` (if available)
- [ ] **Dependencies tidy:** `go mod tidy`
- [ ] **Documentation updated:** If API/behavior changes
- [ ] **12-factor citations:** Added where principles applied

### Automated Checks (Makefile)

```bash
# Run all quality checks
make test          # Unit tests
make lint          # Code linting
make security      # Security scanning
make integration   # Integration tests (if applicable)
make build         # Final build verification
```

### Test Coverage Requirements

**Minimum coverage by component:**
- CLI commands: 60% (focus on critical paths)
- Business logic (investigation, metrics): 80%
- Graph operations: 70%
- Utilities (cache, config): 60%

```bash
# Check coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Anti-Patterns to Avoid

### ❌ Implementing Without Understanding

**DON'T:**
- Write code before reading spec.md and architecture docs
- Copy-paste code without understanding business logic
- Implement features not in spec.md scope
- Ignore existing patterns in codebase

**DO:**
- Read relevant docs FIRST (spec.md, architecture, existing code)
- Ask user for clarification if requirements unclear
- Verify scope alignment before implementation
- Follow existing patterns and conventions

### ❌ Breaking Architectural Constraints

**DON'T:**
- Add unbounded graph queries (violates hop limits)
- Store session state in API service (violates Factor 12: Stateless)
- Pre-compute all metrics (violates selective calculation design)
- Mix persistent graph with ephemeral cache data

**DO:**
- Respect hop limits (max 3 hops in investigation)
- Keep API service stateless (load context per request)
- Calculate metrics on-demand (LLM-guided selection)
- Use graph for persistent, Redis for cache (Factor 5)

### ❌ Security Vulnerabilities

**DON'T:**
- Use user input directly in file paths (path traversal)
- Log sensitive data (API keys, tokens, file contents)
- Implement custom crypto (use standard libraries)
- Trust external data without validation

**DO:**
- Sanitize and validate ALL inputs
- Use allowlists, not denylists
- Leverage Go standard library for security
- Assume all external data is malicious

### ❌ Poor Error Handling

**DON'T:**
- Swallow errors silently (`err := foo(); _ = err`)
- Return generic errors (`return errors.New("error")`)
- Panic in library code
- Expose internal details in user-facing errors

**DO:**
- Propagate errors with context (`fmt.Errorf("failed to X: %w", err)`)
- Provide actionable error messages for users
- Use sentinel errors for expected failures
- Log internal details, show safe messages to users

### ❌ Ignoring Performance

**DON'T:**
- Load entire graph into memory
- Make LLM calls in loops without caching
- Use unbounded queries
- Block on I/O without timeouts

**DO:**
- Load minimal subgraphs (1-hop, then expand if needed)
- Cache LLM results (15-min TTL)
- Set explicit limits in queries
- Use context timeouts for all I/O

---

## AI Agent Workflow

### Step-by-Step Process

**1. Understand the Request**
```
User request → Parse intent → Classify implementation type
Example: "Add ownership churn metric calculation"
Classification: Metric implementation (Tier 2)
```

**2. Read Context**
```
Read: spec.md (metric requirements, FP rate constraints)
Read: 01-architecture/agentic_design.md (Tier 2 metrics, on-demand calculation)
Read: internal/metrics/*.go (existing metrics, patterns)
Read: 12-factor-agents-main/content/factor-08-own-your-control-flow.md (decision logic)
```

**3. Determine Implementation Plan**
```
Decision tree result:
- Security-sensitive? NO (metric calculation)
- Architecture change? NO (fits existing metric framework)
- Where? internal/metrics/ownership_churn.go
- Tests? internal/metrics/ownership_churn_test.go
- Dependencies? Use existing graph client
```

**4. Implement Code**
```
1. Create internal/metrics/ownership_churn.go
2. Implement CalculateOwnershipChurn(ctx, fileID) (float64, error)
3. Add FP rate tracking integration
4. Write comprehensive tests (including FP scenarios)
5. Update internal/metrics/registry.go to register metric
```

**5. Run Quality Checks**
```
1. go build ./...
2. go test ./internal/metrics/
3. go test -race ./...
4. go fmt ./...
5. Verify coverage: go test -cover ./internal/metrics/
```

**6. Update Documentation**
```
1. Add metric to 01-architecture/agentic_design.md (Tier 2 list)
2. Add code comment citing 12-factor: Factor 8 (on-demand calculation)
3. No spec.md update needed (within existing scope)
```

**7. Commit**
```bash
git add internal/metrics/ownership_churn.go
git add internal/metrics/ownership_churn_test.go
git commit -m "Add ownership churn metric (Tier 2)

Implements on-demand calculation of ownership transition risk.
LLM requests this metric during Phase 2 investigation.

12-factor: Factor 8 - Own your control flow (selective calculation)
Includes FP rate tracking for auto-disable if >3% FP rate.

Tests: 85% coverage including FP scenarios"
```

---

## Development Commands Reference

### Build & Test
```bash
# Build CLI
go build -o bin/crisk ./cmd/crisk

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...

# Run specific test
go test -run TestOwnershipChurn ./internal/metrics/
```

### Code Quality
```bash
# Format code
go fmt ./...

# Tidy dependencies
go mod tidy

# Vet code
go vet ./...

# Lint (if golangci-lint installed)
golangci-lint run

# Security scan (if gosec installed)
gosec ./...
```

### Local Development
```bash
# Start local stack (Neo4j, Redis, Postgres)
docker compose up -d

# Run CLI against local deployment
export CRISK_API_URL=http://localhost:8080
go run ./cmd/crisk init

# View logs
docker compose logs -f api

# Stop stack
docker compose down
```

### Makefile Shortcuts
```bash
make build        # Build CLI
make test         # Run all tests
make lint         # Run linters
make security     # Security scan
make integration  # Integration tests
make clean        # Clean build artifacts
```

---

## Example Implementations

### Example 1: Adding a New CLI Command

```go
// cmd/crisk/feedback.go

package main

import (
    "context"
    "fmt"
    "time"

    "github.com/coderisk/coderisk-go/internal/api"
    "github.com/coderisk/coderisk-go/internal/models"
    "github.com/spf13/cobra"
)

// feedbackCmd implements user feedback for metric validation
// 12-factor: Factor 7 - Contact humans with tool calls (metric refinement)
var feedbackCmd = &cobra.Command{
    Use:   "feedback",
    Short: "Provide feedback on risk assessment accuracy",
    Long: `Submit feedback when risk assessment is incorrect.
This helps improve metric accuracy by tracking false positive rates.`,
    RunE: runFeedback,
}

func init() {
    feedbackCmd.Flags().Bool("false-positive", false, "Mark as false positive")
    feedbackCmd.Flags().String("reason", "", "Reason for feedback (required)")
    rootCmd.AddCommand(feedbackCmd)
}

func runFeedback(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Get flags
    isFP, _ := cmd.Flags().GetBool("false-positive")
    reason, _ := cmd.Flags().GetString("reason")

    if reason == "" {
        return fmt.Errorf("--reason is required")
    }

    // Get last risk check from context (cached)
    lastCheck, err := getLastRiskCheck(ctx)
    if err != nil {
        return fmt.Errorf("no recent risk check found: %w", err)
    }

    // Submit feedback
    client := api.NewClient(config.APIEndpoint, config.APIKey)
    feedback := &models.Feedback{
        FileID:        lastCheck.FileID,
        RiskLevel:     lastCheck.RiskLevel,
        IsFalsePositive: isFP,
        Reason:        reason,
        Timestamp:     time.Now(),
    }

    if err := client.SubmitFeedback(ctx, feedback); err != nil {
        return fmt.Errorf("failed to submit feedback: %w", err)
    }

    fmt.Println("✓ Feedback submitted successfully")
    fmt.Printf("  This helps improve metric accuracy.\n")
    return nil
}
```

### Example 2: Implementing a New Metric

```go
// internal/metrics/incident_similarity.go

package metrics

import (
    "context"
    "fmt"
    "math"

    "github.com/coderisk/coderisk-go/internal/graph"
    "github.com/coderisk/coderisk-go/internal/models"
)

// IncidentSimilarity calculates cosine similarity between current change
// and historical incidents (Tier 2 metric, on-demand only)
// 12-factor: Factor 8 - Own your control flow (LLM-requested metric)
type IncidentSimilarity struct {
    graph *graph.Client
    cache *cache.Manager
}

// Calculate computes incident similarity score [0-1]
func (m *IncidentSimilarity) Calculate(ctx context.Context, fileID string) (*Result, error) {
    // Check cache first (15-min TTL)
    cacheKey := fmt.Sprintf("metric:incident_similarity:%s", fileID)
    if cached, ok := m.cache.Get(cacheKey); ok {
        return cached.(*Result), nil
    }

    // Load historical incidents (1-hop from file)
    incidents, err := m.graph.GetIncidents(ctx, fileID)
    if err != nil {
        return nil, fmt.Errorf("failed to load incidents: %w", err)
    }

    if len(incidents) == 0 {
        return &Result{
            Name:     "incident_similarity",
            Value:    0.0,
            Evidence: "No historical incidents found",
            FPRate:   0.0, // No incidents = no FP risk
        }, nil
    }

    // Calculate cosine similarity with most recent incident
    mostRecent := incidents[0]
    similarity := m.cosineSimilarity(ctx, fileID, mostRecent.ID)

    result := &Result{
        Name:     "incident_similarity",
        Value:    similarity,
        Evidence: fmt.Sprintf("%.2f similarity to incident #%s: %q",
            similarity, mostRecent.ID, mostRecent.Title),
        FPRate:   m.getFPRate(), // Track from Postgres
    }

    // Cache result
    m.cache.Set(cacheKey, result, 15*time.Minute)
    return result, nil
}

func (m *IncidentSimilarity) cosineSimilarity(ctx context.Context, fileID, incidentID string) float64 {
    // Compute cosine similarity based on:
    // - File change patterns (lines modified, functions affected)
    // - Temporal proximity (recent changes weighted higher)
    // - Developer overlap (same author = higher similarity)

    // ... implementation ...
    return 0.0 // placeholder
}

func (m *IncidentSimilarity) getFPRate() float64 {
    // Query Postgres for FP rate from user feedback
    // ... implementation ...
    return 0.02 // 2% FP rate example
}
```

---

## Contact & Questions

**If this guide is unclear:**
- Add clarification to this document
- Update examples to be more specific
- Create issue/discussion for team review

**For architecture questions:**
- Check [spec.md](spec.md) first
- Review relevant architecture docs in [01-architecture/](01-architecture/)
- Consult 12-factor principles in [12-factor-agents-main/](12-factor-agents-main/)

**Remember:** This guide should evolve. If you find better patterns, update this document.

---

**This workflow ensures:**
✅ Safe, secure code implementation
✅ Architecture compliance with spec.md
✅ 12-factor principle alignment
✅ High code quality and test coverage
✅ AI agents can implement intelligently
✅ Human reviewers can easily verify
