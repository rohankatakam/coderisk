package graph

import (
	"context"
	"fmt"
	"time"
)

// PoolStats represents connection pool statistics
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 4
//
// Note: The Neo4j Go driver doesn't expose detailed pool statistics directly.
// For production monitoring, use Neo4j's built-in metrics endpoint:
//   http://localhost:7474/db/neo4j/metrics
type PoolStats struct {
	MaxPoolSize int
	// Additional fields would require Neo4j Enterprise metrics API
	// or custom instrumentation
}

// GetPoolStats retrieves current connection pool statistics
// Note: Limited information available from Go driver
func (c *Client) GetPoolStats() PoolStats {
	return PoolStats{
		MaxPoolSize: 50, // From configuration
		// Neo4j Go driver doesn't expose runtime pool metrics
		// Use Neo4j metrics endpoint for detailed monitoring
	}
}

// WatchPoolHealth monitors connection pool health
// Runs periodic health checks to detect connection issues early
//
// Example usage:
//   ctx, cancel := context.WithCancel(context.Background())
//   defer cancel()
//   go client.WatchPoolHealth(ctx, 30*time.Second)
func (c *Client) WatchPoolHealth(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.logger.Info("starting pool health monitor", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("pool health monitor stopped")
			return
		case <-ticker.C:
			// Perform health check
			if err := c.HealthCheck(ctx); err != nil {
				c.logger.Warn("pool health check failed", "error", err)
			} else {
				c.logger.Debug("pool health check passed")
			}
		}
	}
}

// MonitorPoolExhaustion logs warnings if connection acquisition takes too long
// This can indicate pool exhaustion or slow queries holding connections
//
// Usage: Wrap operations that might exhaust the pool
//   start := time.Now()
//   // ... perform query ...
//   client.MonitorPoolExhaustion(time.Since(start), "query_name")
func (c *Client) MonitorPoolExhaustion(duration time.Duration, operation string) {
	// Connection acquisition timeout is 60s (from config)
	// Warn if we're approaching the timeout (>30s = 50% threshold)
	if duration > 30*time.Second {
		c.logger.Warn("connection acquisition slow - possible pool exhaustion",
			"operation", operation,
			"duration_seconds", duration.Seconds(),
			"threshold_seconds", 30)
	}
}

// RecommendedPoolSize returns recommended pool size based on expected concurrency
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 4
func RecommendedPoolSize(expectedConcurrentRequests int) int {
	// Rule of thumb: pool_size = concurrent_requests * 1.5 (for safety margin)
	// Minimum: 10 connections
	// Maximum: 100 connections (to avoid overwhelming Neo4j)

	recommended := expectedConcurrentRequests * 3 / 2 // 1.5x multiplier

	if recommended < 10 {
		return 10
	}
	if recommended > 100 {
		return 100
	}
	return recommended
}

// PoolHealthStatus represents the health of the connection pool
type PoolHealthStatus struct {
	Healthy       bool
	Message       string
	LastCheckTime time.Time
}

// CheckPoolHealth performs a comprehensive health check
// Returns detailed status for monitoring/alerting
func (c *Client) CheckPoolHealth(ctx context.Context) (*PoolHealthStatus, error) {
	startTime := time.Now()

	// Perform connectivity check
	err := c.HealthCheck(ctx)

	status := &PoolHealthStatus{
		LastCheckTime: time.Now(),
	}

	if err != nil {
		status.Healthy = false
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		return status, err
	}

	// Check if health check took too long (>5s indicates issues)
	checkDuration := time.Since(startTime)
	if checkDuration > 5*time.Second {
		status.Healthy = false
		status.Message = fmt.Sprintf("Health check slow: %v (threshold: 5s)", checkDuration)
		return status, fmt.Errorf("health check timeout")
	}

	status.Healthy = true
	status.Message = fmt.Sprintf("Pool healthy (check took %v)", checkDuration)
	return status, nil
}
