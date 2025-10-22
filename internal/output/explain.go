package output

import (
	"fmt"
	"io"
	"time"

	"github.com/rohankatakam/coderisk/internal/types"
)

// ExplainFormatter outputs full investigation trace
type ExplainFormatter struct{}

func (f *ExplainFormatter) Format(result *models.RiskResult, w io.Writer) error {
	// Header
	fmt.Fprintf(w, "🔍 CodeRisk Investigation Report\n")
	fmt.Fprintf(w, "Started: %s\n", result.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "Completed: %s (%.1fs)\n",
		result.EndTime.Format(time.RFC3339),
		result.Duration.Seconds())
	fmt.Fprintf(w, "Agent hops: %d\n\n", len(result.InvestigationTrace))

	// Investigation trace (hop-by-hop)
	for i, hop := range result.InvestigationTrace {
		fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(w, "Hop %d: %s\n", i+1, hop.NodeID)
		fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Changed entities
		if len(hop.ChangedEntities) > 0 {
			fmt.Fprintf(w, "Changed functions:\n")
			for _, entity := range hop.ChangedEntities {
				fmt.Fprintf(w, "  - %s (lines %d-%d)\n",
					entity.Name, entity.StartLine, entity.EndLine)
			}
			fmt.Fprintf(w, "\n")
		}

		// Metrics calculated
		if len(hop.Metrics) > 0 {
			fmt.Fprintf(w, "Metrics calculated:\n")
			for _, metric := range hop.Metrics {
				status := f.metricStatus(metric)
				if metric.Threshold != nil {
					fmt.Fprintf(w, "  %s %s: %.1f (target: <%.1f)\n",
						status, metric.Name, metric.Value, *metric.Threshold)
				} else {
					fmt.Fprintf(w, "  %s %s: %.1f\n",
						status, metric.Name, metric.Value)
				}
			}
			fmt.Fprintf(w, "\n")
		}

		// Agent decision
		fmt.Fprintf(w, "Agent decision: %s\n", hop.Decision)
		if hop.Reasoning != "" {
			fmt.Fprintf(w, "Reasoning: %s\n", hop.Reasoning)
		}
		fmt.Fprintf(w, "\n")
	}

	// Final assessment
	fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(w, "Final Assessment\n")
	fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprintf(w, "Risk Level: %s\n\n", result.RiskLevel)

	// Evidence
	if len(result.Evidence) > 0 {
		fmt.Fprintf(w, "Evidence:\n")
		for i, evidence := range result.Evidence {
			fmt.Fprintf(w, "  %d. %s\n", i+1, evidence)
		}
		fmt.Fprintf(w, "\n")
	}

	// Recommendations (prioritized)
	if len(result.Recommendations) > 0 {
		fmt.Fprintf(w, "Recommendations (priority order):\n")
		for i, rec := range result.Recommendations {
			fmt.Fprintf(w, "  %d. %s\n", i+1, rec)
		}
		fmt.Fprintf(w, "\n")
	}

	// Next steps
	if len(result.NextSteps) > 0 {
		fmt.Fprintf(w, "Suggested next steps:\n")
		for _, step := range result.NextSteps {
			fmt.Fprintf(w, "  → %s\n", step)
		}
	}

	return nil
}

func (f *ExplainFormatter) metricStatus(metric types.Metric) string {
	if metric.Threshold != nil && metric.Value > *metric.Threshold {
		return "❌"
	} else if metric.Warning != nil && metric.Value > *metric.Warning {
		return "⚠️ "
	}
	return "✅"
}
