package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConfidenceAssessment represents the LLM's confidence in risk assessment
type ConfidenceAssessment struct {
	Confidence float64 `json:"confidence"` // 0.0-1.0
	Reasoning  string  `json:"reasoning"`  // Explanation
	NextAction string  `json:"next_action"` // "FINALIZE", "GATHER_MORE_EVIDENCE", "EXPAND_GRAPH"
}

// ConfidencePrompt generates a confidence assessment prompt for the LLM
// 12-factor: Factor 8 - Own your control flow (confidence-based stopping)
func ConfidencePrompt(evidenceChain []string, currentRiskLevel string, hopNumber int) string {
	evidenceList := formatEvidenceList(evidenceChain)

	return fmt.Sprintf(`Based on the evidence gathered so far, assess your confidence in the current risk assessment.

EVIDENCE GATHERED (Hop %d):
%s

CURRENT RISK ASSESSMENT: %s

CONFIDENCE ASSESSMENT TASK:
How confident are you in this risk assessment? (0.0-1.0)

Consider:
1. Do you have enough evidence to make a decisive call?
2. Are there contradicting signals that need resolution?
3. Would gathering more evidence significantly change your assessment?
4. Are there obvious gaps in the investigation?

CONFIDENCE SCALE:
- 0.0-0.4: Very uncertain, need significantly more evidence
- 0.4-0.7: Moderate confidence, could benefit from more data
- 0.7-0.85: Good confidence, additional evidence unlikely to change assessment much
- 0.85-1.0: High confidence, assessment is well-supported and clear

NEXT ACTION OPTIONS:
- "FINALIZE": Confidence is high enough (≥0.85), stop investigation
- "GATHER_MORE_EVIDENCE": Request specific Tier 2 metrics (ownership, incidents, complexity)
- "EXPAND_GRAPH": Explore 2-hop neighbors for broader context

Respond with ONLY a valid JSON object (no markdown, no extra text):
{
  "confidence": 0.85,
  "reasoning": "High coupling (12 deps) + low test coverage (0.25) + security file = clear HIGH risk. Additional evidence unlikely to change assessment.",
  "next_action": "FINALIZE"
}`, hopNumber, evidenceList, currentRiskLevel)
}

// ConfidencePromptWithModificationType adds modification type context to confidence assessment
// Used after Agent 1 Phase 0 integration
func ConfidencePromptWithModificationType(evidenceChain []string, currentRiskLevel string, hopNumber int, modificationType string, modificationReason string) string {
	evidenceList := formatEvidenceList(evidenceChain)

	return fmt.Sprintf(`Based on the evidence gathered so far, assess your confidence in the current risk assessment.

MODIFICATION TYPE: %s
Type Rationale: %s

EVIDENCE GATHERED (Hop %d):
%s

CURRENT RISK ASSESSMENT: %s

CONFIDENCE ASSESSMENT TASK:
How confident are you in this risk assessment? (0.0-1.0)

TYPE-SPECIFIC CONSIDERATIONS:
%s

GENERAL CONSIDERATIONS:
1. Do you have enough evidence to make a decisive call?
2. Are there contradicting signals that need resolution?
3. Would gathering more evidence significantly change your assessment?
4. Are there obvious gaps in the investigation?

CONFIDENCE SCALE:
- 0.0-0.4: Very uncertain, need significantly more evidence
- 0.4-0.7: Moderate confidence, could benefit from more data
- 0.7-0.85: Good confidence, additional evidence unlikely to change assessment much
- 0.85-1.0: High confidence, assessment is well-supported and clear

NEXT ACTION OPTIONS:
- "FINALIZE": Confidence is high enough (≥0.85), stop investigation
- "GATHER_MORE_EVIDENCE": Request specific Tier 2 metrics (ownership, incidents, complexity)
- "EXPAND_GRAPH": Explore 2-hop neighbors for broader context

Respond with ONLY a valid JSON object (no markdown, no extra text):
{
  "confidence": 0.85,
  "reasoning": "Security-sensitive authentication change + coupling + incidents = CRITICAL. Type 9 (Security) base risk confirmed.",
  "next_action": "FINALIZE"
}`, modificationType, modificationReason, hopNumber, evidenceList, currentRiskLevel, getTypeSpecificGuidance(modificationType))
}

