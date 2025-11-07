package linking

import (
	"github.com/rohankatakam/coderisk/internal/linking/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// Phase2PathB handles Path B: Deep Link Finder (No Explicit References)
// Reference: Issue_Flow.md Phase 2 Path B
type Phase2PathB struct {
	stagingDB   *database.StagingClient
	llmClient   *llm.Client
	doraMetrics *types.DORAMetrics
}

// NewPhase2PathB creates a Phase 2 Path B processor
func NewPhase2PathB(stagingDB *database.StagingClient, llmClient *llm.Client, doraMetrics *types.DORAMetrics) *Phase2PathB {
	return &Phase2PathB{
		stagingDB:   stagingDB,
		llmClient:   llmClient,
		doraMetrics: doraMetrics,
	}
}

// ProcessDeepLink processes an issue with no explicit references
func (p *Phase2PathB) ProcessDeepLink(ctx context.Context, repoID int64, issue *types.IssueData) ([]types.LinkOutput, *types.NoLinkOutput, error) {
	startTime := time.Now()

	// Step 2.2.1: Bug Classification
	log.Printf("  Classifying issue #%d closure type...", issue.IssueNumber)
	classification, err := p.classifyBugClosure(ctx, issue)
	if err != nil {
		return nil, nil, fmt.Errorf("bug classification failed: %w", err)
	}

	log.Printf("    Classification: %s (confidence: %.2f)", classification.ClosureClassification, classification.ClassificationConfidence)

	// Check if we should proceed to deep finder
	if !p.shouldProceedToDeepFinder(classification) {
		// Return no-link output
		noLink := &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkReason(classification.ClosureClassification),
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
		log.Printf("    ✓ No PR expected for issue #%d (%s)", issue.IssueNumber, classification.ClosureClassification)
		return nil, noLink, nil
	}

	// Step 2.2.2: Deep Link Finder (Temporal-Semantic Search)
	log.Printf("  Running deep link finder for issue #%d...", issue.IssueNumber)
	candidates, err := p.findCandidatePRs(ctx, repoID, issue)
	if err != nil {
		return nil, nil, fmt.Errorf("candidate PR search failed: %w", err)
	}

	if len(candidates) == 0 {
		noLink := &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkNoTemporalMatches,
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			CandidatesEvaluated:      0,
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
		log.Printf("    ✗ No temporal matches found for issue #%d", issue.IssueNumber)
		return nil, noLink, nil
	}

	log.Printf("    Found %d candidate PRs, running semantic ranking...", len(candidates))

	// Rank candidates semantically
	rankedCandidates, err := p.rankCandidatesSemantically(ctx, issue, candidates)
	if err != nil {
		return nil, nil, fmt.Errorf("semantic ranking failed: %w", err)
	}

	// Step 2.2.2c: Link Creation from Deep Finder
	links, noLink := p.createLinksFromCandidates(issue, rankedCandidates, classification)

	// Track timing
	elapsed := time.Since(startTime).Milliseconds()
	for i := range links {
		links[i].Metadata.PhaseTimings.Phase2MS = elapsed
		links[i].Metadata.LinkCreatedAt = time.Now()
	}

	if len(links) > 0 {
		log.Printf("    ✓ Created %d deep links for issue #%d", len(links), issue.IssueNumber)
	} else {
		log.Printf("    ✗ No links met threshold for issue #%d", issue.IssueNumber)
	}

	return links, noLink, nil
}

// classifyBugClosure classifies the issue closure type
func (p *Phase2PathB) classifyBugClosure(ctx context.Context, issue *types.IssueData) (*types.BugClassificationResult, error) {
	if !p.llmClient.IsEnabled() {
		return nil, fmt.Errorf("LLM client not enabled - cannot classify bug closure")
	}

	systemPrompt := `You are a GitHub issue closure classifier. Analyze ALL comments (not just closing comment) to determine closure type.

Output JSON format:
{
  "closure_classification": "fixed_with_code",
  "classification_confidence": 0.85,
  "classification_rationale": "Detailed explanation...",
  "conversation_summary": "Summary of all comments...",
  "key_decision_snippets": ["Quote 1", "Quote 2"],
  "closing_comment_author": "username",
  "closing_comment_timestamp": "2024-01-01T12:00:00Z"
}

Classification categories:
- "fixed_with_code": Bug was fixed with code changes (expect PR)
- "not_a_bug": Closed as "not a bug", "working as intended", "user error"
- "duplicate": Issue is duplicate of another issue
- "wontfix": Closed as "won't fix", "not planned", "out of scope"
- "user_action_required": Closed with instructions for user (upgrade, config change, etc.)
- "unclear": Cannot determine from available information

Classification signals (match semantically, not just exact keywords):
- Keywords like "upgrade", "install", "update your version", "configuration" → likely user_action_required
- Keywords like "not a bug", "intended behavior", "expected", "by design" → likely not_a_bug
- Keywords like "duplicate of #", "see issue #" → likely duplicate
- Keywords like "won't fix", "not planned", "out of scope" → likely wontfix
- Keywords like "fixed", "pushed a fix", "merged", "should be resolved" → likely fixed_with_code
- No substantive comments or generic comment → unclear

Confidence scoring:
- Explicit keywords present: 0.90-0.95
- Inferred from context: 0.70-0.85
- Ambiguous: 0.50-0.65

IMPORTANT: Analyze the FULL conversation context, not just the closing comment.
Apply comment truncation if any comment > 2000 chars (1000 first + 500 last).`

	// Build user prompt with all comments
	userPrompt := fmt.Sprintf(`Classify this closed issue:

ISSUE #%d: %s
State: %s
Created: %s
Closed: %s
Labels: %v

Body:
%s

`, issue.IssueNumber, issue.Title, issue.State, issue.CreatedAt.Format(time.RFC3339), formatTimePtr(issue.ClosedAt), issue.Labels, truncateText(issue.Body, 500))

	userPrompt += fmt.Sprintf("Comments (%d total):\n", len(issue.Comments))
	for i, comment := range issue.Comments {
		body := comment.Body
		wasTruncated := false
		if len(body) > 2000 {
			body = body[:1000] + "...[TRUNCATED]..." + body[len(body)-500:]
			wasTruncated = true
		}

		truncatedNote := ""
		if wasTruncated {
			truncatedNote = " [TRUNCATED]"
		}

		userPrompt += fmt.Sprintf("\n[%d] %s by %s (%s)%s:\n%s\n",
			i+1,
			comment.CreatedAt.Format(time.RFC3339),
			comment.Author,
			comment.AuthorRole,
			truncatedNote,
			body,
		)
	}

	response, err := p.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM classification failed: %w", err)
	}

	var result types.BugClassificationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse classification response: %w", err)
	}

	result.IssueNumber = issue.IssueNumber

	// Determine if classification confidence is low
	result.LowClassificationConfidence = result.ClassificationConfidence <= 0.70

	return &result, nil
}

