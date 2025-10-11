package phase0

import (
	"testing"
)

func TestDetectEnvironment(t *testing.T) {
	tests := []struct {
		name              string
		filePath          string
		expectEnv         EnvironmentType
		expectProduction  bool
		expectEscalate    bool
		expectRiskLevel   string
		expectIsConfig    bool
		description       string
	}{
		// PRODUCTION CONFIGURATIONS - Should force escalate to CRITICAL
		{
			name:             ".env.production",
			filePath:         ".env.production",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production env file should be CRITICAL",
		},
		{
			name:             "prod.config.yaml",
			filePath:         "config/prod.config.yaml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production config YAML should be CRITICAL",
		},
		{
			name:             "production.json",
			filePath:         "config/production.json",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production JSON config should be CRITICAL",
		},
		{
			name:             "prod directory config",
			filePath:         "config/prod/database.yaml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Config in prod directory should be CRITICAL",
		},
		{
			name:             "production.yml",
			filePath:         "app/production.yml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production YML should be CRITICAL",
		},
		{
			name:             ".env.prod",
			filePath:         ".env.prod",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Short prod env file should be CRITICAL",
		},

		// STAGING CONFIGURATIONS - Should escalate to HIGH
		{
			name:             ".env.staging",
			filePath:         ".env.staging",
			expectEnv:        EnvStaging,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Staging env file should be HIGH",
		},
		{
			name:             "staging.config.yaml",
			filePath:         "config/staging.config.yaml",
			expectEnv:        EnvStaging,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Staging config should be HIGH",
		},
		{
			name:             "stage directory config",
			filePath:         "config/stage/api.json",
			expectEnv:        EnvStaging,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Config in stage directory should be HIGH",
		},

		// DEVELOPMENT CONFIGURATIONS - Should NOT escalate (LOW)
		{
			name:             ".env.local",
			filePath:         ".env.local",
			expectEnv:        EnvDevelopment,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Local env file should be LOW",
		},
		{
			name:             ".env.development",
			filePath:         ".env.development",
			expectEnv:        EnvDevelopment,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Development env file should be LOW",
		},
		{
			name:             "dev.config.yaml",
			filePath:         "config/dev.config.yaml",
			expectEnv:        EnvDevelopment,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Dev config should be LOW",
		},
		{
			name:             "local.yaml",
			filePath:         "config/local.yaml",
			expectEnv:        EnvDevelopment,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Local config should be LOW",
		},
		{
			name:             "development.json",
			filePath:         "config/development.json",
			expectEnv:        EnvDevelopment,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Development JSON should be LOW",
		},

		// TEST CONFIGURATIONS - Should NOT escalate (LOW)
		{
			name:             ".env.test",
			filePath:         ".env.test",
			expectEnv:        EnvTest,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Test env file should be LOW",
		},
		{
			name:             "test.config.yaml",
			filePath:         "config/test.config.yaml",
			expectEnv:        EnvTest,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "LOW",
			expectIsConfig:   true,
			description:      "Test config should be LOW",
		},

		// UNKNOWN ENVIRONMENT CONFIGS - Should escalate as caution (HIGH)
		{
			name:             "config.yaml without environment",
			filePath:         "config/database.yaml",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Unknown environment config should be HIGH (cautious)",
		},
		{
			name:             "generic .env file",
			filePath:         ".env",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Generic .env should be HIGH (could be production)",
		},
		{
			name:             "config directory without env marker",
			filePath:         "config/api.json",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   true,
			expectRiskLevel:  "HIGH",
			expectIsConfig:   true,
			description:      "Config without environment marker should be HIGH",
		},

		// NON-CONFIGURATION FILES - Should return unknown/not config
		{
			name:             "Go source file",
			filePath:         "internal/handlers/auth.go",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "",
			expectIsConfig:   false,
			description:      "Go source should not be detected as config",
		},
		{
			name:             "Python source file",
			filePath:         "src/services/payment.py",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "",
			expectIsConfig:   false,
			description:      "Python source should not be detected as config",
		},
		{
			name:             "JavaScript source file",
			filePath:         "src/app.js",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "",
			expectIsConfig:   false,
			description:      "JavaScript source should not be detected as config",
		},
		{
			name:             "Documentation file",
			filePath:         "README.md",
			expectEnv:        EnvUnknown,
			expectProduction: false,
			expectEscalate:   false,
			expectRiskLevel:  "",
			expectIsConfig:   false,
			description:      "Documentation should not be detected as config",
		},

		// VARIOUS CONFIG FILE TYPES
		{
			name:             "TOML config",
			filePath:         "config/production.toml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "TOML production config should be CRITICAL",
		},
		{
			name:             "INI config",
			filePath:         "config/prod.ini",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "INI production config should be CRITICAL",
		},
		{
			name:             "Properties file",
			filePath:         "application-production.properties",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Properties production config should be CRITICAL",
		},
		{
			name:             "XML config",
			filePath:         "config/production.xml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "XML production config should be CRITICAL",
		},

		// DOCKERFILE AND DOCKER-COMPOSE
		{
			name:             "Dockerfile.production",
			filePath:         "Dockerfile.production",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production Dockerfile should be CRITICAL",
		},
		{
			name:             "docker-compose.prod.yml",
			filePath:         "docker-compose.prod.yml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production docker-compose should be CRITICAL",
		},

		// EDGE CASES
		{
			name:             "Uppercase PRODUCTION",
			filePath:         "config/PRODUCTION.yaml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Case-insensitive production detection",
		},
		{
			name:             "Mixed case Production",
			filePath:         "config/Production.json",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Case-insensitive production detection",
		},
		{
			name:             "Deep nested config",
			filePath:         "app/config/environments/prod/database.yaml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Deep path production config should be CRITICAL",
		},
		{
			name:             "File with 'prod' substring in dev directory",
			filePath:         "config/dev/product.yaml",
			expectEnv:        EnvProduction,
			expectProduction: true,
			expectEscalate:   true,
			expectRiskLevel:  "CRITICAL",
			expectIsConfig:   true,
			description:      "Production pattern 'prod' matches even in dev directory (safety first)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEnvironment(tt.filePath)

			if result.Environment != tt.expectEnv {
				t.Errorf("Environment = %v, want %v\nDescription: %s",
					result.Environment, tt.expectEnv, tt.description)
			}

			if result.IsProduction != tt.expectProduction {
				t.Errorf("IsProduction = %v, want %v\nDescription: %s",
					result.IsProduction, tt.expectProduction, tt.description)
			}

			if result.ForceEscalate != tt.expectEscalate {
				t.Errorf("ForceEscalate = %v, want %v\nDescription: %s",
					result.ForceEscalate, tt.expectEscalate, tt.description)
			}

			if result.IsConfiguration != tt.expectIsConfig {
				t.Errorf("IsConfiguration = %v, want %v\nDescription: %s",
					result.IsConfiguration, tt.expectIsConfig, tt.description)
			}

			riskLevel := result.GetRiskLevel()
			if riskLevel != tt.expectRiskLevel {
				t.Errorf("GetRiskLevel = %v, want %v\nDescription: %s",
					riskLevel, tt.expectRiskLevel, tt.description)
			}

			if result.IsConfiguration && result.Reason == "" {
				t.Error("Expected non-empty reason for configuration file detection")
			}

			t.Logf("Result: Env=%v, Prod=%v, Escalate=%v, Risk=%v, Pattern=%v, Reason=%s",
				result.Environment,
				result.IsProduction,
				result.ForceEscalate,
				riskLevel,
				result.MatchedPattern,
				result.Reason)
		})
	}
}

