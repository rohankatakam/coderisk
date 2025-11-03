# Issue-PR-Commit Linking Orchestration Strategy

## Executive Summary

This document outlines the complete orchestration strategy for implementing all 6 linking patterns from [LINKING_PATTERNS.md](test_data/LINKING_PATTERNS.md) into the existing CodeRisk infrastructure. The goal is to achieve **F1 Score ≥ 75%** through a multi-pattern matching approach that seamlessly integrates with the current architecture.

**Current State:**
- ✅ Pattern 1 (Explicit): 60% coverage - IMPLEMENTED
- ❌ Pattern 2 (Temporal): 20% coverage - NOT INTEGRATED
- ❌ Pattern 3 (Comment): 15% coverage - NOT INTEGRATED
- ⚠️ Pattern 4 (Semantic): 10% coverage - PARTIALLY IMPLEMENTED
- ⚠️ Pattern 5 (Cross-Ref): 8% coverage - PARTIALLY IMPLEMENTED
- ⚠️ Pattern 6 (Merge Commit): 5% coverage - PARTIALLY IMPLEMENTED

**Target State:** All 6 patterns integrated → **Expected F1: 75-80%**

---

## Architecture Overview

### Current Flow (As-Is)

```
┌─────────────────────────────────────────────────────────────┐
│                     cmd/crisk/init.go                       │
│                     Entry Point                             │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 1: GitHub API → PostgreSQL (github.Fetcher)          │
│  • Fetches issues, PRs, commits, comments                   │
│  • Stores in staging tables (github_issues, etc.)           │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 1.5: LLM Reference Extraction                        │
│  • IssueExtractor: Extract from issue body/timeline ✅      │
│  • CommitExtractor: Extract from commit messages ✅         │
│  • Stores refs in github_issue_commit_refs table            │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 2: Build Graph (graph.Builder.BuildGraph)            │
│  • processCommits() → Commit, Developer nodes               │
│  • processPRs() → PullRequest nodes                         │
│  • processIssues() → Issue nodes                            │
│  • linkIssues() → FIXED_BY/ASSOCIATED_WITH edges ⚠️         │
└─────────────────────────────────────────────────────────────┘
```

**Problem:** linkIssues() only uses explicit references from LLM extraction. **Missing:** temporal, comment, semantic, bidirectional, and merge commit patterns.

---

### Target Flow (To-Be)

```
┌─────────────────────────────────────────────────────────────┐
│                     cmd/crisk/init.go                       │
│                     Entry Point                             │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 1: GitHub API → PostgreSQL (github.Fetcher)          │
│  • Fetches issues, PRs, commits, comments                   │
│  • Stores in staging tables                                 │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 1.5: Enhanced LLM Reference Extraction 🆕            │
│  • IssueExtractor: Extract from issue body/timeline         │
│  • CommentAnalyzer: Extract from issue comments 🆕          │
│  • CommitExtractor: Extract from commit messages            │
│  • MergeCommitExtractor: Detect merge commits 🆕            │
│  • Stores refs with confidence + evidence arrays            │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│  Stage 2: Build Graph (graph.Builder.BuildGraph)            │
│  • processCommits() → Commit, Developer nodes               │
│  • processPRs() → PullRequest nodes                         │
│  • processIssues() → Issue nodes                            │
│  • linkIssuesEnhanced() 🆕 → Multi-pattern linking          │
│    ├─ Step 1: Fetch base references from staging           │
│    ├─ Step 2: Apply Temporal Correlator 🆕                  │
│    ├─ Step 3: Apply Semantic Matcher 🆕                     │
│    ├─ Step 4: Apply Bidirectional Validator 🆕              │
│    ├─ Step 5: Combine Evidence & Cap Confidence 🆕          │
│    └─ Step 6: Create FIXED_BY/ASSOCIATED_WITH edges         │
└─────────────────────────────────────────────────────────────┘
```

---

## Integration Strategy

### Phase 1: Extract Comments (Pattern 3) - HIGH PRIORITY

**File:** `internal/github/issue_extractor.go`

**Changes:**
1. Add `fetchIssueComments()` method to fetch comments from GitHub API
2. Add `fetchCollaborators()` to get repo collaborators for role determination
3. Integrate `llm.CommentAnalyzer` into `ExtractReferences()`

**Implementation:**

