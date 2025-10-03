package graph

// Backend defines the interface for graph database operations
// Supports both Neo4j (Cypher) and Neptune (Gremlin)
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
type Backend interface {
	// CreateNode creates a single node in the graph
	CreateNode(node GraphNode) (string, error)

	// CreateNodes creates multiple nodes in batch
	CreateNodes(nodes []GraphNode) ([]string, error)

	// CreateEdge creates a single edge in the graph
	CreateEdge(edge GraphEdge) error

	// CreateEdges creates multiple edges in batch
	CreateEdges(edges []GraphEdge) error

	// ExecuteBatch executes multiple commands in a single transaction
	ExecuteBatch(commands []string) error

	// Query executes a query and returns results
	Query(query string) (interface{}, error)

	// Close closes the backend connection
	Close() error
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
