package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// LLMClientInterface defines the interface for LLM clients
type LLMClientInterface interface {
	Query(ctx context.Context, prompt string) (string, int, error)
	SetModel(model string)
}

// HopNavigator performs hop-by-hop graph traversal
type HopNavigator struct {
	llm       LLMClientInterface
	graph     GraphClient
	maxHops   int
	maxTokens int
}

// NewHopNavigator creates a new hop navigator
func NewHopNavigator(llm LLMClientInterface, graph GraphClient, maxHops int) *HopNavigator {
	return &HopNavigator{
		llm:       llm,
		graph:     graph,
		maxHops:   maxHops,
		maxTokens: 10000, // Budget for entire investigation
	}
}

// Navigate performs multi-hop investigation
func (n *HopNavigator) Navigate(ctx context.Context, req InvestigationRequest) ([]HopResult, error) {
	var hops []HopResult
	totalTokens := 0

	// Hop 1: Immediate neighbors (CONTAINS, IMPORTS, CO_CHANGED)
	hop1, err := n.executeHop(ctx, 1, req, hops)
	if err != nil {
		return nil, fmt.Errorf("hop 1 failed: %w", err)
	}
	hops = append(hops, hop1)
	totalTokens += hop1.TokensUsed

	// Early exit if risk is obvious
	if n.shouldStopEarly(hops) {
		return hops, nil
	}

	// Check token budget
	if totalTokens >= n.maxTokens {
		return hops, nil
	}

	// Hop 2: Explore suspicious neighbors (CAUSED_BY, temporal coupling)
	hop2, err := n.executeHop(ctx, 2, req, hops)
	if err != nil {
		// Non-fatal, return what we have
		return hops, nil
	}
	hops = append(hops, hop2)
	totalTokens += hop2.TokensUsed

	// Check token budget
	if totalTokens >= n.maxTokens {
		return hops, nil
	}

	// Hop 3: Deep context (if still unclear)
	if n.needsDeepDive(hops) && totalTokens < n.maxTokens-2000 {
		hop3, err := n.executeHop(ctx, 3, req, hops)
		if err != nil {
			// Non-fatal, return what we have
			return hops, nil
		}
		hops = append(hops, hop3)
	}

	return hops, nil
}

// executeHop performs a single hop
func (n *HopNavigator) executeHop(ctx context.Context, hopNum int, req InvestigationRequest, previousHops []HopResult) (HopResult, error) {
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

	// 5. Return hop result
	return HopResult{
		HopNumber:      hopNum,
		Query:          prompt,
		Response:       response,
		NodesVisited:   nodesToVisit,
		EdgesTraversed: edgesTraversed,
		TokensUsed:     tokens,
		Duration:       time.Since(start),
	}, nil
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

// shouldStopEarly determines if we can stop after Hop 1
func (n *HopNavigator) shouldStopEarly(hops []HopResult) bool {
	if len(hops) == 0 {
		return false
	}

	// Parse first hop response for keywords indicating obvious risk
	response := strings.ToLower(hops[0].Response)

	// Check for critical indicators
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

	// Check for low risk indicators
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

	// If multiple low-risk indicators, stop early
	return lowRiskCount >= 2
}

// needsDeepDive determines if Hop 3 is needed
func (n *HopNavigator) needsDeepDive(hops []HopResult) bool {
	if len(hops) < 2 {
		return false
	}

	// If Hop 1 and Hop 2 have conflicting conclusions, do Hop 3
	hop1 := strings.ToLower(hops[0].Response)
	hop2 := strings.ToLower(hops[1].Response)

	// Check for uncertainty markers
	uncertaintyKeywords := []string{
		"unclear",
		"uncertain",
		"may",
		"might",
		"possibly",
		"potentially",
		"depends",
	}

	uncertaintyCount := 0
	for _, keyword := range uncertaintyKeywords {
		if strings.Contains(hop1, keyword) || strings.Contains(hop2, keyword) {
			uncertaintyCount++
		}
	}

	// If both hops show uncertainty, we need deeper investigation
	return uncertaintyCount >= 2
}

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