```go
// internal/github/issue_extractor.go

func (e *IssueExtractor) ExtractReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting references from issues...")

	batchSize := 20
	totalRefs := 0

	// Fetch repository info for comment analysis
	repo, err := e.stagingDB.GetRepository(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository: %w", err)
	}

	// Fetch collaborators for role determination
	collaborators, err := e.fetchCollaborators(ctx, repo.Owner, repo.Name)
	if err != nil {
		log.Printf("  ⚠️  Failed to fetch collaborators: %v", err)
		collaborators = []string{} // Continue without collaborators
	}

	// Fetch unprocessed issues
	issues, err := e.stagingDB.FetchUnprocessedIssues(ctx, repoID, 1000)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch issues: %w", err)
	}

	if len(issues) == 0 {
		log.Printf("  ℹ️  No unprocessed issues found")
		return 0, nil
	}

	// Process in batches
	for i := 0; i < len(issues); i += batchSize {
		end := i + batchSize
		if end > len(issues) {
			end = len(issues)
		}

		batch := issues[i:end]
		refs, err := e.processBatchWithComments(ctx, repoID, batch, repo.Owner, collaborators)
		if err != nil {
			log.Printf("  ⚠️  Failed to process batch %d-%d: %v", i, end, err)
			continue
		}

		totalRefs += refs
		log.Printf("  ✓ Processed issues %d-%d: extracted %d references", i+1, end, refs)
	}

	return totalRefs, nil
}

// processBatchWithComments processes a batch of issues with comment analysis
func (e *IssueExtractor) processBatchWithComments(
	ctx context.Context,
	repoID int64,
	issues []database.IssueData,
	repoOwner string,
	collaborators []string,
) (int, error) {
	var allRefs []database.IssueCommitRef

	// Create comment analyzer
	commentAnalyzer := llm.NewCommentAnalyzer(e.llmClient)

	for _, issue := range issues {
		// Fetch comments for this issue
		comments, err := e.fetchIssueComments(ctx, repoID, issue.Number)
		if err != nil {
			log.Printf("  ⚠️  Failed to fetch comments for issue #%d: %v", issue.Number, err)
			comments = []llm.Comment{} // Continue without comments
		}

		// Extract references using comment analyzer
		refs, err := commentAnalyzer.ExtractCommentReferences(
			ctx,
			issue.Number,
			issue.Title,
			issue.Body,
			issue.ClosedAt,
			comments,
			repoOwner,
			collaborators,
		)

		if err != nil {
			log.Printf("  ⚠️  Comment analysis failed for issue #%d: %v", issue.Number, err)
			continue
		}

		// Convert to database format
		for _, ref := range refs {
			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     issue.Number,
				Action:          ref.Action,
				Confidence:      ref.Confidence,
				DetectionMethod: "comment_extraction",
				ExtractedFrom:   ref.Source,
			}

			if ref.Type == "commit" {
				commitSHA := ref.ID
				dbRef.CommitSHA = &commitSHA
			} else if ref.Type == "pr" {
				var prNum int
				fmt.Sscanf(ref.ID, "%d", &prNum)
				dbRef.PRNumber = &prNum
			}

			allRefs = append(allRefs, dbRef)
		}
	}

	// Store all references in batch
	if len(allRefs) > 0 {
		if err := e.stagingDB.StoreIssueCommitRefs(ctx, allRefs); err != nil {
			return 0, fmt.Errorf("failed to store references: %w", err)
		}
	}

	return len(allRefs), nil
}

// fetchIssueComments fetches comments for an issue from GitHub API
func (e *IssueExtractor) fetchIssueComments(ctx context.Context, repoID int64, issueNumber int) ([]llm.Comment, error) {
	// Query github_issue_comments table
	// TODO: Implement based on your staging schema
	return []llm.Comment{}, nil
}

// fetchCollaborators fetches repository collaborators from GitHub API
func (e *IssueExtractor) fetchCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
	// Query github_collaborators table or fetch from API
	// TODO: Implement based on your staging schema
	return []string{}, nil
}
```

**Database Changes:**
```sql
-- Add table for issue comments if not exists
CREATE TABLE IF NOT EXISTS github_issue_comments (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT REFERENCES github_repositories(id),
    issue_id BIGINT,
    comment_id BIGINT UNIQUE,
    author_login TEXT,
    author_type TEXT, -- "User", "Bot"
    body TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    raw_data JSONB,
    processed BOOLEAN DEFAULT FALSE,
    created_at_db TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_github_issue_comments_issue ON github_issue_comments(repo_id, issue_id);
```

