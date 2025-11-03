package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/llm/prompts"
)

// CommentAnalyzer extracts references from issue comments with enhanced context
type CommentAnalyzer struct {
	llmClient *Client
}

// NewCommentAnalyzer creates a new comment analyzer
func NewCommentAnalyzer(llmClient *Client) *CommentAnalyzer {
	return &CommentAnalyzer{
		llmClient: llmClient,
	}
}

// Comment represents an issue comment
type Comment struct {
	ID        int64     `json:"id"`
	IssueID   int64     `json:"issue_id"`
	Author    string    `json:"author"`
	AuthorType string   `json:"author_type"` // "User", "Bot"
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	RawData   json.RawMessage `json:"raw_data,omitempty"`
}

// Reference represents an extracted reference from LLM
type Reference struct {
	Type              string   `json:"type"`               // "commit", "pr", "issue"
	ID                string   `json:"id"`                 // SHA or number
	Action            string   `json:"action"`             // "fixes", "mentions", "duplicate"
	Confidence        float64  `json:"confidence"`         // 0.0-1.0
	Evidence          []string `json:"evidence"`           // ["explicit", "temporal", "semantic", "comment"]
	TemporalDelta     *int     `json:"temporal_delta_minutes,omitempty"`
	SemanticScore     *float64 `json:"semantic_score,omitempty"`
	Source            string   `json:"source"`             // "body", "comment", "timeline"
	CommenterRole     string   `json:"commenter_role,omitempty"` // "owner", "collaborator", "bot", "contributor"
}

// ExtractCommentReferences analyzes issue with comments and context
func (ca *CommentAnalyzer) ExtractCommentReferences(
	ctx context.Context,
	issueNumber int,
	issueTitle string,
	issueBody string,
	closedAt *time.Time,
	comments []Comment,
	repoOwner string,
	collaborators []string,
) ([]Reference, error) {

	// Build comment info for LLM
	commentInfos := make([]prompts.CommentInfo, 0, len(comments))
	for _, comment := range comments {
		role := determineCommenterRole(comment.Author, repoOwner, collaborators, comment.AuthorType)
		commentInfos = append(commentInfos, prompts.CommentInfo{
			Author:    comment.Author,
			Role:      role,
			Body:      comment.Body,
			CreatedAt: comment.CreatedAt.Format(time.RFC3339),
		})
	}

	// Build temporal hints (PRs merged near issue close time)
	// TODO: Query database for recent PRs - for now, pass empty
	recentPRs := []prompts.PRHint{}

	// Build semantic hints (PRs with similar titles)
	// TODO: Query database for similar PRs - for now, pass empty
	similarPRs := []prompts.PRHint{}

	// Generate prompt
	closedAtStr := ""
	if closedAt != nil {
		closedAtStr = closedAt.Format(time.RFC3339)
	}

	userPrompt := prompts.IssueExtractionV2User(
		issueNumber,
		issueTitle,
		issueBody,
		func() *string {
			if closedAtStr == "" {
				return nil
			}
			return &closedAtStr
		}(),
		commentInfos,
		recentPRs,
		similarPRs,
	)

	// Call LLM
	response, err := ca.llmClient.CompleteJSON(
		ctx,
		prompts.IssueExtractionV2System,
		userPrompt,
	)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	var result struct {
		References []Reference `json:"references"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("⚠️  Failed to parse LLM response: %v", err)
		log.Printf("Response: %s", response)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Apply commenter role boosts
	for i := range result.References {
		if result.References[i].Source == "comment" {
			applyCommenterBoost(&result.References[i])
		}
	}

	return result.References, nil
}

// determineCommenterRole classifies the commenter
func determineCommenterRole(author, repoOwner string, collaborators []string, authorType string) string {
	if authorType == "Bot" {
		return "bot"
	}

	if author == repoOwner {
		return "owner"
	}

	for _, collab := range collaborators {
		if author == collab {
			return "collaborator"
		}
	}

	return "contributor"
}

// applyCommenterBoost applies confidence boost based on commenter role
func applyCommenterBoost(ref *Reference) {
	switch ref.CommenterRole {
	case "owner":
		ref.Confidence += 0.10
		if !containsEvidence(ref.Evidence, "owner_comment") {
			ref.Evidence = append(ref.Evidence, "owner_comment")
		}
	case "collaborator":
		ref.Confidence += 0.08
		if !containsEvidence(ref.Evidence, "collaborator_comment") {
			ref.Evidence = append(ref.Evidence, "collaborator_comment")
		}
	case "bot":
		ref.Confidence += 0.05
		if !containsEvidence(ref.Evidence, "bot_comment") {
			ref.Evidence = append(ref.Evidence, "bot_comment")
		}
	case "contributor":
		ref.Confidence += 0.03
		if !containsEvidence(ref.Evidence, "contributor_comment") {
			ref.Evidence = append(ref.Evidence, "contributor_comment")
		}
	}

	// Cap confidence at 0.98
	if ref.Confidence > 0.98 {
		ref.Confidence = 0.98
	}
}

// containsEvidence checks if evidence array contains a specific evidence type
func containsEvidence(evidence []string, target string) bool {
	for _, e := range evidence {
		if e == target {
			return true
		}
	}
	return false
}

// ApplyTemporalBoost applies temporal correlation boost
func ApplyTemporalBoost(ref *Reference, issueClosedAt, prMergedAt time.Time) {
	if issueClosedAt.IsZero() || prMergedAt.IsZero() {
		return
	}

	delta := issueClosedAt.Sub(prMergedAt)
	if delta < 0 {
		delta = -delta // Absolute value
	}

	deltaMinutes := int(delta.Minutes())
	ref.TemporalDelta = &deltaMinutes

	if delta < 5*time.Minute {
		ref.Confidence += 0.15
		ref.Evidence = append(ref.Evidence, "temporal_match_5min")
	} else if delta < 1*time.Hour {
		ref.Confidence += 0.10
		ref.Evidence = append(ref.Evidence, "temporal_match_1hr")
	} else if delta < 24*time.Hour {
		ref.Confidence += 0.05
		ref.Evidence = append(ref.Evidence, "temporal_match_24hr")
	}

	// Cap at 0.98
	if ref.Confidence > 0.98 {
		ref.Confidence = 0.98
	}
}

// ApplySemanticBoost applies semantic similarity boost
func ApplySemanticBoost(ref *Reference, similarity float64) {
	ref.SemanticScore = &similarity

	if similarity >= 0.70 {
		ref.Confidence += 0.15
		ref.Evidence = append(ref.Evidence, "semantic_high")
	} else if similarity >= 0.50 {
		ref.Confidence += 0.10
		ref.Evidence = append(ref.Evidence, "semantic_medium")
	} else if similarity >= 0.30 {
		ref.Confidence += 0.05
		ref.Evidence = append(ref.Evidence, "semantic_low")
	}

	// Cap at 0.98
	if ref.Confidence > 0.98 {
		ref.Confidence = 0.98
	}
}

// CombineEvidence combines multiple evidence sources and adjusts confidence
func CombineEvidence(ref *Reference) {
	// Count unique evidence types
	evidenceTypes := make(map[string]bool)
	for _, e := range ref.Evidence {
		evidenceTypes[e] = true
	}

	// Apply multi-evidence boost (+0.03 per additional source beyond first)
	if len(evidenceTypes) > 1 {
		boost := float64(len(evidenceTypes)-1) * 0.03
		ref.Confidence += boost
	}

	// Cap at 0.98 (never 100% certain)
	if ref.Confidence > 0.98 {
		ref.Confidence = 0.98
	}
}
