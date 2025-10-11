package config

import (
	"strings"
	"testing"
)

func TestSelectConfig_ExactMatch(t *testing.T) {
	tests := []struct {
		name           string
		metadata       RepoMetadata
		expectedConfig string
	}{
		{
			name: "Python Flask web app",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"Flask==2.3.0"},
			},
			expectedConfig: ConfigKeyPythonWeb,
		},
		{
			name: "Go backend service",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod:           []string{"require github.com/go-redis/redis v8.11.0"},
				DirectoryNames:  []string{"internal", "pkg"},
			},
			expectedConfig: ConfigKeyGoBackend,
		},
		{
			name: "TypeScript React frontend",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"react": "^18.2.0",
					},
					"scripts": map[string]interface{}{
						"start": "react-scripts start",
					},
				},
			},
			expectedConfig: ConfigKeyTypeScriptFrontend,
		},
		{
			name: "Python ML project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"tensorflow==2.13.0", "pandas==2.0.0"},
			},
			expectedConfig: ConfigKeyMLProject,
		},
		{
			name: "Go CLI tool",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod:           []string{"require github.com/spf13/cobra v1.7.0"},
				DirectoryNames:  []string{"cmd"},
			},
			expectedConfig: ConfigKeyCLITool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SelectConfig(tt.metadata)
			if config.ConfigKey != tt.expectedConfig {
				t.Errorf("expected config %s, got %s", tt.expectedConfig, config.ConfigKey)
			}
		})
	}
}

func TestSelectConfig_LanguageFallback(t *testing.T) {
	tests := []struct {
		name           string
		metadata       RepoMetadata
		expectedConfig string
		shouldFallback bool
	}{
		{
			name: "Python project with unknown domain",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"requests==2.31.0"},
				DirectoryNames:  []string{"src", "tests"},
			},
			expectedConfig: ConfigKeyPythonWeb, // Fallback to Python web
			shouldFallback: true,
		},
		{
			name: "Go project with unknown domain",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				DirectoryNames:  []string{"internal"},
			},
			expectedConfig: ConfigKeyGoBackend, // Go defaults to backend
			shouldFallback: false,              // Domain inference should detect backend
		},
		{
			name: "JavaScript project (uses TypeScript config)",
			metadata: RepoMetadata{
				PrimaryLanguage: "JavaScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"express": "^4.18.0",
					},
				},
			},
			expectedConfig: ConfigKeyTypeScriptWeb, // JS web → TS web config
			shouldFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SelectConfig(tt.metadata)
			if config.ConfigKey != tt.expectedConfig {
				t.Errorf("expected config %s, got %s", tt.expectedConfig, config.ConfigKey)
			}
		})
	}
}

func TestSelectConfig_DefaultFallback(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
	}{
		{
			name: "Empty metadata",
			metadata: RepoMetadata{
				PrimaryLanguage: "",
			},
		},
		{
			name: "Unknown language and domain",
			metadata: RepoMetadata{
				PrimaryLanguage: "Fortran",
				DirectoryNames:  []string{"legacy"},
			},
		},
		{
			name: "Generic library",
			metadata: RepoMetadata{
				PrimaryLanguage: "C++",
				DirectoryNames:  []string{"src", "include"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SelectConfig(tt.metadata)
			if config.ConfigKey != ConfigKeyDefault {
				t.Errorf("expected default config, got %s", config.ConfigKey)
			}
		})
	}
}

func TestSelectConfigWithReason(t *testing.T) {
	tests := []struct {
		name          string
		metadata      RepoMetadata
		expectedKey   string
		reasonContains string
	}{
		{
			name: "Exact match provides exact reason",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"Flask==2.3.0"},
			},
			expectedKey:   ConfigKeyPythonWeb,
			reasonContains: "Exact match",
		},
		{
			name: "Fallback provides fallback reason",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"requests==2.31.0"},
			},
			expectedKey:   ConfigKeyPythonWeb,
			reasonContains: "fallback",
		},
		{
			name: "Default provides default reason",
			metadata: RepoMetadata{
				PrimaryLanguage: "UnknownLang",
			},
			expectedKey:   ConfigKeyDefault,
			reasonContains: "Default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, reason := SelectConfigWithReason(tt.metadata)
			if config.ConfigKey != tt.expectedKey {
				t.Errorf("expected config %s, got %s", tt.expectedKey, config.ConfigKey)
			}
			if !strings.Contains(reason, tt.reasonContains) {
				t.Errorf("expected reason to contain %q, got: %s", tt.reasonContains, reason)
			}
		})
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Python", "python"},
		{"PYTHON", "python"},
		{"Go", "go"},
		{"Golang", "go"},
		{"golang", "go"},
		{"TypeScript", "typescript"},
		{"JavaScript", "typescript"}, // JS uses TS configs
		{"javascript", "typescript"},
		{"Java", "java"},
		{"Rust", "rust"},
		{"C#", "csharp"},
		{"csharp", "csharp"},
		{"UnknownLang", "unknownlang"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLanguage(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildConfigKey(t *testing.T) {
	tests := []struct {
		language string
		domain   Domain
		expected string
	}{
		{"python", DomainWeb, "python_web"},
		{"go", DomainBackend, "go_backend"},
		{"typescript", DomainFrontend, "typescript_frontend"},
		{"python", DomainML, ConfigKeyMLProject},   // ML is domain-specific only
		{"go", DomainCLI, ConfigKeyCLITool},        // CLI is domain-specific only
		{"", DomainWeb, ConfigKeyDefault},
		{"python", DomainUnknown, ConfigKeyDefault},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := buildConfigKey(tt.language, tt.domain)
			if result != tt.expected {
				t.Errorf("buildConfigKey(%q, %q) = %q, expected %q",
					tt.language, tt.domain, result, tt.expected)
			}
		})
	}
}