**Impact:** +15% coverage (Stagehand #1060, Omnara #187 partially)

---

### Phase 2: Integrate Temporal Correlation (Pattern 2) - HIGH PRIORITY

**File:** `internal/graph/issue_linker.go`

**Changes:**
1. Add `applyTemporalBoost()` method
2. Integrate `TemporalCorrelator` in `createIncidentEdges()`
3. Add helper methods to `StagingClient` for timestamp queries

**Implementation:**

```go
// internal/graph/issue_linker.go

func (l *IssueLinker) createIncidentEdges(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get all extracted references (from LLM + comments)
	refs, err := l.stagingDB.GetIssueCommitRefs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get references: %w", err)
	}

	if len(refs) == 0 {
		return stats, nil
	}

	// 🆕 STEP 1: Apply Temporal Correlation
	log.Printf("  🕐 Applying temporal correlation...")
	correlator := NewTemporalCorrelator(l.stagingDB, l.backend)
	temporalMatches, err := correlator.FindTemporalMatches(ctx, repoID)
	if err != nil {
		log.Printf("  ⚠️  Temporal correlation failed: %v", err)
		temporalMatches = []TemporalMatch{} // Continue without temporal
	}

	// Convert temporal matches to references
	for _, match := range temporalMatches {
		ref := database.IssueCommitRef{
			RepoID:          repoID,
			IssueNumber:     match.IssueNumber,
			Action:          "mentions", // Temporal is weak, not "fixes"
			Confidence:      match.Confidence,
			DetectionMethod: "temporal_correlation",
			ExtractedFrom:   "timestamp_analysis",
		}

		if match.TargetType == "pr" {
			var prNum int
			fmt.Sscanf(match.TargetID, "%d", &prNum)
			ref.PRNumber = &prNum
		} else if match.TargetType == "commit" {
			commitSHA := match.TargetID
			ref.CommitSHA = &commitSHA
		}

		refs = append(refs, ref)
	}

	log.Printf("  ✓ Found %d temporal matches", len(temporalMatches))

	// 🆕 STEP 2: Apply Semantic Matching (Pattern 4)
	log.Printf("  🔍 Applying semantic matching...")
	refs = l.applySemanticBoost(ctx, repoID, refs)

	// 🆕 STEP 3: Apply Bidirectional Validation (Pattern 5)
	log.Printf("  🔄 Applying bidirectional validation...")
	refs = l.applyBidirectionalBoost(ctx, repoID, refs)

	// 🆕 STEP 4: Merge references and combine evidence
	log.Printf("  🔗 Merging references...")
	mergedRefs := l.mergeReferencesEnhanced(refs)

	// Create edges with entity resolution (existing code)
	var edges []GraphEdge
	skippedLowConf := 0
	skippedNotFound := 0

	for _, ref := range mergedRefs {
		// Skip low-confidence references
		if ref.Confidence < 0.5 { // Lowered threshold from 0.4 to 0.5
			skippedLowConf++
			continue
		}

		// ... rest of existing edge creation logic ...
	}

	// Create all edges
	fmt.Printf("\n  📊 Summary: %d references processed\n", len(mergedRefs))
	fmt.Printf("    • Skipped (low confidence): %d\n", skippedLowConf)
	fmt.Printf("    • Skipped (not found): %d\n", skippedNotFound)
	fmt.Printf("    • Edges to create: %d\n\n", len(edges))

	if len(edges) > 0 {
		if err := l.backend.CreateEdges(ctx, edges); err != nil {
			return stats, fmt.Errorf("failed to create edges: %w", err)
		}
		stats.Edges = len(edges)
	}

	return stats, nil
}

// applySemanticBoost applies semantic similarity boost (Pattern 4)
func (l *IssueLinker) applySemanticBoost(ctx context.Context, repoID int64, refs []database.IssueCommitRef) []database.IssueCommitRef {
	// For each reference, compute semantic similarity between issue and PR/commit
	// Use Jaccard similarity on keywords

	for i := range refs {
		ref := &refs[i]

		// Get issue
		issue, err := l.stagingDB.GetIssue(ctx, repoID, ref.IssueNumber)
		if err != nil {
			continue
		}

		// Get PR or commit
		var targetText string
		if ref.PRNumber != nil {
			pr, err := l.stagingDB.GetPullRequest(ctx, repoID, *ref.PRNumber)
			if err != nil {
				continue
			}
			targetText = pr.Title + " " + pr.Body
		} else if ref.CommitSHA != nil {
			commit, err := l.stagingDB.GetCommit(ctx, repoID, *ref.CommitSHA)
			if err != nil {
				continue
			}
			targetText = commit.Message
		}

		// Compute similarity
		similarity := computeSemanticSimilarity(issue.Title+" "+issue.Body, targetText)

		// Apply boost
		if similarity >= 0.70 {
			ref.Confidence += 0.15
		} else if similarity >= 0.50 {
			ref.Confidence += 0.10
		} else if similarity >= 0.30 {
			ref.Confidence += 0.05
		}

		// Cap at 0.98
		if ref.Confidence > 0.98 {
			ref.Confidence = 0.98
		}
	}

	return refs
}

// applyBidirectionalBoost validates bidirectional mentions (Pattern 5)
func (l *IssueLinker) applyBidirectionalBoost(ctx context.Context, repoID int64, refs []database.IssueCommitRef) []database.IssueCommitRef {
	// Check if issue body mentions the PR/commit, and vice versa
	// If both mention each other, boost confidence by +0.10

	for i := range refs {
		ref := &refs[i]

		// Get issue
		issue, err := l.stagingDB.GetIssue(ctx, repoID, ref.IssueNumber)
		if err != nil {
			continue
		}

		// Check if issue mentions the PR/commit
		if ref.PRNumber != nil {
			prMention := fmt.Sprintf("#%d", *ref.PRNumber)
			if strings.Contains(issue.Body, prMention) {
				ref.Confidence += 0.10
				if ref.Confidence > 0.98 {
					ref.Confidence = 0.98
				}
			}
		}
	}

	return refs
}

// mergeReferencesEnhanced merges references with enhanced evidence tracking
func (l *IssueLinker) mergeReferencesEnhanced(refs []database.IssueCommitRef) []database.IssueCommitRef {
	// Group by (issue_number, commit_sha, pr_number) tuple
	type refKey struct {
		IssueNumber int
		CommitSHA   string
		PRNumber    int
	}

	refMap := make(map[refKey]*database.IssueCommitRef)

	for _, ref := range refs {
		commitSHA := ""
		if ref.CommitSHA != nil {
			commitSHA = *ref.CommitSHA
		}

		prNumber := 0
		if ref.PRNumber != nil {
			prNumber = *ref.PRNumber
		}

		key := refKey{
			IssueNumber: ref.IssueNumber,
			CommitSHA:   commitSHA,
			PRNumber:    prNumber,
		}

		existing, exists := refMap[key]
		if !exists {
			// First time seeing this reference
			refCopy := ref
			refMap[key] = &refCopy
		} else {
			// Merge evidence: take max confidence + boost for multiple sources
			if ref.Confidence > existing.Confidence {
				existing.Confidence = ref.Confidence
			}

			// Count unique detection methods
			methods := map[string]bool{
				existing.DetectionMethod: true,
				ref.DetectionMethod:      true,
			}

			// Multi-evidence boost: +0.03 per additional source
			if len(methods) > 1 {
				existing.Confidence += 0.03 * float64(len(methods)-1)
				if existing.Confidence > 0.98 {
					existing.Confidence = 0.98
				}
				existing.DetectionMethod = "multi_source"
			}

			// Prefer "fixes" over "mentions"
			if ref.Action == "fixes" {
				existing.Action = "fixes"
			}
		}
	}

	// Convert map back to slice
	merged := make([]database.IssueCommitRef, 0, len(refMap))
	for _, ref := range refMap {
		merged = append(merged, *ref)
	}

	return merged
}

// computeSemanticSimilarity computes Jaccard similarity between two texts
func computeSemanticSimilarity(text1, text2 string) float64 {
	// Extract keywords (simple: split by space, lowercase, dedupe)
	words1 := extractKeywords(text1)
	words2 := extractKeywords(text2)

	// Compute intersection and union
	intersection := 0
	for word := range words1 {
		if words2[word] {
			intersection++
		}
	}

	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func extractKeywords(text string) map[string]bool {
	// Simple keyword extraction
	words := strings.Fields(strings.ToLower(text))
	keywords := make(map[string]bool)

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
	}

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,;:!?()[]{}\"'")
		if len(word) > 2 && !stopWords[word] {
			keywords[word] = true
		}
	}

	return keywords
}
```

**Database Changes:**

```sql
-- Add helper methods to StagingClient
-- internal/database/staging.go

-- GetIssue retrieves a single issue by number
func (c *StagingClient) GetIssue(ctx context.Context, repoID int64, issueNumber int) (*IssueData, error) {
	query := `
		SELECT id, number, title, body, state, created_at, closed_at, labels, raw_data
		FROM github_issues
		WHERE repo_id = $1 AND number = $2
	`

	var issue IssueData
	err := c.db.QueryRowContext(ctx, query, repoID, issueNumber).Scan(
		&issue.ID,
		&issue.Number,
		&issue.Title,
		&issue.Body,
		&issue.State,
		&issue.CreatedAt,
		&issue.ClosedAt,
		&issue.Labels,
		&issue.RawData,
	)
	if err != nil {
		return nil, err
	}

	return &issue, nil
}

-- GetPullRequest retrieves a single PR by number
func (c *StagingClient) GetPullRequest(ctx context.Context, repoID int64, prNumber int) (*PullRequestData, error) {
	query := `
		SELECT id, number, title, body, state, created_at, merged_at, raw_data
		FROM github_pull_requests
		WHERE repo_id = $1 AND number = $2
	`

	var pr PullRequestData
	err := c.db.QueryRowContext(ctx, query, repoID, prNumber).Scan(
		&pr.ID,
		&pr.Number,
		&pr.Title,
		&pr.Body,
		&pr.State,
		&pr.CreatedAt,
		&pr.MergedAt,
		&pr.RawData,
	)
	if err != nil {
		return nil, err
	}

	return &pr, nil
}

-- GetCommit retrieves a single commit by SHA
func (c *StagingClient) GetCommit(ctx context.Context, repoID int64, commitSHA string) (*CommitData, error) {
	query := `
		SELECT id, sha, message, author_email, author_date, raw_data
		FROM github_commits
		WHERE repo_id = $1 AND sha = $2
	`

	var commit CommitData
	err := c.db.QueryRowContext(ctx, query, repoID, commitSHA).Scan(
		&commit.ID,
		&commit.SHA,
		&commit.Message,
		&commit.AuthorEmail,
		&commit.AuthorDate,
		&commit.RawData,
	)
	if err != nil {
		return nil, err
	}

	return &commit, nil
}
```

**Impact:** +20% coverage (Omnara #221, #189, #187 with temporal)

---

### Phase 3: Implement Merge Commit Detection (Pattern 6) - LOW PRIORITY

**File:** `internal/github/commit_extractor.go`

**Changes:**
1. Add `ExtractMergeCommitReferences()` method
2. Detect merge commits (2+ parents)
3. Extract issue numbers from merge commit messages

**Implementation:**

```go
// internal/github/commit_extractor.go

func (e *CommitExtractor) ExtractMergeCommitReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting references from merge commits...")

	// Fetch all commits
	commits, err := e.stagingDB.FetchUnprocessedCommits(ctx, repoID, 10000)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch commits: %w", err)
	}

	var allRefs []database.IssueCommitRef

	for _, commit := range commits {
		// Check if this is a merge commit (has 2+ parents)
		var rawData struct {
			Parents []struct {
				SHA string `json:"sha"`
			} `json:"parents"`
		}

		if err := json.Unmarshal(commit.RawData, &rawData); err != nil {
			continue
		}

		if len(rawData.Parents) < 2 {
			continue // Not a merge commit
		}

		// Extract issue numbers from merge commit message
		refs := extractIssueNumbers(commit.Message)

		for _, ref := range refs {
			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     ref.IssueNumber,
				CommitSHA:       &commit.SHA,
				Action:          "fixes",
				Confidence:      0.85, // High confidence for merge commits
				DetectionMethod: "merge_commit",
				ExtractedFrom:   "commit_message",
			}

			allRefs = append(allRefs, dbRef)
		}
	}

	// Store references
	if len(allRefs) > 0 {
		if err := e.stagingDB.StoreIssueCommitRefs(ctx, allRefs); err != nil {
			return 0, fmt.Errorf("failed to store references: %w", err)
		}
	}

	log.Printf("  ✓ Extracted %d references from merge commits", len(allRefs))
	return len(allRefs), nil
}

func extractIssueNumbers(message string) []struct{ IssueNumber int } {
	// Extract #123 patterns from message
	re := regexp.MustCompile(`#(\d+)`)
	matches := re.FindAllStringSubmatch(message, -1)

	var refs []struct{ IssueNumber int }
	for _, match := range matches {
		if len(match) > 1 {
			issueNum, _ := strconv.Atoi(match[1])
			refs = append(refs, struct{ IssueNumber int }{issueNum})
		}
	}

	return refs
}
```

**Integration Point:** Add to `cmd/crisk/init.go` Stage 1.5:

```go
// Extract from Merge Commits (Pattern 6)
mergeRefs, err := commitExtractor.ExtractMergeCommitReferences(ctx, repoID)
if err != nil {
	fmt.Printf("  ⚠️  Merge commit extraction failed: %v\n", err)
} else {
	fmt.Printf("  ✓ Extracted %d references from merge commits\n", mergeRefs)
}
```

**Impact:** +5% coverage (Supabase #39872, Omnara #221 merge commits)

---

## Testing Framework

### Unit Tests (Per Pattern)

Create `internal/graph/issue_linker_test.go`:

```go
package graph_test

import (
	"testing"
	"time"

	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/stretchr/testify/assert"
)

func TestTemporalCorrelation(t *testing.T) {
	tests := []struct {
		name           string
		issueClosedAt  time.Time
		prMergedAt     time.Time
		expectedConf   float64
		expectedEvid   string
	}{
		{
			name:          "5 minute match",
			issueClosedAt: time.Now(),
			prMergedAt:    time.Now().Add(-3 * time.Minute),
			expectedConf:  0.75,
			expectedEvid:  "temporal_match_5min",
		},
		{
			name:          "1 hour match",
			issueClosedAt: time.Now(),
			prMergedAt:    time.Now().Add(-30 * time.Minute),
			expectedConf:  0.65,
			expectedEvid:  "temporal_match_1hr",
		},
		{
			name:          "24 hour match",
			issueClosedAt: time.Now(),
			prMergedAt:    time.Now().Add(-12 * time.Hour),
			expectedConf:  0.55,
			expectedEvid:  "temporal_match_24hr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := graph.CreateTemporalMatch(1, tt.issueClosedAt, "pr", "123", tt.prMergedAt)
			assert.Equal(t, tt.expectedConf, match.Confidence)
			assert.Contains(t, match.Evidence, tt.expectedEvid)
		})
	}
}

func TestSemanticSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		expectedSim float64
	}{
		{
			name:       "identical text",
			text1:      "fix authentication bug",
			text2:      "fix authentication bug",
			expectedSim: 1.0,
		},
		{
			name:       "high similarity",
			text1:      "fix mobile interface sync issues",
			text2:      "update mobile interface sync with Claude Code",
			expectedSim: 0.60, // ["mobile", "interface", "sync"] / ["mobile", "interface", "sync", "update", "with", "Claude", "Code"]
		},
		{
			name:       "no similarity",
			text1:      "add new feature",
			text2:      "fix authentication bug",
			expectedSim: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := graph.ComputeSemanticSimilarity(tt.text1, tt.text2)
			assert.InDelta(t, tt.expectedSim, sim, 0.1)
		})
	}
}
```

---

### Integration Tests (Full Pipeline)

Create `cmd/test_full_graph/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	// Connect to databases
	stagingDB, _ := database.NewStagingClient(ctx, "localhost", 5433, "coderisk", "coderisk", "password")
	neo4jDB, _ := graph.NewNeo4jBackend(ctx, "bolt://localhost:7688", "neo4j", "password", "neo4j")

	// Load ground truth
	groundTruth := loadGroundTruth("test_data/omnara_ground_truth.json")

	// Run full pipeline
	repoID := int64(1) // omnara-ai/omnara
	builder := graph.NewBuilder(stagingDB, neo4jDB)
	stats, err := builder.BuildGraph(ctx, repoID, "/tmp/omnara")
	if err != nil {
		fmt.Printf("❌ Graph build failed: %v\n", err)
		os.Exit(1)
	}

	// Validate results
	precision, recall, f1 := validateResults(ctx, neo4jDB, repoID, groundTruth)

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  TEST RESULTS SUMMARY                                        ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Repository:      omnara-ai/omnara                            ║\n")
	fmt.Printf("║ F1 Score:        %.1f%%                                      ║\n", f1*100)
	fmt.Printf("║ Precision:       %.1f%%                                      ║\n", precision*100)
	fmt.Printf("║ Recall:          %.1f%%                                      ║\n", recall*100)
	fmt.Printf("║ Status:          %s                                          ║\n", getStatus(f1))
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")

	if f1 < 0.75 {
		os.Exit(1)
	}
}
```

---

## Implementation Roadmap

### Week 1: Comment Analysis (Pattern 3)
- [ ] Day 1-2: Add comment fetching to `github.Fetcher`
- [ ] Day 3-4: Integrate `CommentAnalyzer` into `IssueExtractor`
- [ ] Day 5: Test with Stagehand #1060

**Expected Result:** F1 Score: 60% → 68% (+8%)

### Week 2: Temporal Correlation (Pattern 2)
- [ ] Day 1-2: Add helper methods to `StagingClient`
- [ ] Day 3-4: Integrate `TemporalCorrelator` into `IssueLinker`
- [ ] Day 5: Test with Omnara #221, #189

**Expected Result:** F1 Score: 68% → 75% (+7%)

### Week 3: Semantic & Bidirectional (Patterns 4 & 5)
- [ ] Day 1-2: Implement semantic similarity matcher
- [ ] Day 3: Implement bidirectional validator
- [ ] Day 4-5: Integrate both into `IssueLinker`

**Expected Result:** F1 Score: 75% → 78% (+3%)

### Week 4: Testing & Tuning
- [ ] Day 1-2: Write unit tests for all patterns
- [ ] Day 3: Run full pipeline tests on all 3 repos
- [ ] Day 4-5: Tune confidence thresholds

**Expected Result:** F1 Score: 78% → 80% (+2%)

---

## Success Metrics

| Metric | Baseline | Target | Actual |
|--------|----------|--------|--------|
| **F1 Score (Overall)** | 60% | 75% | TBD |
| **Omnara** | 50% | 70% | TBD |
| **Supabase** | 85% | 90% | TBD |
| **Stagehand** | 60% | 80% | TBD |
| **Precision** | 70% | 85% | TBD |
| **Recall** | 50% | 70% | TBD |

---

## Risk Mitigation

### Risk 1: LLM Cost Explosion
**Mitigation:** Batch comment analysis (20 issues at a time), cache results in Postgres

### Risk 2: Temporal False Positives
**Mitigation:** Cap temporal-only confidence at 0.75, require secondary evidence for "fixes" action

### Risk 3: Performance Degradation
**Mitigation:** Add indexes on timestamp columns, batch process in chunks of 100

### Risk 4: Integration Complexity
**Mitigation:** Use feature flags, deploy patterns incrementally, A/B test confidence thresholds

---

## Next Steps

1. **Review this document** with the team
2. **Choose implementation order** (recommend: Comment → Temporal → Semantic)
3. **Start with Phase 1** (Comment Analysis) - highest impact, lowest complexity
4. **Run baseline tests** to establish current F1 score
5. **Iterate and tune** after each pattern integration

---

## Appendix: File Modification Checklist

### Files to Modify
- [ ] `internal/github/issue_extractor.go` - Add comment analysis
- [ ] `internal/github/commit_extractor.go` - Add merge commit detection
- [ ] `internal/graph/issue_linker.go` - Add multi-pattern orchestration
- [ ] `internal/graph/temporal_correlator.go` - Complete implementation
- [ ] `internal/database/staging.go` - Add helper methods
- [ ] `cmd/crisk/init.go` - Integrate merge commit extraction

### Files to Create
- [ ] `internal/graph/semantic_matcher.go` - Semantic similarity
- [ ] `internal/graph/bidirectional_validator.go` - Cross-reference validation
- [ ] `cmd/test_full_graph/main.go` - Integration test runner
- [ ] `scripts/test_full_pipeline.sh` - Automated testing script

### Database Changes
- [ ] Add `github_issue_comments` table
- [ ] Add indexes on timestamp columns
- [ ] Add `evidence` JSONB column to `github_issue_commit_refs`

---

**Document Version:** 1.0
**Last Updated:** 2025-11-01
**Author:** Claude (AI Assistant)
**Status:** READY FOR REVIEW
