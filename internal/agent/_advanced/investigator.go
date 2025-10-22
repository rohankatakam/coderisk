package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Investigator orchestrates the full investigation
type Investigator struct {
	llm         LLMClientInterface
	navigator   *HopNavigator
	collector   *EvidenceCollector
	synthesizer *Synthesizer
}

// NewInvestigator creates a new investigator with confidence-driven navigation
func NewInvestigator(llm LLMClientInterface, temporal TemporalClient, incidents IncidentsClient, graph GraphClient) *Investigator {
	navigator := NewHopNavigator(llm, graph, 5) // Max 5 hops (up from 3)
	collector := NewEvidenceCollector(temporal, incidents, graph)
	synthesizer := NewSynthesizer(llm)

	return &Investigator{
		llm:         llm,
		navigator:   navigator,
		collector:   collector,
		synthesizer: synthesizer,
	}
}

// Investigate performs full Phase 2 investigation (PUBLIC - cmd/crisk/check.go uses this)
func (inv *Investigator) Investigate(ctx context.Context, req InvestigationRequest) (RiskAssessment, error) {
	// Set request metadata
	req.RequestID = uuid.New()
	req.StartedAt = time.Now()

	// Step 1: Collect evidence
	evidence, err := inv.collector.Collect(ctx, req.FilePath)
	if err != nil {
		// Don't fail the entire investigation if evidence collection fails
		// Log the error and continue with empty evidence
		evidence = []Evidence{}
	}

	// Step 2: Calculate initial risk score
	riskScore := inv.collector.Score(evidence)

	// Step 3: Navigate graph (hop-by-hop)
	hops, err := inv.navigator.Navigate(ctx, req)
	if err != nil {
		return RiskAssessment{}, fmt.Errorf("navigation failed: %w", err)
	}

	// Step 4: Build investigation object with confidence history and breakthroughs
	finalConfidence := inv.calculateConfidence(hops, evidence)
	if len(hops) > 0 && hops[len(hops)-1].Confidence > 0 {
		// Use confidence from last hop if available and valid
		finalConfidence = hops[len(hops)-1].Confidence
	}

	investigation := Investigation{
		Request:           req,
		Hops:              hops,
		Evidence:          evidence,
		RiskScore:         riskScore,
		Confidence:        finalConfidence,
		ConfidenceHistory: inv.navigator.GetConfidenceHistory(),
		Breakthroughs:     inv.navigator.GetBreakthroughs(),
		StoppingReason:    inv.navigator.GetStoppingReason(hops),
		CompletedAt:       time.Now(),
		TotalTokens:       sumTokens(hops),
	}

	// Step 5: Synthesize final assessment
	assessment, err := inv.synthesizer.Synthesize(ctx, investigation)
	if err != nil {
		return RiskAssessment{}, fmt.Errorf("synthesis failed: %w", err)
	}

	return assessment, nil
}

// calculateConfidence estimates confidence based on evidence quality
func (inv *Investigator) calculateConfidence(hops []HopResult, evidence []Evidence) float64 {
	// More evidence + more hops = higher confidence
	// Evidence score: 0.1 per piece of evidence (max 10 pieces = 1.0)
	evidenceScore := float64(len(evidence)) / 10.0
	if evidenceScore > 1.0 {
		evidenceScore = 1.0
	}

	// Hop score: 0.33 per hop (max 3 hops = 1.0)
	hopScore := float64(len(hops)) / 3.0
	if hopScore > 1.0 {
		hopScore = 1.0
	}

	// Weighted combination: evidence (60%), hops (40%)
	confidence := evidenceScore*0.6 + hopScore*0.4

	// Ensure confidence is in [0, 1]
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

func sumTokens(hops []HopResult) int {
	total := 0
	for _, hop := range hops {
		total += hop.TokensUsed
	}
	return total
}