// shouldProceedToDeepFinder determines if we should run deep finder
func (p *Phase2PathB) shouldProceedToDeepFinder(classification *types.BugClassificationResult) bool {
	// Don't proceed for these closure types
	excludedTypes := []types.ClosureClassification{
		types.ClassNotABug,
		types.ClassDuplicate,
		types.ClassWontFix,
		types.ClassUserActionRequired,
	}

	for _, excludedType := range excludedTypes {
		if classification.ClosureClassification == excludedType {
			return false
		}
	}

	// Proceed for fixed_with_code
	if classification.ClosureClassification == types.ClassFixedWithCode {
		return true
	}

	// For unclear: proceed if confidence > 0.70
	if classification.ClosureClassification == types.ClassUnclear {
		return classification.ClassificationConfidence > 0.70
	}

	return false
}

// findCandidatePRs finds temporally close candidate PRs
func (p *Phase2PathB) findCandidatePRs(ctx context.Context, repoID int64, issue *types.IssueData) ([]types.CandidatePR, error) {
	if issue.ClosedAt == nil {
		return nil, fmt.Errorf("issue has no closed_at timestamp")
	}

	// Calculate DORA-based adaptive time window
	window := p.calculateAdaptiveTimeWindow()

	log.Printf("    Using temporal window: ±%.1f days", window.Hours()/24)

	// Query PRs merged near issue closure time
	prs, err := p.stagingDB.GetPRsMergedNear(ctx, repoID, *issue.ClosedAt, window)
	if err != nil {
		return nil, fmt.Errorf("failed to query candidate PRs: %w", err)
	}

	// Convert to types.CandidatePR with temporal metadata
	var candidates []types.CandidatePR
	for _, prData := range prs {
		if prData.MergedAt == nil {
			continue
		}

		delta := issue.ClosedAt.Sub(*prData.MergedAt)
		deltaSeconds := int64(math.Abs(delta.Seconds()))

		direction := "normal"
		if delta < 0 {
			direction = "reverse"
		}

		// Get PR details
		pr, err := p.stagingDB.GetPRByNumber(ctx, repoID, prData.Number)
		if err != nil {
			log.Printf("    ⚠️  Failed to get PR #%d: %v", prData.Number, err)
			continue
		}

		// Extract comment bodies
		var comments []string
		// Note: We'd need to fetch PR comments from github_pr_comments table
		// For now, we'll work with PR body

		candidate := types.CandidatePR{
			PRNumber:             pr.PRNumber,
			PRTitle:              pr.Title,
			PRDescription:        pr.Body,
			PRComments:           comments,
			PRMergedAt:           *pr.MergedAt,
			TemporalDeltaSeconds: deltaSeconds,
			TemporalDirection:    direction,
			WindowUsedDays:       window.Hours() / 24,
			WeakTemporalSignal:   false, // Will be set if window was expanded
		}

		candidates = append(candidates, candidate)
	}

	// Sort by temporal proximity (closest first)
	sortCandidatesByTemporalProximity(candidates)

	// Select top 5-10 candidates
	maxCandidates := 10
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	return candidates, nil
}

