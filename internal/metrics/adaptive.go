package metrics

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// AdaptivePhase1Result extends Phase1Result with adaptive configuration
type AdaptivePhase1Result struct {
	*Phase1Result
	SelectedConfig config.AdaptiveRiskConfig `json:"selected_config"`
	ConfigReason   string                    `json:"config_reason"`
}

// CalculatePhase1WithMultiplePaths performs Phase 1 assessment across multiple historical paths
// This handles file renames/moves by querying ALL historical paths and merging results
func CalculatePhase1WithMultiplePaths(
	ctx context.Context,
	neo4j *graph.Client,
	repoID string,
	filePaths []string,
	riskConfig config.AdaptiveRiskConfig,
) (*AdaptivePhase1Result, error) {
	// Calculate baseline metrics across ALL historical paths (handles renames)
	coupling, err := CalculateCouplingMultiple(ctx, neo4j, repoID, filePaths)
	if err != nil {
		return nil, fmt.Errorf("coupling calculation failed: %w", err)
	}

	coChange, err := CalculateCoChangeMultiple(ctx, neo4j, repoID, filePaths)
	if err != nil {
		return nil, fmt.Errorf("co-change calculation failed: %w", err)
	}

	testRatio, err := CalculateTestRatioMultiple(ctx, neo4j, repoID, filePaths)
	if err != nil {
		return nil, fmt.Errorf("test ratio calculation failed: %w", err)
	}

	// Override risk classifications using adaptive thresholds
	coupling.RiskLevel = ClassifyCouplingWithThreshold(coupling.Count, riskConfig.CouplingThreshold)
	coChange.RiskLevel = ClassifyCoChangeWithThreshold(coChange.MaxFrequency, riskConfig.CoChangeThreshold)
	testRatio.RiskLevel = ClassifyTestRatioWithThreshold(testRatio.Ratio, riskConfig.TestRatioThreshold)

	// Build standard Phase1Result (use first path for display)
	result := &Phase1Result{
		FilePath:  filePaths[0],
		Coupling:  coupling,
		CoChange:  coChange,
		TestRatio: testRatio,
	}

	// Apply adaptive escalation logic
	result.ShouldEscalate = ShouldEscalateWithConfig(result, riskConfig)
	result.OverallRisk = DetermineOverallRiskWithConfig(result)

	// Wrap in adaptive result
	adaptiveResult := &AdaptivePhase1Result{
		Phase1Result:   result,
		SelectedConfig: riskConfig,
	}

	return adaptiveResult, nil
}

// CalculatePhase1WithConfig performs Phase 1 assessment using adaptive thresholds (single path - DEPRECATED)
// Use CalculatePhase1WithMultiplePaths instead to handle renames properly
func CalculatePhase1WithConfig(
	ctx context.Context,
	neo4j *graph.Client,
	repoID string,
	filePath string,
	riskConfig config.AdaptiveRiskConfig,
) (*AdaptivePhase1Result, error) {
	// Delegate to multi-path version
	return CalculatePhase1WithMultiplePaths(ctx, neo4j, repoID, []string{filePath}, riskConfig)
}

// ClassifyCouplingWithThreshold applies domain-specific coupling threshold
func ClassifyCouplingWithThreshold(count int, threshold int) RiskLevel {
	// Low: ≤ 50% of threshold
	// Medium: 50-100% of threshold
	// High: > threshold
	lowThreshold := threshold / 2

	if count <= lowThreshold {
		return RiskLevelLow
	} else if count <= threshold {
		return RiskLevelMedium
	}
	return RiskLevelHigh
}

// ClassifyCoChangeWithThreshold applies domain-specific co-change threshold
func ClassifyCoChangeWithThreshold(frequency float64, threshold float64) RiskLevel {
	// Low: ≤ 50% of threshold
	// Medium: 50-100% of threshold
	// High: > threshold
	lowThreshold := threshold * 0.5

	if frequency <= lowThreshold {
		return RiskLevelLow
	} else if frequency <= threshold {
		return RiskLevelMedium
	}
	return RiskLevelHigh
}

// ClassifyTestRatioWithThreshold applies domain-specific test ratio threshold
// Note: Lower ratio = higher risk (inverse of other metrics)
func ClassifyTestRatioWithThreshold(ratio float64, threshold float64) RiskLevel {
	// High: ratio < threshold (insufficient coverage)
	// Medium: threshold ≤ ratio < (threshold + 0.3)
	// Low: ratio ≥ (threshold + 0.3) (excellent coverage)
	mediumThreshold := threshold + 0.3

	if ratio < threshold {
		return RiskLevelHigh // Insufficient coverage
	} else if ratio < mediumThreshold {
		return RiskLevelMedium // Adequate coverage
	}
	return RiskLevelLow // Excellent coverage
}

