# Revised MVP Strategy - Simplified Approach

**Date:** 2025-10-27
**Status:** RECOMMENDED PATH FORWARD
**Priority:** SHIP IN 1 WEEK

---

## Executive Summary

After analyzing your docs and current state, here's my **strong recommendation**:

### ✅ MUST DO BEFORE SHIP (4-6 hours):
1. **Implement graph query execution** in collector.go
2. **Use commit message regex** for incident history (NOT Issue nodes yet)
3. **Skip file resolution** for MVP (good enough without it)

### 🔄 DEFER TO WEEK 2-3:
1. **Issue ingestion with Gemini Flash** (simplified, LLM-only approach)
2. **File resolution** (only if users complain about missing data)

### ❌ DON'T DO:
1. Complex regex patterns for issue linking
2. Hybrid regex + LLM approach (over-engineered for MVP)
3. ALIAS_OF relationships (too complex for MVP)

---

## Part 1: File Resolution - Do We Need It?

### Current Problem

From [file_resolution_implementation_plan.md](coderisk/dev_docs/mvp/file_resolution/file_resolution_implementation_plan.md):
- 91% of commits lack MODIFIED edges due to file reorganization
- Historical paths (`shared/config/settings.py`) don't match current paths (`src/shared/config/settings.py`)
- Need multi-level resolution to bridge gap

### BUT... Do You Actually Have This Problem?

**Let me check your current MODIFIED edge coverage:**

From [option_b_results.md](coderisk/dev_docs/testing/option_b_results.md):
- **MODIFIED edges: 1,585 edges (99.0% coverage)** ✅
- **190/192 commits have MODIFIED edges** ✅

**From [validation_report_phase1_complete.md](coderisk/dev_docs/testing/validation_report_phase1_complete.md):**
- Before fix: 8.8% coverage (broken)
- After fix: **99.0% coverage** ✅
- Both `current` and `historical` File nodes working

### 🎯 VERDICT: **YOU DON'T NEED FILE RESOLUTION FOR MVP**

**Why:**
1. ✅ You already have 99% MODIFIED edge coverage
2. ✅ File nodes are properly marked (`current: true`, `historical: true`)
3. ✅ Queries can use multi-path approach: `WHERE f.path IN [current, historical]`
4. ✅ `git log --follow` already implemented and tested ([history.go](coderisk/internal/git/history.go))

**What you have is BETTER than the file resolution plan describes:**
- File resolution plan expected 85% coverage
- You already achieved 99% coverage
- The gap was already fixed in Phase 1!

### Minimal File Path Handling (Already Done)

You already implemented this in [internal/git/history.go](coderisk/internal/git/history.go):

```go
func (ht *HistoryTracker) GetFileHistory(ctx context.Context, filePath string) ([]string, error) {
    // Returns all historical paths for a file
}
```

**How to use it in crisk check:**

```go
// In collector.go, when querying for a file:
func (c *Collector) CollectPhase1Data(ctx context.Context, filePath string) (*types.Phase1Data, error) {
    // Get historical paths
    paths, err := c.historyTracker.GetFileHistory(ctx, filePath)
    if err != nil {
        paths = []string{filePath}  // Fallback to current path only
    }

    // Query with ALL paths
    ownershipResult, err := c.graphBackend.QueryWithParams(ctx, OwnershipQuery, map[string]any{
        "file_paths": paths,  // Pass array of paths
    })
}
```

**Update your queries to accept arrays:**

```cypher
// OLD (single path):
MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $file_path})

// NEW (multi-path):
MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File)
WHERE f.path IN $file_paths
```

**Implementation time:** 30 minutes (just update queries)

### ❌ Skip the Complex File Resolution System

The [file_resolution_implementation_plan.md](coderisk/dev_docs/mvp/file_resolution/file_resolution_implementation_plan.md) proposes:
- ALIAS_OF relationships
- File alias graph builder
- Enhanced MODIFIED edge creation
- TreeSitter import extractor
- **Total: 17-25 hours**

**You don't need this because:**
1. Your current approach (99% coverage) is already excellent
2. Simple multi-path query is sufficient
3. Can add ALIAS_OF edges later if needed

---

## Part 2: Issue Ingestion - Simplified LLM-Only Approach

### Your Original Plan (Complex)

From [issue_ingestion_implementation_plan.md](coderisk/dev_docs/mvp/issue_fixed_by/issue_ingestion_implementation_plan.md):

