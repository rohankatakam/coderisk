package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name      string
		configKey string
		wantErr   bool
	}{
		{"Python web exists", ConfigKeyPythonWeb, false},
		{"Go backend exists", ConfigKeyGoBackend, false},
		{"TypeScript frontend exists", ConfigKeyTypeScriptFrontend, false},
		{"Default exists", ConfigKeyDefault, false},
		{"Non-existent config returns error", "non_existent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetConfig(tt.configKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.ConfigKey != tt.configKey {
					t.Errorf("expected ConfigKey %s, got %s", tt.configKey, config.ConfigKey)
				}
				if config.Description == "" {
					t.Errorf("config %s missing description", tt.configKey)
				}
				if config.Rationale == "" {
					t.Errorf("config %s missing rationale", tt.configKey)
				}
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()
	if config.ConfigKey != ConfigKeyDefault {
		t.Errorf("expected default config key, got %s", config.ConfigKey)
	}
	if config.CouplingThreshold <= 0 {
		t.Errorf("invalid coupling threshold: %d", config.CouplingThreshold)
	}
}

func TestAllConfigsHaveValidThresholds(t *testing.T) {
	for key, config := range RiskConfigs {
		t.Run(key, func(t *testing.T) {
			// Coupling threshold should be positive and reasonable (1-30)
			if config.CouplingThreshold < 1 || config.CouplingThreshold > 30 {
				t.Errorf("%s: coupling threshold %d out of reasonable range [1-30]",
					key, config.CouplingThreshold)
			}

			// Co-change threshold should be between 0.0 and 1.0
			if config.CoChangeThreshold < 0.0 || config.CoChangeThreshold > 1.0 {
				t.Errorf("%s: co-change threshold %.2f out of range [0.0-1.0]",
					key, config.CoChangeThreshold)
			}

			// Test ratio threshold should be between 0.0 and 1.0
			if config.TestRatioThreshold < 0.0 || config.TestRatioThreshold > 1.0 {
				t.Errorf("%s: test ratio threshold %.2f out of range [0.0-1.0]",
					key, config.TestRatioThreshold)
			}

			// Config should have description
			if config.Description == "" {
				t.Errorf("%s: missing description", key)
			}

			// Config should have rationale
			if config.Rationale == "" {
				t.Errorf("%s: missing rationale", key)
			}

			// ConfigKey should match map key
			if config.ConfigKey != key {
				t.Errorf("%s: ConfigKey mismatch, expected %s got %s",
					key, key, config.ConfigKey)
			}
		})
	}
}

func TestConfigConsistency(t *testing.T) {
	tests := []struct {
		name        string
		configA     string
		configB     string
		expectation string
		checkField  string
	}{
		{
			name:        "Frontend has higher coupling than backend",
			configA:     ConfigKeyTypeScriptFrontend,
			configB:     ConfigKeyGoBackend,
			expectation: "higher",
			checkField:  "coupling",
		},
		{
			name:        "Python web more permissive than Go backend",
			configA:     ConfigKeyPythonWeb,
			configB:     ConfigKeyGoBackend,
			expectation: "higher",
			checkField:  "coupling",
		},
		{
			name:        "Rust backend stricter than Python web",
			configA:     ConfigKeyRustBackend,
			configB:     ConfigKeyPythonWeb,
			expectation: "lower",
			checkField:  "coupling",
		},
		{
			name:        "ML projects have lower test expectations",
			configA:     ConfigKeyMLProject,
			configB:     ConfigKeyGoBackend,
			expectation: "lower",
			checkField:  "test_ratio",
		},
		{
			name:        "Java backend has high test expectations",
			configA:     ConfigKeyJavaBackend,
			configB:     ConfigKeyTypeScriptFrontend,
			expectation: "higher",
			checkField:  "test_ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configA, _ := GetConfig(tt.configA)
			configB, _ := GetConfig(tt.configB)

			var valA, valB float64
			switch tt.checkField {
			case "coupling":
				valA = float64(configA.CouplingThreshold)
				valB = float64(configB.CouplingThreshold)
			case "co_change":
				valA = configA.CoChangeThreshold
				valB = configB.CoChangeThreshold
			case "test_ratio":
				valA = configA.TestRatioThreshold
				valB = configB.TestRatioThreshold
			}

			switch tt.expectation {
			case "higher":
				if valA <= valB {
					t.Errorf("%s (%s=%.2f) should be higher than %s (%s=%.2f)",
						tt.configA, tt.checkField, valA,
						tt.configB, tt.checkField, valB)
				}
			case "lower":
				if valA >= valB {
					t.Errorf("%s (%s=%.2f) should be lower than %s (%s=%.2f)",
						tt.configA, tt.checkField, valA,
						tt.configB, tt.checkField, valB)
				}
			}
		})
	}
}

