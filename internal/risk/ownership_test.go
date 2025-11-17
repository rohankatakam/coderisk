package risk

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOwnershipCalculator_Integration runs integration tests against a Neo4j instance
// These tests require a running Neo4j instance with test data
func TestOwnershipCalculator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Connect to test Neo4j instance
	ctx := context.Background()
	driver, err := setupTestNeo4j(t)
	if err != nil {
		t.Skipf("Skipping test: Neo4j not available: %v", err)
		return
	}
	defer driver.Close(ctx)

	calculator := NewOwnershipCalculator(driver, "neo4j", nil)

	// Test 1: Original Author calculation
	t.Run("CalculateOriginalAuthor", func(t *testing.T) {
		// First create some test data
		repoID := int64(999) // Test repo ID
		setupTestData(t, driver, repoID)

		result, err := calculator.CalculateOriginalAuthor(ctx, repoID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.BlocksUpdated >= 0)
		assert.Equal(t, "original_author", result.Phase)

		// Cleanup
		cleanupTestData(t, driver, repoID)
	})

	// Test 2: Last Modifier and Staleness calculation
	t.Run("CalculateLastModifierAndStaleness", func(t *testing.T) {
		repoID := int64(999)
		setupTestData(t, driver, repoID)

		result, err := calculator.CalculateLastModifierAndStaleness(ctx, repoID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.BlocksUpdated >= 0)
		assert.Equal(t, "last_modifier_staleness", result.Phase)

		cleanupTestData(t, driver, repoID)
	})

	// Test 3: Familiarity Map calculation
	t.Run("CalculateFamiliarityMap", func(t *testing.T) {
		repoID := int64(999)
		setupTestData(t, driver, repoID)

		result, err := calculator.CalculateFamiliarityMap(ctx, repoID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.BlocksUpdated >= 0)

		cleanupTestData(t, driver, repoID)
	})

	// Test 4: Verification
	t.Run("VerifyOwnershipProperties", func(t *testing.T) {
		repoID := int64(999)
		setupTestData(t, driver, repoID)

		// Run all calculations first
		_, err := calculator.CalculateOriginalAuthor(ctx, repoID)
		require.NoError(t, err)
		_, err = calculator.CalculateLastModifierAndStaleness(ctx, repoID)
		require.NoError(t, err)
		_, err = calculator.CalculateFamiliarityMap(ctx, repoID)
		require.NoError(t, err)

		// Verify
		incompleteBlocks, err := calculator.VerifyOwnershipProperties(ctx, repoID)
		require.NoError(t, err)
		assert.Equal(t, 0, incompleteBlocks, "All blocks should have ownership properties")

		cleanupTestData(t, driver, repoID)
	})
}

// TestBlockIndexer_Integration tests the complete indexer workflow
func TestBlockIndexer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	driver, err := setupTestNeo4j(t)
	if err != nil {
		t.Skipf("Skipping test: Neo4j not available: %v", err)
		return
	}
	defer driver.Close(ctx)

	indexer := NewBlockIndexer(driver, "neo4j", nil)

	t.Run("IndexOwnership", func(t *testing.T) {
		repoID := int64(999)
		setupTestData(t, driver, repoID)

		result, err := indexer.IndexOwnership(ctx, repoID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, repoID, result.RepoID)
		assert.Equal(t, 3, len(result.PhaseResults), "Should have 3 phases")
		assert.True(t, result.TotalDuration > 0)

		cleanupTestData(t, driver, repoID)
	})

	t.Run("GetIndexingStats", func(t *testing.T) {
		repoID := int64(999)
		setupTestData(t, driver, repoID)

		// Index first
		_, err := indexer.IndexOwnership(ctx, repoID)
		require.NoError(t, err)

		// Get stats
		stats, err := indexer.GetIndexingStats(ctx, repoID)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_blocks")
		assert.Contains(t, stats, "with_original_author")
		assert.Contains(t, stats, "with_last_modifier")
		assert.Contains(t, stats, "with_familiarity_map")

		cleanupTestData(t, driver, repoID)
	})
}

// TestSemanticImportance tests the LLM-based importance classification
func TestSemanticImportance(t *testing.T) {
	// This is a unit test - doesn't require LLM to be configured
	calculator := NewOwnershipCalculator(nil, "", nil)

	t.Run("NoLLM_ReturnsDefault", func(t *testing.T) {
		block := CodeBlock{
			Name:      "processPayment",
			BlockType: "function",
			FilePath:  "src/payment/processor.go",
		}

		importance, err := calculator.CalculateSemanticImportance(context.Background(), block)
		assert.NoError(t, err)
		assert.Equal(t, "P2", importance, "Should return default when LLM not available")
	})
}

