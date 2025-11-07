package clqs

import (
	"context"
	"fmt"
)

const maxEvidenceTypes = 6.0

// CalculateEvidenceDiversity computes Component 3: Evidence Diversity (20% weight)
//
// Measures the average number of distinct evidence types per link.
// Evidence categories: explicit, github_timeline_verified, bidirectional, semantic (any), temporal, file_context
//
// Score = (avg_evidence_types / 6) × 100
func (c *Calculator) CalculateEvidenceDiversity(ctx context.Context, repoID int64) (*ComponentBreakdown, error) {
	// Query evidence diversity statistics
	stats, err := c.db.QueryEvidenceDiversityStats(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query evidence diversity stats: %w", err)
	}

	if stats.TotalLinks == 0 {
		return &ComponentBreakdown{
			Name:         "Evidence Diversity",
			Score:        0,
			Weight:       WeightEvidenceDiversity,
			Contribution: 0,
			Status:       "Poor",
			Details: EvidenceDiversityDetails{
				AvgEvidenceTypes: 0,
				MaxPossible:      6,
				Distribution:     map[string]int{},
				EvidenceTypeUsage: map[string]int{},
			},
		}, nil
	}

	// Normalize to 0-100 scale
	score := (stats.AvgEvidenceTypes / maxEvidenceTypes) * 100.0
	contribution := score * WeightEvidenceDiversity

	// Determine status
	status := determineStatus(score, 80, 70, 60)

	// Build details
	details := EvidenceDiversityDetails{
		AvgEvidenceTypes: stats.AvgEvidenceTypes,
		MaxPossible:      6,
		Distribution: map[string]int{
			"6_types": stats.SixTypes,
			"5_types": stats.FiveTypes,
			"4_types": stats.FourTypes,
			"3_types": stats.ThreeTypes,
			"2_types": stats.TwoTypes,
			"1_type":  stats.OneType,
		},
		EvidenceTypeUsage: map[string]int{
			"explicit":                 stats.ExplicitCount,
			"github_timeline_verified": stats.TimelineCount,
			"bidirectional":            stats.BidirectionalCount,
			"semantic":                 stats.SemanticCount,
			"temporal":                 stats.TemporalCount,
			"file_context":             stats.FileContextCount,
		},
	}

	return &ComponentBreakdown{
		Name:         "Evidence Diversity",
		Score:        score,
		Weight:       WeightEvidenceDiversity,
		Contribution: contribution,
		Status:       status,
		Details:      details,
	}, nil
}
