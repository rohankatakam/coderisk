package config

import (
	"fmt"
	"strings"
)

// SelectConfig chooses the appropriate risk configuration based on repository metadata
// 12-factor: Factor 8 - Own your control flow (explicit config selection logic)
func SelectConfig(metadata RepoMetadata) RiskConfig {
	// Step 1: Infer domain from repository characteristics
	domain := InferDomain(metadata)

	// Step 2: Normalize language name
	language := normalizeLanguage(metadata.PrimaryLanguage)

	// Step 3: Try to find language+domain specific config
	configKey := buildConfigKey(language, domain)
	if configKey != "" && configKey != ConfigKeyDefault {
		if config, err := GetConfig(configKey); err == nil {
			return config
		}
	}

	// Step 4: Try fallback strategies
	// Strategy 1: Try language-specific config without domain
	if config, err := tryLanguageFallback(language, domain); err == nil {
		return config
	}

	// Strategy 2: Try domain-specific config without language
	if config, err := tryDomainFallback(domain); err == nil {
		return config
	}

	// Strategy 3: Return default config
	return GetDefaultConfig()
}

// SelectConfigWithReason returns the selected config along with the selection reasoning
// Useful for debugging and observability
func SelectConfigWithReason(metadata RepoMetadata) (RiskConfig, string) {
	domain := InferDomain(metadata)
	language := normalizeLanguage(metadata.PrimaryLanguage)
	configKey := buildConfigKey(language, domain)

	// Try exact match
	if configKey != "" && configKey != ConfigKeyDefault {
		if config, err := GetConfig(configKey); err == nil {
			reason := fmt.Sprintf("Exact match: language=%s, domain=%s → config=%s",
				language, domain, configKey)
			return config, reason
		}
	}

	// Try language fallback
	if config, err := tryLanguageFallback(language, domain); err == nil {
		reason := fmt.Sprintf("Language fallback: language=%s, domain=%s → config=%s",
			language, domain, config.ConfigKey)
		return config, reason
	}

	// Try domain fallback
	if config, err := tryDomainFallback(domain); err == nil {
		reason := fmt.Sprintf("Domain fallback: language=%s, domain=%s → config=%s",
			language, domain, config.ConfigKey)
		return config, reason
	}

	// Use default
	defaultConfig := GetDefaultConfig()
	reason := fmt.Sprintf("Default fallback: language=%s, domain=%s → config=%s (no specific match found)",
		language, domain, defaultConfig.ConfigKey)
	return defaultConfig, reason
}

// buildConfigKey constructs a config key from language and domain
func buildConfigKey(language string, domain Domain) string {
	if language == "" || domain == DomainUnknown {
		return ConfigKeyDefault
	}

	// Special cases: some domains don't have language-specific configs
	// ML and CLI are domain-specific only
	switch domain {
	case DomainML:
		return ConfigKeyMLProject
	case DomainCLI:
		return ConfigKeyCLITool
	}

	// Format: "language_domain" (e.g., "python_web", "go_backend")
	configKey := fmt.Sprintf("%s_%s", strings.ToLower(language), strings.ToLower(string(domain)))

	// Verify the config exists, otherwise try fallback
	if _, err := GetConfig(configKey); err == nil {
		return configKey
	}

	// Config doesn't exist, signal fallback needed
	return ""
}

// normalizeLanguage converts language names to consistent format
func normalizeLanguage(language string) string {
	languageMap := map[string]string{
		"python":     "python",
		"go":         "go",
		"golang":     "go",
		"typescript": "typescript",
		"javascript": "typescript", // JS uses TypeScript configs
		"java":       "java",
		"rust":       "rust",
		"ruby":       "ruby",
		"php":        "php",
		"c#":         "csharp",
		"csharp":     "csharp",
		"kotlin":     "kotlin",
		"scala":      "scala",
		"elixir":     "elixir",
	}

	normalized, exists := languageMap[strings.ToLower(language)]
	if exists {
		return normalized
	}

	// Return lowercase version for unknown languages
	return strings.ToLower(language)
}

// tryLanguageFallback attempts to find a config using language-specific fallbacks
func tryLanguageFallback(language string, domain Domain) (RiskConfig, error) {
	// Define language-specific fallback strategies
	fallbacks := map[string][]string{
		"python": {
			ConfigKeyPythonWeb,     // Try web first (most common)
			ConfigKeyPythonBackend, // Then backend
		},
		"go": {
			ConfigKeyGoBackend, // Try backend first (most common)
			ConfigKeyGoWeb,     // Then web
		},
		"typescript": {
			ConfigKeyTypeScriptFrontend, // Try frontend first
			ConfigKeyTypeScriptWeb,      // Then full-stack
		},
		"javascript": {
			ConfigKeyTypeScriptFrontend, // JS uses TS configs
			ConfigKeyTypeScriptWeb,
		},
		"java": {
			ConfigKeyJavaBackend, // Java is typically backend
		},
		"rust": {
			ConfigKeyRustBackend, // Rust is typically backend
		},
	}

	// Get fallback list for this language
	fallbackKeys, exists := fallbacks[language]
	if !exists {
		return RiskConfig{}, fmt.Errorf("no fallback for language: %s", language)
	}

	// Try each fallback in order
	for _, key := range fallbackKeys {
		if config, err := GetConfig(key); err == nil {
			return config, nil
		}
	}

	return RiskConfig{}, fmt.Errorf("no fallback config found for language: %s", language)
}

