# Pattern 3: Comment-Based Linking Implementation Plan

**Date:** November 2, 2025
**Status:** 🚧 Planning
**Expected Impact:** +15% recall (from 91.67% to ~100%)

---

## 📋 Overview

Implement Pattern 3 from LINKING_PATTERNS.md to extract issue-PR links from issue/PR comments. This addresses cases where:
- Maintainers comment "Fixed in PR #123" on issues
- Developers comment "This resolves issue #456" in PRs
- Bots post "Deployed in commit abc123" comments

**Coverage:** 15% of real-world cases (LINKING_PATTERNS.md)

---

## 🏗️ Architecture

### Current State
```
GitHub API → Fetcher → github_issue_comments (PostgreSQL)
                    ↓
                (NOT USED YET)
```

### Target State
```
GitHub API → Fetcher → github_issue_comments (PostgreSQL)
                                ↓
                    CommentExtractor (NEW)
                                ↓
                      LLM Comment Analyzer
                                ↓
                github_issue_commit_refs (PostgreSQL)
                                ↓
                         IssueLinker
                                ↓
                     Neo4j (FIXED_BY edges)
```

---

## 📁 Files to Create/Modify

### 1. **NEW:** `internal/github/comment_extractor.go`
**Purpose:** Extract references from issue comments
**Similar to:** `issue_extractor.go`, `commit_extractor.go`

**Key functions:**
```go
type CommentExtractor struct {
    stagingDB *database.StagingClient
    llmClient *llm.Client
    analyzer  *llm.CommentAnalyzer
}

func NewCommentExtractor(stagingDB, llmClient) *CommentExtractor
func (e *CommentExtractor) ProcessIssueComments(ctx, repoID) error
func (e *CommentExtractor) extractCommentReferences(issue, comments) []IssueCommitRef
```

### 2. **MODIFY:** `internal/graph/builder.go`
**Add after Line 115 (temporal correlation):**
```go
// Extract Comment-Based References (Pattern 3: 15% coverage)
// Reference: LINKING_PATTERNS.md - Pattern 3: Comment-based linking
commentStats, err := b.extractCommentReferences(ctx, repoID)
if err != nil {
    return stats, fmt.Errorf("comment extraction failed: %w", err)
}
log.Printf("  ✓ Comment extraction: found %d references", commentStats.Edges)
```

### 3. **USE EXISTING:** `internal/llm/comment_analyzer.go`
**Status:** ✅ Already implemented
**Functions:**
- `ExtractCommentReferences()` - LLM extraction
- `applyCommenterBoost()` - Role-based confidence boost

### 4. **USE EXISTING:** `internal/database/staging.go`
**Tables:**
- `github_issue_comments` (read from) ✅ Already created
- `github_issue_commit_refs` (write to) ✅ Already exists

---

## 🔄 Integration Points

### Phase 1: Fetching (✅ DONE)
**File:** `internal/github/fetcher.go:871`
```go
func (f *Fetcher) FetchIssueComments(ctx, repoID, owner, repo, days) (int, error)
```
**Status:** ✅ Already fetches and stores comments in `github_issue_comments`

### Phase 2: Extraction (🚧 NEW)
**File:** `internal/github/comment_extractor.go` (to create)
```go
func (e *CommentExtractor) ProcessIssueComments(ctx context.Context, repoID int64) error {
    // 1. Fetch issues with comments from database
    issues := e.stagingDB.FetchIssuesWithComments(ctx, repoID)

    // 2. For each issue, get its comments
    for _, issue := range issues {
        comments := e.stagingDB.GetIssueComments(ctx, issue.ID)

        // 3. Call LLM to extract references
        refs, err := e.analyzer.ExtractCommentReferences(
            ctx,
            issue.Number,
            issue.Title,
            issue.Body,
            issue.ClosedAt,
            comments,
            repoOwner,
            collaborators,
        )

        // 4. Convert LLM refs to IssueCommitRefs
        issueCommitRefs := e.convertToIssueCommitRefs(issue, refs)

        // 5. Store in github_issue_commit_refs
        e.stagingDB.StoreIssueCommitRefs(ctx, issueCommitRefs)
    }
}
```

### Phase 3: Linking (✅ EXISTING)
**File:** `internal/graph/issue_linker.go:100`
```go
func (l *IssueLinker) createIncidentEdges(ctx, repoID) (*BuildStats, error) {
    // Get all references (now includes comment-based)
    refs := l.stagingDB.GetIssueCommitRefs(ctx, repoID)

    // Create FIXED_BY / ASSOCIATED_WITH edges
    // ... existing logic handles comment refs automatically
}
```
**Status:** ✅ No changes needed - already processes all refs

---

