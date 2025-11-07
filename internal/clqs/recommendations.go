package clqs

import (
	"fmt"
)

// generateRecommendations generates improvement recommendations based on component scores
func generateRecommendations(components []*ComponentBreakdown, stats *Statistics) []Recommendation {
	var recommendations []Recommendation
	priority := 1

	// Component 1: Link Coverage recommendations
	linkCoverage := components[0]
	if linkCovDetails, ok := linkCoverage.Details.(LinkCoverageDetails); ok {
		if linkCovDetails.UnlinkedIssues > 0 {
			impact := "high"
			if linkCovDetails.CoverageRate >= 0.80 {
				impact = "medium"
			}
			if linkCovDetails.CoverageRate >= 0.90 {
				impact = "low"
			}

			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Link Coverage",
				Message:     fmt.Sprintf("%d eligible issues lack PR links.", linkCovDetails.UnlinkedIssues),
				Action:      "Review unlinked closed issues and manually add references where applicable.",
				ImpactLevel: impact,
			})
			priority++
		}
	}

	// Component 2: Confidence Quality recommendations
	confQuality := components[1]
	if confDetails, ok := confQuality.Details.(ConfidenceQualityDetails); ok {
		if confDetails.LowConfidenceLinks.Count > 0 {
			impact := "medium"
			if confDetails.LowConfidenceLinks.Percentage > 20 {
				impact = "high"
			}

			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Confidence Quality",
				Message:     fmt.Sprintf("%d links have low confidence (<0.70).", confDetails.LowConfidenceLinks.Count),
				Action:      "Review low-confidence links for potential manual labeling or ground truth validation.",
				ImpactLevel: impact,
			})
			priority++
		}
	}

	// Component 3: Evidence Diversity recommendations
	evidenceDiv := components[2]
	if evidDetails, ok := evidenceDiv.Details.(EvidenceDiversityDetails); ok {
		if evidDetails.AvgEvidenceTypes < 4.0 {
			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Evidence Diversity",
				Message:     fmt.Sprintf("Average evidence diversity is %.1f/6 types.", evidDetails.AvgEvidenceTypes),
				Action:      "Encourage use of 'Fixes #N' syntax in PR descriptions to increase explicit references.",
				ImpactLevel: "low",
			})
			priority++
		}

		// Check if bidirectional links are underutilized
		totalLinks := stats.TotalLinks
		if totalLinks > 0 && evidDetails.EvidenceTypeUsage["bidirectional"] < totalLinks/2 {
			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Evidence Diversity",
				Message:     "Bidirectional links (issue mentions PR) are underutilized.",
				Action:      "When closing issues, reference the fixing PR in the closing comment.",
				ImpactLevel: "low",
			})
			priority++
		}
	}

	// Component 4: Temporal Precision recommendations
	temporalPrec := components[3]
	if tempDetails, ok := temporalPrec.Details.(TemporalPrecisionDetails); ok {
		if tempDetails.PrecisionRate < 0.60 {
			impact := "medium"
			if tempDetails.PrecisionRate < 0.40 {
				impact = "high"
			}

			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Temporal Precision",
				Message:     fmt.Sprintf("Only %.0f%% of links have tight temporal correlation (<1 hour).", tempDetails.PrecisionRate*100),
				Action:      "Encourage developers to close issues immediately after merging PRs for better traceability.",
				ImpactLevel: impact,
			})
			priority++
		}
	}

	// Component 5: Semantic Strength recommendations
	semanticStr := components[4]
	if semDetails, ok := semanticStr.Details.(SemanticStrengthDetails); ok {
		if semDetails.AvgSemanticScore < 0.70 {
			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Semantic Strength",
				Message:     fmt.Sprintf("Average semantic similarity is %.0f%%.", semDetails.AvgSemanticScore*100),
				Action:      "Encourage descriptive PR titles and bodies that reference the issue being fixed.",
				ImpactLevel: "low",
			})
			priority++
		}
	}

	// Overall CLQS recommendations
	if len(components) > 0 {
		// Find the weakest component
		weakestComp := components[0]
		for _, comp := range components {
			if comp.Score < weakestComp.Score {
				weakestComp = comp
			}
		}

		if weakestComp.Score < 70 {
			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Category:    "Overall CLQS",
				Message:     fmt.Sprintf("%s is the weakest component (%.1f/100).", weakestComp.Name, weakestComp.Score),
				Action:      fmt.Sprintf("Focus improvement efforts on %s to maximize CLQS gains.", weakestComp.Name),
				ImpactLevel: "high",
			})
		}
	}

	return recommendations
}

// calculateLabelingOpportunities calculates potential CLQS improvements from labeling
func calculateLabelingOpportunities(confQuality *ComponentBreakdown, currentCLQS float64) LabelingOpportunities {
	confDetails, ok := confQuality.Details.(ConfidenceQualityDetails)
	if !ok {
		return LabelingOpportunities{}
	}

	lowConfCount := confDetails.LowConfidenceLinks.Count
	ambiguousCount := 0

	// Estimate ambiguous links (those with confidence 0.50-0.65)
	if confDetails.LowConfidenceLinks.AvgConfidence < 0.65 {
		ambiguousCount = lowConfCount / 2 // Rough estimate
	}

	// Estimate potential CLQS gain
	// If we could boost 50% of low-confidence links to high-confidence:
	// - Confidence Quality component would improve by ~5-10 points
	// - That's 30% weight, so CLQS would improve by ~1.5-3 points
	potentialGain := 0.0
	if lowConfCount > 0 {
		// Conservative estimate: 0.05 points per low-confidence link improved
		potentialGain = float64(lowConfCount) * 0.05
		if potentialGain > 5.0 {
			potentialGain = 5.0 // Cap at 5 points max
		}
	}

	return LabelingOpportunities{
		LowConfidenceCount: lowConfCount,
		AmbiguousCount:     ambiguousCount,
		PotentialCLQSGain:  potentialGain,
	}
}
