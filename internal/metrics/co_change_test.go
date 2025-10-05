package metrics

import (
	"testing"
)

func TestClassifyCoChangeRisk(t *testing.T) {
	tests := []struct {
		name      string
		frequency float64
		expected  RiskLevel
	}{
		{"No co-change", 0.0, RiskLevelLow},
		{"Very low co-change", 0.1, RiskLevelLow},
		{"Low co-change - boundary", 0.3, RiskLevelLow},
		{"Medium co-change - just above threshold", 0.31, RiskLevelMedium},
		{"Medium co-change - mid-range", 0.5, RiskLevelMedium},
		{"Medium co-change - boundary", 0.7, RiskLevelMedium},
		{"High co-change - just above threshold", 0.71, RiskLevelHigh},
		{"Very high co-change", 0.9, RiskLevelHigh},
		{"Maximum co-change", 1.0, RiskLevelHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyCoChangeRisk(tt.frequency)
			if result != tt.expected {
				t.Errorf("classifyCoChangeRisk(%.2f) = %v, want %v", tt.frequency, result, tt.expected)
			}
		})
	}
}

func TestCoChangeResult_ShouldEscalate(t *testing.T) {
	tests := []struct {
		name      string
		frequency float64
		expected  bool
	}{
		{"Low frequency - no escalation", 0.3, false},
		{"Medium frequency - no escalation", 0.5, false},
		{"Boundary - no escalation", 0.7, false},
		{"High frequency - escalate", 0.71, true},
		{"Very high frequency - escalate", 0.9, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CoChangeResult{MaxFrequency: tt.frequency}
			if got := result.ShouldEscalate(); got != tt.expected {
				t.Errorf("ShouldEscalate() for frequency %.2f = %v, want %v", tt.frequency, got, tt.expected)
			}
		})
	}
}

func TestEstimateFrequencyFromCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		minFreq  float64
		maxFreq  float64
		riskWant RiskLevel
	}{
		{"Zero count", 0, 0.0, 0.0, RiskLevelLow},
		{"Low count", 1, 0.1, 0.3, RiskLevelLow},
		{"Medium count", 3, 0.3, 0.7, RiskLevelMedium},
		{"High count", 6, 0.7, 1.0, RiskLevelHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freq := estimateFrequencyFromCount(tt.count)

			// Check frequency is in expected range
			if freq < tt.minFreq || freq > tt.maxFreq {
				t.Errorf("estimateFrequencyFromCount(%d) = %.2f, want range [%.2f, %.2f]",
					tt.count, freq, tt.minFreq, tt.maxFreq)
			}

			// Check that resulting risk level matches expected
			risk := classifyCoChangeRisk(freq)
			if risk != tt.riskWant {
				t.Errorf("estimateFrequencyFromCount(%d) -> risk %v, want %v",
					tt.count, risk, tt.riskWant)
			}
		})
	}
}

func TestCoChangeResult_FormatEvidence(t *testing.T) {
	tests := []struct {
		name      string
		frequency float64
		riskLevel RiskLevel
		wantEmpty bool
	}{
		{"Zero frequency", 0.0, RiskLevelLow, false},
		{"Low frequency", 0.2, RiskLevelLow, false},
		{"High frequency", 0.8, RiskLevelHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CoChangeResult{
				MaxFrequency: tt.frequency,
				RiskLevel:    tt.riskLevel,
			}
			evidence := result.FormatEvidence()

			isEmpty := len(evidence) == 0
			if isEmpty != tt.wantEmpty {
				t.Errorf("FormatEvidence() empty = %v, want %v", isEmpty, tt.wantEmpty)
			}
		})
	}
}
