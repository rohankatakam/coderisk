package linking

import (
	"github.com/rohankatakam/coderisk/internal/linking/types"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// Phase2PathA handles Path A: Explicit Link Validation & Enhancement
// Reference: Issue_Flow.md Phase 2 Path A
type Phase2PathA struct {
	stagingDB *database.StagingClient
	llmClient *llm.Client
}

// NewPhase2PathA creates a Phase 2 Path A processor
func NewPhase2PathA(stagingDB *database.StagingClient, llmClient *llm.Client) *Phase2PathA {
	return &Phase2PathA{
		stagingDB: stagingDB,
		llmClient: llmClient,
	}
}

// ProcessExplicitLink processes an issue with explicit PR references
func (p *Phase2PathA) ProcessExplicitLink(ctx context.Context, repoID int64, issue *types.IssueData, pr *types.PRData, explicitRef types.ExplicitReference) (*types.LinkOutput, error) {
	startTime := time.Now()

	// Step 2.1.1: Bidirectional Reference Detection
	bidirResult, err := p.detectBidirectionalReference(ctx, issue, pr)
	if err != nil {
		return nil, fmt.Errorf("bidirectional detection failed: %w", err)
	}

	// Step 2.1.2: Semantic Similarity Analysis
	semanticResult, err := p.analyzeSemanticSimilarity(ctx, issue, pr)
	if err != nil {
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	// Step 2.1.3: Temporal Correlation Analysis
	temporalResult, err := p.analyzeTemporalCorrelation(issue, pr)
	if err != nil {
		return nil, fmt.Errorf("temporal analysis failed: %w", err)
	}

	// Step 2.1.4: Final Confidence Calculation
	link := p.calculateFinalConfidence(issue, pr, explicitRef, bidirResult, semanticResult, temporalResult)

	// Track timing
	link.Metadata.PhaseTimings.Phase2MS = time.Since(startTime).Milliseconds()
	link.Metadata.LinkCreatedAt = time.Now()

	return link, nil
}

// BidirectionalResult contains bidirectional reference analysis
type BidirectionalResult struct {
	IsBidirectional       bool
	BidirectionalLocations []string
	BidirectionalBoost    float64
	NegativeSignalPenalty float64
	Rationale             string
}

// detectBidirectionalReference checks if issue mentions the PR
func (p *Phase2PathA) detectBidirectionalReference(ctx context.Context, issue *types.IssueData, pr *types.PRData) (*BidirectionalResult, error) {
	if !p.llmClient.IsEnabled() {
		return &BidirectionalResult{}, nil
	}

	systemPrompt := `You are analyzing whether a GitHub Issue mentions a specific Pull Request (reverse direction).

Output JSON format:
{
  "is_bidirectional": true/false,
  "bidirectional_locations": ["issue_description", "issue_comment", "closing_comment"],
  "bidirectional_boost": 0.10,
  "negative_signal_detected": false,
  "negative_signal_penalty": 0.00,
  "rationale": "Detailed explanation..."
}

Rules:
1. Check if issue body or comments explicitly mention the PR number
2. Look for patterns: "#[PR_NUMBER]", "PR #[PR_NUMBER]", "pull request #[PR_NUMBER]", "fixed in #[PR_NUMBER]"
3. Determine mention location (issue_description, issue_comment, closing_comment)
4. Check who mentioned it (issue author, maintainer, bot)
5. **CRITICAL: Detect negative signals**:
   - "not fixed", "still broken", "doesn't work" → negative_signal_penalty = -0.15
   - "partially fixed", "some issues remain" → negative_signal_penalty = -0.08
   - Positive/neutral mention → negative_signal_penalty = 0.00
6. Bidirectional boost calculation:
   - If bidirectional: +0.10
   - If mentioned in closing comment by maintainer: +0.05 additional
   - If multiple mentions: +0.03 per additional (cap at +0.05 total)
   - Bot auto-closing: +0.05 (lower boost)
7. Generate detailed rationale explaining bidirectional reference and any negative signals`

	userPrompt := fmt.Sprintf(`Analyze if this Issue mentions PR #%d:

ISSUE #%d: %s
Created: %s
Closed: %s

Body:
%s

Comments (%d total):
`, pr.PRNumber, issue.IssueNumber, issue.Title, issue.CreatedAt.Format(time.RFC3339), formatTimePtr(issue.ClosedAt), truncateText(issue.Body, 300), len(issue.Comments))

	for i, comment := range issue.Comments {
		if i >= 5 {
			userPrompt += fmt.Sprintf("\n[...%d more comments]", len(issue.Comments)-5)
			break
		}
		userPrompt += fmt.Sprintf("\n[%s by %s]: %s", comment.CreatedAt.Format(time.RFC3339), comment.Author, truncateText(comment.Body, 200))
	}

	userPrompt += fmt.Sprintf("\n\nPR #%d being analyzed", pr.PRNumber)

	response, err := p.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	var result struct {
		IsBidirectional        bool     `json:"is_bidirectional"`
		BidirectionalLocations []string `json:"bidirectional_locations"`
		BidirectionalBoost     float64  `json:"bidirectional_boost"`
		NegativeSignalDetected bool     `json:"negative_signal_detected"`
		NegativeSignalPenalty  float64  `json:"negative_signal_penalty"`
		Rationale              string   `json:"rationale"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &BidirectionalResult{
		IsBidirectional:       result.IsBidirectional,
		BidirectionalLocations: result.BidirectionalLocations,
		BidirectionalBoost:    result.BidirectionalBoost,
		NegativeSignalPenalty: result.NegativeSignalPenalty,
		Rationale:             result.Rationale,
	}, nil
}

// analyzeSemanticSimilarity performs semantic similarity analysis
func (p *Phase2PathA) analyzeSemanticSimilarity(ctx context.Context, issue *types.IssueData, pr *types.PRData) (*types.SemanticScores, error) {
	if !p.llmClient.IsEnabled() {
		return &types.SemanticScores{}, nil
	}

	systemPrompt := `You are analyzing semantic similarity between a GitHub Issue and Pull Request.

Output JSON format:
{
  "title_score": 0.85,
  "body_score": 0.72,
  "comment_score": 0.68,
  "cross_content_score": 0.90,
  "title_keywords": ["auth", "login", "bug"],
  "body_keywords": ["authentication", "session", "error"],
  "comment_keywords": ["fixed", "merged"],
  "title_rationale": "High similarity: both mention authentication bug",
  "body_rationale": "Medium similarity: issue describes problem, PR describes solution",
  "cross_content_rationale": "Issue closing comment strongly matches PR title"
}

Scoring rules (0.0-1.0):
- Title-to-Title: Compare issue title to PR title
- Body-to-Body: Compare issue description to PR description
- Comment-to-PR: Compare issue comments (especially closing) to PR title/body
- Cross-Content: Most important - issue closing comment vs PR title/description

Scores:
- 0.70-1.0: High similarity (clear semantic match)
- 0.50-0.69: Medium similarity (related concepts)
- 0.30-0.49: Low similarity (weak connection)
- 0.0-0.29: No similarity

Extract keyword overlaps and provide rationales for each score.
Handle inverse matches: issue describes problem, PR describes solution.`

	issueClosingComment := ""
	if len(issue.Comments) > 0 {
		issueClosingComment = issue.Comments[len(issue.Comments)-1].Body
	}

	userPrompt := fmt.Sprintf(`Analyze semantic similarity:

ISSUE #%d:
Title: %s
Body: %s
Closing Comment: %s

PR #%d:
Title: %s
Body: %s
`,
		issue.IssueNumber, issue.Title, truncateText(issue.Body, 500),
		truncateText(issueClosingComment, 300),
		pr.PRNumber, pr.Title, truncateText(pr.Body, 500))

	response, err := p.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	var semanticScores types.SemanticScores
	if err := json.Unmarshal([]byte(response), &semanticScores); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &semanticScores, nil
}

// analyzeTemporalCorrelation analyzes temporal relationship
func (p *Phase2PathA) analyzeTemporalCorrelation(issue *types.IssueData, pr *types.PRData) (*types.TemporalAnalysis, error) {
	if issue.ClosedAt == nil || pr.MergedAt == nil {
		return nil, fmt.Errorf("missing timestamps")
	}

	analysis := &types.TemporalAnalysis{
		IssueClosedAt: *issue.ClosedAt,
		PRMergedAt:    *pr.MergedAt,
	}

	// Calculate temporal delta
	delta := issue.ClosedAt.Sub(*pr.MergedAt)
	analysis.TemporalDeltaSeconds = int64(math.Abs(delta.Seconds()))

	// Determine pattern and direction
	if delta >= 0 {
		// Issue closed after PR merged (normal flow)
		analysis.TemporalDirection = "normal"

		if delta < 5*time.Minute {
			analysis.TemporalPattern = types.PatternSimultaneous
		} else if delta < 3*24*time.Hour {
			analysis.TemporalPattern = types.PatternNormal
		} else {
			analysis.TemporalPattern = types.PatternDelayed
		}
	} else {
		// PR merged before issue opened (reverse flow)
		analysis.TemporalDirection = "reverse"
		analysis.TemporalPattern = types.PatternReverse
	}

	return analysis, nil
}

// calculateFinalConfidence computes final confidence score
func (p *Phase2PathA) calculateFinalConfidence(
	issue *types.IssueData,
	pr *types.PRData,
	explicitRef types.ExplicitReference,
	bidirResult *BidirectionalResult,
	semanticScores *types.SemanticScores,
	temporalAnalysis *types.TemporalAnalysis,
) *types.LinkOutput {
	// Calculate boosts
	semanticBoost := p.calculateSemanticBoost(semanticScores)
	temporalBoost := p.calculateTemporalBoost(temporalAnalysis)

	// Build confidence breakdown
	breakdown := types.ConfidenceBreakdown{
		BaseConfidence:       explicitRef.BaseConfidence,
		BidirectionalBoost:   bidirResult.BidirectionalBoost,
		SemanticBoost:        semanticBoost,
		TemporalBoost:        temporalBoost,
		NegativeSignalPenalty: bidirResult.NegativeSignalPenalty,
	}

	// Calculate final confidence
	finalConfidence := breakdown.BaseConfidence +
		breakdown.BidirectionalBoost +
		breakdown.SemanticBoost +
		breakdown.TemporalBoost +
		breakdown.NegativeSignalPenalty

	// Cap at 0.98
	if finalConfidence > 0.98 {
		finalConfidence = 0.98
	}

	// Determine link quality
	linkQuality := p.determineLinkQuality(finalConfidence)

	// Build evidence sources
	evidenceSources := []string{string(explicitRef.DetectionMethod)}
	if bidirResult.IsBidirectional {
		evidenceSources = append(evidenceSources, "bidirectional")
	}
	if semanticScores.TitleScore >= 0.5 {
		evidenceSources = append(evidenceSources, "semantic_title")
	}
	if semanticScores.BodyScore >= 0.5 {
		evidenceSources = append(evidenceSources, "semantic_body")
	}
	if semanticScores.CrossContentScore >= 0.5 {
		evidenceSources = append(evidenceSources, "semantic_cross")
	}
	if temporalBoost > 0 {
		evidenceSources = append(evidenceSources, "temporal")
	}
	if bidirResult.NegativeSignalPenalty < 0 {
		evidenceSources = append(evidenceSources, "negative_signal")
	}

	// Build comprehensive rationale
	rationale := p.buildComprehensiveRationale(explicitRef, bidirResult, semanticScores, temporalAnalysis, finalConfidence)

	// Determine flags
	flags := types.LinkFlags{
		NeedsManualReview: finalConfidence < 0.50,
		SharedPR:          false, // Will be set later if needed
		ReverseTemporal:   temporalAnalysis.TemporalDirection == "reverse",
	}

	// Determine detection method
	detectionMethod := explicitRef.DetectionMethod
	if bidirResult.IsBidirectional {
		detectionMethod = types.DetectionExplicitBidir
	}

	return &types.LinkOutput{
		IssueNumber:           issue.IssueNumber,
		PRNumber:              pr.PRNumber,
		DetectionMethod:       detectionMethod,
		FinalConfidence:       finalConfidence,
		LinkQuality:           linkQuality,
		ConfidenceBreakdown:   breakdown,
		EvidenceSources:       evidenceSources,
		ComprehensiveRationale: rationale,
		SemanticAnalysis:      semanticScores,
		TemporalAnalysis:      temporalAnalysis,
		Flags:                 flags,
		Metadata:              types.LinkMetadata{},
	}
}

// calculateSemanticBoost determines semantic confidence boost
func (p *Phase2PathA) calculateSemanticBoost(scores *types.SemanticScores) float64 {
	maxScore := math.Max(scores.TitleScore, math.Max(scores.BodyScore, scores.CrossContentScore))

	if maxScore >= 0.70 {
		return 0.15
	} else if maxScore >= 0.50 {
		return 0.10
	} else if maxScore >= 0.30 {
		return 0.05
	}
	return 0.00
}

// calculateTemporalBoost determines temporal confidence boost
func (p *Phase2PathA) calculateTemporalBoost(analysis *types.TemporalAnalysis) float64 {
	deltaMinutes := float64(analysis.TemporalDeltaSeconds) / 60.0

	if deltaMinutes < 5 {
		return 0.15
	} else if deltaMinutes < 60 {
		return 0.12
	} else if deltaMinutes < 24*60 {
		return 0.08
	} else if deltaMinutes < 3*24*60 {
		return 0.05
	}
	return 0.00
}

// determineLinkQuality assigns quality tier
func (p *Phase2PathA) determineLinkQuality(confidence float64) types.LinkQuality {
	if confidence >= 0.85 {
		return types.QualityHigh
	} else if confidence >= 0.70 {
		return types.QualityMedium
	}
	return types.QualityLow
}

// buildComprehensiveRationale generates detailed rationale
func (p *Phase2PathA) buildComprehensiveRationale(
	explicitRef types.ExplicitReference,
	bidirResult *BidirectionalResult,
	semanticScores *types.SemanticScores,
	temporalAnalysis *types.TemporalAnalysis,
	finalConfidence float64,
) string {
	rationale := fmt.Sprintf("EXPLICIT link detected with final confidence %.2f.\n\n", finalConfidence)

	// Explicit reference
	rationale += fmt.Sprintf("Explicit Reference:\n")
	rationale += fmt.Sprintf("- Detection method: %s\n", explicitRef.DetectionMethod)
	rationale += fmt.Sprintf("- Reference type: %s\n", explicitRef.ReferenceType)
	rationale += fmt.Sprintf("- Location: %s\n", explicitRef.ReferenceLocation)
	rationale += fmt.Sprintf("- Extracted text: \"%s\"\n", explicitRef.ExtractedText)
	rationale += fmt.Sprintf("- Base confidence: %.2f\n\n", explicitRef.BaseConfidence)

	// Bidirectional
	if bidirResult.IsBidirectional {
		rationale += fmt.Sprintf("Bidirectional Confirmation:\n")
		rationale += fmt.Sprintf("- %s\n", bidirResult.Rationale)
		rationale += fmt.Sprintf("- Bidirectional boost: +%.2f\n\n", bidirResult.BidirectionalBoost)
	}

	// Semantic
	rationale += fmt.Sprintf("Semantic Analysis:\n")
	rationale += fmt.Sprintf("- Title similarity: %.2f - %s\n", semanticScores.TitleScore, semanticScores.TitleRationale)
	rationale += fmt.Sprintf("- Body similarity: %.2f - %s\n", semanticScores.BodyScore, semanticScores.BodyRationale)
	rationale += fmt.Sprintf("- Cross-content: %.2f - %s\n\n", semanticScores.CrossContentScore, semanticScores.CrossContentRationale)

	// Temporal
	rationale += fmt.Sprintf("Temporal Analysis:\n")
	rationale += fmt.Sprintf("- Pattern: %s (%s)\n", temporalAnalysis.TemporalPattern, temporalAnalysis.TemporalDirection)
	rationale += fmt.Sprintf("- Delta: %d seconds\n\n", temporalAnalysis.TemporalDeltaSeconds)

	// Negative signals
	if bidirResult.NegativeSignalPenalty < 0 {
		rationale += fmt.Sprintf("⚠️  Negative Signal Detected: penalty %.2f\n", bidirResult.NegativeSignalPenalty)
	}

	return rationale
}

// Helper functions
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format(time.RFC3339)
}
