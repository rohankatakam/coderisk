package graph

import (
	"math"
	"regexp"
	"strings"
)

// SemanticMatcher provides semantic similarity calculations for issue-PR matching
// Reference: LINKING_PATTERNS.md - Pattern 4: Semantic Similarity (10% of real-world cases)
type SemanticMatcher struct {
	stopWords map[string]bool
}

// NewSemanticMatcher creates a new semantic matcher with common stop words
func NewSemanticMatcher() *SemanticMatcher {
	// Common English stop words that don't contribute to semantic meaning
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "but": true, "by": true, "for": true, "if": true, "in": true,
		"into": true, "is": true, "it": true, "no": true, "not": true, "of": true,
		"on": true, "or": true, "such": true, "that": true, "the": true, "their": true,
		"then": true, "there": true, "these": true, "they": true, "this": true, "to": true,
		"was": true, "will": true, "with": true,
	}

	return &SemanticMatcher{
		stopWords: stopWords,
	}
}

// CalculateSimilarity computes semantic similarity between two text strings
// Returns a score between 0.0 (no overlap) and 1.0 (identical keywords)
func (sm *SemanticMatcher) CalculateSimilarity(text1, text2 string) float64 {
	keywords1 := sm.extractKeywords(text1)
	keywords2 := sm.extractKeywords(text2)

	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	// Use Jaccard similarity: intersection / union
	return sm.jaccardSimilarity(keywords1, keywords2)
}

// ValidateTemporalMatch validates a temporal match using semantic similarity
// Returns (shouldKeep bool, confidenceBoost float64)
//
// Strategy: Accept ALL temporal matches, use semantic similarity only to BOOST confidence
// This prevents false negatives from overly aggressive filtering while still rewarding
// semantic relevance. Precision is maintained by temporal proximity alone.
func (sm *SemanticMatcher) ValidateTemporalMatch(issueText, prText string) (bool, float64) {
	similarity := sm.CalculateSimilarity(issueText, prText)

	const (
		mediumSimilarity = 0.10 // Medium boost threshold
		highSimilarity   = 0.20 // High boost threshold
		mediumBoost      = 0.05 // Small boost for medium similarity
		highBoost        = 0.10 // Larger boost for high similarity
	)

	// Always accept temporal matches - semantic matching only adjusts confidence
	if similarity >= highSimilarity {
		return true, highBoost // Strong semantic match - high boost
	} else if similarity >= mediumSimilarity {
		return true, mediumBoost // Moderate semantic match - small boost
	} else {
		return true, 0.0 // Weak semantic match - no boost but still accept
	}
}

// extractKeywords extracts meaningful keywords from text
// - Converts to lowercase
// - Removes special characters
// - Splits on whitespace
// - Filters stop words
// - Handles common programming terms
func (sm *SemanticMatcher) extractKeywords(text string) map[string]bool {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove URLs
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	text = urlRegex.ReplaceAllString(text, "")

	// Remove markdown syntax
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "##", "")
	text = strings.ReplaceAll(text, "```", "")

	// Extract words (including hyphenated terms and numbers)
	wordRegex := regexp.MustCompile(`\b[a-z0-9]+(?:[_-][a-z0-9]+)*\b`)
	words := wordRegex.FindAllString(text, -1)

	keywords := make(map[string]bool)
	for _, word := range words {
		// Skip stop words
		if sm.stopWords[word] {
			continue
		}

		// Skip very short words (likely noise)
		if len(word) < 2 {
			continue
		}

		// Skip pure numbers unless they look like versions
		if isNumeric(word) && !looksLikeVersion(word) {
			continue
		}

		// Add the keyword
		keywords[word] = true

		// Also add stemmed version for better matching
		stem := simpleStem(word)
		if stem != word {
			keywords[stem] = true
		}
	}

	return keywords
}

// jaccardSimilarity computes Jaccard similarity between two keyword sets
// Formula: |A ∩ B| / |A ∪ B|
func (sm *SemanticMatcher) jaccardSimilarity(set1, set2 map[string]bool) float64 {
	// Count intersection
	intersection := 0
	for keyword := range set1 {
		if set2[keyword] {
			intersection++
		}
	}

	// Count union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// cosineSimilarity computes cosine similarity between two keyword sets
// Alternative to Jaccard, gives higher scores for partial matches
func (sm *SemanticMatcher) cosineSimilarity(set1, set2 map[string]bool) float64 {
	// Count intersection
	intersection := 0
	for keyword := range set1 {
		if set2[keyword] {
			intersection++
		}
	}

	if intersection == 0 {
		return 0.0
	}

	// Cosine similarity: intersection / sqrt(|A| * |B|)
	denominator := math.Sqrt(float64(len(set1)) * float64(len(set2)))
	if denominator == 0 {
		return 0.0
	}

	return float64(intersection) / denominator
}

// simpleStem performs simple stemming (removes common suffixes)
// Not as sophisticated as Porter stemmer, but good enough for our use case
func simpleStem(word string) string {
	// Remove common suffixes
	suffixes := []string{"ing", "ed", "es", "s", "er", "ly"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word) > len(suffix)+2 {
			return word[:len(word)-len(suffix)]
		}
	}

	return word
}

// isNumeric checks if a string is purely numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// looksLikeVersion checks if a string looks like a version number
func looksLikeVersion(s string) bool {
	// Matches patterns like: 1.0, 0.36.0, v1.2.3
	versionRegex := regexp.MustCompile(`^v?\d+\.\d+`)
	return versionRegex.MatchString(s)
}

// CalculateIssueToCommitSimilarity calculates similarity between an issue and a commit
func (sm *SemanticMatcher) CalculateIssueToCommitSimilarity(issueTitle, issueBody, commitMessage string) float64 {
	issueText := issueTitle + " " + issueBody
	return sm.CalculateSimilarity(issueText, commitMessage)
}

// CalculateIssueToPRSimilarity calculates similarity between an issue and a PR
// Uses the MAXIMUM of title-only and full-text similarity to avoid dilution from verbose bodies
func (sm *SemanticMatcher) CalculateIssueToPRSimilarity(issueTitle, issueBody, prTitle, prBody string) float64 {
	// Calculate title-to-title similarity (high signal)
	titleSimilarity := sm.CalculateSimilarity(issueTitle, prTitle)

	// Calculate full-text similarity (includes bodies)
	issueText := issueTitle + " " + issueBody
	prText := prTitle + " " + prBody
	fullTextSimilarity := sm.CalculateSimilarity(issueText, prText)

	// Use the maximum - if titles match strongly, that's enough
	if titleSimilarity > fullTextSimilarity {
		return titleSimilarity
	}
	return fullTextSimilarity
}
