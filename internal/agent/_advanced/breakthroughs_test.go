package agent

import (
	"math"
	"strings"
	"testing"
)

const floatTolerance = 1e-9

func TestNewBreakthroughTracker(t *testing.T) {
	tests := []struct {
		name             string
		initialScore     float64
		initialLevel     RiskLevel
		wantThreshold    float64
		wantBreakthrough int
	}{
		{
			name:             "initialize with low risk",
			initialScore:     0.3,
			initialLevel:     RiskLow,
			wantThreshold:    0.2,
			wantBreakthrough: 0,
		},
		{
			name:             "initialize with high risk",
			initialScore:     0.75,
			initialLevel:     RiskHigh,
			wantThreshold:    0.2,
			wantBreakthrough: 0,
		},
		{
			name:             "initialize with critical risk",
			initialScore:     0.95,
			initialLevel:     RiskCritical,
			wantThreshold:    0.2,
			wantBreakthrough: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewBreakthroughTracker(tt.initialScore, tt.initialLevel)

			if tracker.significanceThreshold != tt.wantThreshold {
				t.Errorf("NewBreakthroughTracker() threshold = %.2f, want %.2f",
					tracker.significanceThreshold, tt.wantThreshold)
			}

			if tracker.previousRiskScore != tt.initialScore {
				t.Errorf("NewBreakthroughTracker() previous score = %.2f, want %.2f",
					tracker.previousRiskScore, tt.initialScore)
			}

			if tracker.previousRiskLevel != tt.initialLevel {
				t.Errorf("NewBreakthroughTracker() previous level = %s, want %s",
					tracker.previousRiskLevel, tt.initialLevel)
			}

			if len(tracker.breakthroughs) != tt.wantBreakthrough {
				t.Errorf("NewBreakthroughTracker() breakthroughs = %d, want %d",
					len(tracker.breakthroughs), tt.wantBreakthrough)
			}
		})
	}
}

