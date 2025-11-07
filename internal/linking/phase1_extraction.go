package linking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/linking/types"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// prInfo holds PR info for batch processing
type prInfo struct {
	id     int64
	number int
	title  string
	body   string
}

// Phase1Extractor handles Phase 1: Explicit Reference Extraction
// Reference: Issue_Flow.md Phase 1
type Phase1Extractor struct {
	stagingDB *database.StagingClient
	llmClient *llm.Client
}

// NewPhase1Extractor creates a Phase 1 extractor
func NewPhase1Extractor(stagingDB *database.StagingClient, llmClient *llm.Client) *Phase1Extractor {
	return &Phase1Extractor{
		stagingDB: stagingDB,
		llmClient: llmClient,
	}
}

// ExtractExplicitReferences extracts explicit issue references from all PRs
// Optimizes by skipping PRs already linked via timeline API
func (p *Phase1Extractor) ExtractExplicitReferences(ctx context.Context, repoID int64, timelineLinks map[int][]types.TimelineLink) (map[int][]types.ExplicitReference, error) {
	log.Printf("Phase 1: Extracting explicit references from PRs...")

	// Build set of (issue, PR) pairs already linked via timeline
	timelinePairs := make(map[string]bool)
	for issueNum, links := range timelineLinks {
		for _, link := range links {
			key := fmt.Sprintf("%d-%d", issueNum, link.PRNumber)
			timelinePairs[key] = true
		}
	}

	log.Printf("  ℹ️  Skipping %d PR-issue pairs already linked via timeline API", len(timelinePairs))

	// Get all merged PRs
	query := `
		SELECT id, number, title, body, merged_at
		FROM github_pull_requests
		WHERE repo_id = $1 AND merged = true AND merged_at IS NOT NULL
		ORDER BY merged_at DESC
	`

	rows, err := p.stagingDB.Query(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRs: %w", err)
	}
	defer rows.Close()


	var prs []prInfo
	for rows.Next() {
		var pr prInfo
		var body *string
		if err := rows.Scan(&pr.id, &pr.number, &pr.title, &body, new(interface{})); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		if body != nil {
			pr.body = *body
		}
		prs = append(prs, pr)
	}

	log.Printf("  Found %d merged PRs to analyze", len(prs))

	// Process PRs in batches
	batchSize := 10
	issueToRefs := make(map[int][]types.ExplicitReference)

	for i := 0; i < len(prs); i += batchSize {
		end := min(i+batchSize, len(prs))
		batch := prs[i:end]

		refs, err := p.processPRBatch(ctx, batch, timelinePairs)
		if err != nil {
			log.Printf("  ⚠️  Failed to process batch %d-%d: %v", i, end, err)
			continue
		}

		// Merge refs into issueToRefs map
		for issueNum, refList := range refs {
			issueToRefs[issueNum] = append(issueToRefs[issueNum], refList...)
		}

		log.Printf("  ✓ Processed PRs %d-%d: extracted %d references", i+1, end, len(refs))
	}

	totalRefs := 0
	for _, refs := range issueToRefs {
		totalRefs += len(refs)
	}

	log.Printf("  ✓ Phase 1 complete: extracted %d explicit references", totalRefs)

	return issueToRefs, nil
}

