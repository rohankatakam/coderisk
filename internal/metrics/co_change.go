package metrics

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/graph"
)

// CoChangeResult represents the temporal co-change metric result
// Reference: risk_assessment_methodology.md §2.2 - Temporal Co-Change
type CoChangeResult struct {
	FilePath     string            `json:"file_path"`
	MaxFrequency float64           `json:"max_frequency"` // Highest co-change frequency
	RiskLevel    RiskLevel         `json:"risk_level"`    // LOW, MEDIUM, HIGH
	Partners     []CoChangePartner `json:"partners"`      // Files that co-change (top 5)
}

// CoChangePartner represents a file that frequently changes together
type CoChangePartner struct {
	FilePath  string  `json:"file_path"`
	Frequency float64 `json:"frequency"` // 0.0 to 1.0
}

// CalculateCoChange computes temporal co-change frequency for a file
// Reference: risk_assessment_methodology.md §2.2
// Formula: co_change_frequency = COUNT(commits where both changed) / COUNT(commits where either changed)
// Pre-computed as CO_CHANGED edges during graph construction
func CalculateCoChange(ctx context.Context, neo4j *graph.Client, repoID, filePath string) (*CoChangeResult, error) {
	// Query Neo4j for co-change count (pre-computed CO_CHANGED edges)
	// Note: This currently returns count, but should return list of partners with frequencies
	// Reference: risk_assessment_methodology.md §2.2 - Graph query
	count, err := neo4j.QueryCoChange(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query co-change for %s: %w", filePath, err)
	}

	// For now, use count as proxy for frequency (will enhance with actual CO_CHANGED edge weights)
	// TODO: Update when graph construction creates CO_CHANGED relationships with frequency property
	maxFrequency := estimateFrequencyFromCount(count)

	// Determine risk level using thresholds from risk_assessment_methodology.md §2.2
	// ≤0.3: LOW, 0.3-0.7: MEDIUM, >0.7: HIGH
	riskLevel := classifyCoChangeRisk(maxFrequency)

	result := &CoChangeResult{
		FilePath:     filePath,
		MaxFrequency: maxFrequency,
		RiskLevel:    riskLevel,
		Partners:     []CoChangePartner{}, // Will populate when CO_CHANGED edges exist
	}

	return result, nil
}

// estimateFrequencyFromCount converts count to estimated frequency
// Temporary heuristic until CO_CHANGED edges have frequency property
func estimateFrequencyFromCount(count int) float64 {
	// Simple heuristic: normalize count to 0-1 range
	// This is a placeholder - real implementation reads edge.frequency property
	if count == 0 {
		return 0.0
	} else if count <= 2 {
		return 0.2 // LOW
	} else if count <= 5 {
		return 0.5 // MEDIUM
	}
	return 0.8 // HIGH
}

// classifyCoChangeRisk applies threshold logic from risk_assessment_methodology.md §2.2
func classifyCoChangeRisk(frequency float64) RiskLevel {
	if frequency <= 0.3 {
		return RiskLevelLow // Weak coupling
	} else if frequency <= 0.7 {
		return RiskLevelMedium // Moderate coupling
	}
	return RiskLevelHigh // Strong evolutionary coupling
}

// FormatEvidence generates human-readable evidence string
// Reference: risk_assessment_methodology.md §2.2 - Evidence format
func (c *CoChangeResult) FormatEvidence() string {
	if c.MaxFrequency == 0 {
		return "No co-change pattern detected"
	}
	return fmt.Sprintf("Co-changes with %.0f%% frequency (%s coupling)", c.MaxFrequency*100, c.RiskLevel)
}

// ShouldEscalate returns true if this metric triggers Phase 2
// Reference: risk_assessment_methodology.md §2.4 - Escalation logic
func (c *CoChangeResult) ShouldEscalate() bool {
	return c.MaxFrequency > 0.7 // frequency > 0.7 → escalate
}

// CalculateCoChangeMultiple computes co-change across multiple file paths
// This handles renamed files by querying ALL historical paths and aggregating results
func CalculateCoChangeMultiple(ctx context.Context, neo4j *graph.Client, repoID string, filePaths []string) (*CoChangeResult, error) {
	if len(filePaths) == 0 {
		return &CoChangeResult{}, nil
	}

	// Query Neo4j with ALL file paths (current + historical renames)
	// This captures co-change history even after renames
	query := `
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)
		WHERE f1.path IN $paths
		WITH COUNT(DISTINCT c) as total_commits
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)-[:MODIFIED]->(f2:File)
		WHERE f1.path IN $paths AND f1.path <> f2.path
		WITH f2.path as partner_file, COUNT(DISTINCT c) as cochange_count, total_commits
		WITH partner_file, cochange_count, (cochange_count * 1.0 / total_commits) as frequency
		ORDER BY cochange_count DESC
		LIMIT 20
		RETURN partner_file, cochange_count, frequency
	`

	results, err := neo4j.ExecuteQuery(ctx, query, map[string]any{
		"paths": filePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query co-change for paths %v: %w", filePaths, err)
	}

	// Parse results into partners
	var partners []CoChangePartner
	var maxFrequency float64

	for _, row := range results {
		partnerFile, _ := row["partner_file"].(string)
		frequency, _ := row["frequency"].(float64)

		if frequency > maxFrequency {
			maxFrequency = frequency
		}

		partners = append(partners, CoChangePartner{
			FilePath:  partnerFile,
			Frequency: frequency,
		})
	}

	// Determine risk level
	riskLevel := classifyCoChangeRisk(maxFrequency)

	// Take top 5 partners
	if len(partners) > 5 {
		partners = partners[:5]
	}

	result := &CoChangeResult{
		FilePath:     filePaths[0], // Use current path as primary
		MaxFrequency: maxFrequency,
		RiskLevel:    riskLevel,
		Partners:     partners,
	}

	return result, nil
}