**5-Phase, 20-29 hour plan:**
1. Phase 1: Validate existing graph (4-6 hours)
2. Phase 2: Issue data ingestion (2-3 hours)
3. Phase 3: Reference extraction (6-8 hours) ← **COMPLEX**
4. Phase 4: Create nodes & edges (4-6 hours)
5. Phase 5: Testing (4-6 hours)

**Phase 3 is complex:**
- Layer 1: Regex pre-filter
- Layer 2: Gemini Flash fallback
- Layer 3: Temporal matching
- Layer 4: False positive filtering

### 🎯 RECOMMENDED: **Simplified Gemini Flash-Only Approach**

**Why simpler is better for MVP:**
1. ✅ Gemini Flash is **$0.27 total** for React-sized repo (cheap!)
2. ✅ Gemini Flash has 95% accuracy vs 90% for regex (better!)
3. ✅ No need to maintain complex regex patterns
4. ✅ Single code path = easier to debug
5. ✅ Batch processing (100 messages/call) = fast enough

**From [extraction_strategy_regex_vs_llm.md](coderisk/dev_docs/mvp/issue_fixed_by/extraction_strategy_regex_vs_llm.md):**

| Approach | Time | Cost | Coverage | Accuracy |
|----------|------|------|----------|----------|
| Regex only | 0.18s | $0 | 85% | 90% |
| **LLM only** | **6 min** | **$0.27** | **95%** | **95%** |
| Hybrid | 5s | $0.27 | 95% | 95% |

**Hybrid saves 5.5 minutes but adds code complexity. Not worth it for MVP.**

### Simplified 3-Phase Plan (8-12 hours total)

#### Phase 1: Issue Data Preparation (2 hours)

**You already have:**
- ✅ `github_issues` table in PostgreSQL
- ✅ `FetchIssues()` working
- ✅ Issue nodes created in Neo4j

**Just verify:**
```sql
SELECT COUNT(*) FROM github_issues WHERE repo_id = 'omnara';
-- Should return 80 issues
```

```cypher
MATCH (i:Issue) RETURN count(i);
-- Should return 80 issues
```

---

#### Phase 2: Gemini Flash Extraction (4-6 hours)

**Create simple extraction pipeline:**

**File:** `internal/github/gemini_extractor.go` (NEW)

```go
package github

import (
    "context"
    "fmt"
    "github.com/google/generative-ai-go/genai"
)

type GeminiExtractor struct {
    client *genai.Client
}

type IssueReference struct {
    IssueNumber  int     `json:"issue_number"`
    CommitSHA    string  `json:"commit_sha,omitempty"`
    Action       string  `json:"action"`  // "fixes", "closes", "resolves", "mentions"
    Confidence   float64 `json:"confidence"`
}

func NewGeminiExtractor(apiKey string) (*GeminiExtractor, error) {
    client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
    if err != nil {
        return nil, err
    }
    return &GeminiExtractor{client: client}, nil
}

func (e *GeminiExtractor) ExtractReferences(ctx context.Context, messages []Message) ([]IssueReference, error) {
    batchSize := 100
    var allRefs []IssueReference

    for i := 0; i < len(messages); i += batchSize {
        end := min(i+batchSize, len(messages))
        batch := messages[i:end]

        refs, err := e.extractBatch(ctx, batch)
        if err != nil {
            return nil, fmt.Errorf("batch %d failed: %w", i/batchSize, err)
        }

        allRefs = append(allRefs, refs...)

        // Rate limit: 1 req/sec (Gemini Flash free tier)
        time.Sleep(1 * time.Second)
    }

    return allRefs, nil
}

func (e *GeminiExtractor) extractBatch(ctx context.Context, messages []Message) ([]IssueReference, error) {
    // Build prompt
    prompt := buildExtractionPrompt(messages)

    // Define schema
    schema := &genai.Schema{
        Type: "object",
        Properties: map[string]*genai.Schema{
            "references": {
                Type: "array",
                Items: &genai.Schema{
                    Type: "object",
                    Properties: map[string]*genai.Schema{
                        "message_index": {Type: "integer"},
                        "issue_number": {Type: "integer"},
                        "action": {
                            Type: "string",
                            Enum: []string{"fixes", "closes", "resolves", "mentions"},
                        },
                        "confidence": {Type: "number"},
                    },
                    Required: []string{"message_index", "issue_number", "action", "confidence"},
                },
            },
        },
    }

    model := e.client.GenerativeModel("gemini-2.0-flash-exp")
    model.ResponseSchema = schema

    resp, err := model.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        return nil, err
    }

    return parseResponse(resp, messages)
}

func buildExtractionPrompt(messages []Message) string {
    prompt := `Extract issue and PR references from GitHub commit messages and PR bodies.

