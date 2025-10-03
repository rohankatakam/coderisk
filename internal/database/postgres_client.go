package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps PostgreSQL connection pool for metric validation storage
// Reference: risk_assessment_methodology.md §5.2 - Validation schema
type Client struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// MetricValidation represents user feedback on a metric result
// Maps to metric_validations table in PostgreSQL
type MetricValidation struct {
	ID             int             `json:"id"`
	MetricName     string          `json:"metric_name"`
	FilePath       string          `json:"file_path"`
	MetricValue    json.RawMessage `json:"metric_value"`
	UserFeedback   *string         `json:"user_feedback"`   // "true_positive", "false_positive", null
	FeedbackReason *string         `json:"feedback_reason"` // Optional explanation
}

// MetricStats represents aggregate statistics for a metric
// Maps to metric_stats table in PostgreSQL
type MetricStats struct {
	MetricName      string  `json:"metric_name"`
	TotalUses       int     `json:"total_uses"`
	FalsePositives  int     `json:"false_positives"`
	TruePositives   int     `json:"true_positives"`
	FPRate          float64 `json:"fp_rate"`
	IsEnabled       bool    `json:"is_enabled"`
	LastUpdated     string  `json:"last_updated"`
}

// NewClient creates a PostgreSQL client from connection parameters
// Security: NEVER hardcode credentials (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - PostgreSQL configuration
func NewClient(ctx context.Context, host string, port int, database, user, password string) (*Client, error) {
	if host == "" || database == "" || user == "" {
		return nil, fmt.Errorf("postgres credentials missing: host=%s, db=%s, user=%s", host, database, user)
	}

	// Build connection string
	// Reference: DEVELOPMENT_WORKFLOW.md §3.3 - Never log passwords
	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		host, port, database, user, password,
	)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Verify connectivity (fail fast on startup)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to postgres at %s:%d: %w", host, port, err)
	}

	logger := slog.Default().With("component", "postgres")
	logger.Info("postgres client connected", "host", host, "port", port, "database", database)

	return &Client{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the PostgreSQL connection pool
func (c *Client) Close() {
	c.pool.Close()
	c.logger.Info("postgres client closed")
}

// HealthCheck verifies PostgreSQL connectivity
// Used by API health endpoint
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}

// RecordMetricUse records a metric calculation result
// Used to track metric usage and enable user feedback
// Reference: risk_assessment_methodology.md §5.2 - Validation tracking
func (c *Client) RecordMetricUse(ctx context.Context, metricName, filePath string, metricValue interface{}) (int, error) {
	// Marshal metric value to JSON
	valueJSON, err := json.Marshal(metricValue)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metric value: %w", err)
	}

	// Insert into metric_validations table
	query := `
		INSERT INTO metric_validations (metric_name, file_path, metric_value)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err = c.pool.QueryRow(ctx, query, metricName, filePath, valueJSON).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to record metric use: %w", err)
	}

	// Update metric stats
	if err := c.incrementMetricUse(ctx, metricName); err != nil {
		c.logger.Warn("failed to update metric stats", "metric", metricName, "error", err)
	}

	c.logger.Debug("metric use recorded", "metric", metricName, "file", filePath, "id", id)
	return id, nil
}

// SubmitFeedback records user feedback on a metric result
// Reference: risk_assessment_methodology.md §5.2 - User validation
func (c *Client) SubmitFeedback(ctx context.Context, validationID int, feedback, reason string) error {
	// Validate feedback value
	if feedback != "true_positive" && feedback != "false_positive" {
		return fmt.Errorf("invalid feedback: must be 'true_positive' or 'false_positive', got '%s'", feedback)
	}

	// Get metric name before updating (needed for stats)
	var metricName string
	err := c.pool.QueryRow(ctx, "SELECT metric_name FROM metric_validations WHERE id = $1", validationID).Scan(&metricName)
	if err != nil {
		return fmt.Errorf("validation ID %d not found: %w", validationID, err)
	}

	// Update validation record
	query := `
		UPDATE metric_validations
		SET user_feedback = $1, feedback_reason = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err = c.pool.Exec(ctx, query, feedback, reason, validationID)
	if err != nil {
		return fmt.Errorf("failed to submit feedback: %w", err)
	}

	// Update metric stats
	if feedback == "false_positive" {
		if err := c.incrementFalsePositives(ctx, metricName); err != nil {
			c.logger.Warn("failed to update FP stats", "metric", metricName, "error", err)
		}
	} else {
		if err := c.incrementTruePositives(ctx, metricName); err != nil {
			c.logger.Warn("failed to update TP stats", "metric", metricName, "error", err)
		}
	}

	c.logger.Info("feedback submitted", "validation_id", validationID, "feedback", feedback, "metric", metricName)
	return nil
}

// GetMetricStats retrieves aggregate statistics for a metric
// Reference: spec.md §6.2 constraint C-10 - FP rate threshold
func (c *Client) GetMetricStats(ctx context.Context, metricName string) (*MetricStats, error) {
	query := `
		SELECT metric_name, total_uses, false_positives, true_positives, fp_rate, is_enabled, last_updated
		FROM metric_stats
		WHERE metric_name = $1
	`

	var stats MetricStats
	err := c.pool.QueryRow(ctx, query, metricName).Scan(
		&stats.MetricName,
		&stats.TotalUses,
		&stats.FalsePositives,
		&stats.TruePositives,
		&stats.FPRate,
		&stats.IsEnabled,
		&stats.LastUpdated,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric stats for %s: %w", metricName, err)
	}

	return &stats, nil
}

// IsMetricEnabled checks if a metric is enabled based on FP rate
// Reference: spec.md §6.2 constraint C-10 - Disable if FP rate > threshold
func (c *Client) IsMetricEnabled(ctx context.Context, metricName string) (bool, error) {
	var isEnabled bool
	err := c.pool.QueryRow(ctx, "SELECT is_enabled FROM metric_stats WHERE metric_name = $1", metricName).Scan(&isEnabled)
	if err != nil {
		return false, fmt.Errorf("failed to check metric enabled status: %w", err)
	}
	return isEnabled, nil
}

// incrementMetricUse increments total_uses counter for a metric
func (c *Client) incrementMetricUse(ctx context.Context, metricName string) error {
	query := `
		UPDATE metric_stats
		SET total_uses = total_uses + 1, last_updated = NOW()
		WHERE metric_name = $1
	`
	_, err := c.pool.Exec(ctx, query, metricName)
	return err
}

// incrementFalsePositives increments false_positives counter
func (c *Client) incrementFalsePositives(ctx context.Context, metricName string) error {
	query := `
		UPDATE metric_stats
		SET false_positives = false_positives + 1, last_updated = NOW()
		WHERE metric_name = $1
	`
	_, err := c.pool.Exec(ctx, query, metricName)
	return err
}

// incrementTruePositives increments true_positives counter
func (c *Client) incrementTruePositives(ctx context.Context, metricName string) error {
	query := `
		UPDATE metric_stats
		SET true_positives = true_positives + 1, last_updated = NOW()
		WHERE metric_name = $1
	`
	_, err := c.pool.Exec(ctx, query, metricName)
	return err
}
