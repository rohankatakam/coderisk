package agent

import (
	"fmt"
	"time"
)

// Breakthrough represents a significant change in risk assessment during investigation
// 12-factor: Factor 8 - Own your control flow (track decision points for explainability)
type Breakthrough struct {
	HopNumber         int       // Which hop triggered the breakthrough
	RiskBefore        float64   // Risk score before this evidence (0.0-1.0)
	RiskAfter         float64   // Risk score after this evidence (0.0-1.0)
	RiskLevelBefore   RiskLevel // Risk level before (LOW, MEDIUM, HIGH, etc.)
	RiskLevelAfter    RiskLevel // Risk level after
	RiskChange        float64   // Absolute change (riskAfter - riskBefore)
	TriggeringEvidence string   // What evidence caused the change
	Reasoning         string    // LLM's explanation for the change
	Timestamp         time.Time // When the breakthrough occurred
	IsEscalation      bool      // True if risk increased, false if decreased
}

// BreakthoughTracker manages breakthrough detection during investigation
type BreakthoughTracker struct {
	breakthroughs        []Breakthrough
	significanceThreshold float64 // Minimum risk change to be considered significant (default: 0.2)
	previousRiskScore    float64
	previousRiskLevel    RiskLevel
}

const floatEpsilon = 1e-10 // Small epsilon for floating point comparisons

// NewBreakthroughTracker creates a new breakthrough tracker
func NewBreakthroughTracker(initialRiskScore float64, initialRiskLevel RiskLevel) *BreakthoughTracker {
	return &BreakthoughTracker{
		breakthroughs:        []Breakthrough{},
		significanceThreshold: 0.2, // 20% change is significant
		previousRiskScore:    initialRiskScore,
		previousRiskLevel:    initialRiskLevel,
	}
}

// SetSignificanceThreshold allows customizing the threshold for breakthroughs
func (bt *BreakthoughTracker) SetSignificanceThreshold(threshold float64) {
	bt.significanceThreshold = threshold
}

// CheckAndRecordBreakthrough checks if current risk represents a breakthrough
// Returns true if a breakthrough was detected and recorded
func (bt *BreakthoughTracker) CheckAndRecordBreakthrough(
	hopNumber int,
	currentRiskScore float64,
	currentRiskLevel RiskLevel,
	triggeringEvidence string,
	reasoning string,
) bool {
	// Calculate risk change
	riskChange := currentRiskScore - bt.previousRiskScore
	absChange := abs(riskChange)

	// Check if change is significant enough (>= to include exact threshold)
	// Use epsilon to handle floating point imprecision
	if absChange < bt.significanceThreshold-floatEpsilon {
		// Update previous values but don't record breakthrough
		bt.previousRiskScore = currentRiskScore
		bt.previousRiskLevel = currentRiskLevel
		return false
	}

	// Record breakthrough
	breakthrough := Breakthrough{
		HopNumber:          hopNumber,
		RiskBefore:         bt.previousRiskScore,
		RiskAfter:          currentRiskScore,
		RiskLevelBefore:    bt.previousRiskLevel,
		RiskLevelAfter:     currentRiskLevel,
		RiskChange:         riskChange,
		TriggeringEvidence: triggeringEvidence,
		Reasoning:          reasoning,
		Timestamp:          time.Now(),
		IsEscalation:       riskChange > 0,
	}

	bt.breakthroughs = append(bt.breakthroughs, breakthrough)

	// Update previous values
	bt.previousRiskScore = currentRiskScore
	bt.previousRiskLevel = currentRiskLevel

	return true
}

// GetBreakthroughs returns all recorded breakthroughs
func (bt *BreakthoughTracker) GetBreakthroughs() []Breakthrough {
	return bt.breakthroughs
}

// HasBreakthroughs returns true if any breakthroughs were detected
func (bt *BreakthoughTracker) HasBreakthroughs() bool {
	return len(bt.breakthroughs) > 0
}