For each message, identify:
1. Issue/PR numbers being referenced
2. The action: "fixes" (closes the issue), "closes" (same as fixes), "resolves" (same as fixes), or "mentions" (just a reference, doesn't fix)
3. Confidence (0.0-1.0) based on clarity of the reference

Rules:
- "Fixes #123", "Closes #456", "Resolves #789" = action: "fixes", confidence: 0.95
- "Fix #123" or "Close #456" = action: "fixes", confidence: 0.90
- "Issue #123" or "See #456" = action: "mentions", confidence: 0.85
- "Don't fix #123" = IGNORE (negation)
- "Similar to #123" = action: "mentions", confidence: 0.70
- No reference = return empty for that message

Messages:

`
    for i, msg := range messages {
        prompt += fmt.Sprintf("%d. [%s] %s: %s\n\n", i+1, msg.Type, msg.ID, msg.Text)
    }

    return prompt
}
```

**Prompt Engineering Best Practices:**

1. **Clear instructions** (what to extract)
2. **Examples** (show expected behavior)
3. **Edge cases** (negations, partial refs)
4. **Structured output** (JSON schema)
5. **Confidence scoring** (explicit rules)

**Expected output for 100 messages:**
```json
{
  "references": [
    {"message_index": 1, "issue_number": 123, "action": "fixes", "confidence": 0.95},
    {"message_index": 3, "issue_number": 456, "action": "closes", "confidence": 0.90},
    {"message_index": 5, "issue_number": 789, "action": "mentions", "confidence": 0.70}
  ]
}
```

---

#### Phase 3: Create FIXED_BY Edges (2-4 hours)

**File:** `internal/graph/issue_linker.go` (NEW)

```go
package graph

type IssueLinker struct {
    backend *Neo4jBackend
}

func (il *IssueLinker) CreateFixedByEdges(ctx context.Context, refs []IssueReference) error {
    // Filter to only "fixes"/"closes"/"resolves" (exclude "mentions")
    var fixRefs []IssueReference
    for _, ref := range refs {
        if ref.Action == "fixes" || ref.Action == "closes" || ref.Action == "resolves" {
            if ref.Confidence >= 0.40 {  // Minimum confidence threshold
                fixRefs = append(fixRefs, ref)
            }
        }
    }

    // Batch create edges (100 at a time)
    batchSize := 100
    for i := 0; i < len(fixRefs); i += batchSize {
        end := min(i+batchSize, len(fixRefs))
        batch := fixRefs[i:end]

        if err := il.createBatch(ctx, batch); err != nil {
            return err
        }
    }

    return nil
}

func (il *IssueLinker) createBatch(ctx context.Context, refs []IssueReference) error {
    query := `
        UNWIND $refs AS ref
        MATCH (i:Issue {number: ref.issue_number})
        MATCH (c:Commit {sha: ref.commit_sha})
        MERGE (i)-[r:FIXED_BY]->(c)
        SET r.confidence = ref.confidence,
            r.detection_method = "llm",
            r.detected_at = datetime()
    `

    params := map[string]interface{}{
        "refs": refs,
    }

    _, err := il.backend.ExecuteWrite(ctx, query, params)
    return err
}
```

**Validation query:**
```cypher
MATCH (i:Issue)-[r:FIXED_BY]->(c:Commit)
RETURN count(r) as edge_count,
       avg(r.confidence) as avg_confidence,
       count(CASE WHEN r.confidence >= 0.75 THEN 1 END) as high_confidence_count
```

---

### Cost & Performance Analysis

**For Omnara repo (80 issues + 192 commits + 149 PRs):**

**Input messages:**
- 192 commits × 50 tokens avg = 9,600 tokens
- 149 PRs × 200 tokens avg = 29,800 tokens
- 80 issues × 100 tokens avg = 8,000 tokens
- **Total: 47,400 tokens**

**Cost:**
- Input: 47,400 tokens × $0.10/1M = **$0.0047**
- Output: ~2,000 tokens × $0.40/1M = **$0.0008**
- **Total: $0.0055 ≈ $0.01** (less than a penny!)

**Time:**
- Messages: 421 total
- Batches: 5 batches (100 messages each)
- Rate limit: 1 req/sec
- **Total: 5 seconds**

**For React repo (35,418 messages):**
- Cost: $0.27
- Time: 355 seconds ≈ **6 minutes**

### Why This is Better Than Hybrid

**Hybrid approach:**
- Regex: 90% coverage, 0.18s, $0
- LLM: 10% coverage, 6 min, $0.27
- **Total: 5s, $0.27, 95% coverage**
- **Code complexity: HIGH (two systems to maintain)**

**LLM-only approach:**
- LLM: 95% coverage, 6 min, $0.27
- **Code complexity: LOW (one system)**
- **5.5 minutes slower, but who cares?** (one-time operation)

**Trade-off:**
- Lose: 5.5 minutes of processing time (one-time)
- Gain: Simpler code, easier to debug, 5% better accuracy

**MVP wisdom:** Simplicity > micro-optimizations

---

## Part 3: Updated Issue Ingestion Plan

### File to Update

**Modify:** [issue_ingestion_implementation_plan.md](coderisk/dev_docs/mvp/issue_fixed_by/issue_ingestion_implementation_plan.md)

Replace Phase 3 (Reference Extraction) with:

```markdown
## Phase 3: Gemini Flash-Only Extraction (4-6 hours)

**Goal:** Extract issue/PR/commit references using Gemini Flash with structured output.

**Why LLM-only (not hybrid):**
- Cost: $0.27 for React-sized repo (acceptable)
- Time: 6 minutes (one-time, acceptable)
- Accuracy: 95% (better than regex 90%)
- Simplicity: Single code path, easier to maintain
- MVP-appropriate: Ship fast, optimize later if needed

### 3.1 Design Extraction Pipeline

**Input sources:**
1. All commit messages (from github_commits table)
2. All PR bodies and titles (from github_pull_requests table)
3. All issue bodies (from github_issues table)

**Output format:**
```go
type IssueReference struct {
    MessageIndex int
    IssueNumber  int
    Action       string  // "fixes", "closes", "resolves", "mentions"
    Confidence   float64 // 0.40 - 0.95
}
```

### 3.2 Gemini Flash Prompt Engineering

**Prompt structure:**
```
Extract issue and PR references from GitHub messages.

For each message, identify:
1. Issue/PR numbers being referenced (#123, #456, etc.)
2. The action:
   - "fixes" = closes the issue (keywords: fix, fixes, close, closes, resolve, resolves)
   - "mentions" = just a reference (keywords: see, related to, similar to)
3. Confidence level (0.0-1.0)

Rules:
- "Fixes #123" = action: "fixes", confidence: 0.95
- "Fix #123" = action: "fixes", confidence: 0.90
- "See #123" = action: "mentions", confidence: 0.85
- "Don't fix #123" = IGNORE (negation)
- "Similar to #123" = action: "mentions", confidence: 0.70

Messages:
1. [commit abc123] Fix null pointer in payment processor (#123)
2. [pr 456] Refactor authentication system
3. [commit def456] Update README
...
```

**Structured output schema:**
```json
{
  "type": "object",
  "properties": {
    "references": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "message_index": {"type": "integer"},
          "issue_number": {"type": "integer"},
          "action": {
            "type": "string",
            "enum": ["fixes", "closes", "resolves", "mentions"]
          },
          "confidence": {"type": "number"}
        }
      }
    }
  }
}
```

### 3.3 Batch Processing

**Batch size:** 100 messages per API call

**Processing pipeline:**
1. Fetch all messages from PostgreSQL
2. Group into batches of 100
3. Call Gemini Flash API (1 req/sec rate limit)
4. Parse structured JSON responses
5. Filter by confidence >= 0.40
6. Filter by action = "fixes" or "closes" or "resolves" (exclude "mentions")
7. Store references in memory

**Performance:**
- Omnara (421 messages): 5 seconds, $0.01
- React (35,418 messages): 6 minutes, $0.27

### 3.4 Error Handling

**Graceful degradation:**
- If API call fails: Retry once with exponential backoff
- If batch fails after retry: Log error, continue with next batch
- If all batches fail: Return partial results (what succeeded)
- Minimum viable: Even 50% of references is useful

**Logging:**
- Log total messages processed
- Log total references extracted
- Log average confidence
- Log API errors

### 3.5 Testing

**Test cases:**
1. Simple: "Fixes #123" → {issue: 123, action: "fixes", confidence: 0.95}
2. Multiple: "Fixes #123, #456" → 2 references
3. Negation: "Don't fix #123" → no reference
4. Mention: "Similar to #123" → {issue: 123, action: "mentions", confidence: 0.70} → filtered out
5. Cross-repo: "Fixes facebook/react#123" → {issue: 123, confidence: 0.90}

**Validation:**
- Sample 100 random results
- Manually verify correctness
- Target: 85%+ accuracy for confidence >= 0.75
```

