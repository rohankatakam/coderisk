package phase0

import (
	"strings"
)

// ModificationType represents the type of code modification
type ModificationType int

const (
	TypeUnknown ModificationType = iota
	TypeStructural
	TypeBehavioral
	TypeConfiguration
	TypeInterface
	TypeTestQuality
	TypeDocumentation
	TypeTemporalPattern
	TypeOwnership
	TypeSecurity
	TypePerformance
)

// String returns the string representation of ModificationType
func (mt ModificationType) String() string {
	switch mt {
	case TypeStructural:
		return "Structural"
	case TypeBehavioral:
		return "Behavioral"
	case TypeConfiguration:
		return "Configuration"
	case TypeInterface:
		return "Interface"
	case TypeTestQuality:
		return "TestQuality"
	case TypeDocumentation:
		return "Documentation"
	case TypeTemporalPattern:
		return "TemporalPattern"
	case TypeOwnership:
		return "Ownership"
	case TypeSecurity:
		return "Security"
	case TypePerformance:
		return "Performance"
	default:
		return "Unknown"
	}
}

// GetBaseRiskLevel returns the base risk level for a modification type
func (mt ModificationType) GetBaseRiskLevel() string {
	switch mt {
	case TypeSecurity:
		return "CRITICAL"
	case TypeInterface:
		return "HIGH"
	case TypeStructural:
		return "HIGH"
	case TypeConfiguration:
		return "MODERATE" // Can vary based on environment
	case TypeBehavioral:
		return "MODERATE"
	case TypePerformance:
		return "MODERATE"
	case TypeTemporalPattern:
		return "MODERATE"
	case TypeOwnership:
		return "MODERATE"
	case TypeTestQuality:
		return "LOW"
	case TypeDocumentation:
		return "VERY_LOW"
	default:
		return "UNKNOWN"
	}
}

// ModificationClassification contains the results of modification type classification
type ModificationClassification struct {
	PrimaryType    ModificationType   // Most significant type
	SecondaryTypes []ModificationType // Additional types detected
	AllTypes       []ModificationType // All types (primary + secondary)
	AggregatedRisk string             // Final risk after aggregation
	Reasons        map[ModificationType]string // Reason for each type
}

// ClassifyModification analyzes a file to determine its modification type(s)
// Supports multi-type detection (e.g., security + behavioral)
// 12-factor: Factor 8 - Own your control flow (explicit type-based routing)
func ClassifyModification(filePath, content string) ModificationClassification {
	result := ModificationClassification{
		PrimaryType:    TypeUnknown,
		SecondaryTypes: []ModificationType{},
		AllTypes:       []ModificationType{},
		Reasons:        make(map[ModificationType]string),
	}

	// Check all types and collect matches
	detectedTypes := make(map[ModificationType]string)

	// Type 9: Security-Sensitive Changes (highest priority)
	if securityResult := DetectSecurityKeywords(filePath, content); securityResult.IsSecuritySensitive {
		detectedTypes[TypeSecurity] = securityResult.Reason
	}

	// Type 6: Documentation Changes
	if docResult := IsDocumentationOnly(filePath); docResult.IsDocumentationOnly {
		detectedTypes[TypeDocumentation] = docResult.Reason
	}

	// Type 3: Configuration Changes
	if envResult := DetectEnvironment(filePath); envResult.IsConfiguration {
		detectedTypes[TypeConfiguration] = envResult.Reason
	}

	// Type 5: Test Quality Changes
	if isTestFile(filePath) {
		detectedTypes[TypeTestQuality] = "Test file modification"
	}

	// Type 4: Interface Changes (API, schema)
	if isInterfaceFile(filePath, content) {
		detectedTypes[TypeInterface] = "Interface/API modification"
	}

	// Type 1: Structural Changes (refactoring, imports)
	if isStructuralChange(filePath, content) {
		detectedTypes[TypeStructural] = "Structural/architectural change"
	}

	// Type 2: Behavioral Changes (logic, algorithms)
	if isBehavioralChange(content) {
		detectedTypes[TypeBehavioral] = "Behavioral/logic change"
	}

	// Type 10: Performance-Critical Changes
	if isPerformanceCritical(filePath, content) {
		detectedTypes[TypePerformance] = "Performance-critical change"
	}

	// If no types detected, return unknown
	if len(detectedTypes) == 0 {
		return result
	}

	// Sort types by priority (security > interface > structural > ...)
	result.AllTypes = sortTypesByPriority(getKeys(detectedTypes))
	result.PrimaryType = result.AllTypes[0]
	if len(result.AllTypes) > 1 {
		result.SecondaryTypes = result.AllTypes[1:]
	}
	result.Reasons = detectedTypes

	// Calculate aggregated risk
	result.AggregatedRisk = calculateAggregatedRisk(result.AllTypes)

	return result
}

// isTestFile checks if a file is a test file
func isTestFile(filePath string) bool {
	filePathLower := strings.ToLower(filePath)

	// Go test files
	if strings.HasSuffix(filePathLower, "_test.go") {
		return true
	}

	// Python test files
	if strings.HasPrefix(getFileNameLower(filePathLower), "test_") ||
		strings.HasSuffix(filePathLower, "_test.py") {
		return true
	}

	// JavaScript/TypeScript test files
	if strings.Contains(filePathLower, ".test.") ||
		strings.Contains(filePathLower, ".spec.") {
		return true
	}

	// Test directories
	if strings.Contains(filePathLower, "/test/") ||
		strings.Contains(filePathLower, "/tests/") ||
		strings.Contains(filePathLower, "/__tests__/") {
		return true
	}

	return false
}

