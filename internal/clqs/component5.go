package clqs

import (
	"context"
	"fmt"
)

// CalculateSemanticStrength computes Component 5: Semantic Strength (5% weight)
//
// Measures the average maximum semantic similarity score across all links.
// For each link, takes the max of (title_score, body_score, comment_score).
//
// Score = avg_max_semantic × 100
func (c *Calculator) CalculateSemanticStrength(ctx context.Context, repoID int64) (*ComponentBreakdown, error) {
	// Query semantic strength statistics
	stats, err := c.db.QuerySemanticStrengthStats(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic strength stats: %w", err)
	}

	if stats.TotalLinks == 0 {
		return &ComponentBreakdown{
			Name:         "Semantic Strength",
			Score:        0,
			Weight:       WeightSemanticStrength,
			Contribution: 0,
			Status:       "Poor",
			Details: SemanticStrengthDetails{
				AvgSemanticScore: 0,
				Distribution:     map[string]int{},
				AvgByType:        map[string]float64{},
			},
		}, nil
	}

	// Normalize to 0-100 scale
	score := stats.AvgMaxSemantic * 100.0
	contribution := score * WeightSemanticStrength

	// Determine status
	status := determineStatus(score, 80, 70, 60)

	// Build details
	details := SemanticStrengthDetails{
		AvgSemanticScore: stats.AvgMaxSemantic,
		Distribution: map[string]int{
			"high_semantic":   stats.HighSemantic,
			"medium_semantic": stats.MediumSemantic,
			"low_semantic":    stats.LowSemantic,
		},
		AvgByType: map[string]float64{
			"title":   stats.AvgTitle,
			"body":    stats.AvgBody,
			"comment": stats.AvgComment,
		},
	}

	return &ComponentBreakdown{
		Name:         "Semantic Strength",
		Score:        score,
		Weight:       WeightSemanticStrength,
		Contribution: contribution,
		Status:       status,
		Details:      details,
	}, nil
}