func TestCheckAndRecordBreakthrough(t *testing.T) {
	tests := []struct {
		name               string
		initialScore       float64
		initialLevel       RiskLevel
		hopNumber          int
		newScore           float64
		newLevel           RiskLevel
		evidence           string
		reasoning          string
		wantBreakthrough   bool
		wantIsEscalation   bool
		wantRiskChange     float64
	}{
		{
			name:             "significant escalation (LOW to HIGH)",
			initialScore:     0.3,
			initialLevel:     RiskLow,
			hopNumber:        2,
			newScore:         0.7,
			newLevel:         RiskHigh,
			evidence:         "Critical incident found: INC-123",
			reasoning:        "Recent production incident significantly increases risk",
			wantBreakthrough: true,
			wantIsEscalation: true,
			wantRiskChange:   0.4,
		},
		{
			name:             "significant de-escalation (HIGH to LOW)",
			initialScore:     0.75,
			initialLevel:     RiskHigh,
			hopNumber:        3,
			newScore:         0.25,
			newLevel:         RiskLow,
			evidence:         "Comprehensive test suite added",
			reasoning:        "New tests cover all critical paths",
			wantBreakthrough: true,
			wantIsEscalation: false,
			wantRiskChange:   -0.5,
		},
		{
			name:             "marginal change below threshold",
			initialScore:     0.5,
			initialLevel:     RiskMedium,
			hopNumber:        1,
			newScore:         0.6,
			newLevel:         RiskHigh,
			evidence:         "Minor coupling increase",
			reasoning:        "Slightly more dependencies",
			wantBreakthrough: false,
			wantIsEscalation: true,
			wantRiskChange:   0.1,
		},
		{
			name:             "exactly at threshold (0.2)",
			initialScore:     0.4,
			initialLevel:     RiskMedium,
			hopNumber:        2,
			newScore:         0.6,
			newLevel:         RiskHigh,
			evidence:         "Ownership transition detected",
			reasoning:        "New owner unfamiliar with code",
			wantBreakthrough: true,
			wantIsEscalation: true,
			wantRiskChange:   0.2,
		},
		{
			name:             "no change",
			initialScore:     0.5,
			initialLevel:     RiskMedium,
			hopNumber:        1,
			newScore:         0.5,
			newLevel:         RiskMedium,
			evidence:         "No new evidence",
			reasoning:        "Risk remains the same",
			wantBreakthrough: false,
			wantIsEscalation: false,
			wantRiskChange:   0.0,
		},
		{
			name:             "critical escalation (MEDIUM to CRITICAL)",
			initialScore:     0.5,
			initialLevel:     RiskMedium,
			hopNumber:        2,
			newScore:         0.95,
			newLevel:         RiskCritical,
			evidence:         "Security vulnerability + production incident",
			reasoning:        "Multiple severe risk factors converge",
			wantBreakthrough: true,
			wantIsEscalation: true,
			wantRiskChange:   0.45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewBreakthroughTracker(tt.initialScore, tt.initialLevel)

			gotBreakthrough := tracker.CheckAndRecordBreakthrough(
				tt.hopNumber,
				tt.newScore,
				tt.newLevel,
				tt.evidence,
				tt.reasoning,
			)

			if gotBreakthrough != tt.wantBreakthrough {
				t.Errorf("CheckAndRecordBreakthrough() detected = %v, want %v",
					gotBreakthrough, tt.wantBreakthrough)
			}

			if tt.wantBreakthrough {
				if len(tracker.breakthroughs) != 1 {
					t.Fatalf("Expected 1 breakthrough, got %d", len(tracker.breakthroughs))
				}

				b := tracker.breakthroughs[0]

				if b.HopNumber != tt.hopNumber {
					t.Errorf("Breakthrough hop = %d, want %d", b.HopNumber, tt.hopNumber)
				}

				if math.Abs(b.RiskBefore-tt.initialScore) > floatTolerance {
					t.Errorf("Breakthrough risk before = %.10f, want %.10f", b.RiskBefore, tt.initialScore)
				}

				if math.Abs(b.RiskAfter-tt.newScore) > floatTolerance {
					t.Errorf("Breakthrough risk after = %.10f, want %.10f", b.RiskAfter, tt.newScore)
				}

				if math.Abs(b.RiskChange-tt.wantRiskChange) > floatTolerance {
					t.Errorf("Breakthrough risk change = %.10f, want %.10f", b.RiskChange, tt.wantRiskChange)
				}

				if b.IsEscalation != tt.wantIsEscalation {
					t.Errorf("Breakthrough is escalation = %v, want %v", b.IsEscalation, tt.wantIsEscalation)
				}

				if b.TriggeringEvidence != tt.evidence {
					t.Errorf("Breakthrough evidence = %q, want %q", b.TriggeringEvidence, tt.evidence)
				}

				if b.Reasoning != tt.reasoning {
					t.Errorf("Breakthrough reasoning = %q, want %q", b.Reasoning, tt.reasoning)
				}

				if b.RiskLevelBefore != tt.initialLevel {
					t.Errorf("Breakthrough level before = %s, want %s", b.RiskLevelBefore, tt.initialLevel)
				}

				if b.RiskLevelAfter != tt.newLevel {
					t.Errorf("Breakthrough level after = %s, want %s", b.RiskLevelAfter, tt.newLevel)
				}
			} else {
				if len(tracker.breakthroughs) != 0 {
					t.Errorf("Expected no breakthroughs, got %d", len(tracker.breakthroughs))
				}
			}

			// Verify tracker state was updated regardless of breakthrough
			if math.Abs(tracker.previousRiskScore-tt.newScore) > floatTolerance {
				t.Errorf("Tracker previous score = %.10f, want %.10f (should update even without breakthrough)",
					tracker.previousRiskScore, tt.newScore)
			}

			if tracker.previousRiskLevel != tt.newLevel {
				t.Errorf("Tracker previous level = %s, want %s (should update even without breakthrough)",
					tracker.previousRiskLevel, tt.newLevel)
			}
		})
	}
}

