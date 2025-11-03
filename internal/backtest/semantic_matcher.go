package backtest

import (
	"context"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// SemanticMatcher validates semantic linking patterns
type SemanticMatcher struct {
	stagingDB *database.StagingClient
	neo4jDB   *graph.Client
}

// NewSemanticMatcher creates a semantic matcher
func NewSemanticMatcher(stagingDB *database.StagingClient, neo4jDB *graph.Client) *SemanticMatcher {
	return &SemanticMatcher{
		stagingDB: stagingDB,
		neo4jDB:   neo4jDB,
	}
}

// SemanticValidationResult represents the validation of semantic links
type SemanticValidationResult struct {
	IssueNumber      int                      `json:"issue_number"`
	IssueTitle       string                   `json:"issue_title"`
	ExpectedPRs      []int                    `json:"expected_prs"`
	SemanticMatches  []SemanticPRMatch        `json:"semantic_matches"`
	Matched          bool                     `json:"matched"`
	MatchedPRs       []int                    `json:"matched_prs"`
	MissedPRs        []int                    `json:"missed_prs"`
	FalsePositives   []int                    `json:"false_positives"`
	KeywordAnalysis  map[string]interface{}   `json:"keyword_analysis"`
}

// SemanticPRMatch represents a semantic match between issue and PR
type SemanticPRMatch struct {
	PRNumber           int                    `json:"pr_number"`
	PRTitle            string                 `json:"pr_title"`
	SemanticScore      float64                `json:"semantic_score"`
	Confidence         float64                `json:"confidence"`
	Evidence           []string               `json:"evidence"`
	KeywordOverlap     map[string]interface{} `json:"keyword_overlap"`
	IssueKeywords      []string               `json:"issue_keywords"`
	PRKeywords         []string               `json:"pr_keywords"`
	CommonKeywords     []string               `json:"common_keywords"`
}

// ValidateSemanticLinks validates semantic linking for a ground truth test case
func (sm *SemanticMatcher) ValidateSemanticLinks(ctx context.Context, testCase GroundTruthTestCase) (*SemanticValidationResult, error) {
	result := &SemanticValidationResult{
		IssueNumber:      testCase.IssueNumber,
		IssueTitle:       testCase.Title,
		ExpectedPRs:      testCase.ExpectedLinks.AssociatedPRs,
		SemanticMatches:  []SemanticPRMatch{},
		MatchedPRs:       []int{},
		MissedPRs:        []int{},
		FalsePositives:   []int{},
		KeywordAnalysis:  make(map[string]interface{}),
	}

	// Extract expected keyword overlap if provided
	var expectedKeywords []string
	if keywords, ok := testCase.PrimaryEvidence["keyword_overlap"].([]interface{}); ok {
		for _, kw := range keywords {
			if kwStr, ok := kw.(string); ok {
				expectedKeywords = append(expectedKeywords, kwStr)
			}
		}
		result.KeywordAnalysis["expected_keywords"] = expectedKeywords
	}

	// Query all PRs in the repo for semantic matching
	query := `
		SELECT
			pr.number,
			pr.title,
			pr.body,
			pr.state
		FROM github_pull_requests pr
		WHERE pr.repo_id = (
			SELECT repo_id FROM github_issues WHERE number = $1 LIMIT 1
		)
		AND pr.state = 'closed'
		ORDER BY pr.number DESC
		LIMIT 100
	`

	rows, err := sm.stagingDB.Query(ctx, query, testCase.IssueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRs: %w", err)
	}
	defer rows.Close()

	// Process each PR
	for rows.Next() {
		var prNumber int
		var prTitle string
		var prBody *string
		var prState string

		if err := rows.Scan(&prNumber, &prTitle, &prBody, &prState); err != nil {
			log.Printf("  ⚠️  Failed to scan PR row: %v", err)
			continue
		}

		// Calculate semantic match
		match := sm.calculateSemanticMatch(testCase.IssueNumber, testCase.Title, prNumber, prTitle, prBody)

		// Only include matches with some semantic similarity
		if match.SemanticScore >= 0.20 {
			result.SemanticMatches = append(result.SemanticMatches, match)
		}
	}

	// Validate against expected PRs
	for _, expectedPR := range testCase.ExpectedLinks.AssociatedPRs {
		found := false
		for _, detected := range result.SemanticMatches {
			if detected.PRNumber == expectedPR {
				result.MatchedPRs = append(result.MatchedPRs, expectedPR)
				found = true
				break
			}
		}
		if !found {
			result.MissedPRs = append(result.MissedPRs, expectedPR)
		}
	}

	// Find false positives (detected but not expected)
	for _, detected := range result.SemanticMatches {
		expected := false
		for _, expectedPR := range testCase.ExpectedLinks.AssociatedPRs {
			if detected.PRNumber == expectedPR {
				expected = true
				break
			}
		}
		// Only count high-confidence semantic matches as false positives
		if !expected && detected.SemanticScore >= 0.60 {
			result.FalsePositives = append(result.FalsePositives, detected.PRNumber)
		}
	}

	result.Matched = len(result.MissedPRs) == 0 && len(result.FalsePositives) == 0

	return result, nil
}

// calculateSemanticMatch calculates semantic similarity and confidence
func (sm *SemanticMatcher) calculateSemanticMatch(
	issueNumber int,
	issueTitle string,
	prNumber int,
	prTitle string,
	prBody *string,
) SemanticPRMatch {
	match := SemanticPRMatch{
		PRNumber:       prNumber,
		PRTitle:        prTitle,
		Evidence:       []string{},
		KeywordOverlap: make(map[string]interface{}),
	}

	// Extract keywords from issue
	issueKeywordsList := sm.extractKeywordList(issueTitle)
	match.IssueKeywords = issueKeywordsList

	// Extract keywords from PR
	prText := prTitle
	if prBody != nil {
		prText += " " + *prBody
	}
	prKeywordsList := sm.extractKeywordList(prText)
	match.PRKeywords = prKeywordsList

	// Calculate Jaccard similarity
	semanticScore := graph.ComputeSemanticSimilarity(issueTitle, prTitle)
	match.SemanticScore = semanticScore

	// Find common keywords
	commonKeywords := []string{}
	for _, issueKW := range issueKeywordsList {
		for _, prKW := range prKeywordsList {
			if issueKW == prKW {
				commonKeywords = append(commonKeywords, issueKW)
				break
			}
		}
	}
	match.CommonKeywords = commonKeywords

	// Store keyword analysis
	match.KeywordOverlap["issue_keyword_count"] = len(issueKeywordsList)
	match.KeywordOverlap["pr_keyword_count"] = len(prKeywordsList)
	match.KeywordOverlap["common_keyword_count"] = len(commonKeywords)
	match.KeywordOverlap["jaccard_similarity"] = semanticScore

	// Apply confidence scoring based on LINKING_PATTERNS.md
	baseConfidence := 0.50

	if semanticScore >= 0.70 {
		baseConfidence = 0.65 // High semantic match
		match.Evidence = append(match.Evidence, "semantic_high")
	} else if semanticScore >= 0.50 {
		baseConfidence = 0.60 // Medium semantic match
		match.Evidence = append(match.Evidence, "semantic_medium")
	} else if semanticScore >= 0.30 {
		baseConfidence = 0.55 // Low semantic match
		match.Evidence = append(match.Evidence, "semantic_low")
	}

	match.Confidence = baseConfidence

	return match
}

// extractKeywordList extracts keywords as a list
func (sm *SemanticMatcher) extractKeywordList(text string) []string {
	// Use the existing keyword extraction from linking_quality_score.go
	// For now, implement a simple version here
	keywordMap := extractKeywordsToMap(text)

	keywordList := []string{}
	for kw := range keywordMap {
		keywordList = append(keywordList, kw)
	}

	return keywordList
}

// extractKeywordsToMap extracts keywords to a map (similar to graph package)
func extractKeywordsToMap(text string) map[string]bool {
	// Import the logic from graph.extractKeywords
	// For simplicity, implement a basic version here
	words := splitWords(text)
	keywords := make(map[string]bool)

	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
	}

	for _, word := range words {
		word = toLowerCase(word)
		word = trimPunctuation(word)

		if len(word) >= 3 && !stopWords[word] {
			keywords[word] = true
		}
	}

	return keywords
}

