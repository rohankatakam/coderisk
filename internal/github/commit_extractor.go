package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// CommitExtractor extracts issue references from commits and PRs using LLM
// Reference: REVISED_MVP_STRATEGY.md - Phase 2: Extract from Commits/PRs
type CommitExtractor struct {
	llmClient *llm.Client
	stagingDB *database.StagingClient
}

// NewCommitExtractor creates a commit extractor
func NewCommitExtractor(llmClient *llm.Client, stagingDB *database.StagingClient) *CommitExtractor {
	return &CommitExtractor{
		llmClient: llmClient,
		stagingDB: stagingDB,
	}
}

// CommitReference represents a reference from a commit/PR to an issue/PR
type CommitReference struct {
	TargetID   int     `json:"target_id"`  // Issue or PR number (resolved later)
	Action     string  `json:"action"`     // "fixes", "closes", "resolves", "mentions"
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
}

// ExtractCommitReferences processes all commits and extracts issue references
func (e *CommitExtractor) ExtractCommitReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting issue references from commits...")

	batchSize := 20 // Batch size for LLM processing
	totalRefs := 0

	// Fetch unprocessed commits
	commits, err := e.stagingDB.FetchUnprocessedCommits(ctx, repoID, 1000)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch commits: %w", err)
	}

	if len(commits) == 0 {
		log.Printf("  ℹ️  No unprocessed commits found")
		return 0, nil
	}

	// Process in batches
	for i := 0; i < len(commits); i += batchSize {
		end := i + batchSize
		if end > len(commits) {
			end = len(commits)
		}

		batch := commits[i:end]
		refs, err := e.processCommitBatch(ctx, repoID, batch)
		if err != nil {
			log.Printf("  ⚠️  Failed to process batch %d-%d: %v", i, end, err)
			continue
		}

		totalRefs += refs
		log.Printf("  ✓ Processed commits %d-%d: extracted %d references", i+1, end, refs)
	}

	return totalRefs, nil
}

// ExtractPRReferences processes all PRs and extracts issue references
func (e *CommitExtractor) ExtractPRReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting issue references from pull requests...")

	batchSize := 20 // Batch size for LLM processing
	totalRefs := 0

	// Fetch unprocessed PRs
	prs, err := e.stagingDB.FetchUnprocessedPRs(ctx, repoID, 1000)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	if len(prs) == 0 {
		log.Printf("  ℹ️  No unprocessed PRs found")
		return 0, nil
	}

	// Process in batches
	for i := 0; i < len(prs); i += batchSize {
		end := i + batchSize
		if end > len(prs) {
			end = len(prs)
		}

		batch := prs[i:end]
		refs, err := e.processPRBatch(ctx, repoID, batch)
		if err != nil {
			log.Printf("  ⚠️  Failed to process batch %d-%d: %v", i, end, err)
			continue
		}

		totalRefs += refs
		log.Printf("  ✓ Processed PRs %d-%d: extracted %d references", i+1, end, refs)
	}

	return totalRefs, nil
}