func TestMultipleBreakthroughs(t *testing.T) {
	tracker := NewBreakthroughTracker(0.3, RiskLow)

	// Hop 1: No breakthrough (small change)
	detected1 := tracker.CheckAndRecordBreakthrough(
		1,
		0.35,
		RiskLow,
		"Minor coupling increase",
		"Slight increase",
	)
	if detected1 {
		t.Error("Hop 1 should not trigger breakthrough (0.05 change)")
	}

	// Hop 2: Breakthrough (escalation to HIGH)
	detected2 := tracker.CheckAndRecordBreakthrough(
		2,
		0.75,
		RiskHigh,
		"Critical incident found",
		"Past production outage",
	)
	if !detected2 {
		t.Error("Hop 2 should trigger breakthrough (0.4 change)")
	}

	// Hop 3: No breakthrough (minor change)
	detected3 := tracker.CheckAndRecordBreakthrough(
		3,
		0.8,
		RiskCritical,
		"Ownership transition",
		"New owner",
	)
	if detected3 {
		t.Error("Hop 3 should not trigger breakthrough (0.05 change)")
	}

	// Hop 4: Breakthrough (de-escalation to MEDIUM)
	detected4 := tracker.CheckAndRecordBreakthrough(
		4,
		0.45,
		RiskMedium,
		"Comprehensive tests added",
		"Test coverage increased",
	)
	if !detected4 {
		t.Error("Hop 4 should trigger breakthrough (0.35 change)")
	}

	// Verify total breakthroughs
	if tracker.GetBreakthroughCount() != 2 {
		t.Errorf("Expected 2 breakthroughs, got %d", tracker.GetBreakthroughCount())
	}

	// Verify breakthrough details
	breakthroughs := tracker.GetBreakthroughs()
	if len(breakthroughs) != 2 {
		t.Fatalf("Expected 2 breakthroughs in list, got %d", len(breakthroughs))
	}

	// First breakthrough (escalation)
	if breakthroughs[0].HopNumber != 2 {
		t.Errorf("First breakthrough hop = %d, want 2", breakthroughs[0].HopNumber)
	}
	if !breakthroughs[0].IsEscalation {
		t.Error("First breakthrough should be escalation")
	}

	// Second breakthrough (de-escalation)
	if breakthroughs[1].HopNumber != 4 {
		t.Errorf("Second breakthrough hop = %d, want 4", breakthroughs[1].HopNumber)
	}
	if breakthroughs[1].IsEscalation {
		t.Error("Second breakthrough should be de-escalation")
	}
}

func TestGetMostSignificantBreakthrough(t *testing.T) {
	tests := []struct {
		name                  string
		breakthroughs         []struct{ score, change float64 }
		wantMostSignificantIdx int
		wantChange            float64
	}{
		{
			name: "single breakthrough",
			breakthroughs: []struct{ score, change float64 }{
				{0.3, 0.25},
			},
			wantMostSignificantIdx: 0,
			wantChange:            0.25,
		},
		{
			name: "multiple breakthroughs - largest is positive",
			breakthroughs: []struct{ score, change float64 }{
				{0.3, 0.25},
				{0.55, 0.45}, // Largest
				{0.95, -0.3},
			},
			wantMostSignificantIdx: 1,
			wantChange:            0.45,
		},
		{
			name: "multiple breakthroughs - largest is negative",
			breakthroughs: []struct{ score, change float64 }{
				{0.8, 0.25},
				{0.95, -0.55}, // Largest (absolute)
				{0.4, 0.3},
			},
			wantMostSignificantIdx: 1,
			wantChange:            -0.55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewBreakthroughTracker(0.3, RiskLow)

			// Add breakthroughs
			currentScore := 0.3
			for i, b := range tt.breakthroughs {
				newScore := currentScore + b.change
				tracker.CheckAndRecordBreakthrough(
					i+1,
					newScore,
					ScoreToRiskLevel(newScore),
					"Evidence",
					"Reasoning",
				)
				currentScore = newScore
			}

			mostSig := tracker.GetMostSignificantBreakthrough()
			if mostSig == nil {
				t.Fatal("GetMostSignificantBreakthrough() returned nil")
			}

			if math.Abs(mostSig.RiskChange-tt.wantChange) > floatTolerance {
				t.Errorf("Most significant breakthrough change = %.10f, want %.10f",
					mostSig.RiskChange, tt.wantChange)
			}
		})
	}
}

