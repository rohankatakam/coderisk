package phase0

import (
	"fmt"
	"time"
)

// Phase0Result contains the results of Phase 0 pre-analysis
// 12-factor: Factor 8 - Own your control flow (explicit routing decision)
type Phase0Result struct {
	// Decision flags
	ForceEscalate bool // True if should skip Phase 1 and go directly to Phase 2
	SkipAnalysis  bool // True if should skip Phase 1/2 entirely (return LOW immediately)

	// Detection results
	ModificationType       ModificationType            // Primary modification type detected
	ModificationTypes      []ModificationType          // All modification types detected
	SecurityResult         SecurityDetectionResult     // Security keyword detection
	DocumentationResult    DocumentationDetectionResult // Documentation-only detection
	EnvironmentResult      EnvironmentDetectionResult  // Environment detection
	ClassificationResult   ModificationClassification  // Full type classification

	// Risk assessment
	AggregatedRisk string // Final risk level after aggregation
	Reason         string // Human-readable explanation of decision

	// Performance
	Duration time.Duration // How long Phase 0 took

	// Metadata
	FilePath string // File being analyzed
}

// RunPhase0 orchestrates all Phase 0 pre-analysis detectors
// This is the main entry point for Phase 0 adaptive pre-analysis
// 12-factor: Factor 3 - Own your context window (fast pre-filtering before expensive analysis)
func RunPhase0(filePath, content string) Phase0Result {
	startTime := time.Now()

	result := Phase0Result{
		FilePath:      filePath,
		ForceEscalate: false,
		SkipAnalysis:  false,
	}

	// Step 1: Check for documentation-only changes (highest priority skip)
	// If documentation, skip all other analysis
	docResult := IsDocumentationOnly(filePath)
	result.DocumentationResult = docResult

	if docResult.IsDocumentationOnly {
		result.SkipAnalysis = true
		result.AggregatedRisk = "LOW"
		result.Reason = fmt.Sprintf("Documentation-only change: %s", docResult.Reason)
		result.Duration = time.Since(startTime)
		return result
	}

	// Step 2: Run all detectors in parallel (conceptually)
	securityResult := DetectSecurityKeywords(filePath, content)
	envResult := DetectEnvironment(filePath)
	classificationResult := ClassifyModification(filePath, content)

	result.SecurityResult = securityResult
	result.EnvironmentResult = envResult
	result.ClassificationResult = classificationResult
	result.ModificationType = classificationResult.PrimaryType
	result.ModificationTypes = classificationResult.AllTypes

	// Step 3: Determine force escalation
	// Security-sensitive changes ALWAYS force escalate to Phase 2
	if securityResult.IsSecuritySensitive && securityResult.ShouldForceEscalate() {
		result.ForceEscalate = true
		result.AggregatedRisk = securityResult.GetRiskLevel()
		result.Reason = fmt.Sprintf("Security-sensitive change detected: %s", securityResult.Reason)
		result.Duration = time.Since(startTime)
		return result
	}

	// Production configuration changes force escalate to HIGH/CRITICAL
	if envResult.IsProduction && envResult.ForceEscalate {
		result.ForceEscalate = true
		result.AggregatedRisk = envResult.GetRiskLevel()
		result.Reason = fmt.Sprintf("Production configuration change: %s", envResult.Reason)
		result.Duration = time.Since(startTime)
		return result
	}

	// Staging configuration changes force escalate to HIGH
	if envResult.Environment == EnvStaging && envResult.ForceEscalate {
		result.ForceEscalate = true
		result.AggregatedRisk = envResult.GetRiskLevel()
		result.Reason = fmt.Sprintf("Staging configuration change: %s", envResult.Reason)
		result.Duration = time.Since(startTime)
		return result
	}

	// Unknown environment configurations force escalate (safety-first)
	if envResult.Environment == EnvUnknown && envResult.IsConfiguration && envResult.ForceEscalate {
		result.ForceEscalate = true
		result.AggregatedRisk = envResult.GetRiskLevel()
		result.Reason = fmt.Sprintf("Unknown environment configuration: %s", envResult.Reason)
		result.Duration = time.Since(startTime)
		return result
	}

	// Step 4: Use classification result for aggregated risk
	// This covers all other modification types
	result.AggregatedRisk = classificationResult.AggregatedRisk

	// Build comprehensive reason
	reasons := []string{}
	if len(classificationResult.AllTypes) > 0 {
		reasons = append(reasons, fmt.Sprintf("Modification type: %s", classificationResult.PrimaryType.String()))
	}
	if len(classificationResult.AllTypes) > 1 {
		secondaryTypes := ""
		for i, t := range classificationResult.SecondaryTypes {
			if i > 0 {
				secondaryTypes += ", "
			}
			secondaryTypes += t.String()
		}
		reasons = append(reasons, fmt.Sprintf("Secondary types: %s", secondaryTypes))
	}
	if classificationResult.AggregatedRisk != "" {
		reasons = append(reasons, fmt.Sprintf("Risk level: %s", classificationResult.AggregatedRisk))
	}

	if len(reasons) > 0 {
		result.Reason = fmt.Sprintf("Phase 0 analysis complete. %s", reasons[0])
		for i := 1; i < len(reasons); i++ {
			result.Reason += fmt.Sprintf("; %s", reasons[i])
		}
	} else {
		result.Reason = "Phase 0 analysis complete. No specific risk indicators detected."
		result.AggregatedRisk = "UNKNOWN"
	}

	result.Duration = time.Since(startTime)
	return result
}