// splitWords splits text into words
func splitWords(text string) []string {
	words := []string{}
	currentWord := ""

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' || char == '\r' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(char)
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}

// toLowerCase converts string to lowercase
func toLowerCase(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

// trimPunctuation removes punctuation from start/end of string
func trimPunctuation(s string) string {
	punctuation := ".,!?;:\"'()[]{}#"

	// Trim from start
	for len(s) > 0 {
		found := false
		for _, p := range punctuation {
			if rune(s[0]) == p {
				s = s[1:]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	// Trim from end
	for len(s) > 0 {
		found := false
		for _, p := range punctuation {
			if rune(s[len(s)-1]) == p {
				s = s[:len(s)-1]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	return s
}

// ValidateAllSemanticCases validates all semantic test cases from ground truth
func (sm *SemanticMatcher) ValidateAllSemanticCases(ctx context.Context, groundTruth *GroundTruth) ([]SemanticValidationResult, error) {
	log.Printf("🔍 Validating semantic linking patterns...")

	results := []SemanticValidationResult{}

	for _, testCase := range groundTruth.TestCases {
		// Only validate semantic cases
		hasSemantic := false
		for _, pattern := range testCase.LinkingPatterns {
			if pattern == "semantic" {
				hasSemantic = true
				break
			}
		}

		if !hasSemantic {
			continue
		}

		log.Printf("  Testing Issue #%d (semantic)", testCase.IssueNumber)

		result, err := sm.ValidateSemanticLinks(ctx, testCase)
		if err != nil {
			log.Printf("    ❌ Error: %v", err)
			continue
		}

		if result.Matched {
			log.Printf("    ✅ Matched: %v", result.MatchedPRs)
			for _, match := range result.SemanticMatches {
				if contains(result.MatchedPRs, match.PRNumber) {
					log.Printf("       PR #%d: similarity=%.2f, confidence=%.2f",
						match.PRNumber, match.SemanticScore, match.Confidence)
					log.Printf("       Common keywords: %v", match.CommonKeywords)
				}
			}
		} else {
			log.Printf("    ❌ Failed:")
			if len(result.MissedPRs) > 0 {
				log.Printf("       Missed PRs: %v", result.MissedPRs)
			}
			if len(result.FalsePositives) > 0 {
				log.Printf("       False Positives: %v", result.FalsePositives)
			}
		}

		results = append(results, *result)
	}

	return results, nil
}

// CalculateSemanticMetrics computes metrics for semantic linking
func (sm *SemanticMatcher) CalculateSemanticMetrics(results []SemanticValidationResult) map[string]interface{} {
	metrics := make(map[string]interface{})

	totalCases := len(results)
	matched := 0
	totalExpected := 0
	totalDetected := 0
	totalMissed := 0
	totalFP := 0

	var semanticScoreSum float64
	var confidenceSum float64
	scoreCount := 0

	for _, result := range results {
		if result.Matched {
			matched++
		}

		totalExpected += len(result.ExpectedPRs)
		totalDetected += len(result.SemanticMatches)
		totalMissed += len(result.MissedPRs)
		totalFP += len(result.FalsePositives)

		// Average scores across all detected links
		for _, match := range result.SemanticMatches {
			semanticScoreSum += match.SemanticScore
			confidenceSum += match.Confidence
			scoreCount++
		}
	}

	metrics["total_cases"] = totalCases
	metrics["matched_cases"] = matched
	metrics["match_rate"] = float64(matched) / float64(totalCases)

	metrics["total_expected_links"] = totalExpected
	metrics["total_detected_links"] = totalDetected
	metrics["total_missed_links"] = totalMissed
	metrics["total_false_positives"] = totalFP

	if totalDetected > 0 {
		metrics["precision"] = float64(totalExpected-totalMissed) / float64(totalDetected)
	} else {
		metrics["precision"] = 0.0
	}

	if totalExpected > 0 {
		metrics["recall"] = float64(totalExpected-totalMissed) / float64(totalExpected)
	} else {
		metrics["recall"] = 0.0
	}

	if scoreCount > 0 {
		metrics["avg_semantic_score"] = semanticScoreSum / float64(scoreCount)
		metrics["avg_confidence"] = confidenceSum / float64(scoreCount)
	} else {
		metrics["avg_semantic_score"] = 0.0
		metrics["avg_confidence"] = 0.0
	}

	precision := metrics["precision"].(float64)
	recall := metrics["recall"].(float64)
	if precision+recall > 0 {
		metrics["f1_score"] = 2 * (precision * recall) / (precision + recall)
	} else {
		metrics["f1_score"] = 0.0
	}

	return metrics
}

// contains checks if a slice contains an int
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
