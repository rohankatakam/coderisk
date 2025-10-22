package agent

import (
	"context"
	"fmt"
	"strings"
)

// Synthesizer generates final risk assessment
type Synthesizer struct {
	llm LLMClientInterface
}

// NewSynthesizer creates a new synthesizer
func NewSynthesizer(llm LLMClientInterface) *Synthesizer {
	return &Synthesizer{llm: llm}
}

// Synthesize generates final risk summary from investigation
func (s *Synthesizer) Synthesize(ctx context.Context, inv Investigation) (RiskAssessment, error) {
	// Build synthesis prompt
	prompt := s.buildSynthesisPrompt(inv)

	// Query LLM for final summary
	summary, tokens, err := s.llm.Query(ctx, prompt)
	if err != nil {
		return RiskAssessment{}, fmt.Errorf("synthesis failed: %w", err)
	}

	inv.TotalTokens += tokens

	// Determine risk level from score
	riskLevel := s.scoreToRiskLevel(inv.RiskScore)

	return RiskAssessment{
		FilePath:      inv.Request.FilePath,
		RiskLevel:     riskLevel,
		RiskScore:     inv.RiskScore,
		Confidence:    inv.Confidence,
		Summary:       strings.TrimSpace(summary),
		Evidence:      inv.Evidence,
		Investigation: &inv,
	}, nil
}

// buildSynthesisPrompt constructs the final synthesis prompt
func (s *Synthesizer) buildSynthesisPrompt(inv Investigation) string {
	// Summarize evidence
	var evidenceSummary strings.Builder
	for i, e := range inv.Evidence {
		evidenceSummary.WriteString(fmt.Sprintf("%d. [%s] %s (severity: %.2f)\n", i+1, e.Type, e.Description, e.Severity))
	}

	// Summarize hops
	var hopsSummary strings.Builder
	for _, hop := range inv.Hops {
		hopsSummary.WriteString(fmt.Sprintf("Hop %d: %s\n", hop.HopNumber, hop.Response))
	}

	return fmt.Sprintf(`You are a code risk assessment AI. Synthesize the investigation findings into a concise risk summary.

File: %s
Change type: %s
Risk score: %.2f

Evidence gathered:
%s

Investigation hops:
%s

Task: Write a 2-3 sentence risk summary for a developer explaining:
1. The primary risk factor
2. Why it matters (impact)
3. What to check before merging

Format: Direct, actionable, no fluff. Be specific about the evidence.`,
		inv.Request.FilePath,
		inv.Request.ChangeType,
		inv.RiskScore,
		evidenceSummary.String(),
		hopsSummary.String(),
	)
}

// scoreToRiskLevel maps numeric score to risk level
func (s *Synthesizer) scoreToRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 0.8:
		return RiskCritical
	case score >= 0.6:
		return RiskHigh
	case score >= 0.4:
		return RiskMedium
	case score >= 0.2:
		return RiskLow
	default:
		return RiskMinimal
	}
}
