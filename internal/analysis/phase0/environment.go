package phase0

import (
	"strings"
)

// EnvironmentType represents the type of environment a configuration file targets
type EnvironmentType string

const (
	EnvProduction  EnvironmentType = "production"
	EnvStaging     EnvironmentType = "staging"
	EnvDevelopment EnvironmentType = "development"
	EnvTest        EnvironmentType = "test"
	EnvUnknown     EnvironmentType = "unknown"
)

// ProductionPatterns contains file name/path patterns that indicate production configurations
// 12-factor: Factor 8 - Own your control flow (explicit production detection criteria)
var ProductionPatterns = []string{
	"production",
	"prod",
	".env.production",
	".env.prod",
	"config/prod",
	"config/production",
	"prod.config",
	"production.yaml",
	"production.yml",
	"production.json",
	"prod.yaml",
	"prod.yml",
	"prod.json",
}

// StagingPatterns contains patterns that indicate staging configurations
var StagingPatterns = []string{
	"staging",
	"stage",
	".env.staging",
	".env.stage",
	"config/staging",
	"config/stage",
	"staging.config",
	"staging.yaml",
	"staging.yml",
	"staging.json",
}

// DevelopmentPatterns contains patterns that indicate development configurations
var DevelopmentPatterns = []string{
	"development",
	"dev",
	"local",
	".env.local",
	".env.development",
	".env.dev",
	"config/dev",
	"config/development",
	"config/local",
	"dev.config",
	"development.yaml",
	"development.yml",
	"development.json",
	"local.yaml",
	"local.yml",
	"local.json",
}

// TestPatterns contains patterns that indicate test configurations
var TestPatterns = []string{
	"test",
	"testing",
	".env.test",
	".env.testing",
	"config/test",
	"test.config",
	"test.yaml",
	"test.yml",
	"test.json",
}

// EnvironmentDetectionResult contains the results of environment detection
type EnvironmentDetectionResult struct {
	Environment      EnvironmentType // Detected environment (production, staging, dev, test, unknown)
	IsProduction     bool            // True if production environment detected
	ForceEscalate    bool            // True if should force escalate to HIGH/CRITICAL
	Reason           string          // Human-readable explanation
	MatchedPattern   string          // The pattern that matched
	IsConfiguration  bool            // True if this is a configuration file
}

// DetectEnvironment analyzes a file path to determine the target environment
// Returns environment type and whether it's a production configuration
// 12-factor: Factor 3 - Own your context window (fast environment classification)
func DetectEnvironment(filePath string) EnvironmentDetectionResult {
	result := EnvironmentDetectionResult{
		Environment:     EnvUnknown,
		IsProduction:    false,
		ForceEscalate:   false,
		IsConfiguration: false,
	}

	filePathLower := strings.ToLower(filePath)

	// First, check if this is a configuration file
	if !IsConfigurationFile(filePathLower) {
		return result
	}

	result.IsConfiguration = true

	// Check for production patterns (highest priority)
	for _, pattern := range ProductionPatterns {
		if strings.Contains(filePathLower, pattern) {
			result.Environment = EnvProduction
			result.IsProduction = true
			result.ForceEscalate = true
			result.MatchedPattern = pattern
			result.Reason = "Production configuration change (high risk)"
			return result
		}
	}

	// Check for staging patterns
	for _, pattern := range StagingPatterns {
		if strings.Contains(filePathLower, pattern) {
			result.Environment = EnvStaging
			result.IsProduction = false
			result.ForceEscalate = true // Staging is also important
			result.MatchedPattern = pattern
			result.Reason = "Staging configuration change (moderate risk)"
			return result
		}
	}

	// Check for test patterns
	for _, pattern := range TestPatterns {
		if strings.Contains(filePathLower, pattern) {
			result.Environment = EnvTest
			result.IsProduction = false
			result.ForceEscalate = false
			result.MatchedPattern = pattern
			result.Reason = "Test configuration change (low risk)"
			return result
		}
	}

	// Check for development patterns
	for _, pattern := range DevelopmentPatterns {
		if strings.Contains(filePathLower, pattern) {
			result.Environment = EnvDevelopment
			result.IsProduction = false
			result.ForceEscalate = false
			result.MatchedPattern = pattern
			result.Reason = "Development configuration change (low risk)"
			return result
		}
	}

	// Unknown environment for configuration file (default to caution)
	result.Environment = EnvUnknown
	result.ForceEscalate = true // Unknown config = treat as potentially production
	result.Reason = "Configuration file with unknown environment (treat as production)"
	return result
}

// IsConfigurationFile checks if a file is a configuration file
func IsConfigurationFile(filePath string) bool {
	filePathLower := strings.ToLower(filePath)

	// Configuration file extensions
	configExtensions := []string{
		".yaml",
		".yml",
		".json",
		".toml",
		".ini",
		".conf",
		".config",
		".properties",
		".xml",
		".env",
	}

	for _, ext := range configExtensions {
		if strings.HasSuffix(filePathLower, ext) {
			return true
		}
	}

	// Configuration file names (without extensions)
	configNames := []string{
		"dockerfile",
		"docker-compose",
		"makefile",
		".env",
		".npmrc",
		".yarnrc",
		"tsconfig",
		"webpack.config",
		"rollup.config",
		"vite.config",
		"next.config",
	}

	fileName := getFileNameLower(filePathLower)
	for _, name := range configNames {
		if strings.HasPrefix(fileName, name) {
			return true
		}
	}

	// Configuration directories
	configDirs := []string{
		"/config/",
		"/configs/",
		"/.config/",
		"/etc/",
	}

	for _, dir := range configDirs {
		if strings.Contains(filePathLower, dir) {
			return true
		}
	}

	return false
}

// GetRiskLevel returns the risk level for the detected environment
func (r EnvironmentDetectionResult) GetRiskLevel() string {
	if !r.IsConfiguration {
		return "" // Not a configuration file
	}

	switch r.Environment {
	case EnvProduction:
		return "CRITICAL"
	case EnvStaging:
		return "HIGH"
	case EnvTest:
		return "LOW"
	case EnvDevelopment:
		return "LOW"
	case EnvUnknown:
		return "HIGH" // Unknown = treat as potentially production
	default:
		return ""
	}
}

// getFileNameLower extracts the lowercase file name from a path
func getFileNameLower(path string) string {
	// Already lowercase from caller
	path = strings.ReplaceAll(path, "\\", "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return path
	}
	return path[lastSlash+1:]
}

// IsProductionConfig is a quick check for production configuration files
func IsProductionConfig(filePath string) bool {
	result := DetectEnvironment(filePath)
	return result.IsProduction
}
