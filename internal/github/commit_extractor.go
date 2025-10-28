package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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

// CommitReference represents a reference from a commit/PR to an issue
type CommitReference struct {
	IssueNumber int     `json:"issue_number"`
	Action      string  `json:"action"`     // "fixes", "closes", "resolves", "mentions"
	Confidence  float64 `json:"confidence"` // 0.0 to 1.0
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
	// Build prompt
	systemPrompt := `You are a Git commit analyzer. Extract issue references from commit messages.

Output JSON format:
{
  "results": [
    {
      "id": "abc123",
      "references": [
        {"issue_number": 123, "action": "fixes", "confidence": 0.95},
        {"issue_number": 456, "action": "mentions", "confidence": 0.75}
      ]
    }
  ]
}

Rules:
- Look for patterns: "Fixes #123", "Closes #456", "Resolves #789", "Related to #123"
- Ignore negations: "Don't fix #123", "Not fixing #456"
- action: "fixes" for closes/resolves/fixes keywords, "mentions" for related/see
- confidence: 0.9-1.0 for "Fixes #123", 0.7-0.9 for "fix #123", 0.5-0.7 for "related to #123"
- Extract all issue numbers (digits only, no #)`

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
	var allRefs []database.IssueCommitRef
	for i, output := range result.Results {
		commit := commits[i]

		for _, ref := range output.References {
			commitSHA := commit.SHA
			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     ref.IssueNumber,
				CommitSHA:       &commitSHA,
				Action:          ref.Action,
				Confidence:      ref.Confidence,
				DetectionMethod: "commit_extraction",
				ExtractedFrom:   "commit_message",
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

// processPRBatch processes a batch of PRs
func (e *CommitExtractor) processPRBatch(ctx context.Context, repoID int64, prs []database.PRData) (int, error) {
	// Build prompt
	systemPrompt := `You are a GitHub PR analyzer. Extract issue references from PR titles and bodies.

Output JSON format:
{
  "results": [
    {
      "id": "123",
      "references": [
        {"issue_number": 456, "action": "fixes", "confidence": 0.95},
        {"issue_number": 789, "action": "mentions", "confidence": 0.75}
      ]
    }
  ]
}

Rules:
- Look for patterns: "Fixes #123", "Closes #456", "Resolves #789", "Related to #123"
- Check both title and body
- Ignore negations and mentions in code blocks
- action: "fixes" for closes/resolves/fixes keywords, "mentions" for related/see
- confidence: 0.9-1.0 for "Fixes #123", 0.7-0.9 for "fix #123", 0.5-0.7 for "related to #123"
- Extract all issue numbers (digits only, no #)`

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
	var allRefs []database.IssueCommitRef
	for i, output := range result.Results {
		pr := prs[i]

		for _, ref := range output.References {
			prNumber := pr.Number
			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     ref.IssueNumber,
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