// GetBreakthroughCount returns the number of breakthroughs detected
func (bt *BreakthoughTracker) GetBreakthroughCount() int {
	return len(bt.breakthroughs)
}

// GetMostSignificantBreakthrough returns the breakthrough with the largest risk change
func (bt *BreakthoughTracker) GetMostSignificantBreakthrough() *Breakthrough {
	if len(bt.breakthroughs) == 0 {
		return nil
	}

	maxIdx := 0
	maxChange := abs(bt.breakthroughs[0].RiskChange)

	for i := 1; i < len(bt.breakthroughs); i++ {
		change := abs(bt.breakthroughs[i].RiskChange)
		if change > maxChange {
			maxChange = change
			maxIdx = i
		}
	}

	return &bt.breakthroughs[maxIdx]
}

// FormatBreakthroughsForLLM formats breakthroughs for inclusion in LLM prompts
func (bt *BreakthoughTracker) FormatBreakthroughsForLLM() string {
	if len(bt.breakthroughs) == 0 {
		return "No significant risk level changes detected during investigation."
	}

	result := fmt.Sprintf("BREAKTHROUGH POINTS (%d detected):\n", len(bt.breakthroughs))
	for i, b := range bt.breakthroughs {
		direction := "↑ ESCALATION"
		if !b.IsEscalation {
			direction = "↓ DE-ESCALATION"
		}

		result += fmt.Sprintf("\n%d. Hop %d %s (%.2f → %.2f, Δ=%.2f)\n",
			i+1, b.HopNumber, direction, b.RiskBefore, b.RiskAfter, b.RiskChange)
		result += fmt.Sprintf("   Level: %s → %s\n", b.RiskLevelBefore, b.RiskLevelAfter)
		result += fmt.Sprintf("   Trigger: %s\n", b.TriggeringEvidence)
		if b.Reasoning != "" {
			result += fmt.Sprintf("   Reason: %s\n", b.Reasoning)
		}
	}

	return result
}

// FormatBreakthroughsForUser formats breakthroughs for display to end users
func (bt *BreakthoughTracker) FormatBreakthroughsForUser() []string {
	if len(bt.breakthroughs) == 0 {
		return []string{}
	}

	formatted := make([]string, len(bt.breakthroughs))
	for i, b := range bt.breakthroughs {
		direction := "escalated"
		emoji := "⚠️"
		if !b.IsEscalation {
			direction = "reduced"
			emoji = "✓"
		}

		formatted[i] = fmt.Sprintf("%s Hop %d: Risk %s from %s to %s (%+.0f%%) - %s",
			emoji,
			b.HopNumber,
			direction,
			b.RiskLevelBefore,
			b.RiskLevelAfter,
			b.RiskChange*100,
			b.TriggeringEvidence,
		)
	}

	return formatted
}

// GetInvestigationSummary generates a summary of the investigation including breakthroughs
func (bt *BreakthoughTracker) GetInvestigationSummary() string {
	if len(bt.breakthroughs) == 0 {
		return "Investigation completed with no significant risk level changes."
	}

	escalations := 0
	deescalations := 0
	for _, b := range bt.breakthroughs {
		if b.IsEscalation {
			escalations++
		} else {
			deescalations++
		}
	}

	summary := fmt.Sprintf("Investigation involved %d breakthrough point(s): ", len(bt.breakthroughs))
	if escalations > 0 {
		summary += fmt.Sprintf("%d risk escalation(s)", escalations)
	}
	if deescalations > 0 {
		if escalations > 0 {
			summary += " and "
		}
		summary += fmt.Sprintf("%d risk reduction(s)", deescalations)
	}
	summary += "."

	return summary
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ScoreToRiskLevel converts a risk score (0.0-1.0) to a RiskLevel
func ScoreToRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 0.8:
		return RiskCritical
	case score >= 0.6:
		return RiskHigh
	case score >= 0.4:
		return RiskMedium
	case score >= 0.2:
		return RiskLow
	default:
		return RiskMinimal
	}
}
