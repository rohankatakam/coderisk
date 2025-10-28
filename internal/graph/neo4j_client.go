package graph

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps Neo4j driver with error handling and query helpers
// Reference: graph_ontology.md - Three-layer ontology
type Client struct {
	driver   neo4j.DriverWithContext
	logger   *slog.Logger
	database string // Database name for queries
}

// NewClient creates a Neo4j client from environment variables
// Security: NEVER hardcode credentials (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - Neo4j configuration
func NewClient(ctx context.Context, uri, user, password string) (*Client, error) {
	return NewClientWithDatabase(ctx, uri, user, password, "neo4j")
}

// NewClientWithDatabase creates a Neo4j client with a specific database
func NewClientWithDatabase(ctx context.Context, uri, user, password, database string) (*Client, error) {
	if uri == "" || user == "" || password == "" {
		return nil, fmt.Errorf("neo4j credentials missing: uri=%s, user=%s", uri, user)
	}

	// Configure connection pool for optimal performance
	// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 4
	driver, err := neo4j.NewDriverWithContext(uri,
		neo4j.BasicAuth(user, password, ""),
		func(config *neo4j.Config) {
			// Connection pool settings
			config.MaxConnectionPoolSize = 50                         // Default: 100, reduced for medium workloads
			config.ConnectionAcquisitionTimeout = 60 * time.Second    // Default: 60s, time to wait for connection
			config.MaxConnectionLifetime = 3600 * time.Second         // Default: 1h, recycle connections hourly
			config.ConnectionLivenessCheckTimeout = 5 * time.Second   // Default: 5s, liveness check timeout

			// Connection timeout settings
			config.SocketConnectTimeout = 5 * time.Second             // Default: 5s, initial connection timeout
			config.SocketKeepalive = true                             // Enable TCP keepalive

			// TLS configuration (commented out for local development)
			// config.Encrypted = true  // Enable for neo4j+s:// URIs in production
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	// Verify connectivity (fail fast on startup)
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to neo4j at %s: %w", uri, err)
	}

	logger := slog.Default().With("component", "neo4j")
	logger.Info("neo4j client connected",
		"uri", uri,
		"user", user,
		"database", database,
		"max_pool_size", 50)

	return &Client{
		driver:   driver,
		logger:   logger,
		database: database,
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
// Uses modern ExecuteQuery API (Neo4j v5.8+) - NEO4J_MODERNIZATION_GUIDE.md §Phase 3
func (c *Client) QueryCoupling(ctx context.Context, filePath string) (int, error) {
	// Cypher query with 1-hop limit (prevents graph explosion)
	// Reference: spec.md §6.2 constraint C-6 - MAX_HOPS limit
	query := `
		MATCH (f:File {path: $path})-[:IMPORTS|CALLS]-(dep)
		RETURN count(DISTINCT dep) as count
	`

	// Get transaction config for metric queries
	// Note: Use context timeout for timeout control (ExecuteQuery doesn't support per-query config)
	queryCtx := ctx
	txConfig := GetConfigForOperation("metric_query")
	if txConfig.Timeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, txConfig.Timeout)
		defer cancel()
	}

	// Use modern ExecuteQuery API with read routing optimization
	result, err := neo4j.ExecuteQuery(queryCtx, c.driver, query,
		map[string]any{"path": filePath},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return 0, fmt.Errorf("coupling query failed for %s: %w", filePath, err)
	}

	// Handle empty results (file not found in graph)
	if len(result.Records) == 0 {
		c.logger.Debug("file not found in graph", "path", filePath)
		return 0, nil
	}

	// Safe type assertion
	count, ok := result.Records[0].Get("count")
	if !ok {
		return 0, fmt.Errorf("coupling query returned no count for %s", filePath)
	}

	countInt, ok := count.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for count: %T (expected int64)", count)
	}

	c.logger.Debug("coupling calculated", "file", filePath, "count", int(countInt))
	return int(countInt), nil
}

