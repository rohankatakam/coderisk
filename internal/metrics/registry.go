package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rohankatakam/coderisk/internal/cache"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// Registry orchestrates all metric calculations for Phase 1 baseline assessment
// Reference: risk_assessment_methodology.md §2 - Tier 1 Metrics
// 12-factor: Factor 10 - Small, focused agents (each metric is independent)
type Registry struct {
	neo4j    *graph.Client
	redis    *cache.Client
	postgres *database.Client
	logger   *slog.Logger
}

// NewRegistry creates a new metric registry
func NewRegistry(neo4j *graph.Client, redis *cache.Client, postgres *database.Client) *Registry {
	return &Registry{
		neo4j:    neo4j,
		redis:    redis,
		postgres: postgres,
		logger:   slog.Default().With("component", "metrics"),
	}
}

// CalculatePhase1 executes all Tier 1 metrics for a file
// Reference: risk_assessment_methodology.md §2.4 - Phase 1 Heuristic
// 12-factor: Factor 8 - Own your control flow (explicit metric orchestration)
func (r *Registry) CalculatePhase1(ctx context.Context, repoID, filePath string) (*Phase1Result, error) {
	start := time.Now()

	r.logger.Info("starting phase 1 assessment", "file", filePath)

	result := &Phase1Result{
		FilePath: filePath,
	}

	// Calculate all Tier 1 metrics in parallel for performance
	// Reference: spec.md §2.4 NFR-1 - Phase 1 must complete in <500ms
	// 12-factor: Factor 3 - Own your context window (run metrics concurrently)

	// Channel for collecting metric results
	type metricResult struct {
		coupling  *CouplingResult
		coChange  *CoChangeResult
		testRatio *TestRatioResult
		err       error
	}

	resultChan := make(chan metricResult, 1)

	go func() {
		var mr metricResult

		// Calculate coupling
		coupling, err := CalculateCoupling(ctx, r.neo4j, r.redis, repoID, filePath)
		if err != nil {
			r.logger.Warn("coupling calculation failed", "error", err)
			mr.err = fmt.Errorf("coupling: %w", err)
		} else {
			mr.coupling = coupling
			// Record metric use in PostgreSQL for validation tracking
			if _, err := r.postgres.RecordMetricUse(ctx, "coupling", filePath, coupling); err != nil {
				r.logger.Warn("failed to record coupling metric", "error", err)
			}
		}

		// Calculate co-change
		coChange, err := CalculateCoChange(ctx, r.neo4j, r.redis, repoID, filePath)
		if err != nil {
			r.logger.Warn("co-change calculation failed", "error", err)
			// Don't fail entire check if one metric fails
		} else {
			mr.coChange = coChange
			if _, err := r.postgres.RecordMetricUse(ctx, "co_change", filePath, coChange); err != nil {
				r.logger.Warn("failed to record co-change metric", "error", err)
			}
		}

		// Calculate test ratio
		testRatio, err := CalculateTestRatio(ctx, r.neo4j, r.redis, repoID, filePath)
		if err != nil {
			r.logger.Warn("test ratio calculation failed", "error", err)
		} else {
			mr.testRatio = testRatio
			if _, err := r.postgres.RecordMetricUse(ctx, "test_ratio", filePath, testRatio); err != nil {
				r.logger.Warn("failed to record test_ratio metric", "error", err)
			}
		}

		resultChan <- mr
	}()

	// Wait for metrics to complete
	mr := <-resultChan

	if mr.err != nil {
		return nil, mr.err
	}

	result.Coupling = mr.coupling
	result.CoChange = mr.coChange
	result.TestRatio = mr.testRatio

	// Determine overall risk and escalation decision
	// Reference: risk_assessment_methodology.md §2.4 - Decision tree
	result.DetermineOverallRisk()

	// Calculate duration
	result.DurationMS = time.Since(start).Milliseconds()

	r.logger.Info("phase 1 complete",
		"file", filePath,
		"risk", result.OverallRisk,
		"escalate", result.ShouldEscalate,
		"duration_ms", result.DurationMS,
	)

	// Check performance target: Phase 1 should complete in <500ms
	// Reference: spec.md §2.4 NFR-1
	if result.DurationMS > 500 {
		r.logger.Warn("phase 1 exceeded performance target",
			"target_ms", 500,
			"actual_ms", result.DurationMS,
		)
	}

	return result, nil
}

// CheckMetricEnabled verifies if a metric should be used based on FP rate
// Reference: spec.md §6.2 constraint C-10 - Auto-disable metrics with >3% FP rate
func (r *Registry) CheckMetricEnabled(ctx context.Context, metricName string) (bool, error) {
	enabled, err := r.postgres.IsMetricEnabled(ctx, metricName)
	if err != nil {
		return true, err // Default to enabled if check fails
	}

	if !enabled {
		r.logger.Warn("metric disabled due to high FP rate", "metric", metricName)
	}

	return enabled, nil
}

// GetMetricStats retrieves aggregate statistics for a metric
// Reference: risk_assessment_methodology.md §5.2 - Validation tracking
func (r *Registry) GetMetricStats(ctx context.Context, metricName string) (*database.MetricStats, error) {
	return r.postgres.GetMetricStats(ctx, metricName)
}
