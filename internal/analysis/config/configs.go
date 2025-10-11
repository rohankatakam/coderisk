package config

import "fmt"

// RiskConfig defines domain-specific risk assessment thresholds
// 12-factor: Factor 3 - Own your context window (adaptive thresholds reduce FP noise)
type RiskConfig struct {
	// ConfigKey is the identifier for this config (e.g., "python_web", "go_backend")
	ConfigKey string

	// Description explains what this config is optimized for
	Description string

	// CouplingThreshold is the max acceptable structural coupling (file dependencies)
	// Higher values = more permissive (fewer false positives for naturally coupled code)
	CouplingThreshold int

	// CoChangeThreshold is the max acceptable temporal coupling frequency [0.0-1.0]
	// Files that change together > this threshold trigger HIGH risk
	CoChangeThreshold float64

	// TestRatioThreshold is the min acceptable test coverage ratio [0.0-1.0]
	// Coverage below this threshold triggers HIGH risk
	TestRatioThreshold float64

	// Rationale explains why these thresholds were chosen
	Rationale string
}

// ConfigKey constants for looking up configs
const (
	ConfigKeyPythonWeb        = "python_web"
	ConfigKeyPythonBackend    = "python_backend"
	ConfigKeyGoBackend        = "go_backend"
	ConfigKeyGoWeb            = "go_web"
	ConfigKeyTypeScriptWeb    = "typescript_web"
	ConfigKeyTypeScriptFrontend = "typescript_frontend"
	ConfigKeyJavaBackend      = "java_backend"
	ConfigKeyRustBackend      = "rust_backend"
	ConfigKeyMLProject        = "ml_project"
	ConfigKeyCLITool          = "cli_tool"
	ConfigKeyDefault          = "default"
)