// tryDomainFallback attempts to find a config based on domain alone
func tryDomainFallback(domain Domain) (RiskConfig, error) {
	// Map domains to reasonable default configs
	domainFallbacks := map[Domain]string{
		DomainWeb:      ConfigKeyPythonWeb,          // Web → Python web (common)
		DomainBackend:  ConfigKeyGoBackend,          // Backend → Go backend (modern default)
		DomainFrontend: ConfigKeyTypeScriptFrontend, // Frontend → TypeScript frontend
		DomainML:       ConfigKeyMLProject,          // ML → ML config
		DomainCLI:      ConfigKeyCLITool,            // CLI → CLI config
	}

	fallbackKey, exists := domainFallbacks[domain]
	if !exists || domain == DomainUnknown {
		return RiskConfig{}, fmt.Errorf("no fallback for domain: %s", domain)
	}

	return GetConfig(fallbackKey)
}

// ConfigSelectionResult provides detailed information about config selection
type ConfigSelectionResult struct {
	SelectedConfig RiskConfig
	Language       string
	Domain         Domain
	ConfigKey      string
	Reason         string
	FallbackUsed   bool
}

// SelectConfigDetailed returns comprehensive information about config selection
// Useful for logging, debugging, and audit trails
func SelectConfigDetailed(metadata RepoMetadata) ConfigSelectionResult {
	domain := InferDomain(metadata)
	language := normalizeLanguage(metadata.PrimaryLanguage)
	configKey := buildConfigKey(language, domain)

	result := ConfigSelectionResult{
		Language:  language,
		Domain:    domain,
		ConfigKey: configKey,
	}

	// Try exact match
	if configKey != "" && configKey != ConfigKeyDefault {
		if config, err := GetConfig(configKey); err == nil {
			result.SelectedConfig = config
			result.Reason = fmt.Sprintf("Exact match: %s", configKey)
			result.FallbackUsed = false
			return result
		}
	}

	// Try fallbacks
	config, reason := SelectConfigWithReason(metadata)
	result.SelectedConfig = config
	result.Reason = reason
	result.FallbackUsed = true

	return result
}

// ValidateConfigSelection checks if the selected config is appropriate
// Returns warning messages if config may not be optimal
func ValidateConfigSelection(metadata RepoMetadata, config RiskConfig) []string {
	var warnings []string

	domain := InferDomain(metadata)

	// Check for potential mismatches
	if domain == DomainFrontend && config.CouplingThreshold < 15 {
		warnings = append(warnings,
			fmt.Sprintf("Frontend domain detected but coupling threshold (%d) is low. Frontend typically has high coupling (15-20).",
				config.CouplingThreshold))
	}

	if domain == DomainML && config.TestRatioThreshold > 0.4 {
		warnings = append(warnings,
			fmt.Sprintf("ML domain detected but test ratio threshold (%.2f) is high. ML projects typically have lower test coverage (0.25-0.35).",
				config.TestRatioThreshold))
	}

	if domain == DomainBackend && config.CouplingThreshold > 15 {
		warnings = append(warnings,
			fmt.Sprintf("Backend domain detected but coupling threshold (%d) is very high. Backend services should have moderate coupling (8-12).",
				config.CouplingThreshold))
	}

	return warnings
}

// GetAllApplicableConfigs returns all configs that could reasonably apply to the repository
// Useful for showing users alternative config options
func GetAllApplicableConfigs(metadata RepoMetadata) []RiskConfig {
	domain := InferDomain(metadata)
	language := normalizeLanguage(metadata.PrimaryLanguage)

	var applicable []RiskConfig

	// Add exact match if exists
	exactKey := buildConfigKey(language, domain)
	if config, err := GetConfig(exactKey); err == nil {
		applicable = append(applicable, config)
	}

	// Add language fallbacks
	if config, err := tryLanguageFallback(language, domain); err == nil {
		// Check if not already added
		if !configInList(config, applicable) {
			applicable = append(applicable, config)
		}
	}

	// Add domain fallback
	if config, err := tryDomainFallback(domain); err == nil {
		if !configInList(config, applicable) {
			applicable = append(applicable, config)
		}
	}

	// Always include default as last option
	defaultConfig := GetDefaultConfig()
	if !configInList(defaultConfig, applicable) {
		applicable = append(applicable, defaultConfig)
	}

	return applicable
}

// configInList checks if a config is already in the list
func configInList(target RiskConfig, list []RiskConfig) bool {
	for _, config := range list {
		if config.ConfigKey == target.ConfigKey {
			return true
		}
	}
	return false
}
