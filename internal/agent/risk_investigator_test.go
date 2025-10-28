package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// MockGraphClient for testing
type MockGraphClient struct {
	QueryResults map[string][]map[string]any
}

func (m *MockGraphClient) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	// Return mock results based on query type
	if len(m.QueryResults) > 0 {
		// Return first available result
		for _, results := range m.QueryResults {
			return results, nil
		}
	}
	return []map[string]any{}, nil
}

// MockPGClient for testing
type MockPGClient struct{}

func (m *MockPGClient) GetCommitPatch(ctx context.Context, commitSHA string) (string, error) {
	return "mock patch content", nil
}

// MockLLMClientForRiskInvestigator for testing
type MockLLMClientForRiskInvestigator struct {
	Responses []openai.ChatCompletionResponse
	CallCount int
}

func (m *MockLLMClientForRiskInvestigator) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if m.CallCount >= len(m.Responses) {
		// Return finish response if out of pre-configured responses
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role: openai.ChatMessageRoleAssistant,
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_finish",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "finish_investigation",
									Arguments: `{"risk_level": "LOW", "confidence": 0.8, "reasoning": "No significant risks found", "recommendations": ["Continue with commit"]}`,
								},
							},
						},
					},
					FinishReason: openai.FinishReasonToolCalls,
				},
			},
			Usage: openai.Usage{TotalTokens: 100},
		}, nil
	}

	resp := m.Responses[m.CallCount]
	m.CallCount++
	return resp, nil
}

func TestRiskInvestigator_GetToolDefinitions(t *testing.T) {
	mockGraph := &MockGraphClient{}
	mockPG := &MockPGClient{}
	llmClient := &LLMClient{} // Just for type

	investigator := NewRiskInvestigator(llmClient, mockGraph, mockPG)
	tools := investigator.getToolDefinitions()

	// Verify we have all expected tools
	expectedTools := map[string]bool{
		"query_ownership":        false,
		"query_cochange_partners": false,
		"query_incident_history":  false,
		"query_blast_radius":      false,
		"query_recent_commits":    false,
		"finish_investigation":    false,
	}

	for _, tool := range tools {
		if tool.Function != nil {
			expectedTools[tool.Function.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("Missing tool: %s", name)
		}
	}

	// Verify tool count
	if len(tools) != 6 {
		t.Errorf("Expected 6 tools, got %d", len(tools))
	}
}

func TestRiskInvestigator_QueryOwnership(t *testing.T) {
	mockGraph := &MockGraphClient{
		QueryResults: map[string][]map[string]any{
			"ownership": {
				{
					"developer":    "alice@example.com",
					"commit_count": float64(15),
				},
				{
					"developer":    "bob@example.com",
					"commit_count": float64(8),
				},
			},
		},
	}
	mockPG := &MockPGClient{}
	llmClient := &LLMClient{}

	investigator := NewRiskInvestigator(llmClient, mockGraph, mockPG)

	args := map[string]any{
		"file_paths": []any{"src/auth/login.py", "auth/login.py"},
	}

	result, err := investigator.queryOwnership(context.Background(), args)
	if err != nil {
		t.Fatalf("queryOwnership failed: %v", err)
	}

	results, ok := result.([]map[string]any)
	if !ok {
		t.Fatal("Result is not array of maps")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify first result
	if results[0]["developer"] != "alice@example.com" {
		t.Errorf("Expected alice@example.com, got %v", results[0]["developer"])
	}
}

func TestRiskInvestigator_HandleFinishInvestigation(t *testing.T) {
	mockGraph := &MockGraphClient{}
	mockPG := &MockPGClient{}
	llmClient := &LLMClient{}

	investigator := NewRiskInvestigator(llmClient, mockGraph, mockPG)

	argsJSON := `{
		"risk_level": "HIGH",
		"confidence": 0.85,
		"reasoning": "Multiple incidents in last 90 days with similar patterns",
		"recommendations": ["Review with security team", "Add integration tests"]
	}`

	investigation := &Investigation{}
	assessment, err := investigator.handleFinishInvestigation(argsJSON, investigation)
	if err != nil {
		t.Fatalf("handleFinishInvestigation failed: %v", err)
	}

	if assessment.RiskLevel != RiskHigh {
		t.Errorf("Expected HIGH risk, got %v", assessment.RiskLevel)
	}

	if assessment.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", assessment.Confidence)
	}

	if len(assessment.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(assessment.Recommendations))
	}

	if assessment.RiskScore != 0.7 {
		t.Errorf("Expected risk score 0.7 for HIGH, got %f", assessment.RiskScore)
	}
}

