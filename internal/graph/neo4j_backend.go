package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jBackend implements Backend interface for Neo4j with Cypher queries
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
// 12-factor: Factor 12 - Stateless design (context passed per-request)
type Neo4jBackend struct {
	driver   neo4j.DriverWithContext
	database string // Database name for all queries
}

// QueryWithParams represents a Cypher query with its parameters
type QueryWithParams struct {
	Query  string
	Params map[string]any
}

// NewNeo4jBackend creates a Neo4j backend instance
func NewNeo4jBackend(ctx context.Context, uri, username, password, database string) (*Neo4jBackend, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &Neo4jBackend{
		driver:   driver,
		database: database,
	}, nil
}

// CreateNode creates a single node using idempotent MERGE
// Security: Uses parameterized queries to prevent Cypher injection
// Routing: Write operation - routes to cluster leader in cluster deployments
// Reference: NEO4J_MODERNIZATION_GUIDE.md §Phase 3 & 4
func (n *Neo4jBackend) CreateNode(ctx context.Context, node GraphNode) (string, error) {
	// Build parameterized query using CypherBuilder
	builder := NewCypherBuilder()
	uniqueKey := getUniqueKey(node.Label)
	uniqueValue := node.Properties[uniqueKey]

	cypher, err := builder.BuildMergeNode(node.Label, uniqueKey, uniqueValue, node.Properties)
	if err != nil {
		return "", fmt.Errorf("failed to build node query: %w", err)
	}

	// Use modern ExecuteQuery API (Neo4j v5.8+)
	result, err := neo4j.ExecuteQuery(ctx, n.driver, cypher,
		builder.Params(),
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(n.database))

	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	// Extract node ID from result
	if len(result.Records) > 0 {
		if id, ok := result.Records[0].Get("id"); ok {
			return fmt.Sprintf("%v", id), nil
		}
	}

	return "", nil
}

// CreateNodes creates multiple nodes in batch using optimized UNWIND pattern
// Security: Uses parameterized queries to prevent Cypher injection
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 2
func (n *Neo4jBackend) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
	if len(nodes) == 0 {
		return []string{}, nil
	}

	// Use optimized batch creator with UNWIND pattern
	batchCreator := NewBatchNodeCreator(n.driver, n.database, DefaultBatchConfig())

	// Group nodes by label for efficient batch processing
	nodesByLabel := make(map[string][]GraphNode)
	for _, node := range nodes {
		nodesByLabel[node.Label] = append(nodesByLabel[node.Label], node)
	}

	// Process each node type with appropriate batch handler
	for label, labelNodes := range nodesByLabel {
		var err error
		switch label {
		case "File":
			err = batchCreator.CreateFileNodes(ctx, labelNodes)
		case "Function":
			err = batchCreator.CreateFunctionNodes(ctx, labelNodes)
		case "Class":
			err = batchCreator.CreateClassNodes(ctx, labelNodes)
		case "Commit":
			err = batchCreator.CreateCommitNodes(ctx, labelNodes)
		case "Developer":
			err = batchCreator.CreateDeveloperNodes(ctx, labelNodes)
		case "Issue":
			err = batchCreator.CreateIssueNodes(ctx, labelNodes)
		case "PullRequest":
			// PullRequests use same pattern as Issues
			err = batchCreator.CreateIssueNodes(ctx, labelNodes)
		case "Incident":
			// Incidents use same pattern as Issues
			err = batchCreator.CreateIssueNodes(ctx, labelNodes)
		default:
			// Fallback to original implementation for unknown types
			// Build parameterized queries for each node
			queries := make([]QueryWithParams, len(labelNodes))
			for i, node := range labelNodes {
				builder := NewCypherBuilder()
				uniqueKey := getUniqueKey(node.Label)
				uniqueValue := node.Properties[uniqueKey]

				cypher, buildErr := builder.BuildMergeNode(node.Label, uniqueKey, uniqueValue, node.Properties)
				if buildErr != nil {
					return nil, fmt.Errorf("failed to build node query for node %d: %w", i, buildErr)
				}

				queries[i] = QueryWithParams{
					Query:  cypher,
					Params: builder.Params(),
				}
			}

			// Execute in single transaction
			err = n.ExecuteBatchWithParams(ctx, queries)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create %s nodes: %w", label, err)
		}
	}

	// Return dummy IDs (Neo4j doesn't return IDs in UNWIND batch operations)
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = fmt.Sprintf("%d", i)
	}

	return ids, nil
}

// CreateEdge creates a single edge using idempotent MERGE
// Security: Uses parameterized queries to prevent Cypher injection
// Routing: Write operation - routes to cluster leader in cluster deployments
// Reference: NEO4J_MODERNIZATION_GUIDE.md §Phase 3 & 4
func (n *Neo4jBackend) CreateEdge(ctx context.Context, edge GraphEdge) error {
	// Parse node IDs
	fromLabel, fromID := parseNodeID(edge.From)
	toLabel, toID := parseNodeID(edge.To)

	// Build parameterized query
	builder := NewCypherBuilder()
	fromKey := getUniqueKey(fromLabel)
	toKey := getUniqueKey(toLabel)

	cypher, err := builder.BuildMergeEdge(
		fromLabel, fromKey, fromID,
		toLabel, toKey, toID,
		edge.Label,
		edge.Properties,
	)
	if err != nil {
		return fmt.Errorf("failed to build edge query: %w", err)
	}

	// Use modern ExecuteQuery API
	result, err := neo4j.ExecuteQuery(ctx, n.driver, cypher,
		builder.Params(),
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(n.database))

	if err != nil {
		return fmt.Errorf("failed to create edge %s: from=%s:%s to=%s:%s: %w",
			edge.Label, fromLabel, fromID, toLabel, toID, err)
	}

	// Check if any records were returned (nodes found and edge created)
	if len(result.Records) == 0 {
		return fmt.Errorf("edge creation returned no results (nodes may not exist): %s: from=%s:%s to=%s:%s",
			edge.Label, fromLabel, fromID, toLabel, toID)
	}

	return nil
}

