package agent

import (
	"context"
	"strings"
	"testing"
)

// TestTypeAwareConfidencePrompts tests that modification types enhance confidence assessment
func TestTypeAwareConfidencePrompts(t *testing.T) {
	t.Run("Security Type - Type-Specific Guidance", func(t *testing.T) {
		llm := &mockLLMCaptureTypeGuidance{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "internal/auth/login.go",
			ModificationType:   "SECURITY",
			ModificationReason: "Security keywords: auth, login, token",
			Baseline: BaselineMetrics{
				CouplingScore:     0.6,
				CoChangeFrequency: 0.5,
				IncidentCount:     2,
			},
		}

		_, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Verify type-specific guidance was included in prompt
		if !llm.receivedTypeSpecificGuidance {
			t.Error("Expected type-specific guidance for SECURITY type")
		}

		// Verify security-specific questions were asked
		expectedGuidance := []string{
			"authentication",
			"authorization",
			"sensitive data",
			"security edge cases",
		}
		for _, keyword := range expectedGuidance {
			if !strings.Contains(llm.capturedPrompt, keyword) {
				t.Errorf("Expected security guidance to contain %q", keyword)
			}
		}
	})

	t.Run("Documentation Type - High Confidence Expected", func(t *testing.T) {
		llm := &mockLLMDocumentationType{
			response:   "Documentation-only change, zero runtime impact",
			confidence: `{"confidence": 0.98, "reasoning": "Documentation with zero runtime impact per type guidance", "next_action": "FINALIZE"}`,
		}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "README.md",
			ModificationType:   "DOCUMENTATION",
			ModificationReason: "Documentation file (zero runtime impact)",
			Baseline: BaselineMetrics{
				CouplingScore:     0.1,
				CoChangeFrequency: 0.1,
				IncidentCount:     0,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Documentation type should trigger early stop with high confidence
		if len(hops) != 1 {
			t.Errorf("Expected 1 hop for documentation (early stop), got %d", len(hops))
		}

		if hops[0].Confidence < 0.95 {
			t.Errorf("Expected very high confidence (≥0.95) for documentation, got %.2f", hops[0].Confidence)
		}
	})

	t.Run("Interface Type - API-Specific Guidance", func(t *testing.T) {
		llm := &mockLLMCaptureTypeGuidance{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "internal/api/routes.go",
			ModificationType:   "INTERFACE",
			ModificationReason: "Interface/API modification",
			Baseline: BaselineMetrics{
				CouplingScore:     0.7,
				CoChangeFrequency: 0.6,
				IncidentCount:     1,
			},
		}

		_, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Verify interface-specific guidance
		expectedGuidance := []string{
			"API contracts",
			"backward compatibility",
			"versioning",
		}
		for _, keyword := range expectedGuidance {
			if !strings.Contains(llm.capturedPrompt, keyword) {
				t.Errorf("Expected interface guidance to contain %q", keyword)
			}
		}
	})

	t.Run("Configuration Type - Environment-Specific Guidance", func(t *testing.T) {
		llm := &mockLLMCaptureTypeGuidance{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           ".env.production",
			ModificationType:   "CONFIGURATION",
			ModificationReason: "Production environment configuration",
			Baseline: BaselineMetrics{
				CouplingScore:     0.3,
				CoChangeFrequency: 0.2,
				IncidentCount:     0,
			},
		}

		_, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Verify configuration-specific guidance
		expectedGuidance := []string{
			"production environment",
			"rollback plan",
			"connection strings",
		}
		for _, keyword := range expectedGuidance {
			if !strings.Contains(llm.capturedPrompt, keyword) {
				t.Errorf("Expected configuration guidance to contain %q", keyword)
			}
		}
	})

	t.Run("Fallback to Standard Prompt When Type Not Provided", func(t *testing.T) {
		llm := &mockLLMCaptureTypeGuidance{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath: "test.go",
			// No ModificationType provided
			Baseline: BaselineMetrics{
				CouplingScore:     0.5,
				CoChangeFrequency: 0.4,
				IncidentCount:     1,
			},
		}

		_, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Should use standard prompt without type-specific guidance
		if strings.Contains(llm.capturedPrompt, "TYPE-SPECIFIC CONSIDERATIONS") {
			t.Error("Should not include type-specific guidance when type not provided")
		}

		// Should still include standard confidence assessment
		if !strings.Contains(llm.capturedPrompt, "CONFIDENCE ASSESSMENT") {
			t.Error("Expected standard confidence assessment prompt")
		}
	})

	t.Run("Multi-Type Modification - Primary Type Guidance", func(t *testing.T) {
		llm := &mockLLMCaptureTypeGuidance{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "internal/api/auth/middleware.go",
			ModificationType:   "SECURITY", // Primary type
			ModificationReason: "Security + Interface + Structural + Behavioral (multi-type)",
			Baseline: BaselineMetrics{
				CouplingScore:     0.8,
				CoChangeFrequency: 0.7,
				IncidentCount:     2,
			},
		}

		_, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Should use SECURITY guidance (primary type)
		if !strings.Contains(llm.capturedPrompt, "authentication") {
			t.Error("Expected security guidance for multi-type with SECURITY primary")
		}

		// Modification reason should mention multi-type nature
		if !strings.Contains(llm.capturedPrompt, "multi-type") {
			t.Error("Expected modification reason to mention multi-type")
		}
	})
}

// TestModificationTypeRiskEscalation tests that types influence confidence thresholds
func TestModificationTypeRiskEscalation(t *testing.T) {
	t.Run("Security Type - Lower Confidence Threshold Acceptable", func(t *testing.T) {
		// Security changes should be cautious - even moderate confidence might warrant escalation
		llm := &mockLLMSecurityType{}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "internal/auth/login.go",
			ModificationType:   "SECURITY",
			ModificationReason: "Security-sensitive authentication logic",
			Baseline: BaselineMetrics{
				CouplingScore:     0.6,
				CoChangeFrequency: 0.5,
				IncidentCount:     1,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Security types should gather comprehensive evidence
		if len(hops) < 2 {
			t.Errorf("Expected security investigation to be thorough (≥2 hops), got %d", len(hops))
		}

		// Final confidence should be high for security
		if hops[len(hops)-1].Confidence < 0.85 {
			t.Errorf("Expected high final confidence for security (≥0.85), got %.2f",
				hops[len(hops)-1].Confidence)
		}
	})

	t.Run("Documentation Type - Quick Finalization", func(t *testing.T) {
		// Documentation changes should finalize quickly with high confidence
		llm := &mockLLMDocumentationType{
			response:   "Documentation update",
			confidence: `{"confidence": 0.98, "reasoning": "Zero runtime impact", "next_action": "FINALIZE"}`,
		}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath:           "docs/API.md",
			ModificationType:   "DOCUMENTATION",
			ModificationReason: "Documentation file",
			Baseline: BaselineMetrics{
				CouplingScore:     0.0,
				CoChangeFrequency: 0.0,
				IncidentCount:     0,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Documentation should stop at hop 1
		if len(hops) != 1 {
			t.Errorf("Expected documentation to finalize at hop 1, got %d hops", len(hops))
		}

		if hops[0].NextAction != "FINALIZE" {
			t.Errorf("Expected FINALIZE action for documentation, got %s", hops[0].NextAction)
		}
	})
}

// TestTypeAwarePromptContent verifies prompt content matches type
func TestTypeAwarePromptContent(t *testing.T) {
	tests := []struct {
		name               string
		modificationType   string
		expectedInPrompt   []string
		unexpectedInPrompt []string
	}{
		{
			name:             "Behavioral type",
			modificationType: "BEHAVIORAL",
			expectedInPrompt: []string{
				"test coverage",
				"edge cases",
				"cyclomatic complexity",
			},
			unexpectedInPrompt: []string{
				"API contracts", // interface-specific
			},
		},
		{
			name:             "Structural type",
			modificationType: "STRUCTURAL",
			expectedInPrompt: []string{
				"files are affected",
				"circular dependency",
				"import paths",
			},
			unexpectedInPrompt: []string{
				"zero runtime impact", // documentation-specific
			},
		},
		{
			name:             "Performance type",
			modificationType: "PERFORMANCE",
			expectedInPrompt: []string{
				"load/performance tests",
				"bottlenecks",
				"caching/concurrency",
			},
			unexpectedInPrompt: []string{
				"backward compatibility", // interface-specific
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLMCaptureTypeGuidance{}

			navigator := NewHopNavigator(llm, nil, 5)
			req := InvestigationRequest{
				FilePath:           "test.go",
				ModificationType:   tt.modificationType,
				ModificationReason: tt.modificationType + " change",
				Baseline: BaselineMetrics{
					CouplingScore:     0.5,
					CoChangeFrequency: 0.4,
					IncidentCount:     1,
				},
			}

			_, err := navigator.Navigate(context.Background(), req)
			if err != nil {
				t.Fatalf("Navigate() error = %v", err)
			}

			// Verify expected keywords are present
			for _, keyword := range tt.expectedInPrompt {
				if !strings.Contains(llm.capturedPrompt, keyword) {
					t.Errorf("Expected %s type prompt to contain %q", tt.modificationType, keyword)
				}
			}

			// Verify unexpected keywords are absent
			for _, keyword := range tt.unexpectedInPrompt {
				if strings.Contains(llm.capturedPrompt, keyword) {
					t.Errorf("Did not expect %s type prompt to contain %q", tt.modificationType, keyword)
				}
			}
		})
	}
}

// Mock LLM implementations

type mockLLMCaptureTypeGuidance struct {
	capturedPrompt             string
	receivedTypeSpecificGuidance bool
	callCount                  int
}

func (m *mockLLMCaptureTypeGuidance) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++
	m.capturedPrompt = prompt

	// Check if this is a confidence prompt with type-specific guidance
	if strings.Contains(prompt, "TYPE-SPECIFIC CONSIDERATIONS") {
		m.receivedTypeSpecificGuidance = true
	}

	// Alternate between investigation and confidence responses
	if m.callCount%2 == 0 {
		// Return high confidence to stop
		return `{"confidence": 0.9, "reasoning": "Clear assessment", "next_action": "FINALIZE"}`, 50, nil
	}

	// Return investigation response
	return "Investigation findings", 50, nil
}

func (m *mockLLMCaptureTypeGuidance) SetModel(model string) {}

type mockLLMDocumentationType struct {
	response   string
	confidence string
	callCount  int
}

func (m *mockLLMDocumentationType) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++
	// Alternate between investigation and confidence
	if m.callCount%2 == 0 {
		return m.confidence, 50, nil
	}
	return m.response, 50, nil
}

func (m *mockLLMDocumentationType) SetModel(model string) {}

type mockLLMSecurityType struct {
	callCount int
}

func (m *mockLLMSecurityType) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++

	// Alternate between investigation and confidence
	if m.callCount%2 == 0 {
		// Gradual confidence increase for security
		switch m.callCount {
		case 2:
			return `{"confidence": 0.55, "reasoning": "Need more security validation", "next_action": "GATHER_MORE_EVIDENCE"}`, 50, nil
		case 4:
			return `{"confidence": 0.75, "reasoning": "Getting clearer", "next_action": "GATHER_MORE_EVIDENCE"}`, 50, nil
		default:
			return `{"confidence": 0.92, "reasoning": "Security thoroughly validated", "next_action": "FINALIZE"}`, 50, nil
		}
	}

	return "Investigating security aspects", 50, nil
}

func (m *mockLLMSecurityType) SetModel(model string) {}