func TestGetMostSignificantBreakthrough_NoBreakthroughs(t *testing.T) {
	tracker := NewBreakthroughTracker(0.3, RiskLow)

	mostSig := tracker.GetMostSignificantBreakthrough()
	if mostSig != nil {
		t.Error("GetMostSignificantBreakthrough() should return nil when no breakthroughs")
	}
}

func TestSetSignificanceThreshold(t *testing.T) {
	tracker := NewBreakthroughTracker(0.3, RiskLow)

	// Change threshold to 0.3 (30%)
	tracker.SetSignificanceThreshold(0.3)

	// 0.2 change should NOT trigger (below new threshold)
	detected1 := tracker.CheckAndRecordBreakthrough(1, 0.5, RiskMedium, "Evidence", "Reason")
	if detected1 {
		t.Error("0.2 change should not trigger with 0.3 threshold")
	}

	// 0.4 change SHOULD trigger (above new threshold)
	detected2 := tracker.CheckAndRecordBreakthrough(2, 0.9, RiskCritical, "Evidence", "Reason")
	if !detected2 {
		t.Error("0.4 change should trigger with 0.3 threshold")
	}
}

func TestFormatBreakthroughsForLLM(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func() *BreakthoughTracker
		wantContains []string
	}{
		{
			name: "no breakthroughs",
			setupFunc: func() *BreakthoughTracker {
				return NewBreakthroughTracker(0.3, RiskLow)
			},
			wantContains: []string{"No significant risk level changes detected"},
		},
		{
			name: "single escalation",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.3, RiskLow)
				tracker.CheckAndRecordBreakthrough(2, 0.7, RiskHigh, "Critical incident", "Past outage")
				return tracker
			},
			wantContains: []string{
				"BREAKTHROUGH POINTS (1 detected)",
				"Hop 2",
				"ESCALATION",
				"0.30 → 0.70",
				"LOW → HIGH",
				"Critical incident",
				"Past outage",
			},
		},
		{
			name: "multiple breakthroughs",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.3, RiskLow)
				tracker.CheckAndRecordBreakthrough(2, 0.75, RiskHigh, "Incident found", "Critical")
				tracker.CheckAndRecordBreakthrough(3, 0.4, RiskMedium, "Tests added", "Coverage improved")
				return tracker
			},
			wantContains: []string{
				"BREAKTHROUGH POINTS (2 detected)",
				"1. Hop 2",
				"ESCALATION",
				"2. Hop 3",
				"DE-ESCALATION",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := tt.setupFunc()
			formatted := tracker.FormatBreakthroughsForLLM()

			for _, want := range tt.wantContains {
				if !strings.Contains(formatted, want) {
					t.Errorf("FormatBreakthroughsForLLM() missing expected substring %q\nGot:\n%s",
						want, formatted)
				}
			}
		})
	}
}

func TestFormatBreakthroughsForUser(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *BreakthoughTracker
		wantCount int
		wantContains []string
	}{
		{
			name: "no breakthroughs",
			setupFunc: func() *BreakthoughTracker {
				return NewBreakthroughTracker(0.3, RiskLow)
			},
			wantCount: 0,
		},
		{
			name: "escalation",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.3, RiskLow)
				tracker.CheckAndRecordBreakthrough(2, 0.7, RiskHigh, "Critical incident", "")
				return tracker
			},
			wantCount: 1,
			wantContains: []string{
				"⚠️",
				"Hop 2",
				"escalated",
				"LOW to HIGH",
				"Critical incident",
			},
		},
		{
			name: "de-escalation",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.7, RiskHigh)
				tracker.CheckAndRecordBreakthrough(3, 0.3, RiskLow, "Tests added", "")
				return tracker
			},
			wantCount: 1,
			wantContains: []string{
				"✓",
				"Hop 3",
				"reduced",
				"HIGH to LOW",
				"Tests added",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := tt.setupFunc()
			formatted := tracker.FormatBreakthroughsForUser()

			if len(formatted) != tt.wantCount {
				t.Errorf("FormatBreakthroughsForUser() count = %d, want %d",
					len(formatted), tt.wantCount)
			}

			if tt.wantCount > 0 {
				combined := strings.Join(formatted, " ")
				for _, want := range tt.wantContains {
					if !strings.Contains(combined, want) {
						t.Errorf("FormatBreakthroughsForUser() missing expected substring %q\nGot:\n%s",
							want, combined)
					}
				}
			}
		})
	}
}