// isInterfaceFile checks if a file defines interfaces or APIs
func isInterfaceFile(filePath, content string) bool {
	filePathLower := strings.ToLower(filePath)
	contentLower := strings.ToLower(content)

	// File path indicators
	interfacePatterns := []string{
		"/api/",
		"/interface/",
		"/interfaces/",
		"/schema/",
		"/proto/",
		"/graphql/",
	}

	for _, pattern := range interfacePatterns {
		if strings.Contains(filePathLower, pattern) {
			return true
		}
	}

	// Content indicators (interface definitions, API endpoints)
	interfaceKeywords := []string{
		"interface",
		"@api",
		"@endpoint",
		"router.get",
		"router.post",
		"router.put",
		"router.delete",
		"@route",
		"openapi",
		"swagger",
	}

	for _, keyword := range interfaceKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// isStructuralChange checks if the change is structural (refactoring, imports)
func isStructuralChange(filePath, content string) bool {
	contentLower := strings.ToLower(content)

	// Import/dependency changes
	structuralKeywords := []string{
		"import",
		"require(",
		"from ",
		"package ",
		"use ",
		"include",
	}

	for _, keyword := range structuralKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// isBehavioralChange checks if the change affects behavior (logic, algorithms)
func isBehavioralChange(content string) bool {
	contentLower := strings.ToLower(content)

	// Logic keywords
	behavioralKeywords := []string{
		"if ",
		"else",
		"switch",
		"case",
		"for ",
		"while",
		"loop",
		"return",
		"throw",
		"raise",
	}

	// Count occurrences (behavioral if significant logic present)
	count := 0
	for _, keyword := range behavioralKeywords {
		if strings.Contains(contentLower, keyword) {
			count++
		}
	}

	return count >= 2 // At least 2 logic keywords
}

// isPerformanceCritical checks if the change is performance-related
func isPerformanceCritical(filePath, content string) bool {
	filePathLower := strings.ToLower(filePath)
	contentLower := strings.ToLower(content)

	// Performance-related paths
	perfPaths := []string{
		"/cache/",
		"/performance/",
		"/optimization/",
		"/worker/",
		"/queue/",
	}

	for _, path := range perfPaths {
		if strings.Contains(filePathLower, path) {
			return true
		}
	}

	// Performance keywords in content
	perfKeywords := []string{
		"cache",
		"optimize",
		"performance",
		"benchmark",
		"throttle",
		"debounce",
		"lazy",
		"async",
		"concurrent",
		"parallel",
	}

	for _, keyword := range perfKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// sortTypesByPriority sorts modification types by risk priority
func sortTypesByPriority(types []ModificationType) []ModificationType {
	// Priority order: Security > Interface > Structural > Configuration >
	// Behavioral > Performance > TemporalPattern > Ownership > TestQuality > Documentation
	priority := map[ModificationType]int{
		TypeSecurity:        10,
		TypeInterface:       9,
		TypeStructural:      8,
		TypeConfiguration:   7,
		TypeBehavioral:      6,
		TypePerformance:     5,
		TypeTemporalPattern: 4,
		TypeOwnership:       3,
		TypeTestQuality:     2,
		TypeDocumentation:   1,
	}

	// Simple bubble sort by priority
	sorted := make([]ModificationType, len(types))
	copy(sorted, types)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if priority[sorted[i]] < priority[sorted[j]] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// calculateAggregatedRisk calculates the final risk level from multiple types
// Formula: final_risk = MAX(type_risks) + Σ(other_types × 0.3)
func calculateAggregatedRisk(types []ModificationType) string {
	if len(types) == 0 {
		return ""  // Empty string for no types (consistent with GetBaseRiskLevel)
	}

	// Get primary type risk (highest priority)
	primaryRisk := types[0].GetBaseRiskLevel()

	// If only one type, return its risk
	if len(types) == 1 {
		return primaryRisk
	}

	// Risk level scores for aggregation
	riskScores := map[string]int{
		"CRITICAL":  5,
		"HIGH":      4,
		"MODERATE":  3,
		"LOW":       2,
		"VERY_LOW":  1,
		"UNKNOWN":   0,
	}

	primaryScore := riskScores[primaryRisk]

	// Add 30% of secondary type scores
	secondaryBoost := 0.0
	for i := 1; i < len(types); i++ {
		secondaryRisk := types[i].GetBaseRiskLevel()
		secondaryBoost += float64(riskScores[secondaryRisk]) * 0.3
	}

	finalScore := float64(primaryScore) + secondaryBoost

	// Convert back to risk level
	if finalScore >= 5.0 {
		return "CRITICAL"
	} else if finalScore >= 4.0 {
		return "HIGH"
	} else if finalScore >= 3.0 {
		return "MODERATE"
	} else if finalScore >= 2.0 {
		return "LOW"
	} else {
		return "VERY_LOW"
	}
}

// getKeys extracts keys from a map
func getKeys(m map[ModificationType]string) []ModificationType {
	keys := make([]ModificationType, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