func TestRiskInvestigator_ExtractStringArray(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		expected []string
	}{
		{
			name: "valid array",
			args: map[string]any{
				"file_paths": []any{"file1.py", "file2.py"},
			},
			key:      "file_paths",
			expected: []string{"file1.py", "file2.py"},
		},
		{
			name:     "missing key",
			args:     map[string]any{},
			key:      "file_paths",
			expected: nil,
		},
		{
			name: "empty array",
			args: map[string]any{
				"file_paths": []any{},
			},
			key:      "file_paths",
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := extractStringArray(test.args, test.key)

			if len(result) != len(test.expected) {
				t.Errorf("Expected length %d, got %d", len(test.expected), len(result))
				return
			}

			for i, val := range result {
				if val != test.expected[i] {
					t.Errorf("At index %d: expected %s, got %s", i, test.expected[i], val)
				}
			}
		})
	}
}

func TestRiskInvestigator_FormatToolResult(t *testing.T) {
	mockGraph := &MockGraphClient{}
	mockPG := &MockPGClient{}
	llmClient := &LLMClient{}

	investigator := NewRiskInvestigator(llmClient, mockGraph, mockPG)

	// Test successful result
	result := []map[string]any{
		{"file": "test.py", "count": float64(5)},
	}

	formatted := investigator.formatToolResult(result, nil)

	// Should be valid JSON
	var parsed any
	if err := json.Unmarshal([]byte(formatted), &parsed); err != nil {
		t.Errorf("Formatted result is not valid JSON: %v", err)
	}

	// Test error result
	formatted = investigator.formatToolResult(nil, fmt.Errorf("query failed"))
	if formatted != "ERROR: query failed" {
		t.Errorf("Expected error format, got: %s", formatted)
	}
}

func TestRiskInvestigator_EmergencyAssessment(t *testing.T) {
	mockGraph := &MockGraphClient{}
	mockPG := &MockPGClient{}
	llmClient := &LLMClient{}

	investigator := NewRiskInvestigator(llmClient, mockGraph, mockPG)

	investigation := &Investigation{
		TotalTokens: 500,
	}

	assessment := investigator.emergencyAssessment(investigation, "Test failure")

	// Should return MEDIUM risk (conservative)
	if assessment.RiskLevel != RiskMedium {
		t.Errorf("Expected MEDIUM risk, got %v", assessment.RiskLevel)
	}

	// Should have low confidence
	if assessment.Confidence >= 0.5 {
		t.Errorf("Expected low confidence (<0.5), got %f", assessment.Confidence)
	}

	// Should have recommendations
	if len(assessment.Recommendations) == 0 {
		t.Error("Expected recommendations in emergency assessment")
	}

	// Should set stopping reason
	if investigation.StoppingReason != "Test failure" {
		t.Errorf("Expected stopping reason 'Test failure', got %s", investigation.StoppingReason)
	}
}

func TestRiskInvestigator_RiskLevelToScore(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected float64
	}{
		{RiskCritical, 0.9},
		{RiskHigh, 0.7},
		{RiskMedium, 0.5},
		{RiskLow, 0.3},
	}

	for _, test := range tests {
		t.Run(string(test.level), func(t *testing.T) {
			score := riskLevelToScore(test.level)
			if score != test.expected {
				t.Errorf("Expected score %f for %s, got %f", test.expected, test.level, score)
			}
		})
	}
}
