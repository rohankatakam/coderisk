package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// RoutingMode defines read/write routing for cluster deployments
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 6
//
// In Neo4j clusters (Causal Cluster or Aura):
// - READ queries route to read replicas (reduces load on leader)
// - WRITE queries route to leader (only leader can write)
//
// For local single-node deployments, routing has no effect.
type RoutingMode string

const (
	// RoutingRead routes to read replicas (for queries)
	RoutingRead RoutingMode = "read"

	// RoutingWrite routes to cluster leader (for writes)
	RoutingWrite RoutingMode = "write"
)

// ExecuteWithRouting executes a query with explicit routing
// This is a wrapper around ExecuteQuery that adds routing hints
//
// Example usage:
//   // Read query (routes to replicas)
//   result, err := ExecuteWithRouting(ctx, driver, query, params, RoutingRead, "neo4j")
//
//   // Write query (routes to leader)
//   result, err := ExecuteWithRouting(ctx, driver, query, params, RoutingWrite, "neo4j")
func ExecuteWithRouting(
	ctx context.Context,
	driver neo4j.DriverWithContext,
	query string,
	params map[string]any,
	mode RoutingMode,
	database string,
) (*neo4j.EagerResult, error) {
	// Build options list
	options := []neo4j.ExecuteQueryConfigurationOption{
		neo4j.ExecuteQueryWithDatabase(database),
	}

	// Add routing hint based on mode
	switch mode {
	case RoutingRead:
		// Route to read replicas (reduces load on leader)
		options = append(options, neo4j.ExecuteQueryWithReadersRouting())
	case RoutingWrite:
		// Route to leader (only leader can accept writes)
		options = append(options, neo4j.ExecuteQueryWithWritersRouting())
	}

	return neo4j.ExecuteQuery(ctx, driver, query, params,
		neo4j.EagerResultTransformer,
		options...)
}

// SessionWithRouting creates a session with explicit routing
// Use this when you need more control over transactions
//
// Example usage:
//   session := SessionWithRouting(ctx, driver, RoutingRead, "neo4j")
//   defer session.Close(ctx)
//
//   result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
//     // ... run queries
//   })
func SessionWithRouting(
	ctx context.Context,
	driver neo4j.DriverWithContext,
	mode RoutingMode,
	database string,
) neo4j.SessionWithContext {
	config := neo4j.SessionConfig{
		DatabaseName: database,
	}

	// Set access mode based on routing
	switch mode {
	case RoutingRead:
		config.AccessMode = neo4j.AccessModeRead
	case RoutingWrite:
		config.AccessMode = neo4j.AccessModeWrite
	}

	return driver.NewSession(ctx, config)
}

// RoutingStrategy determines routing based on operation type
// This helps automatically route queries to the right target
type RoutingStrategy struct {
	// Default routing for unknown operations
	DefaultMode RoutingMode
}

// NewRoutingStrategy creates a strategy with sensible defaults
func NewRoutingStrategy() *RoutingStrategy {
	return &RoutingStrategy{
		DefaultMode: RoutingRead, // Default to read replicas
	}
}

// GetRoutingForOperation determines routing based on operation name
// Returns RoutingRead for queries, RoutingWrite for ingestion
func (rs *RoutingStrategy) GetRoutingForOperation(operation string) RoutingMode {
	// Map operation types to routing modes
	writeOperations := map[string]bool{
		"layer1_ingestion": true,
		"layer2_ingestion": true,
		"layer3_ingestion": true,
		"batch_create":     true,
		"index_creation":   true,
	}

	if writeOperations[operation] {
		return RoutingWrite
	}

	return RoutingRead
}

// ClusterInfo provides information about Neo4j cluster topology
// Note: This requires cluster discovery queries (not always available in local mode)
type ClusterInfo struct {
	IsCluster      bool
	LeaderCount    int
	FollowerCount  int
	ReadReplicaCount int
}

// GetClusterInfo queries Neo4j for cluster topology
// Returns single-node info if not in cluster mode
//
// Note: This query may fail in local mode or if user lacks permissions
func GetClusterInfo(ctx context.Context, driver neo4j.DriverWithContext, database string) (*ClusterInfo, error) {
	// Query cluster topology (works in Enterprise/Aura)
	query := `
		CALL dbms.cluster.overview()
		YIELD role
		RETURN role, count(*) as count
	`

	result, err := neo4j.ExecuteQuery(ctx, driver, query, nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(database))

	if err != nil {
		// Not a cluster or permissions issue - assume single node
		return &ClusterInfo{
			IsCluster:        false,
			LeaderCount:      1,
			FollowerCount:    0,
			ReadReplicaCount: 0,
		}, nil
	}

	info := &ClusterInfo{IsCluster: true}

	for _, record := range result.Records {
		role, _ := record.Get("role")
		count, _ := record.Get("count")

		roleStr, _ := role.(string)
		countInt, _ := count.(int64)

		switch roleStr {
		case "LEADER":
			info.LeaderCount = int(countInt)
		case "FOLLOWER":
			info.FollowerCount = int(countInt)
		case "READ_REPLICA":
			info.ReadReplicaCount = int(countInt)
		}
	}

	return info, nil
}

// RoutingHealthCheck verifies routing is working correctly
// Tests both read and write routing to ensure cluster is healthy
func RoutingHealthCheck(ctx context.Context, driver neo4j.DriverWithContext, database string) error {
	// Test read routing
	readQuery := "RETURN 1 as test"
	_, err := ExecuteWithRouting(ctx, driver, readQuery, nil, RoutingRead, database)
	if err != nil {
		return err
	}

	// Test write routing (create temporary node and delete it)
	writeQuery := "CREATE (n:HealthCheck {timestamp: timestamp()}) DELETE n RETURN count(n)"
	_, err = ExecuteWithRouting(ctx, driver, writeQuery, nil, RoutingWrite, database)
	if err != nil {
		return err
	}

	return nil
}

// IsClusterDeployment detects if running in cluster mode
// Returns true if Neo4j Aura or Enterprise cluster detected
func IsClusterDeployment(ctx context.Context, driver neo4j.DriverWithContext, database string) bool {
	info, err := GetClusterInfo(ctx, driver, database)
	if err != nil {
		return false
	}
	return info.IsCluster
}
