package metrics

import (
	"testing"

	"github.com/coderisk/coderisk-go/internal/analysis/config"
)

func TestClassifyCouplingWithThreshold(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		threshold int
		expected  RiskLevel
	}{
		// Python web (threshold: 15)
		{"Python web - low coupling", 5, 15, RiskLevelLow},     // ≤ 7.5 (50% of 15)
		{"Python web - medium coupling", 10, 15, RiskLevelMedium}, // 7.5-15
		{"Python web - high coupling", 16, 15, RiskLevelHigh},  // > 15

		// Go backend (threshold: 8)
		{"Go backend - low coupling", 3, 8, RiskLevelLow},      // ≤ 4 (50% of 8)
		{"Go backend - medium coupling", 6, 8, RiskLevelMedium}, // 4-8
		{"Go backend - high coupling", 9, 8, RiskLevelHigh},    // > 8

		// TypeScript frontend (threshold: 20)
		{"TS frontend - low coupling", 8, 20, RiskLevelLow},     // ≤ 10 (50% of 20)
		{"TS frontend - medium coupling", 15, 20, RiskLevelMedium}, // 10-20
		{"TS frontend - high coupling", 21, 20, RiskLevelHigh},  // > 20
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyCouplingWithThreshold(tt.count, tt.threshold)
			if result != tt.expected {
				t.Errorf("ClassifyCouplingWithThreshold(%d, %d) = %s, expected %s",
					tt.count, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestClassifyCoChangeWithThreshold(t *testing.T) {
	tests := []struct {
		name      string
		frequency float64
		threshold float64
		expected  RiskLevel
	}{
		// Python web (threshold: 0.75)
		{"Python web - low co-change", 0.3, 0.75, RiskLevelLow},     // ≤ 0.375 (50% of 0.75)
		{"Python web - medium co-change", 0.5, 0.75, RiskLevelMedium}, // 0.375-0.75
		{"Python web - high co-change", 0.8, 0.75, RiskLevelHigh},   // > 0.75

		// Go backend (threshold: 0.6)
		{"Go backend - low co-change", 0.2, 0.6, RiskLevelLow},      // ≤ 0.3 (50% of 0.6)
		{"Go backend - medium co-change", 0.4, 0.6, RiskLevelMedium}, // 0.3-0.6
		{"Go backend - high co-change", 0.7, 0.6, RiskLevelHigh},    // > 0.6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyCoChangeWithThreshold(tt.frequency, tt.threshold)
			if result != tt.expected {
				t.Errorf("ClassifyCoChangeWithThreshold(%.2f, %.2f) = %s, expected %s",
					tt.frequency, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestClassifyTestRatioWithThreshold(t *testing.T) {
	tests := []struct {
		name      string
		ratio     float64
		threshold float64
		expected  RiskLevel
	}{
		// Python web (threshold: 0.4)
		{"Python web - insufficient coverage", 0.3, 0.4, RiskLevelHigh},   // < 0.4
		{"Python web - adequate coverage", 0.5, 0.4, RiskLevelMedium},      // 0.4-0.7
		{"Python web - excellent coverage", 0.8, 0.4, RiskLevelLow},        // ≥ 0.7

		// Go backend (threshold: 0.5)
		{"Go backend - insufficient coverage", 0.4, 0.5, RiskLevelHigh},   // < 0.5
		{"Go backend - adequate coverage", 0.6, 0.5, RiskLevelMedium},      // 0.5-0.8
		{"Go backend - excellent coverage", 0.9, 0.5, RiskLevelLow},        // ≥ 0.8

		// ML project (threshold: 0.25)
		{"ML - insufficient coverage", 0.2, 0.25, RiskLevelHigh},          // < 0.25
		{"ML - adequate coverage", 0.4, 0.25, RiskLevelMedium},             // 0.25-0.55
		{"ML - excellent coverage", 0.6, 0.25, RiskLevelLow},               // ≥ 0.55
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyTestRatioWithThreshold(tt.ratio, tt.threshold)
			if result != tt.expected {
				t.Errorf("ClassifyTestRatioWithThreshold(%.2f, %.2f) = %s, expected %s",
					tt.ratio, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestShouldEscalateWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		result     *Phase1Result
		config     config.RiskConfig
		shouldEscalate bool
	}{
		{
			name: "Python web - normal file (no escalation)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 10},   // ≤ 15 threshold
				CoChange:  &CoChangeResult{MaxFrequency: 0.5}, // ≤ 0.75 threshold
				TestRatio: &TestRatioResult{Ratio: 0.45}, // ≥ 0.4 threshold
			},
			config: config.RiskConfigs[config.ConfigKeyPythonWeb],
			shouldEscalate: false,
		},
		{
			name: "Python web - high coupling (escalate)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 16},   // > 15 threshold
				CoChange:  &CoChangeResult{MaxFrequency: 0.5},
				TestRatio: &TestRatioResult{Ratio: 0.45},
			},
			config: config.RiskConfigs[config.ConfigKeyPythonWeb],
			shouldEscalate: true,
		},
		{
			name: "Go backend - normal file (no escalation)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 6},    // ≤ 8 threshold
				CoChange:  &CoChangeResult{MaxFrequency: 0.4}, // ≤ 0.6 threshold
				TestRatio: &TestRatioResult{Ratio: 0.55}, // ≥ 0.5 threshold
			},
			config: config.RiskConfigs[config.ConfigKeyGoBackend],
			shouldEscalate: false,
		},
		{
			name: "Go backend - high coupling (escalate)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 9},    // > 8 threshold
				CoChange:  &CoChangeResult{MaxFrequency: 0.4},
				TestRatio: &TestRatioResult{Ratio: 0.55},
			},
			config: config.RiskConfigs[config.ConfigKeyGoBackend],
			shouldEscalate: true,
		},
		{
			name: "TypeScript frontend - normal high coupling (no escalation)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 18},   // ≤ 20 threshold (normal for frontend)
				CoChange:  &CoChangeResult{MaxFrequency: 0.7},
				TestRatio: &TestRatioResult{Ratio: 0.35}, // ≥ 0.3 threshold
			},
			config: config.RiskConfigs[config.ConfigKeyTypeScriptFrontend],
			shouldEscalate: false,
		},
		{
			name: "TypeScript frontend - extremely high coupling (escalate)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 25},   // > 20 threshold
				CoChange:  &CoChangeResult{MaxFrequency: 0.7},
				TestRatio: &TestRatioResult{Ratio: 0.35},
			},
			config: config.RiskConfigs[config.ConfigKeyTypeScriptFrontend],
			shouldEscalate: true,
		},
		{
			name: "ML project - low test coverage is acceptable (no escalation)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 8},
				CoChange:  &CoChangeResult{MaxFrequency: 0.5},
				TestRatio: &TestRatioResult{Ratio: 0.3}, // ≥ 0.25 threshold (acceptable for ML)
			},
			config: config.RiskConfigs[config.ConfigKeyMLProject],
			shouldEscalate: false,
		},
		{
			name: "ML project - extremely low test coverage (escalate)",
			result: &Phase1Result{
				Coupling:  &CouplingResult{Count: 8},
				CoChange:  &CoChangeResult{MaxFrequency: 0.5},
				TestRatio: &TestRatioResult{Ratio: 0.2}, // < 0.25 threshold
			},
			config: config.RiskConfigs[config.ConfigKeyMLProject],
			shouldEscalate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldEscalateWithConfig(tt.result, tt.config)
			if result != tt.shouldEscalate {
				t.Errorf("ShouldEscalateWithConfig() = %v, expected %v", result, tt.shouldEscalate)
				t.Logf("Config: %s", tt.config.ConfigKey)
				t.Logf("  Coupling: %d (threshold: %d)", tt.result.Coupling.Count, tt.config.CouplingThreshold)
				t.Logf("  Co-Change: %.2f (threshold: %.2f)", tt.result.CoChange.MaxFrequency, tt.config.CoChangeThreshold)
				t.Logf("  Test Ratio: %.2f (threshold: %.2f)", tt.result.TestRatio.Ratio, tt.config.TestRatioThreshold)
			}
		})
	}
}

func TestAdaptiveThresholds_CompareConfigs(t *testing.T) {
	// Same file metrics, different configs → different escalation decisions
	fileMetrics := &Phase1Result{
		Coupling:  &CouplingResult{Count: 12},
		CoChange:  &CoChangeResult{MaxFrequency: 0.65},
		TestRatio: &TestRatioResult{Ratio: 0.35},
	}

	configs := []struct {
		name            string
		config          config.RiskConfig
		expectedEscalate bool
		reason          string
	}{
		{
			name:            "Python Web",
			config:          config.RiskConfigs[config.ConfigKeyPythonWeb],
			expectedEscalate: true,  // 0.35 < 0.4 test ratio threshold (escalate)
			reason:          "Python web: test ratio below threshold (0.35 < 0.4)",
		},
		{
			name:            "Go Backend",
			config:          config.RiskConfigs[config.ConfigKeyGoBackend],
			expectedEscalate: true,  // 12 > 8 (coupling exceeds threshold)
			reason:          "Go backend has stricter coupling threshold (8)",
		},
		{
			name:            "TypeScript Frontend",
			config:          config.RiskConfigs[config.ConfigKeyTypeScriptFrontend],
			expectedEscalate: false, // 12 ≤ 20 (frontend allows very high coupling)
			reason:          "TypeScript frontend allows highest coupling (threshold 20)",
		},
	}

	t.Log("\n=== Adaptive Threshold Comparison ===")
	t.Logf("File Metrics: Coupling=%d, CoChange=%.2f, TestRatio=%.2f\n",
		fileMetrics.Coupling.Count,
		fileMetrics.CoChange.MaxFrequency,
		fileMetrics.TestRatio.Ratio)

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			shouldEscalate := ShouldEscalateWithConfig(fileMetrics, tc.config)

			t.Logf("Config: %s", tc.config.ConfigKey)
			t.Logf("  Thresholds: Coupling=%d, CoChange=%.2f, TestRatio=%.2f",
				tc.config.CouplingThreshold,
				tc.config.CoChangeThreshold,
				tc.config.TestRatioThreshold)
			t.Logf("  Escalate: %v (%s)", shouldEscalate, tc.reason)

			if shouldEscalate != tc.expectedEscalate {
				t.Errorf("Expected escalate=%v, got %v", tc.expectedEscalate, shouldEscalate)
			}
		})
	}
}

func TestFormatSummaryWithConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping summary format test in short mode")
	}

	pythonWebConfig := config.RiskConfigs[config.ConfigKeyPythonWeb]

	adaptiveResult := &AdaptivePhase1Result{
		Phase1Result: &Phase1Result{
			FilePath: "app/models/user.py",
			Coupling: &CouplingResult{
				FilePath:  "app/models/user.py",
				Count:     12,
				RiskLevel: RiskLevelMedium,
			},
			CoChange: &CoChangeResult{
				FilePath:     "app/models/user.py",
				MaxFrequency: 0.65,
				RiskLevel:    RiskLevelMedium,
			},
			TestRatio: &TestRatioResult{
				FilePath:  "app/models/user.py",
				Ratio:     0.35,
				RiskLevel: RiskLevelHigh,
			},
			OverallRisk:    RiskLevelHigh,
			ShouldEscalate: false,
			DurationMS:     125,
		},
		SelectedConfig: pythonWebConfig,
		ConfigReason:   "Exact match: language=python, domain=web",
	}

	summary := adaptiveResult.FormatSummaryWithConfig()

	t.Log("\n=== Adaptive Summary Example ===")
	t.Log(summary)

	// Verify summary contains key information
	if summary == "" {
		t.Error("Summary should not be empty")
	}
}