// processPRBatch processes a batch of PRs with LLM
func (p *Phase1Extractor) processPRBatch(ctx context.Context, batch []prInfo, timelinePairs map[string]bool) (map[int][]types.ExplicitReference, error) {
	if !p.llmClient.IsEnabled() {
		return nil, fmt.Errorf("LLM client not enabled")
	}

	// Build prompt
	systemPrompt := `You are a GitHub PR analyzer. Extract issue references from PR titles and bodies.

Output JSON format:
{
  "results": [
    {
      "pr_number": 123,
      "references": [
        {
          "issue_number": 456,
          "reference_type": "fixes",
          "reference_location": "pr_title",
          "extracted_text": "Fixes #456",
          "base_confidence": 0.95,
          "external_repo": false
        }
      ]
    }
  ]
}

Reference types (based on keyword strength):
- "fixes": for "fixes", "closes", "resolves" keywords (confidence: 0.90-0.95)
- "addresses": for "addresses", "address" keywords (confidence: 0.80-0.85)
- "for": for "for issue", "for #" patterns (confidence: 0.80-0.85)
- "mentions": for "mentions", "see", "related", "ref" keywords (confidence: 0.60-0.75)
- "other": for any other reference pattern (confidence: 0.60-0.70)

Reference locations:
- "pr_title": found in PR title
- "pr_description": found in PR body/description
- "pr_comment": found in PR comments (if provided)

External repo detection:
- Set external_repo: true if reference is like "owner/repo#123"
- Otherwise external_repo: false

Rules:
- Extract ONLY the issue number (digits only, no # symbol)
- Look for patterns: #123, issue #123, fixes #123, etc.
- IGNORE references in code blocks or comments
- IGNORE negations: "Don't fix #123"
- IGNORE future plans: "Will fix #123 later"
- Extract base_confidence based on keyword strength
- Check BOTH title and body for each PR`

	userPrompt := "Analyze these PRs:\n\n"
	for _, pr := range batch {
		userPrompt += fmt.Sprintf("PR #%d\n", pr.number)
		userPrompt += fmt.Sprintf("Title: %s\n", pr.title)
		if pr.body != "" {
			// Truncate body to 500 chars for extraction
			body := pr.body
			if len(body) > 500 {
				body = body[:500]
			}
			userPrompt += fmt.Sprintf("Body: %s\n", body)
		}
		userPrompt += "\n"
	}

	// Call LLM
	response, err := p.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse response
	var result struct {
		Results []struct {
			PRNumber   int `json:"pr_number"`
			References []struct {
				IssueNumber       int     `json:"issue_number"`
				ReferenceType     string  `json:"reference_type"`
				ReferenceLocation string  `json:"reference_location"`
				ExtractedText     string  `json:"extracted_text"`
				BaseConfidence    float64 `json:"base_confidence"`
				ExternalRepo      bool    `json:"external_repo"`
			} `json:"references"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert to map: issue_number -> []ExplicitReference
	issueToRefs := make(map[int][]types.ExplicitReference)

	for _, prResult := range result.Results {
		for _, ref := range prResult.References {
			// Skip external repo references
			if ref.ExternalRepo {
				continue
			}

			// Skip if already linked via timeline
			key := fmt.Sprintf("%d-%d", ref.IssueNumber, prResult.PRNumber)
			if timelinePairs[key] {
				continue
			}

			explicitRef := types.ExplicitReference{
				IssueNumber:       ref.IssueNumber,
				PRNumber:          prResult.PRNumber, // Include source PR number
				ReferenceType:     types.ReferenceType(ref.ReferenceType),
				ReferenceLocation: types.ReferenceLocation(ref.ReferenceLocation),
				ExtractedText:     ref.ExtractedText,
				BaseConfidence:    ref.BaseConfidence,
				DetectionMethod:   types.DetectionExplicit,
				ExternalRepo:      false,
			}

			issueToRefs[ref.IssueNumber] = append(issueToRefs[ref.IssueNumber], explicitRef)
		}
	}

	return issueToRefs, nil
}

// ConvertTimelineLinksToExplicitRefs converts timeline links to explicit references
func ConvertTimelineLinksToExplicitRefs(timelineLinks map[int][]types.TimelineLink) map[int][]types.ExplicitReference {
	issueToRefs := make(map[int][]types.ExplicitReference)

	for issueNum, links := range timelineLinks {
		for _, link := range links {
			ref := types.ExplicitReference{
				IssueNumber:       issueNum,
				PRNumber:          link.PRNumber, // Include PR number
				ReferenceType:     link.ReferenceType,
				ReferenceLocation: types.LocTimelineAPI,
				ExtractedText:     link.ExtractedText,
				BaseConfidence:    link.BaseConfidence, // 0.95 for timeline-verified
				DetectionMethod:   types.DetectionGitHubTimeline,
				ExternalRepo:      false,
			}

			issueToRefs[issueNum] = append(issueToRefs[issueNum], ref)
		}
	}

	return issueToRefs
}

// MergeExplicitReferences merges timeline refs and LLM-extracted refs
func MergeExplicitReferences(timelineRefs, llmRefs map[int][]types.ExplicitReference) map[int][]types.ExplicitReference {
	merged := make(map[int][]types.ExplicitReference)

	// Add timeline refs (higher confidence)
	for issueNum, refs := range timelineRefs {
		merged[issueNum] = append(merged[issueNum], refs...)
	}

	// Add LLM refs (avoid duplicates)
	for issueNum, refs := range llmRefs {
		merged[issueNum] = append(merged[issueNum], refs...)
	}

	return merged
}
