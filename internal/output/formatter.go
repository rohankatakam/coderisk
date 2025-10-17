package output

import (
	"io"

	"github.com/rohankatakam/coderisk/internal/models"
)

// Formatter defines output formatting interface
type Formatter interface {
	Format(result *models.RiskResult, w io.Writer) error
}

// VerbosityLevel determines output detail
type VerbosityLevel int

const (
	VerbosityQuiet    VerbosityLevel = iota // Level 1: One-line summary
	VerbosityStandard                       // Level 2: Issues + recommendations
	VerbosityExplain                        // Level 3: Full investigation trace
	VerbosityAIMode                         // Level 4: Machine-readable JSON
)

// NewFormatter creates appropriate formatter based on level
func NewFormatter(level VerbosityLevel) Formatter {
	switch level {
	case VerbosityQuiet:
		return &QuietFormatter{}
	case VerbosityStandard:
		return &StandardFormatter{}
	case VerbosityExplain:
		return &ExplainFormatter{}
	case VerbosityAIMode:
		return &AIFormatter{} // Session 3 will implement this
	default:
		return &StandardFormatter{}
	}
}
