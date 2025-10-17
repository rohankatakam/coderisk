package graph

import (
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// TransactionConfig defines timeout and metadata for transactions
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 5
//
// Transaction metadata is logged by Neo4j and visible in query.log
// This helps with debugging slow queries and categorizing operations.
type TransactionConfig struct {
	Timeout  time.Duration
	Metadata map[string]any
}

// DefaultTransactionConfigs returns recommended configs per operation type
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 5
func DefaultTransactionConfigs() map[string]TransactionConfig {
	return map[string]TransactionConfig{
		// Layer 1: Structure ingestion (tree-sitter parsing)
		"layer1_ingestion": {
			Timeout: 5 * time.Minute, // Tree-sitter parsing can be slow for large files
			Metadata: map[string]any{
				"operation": "layer1_ingestion",
				"layer":     "structure",
				"type":      "write",
			},
		},

		// Layer 2: Temporal ingestion (git history)
		"layer2_ingestion": {
			Timeout: 10 * time.Minute, // Git history ingestion can take time
			Metadata: map[string]any{
				"operation": "layer2_ingestion",
				"layer":     "temporal",
				"type":      "write",
			},
		},

		// Layer 3: Incident linking
		"layer3_ingestion": {
			Timeout: 2 * time.Minute, // Incident linking should be fast
			Metadata: map[string]any{
				"operation": "layer3_ingestion",
				"layer":     "incidents",
				"type":      "write",
			},
		},

		// Tier 1 metrics (fast queries)
		"metric_query": {
			Timeout: 30 * time.Second, // Metric queries should be fast
			Metadata: map[string]any{
				"operation": "metric_query",
				"tier":      "tier1",
				"type":      "read",
			},
		},

		// Tier 2 metrics (ownership churn - may be slower)
		"ownership_query": {
			Timeout: 60 * time.Second, // Ownership queries can be complex
			Metadata: map[string]any{
				"operation": "ownership_query",
				"tier":      "tier2",
				"type":      "read",
			},
		},

		// Batch operations (node/edge creation)
		"batch_create": {
			Timeout: 3 * time.Minute, // Batch operations need more time
			Metadata: map[string]any{
				"operation": "batch_create",
				"type":      "write",
			},
		},

		// Index creation
		"index_creation": {
			Timeout: 5 * time.Minute, // Index creation can be slow on large graphs
			Metadata: map[string]any{
				"operation": "index_creation",
				"type":      "schema",
			},
		},

		// Health checks
		"health_check": {
			Timeout: 5 * time.Second, // Health checks must be fast
			Metadata: map[string]any{
				"operation": "health_check",
				"type":      "read",
			},
		},
	}
}

// AsNeo4jConfig converts to Neo4j transaction config functions
// Use with ExecuteQuery or ExecuteRead/ExecuteWrite
func (tc TransactionConfig) AsNeo4jConfig() []func(*neo4j.TransactionConfig) {
	configs := []func(*neo4j.TransactionConfig){}

	// Add timeout if specified
	if tc.Timeout > 0 {
		configs = append(configs, neo4j.WithTxTimeout(tc.Timeout))
	}

	// Add metadata if specified
	if len(tc.Metadata) > 0 {
		configs = append(configs, neo4j.WithTxMetadata(tc.Metadata))
	}

	return configs
}

// AsExecuteQueryOptions converts to ExecuteQuery option functions
// Note: ExecuteQuery API doesn't support per-query timeout/metadata directly
// Use context.WithTimeout() for timeout control instead
// This method is kept for API compatibility but returns empty slice
func (tc TransactionConfig) AsExecuteQueryOptions() []func(*neo4j.ExecuteQueryConfiguration) {
	// ExecuteQuery doesn't support WithTxTimeout or WithTxMetadata options
	// Timeouts should be implemented via context.WithTimeout()
	return []func(*neo4j.ExecuteQueryConfiguration){}
}

// GetConfigForOperation retrieves the appropriate transaction config
// Returns default config if operation not found
func GetConfigForOperation(operation string) TransactionConfig {
	configs := DefaultTransactionConfigs()
	if config, ok := configs[operation]; ok {
		return config
	}

	// Default fallback config
	return TransactionConfig{
		Timeout: 60 * time.Second,
		Metadata: map[string]any{
			"operation": operation,
			"type":      "unknown",
		},
	}
}

// WithCustomMetadata creates a config with custom metadata
// Useful for adding context-specific information
func (tc TransactionConfig) WithCustomMetadata(key string, value any) TransactionConfig {
	// Create a copy of the config
	newConfig := TransactionConfig{
		Timeout:  tc.Timeout,
		Metadata: make(map[string]any),
	}

	// Copy existing metadata
	for k, v := range tc.Metadata {
		newConfig.Metadata[k] = v
	}

	// Add custom metadata
	newConfig.Metadata[key] = value

	return newConfig
}

// WithTimeout creates a config with a custom timeout
// Overrides the default timeout for the operation
func (tc TransactionConfig) WithTimeout(timeout time.Duration) TransactionConfig {
	return TransactionConfig{
		Timeout:  timeout,
		Metadata: tc.Metadata,
	}
}