func TestSelectConfigDetailed(t *testing.T) {
	metadata := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{"Flask==2.3.0"},
	}

	result := SelectConfigDetailed(metadata)

	if result.Language != "python" {
		t.Errorf("expected language 'python', got %s", result.Language)
	}
	if result.Domain != DomainWeb {
		t.Errorf("expected domain 'web', got %s", result.Domain)
	}
	if result.SelectedConfig.ConfigKey != ConfigKeyPythonWeb {
		t.Errorf("expected config %s, got %s", ConfigKeyPythonWeb, result.SelectedConfig.ConfigKey)
	}
	if result.FallbackUsed {
		t.Errorf("expected exact match (no fallback), but fallback was used")
	}
	if result.Reason == "" {
		t.Errorf("expected non-empty reason")
	}
}

func TestValidateConfigSelection(t *testing.T) {
	tests := []struct {
		name            string
		metadata        RepoMetadata
		config          RiskConfig
		expectWarnings  bool
		warningContains string
	}{
		{
			name: "Frontend with low coupling threshold",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"react": "^18.2.0",
					},
				},
			},
			config: RiskConfig{
				ConfigKey:          "test",
				CouplingThreshold:  8, // Too low for frontend
				CoChangeThreshold:  0.7,
				TestRatioThreshold: 0.3,
			},
			expectWarnings:  true,
			warningContains: "Frontend",
		},
		{
			name: "ML project with high test ratio",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"tensorflow==2.13.0"},
			},
			config: RiskConfig{
				ConfigKey:          "test",
				CouplingThreshold:  10,
				CoChangeThreshold:  0.7,
				TestRatioThreshold: 0.7, // Too high for ML
			},
			expectWarnings:  true,
			warningContains: "ML",
		},
		{
			name: "Backend with very high coupling",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				DirectoryNames:  []string{"api", "services"},
			},
			config: RiskConfig{
				ConfigKey:          "test",
				CouplingThreshold:  20, // Too high for backend
				CoChangeThreshold:  0.6,
				TestRatioThreshold: 0.5,
			},
			expectWarnings:  true,
			warningContains: "Backend",
		},
		{
			name: "Appropriate config for Python web",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"Flask==2.3.0"},
			},
			config: RiskConfig{
				ConfigKey:          ConfigKeyPythonWeb,
				CouplingThreshold:  15,
				CoChangeThreshold:  0.75,
				TestRatioThreshold: 0.4,
			},
			expectWarnings:  false,
			warningContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateConfigSelection(tt.metadata, tt.config)
			hasWarnings := len(warnings) > 0

			if hasWarnings != tt.expectWarnings {
				t.Errorf("expected warnings: %v, got warnings: %v (%d warnings)",
					tt.expectWarnings, hasWarnings, len(warnings))
				if len(warnings) > 0 {
					t.Logf("Warnings: %v", warnings)
				}
			}

			if tt.expectWarnings && len(warnings) > 0 {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, tt.warningContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning to contain %q, got warnings: %v",
						tt.warningContains, warnings)
				}
			}
		})
	}
}