// TestEdgeCases tests various edge cases in ownership calculations
func TestEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	driver, err := setupTestNeo4j(t)
	if err != nil {
		t.Skipf("Skipping test: Neo4j not available: %v", err)
		return
	}
	defer driver.Close(ctx)

	calculator := NewOwnershipCalculator(driver, "neo4j", nil)

	t.Run("BlockWithNoModifications", func(t *testing.T) {
		repoID := int64(999)

		// Create a block with only CREATED_BLOCK edge (no modifications)
		session := driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: "neo4j",
		})
		defer session.Close(ctx)

		// Setup: Create developer, commit, and block
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create developer
			_, err := tx.Run(ctx, `
				MERGE (d:Developer {email: "test@example.com", repo_id: $repoID})
			`, map[string]any{"repoID": repoID})
			if err != nil {
				return nil, err
			}

			// Create commit
			_, err = tx.Run(ctx, `
				MERGE (c:Commit {sha: "test123", repo_id: $repoID})
				SET c.committed_at = $timestamp
			`, map[string]any{
				"repoID":    repoID,
				"timestamp": time.Now().Unix(),
			})
			if err != nil {
				return nil, err
			}

			// Create block
			_, err = tx.Run(ctx, `
				MERGE (b:CodeBlock {db_id: 1001, repo_id: $repoID})
				SET b.name = "testFunction",
				    b.block_type = "function",
				    b.file_path = "test.go"
			`, map[string]any{"repoID": repoID})
			if err != nil {
				return nil, err
			}

			// Create CREATED_BLOCK edge
			_, err = tx.Run(ctx, `
				MATCH (c:Commit {sha: "test123", repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: 1001, repo_id: $repoID})
				MERGE (c)-[:CREATED_BLOCK]->(b)
			`, map[string]any{"repoID": repoID})
			if err != nil {
				return nil, err
			}

			// Create AUTHORED edge
			_, err = tx.Run(ctx, `
				MATCH (c:Commit {sha: "test123", repo_id: $repoID})
				MATCH (d:Developer {email: "test@example.com", repo_id: $repoID})
				MERGE (c)<-[:AUTHORED]-(d)
			`, map[string]any{"repoID": repoID})

			return nil, err
		})
		require.NoError(t, err)

		// Calculate last modifier (should handle blocks with no modifications)
		result, err := calculator.CalculateLastModifierAndStaleness(ctx, repoID)
		require.NoError(t, err)
		assert.True(t, result.BlocksUpdated >= 1, "Should handle block with no modifications")

		cleanupTestData(t, driver, repoID)
	})

	t.Run("EmptyRepository", func(t *testing.T) {
		repoID := int64(9999) // Repo with no blocks

		result, err := calculator.CalculateOriginalAuthor(ctx, repoID)
		require.NoError(t, err)
		assert.Equal(t, 0, result.BlocksUpdated, "Empty repo should have 0 updates")
	})
}

// setupTestNeo4j creates a connection to test Neo4j instance
func setupTestNeo4j(t *testing.T) (neo4j.DriverWithContext, error) {
	uri := "bolt://localhost:7688" // Test Neo4j instance
	user := "neo4j"
	password := "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, err
	}

	// Verify connectivity
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, err
	}

	return driver, nil
}

// setupTestData creates test data in Neo4j for testing
func setupTestData(t *testing.T, driver neo4j.DriverWithContext, repoID int64) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create test developer
		_, err := tx.Run(ctx, `
			MERGE (d:Developer {email: "test@example.com", repo_id: $repoID})
		`, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		// Create test commits
		for i := 1; i <= 3; i++ {
			_, err = tx.Run(ctx, `
				MERGE (c:Commit {sha: $sha, repo_id: $repoID})
				SET c.committed_at = $timestamp
			`, map[string]any{
				"sha":       "test" + string(rune(i)),
				"repoID":    repoID,
				"timestamp": time.Now().Add(-time.Duration(i) * time.Hour).Unix(),
			})
			if err != nil {
				return nil, err
			}

			// Create AUTHORED edges
			_, err = tx.Run(ctx, `
				MATCH (c:Commit {sha: $sha, repo_id: $repoID})
				MATCH (d:Developer {email: "test@example.com", repo_id: $repoID})
				MERGE (c)<-[:AUTHORED]-(d)
			`, map[string]any{
				"sha":    "test" + string(rune(i)),
				"repoID": repoID,
			})
			if err != nil {
				return nil, err
			}
		}

		// Create test code blocks
		for i := 1; i <= 2; i++ {
			_, err = tx.Run(ctx, `
				MERGE (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				SET b.name = $name,
				    b.block_type = "function",
				    b.file_path = $filePath
			`, map[string]any{
				"blockID":  1000 + i,
				"repoID":   repoID,
				"name":     "testFunc" + string(rune(i)),
				"filePath": "test" + string(rune(i)) + ".go",
			})
			if err != nil {
				return nil, err
			}

			// Create CREATED_BLOCK edge
			_, err = tx.Run(ctx, `
				MATCH (c:Commit {sha: "test1", repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				MERGE (c)-[:CREATED_BLOCK]->(b)
			`, map[string]any{
				"blockID": 1000 + i,
				"repoID":  repoID,
			})
			if err != nil {
				return nil, err
			}

			// Create MODIFIED_BLOCK edges
			if i == 1 {
				_, err = tx.Run(ctx, `
					MATCH (c:Commit {sha: "test2", repo_id: $repoID})
					MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
					MERGE (c)-[:MODIFIED_BLOCK]->(b)
				`, map[string]any{
					"blockID": 1000 + i,
					"repoID":  repoID,
				})
				if err != nil {
					return nil, err
				}
			}
		}

		return nil, nil
	})

	require.NoError(t, err, "Failed to setup test data")
}

// cleanupTestData removes test data from Neo4j
func cleanupTestData(t *testing.T, driver neo4j.DriverWithContext, repoID int64) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Delete all nodes and relationships for the test repo
		_, err := tx.Run(ctx, `
			MATCH (n {repo_id: $repoID})
			DETACH DELETE n
		`, map[string]any{"repoID": repoID})
		return nil, err
	})

	if err != nil {
		t.Logf("Warning: Failed to cleanup test data: %v", err)
	}
}