// QueryCoChange calculates co-change frequency for a file
// Reference: risk_assessment_methodology.md §2.2 - Co-change metric
// Returns: count of files that changed together in last 90 days
// Uses modern ExecuteQuery API (Neo4j v5.8+) - NEO4J_MODERNIZATION_GUIDE.md §Phase 3
// NOTE: Computes co-change dynamically, filters to default branch (see simplified_graph_schema.md)
func (c *Client) QueryCoChange(ctx context.Context, filePath string) (int, error) {
	// Compute co-change dynamically from commits (default branch only)
	// Reference: simplified_graph_schema.md - Co-change computed at query time
	query := `
		MATCH (f:File {file_path: $file_path})<-[:MODIFIES]-(c1:Commit)
		MATCH (c1)-[:ON_BRANCH]->(b:Branch {is_default: true})
		WHERE c1.author_date > timestamp() - duration({days: 90}) * 1000
		WITH f, collect(c1) as commits
		UNWIND commits as c
		MATCH (c)-[:MODIFIES]->(other:File)
		WHERE other.file_path <> $file_path
		WITH other, count(c) as co_changes, size(commits) as total
		WITH other, co_changes, toFloat(co_changes)/toFloat(total) as frequency
		WHERE frequency > 0.3
		RETURN count(DISTINCT other) as count
	`

	// Get transaction config for metric queries
	// Note: Use context timeout for timeout control (ExecuteQuery doesn't support per-query config)
	queryCtx := ctx
	txConfig := GetConfigForOperation("metric_query")
	if txConfig.Timeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, txConfig.Timeout)
		defer cancel()
	}

	// Use modern ExecuteQuery API with read routing optimization
	result, err := neo4j.ExecuteQuery(queryCtx, c.driver, query,
		map[string]any{"file_path": filePath},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return 0, fmt.Errorf("co-change query failed for %s: %w", filePath, err)
	}

	// Handle empty results (no co-change data for file)
	if len(result.Records) == 0 {
		c.logger.Debug("no co-change data for file", "path", filePath)
		return 0, nil
	}

	// Safe type assertion
	count, ok := result.Records[0].Get("count")
	if !ok {
		return 0, fmt.Errorf("co-change query returned no count for %s", filePath)
	}

	countInt, ok := count.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for count: %T (expected int64)", count)
	}

	c.logger.Debug("co-change calculated", "file", filePath, "count", int(countInt))
	return int(countInt), nil
}

// ExecuteQuery executes a generic Cypher query with parameters
// Used by advanced metrics and custom queries
// Reference: DEVELOPMENT_WORKFLOW.md §3.1 - Input validation
// Uses modern ExecuteQuery API (Neo4j v5.8+) - NEO4J_MODERNIZATION_GUIDE.md §Phase 3
func (c *Client) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	// Get transaction config for generic queries (use metric_query as default)
	// Note: Use context timeout for timeout control (ExecuteQuery doesn't support per-query config)
	queryCtx := ctx
	txConfig := GetConfigForOperation("metric_query")
	if txConfig.Timeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, txConfig.Timeout)
		defer cancel()
	}

	// Use modern ExecuteQuery API with read routing optimization
	result, err := neo4j.ExecuteQuery(queryCtx, c.driver, query, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Convert records to map format
	var records []map[string]any
	for _, record := range result.Records {
		records = append(records, record.AsMap())
	}

	c.logger.Debug("query executed", "record_count", len(records))
	return records, nil
}

// Driver returns the underlying Neo4j driver
// Used for advanced operations like lazy loading
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 3
func (c *Client) Driver() neo4j.DriverWithContext {
	return c.driver
}

// Database returns the configured database name
// Used for advanced operations like lazy loading
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 3
func (c *Client) Database() string {
	return c.database
}

// QueryCouplingMultiple queries coupling across multiple file paths (handles renames)
func (c *Client) QueryCouplingMultiple(ctx context.Context, filePaths []string) (int, error) {
	if len(filePaths) == 0 {
		return 0, nil
	}

	// Cypher query with IN clause to match ANY of the historical paths
	query := `
		MATCH (f:File)-[:IMPORTS|CALLS]-(dep)
		WHERE f.path IN $paths
		RETURN count(DISTINCT dep) as count
	`

	queryCtx := ctx
	txConfig := GetConfigForOperation("metric_query")
	if txConfig.Timeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, txConfig.Timeout)
		defer cancel()
	}

	result, err := neo4j.ExecuteQuery(queryCtx, c.driver, query,
		map[string]any{"paths": filePaths},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return 0, fmt.Errorf("coupling query failed for %v: %w", filePaths, err)
	}

	if len(result.Records) == 0 {
		c.logger.Debug("files not found in graph", "paths", filePaths)
		return 0, nil
	}

	count, ok := result.Records[0].Get("count")
	if !ok {
		return 0, fmt.Errorf("coupling query returned no count for %v", filePaths)
	}

	countInt, ok := count.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected count type: %T", count)
	}

	c.logger.Debug("coupling calculated", "files", filePaths, "count", int(countInt))
	return int(countInt), nil
}