// processCommitBatch processes a batch of commits
func (e *CommitExtractor) processCommitBatch(ctx context.Context, repoID int64, commits []database.CommitData) (int, error) {
	// COLLISION PREVENTION: Filter out merge commits
	// Merge commits already have MERGED_AS edges (100% confidence from PR.merge_commit_sha)
	// We don't want to create duplicate ASSOCIATED_WITH edges with lower confidence
	// Reference: Gap analysis 2025-11-13 - Prevent collision between MERGED_AS and ASSOCIATED_WITH
	var nonMergeCommits []database.CommitData
	for _, commit := range commits {
		// Skip commits that start with "Merge pull request"
		// These are GitHub's automatic merge commit messages
		if !strings.HasPrefix(commit.Message, "Merge pull request") {
			nonMergeCommits = append(nonMergeCommits, commit)
		}
	}

	// If all commits were merge commits, nothing to process
	if len(nonMergeCommits) == 0 {
		return 0, nil
	}

	// Process only non-merge commits
	commits = nonMergeCommits

	// Build prompt
	systemPrompt := `You are a Git commit analyzer. Extract issue/PR references from commit messages.

Output JSON format:
{
  "results": [
    {
      "id": "abc123",
      "references": [
        {"target_id": 123, "action": "fixes", "confidence": 0.95},
        {"target_id": 456, "action": "mentions", "confidence": 0.75}
      ]
    }
  ]
}

Rules:
- Extract ONLY the number from patterns like "Fixes #123", "#42", "PR#42"
- target_id: just the digits (e.g., 123, not "#123" or "PR#123")
- action: "fixes" for closes/resolves/fixes/fix keywords, "mentions" for related/see/ref
- confidence: 0.9-1.0 for "Fixes #123", 0.7-0.9 for "fix #123", 0.5-0.7 for "related #123"
- IGNORE negations: "Don't fix #123", "Not fixing #456"
- IGNORE discussions: "Similar to #123 but different"
- IGNORE future plans: "Will fix #123 later", "TODO: fix #123"`

	userPrompt := "Analyze these commits:\n\n"
	for _, commit := range commits {
		userPrompt += fmt.Sprintf("Commit %s:\n%s\n\n", commit.SHA[:10], truncateText(commit.Message, 500))
	}

	// Call OpenAI
	response, err := e.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return 0, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse response
	var result struct {
		Results []struct {
			ID         string            `json:"id"`
			References []CommitReference `json:"references"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Store extracted references
	// CRITICAL FIX: Match by SHA instead of array index to prevent misattribution
	// See: EXTRACTION_BUG_ROOT_CAUSE.md - Bug discovered 2025-11-13
	var allRefs []database.IssueCommitRef

	// Create a map of SHA → commit for O(1) lookup
	commitMap := make(map[string]database.CommitData)
	for _, commit := range commits {
		// Use first 10 chars of SHA as key to match LLM output format
		key := commit.SHA
		if len(key) > 10 {
			key = key[:10]
		}
		commitMap[key] = commit
		// Also store full SHA for backward compatibility
		commitMap[commit.SHA] = commit
	}

	// Match LLM results by ID (commit SHA), not array index
	validationStats := struct {
		matched   int
		notFound  int
		validated int
		failed    int
	}{}

	for _, output := range result.Results {
		// Look up commit by SHA
		commit, found := commitMap[output.ID]
		if !found {
			log.Printf("  ⚠️  LLM returned unknown commit ID %s - skipping", output.ID)
			validationStats.notFound++
			continue
		}
		validationStats.matched++

		for _, ref := range output.References {
			commitSHA := commit.SHA

			// VALIDATION: Verify the reference actually exists in the commit message
			// This catches LLM hallucinations and cross-contamination bugs
			messageText := strings.ToLower(commit.Message)
			targetStr := fmt.Sprintf("#%d", ref.TargetID)
			targetStrLower := strings.ToLower(targetStr)

			validated := strings.Contains(messageText, targetStrLower) ||
				strings.Contains(messageText, fmt.Sprintf("pr %d", ref.TargetID)) ||
				strings.Contains(messageText, fmt.Sprintf("pr#%d", ref.TargetID)) ||
				strings.Contains(messageText, fmt.Sprintf("issue %d", ref.TargetID)) ||
				strings.Contains(messageText, fmt.Sprintf("issue#%d", ref.TargetID))

			if validated {
				validationStats.validated++
			} else {
				// Lower confidence for unvalidated references
				log.Printf("  ⚠️  Validation failed: Commit %.7s doesn't contain %s (message: %q)",
					commit.SHA, targetStr, truncateText(commit.Message, 60))

				// Reduce confidence significantly but don't discard entirely
				// (might be legitimate reference in non-standard format)
				ref.Confidence = ref.Confidence * 0.3
				validationStats.failed++

				// Skip if confidence drops too low
				if ref.Confidence < 0.2 {
					log.Printf("  ⚠️  Skipping low-confidence reference after validation failure")
					continue
				}
			}

			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     ref.TargetID, // Will be resolved to Issue or PR later
				CommitSHA:       &commitSHA,
				Action:          ref.Action,
				Confidence:      ref.Confidence,
				DetectionMethod: "commit_extraction",
				ExtractedFrom:   "commit_message",
			}

			allRefs = append(allRefs, dbRef)
		}
	}

	// Log validation statistics
	if validationStats.matched > 0 {
		log.Printf("  📊 Validation: %d matched, %d validated, %d failed, %d not found",
			validationStats.matched, validationStats.validated,
			validationStats.failed, validationStats.notFound)
	}

	// Store all references in batch
	if len(allRefs) > 0 {
		if err := e.stagingDB.StoreIssueCommitRefs(ctx, allRefs); err != nil {
			return 0, fmt.Errorf("failed to store references: %w", err)
		}
	}

	return len(allRefs), nil
}

// processPRBatch processes a batch of PRs
func (e *CommitExtractor) processPRBatch(ctx context.Context, repoID int64, prs []database.PRData) (int, error) {
	// Build prompt
	systemPrompt := `You are a GitHub PR analyzer. Extract issue/PR references from PR titles and bodies.

Output JSON format:
{
  "results": [
    {
      "id": "123",
      "references": [
        {"target_id": 456, "action": "fixes", "confidence": 0.95},
        {"target_id": 789, "action": "mentions", "confidence": 0.75}
      ]
    }
  ]
}

Rules:
- Extract ONLY the number from patterns like "Fixes #123", "#42", "PR#42"
- target_id: just the digits (e.g., 123, not "#123" or "PR#123")
- Check both title and body
- action: "fixes" for closes/resolves/fixes/fix keywords, "mentions" for related/see/ref
- confidence: 0.9-1.0 for "Fixes #123", 0.7-0.9 for "fix #123", 0.5-0.7 for "related #123"
- IGNORE negations and mentions in code blocks
- IGNORE discussions: "Similar to #123 but different"
- IGNORE future plans: "Will fix #123 later"`

	userPrompt := "Analyze these PRs:\n\n"
	for _, pr := range prs {
		userPrompt += fmt.Sprintf("PR #%d: %s\n", pr.Number, pr.Title)
		if pr.Body != "" {
			userPrompt += fmt.Sprintf("Body: %s\n", truncateText(pr.Body, 500))
		}
		userPrompt += "\n"
	}

	// Call OpenAI
	response, err := e.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return 0, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse response
	var result struct {
		Results []struct {
			ID         string            `json:"id"`
			References []CommitReference `json:"references"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Store extracted references
	// CRITICAL FIX: Match by PR number instead of array index to prevent misattribution
	// See: EXTRACTION_BUG_ROOT_CAUSE.md - Bug discovered 2025-11-13
	var allRefs []database.IssueCommitRef

	// Create a map of PR number → PR data for O(1) lookup
	prMap := make(map[string]database.PRData)
	for _, pr := range prs {
		// LLM returns PR number as string in the "id" field
		prMap[fmt.Sprintf("%d", pr.Number)] = pr
	}

	// Match LLM results by ID (PR number), not array index
	validationStats := struct {
		matched   int
		notFound  int
		validated int
		failed    int
	}{}

	for _, output := range result.Results {
		// Look up PR by number
		pr, found := prMap[output.ID]
		if !found {
			log.Printf("  ⚠️  LLM returned unknown PR ID %s - skipping", output.ID)
			validationStats.notFound++
			continue
		}
		validationStats.matched++

		for _, ref := range output.References {
			prNumber := pr.Number

			// VALIDATION: Verify the reference actually exists in PR title or body
			// This catches LLM hallucinations and cross-contamination bugs
			titleText := strings.ToLower(pr.Title)
			bodyText := strings.ToLower(pr.Body)
			combinedText := titleText + " " + bodyText
			targetStr := fmt.Sprintf("#%d", ref.TargetID)
			targetStrLower := strings.ToLower(targetStr)

			validated := strings.Contains(combinedText, targetStrLower) ||
				strings.Contains(combinedText, fmt.Sprintf("pr %d", ref.TargetID)) ||
				strings.Contains(combinedText, fmt.Sprintf("pr#%d", ref.TargetID)) ||
				strings.Contains(combinedText, fmt.Sprintf("issue %d", ref.TargetID)) ||
				strings.Contains(combinedText, fmt.Sprintf("issue#%d", ref.TargetID))

			if validated {
				validationStats.validated++
			} else {
				// Lower confidence for unvalidated references
				log.Printf("  ⚠️  Validation failed: PR #%d doesn't contain %s (title: %q)",
					pr.Number, targetStr, truncateText(pr.Title, 60))

				// Reduce confidence significantly but don't discard entirely
				ref.Confidence = ref.Confidence * 0.3
				validationStats.failed++

				// Skip if confidence drops too low
				if ref.Confidence < 0.2 {
					log.Printf("  ⚠️  Skipping low-confidence reference after validation failure")
					continue
				}
			}

			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     ref.TargetID, // Will be resolved to Issue or PR later
				PRNumber:        &prNumber,
				Action:          ref.Action,
				Confidence:      ref.Confidence,
				DetectionMethod: "pr_extraction",
				ExtractedFrom:   "pr_body",
			}

			// If PR has merge commit SHA, also link to commit
			if pr.MergeCommitSHA != nil {
				dbRef.CommitSHA = pr.MergeCommitSHA
			}

			allRefs = append(allRefs, dbRef)
		}
	}

	// Log validation statistics
	if validationStats.matched > 0 {
		log.Printf("  📊 Validation: %d matched, %d validated, %d failed, %d not found",
			validationStats.matched, validationStats.validated,
			validationStats.failed, validationStats.notFound)
	}

	// Store all references in batch
	if len(allRefs) > 0 {
		if err := e.stagingDB.StoreIssueCommitRefs(ctx, allRefs); err != nil {
			return 0, fmt.Errorf("failed to store references: %w", err)
		}
	}

	return len(allRefs), nil
}

// truncateText limits string length with ellipsis
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
