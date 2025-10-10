package output

import (
	"context"
	"log/slog"

	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/metrics"
	"github.com/coderisk/coderisk-go/internal/models"
)

// CalculateBlastRadius determines impact of changing a file
// 12-factor: Factor 4 - Tools are structured outputs
func CalculateBlastRadius(ctx context.Context, filePath string, graphClient *graph.Client) BlastRadius {
	radius := BlastRadius{}

	// Query 1: Direct dependents (1-hop)
	directQuery := `
		MATCH (f:File {path: $filePath})<-[:IMPORTS]-(dep:File)
		RETURN count(DISTINCT dep) as count
	`

	result, err := graphClient.ExecuteQuery(ctx, directQuery, map[string]any{
		"filePath": filePath,
	})
	if err != nil {
		slog.Warn("blast radius query failed", "error", err, "file", filePath)
		return radius
	}

	if len(result) > 0 {
		if count, ok := result[0]["count"].(int64); ok {
			radius.DirectDependents = int(count)
		}
	}

	// Query 2: Transitive dependents (2-3 hops)
	transitiveQuery := `
		MATCH (f:File {path: $filePath})<-[:IMPORTS*1..3]-(dep:File)
		RETURN count(DISTINCT dep) as count
	`

	result, err = graphClient.ExecuteQuery(ctx, transitiveQuery, map[string]any{
		"filePath": filePath,
	})
	if err == nil && len(result) > 0 {
		if count, ok := result[0]["count"].(int64); ok {
			radius.TransitiveDependents = int(count)
			radius.TotalAffectedFiles = radius.TransitiveDependents
		}
	}

	// Query 3: Critical paths (files on critical execution paths)
	// Heuristic: Files imported by >10 other files
	criticalQuery := `
		MATCH (f:File {path: $filePath})<-[:IMPORTS*1..2]-(dep:File)
		WHERE size((dep)<-[:IMPORTS]-()) > 10
		RETURN DISTINCT dep.path as path
		LIMIT 5
	`

	result, err = graphClient.ExecuteQuery(ctx, criticalQuery, map[string]any{
		"filePath": filePath,
	})
	if err == nil {
		for _, row := range result {
			if path, ok := row["path"].(string); ok {
				radius.CriticalPaths = append(radius.CriticalPaths, path)
			}
		}
	}

	return radius
}

// GetTemporalCoupling retrieves co-change patterns from Layer 2
// 12-factor: Factor 4 - Tools are structured outputs
func GetTemporalCoupling(ctx context.Context, filePath string, graphClient *graph.Client, minFrequency float64) []TemporalCouplingPair {
	var pairs []TemporalCouplingPair

	query := `
		MATCH (f:File {path: $filePath})-[r:CO_CHANGED]-(other:File)
		WHERE r.frequency >= $minFrequency
		RETURN other.path as file_b,
		       r.frequency as frequency,
		       r.co_changes as co_changes,
		       r.window_days as window_days
		ORDER BY r.frequency DESC
		LIMIT 10
	`

	result, err := graphClient.ExecuteQuery(ctx, query, map[string]any{
		"filePath":     filePath,
		"minFrequency": minFrequency,
	})
	if err != nil {
		slog.Warn("temporal coupling query failed", "error", err, "file", filePath)
		return pairs
	}

	for _, row := range result {
		pair := TemporalCouplingPair{
			FileA:      filePath,
			FileB:      row["file_b"].(string),
			Frequency:  row["frequency"].(float64),
			CoChanges:  int(row["co_changes"].(int64)),
			WindowDays: int(row["window_days"].(int64)),
		}
		pairs = append(pairs, pair)
	}

	return pairs
}

// IdentifyHotspots finds risky areas in the codebase
// 12-factor: Factor 4 - Tools are structured outputs
func IdentifyHotspots(ctx context.Context, phase1 *metrics.Phase1Result, riskResult *models.RiskResult, graphClient *graph.Client) []Hotspot {
	var hotspots []Hotspot

	// Hotspot criteria:
	// 1. High churn (many recent changes)
	// 2. Low test coverage
	// 3. Incident-prone (linked to incidents)
	// 4. Complex code (high cyclomatic complexity)

	// Check current file
	if shouldBeHotspot(phase1, riskResult) {
		hotspot := Hotspot{
			File:          phase1.FilePath,
			Score:         calculateHotspotScore(phase1, riskResult),
			Reason:        determineHotspotReason(phase1, riskResult),
			ChurnRate:     0.0, // TODO: Calculate from git history
			TestCoverage:  getTestCoverageValue(phase1),
			IncidentCount: 0, // TODO: Query from incidents database
		}
		hotspots = append(hotspots, hotspot)
	}

	// Query for related hotspots (co-changed files with similar issues)
	query := `
		MATCH (f:File {path: $filePath})-[r:CO_CHANGED]-(other:File)
		WHERE r.frequency > 0.6
		RETURN other.path as path
		LIMIT 5
	`

	result, err := graphClient.ExecuteQuery(ctx, query, map[string]any{
		"filePath": phase1.FilePath,
	})
	if err == nil {
		for _, row := range result {
			// Note: Would need metrics for other files to calculate score
			// For now, flag as potential hotspot
			hotspot := Hotspot{
				File:   row["path"].(string),
				Score:  0.6, // Default score for co-changed files
				Reason: "frequent_co_change",
			}
			hotspots = append(hotspots, hotspot)
		}
	}

	return hotspots
}

func shouldBeHotspot(phase1 *metrics.Phase1Result, result *models.RiskResult) bool {
	// Check for high risk factors
	hasCouplingIssue := phase1.Coupling != nil && phase1.Coupling.ShouldEscalate()
	hasTestIssue := phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate()
	hasCoChangeIssue := phase1.CoChange != nil && phase1.CoChange.ShouldEscalate()

	return hasCouplingIssue || hasTestIssue || hasCoChangeIssue
}

func calculateHotspotScore(phase1 *metrics.Phase1Result, result *models.RiskResult) float64 {
	score := 0.0

	// Factor 1: Coupling (0-0.3)
	if phase1.Coupling != nil {
		couplingCount := float64(phase1.Coupling.Count)
		if couplingCount > 15 {
			score += 0.3
		} else if couplingCount > 10 {
			score += 0.15
		}
	}

	// Factor 2: Test coverage (0-0.3)
	if phase1.TestRatio != nil {
		score += (1.0 - phase1.TestRatio.Ratio) * 0.3
	}

	// Factor 3: Co-change frequency (0-0.4)
	if phase1.CoChange != nil {
		score += phase1.CoChange.MaxFrequency * 0.4
	}

	return minFloat64(score, 1.0)
}

func determineHotspotReason(phase1 *metrics.Phase1Result, result *models.RiskResult) string {
	// Prioritize reasons
	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
			return "high_coupling_low_coverage"
		}
		return "high_coupling"
	}

	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		return "low_test_coverage"
	}

	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		return "high_temporal_coupling"
	}

	return "multiple_risk_factors"
}

func getTestCoverageValue(phase1 *metrics.Phase1Result) float64 {
	if phase1.TestRatio != nil {
		return phase1.TestRatio.Ratio
	}
	return 0.0
}
