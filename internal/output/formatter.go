package output

import (
	"io"
	"os"

	"github.com/rohankatakam/coderisk/internal/types"
)

// Formatter defines output formatting interface
type Formatter interface {
	Format(result *types.RiskResult, w io.Writer) error
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

// === Merged from verbosity.go ===

// GetDefaultVerbosity returns appropriate default based on environment
func GetDefaultVerbosity() VerbosityLevel {
	// Pre-commit hook context (GIT_AUTHOR_DATE set by git)
	if os.Getenv("GIT_AUTHOR_DATE") != "" {
		return VerbosityQuiet
	}

	// CI/CD context
	if os.Getenv("CI") == "true" {
		return VerbosityStandard
	}

	// AI assistant context (detected by special env var)
	if os.Getenv("CRISK_AI_MODE") == "1" {
		return VerbosityAIMode
	}

	// Interactive terminal (default)
	return VerbosityStandard
}
