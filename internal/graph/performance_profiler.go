package graph

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// PerformanceProfile represents a single query performance measurement
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 8
type PerformanceProfile struct {
	Operation    string
	Query        string
	Duration     time.Duration
	RecordsCount int
	Timestamp    time.Time
	Metadata     map[string]any
}

// PerformanceProfiler tracks query performance for regression detection
type PerformanceProfiler struct {
	profiles []PerformanceProfile
	logger   *slog.Logger
	enabled  bool
}

// NewPerformanceProfiler creates a profiler
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		profiles: make([]PerformanceProfile, 0),
		logger:   slog.Default().With("component", "performance_profiler"),
		enabled:  true,
	}
}

// Profile wraps a query execution and records performance
//
// Example usage:
//   profiler := NewPerformanceProfiler()
//   result, err := profiler.Profile(ctx, "QueryCoupling", query, func() (any, error) {
//     return client.QueryCoupling(ctx, filePath)
//   })
func (pp *PerformanceProfiler) Profile(
	ctx context.Context,
	operation string,
	query string,
	fn func() (any, error),
) (any, error) {
	if !pp.enabled {
		return fn()
	}

	start := time.Now()
	result, err := fn()
	duration := time.Since(start)

	// Extract record count if available
	recordCount := 0
	if result != nil {
		// Try to get count from various result types
		switch r := result.(type) {
		case []map[string]any:
			recordCount = len(r)
		case int:
			recordCount = r
		}
	}

	profile := PerformanceProfile{
		Operation:    operation,
		Query:        query,
		Duration:     duration,
		RecordsCount: recordCount,
		Timestamp:    time.Now(),
		Metadata:     make(map[string]any),
	}

	if err != nil {
		profile.Metadata["error"] = err.Error()
	}

	pp.profiles = append(pp.profiles, profile)

	// Log slow queries (>1s)
	if duration > time.Second {
		pp.logger.Warn("slow query detected",
			"operation", operation,
			"duration_seconds", duration.Seconds(),
			"records", recordCount)
	}

	return result, err
}

// GetProfiles returns all collected profiles
func (pp *PerformanceProfiler) GetProfiles() []PerformanceProfile {
	return pp.profiles
}

// GetProfilesByOperation returns profiles for a specific operation
func (pp *PerformanceProfiler) GetProfilesByOperation(operation string) []PerformanceProfile {
	var result []PerformanceProfile
	for _, p := range pp.profiles {
		if p.Operation == operation {
			result = append(result, p)
		}
	}
	return result
}

// GetStats calculates statistics for an operation
func (pp *PerformanceProfiler) GetStats(operation string) *PerformanceStats {
	profiles := pp.GetProfilesByOperation(operation)
	if len(profiles) == 0 {
		return nil
	}

	stats := &PerformanceStats{
		Operation:    operation,
		SampleCount:  len(profiles),
		TotalRecords: 0,
	}

	var totalDuration time.Duration
	minDuration := profiles[0].Duration
	maxDuration := profiles[0].Duration

	for _, p := range profiles {
		totalDuration += p.Duration
		stats.TotalRecords += p.RecordsCount

		if p.Duration < minDuration {
			minDuration = p.Duration
		}
		if p.Duration > maxDuration {
			maxDuration = p.Duration
		}
	}

	stats.AvgDuration = totalDuration / time.Duration(len(profiles))
	stats.MinDuration = minDuration
	stats.MaxDuration = maxDuration

	return stats
}

// PerformanceStats aggregated statistics for an operation
type PerformanceStats struct {
	Operation    string
	SampleCount  int
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
	TotalRecords int
}

// LogStats logs performance statistics
func (pp *PerformanceProfiler) LogStats() {
	operations := make(map[string]bool)
	for _, p := range pp.profiles {
		operations[p.Operation] = true
	}

	pp.logger.Info("performance profile summary",
		"total_queries", len(pp.profiles),
		"unique_operations", len(operations))

	for operation := range operations {
		stats := pp.GetStats(operation)
		pp.logger.Info("operation stats",
			"operation", operation,
			"samples", stats.SampleCount,
			"avg_duration_ms", stats.AvgDuration.Milliseconds(),
			"min_duration_ms", stats.MinDuration.Milliseconds(),
			"max_duration_ms", stats.MaxDuration.Milliseconds(),
			"total_records", stats.TotalRecords)
	}
}

