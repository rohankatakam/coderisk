package metrics

import (
	"context"
	"fmt"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/graph"
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
// 12-factor: Factor 5 - Unify execution state (reads pre-computed edges from graph)
func CalculateCoChange(ctx context.Context, neo4j *graph.Client, redis *cache.Client, repoID, filePath string) (*CoChangeResult, error) {
	// Try cache first (15-min TTL per risk_assessment_methodology.md §2.2)
	cacheKey := cache.CacheKey("co_change", repoID, filePath)
	var cached CoChangeResult
	hit, err := redis.Get(ctx, cacheKey, &cached)
	if err != nil {
		fmt.Printf("cache error (non-fatal): %v\n", err)
	} else if hit {
		return &cached, nil
	}

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

	// Cache result (15-min TTL)
	if err := redis.Set(ctx, cacheKey, result); err != nil {
		fmt.Printf("failed to cache co-change result: %v\n", err)
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
