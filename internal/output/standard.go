package output

import (
	"fmt"
	"io"

	"github.com/coderisk/coderisk-go/internal/models"
)

// StandardFormatter outputs issues + recommendations (default)
type StandardFormatter struct{}

func (f *StandardFormatter) Format(result *models.RiskResult, w io.Writer) error {
	// Header
	fmt.Fprintf(w, "🔍 CodeRisk Analysis\n")
	if result.Branch != "" {
		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
	}
	fmt.Fprintf(w, "Files changed: %d\n", result.FilesChanged)
	fmt.Fprintf(w, "Risk level: %s\n\n", result.RiskLevel)

	// Issues
	if len(result.Issues) > 0 {
		fmt.Fprintf(w, "Issues:\n")
		for i, issue := range result.Issues {
			fmt.Fprintf(w, "%d. %s %s - %s\n",
				i+1,
				severityEmoji(issue.Severity),
				issue.File,
				issue.Message,
			)
		}
		fmt.Fprintf(w, "\n")
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Fprintf(w, "Recommendations:\n")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(w, "- %s\n", rec)
		}
		fmt.Fprintf(w, "\n")
	}

	// Next steps
	if result.RiskLevel != "LOW" && result.RiskLevel != "NONE" {
		fmt.Fprintf(w, "Run 'crisk check --explain' for investigation trace\n")
	}

	return nil
}

func severityEmoji(severity string) string {
	switch severity {
	case "HIGH", "CRITICAL":
		return "🔴"
	case "MEDIUM":
		return "⚠️ "
	case "LOW":
		return "ℹ️ "
	default:
		return "•"
	}
}
