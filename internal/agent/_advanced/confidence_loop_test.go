package agent

import (
	"context"
	"strings"
	"testing"
)

// TestConfidenceDrivenLoop tests the confidence-driven investigation loop
func TestConfidenceDrivenLoop(t *testing.T) {
	t.Run("Early Stop - High Confidence After Hop 1", func(t *testing.T) {
		// Mock LLM that returns high confidence immediately
		llm := &mockLLMHighConfidence{responses: []string{
			"This is a documentation-only change with zero runtime impact.",
			`{"confidence": 0.95, "reasoning": "Documentation change, very clear", "next_action": "FINALIZE"}`,
		}}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath: "README.md",
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

		// Should stop after hop 1 due to high confidence
		if len(hops) != 1 {
			t.Errorf("Expected 1 hop (early stop), got %d", len(hops))
		}

		if hops[0].Confidence < 0.85 {
			t.Errorf("Expected confidence >= 0.85, got %.2f", hops[0].Confidence)
		}

		// Check confidence history
		history := navigator.GetConfidenceHistory()
		if len(history) == 0 {
			t.Error("Expected confidence history to be recorded")
		}

		// Check stopping reason
		reason := navigator.GetStoppingReason(hops)
		if !strings.Contains(reason, "confidence") {
			t.Errorf("Expected stopping reason to mention confidence, got: %s", reason)
		}
	})

	t.Run("Continue Investigation - Low Confidence", func(t *testing.T) {
		// Mock LLM that starts with low confidence, gradually increases
		llm := &mockLLMGradualConfidence{
			responses: []string{
				"Found some coupling, need more data.",
				`{"confidence": 0.4, "reasoning": "Need incident history", "next_action": "GATHER_MORE_EVIDENCE"}`,
				"Found 2 incidents in past month.",
				`{"confidence": 0.65, "reasoning": "Need ownership data", "next_action": "GATHER_MORE_EVIDENCE"}`,
				"Ownership transitioned 14 days ago.",
				`{"confidence": 0.88, "reasoning": "All evidence gathered, clear HIGH risk", "next_action": "FINALIZE"}`,
			},
		}

		navigator := NewHopNavigator(llm, nil, 5)
		req := InvestigationRequest{
			FilePath: "internal/auth/login.go",
			Baseline: BaselineMetrics{
				CouplingScore:     0.6,
				CoChangeFrequency: 0.5,
				IncidentCount:     2,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Should continue for 3 hops before reaching high confidence
		if len(hops) != 3 {
			t.Errorf("Expected 3 hops, got %d", len(hops))
		}

		// Check confidence progression
		if hops[0].Confidence >= hops[1].Confidence {
			t.Error("Expected confidence to increase from hop 1 to hop 2")
		}
		if hops[1].Confidence >= hops[2].Confidence {
			t.Error("Expected confidence to increase from hop 2 to hop 3")
		}

		// Final confidence should be high
		if hops[2].Confidence < 0.85 {
			t.Errorf("Expected final confidence >= 0.85, got %.2f", hops[2].Confidence)
		}

		// Check confidence history
		history := navigator.GetConfidenceHistory()
		if len(history) != 3 {
			t.Errorf("Expected 3 confidence points, got %d", len(history))
		}
	})

	t.Run("Max Hops Reached", func(t *testing.T) {
		// Mock LLM that never reaches high confidence
		llm := &mockLLMLowConfidence{
			response: "Unclear, need more data.",
			confidence: `{"confidence": 0.5, "reasoning": "Still uncertain", "next_action": "GATHER_MORE_EVIDENCE"}`,
		}

		navigator := NewHopNavigator(llm, nil, 5)
		navigator.SetConfidenceThreshold(0.85)

		req := InvestigationRequest{
			FilePath: "internal/complex.go",
			Baseline: BaselineMetrics{
				CouplingScore:     0.5,
				CoChangeFrequency: 0.5,
				IncidentCount:     1,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Should hit max hops (5)
		if len(hops) != 5 {
			t.Errorf("Expected 5 hops (max), got %d", len(hops))
		}

		// Check stopping reason
		reason := navigator.GetStoppingReason(hops)
		if !strings.Contains(reason, "Max hops") {
			t.Errorf("Expected stopping reason to mention max hops, got: %s", reason)
		}
	})

	t.Run("Custom Confidence Threshold", func(t *testing.T) {
		// Mock LLM with moderate confidence
		llm := &mockLLMHighConfidence{responses: []string{
			"Some risk factors present.",
			`{"confidence": 0.75, "reasoning": "Clear enough", "next_action": "FINALIZE"}`,
		}}

		navigator := NewHopNavigator(llm, nil, 5)
		navigator.SetConfidenceThreshold(0.7) // Lower threshold

		req := InvestigationRequest{
			FilePath: "test.go",
			Baseline: BaselineMetrics{
				CouplingScore:     0.3,
				CoChangeFrequency: 0.3,
				IncidentCount:     0,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		// Should stop at hop 1 with confidence 0.75 (threshold 0.7)
		if len(hops) != 1 {
			t.Errorf("Expected 1 hop (custom threshold), got %d", len(hops))
		}
	})
}

// Mock LLM implementations

type mockLLMHighConfidence struct {
	responses []string
	callCount int
}

func (m *mockLLMHighConfidence) Query(ctx context.Context, prompt string) (string, int, error) {
	if m.callCount >= len(m.responses) {
		return "Generic response", 50, nil
	}
	response := m.responses[m.callCount]
	m.callCount++
	return response, 50, nil
}

func (m *mockLLMHighConfidence) SetModel(model string) {}

type mockLLMGradualConfidence struct {
	responses []string
	callCount int
}

func (m *mockLLMGradualConfidence) Query(ctx context.Context, prompt string) (string, int, error) {
	if m.callCount >= len(m.responses) {
		return "Generic response", 50, nil
	}
	response := m.responses[m.callCount]
	m.callCount++
	return response, 50, nil
}

func (m *mockLLMGradualConfidence) SetModel(model string) {}

type mockLLMLowConfidence struct {
	response   string
	confidence string
	callCount  int
}

func (m *mockLLMLowConfidence) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++
	// Alternate between investigation and confidence responses
	if m.callCount%2 == 1 {
		return m.response, 50, nil
	}
	return m.confidence, 50, nil
}

func (m *mockLLMLowConfidence) SetModel(model string) {}

// TestConfidenceHistoryTracking tests that confidence is properly tracked
func TestConfidenceHistoryTracking(t *testing.T) {
	llm := &mockLLMGradualConfidence{
		responses: []string{
			"Evidence A",
			`{"confidence": 0.5, "reasoning": "Initial assessment", "next_action": "GATHER_MORE_EVIDENCE"}`,
			"Evidence B",
			`{"confidence": 0.75, "reasoning": "More data collected", "next_action": "GATHER_MORE_EVIDENCE"}`,
			"Evidence C",
			`{"confidence": 0.92, "reasoning": "Comprehensive evidence", "next_action": "FINALIZE"}`,
		},
	}

	navigator := NewHopNavigator(llm, nil, 5)
	req := InvestigationRequest{
		FilePath: "test.go",
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

	history := navigator.GetConfidenceHistory()

	// Should have 3 confidence points
	if len(history) != 3 {
		t.Fatalf("Expected 3 confidence points, got %d", len(history))
	}

	// Verify progression
	for i := 0; i < len(history)-1; i++ {
		if history[i].Confidence >= history[i+1].Confidence {
			t.Errorf("Confidence should increase: hop %d (%.2f) >= hop %d (%.2f)",
				i+1, history[i].Confidence, i+2, history[i+1].Confidence)
		}
	}

	// Verify hop numbers
	for i, point := range history {
		if point.HopNumber != i+1 {
			t.Errorf("Expected hop number %d, got %d", i+1, point.HopNumber)
		}
	}

	// Verify reasoning is captured
	for _, point := range history {
		if point.Reasoning == "" {
			t.Error("Expected reasoning to be captured")
		}
	}
}

// TestBreakthroughIntegration tests that breakthroughs are detected during investigation
func TestBreakthroughIntegration(t *testing.T) {
	// This will be tested once we have the full integration
	// For now, we just verify the structure is in place
	llm := &mockLLMHighConfidence{responses: []string{
		"Low risk initially",
		`{"confidence": 0.5, "reasoning": "Initial", "next_action": "GATHER_MORE_EVIDENCE"}`,
	}}

	navigator := NewHopNavigator(llm, nil, 5)
	req := InvestigationRequest{
		FilePath: "test.go",
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

	// Breakthroughs should be accessible (even if empty)
	breakthroughs := navigator.GetBreakthroughs()
	if breakthroughs == nil {
		t.Error("Expected breakthroughs to be non-nil (even if empty)")
	}
}

// TestConfidenceAssessmentFailureFallback tests fallback when confidence assessment fails
func TestConfidenceAssessmentFailureFallback(t *testing.T) {
	// Mock LLM that returns invalid confidence JSON
	llm := &mockLLMInvalidConfidence{}

	navigator := NewHopNavigator(llm, nil, 5)
	req := InvestigationRequest{
		FilePath: "test.go",
		Baseline: BaselineMetrics{
			CouplingScore:     0.8,
			CoChangeFrequency: 0.7,
			IncidentCount:     3,
		},
	}

	hops, err := navigator.Navigate(context.Background(), req)
	if err != nil {
		t.Fatalf("Navigate() should not fail even with invalid confidence: %v", err)
	}

	// Should still complete investigation using fallback heuristics
	if len(hops) == 0 {
		t.Error("Expected at least 1 hop even with confidence assessment failure")
	}
}

type mockLLMInvalidConfidence struct {
	callCount int
}

func (m *mockLLMInvalidConfidence) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++
	// Always return invalid JSON for confidence prompts
	if strings.Contains(prompt, "confidence") || m.callCount%2 == 0 {
		return "INVALID JSON", 50, nil
	}
	// Return valid investigation response
	if m.callCount > 6 {
		// Include keywords that trigger early stop heuristic
		return "This is critical risk with production outage potential", 50, nil
	}
	return "Investigating...", 50, nil
}

func (m *mockLLMInvalidConfidence) SetModel(model string) {}

// TestConfidencePromptIntegration verifies confidence prompts are used
func TestConfidencePromptIntegration(t *testing.T) {
	llm := &mockLLMCapturePrompts{}

	navigator := NewHopNavigator(llm, nil, 5)
	req := InvestigationRequest{
		FilePath: "test.go",
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

	// Verify confidence prompts were sent
	if !llm.receivedConfidencePrompt {
		t.Error("Expected confidence assessment prompt to be sent")
	}

	// Verify evidence chain is included in prompts
	if !llm.receivedEvidenceChain {
		t.Error("Expected evidence chain to be included in confidence prompts")
	}
}

type mockLLMCapturePrompts struct {
	receivedConfidencePrompt bool
	receivedEvidenceChain    bool
	callCount                int
}

func (m *mockLLMCapturePrompts) Query(ctx context.Context, prompt string) (string, int, error) {
	m.callCount++

	// Check if this is a confidence prompt
	if strings.Contains(prompt, "CONFIDENCE ASSESSMENT") || strings.Contains(prompt, "How confident are you") {
		m.receivedConfidencePrompt = true

		// Check if evidence chain is included
		if strings.Contains(prompt, "EVIDENCE GATHERED") || strings.Contains(prompt, "Coupling:") {
			m.receivedEvidenceChain = true
		}

		// Return valid confidence response
		return `{"confidence": 0.9, "reasoning": "Clear", "next_action": "FINALIZE"}`, 50, nil
	}

	// Return investigation response
	return "Investigation findings", 50, nil
}

func (m *mockLLMCapturePrompts) SetModel(model string) {}
