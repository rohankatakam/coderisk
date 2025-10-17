package metrics

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/cache"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// CouplingResult represents the structural coupling metric result
// Reference: risk_assessment_methodology.md §2.1 - Structural Coupling
type CouplingResult struct {
	FilePath     string    `json:"file_path"`
	Count        int       `json:"count"`        // Number of dependencies
	RiskLevel    RiskLevel `json:"risk_level"`   // LOW, MEDIUM, HIGH
	Dependencies []string  `json:"dependencies"` // List of dependent files (optional)
}

// CalculateCoupling computes structural coupling for a file
// Reference: risk_assessment_methodology.md §2.1
// Formula: coupling_score(file) = COUNT(DISTINCT neighbor) WHERE (file)-[:IMPORTS|CALLS]-(neighbor)
// 12-factor: Factor 5 - Unify execution state (reads from graph, caches in Redis)
func CalculateCoupling(ctx context.Context, neo4j *graph.Client, redis *cache.Client, repoID, filePath string) (*CouplingResult, error) {
	// Try cache first (15-min TTL per risk_assessment_methodology.md §2.1)
	// 12-factor: Factor 3 - Own your context window (avoid redundant queries)
	cacheKey := cache.CacheKey("coupling", repoID, filePath)
	var cached CouplingResult
	hit, err := redis.Get(ctx, cacheKey, &cached)
	if err != nil {
		// Cache error is non-fatal, continue to Neo4j
		// Reference: DEVELOPMENT_WORKFLOW.md §4.2 - Error handling
		fmt.Printf("cache error (non-fatal): %v\n", err)
	} else if hit {
		return &cached, nil
	}

	// Query Neo4j for coupling count
	// Reference: risk_assessment_methodology.md §2.1 - Graph query
	count, err := neo4j.QueryCoupling(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query coupling for %s: %w", filePath, err)
	}

	// Determine risk level using thresholds from risk_assessment_methodology.md §2.1
	// ≤5: LOW, 5-10: MEDIUM, >10: HIGH
	riskLevel := classifyCouplingRisk(count)

	result := &CouplingResult{
		FilePath:  filePath,
		Count:     count,
		RiskLevel: riskLevel,
		// Dependencies list omitted for performance (can add with separate query)
	}

	// Cache result (15-min TTL)
	if err := redis.Set(ctx, cacheKey, result); err != nil {
		// Cache write failure is non-fatal
		fmt.Printf("failed to cache coupling result: %v\n", err)
	}

	return result, nil
}

// classifyCouplingRisk applies threshold logic from risk_assessment_methodology.md §2.1
func classifyCouplingRisk(count int) RiskLevel {
	if count <= 5 {
		return RiskLevelLow // Limited blast radius
	} else if count <= 10 {
		return RiskLevelMedium // Moderate impact
	}
	return RiskLevelHigh // Wide impact, escalate to Phase 2
}

// FormatEvidence generates human-readable evidence string
// Reference: risk_assessment_methodology.md §2.1 - Evidence format
func (c *CouplingResult) FormatEvidence() string {
	return fmt.Sprintf("File is connected to %d other files (%s coupling)", c.Count, c.RiskLevel)
}

// ShouldEscalate returns true if this metric triggers Phase 2
// Reference: risk_assessment_methodology.md §2.4 - Escalation logic
func (c *CouplingResult) ShouldEscalate() bool {
	return c.Count > 10 // coupling_count > 10 → escalate
}
