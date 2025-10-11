package prompts

import (
	"fmt"
	"testing"
)

// TestExamples demonstrates confidence prompts with realistic scenarios
func TestExamples(t *testing.T) {
	// Example 1: Early stop scenario (high confidence after hop 1)
	t.Run("Example 1: Early Stop - Documentation Change", func(t *testing.T) {
		evidence := []string{
			"File: README.md (documentation file)",
			"All changes are comment additions",
			"Zero runtime impact detected",
		}

		prompt := ConfidencePromptWithModificationType(
			evidence,
			"VERY_LOW",
			1,
			"DOCUMENTATION",
			"Documentation file (zero runtime impact)",
		)

		fmt.Println("\n=== EXAMPLE 1: EARLY STOP (Documentation) ===")
		fmt.Printf("Hop: 1\n")
		fmt.Printf("Risk Level: VERY_LOW\n")
		fmt.Printf("Expected Confidence: 0.95+\n")
		fmt.Printf("Expected Action: FINALIZE\n\n")
		fmt.Printf("Prompt (truncated):\n%s\n", truncateForDisplay(prompt, 500))

		// Simulate expected LLM response
		expectedResponse := `{
  "confidence": 0.98,
  "reasoning": "Documentation-only change with zero runtime impact. No code modifications detected. Confidence is very high (≥0.95) as specified for DOCUMENTATION type.",
  "next_action": "FINALIZE"
}`
		fmt.Printf("\nExpected LLM Response:\n%s\n", expectedResponse)

		parsed, err := ParseConfidenceAssessment(expectedResponse)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		fmt.Printf("\nParsed Assessment:\n")
		fmt.Printf("  Confidence: %.2f\n", parsed.Confidence)
		fmt.Printf("  Action: %s\n", parsed.NextAction)
		fmt.Printf("  Result: ✅ Stop investigation at hop 1 (high confidence)\n")
	})

	// Example 2: Need more evidence (moderate confidence)
	t.Run("Example 2: Gather More Evidence - Security Change", func(t *testing.T) {
		evidence := []string{
			"File: internal/auth/login.go (security-sensitive)",
			"Security keywords detected: authenticate, token, session",
			"Coupling: 12 dependencies (HIGH)",
			"Test coverage: 0.45 (MODERATE)",
		}

		prompt := ConfidencePromptWithModificationType(
			evidence,
			"HIGH",
			1,
			"SECURITY",
			"Security keywords: auth, login, token",
		)

		fmt.Println("\n=== EXAMPLE 2: GATHER MORE EVIDENCE (Security) ===")
		fmt.Printf("Hop: 1\n")
		fmt.Printf("Risk Level: HIGH\n")
		fmt.Printf("Expected Confidence: 0.65\n")
		fmt.Printf("Expected Action: GATHER_MORE_EVIDENCE\n\n")
		fmt.Printf("Prompt (truncated):\n%s\n", truncateForDisplay(prompt, 500))

		expectedResponse := `{
  "confidence": 0.65,
  "reasoning": "Security-sensitive authentication change with high coupling. Need incident history and ownership data to confirm risk level. Test coverage is moderate but not comprehensive for security-critical code.",
  "next_action": "GATHER_MORE_EVIDENCE"
}`
		fmt.Printf("\nExpected LLM Response:\n%s\n", expectedResponse)

		parsed, err := ParseConfidenceAssessment(expectedResponse)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		fmt.Printf("\nParsed Assessment:\n")
		fmt.Printf("  Confidence: %.2f\n", parsed.Confidence)
		fmt.Printf("  Action: %s\n", parsed.NextAction)
		fmt.Printf("  Result: ⏩ Continue to hop 2 (gather incident/ownership data)\n")
	})

	// Example 3: High confidence after hop 2 (ready to finalize)
	t.Run("Example 3: Finalize After Hop 2 - Critical Security", func(t *testing.T) {
		evidence := []string{
			"File: internal/auth/login.go (security-sensitive)",
			"Security keywords detected: authenticate, token, session",
			"Coupling: 12 dependencies (HIGH)",
			"Test coverage: 0.45 (MODERATE)",
			"[HOP 2] Past incident: INC-123 'Auth timeout' (21 days ago, CRITICAL)",
			"[HOP 2] Ownership transition: 14 days ago (new owner: alice@)",
		}

		prompt := ConfidencePromptWithModificationType(
			evidence,
			"CRITICAL",
			2,
			"SECURITY",
			"Security keywords: auth, login, token",
		)

		fmt.Println("\n=== EXAMPLE 3: FINALIZE AFTER HOP 2 (Critical Security) ===")
		fmt.Printf("Hop: 2\n")
		fmt.Printf("Risk Level: CRITICAL\n")
		fmt.Printf("Expected Confidence: 0.92\n")
		fmt.Printf("Expected Action: FINALIZE\n\n")
		fmt.Printf("Prompt (truncated):\n%s\n", truncateForDisplay(prompt, 500))

		expectedResponse := `{
  "confidence": 0.92,
  "reasoning": "Security-sensitive authentication change + high coupling (12 deps) + recent critical incident (INC-123, 21 days ago) + ownership transition (14 days) = clear CRITICAL risk. All key risk dimensions covered. Additional evidence unlikely to change assessment.",
  "next_action": "FINALIZE"
}`
		fmt.Printf("\nExpected LLM Response:\n%s\n", expectedResponse)

		parsed, err := ParseConfidenceAssessment(expectedResponse)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		fmt.Printf("\nParsed Assessment:\n")
		fmt.Printf("  Confidence: %.2f\n", parsed.Confidence)
		fmt.Printf("  Action: %s\n", parsed.NextAction)
		fmt.Printf("  Result: ✅ Stop investigation at hop 2 (high confidence, all evidence gathered)\n")
	})

	// Example 4: Expand graph (need broader context)
	t.Run("Example 4: Expand Graph - Structural Refactoring", func(t *testing.T) {
		evidence := []string{
			"File: internal/services/payment.go (structural change)",
			"Import changes: 5 new dependencies added",
			"Coupling: 8 dependencies (MODERATE)",
			"Co-change frequency: 0.45 (MODERATE)",
		}

		prompt := ConfidencePromptWithModificationType(
			evidence,
			"MODERATE",
			2,
			"STRUCTURAL",
			"Structural/architectural change",
		)

		fmt.Println("\n=== EXAMPLE 4: EXPAND GRAPH (Structural Refactoring) ===")
		fmt.Printf("Hop: 2\n")
		fmt.Printf("Risk Level: MODERATE\n")
		fmt.Printf("Expected Confidence: 0.55\n")
		fmt.Printf("Expected Action: EXPAND_GRAPH\n\n")
		fmt.Printf("Prompt (truncated):\n%s\n", truncateForDisplay(prompt, 500))

		expectedResponse := `{
  "confidence": 0.55,
  "reasoning": "Structural refactoring with new dependencies. Need to explore 2-hop neighbors to understand cascading impact of new imports. Current coupling is moderate, but don't know full ripple effect yet.",
  "next_action": "EXPAND_GRAPH"
}`
		fmt.Printf("\nExpected LLM Response:\n%s\n", expectedResponse)

		parsed, err := ParseConfidenceAssessment(expectedResponse)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		fmt.Printf("\nParsed Assessment:\n")
		fmt.Printf("  Confidence: %.2f\n", parsed.Confidence)
		fmt.Printf("  Action: %s\n", parsed.NextAction)
		fmt.Printf("  Result: ⏩ Continue to hop 3 (load 2-hop neighbors for broader context)\n")
	})

	// Example 5: Max hops reached (still uncertain)
	t.Run("Example 5: Max Hops - Complex Multi-Type Change", func(t *testing.T) {
		evidence := []string{
			"File: internal/api/auth/middleware.go (complex change)",
			"Types: Security + Interface + Structural + Behavioral",
			"Coupling: 15 dependencies (HIGH)",
			"Test coverage: 0.35 (MODERATE)",
			"[HOP 2] Past incidents: 2 (INC-87, INC-123)",
			"[HOP 3] 2-hop impact: 45 files in dependency graph",
			"[HOP 4] Conflicting signals: high coupling but good test history",
			"[HOP 5] Ownership: Primary owner (bob@) for 2 years (stable)",
		}

		prompt := ConfidencePromptWithModificationType(
			evidence,
			"HIGH",
			5, // Max hops reached
			"SECURITY",
			"Security + Interface + Structural + Behavioral (multi-type)",
		)

		fmt.Println("\n=== EXAMPLE 5: MAX HOPS REACHED (Complex Multi-Type) ===")
		fmt.Printf("Hop: 5 (MAX)\n")
		fmt.Printf("Risk Level: HIGH\n")
		fmt.Printf("Expected Confidence: 0.78 (below threshold but forced to stop)\n")
		fmt.Printf("Expected Action: FINALIZE (no choice, budget exhausted)\n\n")
		fmt.Printf("Prompt (truncated):\n%s\n", truncateForDisplay(prompt, 500))

		expectedResponse := `{
  "confidence": 0.78,
  "reasoning": "Complex multi-type change with conflicting signals. High coupling and incidents suggest HIGH risk, but stable ownership and test history are positive. Confidence below 0.85 ideal threshold, but sufficient evidence to assess as HIGH risk with moderate-high confidence.",
  "next_action": "FINALIZE"
}`
		fmt.Printf("\nExpected LLM Response:\n%s\n", expectedResponse)

		parsed, err := ParseConfidenceAssessment(expectedResponse)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		fmt.Printf("\nParsed Assessment:\n")
		fmt.Printf("  Confidence: %.2f\n", parsed.Confidence)
		fmt.Printf("  Action: %s\n", parsed.NextAction)
		fmt.Printf("  Result: ⚠️  Finalize at max hops (confidence below ideal but budget exhausted)\n")
	})
}

func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated for display)"
}
