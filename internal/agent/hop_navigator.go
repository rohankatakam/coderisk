package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/agent/prompts"
)

// LLMClientInterface defines the interface for LLM clients
type LLMClientInterface interface {
	Query(ctx context.Context, prompt string) (string, int, error)
	SetModel(model string)
}

// HopNavigator performs hop-by-hop graph traversal with confidence-driven stopping
// 12-factor: Factor 8 - Own your control flow (adaptive investigation depth)
type HopNavigator struct {
	llm                  LLMClientInterface
	graph                GraphClient
	maxHops              int     // Maximum hops allowed (default: 5)
	maxTokens            int     // Token budget
	confidenceThreshold  float64 // Stop when confidence >= this (default: 0.85)
	breakthroughTracker  *BreakthoughTracker
	confidenceHistory    []ConfidencePoint
	evidenceChain        []string // Accumulated evidence descriptions
}

// NewHopNavigator creates a new hop navigator with confidence-driven stopping
func NewHopNavigator(llm LLMClientInterface, graph GraphClient, maxHops int) *HopNavigator {
	if maxHops <= 0 {
		maxHops = 5 // Default max hops
	}
	return &HopNavigator{
		llm:                 llm,
		graph:               graph,
		maxHops:             maxHops,
		maxTokens:           10000, // Budget for entire investigation
		confidenceThreshold: 0.85,  // Stop when 85% confident
		confidenceHistory:   []ConfidencePoint{},
		evidenceChain:       []string{},
	}
}

// SetConfidenceThreshold allows customizing the stopping threshold
func (n *HopNavigator) SetConfidenceThreshold(threshold float64) {
	n.confidenceThreshold = threshold
}

// Navigate performs confidence-driven multi-hop investigation
// Stops when confidence >= threshold OR max hops reached
func (n *HopNavigator) Navigate(ctx context.Context, req InvestigationRequest) ([]HopResult, error) {
	var hops []HopResult
	totalTokens := 0

	// Initialize breakthrough tracker with baseline risk
	initialRiskScore := calculateInitialRiskScore(req.Baseline)
	initialRiskLevel := ScoreToRiskLevel(initialRiskScore)
	n.breakthroughTracker = NewBreakthroughTracker(initialRiskScore, initialRiskLevel)

	// Initialize evidence chain with baseline
	n.evidenceChain = []string{
		fmt.Sprintf("Coupling: %.2f", req.Baseline.CouplingScore),
		fmt.Sprintf("Co-change frequency: %.2f", req.Baseline.CoChangeFrequency),
		fmt.Sprintf("Past incidents: %d", req.Baseline.IncidentCount),
	}
	if req.Baseline.OwnershipDays > 0 {
		n.evidenceChain = append(n.evidenceChain,
			fmt.Sprintf("Ownership transition: %d days ago", req.Baseline.OwnershipDays))
	}

	currentRiskScore := initialRiskScore
	currentRiskLevel := initialRiskLevel

	// Confidence-driven loop: Continue until confidence >= threshold OR max hops
	for hopNum := 1; hopNum <= n.maxHops; hopNum++ {
		// Check token budget
		if totalTokens >= n.maxTokens {
			n.recordConfidence(hopNum-1, 0.0, currentRiskScore, currentRiskLevel,
				"Token budget exhausted", "FINALIZE")
			break
		}

		// Execute hop
		hop, err := n.executeHop(ctx, hopNum, req, hops, currentRiskScore, currentRiskLevel)
		if err != nil {
			// Non-fatal error, finalize with what we have
			if len(hops) == 0 {
				return nil, fmt.Errorf("hop %d failed: %w", hopNum, err)
			}
			break
		}

		hops = append(hops, hop)
		totalTokens += hop.TokensUsed

		// Assess confidence after this hop (with modification type context if available)
		confidenceAssessment, err := n.assessConfidence(ctx, hopNum, currentRiskLevel,
			req.ModificationType, req.ModificationReason)
		if err != nil {
			// If confidence assessment fails, continue with heuristic
			// Use old logic as fallback
			if n.shouldStopEarlyFallback(hops) {
				hop.Confidence = 0.9
				hop.NextAction = "FINALIZE"
				n.recordConfidence(hopNum, 0.9, currentRiskScore, currentRiskLevel,
					"Early stop heuristic (fallback)", "FINALIZE")
				break
			}
			hop.Confidence = 0.5
			hop.NextAction = "GATHER_MORE_EVIDENCE"
			continue
		}

		// Update hop with confidence assessment
		hop.Confidence = confidenceAssessment.Confidence
		hop.NextAction = confidenceAssessment.NextAction
		hops[len(hops)-1] = hop // Update the hop in the slice

		// Record confidence point
		n.recordConfidence(hopNum, confidenceAssessment.Confidence, currentRiskScore,
			currentRiskLevel, confidenceAssessment.Reasoning, confidenceAssessment.NextAction)

		// Check for breakthrough (risk level change)
		// Parse new risk score from LLM response (simplified - would be more sophisticated)
		newRiskScore := currentRiskScore // TODO: Extract from LLM response
		newRiskLevel := ScoreToRiskLevel(newRiskScore)

		if newRiskLevel != currentRiskLevel {
			n.breakthroughTracker.CheckAndRecordBreakthrough(
				hopNum,
				newRiskScore,
				newRiskLevel,
				hop.Response[:min(100, len(hop.Response))], // First 100 chars as trigger
				confidenceAssessment.Reasoning,
			)
			currentRiskScore = newRiskScore
			currentRiskLevel = newRiskLevel
		}

		// Decision: Should we stop?
		if confidenceAssessment.Confidence >= n.confidenceThreshold {
			// High confidence, stop investigation
			break
		}

		if confidenceAssessment.NextAction == "FINALIZE" {
			// LLM explicitly requested finalization
			break
		}

		// Continue to next hop
	}

	return hops, nil
}

