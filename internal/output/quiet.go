package output

import (
	"fmt"
	"io"

	"github.com/rohankatakam/coderisk/internal/types"
)

// QuietFormatter outputs one-line summary (for pre-commit hooks)
type QuietFormatter struct{}

func (f *QuietFormatter) Format(result *types.RiskResult, w io.Writer) error {
	// Success case
	if result.RiskLevel == "LOW" || result.RiskLevel == "NONE" {
		fmt.Fprintf(w, "✅ %s risk\n", result.RiskLevel)
		return nil
	}

	// Risk detected case
	issueCount := len(result.Issues)
	fmt.Fprintf(w, "⚠️  %s risk: %d issues detected\n", result.RiskLevel, issueCount)
	fmt.Fprintf(w, "Run 'crisk check' for details\n")

	return nil
}
