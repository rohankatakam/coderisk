package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/agent"
)

// DisplayPhase2Summary shows investigation summary (standard mode)
// 12-factor: Factor 4 - Tools are structured outputs (human-readable summary)
func DisplayPhase2Summary(assessment agent.RiskAssessment) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Investigation Summary\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show key evidence
	if len(assessment.Evidence) > 0 {
		fmt.Println("\nKey Evidence:")
		for i, evidence := range assessment.Evidence {
			fmt.Printf("%d. [%s] %s\n", i+1, evidence.Type, evidence.Description)
		}
	}

	// Show final assessment
	fmt.Printf("\nRisk Level: %s (confidence: %.0f%%)\n",
		assessment.RiskLevel,
		assessment.Confidence*100)

	// Show summary
	if assessment.Summary != "" {
		fmt.Printf("\nSummary: %s\n", assessment.Summary)
	}

	// Show investigation stats if available
	if assessment.Investigation != nil {
		duration := assessment.Investigation.CompletedAt.Sub(assessment.Investigation.Request.StartedAt)
		fmt.Printf("\nInvestigation completed in %.1fs (%d hops, %d tokens)\n",
			duration.Seconds(),
			len(assessment.Investigation.Hops),
			assessment.Investigation.TotalTokens)
	}
}

// DisplayPhase2Trace shows full hop-by-hop investigation (explain mode)
// 12-factor: Factor 4 - Tools are structured outputs (detailed trace for debugging)
func DisplayPhase2Trace(assessment agent.RiskAssessment) {
	if assessment.Investigation == nil {
		DisplayPhase2Summary(assessment)
		return
	}

	investigation := assessment.Investigation

	fmt.Println("\n🔍 CodeRisk Investigation Report")
	fmt.Printf("Started: %s\n", investigation.Request.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Completed: %s (%.1fs)\n",
		investigation.CompletedAt.Format("2006-01-02 15:04:05"),
		investigation.CompletedAt.Sub(investigation.Request.StartedAt).Seconds())
	fmt.Printf("Agent hops: %d\n", len(investigation.Hops))

	// Show each hop
	for _, hop := range investigation.Hops {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Hop %d\n", hop.HopNumber)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if len(hop.NodesVisited) > 0 {
			fmt.Printf("\nNodes visited: %v\n", hop.NodesVisited)
		}

		if len(hop.EdgesTraversed) > 0 {
			fmt.Printf("Edges traversed: %v\n", hop.EdgesTraversed)
		}

		if hop.Query != "" {
			fmt.Printf("\nQuery: %s\n", hop.Query)
		}

		if hop.Response != "" {
			// Truncate long responses
			response := hop.Response
			if len(response) > 200 {
				response = response[:200] + "..."
			}
			fmt.Printf("Response: %s\n", response)
		}

		fmt.Printf("\nTokens: %d | Duration: %dms\n", hop.TokensUsed, hop.Duration.Milliseconds())
	}

	// Show final assessment
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Final Assessment")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	DisplayPhase2Summary(assessment)
}

// DisplayPhase2JSON outputs investigation trace in AI mode JSON format
// 12-factor: Factor 4 - Tools are structured outputs (machine-readable for AI)
func DisplayPhase2JSON(assessment agent.RiskAssessment) {
	if assessment.Investigation == nil {
		// No investigation trace, output basic assessment
		data := map[string]interface{}{
			"risk_level": assessment.RiskLevel,
			"confidence": assessment.Confidence,
			"summary":    assessment.Summary,
			"evidence":   assessment.Evidence,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	// Convert investigation to trace format
	trace := convertInvestigationToTrace(assessment.Investigation)

	// Output full investigation data
	data := map[string]interface{}{
		"risk_level":          assessment.RiskLevel,
		"confidence":          assessment.Confidence,
		"summary":             assessment.Summary,
		"evidence":            assessment.Evidence,
		"investigation_trace": trace,
		"investigation_stats": map[string]interface{}{
			"total_hops":   len(assessment.Investigation.Hops),
			"total_tokens": assessment.Investigation.TotalTokens,
			"duration_ms":  assessment.Investigation.CompletedAt.Sub(assessment.Investigation.Request.StartedAt).Milliseconds(),
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

// Helper functions

// convertInvestigationToTrace converts investigation hops to JSON-friendly format
func convertInvestigationToTrace(investigation *agent.Investigation) []map[string]interface{} {
	trace := make([]map[string]interface{}, len(investigation.Hops))
	for i, hop := range investigation.Hops {
		trace[i] = map[string]interface{}{
			"hop":             hop.HopNumber,
			"nodes_visited":   hop.NodesVisited,
			"edges_traversed": hop.EdgesTraversed,
			"query":           hop.Query,
			"response":        hop.Response,
			"tokens_used":     hop.TokensUsed,
			"duration_ms":     hop.Duration.Milliseconds(),
		}
	}
	return trace
}

// formatTime formats minutes into human-readable duration
func formatTime(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dmin", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dmin", hours, mins)
}

// formatDuration formats a duration into human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
