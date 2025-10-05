package fixtures

import "testing"

func TestLowRiskFunction(t *testing.T) {
	if result := LowRiskFunction(5); result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}
}

func TestLowRiskFunctionZero(t *testing.T) {
	if result := LowRiskFunction(0); result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestLowRiskFunctionNegative(t *testing.T) {
	if result := LowRiskFunction(-3); result != -6 {
		t.Errorf("Expected -6, got %d", result)
	}
}
