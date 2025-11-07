package clqs

import (
	"context"
	"fmt"
)

const tightTemporalThreshold = 3600 // 1 hour in seconds

// CalculateTemporalPrecision computes Component 4: Temporal Precision (10% weight)
//
// Measures what percentage of links have tight temporal correlation (<1 hour between issue close and PR merge).
// Tight temporal correlation indicates good development practices and precise traceability.
//
// Score = (tight_temporal_links / total_links) × 100
func (c *Calculator) CalculateTemporalPrecision(ctx context.Context, repoID int64) (*ComponentBreakdown, error) {
	// Query temporal precision statistics
	stats, err := c.db.QueryTemporalPrecisionStats(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query temporal precision stats: %w", err)
	}

	if stats.TotalLinks == 0 {
		return &ComponentBreakdown{
			Name:         "Temporal Precision",
			Score:        0,
			Weight:       WeightTemporalPrecision,
			Contribution: 0,
			Status:       "Poor",
			Details: TemporalPrecisionDetails{
				TightTemporalLinks: 0,
				TotalLinks:         0,
				PrecisionRate:      0,
				Distribution:       map[string]int{},
			},
		}, nil
	}

	// Calculate precision rate
	precisionRate := float64(stats.TightTemporalLinks) / float64(stats.TotalLinks)
	score := precisionRate * 100.0
	contribution := score * WeightTemporalPrecision

	// Determine status
	status := determineStatus(score, 80, 60, 40)

	// Build details
	details := TemporalPrecisionDetails{
		TightTemporalLinks: stats.TightTemporalLinks,
		TotalLinks:         stats.TotalLinks,
		PrecisionRate:      precisionRate,
		Distribution: map[string]int{
			"under_5min":  stats.Under5Min,
			"5min_to_1hr": stats.FiveMinTo1Hr,
			"1hr_to_24hr": stats.OneHrTo24Hr,
			"over_24hr":   stats.Over24Hr,
		},
	}

	return &ComponentBreakdown{
		Name:         "Temporal Precision",
		Score:        score,
		Weight:       WeightTemporalPrecision,
		Contribution: contribution,
		Status:       status,
		Details:      details,
	}, nil
}
