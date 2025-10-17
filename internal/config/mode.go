package config

import (
	"os"
	"strings"
)

// DeploymentMode represents the deployment context
type DeploymentMode string

const (
	// ModeDevelopment represents local development (git clone + make dev)
	// - Uses .env file for configuration
	// - Docker Compose manages services
	// - Passwords from .env are acceptable (local containers only)
	// - Used by: make dev, contributors, local testing
	ModeDevelopment DeploymentMode = "development"

	// ModePackaged represents packaged installation (brew install, releases)
	// - Single binary distribution (no .env file)
	// - User manages Docker separately
	// - Credentials via: env vars, keychain, config file, or interactive prompt
	// - Used by: brew install crisk, GoReleaser binaries
	ModePackaged DeploymentMode = "packaged"

	// ModeCI represents CI/CD pipeline execution
	// - All credentials from environment variables
	// - No interactive prompts allowed
	// - Strict validation, fail fast
	// - Used by: GitHub Actions, GitLab CI, etc.
	ModeCI DeploymentMode = "ci"
)

// DetectMode determines the deployment context based on environment
func DetectMode() DeploymentMode {
	// Explicit mode override (highest priority)
	if mode := os.Getenv("CRISK_MODE"); mode != "" {
		switch strings.ToLower(mode) {
		case "development", "dev":
			return ModeDevelopment
		case "packaged", "pkg", "production", "prod":
			return ModePackaged
		case "ci", "cicd":
			return ModeCI
		}
	}

	// CI environment detection
	if isCI() {
		return ModeCI
	}

	// Development mode indicators (in order of priority)
	// 1. .env file exists (Docker Compose development)
	if _, err := os.Stat(".env"); err == nil {
		return ModeDevelopment
	}

	// 2. Inside git repository with go.mod (source development)
	if _, err := os.Stat(".git"); err == nil {
		if _, err := os.Stat("go.mod"); err == nil {
			return ModeDevelopment
		}
	}

	// 3. go.mod exists (running from source)
	if _, err := os.Stat("go.mod"); err == nil {
		return ModeDevelopment
	}

	// 4. Makefile exists (development environment)
	if _, err := os.Stat("Makefile"); err == nil {
		return ModeDevelopment
	}

	// Otherwise: packaged installation (brew, direct binary)
	return ModePackaged
}

// isCI detects if running in a CI/CD environment
func isCI() bool {
	// Common CI environment variables
	ciEnvVars := []string{
		"CI",                    // Generic CI indicator
		"CONTINUOUS_INTEGRATION", // Generic CI indicator
		"GITHUB_ACTIONS",        // GitHub Actions
		"GITLAB_CI",             // GitLab CI
		"CIRCLECI",              // CircleCI
		"TRAVIS",                // Travis CI
		"JENKINS_URL",           // Jenkins
		"BUILDKITE",             // Buildkite
		"DRONE",                 // Drone CI
		"TF_BUILD",              // Azure Pipelines
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// IsDevelopment returns true if running in development mode (git clone)
func IsDevelopment() bool {
	return DetectMode() == ModeDevelopment
}

// IsPackaged returns true if running from packaged installation (brew)
func IsPackaged() bool {
	return DetectMode() == ModePackaged
}

// IsCI returns true if running in CI/CD
func IsCI() bool {
	return DetectMode() == ModeCI
}

// GetMode returns the current deployment mode
func GetMode() DeploymentMode {
	return DetectMode()
}

// String returns the string representation of the mode
func (m DeploymentMode) String() string {
	return string(m)
}

// AllowsDevelopmentDefaults returns true if mode allows .env defaults
func (m DeploymentMode) AllowsDevelopmentDefaults() bool {
	return m == ModeDevelopment
}

// RequiresSecureCredentials returns true if mode requires secure passwords
func (m DeploymentMode) RequiresSecureCredentials() bool {
	return m == ModePackaged || m == ModeCI
}

// AllowsInteractivePrompts returns true if interactive prompts are allowed
func (m DeploymentMode) AllowsInteractivePrompts() bool {
	return m == ModePackaged
}

// RequiresStrictValidation returns true if mode requires strict validation
func (m DeploymentMode) RequiresStrictValidation() bool {
	return m == ModeCI
}

// Description returns a human-readable description of the mode
func (m DeploymentMode) Description() string {
	switch m {
	case ModeDevelopment:
		return "Local development (git clone + make dev)"
	case ModePackaged:
		return "Packaged installation (brew install)"
	case ModeCI:
		return "CI/CD pipeline"
	default:
		return "Unknown mode"
	}
}

// ConfigSource returns where credentials should come from
func (m DeploymentMode) ConfigSource() string {
	switch m {
	case ModeDevelopment:
		return ".env file"
	case ModePackaged:
		return "environment variables, keychain, or interactive config"
	case ModeCI:
		return "environment variables only"
	default:
		return "unknown"
	}
}
