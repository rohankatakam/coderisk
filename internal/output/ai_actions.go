package output

import (
	"fmt"
	"sort"

	"github.com/rohankatakam/coderisk/internal/metrics"
	"github.com/rohankatakam/coderisk/internal/models"
)

// GenerateAIActions creates actionable prompts for AI coding assistants
// 12-factor: Factor 4 - Tools are structured outputs
func GenerateAIActions(phase1 *metrics.Phase1Result, riskResult *models.RiskResult) []AIAssistantAction {
	var actions []AIAssistantAction

	// Generate actions based on detected issues
	for _, issue := range riskResult.Issues {
		switch issue.Category {
		case "quality":
			// Low test coverage issue
			actions = append(actions, generateTestAction(riskResult, issue))

		case "coupling":
			// High coupling issue
			actions = append(actions, generateDecouplingAction(riskResult, issue))

		case "temporal":
			// Temporal coupling issue
			actions = append(actions, generateTemporalAnalysisAction(riskResult, issue))
		}
	}

	// Sort by confidence (highest first), then by risk reduction
	sortActionsByConfidence(actions)

	return actions
}

func generateTestAction(result *models.RiskResult, issue models.RiskIssue) AIAssistantAction {
	// Extract coverage from metrics if available
	coverage := 0.0
	for _, file := range result.Files {
		if file.Path == issue.File {
			if metric, ok := file.Metrics["test_coverage"]; ok {
				coverage = metric.Value
			}
			break
		}
	}

	targetCoverage := 80.0
	currentCoverage := coverage * 100.0

	return AIAssistantAction{
		ActionType:     "add_test",
		Confidence:     0.9, // High confidence - test addition is straightforward
		Description:    fmt.Sprintf("Add unit tests to improve coverage from %.1f%% to %.0f%%", currentCoverage, targetCoverage),
		EstimatedLines: 50, // Rough estimate for test file
		FilePath:       issue.File,
		Prompt: fmt.Sprintf(`Add comprehensive unit tests for %s to achieve %.0f%% coverage.

Current coverage: %.1f%%
Focus on:
- Edge cases and error conditions
- Critical business logic paths
- Input validation

Generate tests using the project's existing test framework.`, issue.File, targetCoverage, currentCoverage),
		ReadyToExecute: true,
		RiskReduction:  0.3, // Tests reduce risk by ~30%
	}
}

func generateDecouplingAction(result *models.RiskResult, issue models.RiskIssue) AIAssistantAction {
	// Extract coupling count from metrics
	couplingCount := 0.0
	for _, file := range result.Files {
		if file.Path == issue.File {
			if metric, ok := file.Metrics["coupling"]; ok {
				couplingCount = metric.Value
			}
			break
		}
	}

	return AIAssistantAction{
		ActionType:     "refactor",
		Confidence:     0.7, // Medium confidence - refactoring needs human review
		Description:    fmt.Sprintf("Reduce coupling from %.0f dependencies to <10", couplingCount),
		EstimatedLines: 50, // Typical refactor size
		FilePath:       issue.File,
		Prompt: fmt.Sprintf(`Refactor %s to reduce coupling from %.0f dependencies.

Techniques to consider:
- Extract interfaces for external dependencies
- Apply dependency injection pattern
- Move business logic to separate modules
- Use facade pattern to simplify dependencies

Maintain existing behavior and test coverage.`, issue.File, couplingCount),
		ReadyToExecute: false, // Requires human review
		RiskReduction:  0.4,   // Decoupling significantly reduces risk
	}
}

func generateTemporalAnalysisAction(result *models.RiskResult, issue models.RiskIssue) AIAssistantAction {
	// Extract co-change frequency from metrics
	coChangeFreq := 0.0
	for _, file := range result.Files {
		if file.Path == issue.File {
			if metric, ok := file.Metrics["co_change"]; ok {
				coChangeFreq = metric.Value
			}
			break
		}
	}

	return AIAssistantAction{
		ActionType:     "investigate_coupling",
		Confidence:     0.65,
		Description:    fmt.Sprintf("Investigate temporal coupling (%.1f%% co-change frequency)", coChangeFreq*100),
		EstimatedLines: 20,
		FilePath:       issue.File,
		Prompt: fmt.Sprintf(`Investigate temporal coupling for %s (co-change frequency: %.1f%%).

Files that change together may indicate:
- Shared abstractions that should be extracted
- Hidden dependencies that should be made explicit
- Opportunity to consolidate related functionality

Review the files that frequently change with this one and consider:
1. Are they logically related?
2. Should they be in the same module?
3. Is there a better way to organize this code?`, issue.File, coChangeFreq*100),
		ReadyToExecute: false, // Investigation requires human judgment
		RiskReduction:  0.25,
	}
}

func sortActionsByConfidence(actions []AIAssistantAction) {
	// Sort descending by confidence, then by risk reduction
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Confidence != actions[j].Confidence {
			return actions[i].Confidence > actions[j].Confidence
		}
		return actions[i].RiskReduction > actions[j].RiskReduction
	})
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minFloat64 helper function
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat64 helper function
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
