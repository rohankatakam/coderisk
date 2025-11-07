package linking

import (
	"github.com/rohankatakam/coderisk/internal/linking/types"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
)

// Phase0Preprocessor handles Phase 0: Pre-processing
// Reference: Issue_Flow.md Phase 0
type Phase0Preprocessor struct {
	stagingDB *database.StagingClient
}

// NewPhase0Preprocessor creates a Phase 0 preprocessor
func NewPhase0Preprocessor(stagingDB *database.StagingClient) *Phase0Preprocessor {
	return &Phase0Preprocessor{
		stagingDB: stagingDB,
	}
}

// RunPreprocessing executes Phase 0: DORA metrics + Timeline link extraction
func (p *Phase0Preprocessor) RunPreprocessing(ctx context.Context, repoID int64, days int) (*types.DORAMetrics, map[int][]types.TimelineLink, error) {
	log.Printf("Phase 0: Pre-processing...")

	// Compute DORA metrics
	log.Printf("  Computing DORA metrics...")
	metrics, err := p.computeDORAMetrics(ctx, repoID, days)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute DORA metrics: %w", err)
	}

	log.Printf("  ✓ DORA metrics computed:")
	log.Printf("    Median lead time: %.2f hours", metrics.MedianLeadTimeHours)
	log.Printf("    Sample size: %d PRs", metrics.SampleSize)
	if metrics.InsufficientHistory {
		log.Printf("    ⚠️  Insufficient history (< 10 PRs) - using fixed ±3 day window")
	}

	// Extract timeline links
	log.Printf("  Extracting GitHub-verified timeline links...")
	timelineLinks, err := p.extractTimelineLinks(ctx, repoID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract timeline links: %w", err)
	}

	metrics.CrossReferenceLinksFound = len(timelineLinks)
	log.Printf("  ✓ Extracted %d GitHub-verified timeline links", len(timelineLinks))

	return metrics, timelineLinks, nil
}

// computeDORAMetrics calculates repository-level DORA metrics
func (p *Phase0Preprocessor) computeDORAMetrics(ctx context.Context, repoID int64, days int) (*types.DORAMetrics, error) {
	return p.stagingDB.ComputeDORAMetrics(ctx, repoID, days)
}

// extractTimelineLinks extracts GitHub-verified cross-reference links from timeline events
func (p *Phase0Preprocessor) extractTimelineLinks(ctx context.Context, repoID int64) (map[int][]types.TimelineLink, error) {
	// Query timeline events for cross-references
	query := `
		SELECT
			i.number as issue_number,
			t.source_number as pr_number,
			t.source_title as pr_title,
			t.source_body as pr_body
		FROM github_issue_timeline t
		JOIN github_issues i ON t.issue_id = i.id
		WHERE i.repo_id = $1
		  AND t.event_type = 'cross-referenced'
		  AND t.source_type = 'pull_request'
		  AND t.source_number IS NOT NULL
	`

	rows, err := p.stagingDB.Query(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline events: %w", err)
	}
	defer rows.Close()

	timelineMap := make(map[int][]types.TimelineLink)

	for rows.Next() {
		var issueNumber, prNumber int
		var prTitle, prBody *string

		if err := rows.Scan(&issueNumber, &prNumber, &prTitle, &prBody); err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}

		// Extract reference type and text from PR body
		refType, extractedText := p.extractReferenceType(prTitle, prBody, issueNumber)

		link := types.TimelineLink{
			IssueNumber:    issueNumber,
			PRNumber:       prNumber,
			ReferenceType:  refType,
			ExtractedText:  extractedText,
			BaseConfidence: 0.95, // GitHub-verified links have high confidence
		}

		timelineMap[issueNumber] = append(timelineMap[issueNumber], link)
	}

	return timelineMap, rows.Err()
}

// extractReferenceType determines the reference type from PR content
func (p *Phase0Preprocessor) extractReferenceType(prTitle, prBody *string, issueNumber int) (types.ReferenceType, string) {
	// Combine title and body for analysis
	text := ""
	if prTitle != nil {
		text += *prTitle + " "
	}
	if prBody != nil {
		text += *prBody
	}

	// Truncate to first 500 chars for reference extraction
	if len(text) > 500 {
		text = text[:500]
	}

	issueRef := fmt.Sprintf("#%d", issueNumber)

	// Look for common patterns around the issue reference
	patterns := []struct {
		keywords []string
		refType  types.ReferenceType
	}{
		{[]string{"fixes", "fix", "fixed", "close", "closes", "closed", "resolve", "resolves", "resolved"}, types.RefFixes},
		{[]string{"addresses", "address"}, types.RefAddresses},
		{[]string{"for issue", "for #"}, types.RefFor},
		{[]string{"mentioned", "mentions", "see", "ref", "related"}, types.RefMentions},
	}

	for _, pattern := range patterns {
		for _, keyword := range pattern.keywords {
			if containsPattern(text, keyword, issueRef) {
				// Extract a snippet around the reference
				extractedText := extractSnippet(text, issueRef, 50)
				return pattern.refType, extractedText
			}
		}
	}

	// Default to mentions if no specific pattern found
	return types.RefMentions, extractSnippet(text, issueRef, 50)
}

// containsPattern checks if text contains keyword near issueRef
func containsPattern(text, keyword, issueRef string) bool {
	// Simple case-insensitive substring search
	// In production, use regex for more robust matching
	textLower := toLower(text)
	keywordLower := toLower(keyword)

	keywordIdx := indexOf(textLower, keywordLower)
	if keywordIdx == -1 {
		return false
	}

	refIdx := indexOf(text, issueRef)
	if refIdx == -1 {
		return false
	}

	// Check if keyword is within 100 chars of reference
	return abs(keywordIdx-refIdx) < 100
}

// extractSnippet extracts text around a reference
func extractSnippet(text, ref string, contextChars int) string {
	idx := indexOf(text, ref)
	if idx == -1 {
		// Reference not found, return beginning
		if len(text) > contextChars*2 {
			return text[:contextChars*2]
		}
		return text
	}

	start := max(0, idx-contextChars)
	end := min(len(text), idx+len(ref)+contextChars)

	snippet := text[start:end]

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// Helper functions
func toLower(s string) string {
	// Simple lowercase conversion
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetTimelineLinksForIssue retrieves timeline links for a specific issue
func GetTimelineLinksForIssue(timelineMap map[int][]types.TimelineLink, issueNumber int) []types.TimelineLink {
	return timelineMap[issueNumber]
}

// HasTimelineLinks checks if an issue has any timeline links
func HasTimelineLinks(timelineMap map[int][]types.TimelineLink, issueNumber int) bool {
	links, exists := timelineMap[issueNumber]
	return exists && len(links) > 0
}


// truncateComment truncates comment to 1000 first + 500 last if > 2000 chars
func TruncateComment(comment string) (string, bool) {
	if len(comment) <= 2000 {
		return comment, false
	}

	truncated := comment[:1000] + "...[truncated]..." + comment[len(comment)-500:]
	return truncated, true
}

// parseLabels extracts label names from JSONB
func parseLabels(labelsJSON []byte) []string {
	if len(labelsJSON) == 0 {
		return nil
	}

	var labels []map[string]interface{}
	if err := json.Unmarshal(labelsJSON, &labels); err != nil {
		return nil
	}

	var labelNames []string
	for _, label := range labels {
		if name, ok := label["name"].(string); ok {
			labelNames = append(labelNames, name)
		}
	}

	return labelNames
}