// calculateAdaptiveTimeWindow calculates DORA-based adaptive window
func (p *Phase2PathB) calculateAdaptiveTimeWindow() time.Duration {
	if p.doraMetrics.InsufficientHistory {
		// Use fixed ±3 days for new/low-activity repos
		return 3 * 24 * time.Hour
	}

	// DORA-based window: ±max(36 hours, 0.75 × median_lead_time, cap at 7 days)
	minHours := 36.0
	scaledLeadTime := 0.75 * p.doraMetrics.MedianLeadTimeHours
	maxHours := 7 * 24.0

	windowHours := math.Max(minHours, math.Min(scaledLeadTime, maxHours))

	return time.Duration(windowHours) * time.Hour
}

// rankCandidatesSemantically ranks candidates by semantic similarity
func (p *Phase2PathB) rankCandidatesSemantically(ctx context.Context, issue *types.IssueData, candidates []types.CandidatePR) ([]types.CandidatePR, error) {
	if !p.llmClient.IsEnabled() {
		return nil, fmt.Errorf("LLM client not enabled - cannot rank candidates")
	}

	// Process candidates in batch
	systemPrompt := `You are ranking candidate PRs by semantic similarity to a GitHub Issue.

Output JSON format:
{
  "rankings": [
    {
      "pr_number": 123,
      "title_score": 0.85,
      "body_score": 0.72,
      "comment_score": 0.90,
      "title_keywords": ["auth", "bug"],
      "body_keywords": ["fix", "login"],
      "comment_keywords": ["resolved"],
      "title_rationale": "High similarity...",
      "body_rationale": "Medium similarity...",
      "cross_content_rationale": "Issue closing comment matches PR title",
      "file_context_score": 0.8,
      "ranking_score": 0.78
    }
  ]
}

Scoring rules (0.0-1.0):
- Title Semantic: Compare issue title to PR title
- Body Semantic: Compare issue description to PR description
- Comment Semantic: Compare issue comments (especially closing) to PR title/body (CRITICAL)
- File Context: Match issue category (ui/api/docs) to PR file changes

Ranking score calculation:
ranking_score = 0.30 * temporal_proximity_score
              + 0.30 * comment_semantic_score
              + 0.20 * body_semantic_score
              + 0.15 * title_semantic_score
              + 0.05 * file_context_score

Provide detailed rationales and keyword overlaps.`

	issueClosingComment := ""
	if len(issue.Comments) > 0 {
		issueClosingComment = issue.Comments[len(issue.Comments)-1].Body
	}

	userPrompt := fmt.Sprintf(`Rank these candidate PRs for Issue #%d:

ISSUE:
Title: %s
Body: %s
Closing Comment: %s

CANDIDATES:
`, issue.IssueNumber, issue.Title, truncateText(issue.Body, 500), truncateText(issueClosingComment, 300))

	for _, candidate := range candidates {
		userPrompt += fmt.Sprintf("\nPR #%d:\n", candidate.PRNumber)
		userPrompt += fmt.Sprintf("Title: %s\n", candidate.PRTitle)
		userPrompt += fmt.Sprintf("Body: %s\n", truncateText(candidate.PRDescription, 500))
		userPrompt += fmt.Sprintf("Temporal Delta: %d seconds (%s)\n", candidate.TemporalDeltaSeconds, candidate.TemporalDirection)
	}

	response, err := p.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM ranking failed: %w", err)
	}

	var result struct {
		Rankings []struct {
			PRNumber              int      `json:"pr_number"`
			TitleScore            float64  `json:"title_score"`
			BodyScore             float64  `json:"body_score"`
			CommentScore          float64  `json:"comment_score"`
			TitleKeywords         []string `json:"title_keywords"`
			BodyKeywords          []string `json:"body_keywords"`
			CommentKeywords       []string `json:"comment_keywords"`
			TitleRationale        string   `json:"title_rationale"`
			BodyRationale         string   `json:"body_rationale"`
			CrossContentRationale string   `json:"cross_content_rationale"`
			FileContextScore      float64  `json:"file_context_score"`
			RankingScore          float64  `json:"ranking_score"`
		} `json:"rankings"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse ranking response: %w", err)
	}

	// Merge ranking results into candidates
	for i := range candidates {
		for _, ranking := range result.Rankings {
			if ranking.PRNumber == candidates[i].PRNumber {
				candidates[i].RankingScore = ranking.RankingScore

				// Build semantic scores
				candidates[i].SemanticScores = types.SemanticScores{
					TitleScore:            ranking.TitleScore,
					BodyScore:             ranking.BodyScore,
					CommentScore:          ranking.CommentScore,
					TitleKeywords:         ranking.TitleKeywords,
					BodyKeywords:          ranking.BodyKeywords,
					CommentKeywords:       ranking.CommentKeywords,
					TitleRationale:        ranking.TitleRationale,
					BodyRationale:         ranking.BodyRationale,
					CrossContentRationale: ranking.CrossContentRationale,
				}

				break
			}
		}
	}

	// Sort by ranking score (highest first)
	sortCandidatesByRankingScore(candidates)

	return candidates, nil
}

// createLinksFromCandidates creates links from ranked candidates
func (p *Phase2PathB) createLinksFromCandidates(issue *types.IssueData, candidates []types.CandidatePR, classification *types.BugClassificationResult) ([]types.LinkOutput, *types.NoLinkOutput) {
	if len(candidates) == 0 {
		return nil, &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkNoSemanticMatches,
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			CandidatesEvaluated:      len(candidates),
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
	}

	topScore := candidates[0].RankingScore

	// Apply thresholds
	threshold := 0.65
	if classification.LowClassificationConfidence && topScore < 0.70 {
		// Ambiguous classification + weak match → no link
		return nil, &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkAmbiguousClassificationWeak,
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			CandidatesEvaluated:      len(candidates),
			BestCandidateScore:       topScore,
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
	}

	// Safety brake: prevent temporal coincidence false positives
	if p.applySafetyBrake(candidates[0]) {
		return nil, &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkTemporalCoincidence,
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			CandidatesEvaluated:      len(candidates),
			BestCandidateScore:       topScore,
			SafetyBrakeReason:        "temporal_proximity < 0.20 and all semantic < 0.50",
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
	}

	// Select candidates within threshold
	var selectedCandidates []types.CandidatePR
	minScore := math.Max(threshold, topScore-0.10)

	for _, candidate := range candidates {
		if candidate.RankingScore >= minScore && len(selectedCandidates) < 3 {
			selectedCandidates = append(selectedCandidates, candidate)
		}
	}

	if len(selectedCandidates) == 0 {
		return nil, &types.NoLinkOutput{
			IssueNumber:              issue.IssueNumber,
			NoLinksReason:            types.NoLinkNoSemanticMatches,
			Classification:           classification.ClosureClassification,
			ClassificationConfidence: classification.ClassificationConfidence,
			ClassificationRationale:  classification.ClassificationRationale,
			ConversationSummary:      classification.ConversationSummary,
			CandidatesEvaluated:      len(candidates),
			BestCandidateScore:       topScore,
			IssueClosedAt:            *issue.ClosedAt,
			AnalyzedAt:               time.Now(),
		}
	}

	// Create links
	var links []types.LinkOutput
	for i, candidate := range selectedCandidates {
		// Calculate final confidence for deep links
		finalConfidence := 0.50 + (0.35 * candidate.RankingScore)
		if finalConfidence > 0.85 {
			finalConfidence = 0.85 // Deep links capped at 0.85
		}

		// Determine link quality
		linkQuality := types.QualityMedium
		if i > 0 || (len(selectedCandidates) >= 2 && math.Abs(topScore-selectedCandidates[1].RankingScore) < 0.03) {
			linkQuality = types.QualityLow
		}

		// Build evidence sources
		evidenceSources := []string{"temporal", "semantic_comment", "semantic_body", "semantic_title"}

		// Build rationale
		rationale := fmt.Sprintf("DEEP LINK found via temporal-semantic analysis (confidence: %.2f).\n\n", finalConfidence)
		rationale += fmt.Sprintf("Detection Method: deep_link_finder\n")
		rationale += fmt.Sprintf("Classification: %s (confidence: %.2f)\n\n", classification.ClosureClassification, classification.ClassificationConfidence)
		rationale += fmt.Sprintf("Temporal: Delta %d seconds (%s), Window %.1f days\n", candidate.TemporalDeltaSeconds, candidate.TemporalDirection, candidate.WindowUsedDays)
		rationale += fmt.Sprintf("Semantic: Title=%.2f, Body=%.2f, Comment=%.2f\n", candidate.SemanticScores.TitleScore, candidate.SemanticScores.BodyScore, candidate.SemanticScores.CommentScore)
		rationale += fmt.Sprintf("Ranking Score: %.2f\n", candidate.RankingScore)

		// Determine temporal pattern based on delta and direction
		temporalPattern := types.PatternNormal
		absDelta := math.Abs(float64(candidate.TemporalDeltaSeconds))
		if candidate.TemporalDirection == "reverse" {
			temporalPattern = types.PatternReverse
		} else if absDelta < 300 { // < 5 minutes
			temporalPattern = types.PatternSimultaneous
		} else if absDelta > 86400 { // > 24 hours
			temporalPattern = types.PatternDelayed
		}

		link := types.LinkOutput{
			IssueNumber:           issue.IssueNumber,
			PRNumber:              candidate.PRNumber,
			DetectionMethod:       types.DetectionDeepLinkFinder,
			FinalConfidence:       finalConfidence,
			LinkQuality:           linkQuality,
			ConfidenceBreakdown:   types.ConfidenceBreakdown{BaseConfidence: 0.50},
			EvidenceSources:       evidenceSources,
			ComprehensiveRationale: rationale,
			SemanticAnalysis:      &candidate.SemanticScores,
			TemporalAnalysis: &types.TemporalAnalysis{
				IssueClosedAt:        *issue.ClosedAt,
				PRMergedAt:           candidate.PRMergedAt,
				TemporalDeltaSeconds: candidate.TemporalDeltaSeconds,
				TemporalPattern:      temporalPattern,
				TemporalDirection:    candidate.TemporalDirection,
			},
			Flags: types.LinkFlags{
				NeedsManualReview: classification.LowClassificationConfidence,
				ReverseTemporal:   candidate.TemporalDirection == "reverse",
			},
			Metadata: types.LinkMetadata{},
		}

		links = append(links, link)
	}

	return links, nil
}

// applySafetyBrake checks if safety brake should reject the link
func (p *Phase2PathB) applySafetyBrake(candidate types.CandidatePR) bool {
	// Calculate temporal proximity score
	temporalProximity := 1.0 - (float64(candidate.TemporalDeltaSeconds) / (candidate.WindowUsedDays * 86400))

	// Safety brake conditions
	if temporalProximity < 0.20 &&
		candidate.SemanticScores.TitleScore < 0.50 &&
		candidate.SemanticScores.BodyScore < 0.50 &&
		candidate.SemanticScores.CommentScore < 0.50 {
		return true
	}

	return false
}

// Helper functions for sorting
func sortCandidatesByTemporalProximity(candidates []types.CandidatePR) {
	// Bubble sort by temporal delta (smallest first)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].TemporalDeltaSeconds < candidates[i].TemporalDeltaSeconds {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
}

func sortCandidatesByRankingScore(candidates []types.CandidatePR) {
	// Bubble sort by ranking score (highest first)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].RankingScore > candidates[i].RankingScore {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
}