---

## Part 4: Final Recommendation

### Week 1: Ship MVP (6-8 hours total)

**Day 1 (4 hours):**
1. ✅ Implement graph query execution in collector.go
2. ✅ Update queries to accept file path arrays (`WHERE f.path IN $file_paths`)
3. ✅ Use commit message regex for incident history
4. ✅ Test end-to-end with omnara

**Day 2 (2 hours):**
1. ✅ Write docs (README, installation)
2. ✅ Record demo video
3. ✅ Prepare HackerNews post

**Ship it!** 🚀

### Week 2-3: Add Issue Linking (8-12 hours)

**Implementation:**
1. Verify Issue nodes in Neo4j (already working)
2. Create `gemini_extractor.go` with LLM-only approach
3. Create `issue_linker.go` to create FIXED_BY edges
4. Run extraction on omnara (5 seconds, $0.01)
5. Validate results
6. Update incident history queries to use Issue nodes

**Why defer:**
- MVP works without it (commit message regex is good enough)
- Gives you time to get user feedback first
- Simpler launch (less complexity to debug)

### Post-MVP: File Resolution (ONLY IF NEEDED)

**When to implement:**
- If users complain about missing historical commits
- If 99% MODIFIED coverage drops (repo-specific issue)
- If you need ALIAS_OF edges for advanced queries

