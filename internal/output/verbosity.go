package output

import (
	"os"
)

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
