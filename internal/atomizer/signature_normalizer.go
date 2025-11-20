package atomizer

import (
	"regexp"
	"strings"
)

// NormalizeSignature standardizes signature format for consistent matching
func NormalizeSignature(signature string) string {
	if signature == "" {
		return ""
	}

	// Remove all whitespace
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(signature, "")

	// Convert common type aliases
	// Use word boundary or specific delimiters to avoid partial replacements
	// Handle int64/int32 as standalone types (after parameter names or colons)
	normalized = strings.ReplaceAll(normalized, "int64", "int")
	normalized = strings.ReplaceAll(normalized, "int32", "int")

	// Convert str to string only when it's a type (after colon, before comma or paren)
	normalized = regexp.MustCompile(`:str([,\)])`).ReplaceAllString(normalized, ":string$1")

	// Convert bool to boolean (use regex to match word boundaries)
	// Match :bool followed by comma, paren, or end of string
	normalized = regexp.MustCompile(`:bool([,\)]|$)`).ReplaceAllString(normalized, ":boolean$1")

	// Preserve generics like Promise<string>
	// Already handled by removing whitespace only

	// Lowercase parameter names (but preserve type case)
	// This is complex - skip for now, keep as-is

	return normalized
}

// ExtractParameterCount returns the number of parameters in a signature
func ExtractParameterCount(signature string) int {
	if signature == "" || signature == "()" {
		return 0
	}

	// Extract content between first ( and last )
	start := strings.Index(signature, "(")
	end := strings.LastIndex(signature, ")")

	if start == -1 || end == -1 || start >= end {
		return 0
	}

	content := signature[start+1 : end]
	if strings.TrimSpace(content) == "" {
		return 0
	}

	// Split by comma (simple approach)
	params := strings.Split(content, ",")
	return len(params)
}

// SignaturesMatch compares two signatures with optional fuzzy matching
func SignaturesMatch(sig1, sig2 string, fuzzy bool) bool {
	norm1 := NormalizeSignature(sig1)
	norm2 := NormalizeSignature(sig2)

	if !fuzzy {
		return norm1 == norm2
	}

	// Exact match
	if norm1 == norm2 {
		return true
	}

	// Parameter count must match
	if ExtractParameterCount(norm1) != ExtractParameterCount(norm2) {
		return false
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(norm1, norm2)
	// Use 20% threshold of the longer string for more lenient matching
	maxLen := len(norm1)
	if len(norm2) > maxLen {
		maxLen = len(norm2)
	}
	threshold := maxLen / 5 // 20% threshold

	return distance <= threshold
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				min(matrix[i-1][j]+1, matrix[i][j-1]+1),
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}
