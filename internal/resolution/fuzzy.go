package resolution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// CodeBlockCandidate represents a potential match for entity resolution
type CodeBlockCandidate struct {
	ID                int64
	Name              string
	CanonicalFilePath string
	StartLine         int
	EndLine           int
	Content           string // Full content of the code block
}

// ResolutionResult contains the fuzzy resolution outcome
type ResolutionResult struct {
	MatchedCandidate *CodeBlockCandidate
	Confidence       float64
	Method           string // "unique", "fuzzy_llm", "heuristic"
	Reason           string
}

// FuzzyResolver performs LLM-based disambiguation for duplicate block names
type FuzzyResolver struct {
	llmClient *llm.Client
	logger    *slog.Logger
}

// NewFuzzyResolver creates a new fuzzy entity resolver
func NewFuzzyResolver(llmClient *llm.Client) *FuzzyResolver {
	return &FuzzyResolver{
		llmClient: llmClient,
		logger:    slog.Default().With("component", "fuzzy_resolver"),
	}
}

// ResolveCodeBlock disambiguates between multiple code blocks with the same name
// Uses hybrid context strategy: first 10 + last 5 + smart middle lines
func (fr *FuzzyResolver) ResolveCodeBlock(ctx context.Context, candidates []CodeBlockCandidate, diffContent string) (*ResolutionResult, error) {
	// Edge case 1: No candidates
	if len(candidates) == 0 {
		return &ResolutionResult{
			MatchedCandidate: nil,
			Confidence:       0.0,
			Method:           "no_match",
			Reason:           "No matching code blocks found",
		}, nil
	}

	// Edge case 2: Single candidate (unique match)
	if len(candidates) == 1 {
		fr.logger.Debug("unique match found, no disambiguation needed",
			"block_name", candidates[0].Name,
			"file", candidates[0].CanonicalFilePath,
		)
		return &ResolutionResult{
			MatchedCandidate: &candidates[0],
			Confidence:       1.0,
			Method:           "unique",
			Reason:           "Only one block with this name exists",
		}, nil
	}

	// Edge case 3: Multiple candidates require LLM disambiguation
	fr.logger.Info("multiple candidates found, using fuzzy LLM resolution",
		"block_name", candidates[0].Name,
		"candidate_count", len(candidates),
	)

	// If LLM unavailable, use heuristic fallback
	if fr.llmClient == nil || !fr.llmClient.IsEnabled() {
		return fr.heuristicResolve(candidates, diffContent)
	}

	// Use LLM for semantic matching
	return fr.llmResolve(ctx, candidates, diffContent)
}

// llmResolve uses LLM to semantically match the diff to a candidate
func (fr *FuzzyResolver) llmResolve(ctx context.Context, candidates []CodeBlockCandidate, diffContent string) (*ResolutionResult, error) {
	// Extract diff excerpt for context
	excerpt := git.ExtractExcerptForResolution(diffContent, 1500) // 1500 token budget

	// Build candidate contexts with hybrid excerpts
	var candidateContexts []string
	for i, c := range candidates {
		context := fr.extractHybridContext(c.Content, 500) // ~500 tokens per candidate
		candidateContexts = append(candidateContexts, fmt.Sprintf("Candidate %d (lines %d-%d):\n%s",
			i, c.StartLine, c.EndLine, context))
	}

	// Construct LLM prompt
	systemPrompt := `You are a code entity resolver. Your job is to match a code diff to the correct code block from multiple candidates with the same name.

Analyze the semantic content of the diff and each candidate to determine the best match.

Return a JSON object with:
{
  "matched_index": <0-based index of matched candidate, or -1 if no good match>,
  "confidence": <0.0-1.0, where >0.7 is high confidence>,
  "reasoning": "<brief explanation of why this candidate matches>"
}

IMPORTANT: Only return high confidence (>0.7) if you're certain. If unsure, return low confidence.`

	userPrompt := fmt.Sprintf(`Diff excerpt showing code change:
%s

Candidates (%d total):
%s

Which candidate is being modified by this diff?`, excerpt.FormatExcerpt(), len(candidates), strings.Join(candidateContexts, "\n\n"))

	// Call LLM
	response, err := fr.llmClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		fr.logger.Warn("LLM resolution failed, falling back to heuristic",
			"error", err,
		)
		return fr.heuristicResolve(candidates, diffContent)
	}

	// Parse JSON response
	var llmResult struct {
		MatchedIndex int     `json:"matched_index"`
		Confidence   float64 `json:"confidence"`
		Reasoning    string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &llmResult); err != nil {
		fr.logger.Warn("failed to parse LLM response, falling back to heuristic",
			"error", err,
			"response", response,
		)
		return fr.heuristicResolve(candidates, diffContent)
	}

	// Validate result
	if llmResult.MatchedIndex < 0 || llmResult.MatchedIndex >= len(candidates) {
		return &ResolutionResult{
			MatchedCandidate: nil,
			Confidence:       llmResult.Confidence,
			Method:           "fuzzy_llm",
			Reason:           llmResult.Reasoning,
		}, nil
	}

	// Check confidence threshold
	if llmResult.Confidence < 0.7 {
		fr.logger.Warn("LLM confidence below threshold, resolution uncertain",
			"confidence", llmResult.Confidence,
			"reasoning", llmResult.Reasoning,
		)
		return &ResolutionResult{
			MatchedCandidate: nil,
			Confidence:       llmResult.Confidence,
			Method:           "fuzzy_llm",
			Reason:           fmt.Sprintf("Low confidence (%.2f): %s", llmResult.Confidence, llmResult.Reasoning),
		}, nil
	}

	// Success
	matched := candidates[llmResult.MatchedIndex]
	fr.logger.Info("fuzzy LLM resolution succeeded",
		"block_name", matched.Name,
		"matched_candidate", llmResult.MatchedIndex,
		"confidence", llmResult.Confidence,
		"reasoning", llmResult.Reasoning,
	)

	return &ResolutionResult{
		MatchedCandidate: &matched,
		Confidence:       llmResult.Confidence,
		Method:           "fuzzy_llm",
		Reason:           llmResult.Reasoning,
	}, nil
}

