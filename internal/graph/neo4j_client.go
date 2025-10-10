package graph

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps Neo4j driver with error handling and query helpers
// Reference: graph_ontology.md - Three-layer ontology
type Client struct {
	driver neo4j.DriverWithContext
	logger *slog.Logger
}

// NewClient creates a Neo4j client from environment variables
// Security: NEVER hardcode credentials (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - Neo4j configuration
func NewClient(ctx context.Context, uri, user, password string) (*Client, error) {
	if uri == "" || user == "" || password == "" {
		return nil, fmt.Errorf("neo4j credentials missing: uri=%s, user=%s", uri, user)
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	// Verify connectivity (fail fast on startup)
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to neo4j at %s: %w", uri, err)
	}

	logger := slog.Default().With("component", "neo4j")
	logger.Info("neo4j client connected", "uri", uri, "user", user)

	return &Client{
		driver: driver,
		logger: logger,
	}, nil
}

// Close closes the Neo4j driver connection
func (c *Client) Close(ctx context.Context) error {
	if err := c.driver.Close(ctx); err != nil {
		return fmt.Errorf("failed to close neo4j driver: %w", err)
	}
	c.logger.Info("neo4j client closed")
	return nil
}

// HealthCheck verifies Neo4j connectivity
// Used by API health endpoint
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("neo4j health check failed: %w", err)
	}
	return nil
}

// QueryCoupling calculates structural coupling metric for a file
// Reference: risk_assessment_methodology.md §2.1 - Coupling metric
// Returns: count of direct dependencies (IMPORTS, CALLS edges)
func (c *Client) QueryCoupling(ctx context.Context, filePath string) (int, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Cypher query with 1-hop limit (prevents graph explosion)
	// Reference: spec.md §6.2 constraint C-6 - MAX_HOPS limit
	query := `
		MATCH (f:File {path: $path})-[:IMPORTS|CALLS]-(dep)
		RETURN count(DISTINCT dep) as count
	`

	result, err := session.Run(ctx, query, map[string]any{"path": filePath})
	if err != nil {
		return 0, fmt.Errorf("coupling query failed for %s: %w", filePath, err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		// File not found in graph - return 0 (not an error)
		if result.Next(ctx) == false {
			c.logger.Debug("file not found in graph", "path", filePath)
			return 0, nil
		}
		return 0, fmt.Errorf("no coupling result for %s: %w", filePath, err)
	}

	count, ok := record.Get("count")
	if !ok {
		return 0, fmt.Errorf("coupling query returned no count for %s", filePath)
	}

	countInt := int(count.(int64))
	c.logger.Debug("coupling calculated", "file", filePath, "count", countInt)

	return countInt, nil
}

// QueryCoChange calculates co-change frequency for a file
// Reference: risk_assessment_methodology.md §2.2 - Co-change metric
// Returns: count of files that changed together in last 90 days
func (c *Client) QueryCoChange(ctx context.Context, filePath string) (int, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Query CO_CHANGED edges (created during graph construction)
	// Reference: graph_ontology.md §3.2.4 - CO_CHANGED relationship
	query := `
		MATCH (f:File {file_path: $file_path})-[r:CO_CHANGED]-(other)
		WHERE r.window_days = 90
		RETURN count(DISTINCT other) as count
	`

	result, err := session.Run(ctx, query, map[string]any{"file_path": filePath})
	if err != nil {
		return 0, fmt.Errorf("co-change query failed for %s: %w", filePath, err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		if result.Next(ctx) == false {
			c.logger.Debug("no co-change data for file", "path", filePath)
			return 0, nil
		}
		return 0, fmt.Errorf("no co-change result for %s: %w", filePath, err)
	}

	count, ok := record.Get("count")
	if !ok {
		return 0, fmt.Errorf("co-change query returned no count for %s", filePath)
	}

	countInt := int(count.(int64))
	c.logger.Debug("co-change calculated", "file", filePath, "count", countInt)

	return countInt, nil
}

// ExecuteQuery executes a generic Cypher query with parameters
// Used by advanced metrics and custom queries
// Reference: DEVELOPMENT_WORKFLOW.md §3.1 - Input validation
func (c *Client) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	var records []map[string]any
	for result.Next(ctx) {
		records = append(records, result.Record().AsMap())
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("query iteration failed: %w", err)
	}

	c.logger.Debug("query executed", "record_count", len(records))
	return records, nil
}