// CreateEdges creates multiple edges in batch using optimized UNWIND pattern
// Security: Uses parameterized queries to prevent Cypher injection
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 2
func (n *Neo4jBackend) CreateEdges(ctx context.Context, edges []GraphEdge) error {
	if len(edges) == 0 {
		return nil
	}

	// Log diagnostic info for first edge (helps debug path issues)
	if len(edges) > 0 {
		fromLabel, fromID := parseNodeID(edges[0].From)
		toLabel, toID := parseNodeID(edges[0].To)
		fmt.Printf("DEBUG: Creating %d edges using UNWIND batch pattern. First edge: %s (%s:%s) -> %s (%s:%s)\n",
			len(edges), edges[0].Label, fromLabel, fromID, edges[0].Label, toLabel, toID)
	}

	// Use optimized batch creator with UNWIND pattern
	batchCreator := NewBatchNodeCreator(n.driver, n.database, DefaultBatchConfig())

	// Create all edges in batches grouped by type
	return batchCreator.CreateEdgesBatch(ctx, edges)
}

// ExecuteBatch executes multiple commands in a single transaction (deprecated)
// Use ExecuteBatchWithParams for parameterized queries
func (n *Neo4jBackend) ExecuteBatch(ctx context.Context, commands []string) error {
	// Convert to QueryWithParams format
	queries := make([]QueryWithParams, len(commands))
	for i, cmd := range commands {
		queries[i] = QueryWithParams{
			Query:  cmd,
			Params: nil,
		}
	}
	return n.ExecuteBatchWithParams(ctx, queries)
}

// ExecuteBatchWithParams executes multiple parameterized queries in a single transaction
// Security: Uses parameterized queries to prevent Cypher injection
func (n *Neo4jBackend) ExecuteBatchWithParams(ctx context.Context, queries []QueryWithParams) error {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i, q := range queries {
			if _, err := tx.Run(ctx, q.Query, q.Params); err != nil {
				return nil, fmt.Errorf("batch command %d failed: %w", i, err)
			}
		}
		return nil, nil
	})

	return err
}

// Query executes a Cypher query and returns results
// Uses modern ExecuteQuery API for better performance
// Routing: Read operation - routes to read replicas in cluster deployments
func (n *Neo4jBackend) Query(ctx context.Context, query string) (interface{}, error) {
	result, err := neo4j.ExecuteQuery(ctx, n.driver, query,
		nil, // No parameters for generic queries
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(n.database),
		neo4j.ExecuteQueryWithReadersRouting()) // Read optimization

	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if len(result.Records) > 0 {
		if count, ok := result.Records[0].Get("count"); ok {
			return count, nil
		}
	}

	return 0, nil
}

// Close closes the Neo4j driver connection
func (n *Neo4jBackend) Close(ctx context.Context) error {
	return n.driver.Close(ctx)
}

// ===================================
// Helper Functions
// ===================================
// Note: Old generateCypherNode, generateCypherEdge, and formatCypherValue functions
// have been removed as they used vulnerable string concatenation.
// All query generation now uses CypherBuilder with parameterized queries.

// parseNodeID extracts label and ID from node reference (e.g., "commit:abc123" -> "Commit", "abc123")
func parseNodeID(nodeID string) (label, id string) {
	parts := strings.SplitN(nodeID, ":", 2)
	if len(parts) == 2 {
		// Capitalize first letter for label
		label = strings.ToUpper(string(parts[0][0])) + parts[0][1:]
		return label, parts[1]
	}

	// If no prefix, assume it's a file path (backwards compatibility)
	// File paths don't have colons at the start but may have them in the path
	if strings.Contains(nodeID, "/") || strings.Contains(nodeID, ".") {
		return "File", nodeID
	}

	return "Unknown", nodeID
}

// getUniqueKey returns the unique identifier field for each node type
func getUniqueKey(label string) string {
	keys := map[string]string{
		"Commit":      "sha",
		"commit":      "sha",
		"Developer":   "email",
		"developer":   "email",
		"Issue":       "number",
		"issue":       "number",
		"PullRequest": "number",
		"pullrequest": "number",
		// Incident nodes (Layer 3)
		"Incident": "id", // For Incidents, id = UUID string
		"incident": "id",
		// Tree-sitter entity types (Layer 1)
		"File":     "file_path", // For Files, use file_path for matching
		"file":     "file_path",
		"Function": "unique_id", // For Functions, unique_id = filepath:name:line
		"function": "unique_id",
		"Class":    "unique_id", // For Classes, unique_id = filepath:name:line
		"class":    "unique_id",
		"Import":   "unique_id", // For Imports, unique_id = filepath:name:line
		"import":   "unique_id",
	}

	if key, ok := keys[label]; ok {
		return key
	}
	return "id"
}