func TestGetInvestigationSummary(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func() *BreakthoughTracker
		wantContains []string
	}{
		{
			name: "no breakthroughs",
			setupFunc: func() *BreakthoughTracker {
				return NewBreakthroughTracker(0.3, RiskLow)
			},
			wantContains: []string{"no significant risk level changes"},
		},
		{
			name: "only escalations",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.3, RiskLow)
				tracker.CheckAndRecordBreakthrough(1, 0.6, RiskHigh, "E1", "")
				tracker.CheckAndRecordBreakthrough(2, 0.9, RiskCritical, "E2", "")
				return tracker
			},
			wantContains: []string{"2 breakthrough point(s)", "2 risk escalation(s)"},
		},
		{
			name: "only de-escalations",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.8, RiskCritical)
				tracker.CheckAndRecordBreakthrough(1, 0.4, RiskMedium, "D1", "")
				return tracker
			},
			wantContains: []string{"1 breakthrough point(s)", "1 risk reduction(s)"},
		},
		{
			name: "mixed escalations and de-escalations",
			setupFunc: func() *BreakthoughTracker {
				tracker := NewBreakthroughTracker(0.3, RiskLow)
				tracker.CheckAndRecordBreakthrough(1, 0.7, RiskHigh, "E1", "")
				tracker.CheckAndRecordBreakthrough(2, 0.4, RiskMedium, "D1", "")
				tracker.CheckAndRecordBreakthrough(3, 0.8, RiskCritical, "E2", "")
				return tracker
			},
			wantContains: []string{
				"3 breakthrough point(s)",
				"2 risk escalation(s)",
				"1 risk reduction(s)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := tt.setupFunc()
			summary := tracker.GetInvestigationSummary()

			for _, want := range tt.wantContains {
				if !strings.Contains(summary, want) {
					t.Errorf("GetInvestigationSummary() missing expected substring %q\nGot: %s",
						want, summary)
				}
			}
		})
	}
}

func TestScoreToRiskLevel(t *testing.T) {
	tests := []struct {
		score float64
		want  RiskLevel
	}{
		{0.0, RiskMinimal},
		{0.1, RiskMinimal},
		{0.19, RiskMinimal},
		{0.2, RiskLow},
		{0.3, RiskLow},
		{0.39, RiskLow},
		{0.4, RiskMedium},
		{0.5, RiskMedium},
		{0.59, RiskMedium},
		{0.6, RiskHigh},
		{0.7, RiskHigh},
		{0.79, RiskHigh},
		{0.8, RiskCritical},
		{0.9, RiskCritical},
		{1.0, RiskCritical},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			got := ScoreToRiskLevel(tt.score)
			if got != tt.want {
				t.Errorf("ScoreToRiskLevel(%.2f) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

// Benchmark tests

func BenchmarkCheckAndRecordBreakthrough(b *testing.B) {
	tracker := NewBreakthroughTracker(0.3, RiskLow)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between breakthrough and non-breakthrough
		score := 0.3 + float64(i%2)*0.4
		tracker.CheckAndRecordBreakthrough(i, score, ScoreToRiskLevel(score), "Evidence", "Reason")
	}
}

func BenchmarkFormatBreakthroughsForLLM(b *testing.B) {
	tracker := NewBreakthroughTracker(0.3, RiskLow)
	tracker.CheckAndRecordBreakthrough(1, 0.7, RiskHigh, "E1", "R1")
	tracker.CheckAndRecordBreakthrough(2, 0.4, RiskMedium, "E2", "R2")
	tracker.CheckAndRecordBreakthrough(3, 0.8, RiskCritical, "E3", "R3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.FormatBreakthroughsForLLM()
	}
}
