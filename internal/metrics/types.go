package metrics

import "fmt"

// RiskLevel represents the risk classification for a metric
// Reference: risk_assessment_methodology.md - Threshold logic across all metrics
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "LOW"
	RiskLevelMedium RiskLevel = "MEDIUM"
	RiskLevelHigh   RiskLevel = "HIGH"
)

// String returns the string representation of RiskLevel
func (r RiskLevel) String() string {
	return string(r)
}

// Phase1Result aggregates all Tier 1 metric results
// Reference: risk_assessment_methodology.md §2.4 - Phase 1 Heuristic
type Phase1Result struct {
	FilePath       string           `json:"file_path"`
	OverallRisk    RiskLevel        `json:"overall_risk"`
	ShouldEscalate bool             `json:"should_escalate"`
	Coupling       *CouplingResult  `json:"coupling"`
	CoChange       *CoChangeResult  `json:"co_change"`
	TestRatio      *TestRatioResult `json:"test_ratio"`
	DurationMS     int64            `json:"duration_ms"`
}

// DetermineOverallRisk applies Phase 1 heuristic logic
// Reference: risk_assessment_methodology.md §2.4 - Decision tree
// Logic: IF (coupling > 10) OR (co_change > 0.7) OR (test_ratio < 0.3) THEN HIGH
func (p *Phase1Result) DetermineOverallRisk() {
	// Check escalation conditions (OR logic)
	shouldEscalate := false

	if p.Coupling != nil && p.Coupling.ShouldEscalate() {
		shouldEscalate = true
	}
	if p.CoChange != nil && p.CoChange.ShouldEscalate() {
		shouldEscalate = true
	}
	if p.TestRatio != nil && p.TestRatio.ShouldEscalate() {
		shouldEscalate = true
	}

	p.ShouldEscalate = shouldEscalate

	if shouldEscalate {
		p.OverallRisk = RiskLevelHigh
	} else {
		// If no escalation, use highest individual risk
		p.OverallRisk = p.aggregateRiskLevel()
	}
}

// aggregateRiskLevel finds the highest risk level among metrics
func (p *Phase1Result) aggregateRiskLevel() RiskLevel {
	highest := RiskLevelLow

	if p.Coupling != nil && p.Coupling.RiskLevel == RiskLevelHigh {
		return RiskLevelHigh
	}
	if p.CoChange != nil && p.CoChange.RiskLevel == RiskLevelHigh {
		return RiskLevelHigh
	}
	if p.TestRatio != nil && p.TestRatio.RiskLevel == RiskLevelHigh {
		return RiskLevelHigh
	}

	if p.Coupling != nil && p.Coupling.RiskLevel == RiskLevelMedium {
		highest = RiskLevelMedium
	}
	if p.CoChange != nil && p.CoChange.RiskLevel == RiskLevelMedium {
		highest = RiskLevelMedium
	}
	if p.TestRatio != nil && p.TestRatio.RiskLevel == RiskLevelMedium {
		highest = RiskLevelMedium
	}

	return highest
}

// FormatSummary generates human-readable summary
func (p *Phase1Result) FormatSummary() string {
	summary := fmt.Sprintf("File: %s\n", p.FilePath)
	summary += fmt.Sprintf("Overall Risk: %s\n", p.OverallRisk)
	summary += fmt.Sprintf("Phase 2 Escalation: %v\n\n", p.ShouldEscalate)
	summary += "Evidence (Tier 1 Metrics):\n"

	if p.Coupling != nil {
		summary += fmt.Sprintf("  • Coupling: %s\n", p.Coupling.FormatEvidence())
	}
	if p.CoChange != nil {
		summary += fmt.Sprintf("  • Co-Change: %s\n", p.CoChange.FormatEvidence())
	}
	if p.TestRatio != nil {
		summary += fmt.Sprintf("  • Test Coverage: %s\n", p.TestRatio.FormatEvidence())
	}

	summary += fmt.Sprintf("\nDuration: %dms\n", p.DurationMS)
	return summary
}
