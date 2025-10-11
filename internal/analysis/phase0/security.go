package phase0

import (
	"strings"
)

// SecurityKeywords contains all keywords that indicate security-sensitive code
// 12-factor: Factor 3 - Own your context window (fast pre-filtering to avoid expensive analysis)
var SecurityKeywords = []string{
	// Authentication
	"auth", "login", "logout", "signin", "signout", "authenticate",
	"session", "token", "jwt", "oauth", "saml", "sso",

	// Authorization
	"authorize", "permission", "role", "access", "grant", "deny",
	"admin", "privilege", "acl", "rbac", "sudo",

	// Cryptography
	"encrypt", "decrypt", "hash", "sign", "verify", "crypto",
	"cipher", "key", "secret", "password", "salt", "bcrypt",

	// Validation & Sanitization
	"validate", "sanitize", "escape", "filter",

	// Sensitive Data
	"pii", "ssn", "credit_card", "personal", "private", "confidential",
}

// SecurityPathPatterns contains file path patterns that indicate security-sensitive files
var SecurityPathPatterns = []string{
	"auth", "security", "permission", "credential", "secrets",
}

// SecurityDetectionResult contains the results of security keyword detection
type SecurityDetectionResult struct {
	IsSecuritySensitive bool     // True if security keywords/patterns detected
	MatchedKeywords     []string // Keywords found in content
	MatchedPathPatterns []string // Path patterns matched
	Reason              string   // Human-readable explanation
}

// DetectSecurityKeywords analyzes a file path and content to detect security-sensitive changes
// Returns true if security keywords are found in path or content
// 12-factor: Factor 8 - Own your control flow (explicit decision criteria for escalation)
func DetectSecurityKeywords(filePath, content string) SecurityDetectionResult {
	result := SecurityDetectionResult{
		MatchedKeywords:     []string{},
		MatchedPathPatterns: []string{},
	}

	// Check file path patterns (case-insensitive)
	filePathLower := strings.ToLower(filePath)
	for _, pattern := range SecurityPathPatterns {
		if strings.Contains(filePathLower, pattern) {
			result.MatchedPathPatterns = append(result.MatchedPathPatterns, pattern)
		}
	}

	// Check keywords in content (case-insensitive but preserving structure for boundaries)
	// We don't lowercase the content/path here to preserve CamelCase boundaries
	for _, keyword := range SecurityKeywords {
		if containsKeywordCaseInsensitive(content, keyword) || containsKeywordCaseInsensitive(filePath, keyword) {
			// Only add unique keywords
			if !contains(result.MatchedKeywords, keyword) {
				result.MatchedKeywords = append(result.MatchedKeywords, keyword)
			}
		}
	}

	// Determine if security-sensitive
	result.IsSecuritySensitive = len(result.MatchedKeywords) > 0 || len(result.MatchedPathPatterns) > 0

	// Build reason
	if result.IsSecuritySensitive {
		reasons := []string{}
		if len(result.MatchedPathPatterns) > 0 {
			reasons = append(reasons, "security-sensitive file path detected")
		}
		if len(result.MatchedKeywords) > 0 {
			reasons = append(reasons, "security keywords detected")
		}
		result.Reason = strings.Join(reasons, "; ")
	}

	return result
}

// containsKeywordCaseInsensitive checks if a keyword is present in text with word boundary awareness
// This reduces false positives by ensuring keyword is not embedded in unrelated words
// Handles CamelCase identifiers (e.g., "CheckPermission" matches "permission")
// Performs case-insensitive matching while preserving CamelCase boundaries
func containsKeywordCaseInsensitive(text, keyword string) bool {
	// Convert keyword to lowercase for comparison
	keywordLower := strings.ToLower(keyword)
	textLower := strings.ToLower(text)

	// Use word boundary matching to avoid false positives like "processor" matching "sso"
	// We look for the keyword preceded and followed by:
	// - Non-letter characters OR
	// - Uppercase letters in ORIGINAL text (CamelCase boundary)

	index := 0
	for {
		idx := strings.Index(textLower[index:], keywordLower)
		if idx == -1 {
			return false // Keyword not found
		}

		idx += index // Adjust for offset

		// Check if keyword is at word boundary using ORIGINAL text for case detection
		// Before: start of string OR non-letter OR uppercase in original (CamelCase boundary)
		beforeOK := idx == 0 || !isLetter(rune(text[idx-1])) || isUppercase(rune(text[idx]))

		// After: end of string OR non-letter OR uppercase in original (CamelCase boundary)
		afterIdx := idx + len(keywordLower)
		afterOK := afterIdx >= len(text) || !isLetter(rune(text[afterIdx])) || isUppercase(rune(text[afterIdx]))

		if beforeOK && afterOK {
			return true // Found keyword at word boundary
		}

		// Continue searching after this match
		index = idx + 1
		if index >= len(textLower) {
			return false
		}
	}
}

// isLetter checks if a character is a letter (a-z, A-Z)
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isUppercase checks if a character is an uppercase letter (A-Z)
func isUppercase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ShouldForceEscalate determines if a security detection should force escalation to Phase 2
// Returns true if detection is strong enough to skip Phase 1 baseline
func (r SecurityDetectionResult) ShouldForceEscalate() bool {
	// Force escalate if we found security patterns in path OR multiple keywords in content
	if len(r.MatchedPathPatterns) > 0 {
		return true // Security-sensitive file paths always escalate
	}
	if len(r.MatchedKeywords) >= 2 {
		return true // Multiple security keywords indicate high confidence
	}
	return false
}

// GetRiskLevel returns the risk level for security-sensitive changes
// Returns "CRITICAL" for high-confidence security changes, "HIGH" for moderate confidence
func (r SecurityDetectionResult) GetRiskLevel() string {
	if !r.IsSecuritySensitive {
		return "" // Not security-sensitive, don't override risk level
	}

	// High confidence: path patterns + keywords OR multiple keywords
	if len(r.MatchedPathPatterns) > 0 && len(r.MatchedKeywords) > 0 {
		return "CRITICAL"
	}
	if len(r.MatchedKeywords) >= 3 {
		return "CRITICAL"
	}

	// Moderate confidence: path patterns OR 1-2 keywords
	if len(r.MatchedPathPatterns) > 0 || len(r.MatchedKeywords) >= 1 {
		return "HIGH"
	}

	return "HIGH" // Default for security-sensitive
}