func TestGetAllApplicableConfigs(t *testing.T) {
	tests := []struct {
		name           string
		metadata       RepoMetadata
		minConfigs     int
		shouldContain  []string
	}{
		{
			name: "Python web should have multiple options",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"Flask==2.3.0"},
			},
			minConfigs:    2, // At least exact + default
			shouldContain: []string{ConfigKeyPythonWeb, ConfigKeyDefault},
		},
		{
			name: "Unknown language should return default",
			metadata: RepoMetadata{
				PrimaryLanguage: "UnknownLang",
			},
			minConfigs:    1,
			shouldContain: []string{ConfigKeyDefault},
		},
		{
			name: "Go backend should have Go-specific options",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				DirectoryNames:  []string{"internal", "pkg"},
			},
			minConfigs:    2,
			shouldContain: []string{ConfigKeyGoBackend, ConfigKeyDefault},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs := GetAllApplicableConfigs(tt.metadata)

			if len(configs) < tt.minConfigs {
				t.Errorf("expected at least %d configs, got %d", tt.minConfigs, len(configs))
			}

			// Check for required configs
			configKeys := make(map[string]bool)
			for _, config := range configs {
				configKeys[config.ConfigKey] = true
			}

			for _, required := range tt.shouldContain {
				if !configKeys[required] {
					t.Errorf("expected configs to contain %s, but it was missing", required)
				}
			}

			// Verify no duplicates
			if len(configs) != len(configKeys) {
				t.Errorf("duplicate configs found in result")
			}
		})
	}
}

func TestTryLanguageFallback(t *testing.T) {
	tests := []struct {
		language      string
		domain        Domain
		expectSuccess bool
		expectedKey   string
	}{
		{"python", DomainUnknown, true, ConfigKeyPythonWeb},
		{"go", DomainUnknown, true, ConfigKeyGoBackend},
		{"typescript", DomainUnknown, true, ConfigKeyTypeScriptFrontend},
		{"java", DomainUnknown, true, ConfigKeyJavaBackend},
		{"rust", DomainUnknown, true, ConfigKeyRustBackend},
		{"unknownlang", DomainUnknown, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			config, err := tryLanguageFallback(tt.language, tt.domain)
			gotSuccess := (err == nil)

			if gotSuccess != tt.expectSuccess {
				t.Errorf("tryLanguageFallback(%q) success=%v, expected=%v, err=%v",
					tt.language, gotSuccess, tt.expectSuccess, err)
			}

			if tt.expectSuccess && config.ConfigKey != tt.expectedKey {
				t.Errorf("expected config %s, got %s", tt.expectedKey, config.ConfigKey)
			}
		})
	}
}

func TestTryDomainFallback(t *testing.T) {
	tests := []struct {
		domain        Domain
		expectSuccess bool
		expectedKey   string
	}{
		{DomainWeb, true, ConfigKeyPythonWeb},
		{DomainBackend, true, ConfigKeyGoBackend},
		{DomainFrontend, true, ConfigKeyTypeScriptFrontend},
		{DomainML, true, ConfigKeyMLProject},
		{DomainCLI, true, ConfigKeyCLITool},
		{DomainUnknown, false, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			config, err := tryDomainFallback(tt.domain)
			gotSuccess := (err == nil)

			if gotSuccess != tt.expectSuccess {
				t.Errorf("tryDomainFallback(%q) success=%v, expected=%v",
					tt.domain, gotSuccess, tt.expectSuccess)
			}

			if tt.expectSuccess && config.ConfigKey != tt.expectedKey {
				t.Errorf("expected config %s, got %s", tt.expectedKey, config.ConfigKey)
			}
		})
	}
}

// Integration test: omnara test repository
func TestSelectConfig_OmnaraRepo(t *testing.T) {
	omnaraMetadata := RepoMetadata{
		PrimaryLanguage: "TypeScript",
		PackageJSON: map[string]interface{}{
			"dependencies": map[string]interface{}{
				"next":  "14.0.4",
				"react": "^18",
			},
			"scripts": map[string]interface{}{
				"dev":   "next dev",
				"build": "next build",
			},
		},
		DirectoryNames: []string{"app", "components", "lib", "public"},
	}

	config := SelectConfig(omnaraMetadata)

	// Next.js should be detected as web
	if config.ConfigKey != ConfigKeyTypeScriptWeb {
		t.Errorf("omnara should use typescript_web config, got %s", config.ConfigKey)
	}

	// Verify thresholds are appropriate for full-stack web app
	if config.CouplingThreshold < 15 {
		t.Errorf("omnara coupling threshold too strict (%d), Next.js apps have high coupling",
			config.CouplingThreshold)
	}

	t.Logf("Omnara config selection:")
	t.Logf("  Config: %s", config.ConfigKey)
	t.Logf("  Coupling Threshold: %d", config.CouplingThreshold)
	t.Logf("  Co-Change Threshold: %.2f", config.CoChangeThreshold)
	t.Logf("  Test Ratio Threshold: %.2f", config.TestRatioThreshold)
}

// Benchmark config selection performance
func BenchmarkSelectConfig(b *testing.B) {
	metadata := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{"Flask==2.3.0"},
		DirectoryNames:  []string{"app", "templates"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SelectConfig(metadata)
	}
}

func BenchmarkSelectConfigWithReason(b *testing.B) {
	metadata := RepoMetadata{
		PrimaryLanguage: "Go",
		GoMod:           []string{"require github.com/gin-gonic/gin v1.9.0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SelectConfigWithReason(metadata)
	}
}