// ShouldSkipPhase1And2 returns true if Phase 1 and Phase 2 should be skipped entirely
func (r Phase0Result) ShouldSkipPhase1And2() bool {
	return r.SkipAnalysis
}

// ShouldSkipPhase1 returns true if Phase 1 should be skipped (but Phase 2 should run)
func (r Phase0Result) ShouldSkipPhase1() bool {
	return r.ForceEscalate
}

// ShouldRunPhase1 returns true if Phase 1 baseline assessment should run
func (r Phase0Result) ShouldRunPhase1() bool {
	return !r.SkipAnalysis && !r.ForceEscalate
}

// GetFinalRiskLevel returns the final risk level from Phase 0
func (r Phase0Result) GetFinalRiskLevel() string {
	if r.AggregatedRisk == "" {
		return "UNKNOWN"
	}
	return r.AggregatedRisk
}

// Summary returns a human-readable summary of the Phase 0 analysis
func (r Phase0Result) Summary() string {
	summary := fmt.Sprintf("Phase 0 Analysis (%s):\n", r.Duration)
	summary += fmt.Sprintf("  File: %s\n", r.FilePath)
	summary += fmt.Sprintf("  Risk Level: %s\n", r.GetFinalRiskLevel())
	summary += fmt.Sprintf("  Skip Analysis: %v\n", r.SkipAnalysis)
	summary += fmt.Sprintf("  Force Escalate: %v\n", r.ForceEscalate)

	if len(r.ModificationTypes) > 0 {
		summary += fmt.Sprintf("  Modification Types: ")
		for i, t := range r.ModificationTypes {
			if i > 0 {
				summary += ", "
			}
			summary += t.String()
		}
		summary += "\n"
	}

	summary += fmt.Sprintf("  Reason: %s\n", r.Reason)

	return summary
}

// IsHighConfidence returns true if Phase 0 is highly confident in its assessment
// High confidence means we can skip Phase 1/2 (either force escalate or skip entirely)
func (r Phase0Result) IsHighConfidence() bool {
	return r.ForceEscalate || r.SkipAnalysis
}

// GetPhaseTransition returns a description of what should happen next
func (r Phase0Result) GetPhaseTransition() string {
	if r.SkipAnalysis {
		return "Skip Phase 1/2 → Return LOW immediately"
	}
	if r.ForceEscalate {
		return fmt.Sprintf("Skip Phase 1 → Escalate to Phase 2 (%s)", r.GetFinalRiskLevel())
	}
	return "Proceed to Phase 1 baseline assessment"
}