// ShouldEscalateWithConfig determines if Phase 2 escalation is needed using adaptive thresholds
// REVISED STRATEGY: Escalate more aggressively when we lack confidence
func ShouldEscalateWithConfig(result *Phase1Result, riskConfig config.AdaptiveRiskConfig) bool {
	// Strategy 1: Escalate if ANY metric exceeds threshold (original logic)
	if result.Coupling != nil && result.Coupling.Count > riskConfig.CouplingThreshold {
		return true
	}
	if result.CoChange != nil && result.CoChange.MaxFrequency > riskConfig.CoChangeThreshold {
		return true
	}
	if result.TestRatio != nil && result.TestRatio.Ratio < riskConfig.TestRatioThreshold {
		return true
	}

	// Strategy 2: Escalate if we have INSUFFICIENT DATA (lack of confidence)
	// Missing data should trigger investigation, not give false confidence
	hasNoData := true
	if result.Coupling != nil && result.Coupling.Count > 0 {
		hasNoData = false
	}
	if result.CoChange != nil && result.CoChange.MaxFrequency > 0 {
		hasNoData = false
	}
	if result.TestRatio != nil && result.TestRatio.Ratio > 0 {
		hasNoData = false
	}

	// If ALL metrics are zero/missing, we have no confidence - escalate
	if hasNoData {
		return true
	}

	// Strategy 3: Lower thresholds for more Phase 2 engagement
	// Even if below threshold, escalate if metrics show ANY risk signals
	if result.Coupling != nil && result.Coupling.Count > 5 {
		return true // 50% of default threshold
	}
	if result.CoChange != nil && result.CoChange.MaxFrequency > 0.4 {
		return true // ~57% of threshold (files changing together 40%+ of time)
	}

	return false
}

// DetermineOverallRiskWithConfig applies adaptive risk aggregation logic
func DetermineOverallRiskWithConfig(result *Phase1Result) RiskLevel {
	if result.ShouldEscalate {
		return RiskLevelHigh
	}

	// If no escalation, use highest individual risk level
	return result.aggregateRiskLevel()
}

// FormatSummaryWithConfig generates summary including config information
func (a *AdaptivePhase1Result) FormatSummaryWithConfig() string {
	summary := fmt.Sprintf("File: %s\n", a.FilePath)
	summary += fmt.Sprintf("Configuration: %s (%s)\n", a.SelectedConfig.ConfigKey, a.SelectedConfig.Description)
	summary += fmt.Sprintf("Overall Risk: %s\n", a.OverallRisk)
	summary += fmt.Sprintf("Phase 2 Escalation: %v\n\n", a.ShouldEscalate)

	summary += "Thresholds (Adaptive):\n"
	summary += fmt.Sprintf("  • Coupling Threshold: %d\n", a.SelectedConfig.CouplingThreshold)
	summary += fmt.Sprintf("  • Co-Change Threshold: %.2f\n", a.SelectedConfig.CoChangeThreshold)
	summary += fmt.Sprintf("  • Test Ratio Threshold: %.2f\n\n", a.SelectedConfig.TestRatioThreshold)

	summary += "Evidence (Tier 1 Metrics):\n"

	if a.Coupling != nil {
		summary += fmt.Sprintf("  • Coupling: %s", a.Coupling.FormatEvidence())
		if a.Coupling.Count > a.SelectedConfig.CouplingThreshold {
			summary += fmt.Sprintf(" [EXCEEDS threshold of %d]", a.SelectedConfig.CouplingThreshold)
		}
		summary += "\n"
	}

	if a.CoChange != nil {
		summary += fmt.Sprintf("  • Co-Change: %s", a.CoChange.FormatEvidence())
		if a.CoChange.MaxFrequency > a.SelectedConfig.CoChangeThreshold {
			summary += fmt.Sprintf(" [EXCEEDS threshold of %.2f]", a.SelectedConfig.CoChangeThreshold)
		}
		summary += "\n"
	}

	if a.TestRatio != nil {
		summary += fmt.Sprintf("  • Test Coverage: %s", a.TestRatio.FormatEvidence())
		if a.TestRatio.Ratio < a.SelectedConfig.TestRatioThreshold {
			summary += fmt.Sprintf(" [BELOW threshold of %.2f]", a.SelectedConfig.TestRatioThreshold)
		}
		summary += "\n"
	}

	if a.ConfigReason != "" {
		summary += fmt.Sprintf("\nConfig Selection: %s\n", a.ConfigReason)
	}

	summary += fmt.Sprintf("\nDuration: %dms\n", a.DurationMS)
	return summary
}
