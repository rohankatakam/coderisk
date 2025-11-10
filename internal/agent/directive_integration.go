package agent

import (
	"context"
	"fmt"
)

// DirectiveDecision represents a decision point in the investigation
type DirectiveDecision struct {
	ShouldPause  bool
	Directive    *DirectiveMessage
	DecisionNum  int
}

// CheckForDirectiveNeeded determines if investigation should pause for human decision
// Based on 12-factor agent principles: Factor #6 (Launch/Pause/Resume) and Factor #7 (Contact Humans)
func CheckForDirectiveNeeded(
	assessment *RiskAssessment,
	filePath string,
	missingData map[string]bool,
) *DirectiveDecision {
	// Check for missing co-change data
	if missingData != nil && missingData["cochange"] {
		return &DirectiveDecision{
			ShouldPause:  true,
			Directive:    buildMissingDataDirective(filePath),
			DecisionNum:  1,
		}
	}

	// Check for MEDIUM risk with moderate confidence
	if assessment.RiskLevel == RiskMedium && assessment.Confidence < 0.75 {
		return &DirectiveDecision{
			ShouldPause:  true,
			Directive:    buildUncertainRiskDirective(assessment, filePath),
			DecisionNum:  1,
		}
	}

	// Check for HIGH/CRITICAL risk
	if assessment.RiskLevel == RiskHigh || assessment.RiskLevel == RiskCritical {
		return &DirectiveDecision{
			ShouldPause:  true,
			Directive:    buildHighRiskDirective(assessment, filePath),
			DecisionNum:  1,
		}
	}

	// No directive needed - investigation can complete
	return &DirectiveDecision{
		ShouldPause:  false,
		Directive:    nil,
		DecisionNum:  0,
	}
}

// buildMissingDataDirective creates a directive for missing co-change data
func buildMissingDataDirective(filePath string) *DirectiveMessage {
	evidence := []DirectiveEvidence{
		{
			Type:   "missing_data",
			Data:   "Co-change partner query failed - unable to verify related files",
			Source: "Phase 2 investigation",
		},
	}

	return BuildDeepInvestigationDirective(
		"1-2 minutes",
		0.02,
		fmt.Sprintf("Unable to verify co-change patterns for %s. This is a critical risk signal that detects incomplete changes.", filePath),
		evidence,
	)
}

// buildUncertainRiskDirective creates a directive for uncertain risk assessment
func buildUncertainRiskDirective(assessment *RiskAssessment, filePath string) *DirectiveMessage {
	evidence := []DirectiveEvidence{
		{
			Type:   "risk_assessment",
			Data:   fmt.Sprintf("Risk: %s, Confidence: %.0f%%", assessment.RiskLevel, assessment.Confidence*100),
			Source: "Phase 2 investigation",
		},
		{
			Type:   "summary",
			Data:   assessment.Summary,
			Source: "LLM analysis",
		},
	}

	return BuildDeepInvestigationDirective(
		"2-3 minutes",
		0.05,
		fmt.Sprintf("Risk assessment for %s shows MEDIUM risk with %d%% confidence. Uncertainty suggests deeper investigation may be valuable.", filePath, int(assessment.Confidence*100)),
		evidence,
	)
}

// buildHighRiskDirective creates a directive for high risk files
func buildHighRiskDirective(assessment *RiskAssessment, filePath string) *DirectiveMessage {
	evidence := []DirectiveEvidence{
		{
			Type:   "risk_assessment",
			Data:   fmt.Sprintf("Risk: %s, Confidence: %.0f%%", assessment.RiskLevel, assessment.Confidence*100),
			Source: "Phase 2 investigation",
		},
		{
			Type:   "summary",
			Data:   assessment.Summary,
			Source: "LLM analysis",
		},
	}

	// Extract recommendations as evidence
	for _, rec := range assessment.Recommendations {
		evidence = append(evidence, DirectiveEvidence{
			Type:   "recommendation",
			Data:   rec,
			Source: "Phase 2 analysis",
		})
	}

	return BuildEscalationDirective(
		fmt.Sprintf("File %s is %s risk - requires manual review", filePath, assessment.RiskLevel),
		"Do not commit until risks are addressed. Consider: code review with senior developer, additional tests, gradual rollout strategy.",
		evidence,
	)
}

// HandleUserChoice processes user's choice and returns next action
func HandleUserChoice(choice string, directive *DirectiveMessage) (shouldContinue bool, shouldAbort bool) {
	switch choice {
	case "a", "approve":
		// User approved - continue with investigation or accept result
		return true, false
	case "s", "skip":
		// User skipped - continue but flag as higher risk
		return true, false
	case "x", "abort":
		// User aborted - stop investigation
		return false, true
	case "e", "edit", "modify":
		// User wants to modify - for now, treat as skip
		// TODO: Implement message editing flow
		return true, false
	default:
		// Invalid choice - ask again (caller should loop)
		return false, false
	}
}

// CheckpointSaver interface for saving investigation state
type CheckpointSaver interface {
	Save(ctx context.Context, investigation *DirectiveInvestigation) (string, error)
}
