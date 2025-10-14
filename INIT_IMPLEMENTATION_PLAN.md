# CodeRisk `init` Command - Full 3-Layer Implementation Plan

**Status:** Layer 1 missing, Layers 2 & 3 implemented
**Goal:** Integrate all 3 layers into production `crisk init` command
**Timeline:** 2-3 hours

---

## Current State Analysis

### ✅ What's Working (Layers 2 & 3)

**Layer 2: Temporal (GitHub API)**
- `internal/github/fetcher.go` - Fetches commits, issues, PRs
- `internal/temporal/*.go` - Co-change analysis
- `internal/graph/builder.go:processCommits()` - Creates commit/developer nodes
- **Status:** IMPLEMENTED & WORKING

**Layer 3: Incidents (GitHub Issues)**
- `internal/github/fetcher.go:FetchIssues()` - Fetches issues with labels
- `internal/graph/builder.go:processIssues()` - Creates incident nodes
- **Status:** IMPLEMENTED & WORKING

### ❌ What's Missing (Layer 1)

**Layer 1: Structure (Tree-sitter)**
- `internal/ingestion/processor.go` - EXISTS but NOT CALLED from `init`
- `internal/treesitter/*.go` - Parser EXISTS
- **Status:** IMPLEMENTED but NOT INTEGRATED into `cmd/crisk/init.go`

**Current Flow:**
```
init command:
  1. Fetch GitHub data → PostgreSQL     ✅
  2. Build graph (L2/L3) → Neo4j        ✅
  3. Validate                            ✅
  MISSING: Clone + Tree-sitter parsing!  ❌
```

---

## Implementation Plan

### Phase 1: Add Layer 1 to `init` Command (30 min)

**File:** `cmd/crisk/init.go`

**Changes Needed:**

1. **Add Stage 0: Clone + Parse (before GitHub fetch)**
   ```go
   // Stage 0: Clone repository and parse with tree-sitter (Layer 1)
   fmt.Printf("\n[0/4] Cloning and parsing repository...\n")

   // Use existing code from init_local.go
   repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
   repoPath, err := ingestion.CloneRepository(ctx, repoURL)

   // Parse with tree-sitter
   processor := ingestion.NewProcessor(ingestion.DefaultProcessorConfig(), graphBackend, nil)
   parseResult, err := processor.ProcessRepository(ctx, repoURL)
   ```

2. **Update stage numbers:** 0/4, 1/4, 2/4, 3/4

3. **Import required packages:**
   ```go
   "github.com/coderisk/coderisk-go/internal/ingestion"
   ```

4. **Pass graphBuilder to processor** (for Layer 2 temporal analysis)

### Phase 2: Add GitHub Token to Configure (15 min)

**File:** `cmd/crisk/configure.go`

**Add after Step 2 (LLM Model):**

```go
// Step 3: GitHub Token
fmt.Println("Step 3/5: GitHub Token (Required for Production)")
fmt.Println("Enables temporal analysis (commits, co-changes, ownership)")
fmt.Println("Create token: https://github.com/settings/tokens")
fmt.Println("Scopes: repo (private) or public_repo (public)")

// Similar flow to OpenAI key with keychain option
```

**Update keyring manager** to support GitHub token.

### Phase 3: Update Validation (10 min)

**File:** `cmd/crisk/init.go:validateGraph()`

**Add checks for all 3 layers:**

```go
requiredLabels := []string{
    "File",      // Layer 1
    "Function",  // Layer 1
    "Commit",    // Layer 2
    "Developer", // Layer 2
    "Issue",     // Layer 3
}
```

### Phase 4: Update Documentation (10 min)

**Files to update:**
- `README.md` - Installation section
- `.env.example` - Clarify GITHUB_TOKEN required
- `cmd/crisk/init.go` docstring

---

## Testing Requirements

See [INIT_TESTING_STRATEGY.md](./INIT_TESTING_STRATEGY.md) for comprehensive test plan.

**Smoke Test:**
```bash
# Clean start
docker compose down -v
docker compose up -d

# Configure
crisk configure
# Enter: OpenAI key, GitHub token

# Full init (all 3 layers)
crisk init omnara-ai/omnara

# Verify in Neo4j Browser (http://localhost:7475)
# Should see: Files, Functions, Commits, Developers, Issues
```

---

## Success Criteria

- [ ] `crisk init` clones repo and parses with tree-sitter
- [ ] All 3 layers visible in Neo4j
- [ ] Validation passes for all node types
- [ ] `crisk check` works with full graph
- [ ] Documentation updated
- [ ] Tests pass (see testing strategy)

---

## Migration Path

**For users with existing data:**

1. If only ran `init-local`:
   - Run `crisk init` - will add Layers 2 & 3

2. If only ran `init` (current broken version):
   - Re-run `crisk init` - will add Layer 1
   - Or: `docker compose down -v && crisk init` (clean slate)

**Deprecation Plan:**
- Keep `init-local` for Week 1 demos only
- Document it as "limited functionality"
- Main production command: `crisk init`

---

## Files to Modify

| File | Changes | Estimated Time |
|------|---------|----------------|
| `cmd/crisk/init.go` | Add Layer 1 integration | 30 min |
| `cmd/crisk/configure.go` | Add GitHub token wizard | 15 min |
| `internal/config/keyring.go` | Support GitHub token | 10 min |
| `internal/config/config.go` | Add GitHub config fields | 5 min |
| `README.md` | Update install instructions | 10 min |
| `.env.example` | Clarify requirements | 5 min |

**Total:** ~75 minutes

---

## Next Steps

1. Read [INIT_TESTING_STRATEGY.md](./INIT_TESTING_STRATEGY.md)
2. Execute Phase 1 (Layer 1 integration)
3. Execute Phase 2 (GitHub token config)
4. Run tests from testing strategy
5. Deploy beta.5 with full 3-layer support
