package metrics

import (
	"testing"
)

func TestClassifyCouplingRisk(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected RiskLevel
	}{
		{"No coupling", 0, RiskLevelLow},
		{"Low coupling - boundary", 5, RiskLevelLow},
		{"Medium coupling - just above threshold", 6, RiskLevelMedium},
		{"Medium coupling - mid-range", 8, RiskLevelMedium},
		{"Medium coupling - boundary", 10, RiskLevelMedium},
		{"High coupling - just above threshold", 11, RiskLevelHigh},
		{"High coupling - well above threshold", 20, RiskLevelHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyCouplingRisk(tt.count)
			if result != tt.expected {
				t.Errorf("classifyCouplingRisk(%d) = %v, want %v", tt.count, result, tt.expected)
			}
		})
	}
}

func TestCouplingResult_ShouldEscalate(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected bool
	}{
		{"Low coupling - no escalation", 5, false},
		{"Medium coupling - no escalation", 10, false},
		{"High coupling - escalate", 11, true},
		{"Very high coupling - escalate", 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CouplingResult{Count: tt.count}
			if got := result.ShouldEscalate(); got != tt.expected {
				t.Errorf("ShouldEscalate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCouplingResult_FormatEvidence(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		riskLevel RiskLevel
		contains  string
	}{
		{"Low coupling evidence", 3, RiskLevelLow, "3 other files"},
		{"Medium coupling evidence", 8, RiskLevelMedium, "8 other files"},
		{"High coupling evidence", 15, RiskLevelHigh, "15 other files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CouplingResult{
				Count:     tt.count,
				RiskLevel: tt.riskLevel,
			}
			evidence := result.FormatEvidence()
			if len(evidence) == 0 {
				t.Error("FormatEvidence() returned empty string")
			}
			// Check that evidence contains the count
			if !contains(evidence, tt.contains) {
				t.Errorf("FormatEvidence() = %q, should contain %q", evidence, tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