## 📊 Confidence Scoring

**Base confidence (from LLM):** 0.60-0.70
**Commenter role boost:**
- Owner/Maintainer: +0.10 → 0.85-0.90
- Collaborator: +0.08 → 0.80-0.85
- Bot: +0.05 → 0.75-0.80
- Contributor: +0.03 → 0.70-0.75

**Combined with other patterns:**
- Comment (0.85) + Temporal (0.15) = 0.98 (capped)
- Comment (0.75) + Semantic (0.10) = 0.85

---

## 🧪 Testing Strategy

### 1. Unit Tests
**File:** `internal/github/comment_extractor_test.go`
```go
func TestExtractCommentReferences(t *testing.T)
func TestCommenterRoleBoost(t *testing.T)
func TestCommentPatterns(t *testing.T)
```

### 2. Integration Tests
**Test with real Omnara data:**
- Issue #1060 (Stagehand): "Fixed in PR #1065" (comment only)
- Issue #187 (Omnara): Comment + temporal + semantic

### 3. Backtesting
**Ground truth update:**
- Add 2-3 comment-based test cases
- Re-run backtest to verify +15% recall improvement

---

## 📅 Implementation Steps

### Step 1: Create Comment Extractor (2-3 hours)
1. Create `internal/github/comment_extractor.go`
2. Implement `ProcessIssueComments()`
3. Add conversion logic to `IssueCommitRef` format

### Step 2: Database Queries (1 hour)
1. Add `FetchIssuesWithComments()` to `staging.go`
2. Add `GetIssueComments()` to `staging.go`
3. Query for collaborators/repo owner

### Step 3: Integration (1 hour)
1. Modify `internal/graph/builder.go`
2. Add comment extraction step after temporal correlation
3. Wire up CommentExtractor with LLM client

### Step 4: Testing (2 hours)
1. Write unit tests for comment extraction
2. Test with Omnara repo data
3. Run backtesting to verify metrics

### Step 5: Documentation (30 min)
1. Update LINKING_PATTERNS.md status to ✅
2. Document comment patterns found
3. Create examples for future reference

**Total Estimated Time:** 6-7 hours

---

## 🎯 Success Criteria

### Functional Requirements
- ✅ Extract "Fixed in PR #N" from issue comments
- ✅ Extract "Resolves issue #N" from PR comments
- ✅ Apply role-based confidence boosting
- ✅ Store refs in `github_issue_commit_refs`
- ✅ Create FIXED_BY edges in Neo4j

### Performance Requirements
- **Recall:** +15% (from 91.67% to ~100%)
- **Precision:** Maintain 100% (no false positives from comments)
- **F1 Score:** Improve from 95.65% to ~100%

### Test Requirements
- ✅ All unit tests pass
- ✅ Backtest with 15 test cases: P=100%, R≥95%
- ✅ No regressions in existing patterns

---

## 🚧 Known Challenges

### Challenge 1: LLM Cost
**Issue:** Processing comments for every issue = many LLM calls
**Solution:** Only process issues with >0 comments, cache results

### Challenge 2: Comment Noise
**Issue:** Many comments are unrelated discussions
**Solution:** LLM filters for linking keywords ("fixed", "resolved", "PR #")

### Challenge 3: Conflicting Comments
**Issue:** Multiple comments may reference different PRs
**Solution:** Use latest comment from highest authority (owner > collaborator)

---

## 🔄 Rollback Plan

If implementation causes issues:
1. **Quick fix:** Comment out comment extraction in `builder.go`
2. **Data fix:** Delete comment-based refs from `github_issue_commit_refs`
3. **Revert:** Git revert the commit

---

## 📈 Expected Results

### Before (Current)
- **Precision:** 100%
- **Recall:** 91.67% (11/12 detected)
- **F1 Score:** 95.65%
- **Patterns:** Explicit (60%) + Temporal (20%) = 80% coverage

### After (With Comment-Based)
- **Precision:** 100% (target)
- **Recall:** ~100% (12/12 detected, +1 from comments)
- **F1 Score:** ~100%
- **Patterns:** Explicit (60%) + Temporal (20%) + Comment (15%) = 95% coverage

---

## ✅ Next Actions

1. Review this plan with user
2. Get approval to proceed
3. Start implementation with Step 1 (Comment Extractor)
4. Test incrementally after each step
5. Run final backtesting to validate results

---

**Note:** This plan is based on existing architecture and patterns from:
- `internal/github/issue_extractor.go` (PR body extraction)
- `internal/github/commit_extractor.go` (commit message extraction)
- `internal/llm/comment_analyzer.go` (LLM comment analysis)
- `LINKING_PATTERNS.md` (Pattern 3 specification)