// heuristicResolve uses deterministic rules when LLM unavailable
func (fr *FuzzyResolver) heuristicResolve(candidates []CodeBlockCandidate, diffContent string) (*ResolutionResult, error) {
	// Heuristic 1: Prefer candidate with most similar line range
	// (Assumes code blocks don't move drastically between commits)

	// Extract changed line numbers from diff
	changedLines := extractChangedLineNumbers(diffContent)
	if len(changedLines) == 0 {
		// Can't determine, return first candidate with low confidence
		fr.logger.Warn("heuristic resolution: no line numbers in diff, returning first candidate")
		return &ResolutionResult{
			MatchedCandidate: &candidates[0],
			Confidence:       0.5,
			Method:           "heuristic",
			Reason:           "No line numbers in diff, selected first candidate (low confidence)",
		}, nil
	}

	// Find candidate with most overlap with changed lines
	bestCandidate := 0
	bestOverlap := 0

	for i, c := range candidates {
		overlap := 0
		for _, line := range changedLines {
			if line >= c.StartLine && line <= c.EndLine {
				overlap++
			}
		}
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestCandidate = i
		}
	}

	confidence := float64(bestOverlap) / float64(len(changedLines))
	if confidence < 0.5 {
		confidence = 0.5 // Minimum confidence for heuristic
	}

	fr.logger.Info("heuristic resolution completed",
		"block_name", candidates[bestCandidate].Name,
		"matched_candidate", bestCandidate,
		"confidence", confidence,
		"overlap", bestOverlap,
	)

	return &ResolutionResult{
		MatchedCandidate: &candidates[bestCandidate],
		Confidence:       confidence,
		Method:           "heuristic",
		Reason:           fmt.Sprintf("Line range overlap: %d/%d lines", bestOverlap, len(changedLines)),
	}, nil
}

// extractHybridContext extracts first 10 + last 5 + smart middle from code content
func (fr *FuzzyResolver) extractHybridContext(fullContent string, maxTokens int) string {
	lines := strings.Split(fullContent, "\n")

	// If small enough, return all
	if len(lines) <= 20 || len(fullContent) < maxTokens*4 {
		return fullContent
	}

	// Extract first 10 lines
	firstLines := lines[:min(10, len(lines))]

	// Extract last 5 lines
	lastStart := max(10, len(lines)-5)
	lastLines := lines[lastStart:]

	// Extract smart middle (code-dense lines)
	middleStart := 10
	middleEnd := lastStart
	middleLines := selectCodeDenseLines(lines[middleStart:middleEnd], maxTokens/4) // Reserve tokens for middle

	// Build output
	var b strings.Builder
	for _, line := range firstLines {
		b.WriteString(line + "\n")
	}

	if len(middleLines) > 0 {
		b.WriteString("... [truncated] ...\n")
		for _, line := range middleLines {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("... [truncated] ...\n")
	for _, line := range lastLines {
		b.WriteString(line + "\n")
	}

	return b.String()
}

// extractChangedLineNumbers extracts line numbers from diff @@ headers
func extractChangedLineNumbers(diffContent string) []int {
	var lineNumbers []int
	lines := strings.Split(diffContent, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			start, end := git.ParseAtHeaders(line)
			for i := start; i <= end; i++ {
				lineNumbers = append(lineNumbers, i)
			}
		}
	}

	return lineNumbers
}

// selectCodeDenseLines is a simplified version for context extraction
func selectCodeDenseLines(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}

	// Score each line
	type scoredLine struct {
		line  string
		score int
		index int
	}

	var scored []scoredLine
	for i, line := range lines {
		score := calculateCodeDensity(line)
		scored = append(scored, scoredLine{line, score, i})
	}

	// Sort by score (descending)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Take top N lines
	topLines := scored[:maxLines]

	// Sort by original index to preserve order
	for i := 0; i < len(topLines); i++ {
		for j := i + 1; j < len(topLines); j++ {
			if topLines[j].index < topLines[i].index {
				topLines[i], topLines[j] = topLines[j], topLines[i]
			}
		}
	}

	result := make([]string, len(topLines))
	for i, sl := range topLines {
		result[i] = sl.line
	}

	return result
}

// calculateCodeDensity scores a line by code content
func calculateCodeDensity(line string) int {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return 0
	}

	// Skip comments
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") {
		return 0
	}

	// Score code indicators
	score := 0
	score += strings.Count(trimmed, "(") * 2
	score += strings.Count(trimmed, "{") * 2
	score += strings.Count(trimmed, "=") * 1
	score += strings.Count(trimmed, ".") * 1
	score += len(strings.Fields(trimmed))

	return score
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
