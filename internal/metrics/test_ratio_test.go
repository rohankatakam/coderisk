package metrics

import (
	"testing"
)

func TestClassifyTestRatioRisk(t *testing.T) {
	tests := []struct {
		name     string
		ratio    float64
		expected RiskLevel
	}{
		{"No tests", 0.0, RiskLevelHigh},
		{"Very low coverage", 0.1, RiskLevelHigh},
		{"Low coverage - boundary", 0.29, RiskLevelHigh},
		{"Medium coverage - just above threshold", 0.3, RiskLevelMedium},
		{"Medium coverage - mid-range", 0.5, RiskLevelMedium},
		{"Medium coverage - boundary", 0.79, RiskLevelMedium},
		{"Excellent coverage - just above threshold", 0.8, RiskLevelLow},
		{"Very high coverage", 0.95, RiskLevelLow},
		{"Perfect coverage", 1.0, RiskLevelLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyTestRatioRisk(tt.ratio)
			if result != tt.expected {
				t.Errorf("classifyTestRatioRisk(%.2f) = %v, want %v", tt.ratio, result, tt.expected)
			}
		})
	}
}

func TestTestRatioResult_ShouldEscalate(t *testing.T) {
	tests := []struct {
		name     string
		ratio    float64
		expected bool
	}{
		{"No tests - escalate", 0.0, true},
		{"Low coverage - escalate", 0.2, true},
		{"Boundary - escalate", 0.29, true},
		{"Adequate coverage - no escalation", 0.3, false},
		{"Good coverage - no escalation", 0.5, false},
		{"Excellent coverage - no escalation", 0.9, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestRatioResult{Ratio: tt.ratio}
			if got := result.ShouldEscalate(); got != tt.expected {
				t.Errorf("ShouldEscalate() for ratio %.2f = %v, want %v", tt.ratio, got, tt.expected)
			}
		})
	}
}

func TestCalculateSmoothedRatio(t *testing.T) {
	tests := []struct {
		name      string
		testLOC   int
		sourceLOC int
		wantMin   float64
		wantMax   float64
	}{
		{"No tests, no source", 0, 0, 0.99, 1.01}, // (0+1)/(0+1) = 1.0
		{"No tests, has source", 0, 100, 0.0099, 0.0100}, // (0+1)/(100+1) ≈ 0.0099
		{"Equal LOC", 100, 100, 0.999, 1.001}, // (100+1)/(100+1) ≈ 1.0
		{"More tests than source", 200, 100, 1.99, 2.00}, // (200+1)/(100+1) ≈ 1.99
		{"Minimal test", 10, 100, 0.108, 0.110}, // (10+1)/(100+1) ≈ 0.109
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := calculateSmoothedRatio(tt.testLOC, tt.sourceLOC)

			if ratio < tt.wantMin || ratio > tt.wantMax {
				t.Errorf("calculateSmoothedRatio(%d, %d) = %.4f, want range [%.4f, %.4f]",
					tt.testLOC, tt.sourceLOC, ratio, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDiscoverTestFiles(t *testing.T) {
	tests := []struct {
		name         string
		sourceFile   string
		wantContains string
	}{
		{"Go file", "pkg/foo/bar.go", "bar_test.go"},
		{"Python file - suffix", "src/auth.py", "auth_test.py"},
		{"Python file - prefix", "src/auth.py", "test_auth.py"},
		{"JavaScript file - .test", "components/Button.js", "Button.test.js"},
		{"JavaScript file - .spec", "components/Button.js", "Button.spec.js"},
		{"TypeScript file", "services/api.ts", "api.test.ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFiles := discoverTestFiles(tt.sourceFile)

			if len(testFiles) == 0 {
				t.Errorf("discoverTestFiles(%q) returned empty array", tt.sourceFile)
			}

			// Check that at least one test file path contains the expected pattern
			found := false
			for _, tf := range testFiles {
				if containsMiddle(tf, tt.wantContains) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("discoverTestFiles(%q) = %v, should contain path with %q",
					tt.sourceFile, testFiles, tt.wantContains)
			}
		})
	}
}

func TestEstimateTestLOC(t *testing.T) {
	tests := []struct {
		name          string
		testFileCount int
		wantMin       int
		wantMax       int
	}{
		{"No test files", 0, 0, 0},
		{"One test file", 1, 40, 60},
		{"Multiple test files", 3, 120, 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := estimateTestLOC(tt.testFileCount)

			if loc < tt.wantMin || loc > tt.wantMax {
				t.Errorf("estimateTestLOC(%d) = %d, want range [%d, %d]",
					tt.testFileCount, loc, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestTestRatioResult_FormatEvidence(t *testing.T) {
	tests := []struct {
		name         string
		testFiles    []string
		ratio        float64
		sourceLOC    int
		testLOC      int
		wantContains string
	}{
		{
			name:         "No test files",
			testFiles:    []string{},
			ratio:        0.0,
			sourceLOC:    100,
			testLOC:      0,
			wantContains: "No test files",
		},
		{
			name:         "Has test files",
			testFiles:    []string{"foo_test.go"},
			ratio:        0.5,
			sourceLOC:    100,
			testLOC:      50,
			wantContains: "Test ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestRatioResult{
				TestFiles: tt.testFiles,
				Ratio:     tt.ratio,
				SourceLOC: tt.sourceLOC,
				TestLOC:   tt.testLOC,
			}
			evidence := result.FormatEvidence()

			if !containsMiddle(evidence, tt.wantContains) {
				t.Errorf("FormatEvidence() = %q, should contain %q", evidence, tt.wantContains)
			}
		})
	}
}
