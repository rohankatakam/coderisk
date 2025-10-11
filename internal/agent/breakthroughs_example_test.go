package agent

import (
	"fmt"
	"strings"
	"testing"
)

// TestBreakthroughExample demonstrates breakthrough tracking in a realistic investigation scenario
func TestBreakthroughExample(t *testing.T) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BREAKTHROUGH DETECTION EXAMPLE: Security File Investigation")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Scenario: Investigating changes to an authentication file
	fmt.Println("Scenario: Developer modifies internal/auth/login.go")
	fmt.Println("Initial Assessment: MODERATE risk (coupling detected)")
	fmt.Println()

	// Initialize tracker with baseline risk
	tracker := NewBreakthroughTracker(0.45, RiskMedium)
	fmt.Printf("Initial State:\n")
	fmt.Printf("  Risk Score: 0.45\n")
	fmt.Printf("  Risk Level: MEDIUM\n")
	fmt.Printf("  Significance Threshold: 0.20 (20%% change)\n\n")

	fmt.Println(strings.Repeat("-", 80))

	// Hop 1: Minor evidence (no breakthrough)
	fmt.Println("\nHOP 1: Structural Analysis")
	fmt.Println("Evidence: Coupling increased from 8 to 10 dependencies")
	newScore1 := 0.50
	detected1 := tracker.CheckAndRecordBreakthrough(
		1,
		newScore1,
		RiskMedium,
		"Coupling: 10 dependencies (slight increase)",
		"Two new service dependencies added",
	)
	fmt.Printf("New Risk Score: 0.50 (Δ +0.05)\n")
	fmt.Printf("Breakthrough Detected: %v (change below 0.20 threshold)\n", detected1)
	if !detected1 {
		fmt.Println("✓ Continue investigation - change not significant enough")
	}

	fmt.Println(strings.Repeat("-", 80))

	// Hop 2: Significant evidence (BREAKTHROUGH - escalation)
	fmt.Println("\nHOP 2: Historical Analysis")
	fmt.Println("Evidence: Critical incident found - 'Auth bypass vulnerability' (INC-456, 18 days ago)")
	newScore2 := 0.85
	detected2 := tracker.CheckAndRecordBreakthrough(
		2,
		newScore2,
		RiskCritical,
		"Past incident: INC-456 'Auth bypass' (18 days ago, CRITICAL severity)",
		"Recent production security incident significantly elevates risk",
	)
	fmt.Printf("New Risk Score: 0.85 (Δ +0.35)\n")
	fmt.Printf("Risk Level: MEDIUM → CRITICAL\n")
	fmt.Printf("Breakthrough Detected: %v ⚠️\n", detected2)
	if detected2 {
		fmt.Println("⚠️ ESCALATION: Risk level changed significantly!")
		fmt.Println("   Reason: Historical incident + security-sensitive file")
	}

	fmt.Println(strings.Repeat("-", 80))

	// Hop 3: Additional context (no breakthrough)
	fmt.Println("\nHOP 3: Ownership Analysis")
	fmt.Println("Evidence: Code owner unchanged for 2 years (stable ownership)")
	newScore3 := 0.82
	detected3 := tracker.CheckAndRecordBreakthrough(
		3,
		newScore3,
		RiskCritical,
		"Ownership: Primary owner (alice@) for 2 years (stable)",
		"Stable ownership slightly reduces concern",
	)
	fmt.Printf("New Risk Score: 0.82 (Δ -0.03)\n")
	fmt.Printf("Breakthrough Detected: %v (change below threshold)\n", detected3)
	if !detected3 {
		fmt.Println("✓ No breakthrough - risk remains CRITICAL")
	}

	fmt.Println(strings.Repeat("-", 80))

	// Final summary
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("INVESTIGATION SUMMARY")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	fmt.Printf("Total Hops: 3\n")
	fmt.Printf("Breakthroughs Detected: %d\n", tracker.GetBreakthroughCount())
	fmt.Printf("Final Risk Score: 0.82\n")
	fmt.Printf("Final Risk Level: CRITICAL\n\n")

	fmt.Println("Breakthrough Points:")
	for i, formatted := range tracker.FormatBreakthroughsForUser() {
		fmt.Printf("  %d. %s\n", i+1, formatted)
	}

	mostSig := tracker.GetMostSignificantBreakthrough()
	if mostSig != nil {
		fmt.Printf("\nMost Significant Change:\n")
		fmt.Printf("  Hop %d: %.2f → %.2f (Δ +%.0f%%)\n",
			mostSig.HopNumber,
			mostSig.RiskBefore,
			mostSig.RiskAfter,
			mostSig.RiskChange*100)
		fmt.Printf("  Trigger: %s\n", mostSig.TriggeringEvidence)
	}

	fmt.Println("\n" + tracker.GetInvestigationSummary())

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("LLM-FORMATTED BREAKTHROUGH CONTEXT")
	fmt.Println(strings.Repeat("=", 80) + "\n")
	fmt.Println(tracker.FormatBreakthroughsForLLM())

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")

	// Verify expectations
	if tracker.GetBreakthroughCount() != 1 {
		t.Errorf("Expected 1 breakthrough, got %d", tracker.GetBreakthroughCount())
	}

	breakthroughs := tracker.GetBreakthroughs()
	if len(breakthroughs) > 0 {
		b := breakthroughs[0]
		if b.HopNumber != 2 {
			t.Errorf("Breakthrough should be at hop 2, got %d", b.HopNumber)
		}
		if !b.IsEscalation {
			t.Error("Breakthrough should be an escalation")
		}
		if b.RiskLevelBefore != RiskMedium || b.RiskLevelAfter != RiskCritical {
			t.Errorf("Breakthrough should be MEDIUM→CRITICAL, got %s→%s",
				b.RiskLevelBefore, b.RiskLevelAfter)
		}
	}
}

func TestBreakthroughExample_DeEscalation(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DE-ESCALATION EXAMPLE: Test Coverage Improvement")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	tracker := NewBreakthroughTracker(0.75, RiskHigh)
	fmt.Printf("Initial: Risk 0.75 (HIGH)\n\n")

	// Hop 1: Tests added (BREAKTHROUGH - de-escalation)
	fmt.Println("HOP 1: Comprehensive test suite added")
	detected := tracker.CheckAndRecordBreakthrough(
		1,
		0.35,
		RiskLow,
		"Test coverage increased from 0.20 to 0.85 (comprehensive suite)",
		"New integration tests cover all critical auth paths",
	)
	fmt.Printf("New Risk: 0.35 (Δ -0.40, HIGH → LOW)\n")
	fmt.Printf("Breakthrough Detected: %v ✓\n\n", detected)

	if detected {
		breakthroughs := tracker.GetBreakthroughs()
		fmt.Printf("DE-ESCALATION:\n")
		fmt.Printf("  %s\n\n", tracker.FormatBreakthroughsForUser()[0])
		fmt.Println(tracker.GetInvestigationSummary())

		// Verify
		if breakthroughs[0].IsEscalation {
			t.Error("Should be de-escalation, not escalation")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
}