**For now:** Your multi-path query approach is sufficient.

---

## Part 5: Updated Implementation Checklist

### ✅ Week 1: MVP Launch

**File: `internal/risk/collector.go`**
- [ ] Implement `CollectPhase1Data()` to execute all 5 queries
- [ ] Add `GetFileHistory()` call for multi-path resolution
- [ ] Update queries to accept `file_paths` array parameter
- [ ] Add result parsing functions
- [ ] Test with omnara repo

**File: `internal/risk/queries.go`**
- [ ] Update all queries: `{path: $file_path}` → `WHERE f.path IN $file_paths`
- [ ] Update incident query to use commit message regex
- [ ] Test queries in Neo4j browser

**File: `cmd/crisk/check.go`**
- [ ] Verify file path resolution works
- [ ] Test output formatting
- [ ] Verify Phase 2 LLM escalation

**Testing:**
- [ ] Run `crisk check` on 5 different files
- [ ] Verify ownership shows correct developers
- [ ] Verify blast radius shows dependencies
- [ ] Verify co-change shows frequent partners
- [ ] Verify incidents show recent bugs

### 🔄 Week 2-3: Issue Linking

**New Files:**
- [ ] `internal/github/gemini_extractor.go`
- [ ] `internal/github/gemini_extractor_test.go`
- [ ] `internal/graph/issue_linker.go`
- [ ] `internal/graph/issue_linker_test.go`

**Implementation:**
- [ ] Set up Gemini API client
- [ ] Write extraction prompt
- [ ] Implement batch processing
- [ ] Create FIXED_BY edges
- [ ] Validate accuracy (sample 100)

**Update Queries:**
- [ ] Replace commit message regex with Issue node queries
- [ ] Add confidence filtering (>= 0.75 for display)

---

## Conclusion

### Your MVP Strategy Should Be:

**✅ DO NOW (Week 1):**
1. Implement graph queries (4 hours)
2. Use multi-path file resolution (already have git log --follow)
3. Use commit message regex for incidents (good enough)
4. Ship and get user feedback

**🔄 DO NEXT (Week 2-3):**
1. LLM-only issue extraction (simpler than hybrid)
2. Create FIXED_BY edges
3. Improve incident history with Issue nodes

**❌ DON'T DO:**
1. Complex file resolution system (already at 99% coverage!)
2. Hybrid regex + LLM (over-engineered for MVP)
3. ALIAS_OF relationships (can add later)

### Why This Strategy Wins:

1. **Faster to ship** (6-8 hours vs 20-29 hours)
2. **Simpler code** (less to debug)
3. **Still excellent** (99% coverage, good incident detection)
4. **Iterate based on user feedback** (don't build what users don't need)
5. **Cost is negligible** ($0.27 for large repos, $0.01 for omnara)

**Ship the simple version. Iterate based on real feedback. Don't optimize prematurely.**

---

**Last Updated:** 2025-10-27
**Recommendation:** Implement LLM-only extraction, skip file resolution complexity
**Priority:** Ship MVP in Week 1, add issues in Week 2-3