// Pre-defined risk configurations for different domain types
// Based on empirical analysis in ADR-005 and testing results
var RiskConfigs = map[string]RiskConfig{
	ConfigKeyPythonWeb: {
		ConfigKey:          ConfigKeyPythonWeb,
		Description:        "Python web applications (Flask, Django, FastAPI)",
		CouplingThreshold:  15, // Web apps have higher natural coupling (routes, models, views)
		CoChangeThreshold:  0.75,
		TestRatioThreshold: 0.4, // Python web apps often have integration tests, not just unit tests
		Rationale: `Python web frameworks encourage tight coupling between routes, models, and views.
		Flask apps typically have 10-20 imports per file. Django models are tightly coupled to views.
		Higher coupling threshold (15) reduces false positives while still catching outliers (>15).
		Test ratio 0.4 accounts for integration-heavy test suites common in web apps.`,
	},

	ConfigKeyPythonBackend: {
		ConfigKey:          ConfigKeyPythonBackend,
		Description:        "Python backend services (APIs, workers, processors)",
		CouplingThreshold:  12,
		CoChangeThreshold:  0.7,
		TestRatioThreshold: 0.5, // Backend services should have higher test coverage
		Rationale: `Python backend services are more modular than web apps but less than Go.
		Typical service file has 8-12 dependencies. Coupling threshold 12 balances modularity
		with Python's import patterns. Higher test expectations for backend logic.
		Co-change threshold 0.7 catches tightly coupled service layers.`,
	},

	ConfigKeyGoBackend: {
		ConfigKey:          ConfigKeyGoBackend,
		Description:        "Go backend services and microservices",
		CouplingThreshold:  8, // Go encourages small, focused packages
		CoChangeThreshold:  0.6,
		TestRatioThreshold: 0.5, // Go culture emphasizes testing
		Rationale: `Go's package system and interface-based design encourage low coupling.
		Well-designed Go services have 5-8 dependencies per file. Stricter coupling threshold
		(8) aligns with Go best practices. Co-change 0.6 reflects microservice independence.
		Test ratio 0.5 aligns with Go's strong testing culture.`,
	},

	ConfigKeyGoWeb: {
		ConfigKey:          ConfigKeyGoWeb,
		Description:        "Go web applications (Gin, Echo, Fiber)",
		CouplingThreshold:  10, // Web routers increase coupling slightly
		CoChangeThreshold:  0.65,
		TestRatioThreshold: 0.45,
		Rationale: `Go web frameworks add coupling through route handlers, middleware, and models.
		Slightly more permissive than pure backend (10 vs 8) but still stricter than Python web.
		Test ratio 0.45 accounts for handler-focused testing patterns.`,
	},

	ConfigKeyTypeScriptWeb: {
		ConfigKey:          ConfigKeyTypeScriptWeb,
		Description:        "TypeScript/JavaScript full-stack web apps (Next.js, Remix)",
		CouplingThreshold:  18, // Server + client code increases coupling
		CoChangeThreshold:  0.8,
		TestRatioThreshold: 0.35,
		Rationale: `Full-stack frameworks like Next.js have high coupling between client/server code.
		API routes, server components, and client components all interconnected.
		Coupling threshold 18 reflects this hybrid architecture. Lower test ratio (0.35)
		accounts for end-to-end test focus over unit tests.`,
	},

	ConfigKeyTypeScriptFrontend: {
		ConfigKey:          ConfigKeyTypeScriptFrontend,
		Description:        "TypeScript/JavaScript frontend apps (React, Vue, Angular)",
		CouplingThreshold:  20, // React component trees are highly coupled
		CoChangeThreshold:  0.8,
		TestRatioThreshold: 0.3, // Frontend often has lower test coverage
		Rationale: `React/Vue component trees create high natural coupling. Parent components
		import many child components, hooks, and utilities. A typical React component file
		has 15-25 imports. Coupling threshold 20 prevents false positives on normal component files.
		Co-change 0.8 reflects component co-evolution. Test ratio 0.3 accounts for visual testing
		focus and lower unit test coverage in frontend codebases.`,
	},

	ConfigKeyJavaBackend: {
		ConfigKey:          ConfigKeyJavaBackend,
		Description:        "Java backend services (Spring Boot, Jakarta EE)",
		CouplingThreshold:  12,
		CoChangeThreshold:  0.65,
		TestRatioThreshold: 0.6, // Java has strong testing culture
		Rationale: `Java Spring applications use dependency injection, increasing import counts.
		Enterprise Java files typically have 10-15 imports. Coupling threshold 12 balances
		Spring's DI patterns with modularity. High test ratio (0.6) reflects enterprise
		testing standards and mature tooling (JUnit, Mockito).`,
	},

	ConfigKeyRustBackend: {
		ConfigKey:          ConfigKeyRustBackend,
		Description:        "Rust backend services and systems",
		CouplingThreshold:  7, // Rust's module system encourages tight scoping
		CoChangeThreshold:  0.55,
		TestRatioThreshold: 0.55,
		Rationale: `Rust's ownership system and module design encourage minimal coupling.
		Well-structured Rust code has 4-7 dependencies per file. Strict coupling threshold (7)
		aligns with Rust best practices. Low co-change (0.55) reflects explicit dependency management.
		Test ratio 0.55 reflects strong testing culture and built-in test support.`,
	},

	ConfigKeyMLProject: {
		ConfigKey:          ConfigKeyMLProject,
		Description:        "Machine learning and data science projects",
		CouplingThreshold:  10,
		CoChangeThreshold:  0.7,
		TestRatioThreshold: 0.25, // ML projects often lack comprehensive tests
		Rationale: `ML projects import many libraries (numpy, pandas, torch, sklearn) increasing coupling.
		Typical notebook/script has 8-12 imports. Coupling threshold 10 accounts for heavy dependencies.
		Lower test ratio (0.25) reflects experimental nature of ML code and focus on notebooks.
		Co-change 0.7 reflects model/data/training co-evolution.`,
	},

	ConfigKeyCLITool: {
		ConfigKey:          ConfigKeyCLITool,
		Description:        "Command-line tools and utilities",
		CouplingThreshold:  10,
		CoChangeThreshold:  0.6,
		TestRatioThreshold: 0.4,
		Rationale: `CLI tools have moderate coupling (commands, flags, config, output formatting).
		Typical CLI file has 8-12 imports. Coupling threshold 10 accounts for command structure.
		Test ratio 0.4 reflects mix of unit tests and integration tests for CLI behavior.
		Co-change 0.6 reflects command/config co-evolution.`,
	},

	ConfigKeyDefault: {
		ConfigKey:          ConfigKeyDefault,
		Description:        "Default conservative thresholds (fallback)",
		CouplingThreshold:  10,
		CoChangeThreshold:  0.7,
		TestRatioThreshold: 0.3,
		Rationale: `Conservative middle-ground for unknown domains. Coupling 10 catches
		obvious outliers without excessive false positives. Co-change 0.7 flags tight coupling.
		Test ratio 0.3 is achievable baseline for most projects. Used when domain cannot be
		reliably inferred or for mixed/unknown project types.`,
	},
}

