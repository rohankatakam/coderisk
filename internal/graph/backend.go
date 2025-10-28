package graph

import "context"

// Backend defines the interface for graph database operations
// Supports both Neo4j (Cypher) and Neptune (Gremlin)
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
// 12-factor: Factor 12 - Stateless design (context passed per-request, not stored)
type Backend interface {
	// CreateNode creates a single node in the graph
	CreateNode(ctx context.Context, node GraphNode) (string, error)

	// CreateNodes creates multiple nodes in batch
	CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error)

	// CreateEdge creates a single edge in the graph
	CreateEdge(ctx context.Context, edge GraphEdge) error

	// CreateEdges creates multiple edges in batch
	CreateEdges(ctx context.Context, edges []GraphEdge) error

	// ExecuteBatch executes multiple commands in a single transaction
	ExecuteBatch(ctx context.Context, commands []string) error

	// Query executes a query and returns results (simple, no parameters)
	Query(ctx context.Context, query string) (interface{}, error)

	// QueryWithParams executes a parameterized query and returns results
	QueryWithParams(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)

	// Close closes the backend connection
	Close(ctx context.Context) error
}

// GraphNode represents a node in the graph
type GraphNode struct {
	Label      string                 // Node type: "Commit", "Developer", "Issue", etc.
	ID         string                 // Unique identifier for the node
	Properties map[string]interface{} // Node properties
}

// GraphEdge represents an edge in the graph
type GraphEdge struct {
	Label      string                 // Edge type: "AUTHORED", "MODIFIES", "FIXES", etc.
	From       string                 // Source node ID
	To         string                 // Target node ID
	Properties map[string]interface{} // Edge properties
}