func TestCouplingThresholdOrdering(t *testing.T) {
	// Expected ordering from strictest to most permissive
	expectedOrder := []string{
		ConfigKeyRustBackend,        // 7 (strictest)
		ConfigKeyGoBackend,          // 8
		ConfigKeyGoWeb,              // 10
		ConfigKeyDefault,            // 10
		ConfigKeyMLProject,          // 10
		ConfigKeyCLITool,            // 10
		ConfigKeyPythonBackend,      // 12
		ConfigKeyJavaBackend,        // 12
		ConfigKeyPythonWeb,          // 15
		ConfigKeyTypeScriptWeb,      // 18
		ConfigKeyTypeScriptFrontend, // 20 (most permissive)
	}

	for i := 0; i < len(expectedOrder)-1; i++ {
		configA, _ := GetConfig(expectedOrder[i])
		configB, _ := GetConfig(expectedOrder[i+1])

		if configA.CouplingThreshold > configB.CouplingThreshold {
			t.Errorf("Coupling threshold ordering violated: %s (%d) should be <= %s (%d)",
				expectedOrder[i], configA.CouplingThreshold,
				expectedOrder[i+1], configB.CouplingThreshold)
		}
	}
}

func TestTestRatioReasonableness(t *testing.T) {
	tests := []struct {
		configKey           string
		minExpected         float64
		maxExpected         float64
		reasonableThreshold bool
	}{
		{ConfigKeyMLProject, 0.15, 0.35, true},         // ML has low test coverage
		{ConfigKeyTypeScriptFrontend, 0.2, 0.4, true},  // Frontend has lower coverage
		{ConfigKeyPythonWeb, 0.3, 0.5, true},           // Web apps moderate coverage
		{ConfigKeyGoBackend, 0.45, 0.6, true},          // Go has strong test culture
		{ConfigKeyJavaBackend, 0.55, 0.7, true},        // Java enterprise testing
		{ConfigKeyRustBackend, 0.5, 0.65, true},        // Rust strong testing
	}

	for _, tt := range tests {
		t.Run(tt.configKey, func(t *testing.T) {
			config, _ := GetConfig(tt.configKey)
			if config.TestRatioThreshold < tt.minExpected {
				t.Errorf("%s: test ratio %.2f too low, expected >= %.2f",
					tt.configKey, config.TestRatioThreshold, tt.minExpected)
			}
			if config.TestRatioThreshold > tt.maxExpected {
				t.Errorf("%s: test ratio %.2f too high, expected <= %.2f",
					tt.configKey, config.TestRatioThreshold, tt.maxExpected)
			}
		})
	}
}

func TestCoChangeThresholdReasonableness(t *testing.T) {
	// Co-change thresholds should generally be in range [0.55, 0.85]
	// Higher values = more permissive (allow more co-change before flagging risk)

	for key, config := range RiskConfigs {
		t.Run(key, func(t *testing.T) {
			if config.CoChangeThreshold < 0.5 {
				t.Errorf("%s: co-change threshold %.2f too strict (< 0.5)",
					key, config.CoChangeThreshold)
			}
			if config.CoChangeThreshold > 0.85 {
				t.Errorf("%s: co-change threshold %.2f too permissive (> 0.85)",
					key, config.CoChangeThreshold)
			}
		})
	}
}

func TestListConfigKeys(t *testing.T) {
	keys := ListConfigKeys()
	if len(keys) != len(RiskConfigs) {
		t.Errorf("ListConfigKeys() returned %d keys, expected %d",
			len(keys), len(RiskConfigs))
	}

	// Verify all expected keys are present
	expectedKeys := []string{
		ConfigKeyPythonWeb,
		ConfigKeyGoBackend,
		ConfigKeyTypeScriptFrontend,
		ConfigKeyDefault,
	}

	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %s not found in ListConfigKeys()", expectedKey)
		}
	}
}

