package clqs

import (
	"context"
	"fmt"
)

// Confidence tier weights (from CLQS spec)
const (
	highConfidenceWeight   = 1.0
	mediumConfidenceWeight = 0.7
	lowConfidenceWeight    = 0.3
)

// CalculateConfidenceQuality computes Component 2: Confidence Quality (30% weight)
//
// Measures the average confidence of links, weighted by confidence tier.
// High (≥0.85): weight=1.0, Medium (0.70-0.84): weight=0.7, Low (<0.70): weight=0.3
//
// Score = weighted_average × 100
func (c *Calculator) CalculateConfidenceQuality(ctx context.Context, repoID int64) (*ComponentBreakdown, error) {
	// Query confidence statistics
	stats, err := c.db.QueryConfidenceStats(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query confidence stats: %w", err)
	}

	if stats.TotalLinks == 0 {
		return &ComponentBreakdown{
			Name:         "Confidence Quality",
			Score:        0,
			Weight:       WeightConfidenceQuality,
			Contribution: 0,
			Status:       "Poor",
			Details: ConfidenceQualityDetails{
				WeightedAverage: 0,
			},
		}, nil
	}

	// Calculate weighted average
	weightedSum := (float64(stats.HighCount) * stats.HighAvg * highConfidenceWeight) +
		(float64(stats.MediumCount) * stats.MediumAvg * mediumConfidenceWeight) +
		(float64(stats.LowCount) * stats.LowAvg * lowConfidenceWeight)

	weightSum := (float64(stats.HighCount) * highConfidenceWeight) +
		(float64(stats.MediumCount) * mediumConfidenceWeight) +
		(float64(stats.LowCount) * lowConfidenceWeight)

	var weightedAvg float64
	if weightSum > 0 {
		weightedAvg = weightedSum / weightSum
	}

	// Normalize to 0-100 scale
	score := weightedAvg * 100.0
	contribution := score * WeightConfidenceQuality

	// Determine status
	status := determineStatus(score, 85, 75, 60)

	// Build details
	details := ConfidenceQualityDetails{
		HighConfidenceLinks: ConfidenceTier{
			Count:         stats.HighCount,
			Percentage:    float64(stats.HighCount) / float64(stats.TotalLinks) * 100,
			AvgConfidence: stats.HighAvg,
		},
		MediumConfidenceLinks: ConfidenceTier{
			Count:         stats.MediumCount,
			Percentage:    float64(stats.MediumCount) / float64(stats.TotalLinks) * 100,
			AvgConfidence: stats.MediumAvg,
		},
		LowConfidenceLinks: ConfidenceTier{
			Count:         stats.LowCount,
			Percentage:    float64(stats.LowCount) / float64(stats.TotalLinks) * 100,
			AvgConfidence: stats.LowAvg,
		},
		WeightedAverage: weightedAvg,
	}

	return &ComponentBreakdown{
		Name:         "Confidence Quality",
		Score:        score,
		Weight:       WeightConfidenceQuality,
		Contribution: contribution,
		Status:       status,
		Details:      details,
	}, nil
}
