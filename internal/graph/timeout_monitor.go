package graph

import (
	"context"
	"log/slog"
	"time"
)

// TimeoutMonitor tracks query execution times and warns about approaching timeouts
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 5
type TimeoutMonitor struct {
	logger       *slog.Logger
	warningRatio float64 // Warn when execution reaches this % of timeout
}

// NewTimeoutMonitor creates a monitor with default settings
func NewTimeoutMonitor() *TimeoutMonitor {
	return &TimeoutMonitor{
		logger:       slog.Default().With("component", "timeout_monitor"),
		warningRatio: 0.8, // Warn at 80% of timeout
	}
}

// MonitorQueryExecution wraps a query execution and logs warnings if approaching timeout
// Returns the duration the query took
//
// Example usage:
//   monitor := NewTimeoutMonitor()
//   duration := monitor.MonitorQueryExecution(ctx, "QueryCoupling", 30*time.Second, func() error {
//     result, err := client.QueryCoupling(ctx, filePath)
//     return err
//   })
func (tm *TimeoutMonitor) MonitorQueryExecution(
	ctx context.Context,
	operation string,
	timeout time.Duration,
	fn func() error,
) time.Duration {
	start := time.Now()

	err := fn()
	duration := time.Since(start)

	// Calculate warning threshold
	warningThreshold := time.Duration(float64(timeout) * tm.warningRatio)

	// Log execution info
	if err != nil {
		// Check if it was a timeout error
		if duration >= timeout {
			tm.logger.Error("query timed out",
				"operation", operation,
				"duration_seconds", duration.Seconds(),
				"timeout_seconds", timeout.Seconds(),
				"error", err)
		} else {
			tm.logger.Warn("query failed",
				"operation", operation,
				"duration_seconds", duration.Seconds(),
				"timeout_seconds", timeout.Seconds(),
				"error", err)
		}
	} else if duration >= warningThreshold {
		// Warn if approaching timeout (even if successful)
		percentUsed := (duration.Seconds() / timeout.Seconds()) * 100
		tm.logger.Warn("query approaching timeout",
			"operation", operation,
			"duration_seconds", duration.Seconds(),
			"timeout_seconds", timeout.Seconds(),
			"percent_used", percentUsed)
	} else {
		// Normal execution
		tm.logger.Debug("query completed",
			"operation", operation,
			"duration_seconds", duration.Seconds())
	}

	return duration
}

// MonitorWithContext wraps a query execution with context timeout
// Automatically cancels if exceeding the specified timeout
//
// Example usage:
//   monitor := NewTimeoutMonitor()
//   err := monitor.MonitorWithContext(ctx, "layer2_ingestion", 10*time.Minute, func(ctx context.Context) error {
//     return processor.ProcessLayer2(ctx)
//   })
func (tm *TimeoutMonitor) MonitorWithContext(
	ctx context.Context,
	operation string,
	timeout time.Duration,
	fn func(context.Context) error,
) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Execute with timeout context
	err := fn(timeoutCtx)
	duration := time.Since(start)

	// Log results
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			tm.logger.Error("operation timed out",
				"operation", operation,
				"duration_seconds", duration.Seconds(),
				"timeout_seconds", timeout.Seconds())
		} else {
			tm.logger.Warn("operation failed",
				"operation", operation,
				"duration_seconds", duration.Seconds(),
				"error", err)
		}
		return err
	}

	// Warn if using significant portion of timeout
	warningThreshold := time.Duration(float64(timeout) * tm.warningRatio)
	if duration >= warningThreshold {
		percentUsed := (duration.Seconds() / timeout.Seconds()) * 100
		tm.logger.Warn("operation approaching timeout",
			"operation", operation,
			"duration_seconds", duration.Seconds(),
			"timeout_seconds", timeout.Seconds(),
			"percent_used", percentUsed)
	} else {
		tm.logger.Info("operation completed",
			"operation", operation,
			"duration_seconds", duration.Seconds())
	}

	return nil
}

// TimeoutStats tracks timeout statistics for analysis
type TimeoutStats struct {
	Operation         string
	TotalExecutions   int
	TimeoutCount      int
	AverageDuration   time.Duration
	MaxDuration       time.Duration
	TimeoutPercentage float64
}

// TimeoutTracker collects timeout statistics over time
type TimeoutTracker struct {
	stats  map[string]*TimeoutStats
	logger *slog.Logger
}

// NewTimeoutTracker creates a new tracker
func NewTimeoutTracker() *TimeoutTracker {
	return &TimeoutTracker{
		stats:  make(map[string]*TimeoutStats),
		logger: slog.Default().With("component", "timeout_tracker"),
	}
}

// RecordExecution records an execution result
func (tt *TimeoutTracker) RecordExecution(operation string, duration time.Duration, timedOut bool) {
	if tt.stats[operation] == nil {
		tt.stats[operation] = &TimeoutStats{
			Operation: operation,
		}
	}

	stats := tt.stats[operation]
	stats.TotalExecutions++

	if timedOut {
		stats.TimeoutCount++
	}

	// Update average duration
	if stats.TotalExecutions == 1 {
		stats.AverageDuration = duration
	} else {
		totalDuration := stats.AverageDuration.Nanoseconds() * int64(stats.TotalExecutions-1)
		stats.AverageDuration = time.Duration((totalDuration + duration.Nanoseconds()) / int64(stats.TotalExecutions))
	}

	// Update max duration
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	// Calculate timeout percentage
	if stats.TotalExecutions > 0 {
		stats.TimeoutPercentage = float64(stats.TimeoutCount) / float64(stats.TotalExecutions) * 100
	}
}

// GetStats returns statistics for an operation
func (tt *TimeoutTracker) GetStats(operation string) *TimeoutStats {
	return tt.stats[operation]
}

// GetAllStats returns all collected statistics
func (tt *TimeoutTracker) GetAllStats() map[string]*TimeoutStats {
	return tt.stats
}

// LogSummary logs a summary of all timeout statistics
func (tt *TimeoutTracker) LogSummary() {
	if len(tt.stats) == 0 {
		tt.logger.Info("no timeout statistics collected")
		return
	}

	tt.logger.Info("timeout statistics summary")
	for operation, stats := range tt.stats {
		tt.logger.Info("operation stats",
			"operation", operation,
			"total_executions", stats.TotalExecutions,
			"timeout_count", stats.TimeoutCount,
			"timeout_percentage", stats.TimeoutPercentage,
			"avg_duration_seconds", stats.AverageDuration.Seconds(),
			"max_duration_seconds", stats.MaxDuration.Seconds())
	}
}
