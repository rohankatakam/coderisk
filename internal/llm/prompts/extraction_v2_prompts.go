package prompts

import "fmt"

// IssueExtractionV2System is the system prompt for enhanced issue reference extraction
const IssueExtractionV2System = `You are a GitHub issue analyzer extracting ALL references with high accuracy.

Your task is to analyze issue data and identify ALL references to:
1. Pull Requests (PRs)
2. Commits (by SHA)
3. Other issues

ANALYSIS SOURCES:
- Issue body text
- Issue comments (especially from maintainers)
- Timeline events
- Temporal context (recent PRs/commits)
- Semantic hints (similar PR titles)

OUTPUT JSON FORMAT:
{
  "references": [
    {
      "type": "commit" | "pr" | "issue",
      "id": "sha_or_number",
      "action": "fixes" | "mentions" | "duplicate",
      "confidence": 0.0-1.0,
      "evidence": ["explicit", "temporal", "semantic", "comment", "timeline"],
      "temporal_delta_minutes": <number> (optional),
      "semantic_score": 0.0-1.0 (optional),
      "source": "body" | "comment" | "timeline",
      "commenter_role": "owner" | "collaborator" | "bot" | "contributor" (if from comment)
    }
  ]
}

CONFIDENCE SCORING RULES:
1. Explicit References (0.90-0.95):
   - "Fixes #123", "Closes #456", "Resolves #789"
   - "Fixed in PR #120", "Resolved by commit abc123"
   - Keywords: fixes, closes, resolves, fixed, closed, resolved

2. Maintainer Comments (0.80-0.90):
   - Owner/Collaborator says "Fixed in PR #123"
   - High confidence due to authority
   - Apply +0.10 boost for owner, +0.08 for collaborator

3. Bot Comments (0.75-0.80):
   - GitHub Actions, bots posting "Merged PR #123"
   - Automated messages are reliable
   - Apply +0.05 boost for bot comments

4. Temporal Correlation (0.70-0.80):
   - PR merged within 5 minutes of issue close: +0.15
   - PR merged within 1 hour of issue close: +0.10
   - PR merged within 24 hours of issue close: +0.05
   - Base confidence: 0.50, then apply temporal boost

5. Semantic Match (0.60-0.75):
   - PR title contains same keywords as issue title
   - High keyword overlap (>70%): 0.75
   - Medium keyword overlap (50-70%): 0.65
   - Low keyword overlap (30-50%): 0.60

6. Multiple Evidence (combine scores):
   - Explicit + Temporal: 0.95 (cap)
   - Explicit + Semantic: 0.95 (cap)
   - Temporal + Semantic: 0.80-0.90
   - Comment + Temporal: 0.90-0.95
   - Multiple sources: +0.03 per additional (max 0.98)

EVIDENCE TYPES:
- "explicit": Direct keyword match ("Fixes #123")
- "temporal": Timestamp correlation
- "semantic": Keyword/title similarity
- "comment": Mentioned in comments
- "timeline": GitHub timeline event
- "bidirectional": Both issue and PR mention each other

ACTION CLASSIFICATION:
- "fixes": Use for closes, resolves, fixes, fixed, closed, resolved
- "mentions": Use for "see", "related to", "duplicate of" (without fixes)
- "duplicate": Use for "duplicate of #123"

SPECIAL CASES:
- "Not fixed yet" → Do NOT create reference (negative evidence)
- "Duplicate of #456" → Create "duplicate" action to issue #456
- Multiple conflicting comments → Use latest from highest authority (owner > collaborator > bot > contributor)
- Generic titles ("Fix bug") → Lower confidence for semantic match
- No evidence found → Return empty references array`

// IssueExtractionV2User generates the user prompt with context
func IssueExtractionV2User(issueNumber int, title, body string, closedAt *string,
	comments []CommentInfo, recentPRs []PRHint, similarPRs []PRHint) string {

	prompt := "ANALYZE THIS ISSUE:\n\n"
	prompt += "═══════════════════════════════════════════════════════════\n"
	prompt += fmt.Sprintf("Issue #%d: %s\n", issueNumber, title)

	if closedAt != nil {
		prompt += fmt.Sprintf("Closed at: %s\n", *closedAt)
	}
	prompt += "State: closed\n"
	prompt += "═══════════════════════════════════════════════════════════\n\n"

	// Issue body
	if body != "" {
		prompt += "ISSUE BODY:\n"
		prompt += truncate(body, 1000)
		prompt += "\n\n"
	}

	// Comments
	if len(comments) > 0 {
		prompt += "COMMENTS:\n"
		for i, comment := range comments {
			if i >= 10 {
				break // Limit to 10 comments
			}
			prompt += fmt.Sprintf("  [%s - %s] %s\n",
				comment.Author, comment.Role, truncate(comment.Body, 300))
		}
		prompt += "\n"
	}

	// Temporal hints
	if len(recentPRs) > 0 {
		prompt += "TEMPORAL HINTS (PRs merged near issue close time):\n"
		for i, pr := range recentPRs {
			if i >= 5 {
				break
			}
			prompt += fmt.Sprintf("  PR #%d: %s (merged %d min %s issue close)\n",
				pr.Number, truncate(pr.Title, 80), pr.DeltaMinutes, pr.Direction)
		}
		prompt += "\n"
	}

	// Semantic hints
	if len(similarPRs) > 0 {
		prompt += "SEMANTIC HINTS (PRs with similar keywords):\n"
		for i, pr := range similarPRs {
			if i >= 5 {
				break
			}
			prompt += fmt.Sprintf("  PR #%d: %s (similarity: %.0f%%)\n",
				pr.Number, truncate(pr.Title, 80), pr.Similarity*100)
		}
		prompt += "\n"
	}

	prompt += "EXTRACT ALL REFERENCES with confidence scores and evidence.\n"
	return prompt
}

// CommentInfo represents a comment for LLM analysis
type CommentInfo struct {
	Author string
	Role   string // "owner", "collaborator", "bot", "contributor"
	Body   string
	CreatedAt string
}

// PRHint represents a PR hint for temporal/semantic matching
type PRHint struct {
	Number       int
	Title        string
	DeltaMinutes int     // For temporal hints
	Direction    string  // "before" or "after"
	Similarity   float64 // For semantic hints (0.0-1.0)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}