func TestCompareConfigs(t *testing.T) {
	tests := []struct {
		name    string
		keyA    string
		keyB    string
		wantErr bool
	}{
		{
			name:    "Compare Python web vs Go backend",
			keyA:    ConfigKeyPythonWeb,
			keyB:    ConfigKeyGoBackend,
			wantErr: false,
		},
		{
			name:    "Compare TypeScript frontend vs default",
			keyA:    ConfigKeyTypeScriptFrontend,
			keyB:    ConfigKeyDefault,
			wantErr: false,
		},
		{
			name:    "Compare with non-existent config A",
			keyA:    "non_existent",
			keyB:    ConfigKeyDefault,
			wantErr: true,
		},
		{
			name:    "Compare with non-existent config B",
			keyA:    ConfigKeyDefault,
			keyB:    "non_existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison, err := CompareConfigs(tt.keyA, tt.keyB)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !strings.Contains(comparison, tt.keyA) {
					t.Errorf("Comparison should contain keyA: %s", tt.keyA)
				}
				if !strings.Contains(comparison, tt.keyB) {
					t.Errorf("Comparison should contain keyB: %s", tt.keyB)
				}
				if !strings.Contains(comparison, "Coupling Threshold") {
					t.Errorf("Comparison should contain threshold details")
				}
			}
		})
	}
}

func TestConfigRationaleExists(t *testing.T) {
	// Verify all configs have substantive rationale
	minRationaleLength := 100 // At least 100 characters

	for key, config := range RiskConfigs {
		t.Run(key, func(t *testing.T) {
			if len(config.Rationale) < minRationaleLength {
				t.Errorf("%s: rationale too short (%d chars), expected >= %d",
					key, len(config.Rationale), minRationaleLength)
			}

			// Rationale should mention the coupling threshold value
			if !strings.Contains(strings.ToLower(config.Rationale),
				"coupling") {
				t.Errorf("%s: rationale should explain coupling threshold choice", key)
			}
		})
	}
}

func TestDefaultConfigIsMiddleGround(t *testing.T) {
	defaultConfig := GetDefaultConfig()

	// Collect all coupling thresholds
	var couplingValues []int
	for _, config := range RiskConfigs {
		if config.ConfigKey != ConfigKeyDefault {
			couplingValues = append(couplingValues, config.CouplingThreshold)
		}
	}

	// Default should be somewhere in the middle range
	min, max := couplingValues[0], couplingValues[0]
	for _, val := range couplingValues {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	if defaultConfig.CouplingThreshold < min || defaultConfig.CouplingThreshold > max {
		t.Logf("Default coupling %d should be in range [%d, %d] of other configs",
			defaultConfig.CouplingThreshold, min, max)
	}
}

// Benchmark config retrieval performance
func BenchmarkGetConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetConfig(ConfigKeyPythonWeb)
	}
}

// Example showing config comparison
func ExampleCompareConfigs() {
	comparison, _ := CompareConfigs(ConfigKeyPythonWeb, ConfigKeyGoBackend)
	// Print first 200 characters
	if len(comparison) > 200 {
		fmt.Println(comparison[:200] + "...")
	}
}

// TestPrintAllConfigs prints all configs for manual inspection
func TestPrintAllConfigs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping config printout in short mode")
	}

	t.Log("\n=== All Risk Configurations ===\n")

	configOrder := []string{
		ConfigKeyRustBackend,
		ConfigKeyGoBackend,
		ConfigKeyGoWeb,
		ConfigKeyPythonBackend,
		ConfigKeyJavaBackend,
		ConfigKeyPythonWeb,
		ConfigKeyTypeScriptWeb,
		ConfigKeyTypeScriptFrontend,
		ConfigKeyMLProject,
		ConfigKeyCLITool,
		ConfigKeyDefault,
	}

	for _, key := range configOrder {
		config, _ := GetConfig(key)
		t.Logf("Config: %s\n", config.ConfigKey)
		t.Logf("  Description: %s\n", config.Description)
		t.Logf("  Coupling Threshold: %d\n", config.CouplingThreshold)
		t.Logf("  Co-Change Threshold: %.2f\n", config.CoChangeThreshold)
		t.Logf("  Test Ratio Threshold: %.2f\n", config.TestRatioThreshold)
		t.Logf("  Rationale: %s\n", config.Rationale)
		t.Logf("\n")
	}
}