// Helper to calculate initial risk score from baseline metrics
func calculateInitialRiskScore(baseline BaselineMetrics) float64 {
	// Weighted combination of baseline metrics
	score := baseline.CouplingScore*0.4 +
		baseline.CoChangeFrequency*0.3

	// Incident penalty (0.1 per incident, max 0.3)
	incidentPenalty := float64(baseline.IncidentCount) * 0.1
	if incidentPenalty > 0.3 {
		incidentPenalty = 0.3
	}
	score += incidentPenalty

	// Ensure in [0, 1]
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// recordConfidence records a confidence point in history
func (n *HopNavigator) recordConfidence(hopNum int, confidence, riskScore float64,
	riskLevel RiskLevel, reasoning, nextAction string) {
	point := ConfidencePoint{
		HopNumber:  hopNum,
		Confidence: confidence,
		RiskScore:  riskScore,
		RiskLevel:  riskLevel,
		Reasoning:  reasoning,
		NextAction: nextAction,
	}
	n.confidenceHistory = append(n.confidenceHistory, point)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// assessConfidence calls LLM to assess confidence in current risk assessment
func (n *HopNavigator) assessConfidence(ctx context.Context, hopNum int, currentRiskLevel RiskLevel,
	modificationType, modificationReason string) (prompts.ConfidenceAssessment, error) {

	// Build confidence assessment prompt with modification type context (if available)
	var prompt string
	if modificationType != "" {
		// Use type-aware prompt for better guidance
		prompt = prompts.ConfidencePromptWithModificationType(
			n.evidenceChain,
			string(currentRiskLevel),
			hopNum,
			modificationType,
			modificationReason,
		)
	} else {
		// Fallback to standard prompt
		prompt = prompts.ConfidencePrompt(n.evidenceChain, string(currentRiskLevel), hopNum)
	}

	// Query LLM
	response, _, err := n.llm.Query(ctx, prompt)
	if err != nil {
		return prompts.ConfidenceAssessment{}, fmt.Errorf("LLM confidence query failed: %w", err)
	}

	// Parse response
	assessment, err := prompts.ParseConfidenceAssessment(response)
	if err != nil {
		return prompts.ConfidenceAssessment{}, fmt.Errorf("failed to parse confidence assessment: %w", err)
	}

	return assessment, nil
}

// shouldStopEarlyFallback provides heuristic-based early stopping (fallback if confidence assessment fails)
func (n *HopNavigator) shouldStopEarlyFallback(hops []HopResult) bool {
	if len(hops) == 0 {
		return false
	}

	// Use old keyword-based logic as fallback
	response := strings.ToLower(hops[len(hops)-1].Response)

	criticalKeywords := []string{
		"critical incident",
		"critical risk",
		"production outage",
		"severe",
		"highly coupled",
		"very high risk",
	}

	for _, keyword := range criticalKeywords {
		if strings.Contains(response, keyword) {
			return true
		}
	}

	lowRiskKeywords := []string{
		"minimal risk",
		"low risk",
		"no significant",
		"no major",
		"safe to",
	}

	lowRiskCount := 0
	for _, keyword := range lowRiskKeywords {
		if strings.Contains(response, keyword) {
			lowRiskCount++
		}
	}

	return lowRiskCount >= 2
}

// executeHop performs a single hop
func (n *HopNavigator) executeHop(ctx context.Context, hopNum int, req InvestigationRequest,
	previousHops []HopResult, currentRiskScore float64, currentRiskLevel RiskLevel) (HopResult, error) {
	start := time.Now()

	// 1. Build prompt based on hop number and previous results
	prompt := n.buildHopPrompt(hopNum, req, previousHops)

	// 2. Query LLM
	response, tokens, err := n.llm.Query(ctx, prompt)
	if err != nil {
		return HopResult{}, fmt.Errorf("LLM query failed: %w", err)
	}

	// 3. Parse LLM response to determine which nodes/edges to explore
	nodesToVisit := n.parseNodesFromResponse(response)
	edgesTraversed := n.parseEdgesFromResponse(response)

	// 4. Query graph for those nodes (if graph client available)
	if n.graph != nil && len(nodesToVisit) > 0 {
		// For now, this is a placeholder - graph integration comes later
		_, _ = n.graph.GetNodes(ctx, nodesToVisit)
	}

	// 5. Add evidence from this hop to evidence chain
	// Extract key findings (simplified - would be more sophisticated in production)
	evidenceSummary := fmt.Sprintf("[HOP %d] %s", hopNum, extractKeyFindings(response))
	n.evidenceChain = append(n.evidenceChain, evidenceSummary)

	// 6. Return hop result
	return HopResult{
		HopNumber:      hopNum,
		Query:          prompt,
		Response:       response,
		NodesVisited:   nodesToVisit,
		EdgesTraversed: edgesTraversed,
		TokensUsed:     tokens,
		Duration:       time.Since(start),
		Confidence:     0.0, // Will be filled by Navigate()
		NextAction:     "",  // Will be filled by Navigate()
	}, nil
}

// extractKeyFindings extracts key findings from LLM response (simplified)
func extractKeyFindings(response string) string {
	// In a production system, this would use more sophisticated parsing
	// For now, just take first sentence or first 100 chars
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 {
			if len(line) > 100 {
				return line[:100] + "..."
			}
			return line
		}
	}
	if len(response) > 100 {
		return response[:100] + "..."
	}
	return response
}

// GetConfidenceHistory returns the confidence progression across hops
func (n *HopNavigator) GetConfidenceHistory() []ConfidencePoint {
	return n.confidenceHistory
}

// GetBreakthroughs returns detected breakthroughs
func (n *HopNavigator) GetBreakthroughs() []Breakthrough {
	if n.breakthroughTracker == nil {
		return []Breakthrough{}
	}
	return n.breakthroughTracker.GetBreakthroughs()
}

// GetStoppingReason returns why the investigation stopped
func (n *HopNavigator) GetStoppingReason(hops []HopResult) string {
	if len(hops) == 0 {
		return "No hops executed"
	}

	lastHop := hops[len(hops)-1]

	if lastHop.Confidence >= n.confidenceThreshold {
		return fmt.Sprintf("High confidence reached (%.2f >= %.2f)",
			lastHop.Confidence, n.confidenceThreshold)
	}

	if lastHop.NextAction == "FINALIZE" {
		return "LLM requested finalization"
	}

	if len(hops) >= n.maxHops {
		return fmt.Sprintf("Max hops reached (%d/%d)", len(hops), n.maxHops)
	}

	return "Investigation completed"
}

// buildHopPrompt constructs the LLM prompt for a specific hop
func (n *HopNavigator) buildHopPrompt(hopNum int, req InvestigationRequest, previousHops []HopResult) string {
	switch hopNum {
	case 1:
		// Focus on immediate context
		return fmt.Sprintf(`You are analyzing a code change for risk.

File: %s
Change type: %s
Baseline metrics:
- Coupling score: %.2f
- Co-change frequency: %.2f
- Past incidents: %d
- Ownership transition: %d days ago

Diff preview:
%s

Question: What are the immediate risk factors? Consider:
1. Structural dependencies (IMPORTS, CONTAINS)
2. Files that frequently change together (CO_CHANGED)
3. Past incidents linked to this file (CAUSED_BY)

Provide a structured analysis focusing on high-risk relationships. Be concise (2-3 paragraphs).`,
			req.FilePath,
			req.ChangeType,
			req.Baseline.CouplingScore,
			req.Baseline.CoChangeFrequency,
			req.Baseline.IncidentCount,
			req.Baseline.OwnershipDays,
			truncateDiff(req.DiffPreview, 500),
		)

	case 2:
		// Explore suspicious connections
		prevFindings := ""
		if len(previousHops) > 0 {
			prevFindings = previousHops[0].Response
		}

		return fmt.Sprintf(`Based on Hop 1 findings, investigate deeper:

Previous findings:
%s

Question: Which of these connections pose the highest risk?
1. For each CAUSED_BY edge: What was the incident severity and recency?
2. For each CO_CHANGED edge: How strong is the coupling (frequency)?
3. Are there ownership transitions that increase risk?

Rank the top 3 risk factors. Be specific and concise (2-3 paragraphs).`,
			truncateText(prevFindings, 800),
		)

	case 3:
		// Deep context for complex cases
		hop1Findings := ""
		hop2Findings := ""
		if len(previousHops) > 0 {
			hop1Findings = previousHops[0].Response
		}
		if len(previousHops) > 1 {
			hop2Findings = previousHops[1].Response
		}

		return fmt.Sprintf(`Final deep-dive investigation:

Hop 1 findings:
%s

Hop 2 findings:
%s

Question: Is there any hidden context that changes the risk assessment?
1. Cascading dependencies (2-3 hops away)
2. Behavioral patterns (temporal coupling across multiple files)
3. Systemic risks (architecture smells)

Provide final risk verdict with confidence level. Be concise (2-3 paragraphs).`,
			truncateText(hop1Findings, 600),
			truncateText(hop2Findings, 600),
		)

	default:
		return ""
	}
}

// Note: Old shouldStopEarly() and needsDeepDive() methods removed.
// Replaced by confidence-driven stopping logic in Navigate()

// parseNodesFromResponse extracts node IDs mentioned in LLM response
func (n *HopNavigator) parseNodesFromResponse(response string) []string {
	// Simple heuristic: look for file paths
	// In production, this would be more sophisticated (e.g., regex for file extensions)
	var nodes []string

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		// Look for common file patterns
		if strings.Contains(line, ".py") || strings.Contains(line, ".go") ||
			strings.Contains(line, ".js") || strings.Contains(line, ".java") {
			// Extract the file path (simplified)
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, "/") && (strings.Contains(word, ".py") ||
					strings.Contains(word, ".go") || strings.Contains(word, ".js")) {
					nodes = append(nodes, strings.Trim(word, ".,;:"))
				}
			}
		}
	}

	return nodes
}

// parseEdgesFromResponse extracts edge types mentioned in LLM response
func (n *HopNavigator) parseEdgesFromResponse(response string) []string {
	var edges []string

	edgeKeywords := map[string]string{
		"import":     "IMPORTS",
		"co-change":  "CO_CHANGED",
		"caused by":  "CAUSED_BY",
		"contains":   "CONTAINS",
		"dependency": "DEPENDS_ON",
	}

	responseLower := strings.ToLower(response)
	for keyword, edgeType := range edgeKeywords {
		if strings.Contains(responseLower, keyword) {
			edges = append(edges, edgeType)
		}
	}

	return edges
}

// Helper functions

func truncateDiff(diff string, maxLen int) string {
	if len(diff) <= maxLen {
		return diff
	}
	return diff[:maxLen] + "\n... (truncated)"
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
