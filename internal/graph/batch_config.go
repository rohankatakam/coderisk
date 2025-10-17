package graph

// BatchConfig defines optimal batch sizes for different node types
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 2
//
// These batch sizes are based on Neo4j best practices:
// - Small batches (100-200): Complex nodes with many properties
// - Medium batches (500-1000): Simple nodes with few properties
// - Large batches (1000-5000): Edges with minimal properties
type BatchConfig struct {
	// Layer 1: Structure nodes
	FileBatchSize     int // Optimal: 500-1000
	FunctionBatchSize int // Optimal: 1000-2000
	ClassBatchSize    int // Optimal: 500-1000

	// Layer 2: Temporal nodes
	CommitBatchSize    int // Optimal: 200-500
	DeveloperBatchSize int // Optimal: 100-200

	// Layer 3: Incident nodes
	IncidentBatchSize int // Optimal: 50-100

	// Edges
	EdgeBatchSize int // Optimal: 1000-5000
}

// DefaultBatchConfig returns optimized batch sizes for medium repos (~5K files)
// Based on Neo4j best practices and CodeRisk testing
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		FileBatchSize:      1000,
		FunctionBatchSize:  2000,
		ClassBatchSize:     1000,
		CommitBatchSize:    500,
		DeveloperBatchSize: 200,
		IncidentBatchSize:  100,
		EdgeBatchSize:      5000,
	}
}

// SmallRepoBatchConfig for repos < 500 files
// Uses smaller batches to reduce memory pressure
func SmallRepoBatchConfig() BatchConfig {
	return BatchConfig{
		FileBatchSize:      200,
		FunctionBatchSize:  500,
		ClassBatchSize:     200,
		CommitBatchSize:    100,
		DeveloperBatchSize: 50,
		IncidentBatchSize:  50,
		EdgeBatchSize:      1000,
	}
}

// LargeRepoBatchConfig for repos > 10K files
// Uses larger batches for maximum throughput
func LargeRepoBatchConfig() BatchConfig {
	return BatchConfig{
		FileBatchSize:      2000,
		FunctionBatchSize:  5000,
		ClassBatchSize:     2000,
		CommitBatchSize:    1000,
		DeveloperBatchSize: 500,
		IncidentBatchSize:  200,
		EdgeBatchSize:      10000,
	}
}

// GetBatchSizeForLabel returns the appropriate batch size for a given node label
func (bc BatchConfig) GetBatchSizeForLabel(label string) int {
	switch label {
	case "File":
		return bc.FileBatchSize
	case "Function":
		return bc.FunctionBatchSize
	case "Class":
		return bc.ClassBatchSize
	case "Commit":
		return bc.CommitBatchSize
	case "Developer":
		return bc.DeveloperBatchSize
	case "Incident":
		return bc.IncidentBatchSize
	case "Issue":
		return bc.IncidentBatchSize // Same as incidents
	case "PullRequest":
		return bc.CommitBatchSize // Same as commits
	default:
		return 500 // Default for unknown types
	}
}
