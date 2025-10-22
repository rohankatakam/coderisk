package agent

import (
	"context"
	"testing"
	"time"
)

// Mock LLM client for testing
type MockLLMClient struct {
	response string
	tokens   int
	err      error
}

func (m *MockLLMClient) Query(ctx context.Context, prompt string) (string, int, error) {
	if m.err != nil {
		return "", 0, m.err
	}
	return m.response, m.tokens, nil
}

func (m *MockLLMClient) SetModel(model string) {
	// No-op for mock
}

// TestSimpleInvestigator_BasicFlow tests the basic investigation flow
func TestSimpleInvestigator_BasicFlow(t *testing.T) {
	// Mock LLM response
	mockResponse := `{
		"risk_level": "MEDIUM",
		"due_diligence_summary": "File has moderate coupling and no recent incidents",
		"coordination_needed": {
			"should_contact_owner": false,
			"should_contact_others": [],
			"reason": "Low risk change"
		},
		"forgotten_updates": {
			"likely_forgotten_files": [],
			"reason": "No co-change patterns detected"
		},
		"incident_risk": {
			"similar_incident": "",
			"pattern": "",
			"prevention": ""
		},
		"recommendations": [
			"Review code changes",
			"Verify tests pass"
		]
	}`

	mockLLM := &MockLLMClient{
		response: mockResponse,
		tokens:   150,
	}

	inv := NewSimpleInvestigator(mockLLM, nil, nil, nil)

	req := InvestigationRequest{
		FilePath:   "test/file.go",
		ChangeType: "modify",
		Baseline: BaselineMetrics{
			CouplingScore:     0.5,
			CoChangeFrequency: 0.3,
			IncidentCount:     0,
			TestCoverage:      0.8,
		},
		StartedAt: time.Now(),
	}

	ctx := context.Background()
	assessment, err := inv.Investigate(ctx, req)

	if err != nil {
		t.Fatalf("Investigate failed: %v", err)
	}

	if assessment.RiskLevel != RiskMedium {
		t.Errorf("Expected MEDIUM risk, got %s", assessment.RiskLevel)
	}

	if assessment.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %.2f", assessment.Confidence)
	}

	if len(assessment.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(assessment.Recommendations))
	}
}

// TestSimpleInvestigator_LLMFailure tests fallback behavior when LLM fails
func TestSimpleInvestigator_LLMFailure(t *testing.T) {
	mockLLM := &MockLLMClient{
		err: context.DeadlineExceeded,
	}

	inv := NewSimpleInvestigator(mockLLM, nil, nil, nil)

	req := InvestigationRequest{
		FilePath:   "test/file.go",
		ChangeType: "modify",
		Baseline: BaselineMetrics{
			CouplingScore:     0.5,
			CoChangeFrequency: 0.3,
			IncidentCount:     0,
			TestCoverage:      0.8,
		},
		StartedAt: time.Now(),
	}

	ctx := context.Background()
	assessment, err := inv.Investigate(ctx, req)

	// Should not error, should return fallback
	if err != nil {
		t.Fatalf("Expected fallback, got error: %v", err)
	}

	// Fallback should return MEDIUM risk conservatively
	if assessment.RiskLevel != RiskMedium {
		t.Errorf("Expected MEDIUM risk fallback, got %s", assessment.RiskLevel)
	}

	// Fallback should have low confidence
	if assessment.Confidence != 0.3 {
		t.Errorf("Expected low confidence 0.3, got %.2f", assessment.Confidence)
	}

	// Should have basic recommendations
	if len(assessment.Recommendations) == 0 {
		t.Error("Expected fallback to provide recommendations")
	}
}

// TestSimpleInvestigator_InvalidJSON tests handling of invalid LLM response
func TestSimpleInvestigator_InvalidJSON(t *testing.T) {
	mockLLM := &MockLLMClient{
		response: "This is not valid JSON",
		tokens:   50,
	}

	inv := NewSimpleInvestigator(mockLLM, nil, nil, nil)

	req := InvestigationRequest{
		FilePath:   "test/file.go",
		ChangeType: "modify",
		Baseline: BaselineMetrics{
			CouplingScore:     0.5,
			CoChangeFrequency: 0.3,
			IncidentCount:     0,
			TestCoverage:      0.8,
		},
		StartedAt: time.Now(),
	}

	ctx := context.Background()
	assessment, err := inv.Investigate(ctx, req)

	// Should not error, should fallback
	if err != nil {
		t.Fatalf("Expected fallback, got error: %v", err)
	}

	// Should use fallback assessment
	if assessment.Confidence >= 0.5 {
		t.Errorf("Expected low confidence for invalid JSON, got %.2f", assessment.Confidence)
	}
}

// TestExtractJSON tests JSON extraction from markdown code blocks
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in generic code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseRiskLevel tests risk level parsing
func TestParseRiskLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected RiskLevel
	}{
		{"CRITICAL", RiskCritical},
		{"HIGH", RiskHigh},
		{"MEDIUM", RiskMedium},
		{"LOW", RiskLow},
		{"critical", RiskCritical}, // Test case insensitivity
		{"unknown", RiskMedium},    // Test default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseRiskLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseRiskLevel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