// GetConfig returns the risk config for the given key
func GetConfig(key string) (RiskConfig, error) {
	config, exists := RiskConfigs[key]
	if !exists {
		return RiskConfig{}, fmt.Errorf("config not found: %s", key)
	}
	return config, nil
}

// GetDefaultConfig returns the default fallback config
func GetDefaultConfig() RiskConfig {
	return RiskConfigs[ConfigKeyDefault]
}

// ListConfigKeys returns all available config keys
func ListConfigKeys() []string {
	keys := make([]string, 0, len(RiskConfigs))
	for key := range RiskConfigs {
		keys = append(keys, key)
	}
	return keys
}

// CompareConfigs returns a comparison of two configs (for debugging/analysis)
func CompareConfigs(keyA, keyB string) (string, error) {
	configA, err := GetConfig(keyA)
	if err != nil {
		return "", err
	}
	configB, err := GetConfig(keyB)
	if err != nil {
		return "", err
	}

	comparison := fmt.Sprintf(`
Comparing Configs: %s vs %s

%s: %s
%s: %s

Coupling Threshold:     %d vs %d (diff: %d)
Co-Change Threshold:    %.2f vs %.2f (diff: %.2f)
Test Ratio Threshold:   %.2f vs %.2f (diff: %.2f)

Permissiveness Ranking:
- Coupling:   %s is more %s (higher = more permissive)
- Co-Change:  %s is more %s
- Test Ratio: %s is more %s (lower = more permissive)
`,
		keyA, keyB,
		keyA, configA.Description,
		keyB, configB.Description,
		configA.CouplingThreshold, configB.CouplingThreshold,
		configB.CouplingThreshold-configA.CouplingThreshold,
		configA.CoChangeThreshold, configB.CoChangeThreshold,
		configB.CoChangeThreshold-configA.CoChangeThreshold,
		configA.TestRatioThreshold, configB.TestRatioThreshold,
		configB.TestRatioThreshold-configA.TestRatioThreshold,
		getMorePermissiveLabel(configA.CouplingThreshold, configB.CouplingThreshold, keyA, keyB),
		getPermissivenessWord(configA.CouplingThreshold, configB.CouplingThreshold),
		getMorePermissiveLabel(int(configA.CoChangeThreshold*100), int(configB.CoChangeThreshold*100), keyA, keyB),
		getPermissivenessWord(int(configA.CoChangeThreshold*100), int(configB.CoChangeThreshold*100)),
		getMorePermissiveLabel(int(configB.TestRatioThreshold*100), int(configA.TestRatioThreshold*100), keyA, keyB),
		getPermissivenessWord(int(configB.TestRatioThreshold*100), int(configA.TestRatioThreshold*100)),
	)

	return comparison, nil
}

func getMorePermissiveLabel(valA, valB int, labelA, labelB string) string {
	if valA > valB {
		return labelA
	}
	return labelB
}

func getPermissivenessWord(valA, valB int) string {
	diff := valA - valB
	if diff > 0 {
		return "permissive"
	} else if diff < 0 {
		return "strict"
	}
	return "equal"
}