// ParseConfidenceAssessment parses the LLM's JSON response
func ParseConfidenceAssessment(response string) (ConfidenceAssessment, error) {
	// Clean up response (remove markdown code blocks if present)
	cleaned := cleanJSONResponse(response)

	var assessment ConfidenceAssessment
	err := json.Unmarshal([]byte(cleaned), &assessment)
	if err != nil {
		return ConfidenceAssessment{}, fmt.Errorf("failed to parse confidence assessment: %w (response: %s)", err, cleaned)
	}

	// Validate confidence is in [0, 1]
	if assessment.Confidence < 0.0 || assessment.Confidence > 1.0 {
		return ConfidenceAssessment{}, fmt.Errorf("confidence must be between 0.0 and 1.0, got %.2f", assessment.Confidence)
	}

	// Validate next action
	validActions := map[string]bool{
		"FINALIZE":              true,
		"GATHER_MORE_EVIDENCE": true,
		"EXPAND_GRAPH":         true,
	}
	if !validActions[assessment.NextAction] {
		return ConfidenceAssessment{}, fmt.Errorf("invalid next_action: %s (must be FINALIZE, GATHER_MORE_EVIDENCE, or EXPAND_GRAPH)", assessment.NextAction)
	}

	return assessment, nil
}

// Helper functions

func formatEvidenceList(evidenceChain []string) string {
	if len(evidenceChain) == 0 {
		return "  (No evidence gathered yet)"
	}

	var builder strings.Builder
	for i, evidence := range evidenceChain {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, evidence))
	}
	return builder.String()
}

func cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Find the first { and last }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 || start > end {
		return response // Return as-is if no JSON found
	}

	return response[start : end+1]
}

func getTypeSpecificGuidance(modificationType string) string {
	guidance := map[string]string{
		"SECURITY": `- Have all authentication/authorization flows been validated?
- Are there tests covering security edge cases?
- Is sensitive data properly protected?
- Are there similar historical incidents?`,

		"INTERFACE": `- Will this break existing API contracts?
- Are there backward compatibility concerns?
- Have all API consumers been identified?
- Is versioning strategy in place?`,

		"CONFIGURATION": `- Is this a production environment configuration?
- Have connection strings/credentials been validated?
- Is there a rollback plan?
- Have downstream services been considered?`,

		"DOCUMENTATION": `- This is documentation-only (zero runtime impact)
- Confidence should be very high (≥0.95)
- FINALIZE immediately unless code changes detected`,

		"STRUCTURAL": `- How many files are affected by this refactoring?
- Are there circular dependency risks?
- Have all import paths been updated?
- Is the test suite comprehensive?`,

		"BEHAVIORAL": `- Does test coverage validate the logic change?
- Are there edge cases not covered?
- What is the cyclomatic complexity?
- Are there similar past incidents?`,

		"TEMPORAL_PATTERN": `- Why is this file a hotspot (high churn)?
- Are there recurring incidents linked to this file?
- Is ownership stable?
- What is the co-change pattern?`,

		"OWNERSHIP": `- Is the new owner familiar with this code?
- How complex is the modified code?
- Should a code owner review be required?
- Are there knowledge transfer gaps?`,

		"PERFORMANCE": `- Have load/performance tests been added?
- Are there potential bottlenecks?
- Is caching/concurrency handled correctly?
- Could this introduce resource leaks?`,

		"TEST_QUALITY": `- Are tests being added or removed?
- Do new tests cover critical paths?
- Are assertions being weakened?
- Is test coverage increasing?`,
	}

	if specific, ok := guidance[modificationType]; ok {
		return specific
	}

	return `- Standard risk assessment considerations apply
- Evaluate based on coupling, coverage, and incidents`
}