// Reset clears all collected profiles
func (pp *PerformanceProfiler) Reset() {
	pp.profiles = make([]PerformanceProfile, 0)
}

// Enable/Disable profiling
func (pp *PerformanceProfiler) SetEnabled(enabled bool) {
	pp.enabled = enabled
}

// PerformanceBaseline represents expected performance metrics
// Used for regression detection
type PerformanceBaseline struct {
	Operation       string
	MaxDuration     time.Duration
	MaxRecords      int
	Description     string
}

// DefaultBaselines returns baseline performance targets
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md targets
func DefaultBaselines() []PerformanceBaseline {
	return []PerformanceBaseline{
		{
			Operation:   "QueryCoupling",
			MaxDuration: 150 * time.Millisecond,
			MaxRecords:  1000,
			Description: "Tier 1 metric - structural coupling",
		},
		{
			Operation:   "QueryCoChange",
			MaxDuration: 150 * time.Millisecond,
			MaxRecords:  500,
			Description: "Tier 1 metric - temporal co-change",
		},
		{
			Operation:   "ownership_query",
			MaxDuration: 1 * time.Second,
			MaxRecords:  100,
			Description: "Tier 2 metric - ownership churn",
		},
		{
			Operation:   "layer1_ingestion",
			MaxDuration: 5 * time.Minute,
			MaxRecords:  10000,
			Description: "Layer 1 ingestion for 5K files",
		},
		{
			Operation:   "layer2_ingestion",
			MaxDuration: 10 * time.Minute,
			MaxRecords:  5000,
			Description: "Layer 2 git history ingestion",
		},
	}
}

// CheckRegression compares profile against baseline
// Returns true if regression detected (exceeded baseline)
func CheckRegression(profile PerformanceProfile, baseline PerformanceBaseline) (bool, string) {
	if profile.Duration > baseline.MaxDuration {
		return true, fmt.Sprintf("Duration %v exceeds baseline %v",
			profile.Duration, baseline.MaxDuration)
	}

	if profile.RecordsCount > baseline.MaxRecords {
		return true, fmt.Sprintf("Record count %d exceeds baseline %d",
			profile.RecordsCount, baseline.MaxRecords)
	}

	return false, ""
}

// RegressionDetector checks for performance regressions
type RegressionDetector struct {
	baselines map[string]PerformanceBaseline
	logger    *slog.Logger
}

// NewRegressionDetector creates a detector with default baselines
func NewRegressionDetector() *RegressionDetector {
	baselines := make(map[string]PerformanceBaseline)
	for _, b := range DefaultBaselines() {
		baselines[b.Operation] = b
	}

	return &RegressionDetector{
		baselines: baselines,
		logger:    slog.Default().With("component", "regression_detector"),
	}
}

// Check checks if a profile represents a regression
func (rd *RegressionDetector) Check(profile PerformanceProfile) (bool, string) {
	baseline, ok := rd.baselines[profile.Operation]
	if !ok {
		// No baseline for this operation
		return false, ""
	}

	return CheckRegression(profile, baseline)
}

// CheckAll checks all profiles for regressions
// Returns list of regressions found
func (rd *RegressionDetector) CheckAll(profiles []PerformanceProfile) []string {
	var regressions []string

	for _, profile := range profiles {
		isRegression, message := rd.Check(profile)
		if isRegression {
			msg := fmt.Sprintf("[%s] %s (duration: %v)",
				profile.Operation, message, profile.Duration)
			regressions = append(regressions, msg)
			rd.logger.Warn("performance regression detected",
				"operation", profile.Operation,
				"message", message,
				"duration_seconds", profile.Duration.Seconds())
		}
	}

	return regressions
}

// AddBaseline adds a custom baseline
func (rd *RegressionDetector) AddBaseline(baseline PerformanceBaseline) {
	rd.baselines[baseline.Operation] = baseline
}
