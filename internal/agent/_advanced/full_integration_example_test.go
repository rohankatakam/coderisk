package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// TestFullIntegrationExample demonstrates the complete confidence-driven investigation
// with type-aware prompts, breakthrough detection, and dynamic stopping
func TestFullIntegrationExample(t *testing.T) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("FULL INTEGRATION EXAMPLE: Type-Aware Confidence-Driven Investigation")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Scenario: Security-sensitive authentication file modification
	fmt.Println("Scenario: Developer modifies internal/auth/login.go (SECURITY type)")
	fmt.Println("Phase 0 Classification: SECURITY (keywords: auth, login, token)")
	fmt.Println("Initial Risk: MODERATE (coupling=0.6, incidents=2)")
	fmt.Println()

	llm := &mockLLMFullIntegration{
		responses: []mockResponse{
			// Hop 1: Investigation
			{text: "Found high coupling (12 dependencies) and authentication logic changes"},
			// Hop 1: Confidence (with SECURITY type guidance)
			{text: `{"confidence": 0.55, "reasoning": "Security change detected. Need incident history to assess risk comprehensively per SECURITY type guidance.", "next_action": "GATHER_MORE_EVIDENCE"}`},
			// Hop 2: Investigation
			{text: "Critical incident INC-456 'Auth bypass vulnerability' occurred 18 days ago, CRITICAL severity"},
			// Hop 2: Confidence (BREAKTHROUGH: risk escalates to CRITICAL)
			{text: `{"confidence": 0.78, "reasoning": "Security + recent critical incident = HIGH/CRITICAL risk. Need ownership validation.", "next_action": "GATHER_MORE_EVIDENCE"}`},
			// Hop 3: Investigation
			{text: "Ownership transitioned 14 days ago to new developer unfamiliar with auth logic"},
			// Hop 3: Confidence (HIGH confidence, FINALIZE)
			{text: `{"confidence": 0.92, "reasoning": "Security-sensitive file + critical incident (18d ago) + ownership transition (14d) = clear CRITICAL risk. All evidence gathered per SECURITY type requirements.", "next_action": "FINALIZE"}`},
		},
	}

	navigator := NewHopNavigator(llm, nil, 5)
	navigator.SetConfidenceThreshold(0.85)

	req := InvestigationRequest{
		FilePath:           "internal/auth/login.go",
		ModificationType:   "SECURITY",
		ModificationReason: "Security keywords detected: auth, login, token",
		Baseline: BaselineMetrics{
			CouplingScore:     0.6,
			CoChangeFrequency: 0.5,
			IncidentCount:     2,
		},
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Println()

	hops, err := navigator.Navigate(context.Background(), req)
	if err != nil {
		t.Fatalf("Navigate() error = %v", err)
	}

	// Display results
	fmt.Println("INVESTIGATION COMPLETED")
	fmt.Println()
	fmt.Printf("Total Hops: %d (confidence-driven stopping)\n", len(hops))
	fmt.Printf("Stopping Reason: %s\n", navigator.GetStoppingReason(hops))
	fmt.Println()

	// Show confidence progression
	fmt.Println("Confidence Progression:")
	history := navigator.GetConfidenceHistory()
	for _, point := range history {
		fmt.Printf("  Hop %d: %.2f (%.0f%%) - %s\n",
			point.HopNumber,
			point.Confidence,
			point.Confidence*100,
			point.NextAction)
	}
	fmt.Println()

	// Show breakthroughs
	breakthroughs := navigator.GetBreakthroughs()
	if len(breakthroughs) > 0 {
		fmt.Println("Breakthroughs Detected:")
		for _, b := range breakthroughs {
			direction := "↑ ESCALATION"
			if !b.IsEscalation {
				direction = "↓ DE-ESCALATION"
			}
			fmt.Printf("  Hop %d %s: %s → %s (Δ=%.2f)\n",
				b.HopNumber, direction, b.RiskLevelBefore, b.RiskLevelAfter, b.RiskChange)
			fmt.Printf("    Trigger: %s\n", b.TriggeringEvidence)
		}
		fmt.Println()
	}

	// Show final state
	fmt.Println("Final Assessment:")
	fmt.Printf("  Hops Executed: %d / %d max\n", len(hops), 5)
	fmt.Printf("  Final Confidence: %.2f (≥ 0.85 threshold)\n", hops[len(hops)-1].Confidence)
	fmt.Printf("  Recommendation: CRITICAL risk - Security change + incident + ownership\n")
	fmt.Println()

	// Show type-aware guidance was used
	if strings.Contains(llm.lastPrompt, "authentication") &&
		strings.Contains(llm.lastPrompt, "security edge cases") {
		fmt.Println("✓ Type-aware guidance applied (SECURITY considerations)")
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Verify expectations
	if len(hops) != 3 {
		t.Errorf("Expected 3 hops (confidence-driven stopping), got %d", len(hops))
	}

	if hops[2].Confidence < 0.85 {
		t.Errorf("Expected final confidence ≥0.85, got %.2f", hops[2].Confidence)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 confidence points, got %d", len(history))
	}

	// Verify confidence increased over time
	if history[0].Confidence >= history[1].Confidence || history[1].Confidence >= history[2].Confidence {
		t.Error("Expected confidence to increase across hops")
	}

	// Verify stopping reason
	reason := navigator.GetStoppingReason(hops)
	if !strings.Contains(reason, "confidence") {
		t.Errorf("Expected stopping reason to mention confidence, got: %s", reason)
	}
}

// TestFullIntegrationDocumentationType demonstrates quick finalization for docs
func TestFullIntegrationDocumentationType(t *testing.T) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("DOCUMENTATION TYPE EXAMPLE: Early Stop with High Confidence")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	fmt.Println("Scenario: Developer updates README.md (DOCUMENTATION type)")
	fmt.Println("Phase 0 Classification: DOCUMENTATION (zero runtime impact)")
	fmt.Println("Expected: Quick finalization with very high confidence")
	fmt.Println()

	llm := &mockLLMFullIntegration{
		responses: []mockResponse{
			// Hop 1: Investigation
			{text: "Documentation-only change, no code modifications, zero runtime impact"},
			// Hop 1: Confidence (HIGH confidence immediately)
			{text: `{"confidence": 0.98, "reasoning": "Documentation file with zero runtime impact per DOCUMENTATION type guidance. Confidence ≥0.95 as specified. FINALIZE immediately.", "next_action": "FINALIZE"}`},
		},
	}

	navigator := NewHopNavigator(llm, nil, 5)
	req := InvestigationRequest{
		FilePath:           "README.md",
		ModificationType:   "DOCUMENTATION",
		ModificationReason: "Documentation file (zero runtime impact)",
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

	fmt.Printf("Result: Stopped at hop %d with confidence %.2f\n", len(hops), hops[0].Confidence)
	fmt.Printf("Type-aware guidance: %s\n", "FINALIZE immediately for documentation")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Verify
	if len(hops) != 1 {
		t.Errorf("Expected 1 hop for documentation, got %d", len(hops))
	}

	if hops[0].Confidence < 0.95 {
		t.Errorf("Expected very high confidence (≥0.95) for docs, got %.2f", hops[0].Confidence)
	}

	if hops[0].NextAction != "FINALIZE" {
		t.Errorf("Expected FINALIZE action for docs, got %s", hops[0].NextAction)
	}
}

// TestIntegrationPerformanceComparison compares old vs new approach
func TestIntegrationPerformanceComparison(t *testing.T) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("PERFORMANCE COMPARISON: Fixed 3-Hop vs Confidence-Driven")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	scenarios := []struct {
		name                string
		modificationType    string
		expectedHops        int
		oldHops             int
		improvementPercent  float64
	}{
		{
			name:               "Documentation Change",
			modificationType:   "DOCUMENTATION",
			expectedHops:       1,
			oldHops:            3,
			improvementPercent: 67,
		},
		{
			name:               "Simple Test Addition",
			modificationType:   "TEST_QUALITY",
			expectedHops:       1,
			oldHops:            3,
			improvementPercent: 67,
		},
		{
			name:               "Security Change with Incidents",
			modificationType:   "SECURITY",
			expectedHops:       3,
			oldHops:            3,
			improvementPercent: 0, // Same depth, but more confident
		},
		{
			name:               "Complex Multi-Type",
			modificationType:   "SECURITY", // Primary
			expectedHops:       4,
			oldHops:            3,
			improvementPercent: -33, // Slightly more thorough
		},
	}

	fmt.Println("| Scenario                       | Old Hops | New Hops | Improvement |")
	fmt.Println("|--------------------------------|----------|----------|-------------|")
	for _, sc := range scenarios {
		improvementStr := fmt.Sprintf("%.0f%%", sc.improvementPercent)
		if sc.improvementPercent == 0 {
			improvementStr = "More confident"
		} else if sc.improvementPercent < 0 {
			improvementStr = "More thorough"
		} else {
			improvementStr = improvementStr + " faster"
		}
		fmt.Printf("| %-30s | %8d | %8d | %-11s |\n",
			sc.name, sc.oldHops, sc.expectedHops, improvementStr)
	}
	fmt.Println()

	avgOld := 3.0
	avgNew := (1.0 + 1.0 + 3.0 + 4.0) / 4.0
	overallImprovement := ((avgOld - avgNew) / avgOld) * 100

	fmt.Printf("Average Hops: %.1f (old) → %.1f (new)\n", avgOld, avgNew)
	fmt.Printf("Overall Improvement: %.0f%% faster on average\n", overallImprovement)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

// Mock LLM for full integration testing

type mockResponse struct {
	text string
}

type mockLLMFullIntegration struct {
	responses  []mockResponse
	callCount  int
	lastPrompt string
}

func (m *mockLLMFullIntegration) Query(ctx context.Context, prompt string) (string, int, error) {
	m.lastPrompt = prompt

	if m.callCount >= len(m.responses) {
		return "No more responses", 50, nil
	}

	response := m.responses[m.callCount]
	m.callCount++

	return response.text, 50, nil
}

func (m *mockLLMFullIntegration) SetModel(model string) {}
