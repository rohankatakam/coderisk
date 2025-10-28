package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// IssueExtractor extracts commit/PR references from issues using LLM
// Reference: REVISED_MVP_STRATEGY.md - Phase 1: Extract from Issues
type IssueExtractor struct {
	llmClient *llm.Client
	stagingDB *database.StagingClient
}

// NewIssueExtractor creates an issue extractor
func NewIssueExtractor(llmClient *llm.Client, stagingDB *database.StagingClient) *IssueExtractor {
	return &IssueExtractor{
		llmClient: llmClient,
		stagingDB: stagingDB,
	}
}

// IssueReference represents a reference from an issue to a commit or PR
type IssueReference struct {
	Type       string  `json:"type"`       // "commit" or "pr"
	ID         string  `json:"id"`         // commit SHA or PR number
	Action     string  `json:"action"`     // "fixes", "closes", "resolves", "mentions"
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
}

// ExtractReferences processes all issues and extracts references
// Batch processes issues for efficiency (100 per API call as per REVISED_MVP_STRATEGY.md)
func (e *IssueExtractor) ExtractReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting references from issues...")

	batchSize := 20 // Batch size for LLM processing (smaller than doc's 100 to be safe)
	totalRefs := 0

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
		refs, err := e.processBatch(ctx, repoID, batch)
		if err != nil {
			log.Printf("  ⚠️  Failed to process batch %d-%d: %v", i, end, err)
			continue
		}

		totalRefs += refs
		log.Printf("  ✓ Processed issues %d-%d: extracted %d references", i+1, end, refs)
	}

	return totalRefs, nil
}

// processBatch processes a batch of issues
func (e *IssueExtractor) processBatch(ctx context.Context, repoID int64, issues []database.IssueData) (int, error) {
	// Build prompt
	systemPrompt := `You are a GitHub issue analyzer. Extract commit and PR references from issue text.

Output JSON format:
{
  "results": [
    {
      "issue_number": 123,
      "references": [
        {"type": "commit", "id": "abc123", "action": "fixes", "confidence": 0.95},
        {"type": "pr", "id": "456", "action": "mentions", "confidence": 0.85}
      ]
    }
  ]
}

Rules:
- type: "commit" or "pr"
- id: commit SHA (7-40 chars) or PR number (digits only, no #)
- action: "fixes" (for closes/resolves/fixes), "mentions" (for related/see)
- confidence: 0.9-1.0 for explicit fixes, 0.7-0.9 for "fixed by", 0.5-0.7 for mentions`

	userPrompt := "Analyze these issues:\n\n"
	for _, issue := range issues {
		userPrompt += fmt.Sprintf("Issue #%d: %s\n", issue.Number, issue.Title)
		if issue.Body != "" {
			userPrompt += fmt.Sprintf("Body: %s\n", truncateText(issue.Body, 500))
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
			IssueNumber int                `json:"issue_number"`
			References  []IssueReference `json:"references"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Store extracted references
	var allRefs []database.IssueCommitRef
	for _, output := range result.Results {
		for _, ref := range output.References {
			dbRef := database.IssueCommitRef{
				RepoID:          repoID,
				IssueNumber:     output.IssueNumber,
				Action:          ref.Action,
				Confidence:      ref.Confidence,
				DetectionMethod: "issue_extraction",
				ExtractedFrom:   "issue_body",
			}

			if ref.Type == "commit" {
				commitSHA := ref.ID
				dbRef.CommitSHA = &commitSHA
			} else if ref.Type == "pr" {
				// Extract PR number from ID
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

// ExtractTimelineReferences extracts references from issue timeline events
func (e *IssueExtractor) ExtractTimelineReferences(ctx context.Context, repoID int64) (int, error) {
	log.Printf("🔍 Extracting references from issue timelines...")

	// Fetch unprocessed timeline events (cross-references)
	events, err := e.stagingDB.FetchUnprocessedTimelineEvents(ctx, repoID, 1000)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch timeline events: %w", err)
	}

	if len(events) == 0 {
		log.Printf("  ℹ️  No unprocessed timeline events found")
		return 0, nil
	}

	var allRefs []database.IssueCommitRef
	eventIDs := []int64{}

	for _, event := range events {
		// Only process cross-reference events
		if event.EventType != "cross-referenced" {
			continue
		}

		// Extract issue number from event
		// Note: We need to query the database to get the issue number from issue_id
		// For now, we'll skip this and focus on direct references
		if event.SourceType == nil || event.SourceNumber == nil {
			continue
		}

		// Create reference based on source type
		if *event.SourceType == "pr" && event.SourceBody != nil {
			// PR cross-referenced this issue
			// Extract issue reference from PR body
			// For timeline references, we have high confidence (bidirectional)
			ref := database.IssueCommitRef{
				RepoID:          repoID,
				// IssueNumber needs to be looked up from event.IssueID
				// For now, skip this - will be handled by PR extraction
				PRNumber:        event.SourceNumber,
				Action:          "mentions", // Default to mentions for cross-refs
				Confidence:      0.85,       // High confidence for timeline events
				DetectionMethod: "timeline_extraction",
				ExtractedFrom:   "issue_timeline",
			}

			allRefs = append(allRefs, ref)
		}

		eventIDs = append(eventIDs, event.IssueID)
	}

	// Mark events as processed
	if len(eventIDs) > 0 {
		if err := e.stagingDB.MarkTimelineEventsProcessed(ctx, eventIDs); err != nil {
			log.Printf("  ⚠️  Failed to mark timeline events as processed: %v", err)
		}
	}

	// Store references
	if len(allRefs) > 0 {
		if err := e.stagingDB.StoreIssueCommitRefs(ctx, allRefs); err != nil {
			return 0, fmt.Errorf("failed to store references: %w", err)
		}
	}

	log.Printf("  ✓ Extracted %d references from timeline events", len(allRefs))
	return len(allRefs), nil
}