func TestIsConfigurationFile(t *testing.T) {
	tests := []struct {
		filePath string
		expected bool
		reason   string
	}{
		// Configuration files (should return true)
		{"config.yaml", true, "YAML extension"},
		{"config.yml", true, "YML extension"},
		{"config.json", true, "JSON extension"},
		{"config.toml", true, "TOML extension"},
		{"config.ini", true, "INI extension"},
		{"app.conf", true, "CONF extension"},
		{"server.config", true, "CONFIG extension"},
		{"application.properties", true, "Properties extension"},
		{"config.xml", true, "XML extension"},
		{".env", true, ".env file"},
		{".env.local", true, ".env variant"},
		{"Dockerfile", true, "Dockerfile"},
		{"docker-compose.yml", true, "docker-compose"},
		{"Makefile", true, "Makefile"},
		{"tsconfig.json", true, "tsconfig"},
		{"webpack.config.js", true, "webpack.config"},
		{"config/database.yaml", true, "In config directory"},
		{"configs/api.json", true, "In configs directory"},
		{".config/app.json", true, "In .config directory"},

		// Non-configuration files (should return false)
		{"main.go", false, "Go source"},
		{"app.py", false, "Python source"},
		{"index.js", false, "JavaScript source"},
		{"component.tsx", false, "TypeScript React"},
		{"README.md", false, "Documentation"},
		{"test.txt", false, "Text file"},
		{"src/handlers/auth.go", false, "Source in src"},
		{"internal/models/user.go", false, "Source in internal"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := IsConfigurationFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsConfigurationFile(%s) = %v, want %v (reason: %s)",
					tt.filePath, result, tt.expected, tt.reason)
			}
		})
	}
}

func TestIsProductionConfig(t *testing.T) {
	// Quick test for the simplified helper function
	tests := []struct {
		filePath string
		expected bool
	}{
		{".env.production", true},
		{"config/prod.yaml", true},
		{"production.json", true},
		{".env.development", false},
		{"config/dev.yaml", false},
		{"main.go", false},
		{".env.staging", false},
		{"config/test.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := IsProductionConfig(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsProductionConfig(%s) = %v, want %v",
					tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestEnvironmentPriority(t *testing.T) {
	// Test that environment detection follows correct priority
	// Production > Staging > Test > Development
	tests := []struct {
		name        string
		filePath    string
		expectedEnv EnvironmentType
		description string
	}{
		{
			name:        "prod overrides dev in path",
			filePath:    "config/dev/prod.yaml",
			expectedEnv: EnvProduction,
			description: "Production keyword should take priority",
		},
		{
			name:        "staging overrides test in path",
			filePath:    "config/test/staging.yaml",
			expectedEnv: EnvStaging,
			description: "Staging should take priority over test",
		},
		{
			name:        "test overrides dev in path",
			filePath:    "config/dev/test.yaml",
			expectedEnv: EnvTest,
			description: "Test should take priority over development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEnvironment(tt.filePath)
			if result.Environment != tt.expectedEnv {
				t.Errorf("Environment = %v, want %v\nDescription: %s",
					result.Environment, tt.expectedEnv, tt.description)
			}
		})
	}
}

// Benchmark environment detection performance
func BenchmarkDetectEnvironment(b *testing.B) {
	paths := []string{
		".env.production",
		"config/prod.yaml",
		".env.development",
		"config/staging.json",
		"main.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = DetectEnvironment(path)
		}
	}
}
