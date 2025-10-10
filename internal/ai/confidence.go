package ai

import (
	"github.com/coderisk/coderisk-go/internal/models"
)

// ConfidenceCalculator determines auto-fix confidence scores
type ConfidenceCalculator struct{}

// NewConfidenceCalculator creates a new confidence calculator
func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{}
}

// Calculate computes confidence score (0.0-1.0) for auto-fixing an issue
// Threshold: >0.85 = ready_to_execute (safe to auto-fix)
func (c *ConfidenceCalculator) Calculate(issue models.RiskIssue, file models.FileRisk) float64 {
	// Start with base confidence by fix type
	baseConfidence := c.baseConfidenceByType(issue.FixType)

	// Apply adjustments based on context
	complexityPenalty := c.complexityPenalty(file.Metrics)
	languageBonus := c.languageBonus(file.Language)
	severityAdjustment := c.severityAdjustment(issue.Severity)

	// Calculate final confidence
	confidence := baseConfidence + complexityPenalty + languageBonus + severityAdjustment

	// Clamp to [0.0, 1.0]
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// baseConfidenceByType returns base confidence for each fix type
func (c *ConfidenceCalculator) baseConfidenceByType(fixType string) float64 {
	// Base confidence by fix type (based on task clarity and risk)
	baseScores := map[string]float64{
		"generate_tests":     0.92, // High confidence (well-defined task)
		"add_error_handling": 0.85, // Medium-high (straightforward patterns)
		"fix_security":       0.80, // Medium (requires domain knowledge)
		"reduce_coupling":    0.65, // Low-medium (architectural decision)
		"reduce_complexity":  0.60, // Low (subjective refactoring)
	}

	if score, ok := baseScores[fixType]; ok {
		return score
	}
	return 0.50 // Default: requires review
}

// complexityPenalty reduces confidence for high complexity code
func (c *ConfidenceCalculator) complexityPenalty(metrics map[string]models.Metric) float64 {
	// Check cyclomatic complexity
	if complexity, ok := metrics["complexity"]; ok {
		complexityValue := complexity.Value

		// Penalty for high complexity
		if complexityValue > 15 {
			return -0.15 // Very high complexity = lower confidence
		} else if complexityValue > 10 {
			return -0.08 // High complexity = moderate penalty
		} else if complexityValue > 7 {
			return -0.03 // Medium complexity = slight penalty
		}
	}

	return 0.0 // No penalty for low complexity
}

// languageBonus increases confidence for well-supported languages
func (c *ConfidenceCalculator) languageBonus(language string) float64 {
	// Higher confidence for well-supported languages with mature tooling
	supportedLanguages := map[string]float64{
		"python":     0.05,
		"javascript": 0.05,
		"typescript": 0.06, // TypeScript gets extra bonus (type safety helps AI)
		"go":         0.04,
		"java":       0.04,
		"ruby":       0.03,
		"rust":       0.03,
	}

	if bonus, ok := supportedLanguages[language]; ok {
		return bonus
	}
	return -0.05 // Lower confidence for less common languages
}

// severityAdjustment modifies confidence based on issue severity
func (c *ConfidenceCalculator) severityAdjustment(severity string) float64 {
	// Higher severity = need more confidence
	switch severity {
	case "critical":
		return -0.10 // Critical issues need human review
	case "high":
		return -0.05 // High severity = be more conservative
	case "medium":
		return 0.00 // No adjustment
	case "low":
		return 0.05 // Low severity = can be more aggressive
	default:
		return 0.00
	}
}

// ShouldAutoFix determines if confidence is high enough for auto-fix
func (c *ConfidenceCalculator) ShouldAutoFix(confidence float64) bool {
	return confidence > 0.85
}

// GetConfidenceLevel returns a human-readable confidence level
func (c *ConfidenceCalculator) GetConfidenceLevel(confidence float64) string {
	if confidence > 0.90 {
		return "very_high"
	} else if confidence > 0.85 {
		return "high"
	} else if confidence > 0.70 {
		return "medium"
	} else if confidence > 0.50 {
		return "low"
	} else {
		return "very_low"
	}
}

// EstimateFixTime estimates time to fix in minutes based on fix type and complexity
func (c *ConfidenceCalculator) EstimateFixTime(fixType string, complexity float64) int {
	baseTime := map[string]int{
		"generate_tests":     30,  // 30 min for test generation
		"add_error_handling": 15,  // 15 min for error handling
		"fix_security":       45,  // 45 min for security fixes
		"reduce_coupling":    120, // 2 hours for architectural changes
		"reduce_complexity":  90,  // 1.5 hours for refactoring
	}

	base, ok := baseTime[fixType]
	if !ok {
		base = 30 // Default estimate
	}

	// Adjust for complexity
	if complexity > 15 {
		return int(float64(base) * 2.0) // Double time for very complex code
	} else if complexity > 10 {
		return int(float64(base) * 1.5) // 1.5x time for complex code
	} else if complexity > 7 {
		return int(float64(base) * 1.2) // 1.2x time for moderately complex code
	}

	return base
}

// EstimateLines estimates lines of code that will be added/changed
func (c *ConfidenceCalculator) EstimateLines(fixType string) int {
	estimates := map[string]int{
		"generate_tests":     80, // Typical test file
		"add_error_handling": 10, // Try-catch blocks
		"fix_security":       20, // Security validation
		"reduce_coupling":    50, // Interface + refactoring
		"reduce_complexity":  30, // Code reorganization
	}

	if est, ok := estimates[fixType]; ok {
		return est
	}
	return 20 // Default estimate
}

// DetermineExpectedFiles returns files that will be created/modified
func (c *ConfidenceCalculator) DetermineExpectedFiles(fixType string, originalFile string, language string) []string {
	switch fixType {
	case "generate_tests":
		// Will create test file
		testFile := c.getTestFileName(originalFile, language)
		return []string{testFile}
	case "add_error_handling", "fix_security", "reduce_complexity":
		// Will modify original file
		return []string{originalFile}
	case "reduce_coupling":
		// Might create new interface file + modify original
		return []string{originalFile, "new_interface_file"}
	default:
		return []string{originalFile}
	}
}

// getTestFileName determines test file name based on language conventions
func (c *ConfidenceCalculator) getTestFileName(originalFile string, language string) string {
	// Use same logic as PromptGenerator
	pg := NewPromptGenerator()
	return pg.testFileName(originalFile, language)
}
