package risk

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection and ensures tables exist
func setupTestDB(t *testing.T) (*sql.DB, int64) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)

	// Verify connection
	err = db.Ping()
	require.NoError(t, err)

	// Create a test repo
	var repoID int64
	err = db.QueryRow(`
		INSERT INTO github_repositories (owner, name, full_name, url, default_branch, created_at, updated_at)
		VALUES ('test', 'coupling-test', 'test/coupling-test', 'https://github.com/test/coupling-test', 'main', NOW(), NOW())
		ON CONFLICT (owner, name) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`).Scan(&repoID)
	require.NoError(t, err)

	return db, repoID
}

// cleanupTestData removes test data from the database
func cleanupTestData(t *testing.T, db *sql.DB, repoID int64) {
	// Clean up in reverse order of foreign key dependencies
	_, err := db.Exec("DELETE FROM code_block_co_changes WHERE repo_id = $1", repoID)
	require.NoError(t, err)

	_, err = db.Exec("DELETE FROM code_block_modifications WHERE repo_id = $1", repoID)
	require.NoError(t, err)

	_, err = db.Exec("DELETE FROM code_blocks WHERE repo_id = $1", repoID)
	require.NoError(t, err)

	_, err = db.Exec("DELETE FROM github_repositories WHERE id = $1", repoID)
	require.NoError(t, err)
}

// createTestCodeBlock creates a test code block
func createTestCodeBlock(t *testing.T, db *sql.DB, repoID int64, filePath, name, blockType string) int64 {
	var blockID int64
	err := db.QueryRow(`
		INSERT INTO code_blocks (repo_id, file_path, block_name, block_type, language, first_seen_commit_sha, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'go', 'test-sha', NOW(), NOW())
		RETURNING id
	`, repoID, filePath, name, blockType).Scan(&blockID)
	require.NoError(t, err)
	return blockID
}

// createTestModification creates a test modification record
func createTestModification(t *testing.T, db *sql.DB, repoID, blockID int64, commitSHA, authorEmail string, timestamp time.Time) {
	_, err := db.Exec(`
		INSERT INTO code_block_modifications (repo_id, code_block_id, commit_sha, author_email, modified_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (code_block_id, commit_sha) DO NOTHING
	`, repoID, blockID, commitSHA, authorEmail, timestamp)
	require.NoError(t, err)
}

func TestCouplingCalculator_CalculateCoChanges(t *testing.T) {
	db, repoID := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db, repoID)

	ctx := context.Background()

	// Create test blocks
	blockA := createTestCodeBlock(t, db, repoID, "src/parser.go", "parseExpression", "function")
	blockB := createTestCodeBlock(t, db, repoID, "src/validator.go", "validateExpression", "function")
	blockC := createTestCodeBlock(t, db, repoID, "src/utils.go", "formatError", "function")

	// Create co-change pattern:
	// - blockA and blockB change together in 3 out of 4 commits (75% co-change rate)
	// - blockA and blockC change together in 2 out of 4 commits (50% co-change rate)
	// - blockB and blockC change together in 1 out of 3 commits (33% co-change rate - below threshold)

	baseTime := time.Now().Add(-24 * time.Hour)

	// Commit 1: A + B
	createTestModification(t, db, repoID, blockA, "commit1", "alice@example.com", baseTime)
	createTestModification(t, db, repoID, blockB, "commit1", "alice@example.com", baseTime)

	// Commit 2: A + B
	createTestModification(t, db, repoID, blockA, "commit2", "bob@example.com", baseTime.Add(1*time.Hour))
	createTestModification(t, db, repoID, blockB, "commit2", "bob@example.com", baseTime.Add(1*time.Hour))

	// Commit 3: A + B + C
	createTestModification(t, db, repoID, blockA, "commit3", "alice@example.com", baseTime.Add(2*time.Hour))
	createTestModification(t, db, repoID, blockB, "commit3", "alice@example.com", baseTime.Add(2*time.Hour))
	createTestModification(t, db, repoID, blockC, "commit3", "alice@example.com", baseTime.Add(2*time.Hour))

	// Commit 4: A + C
	createTestModification(t, db, repoID, blockA, "commit4", "charlie@example.com", baseTime.Add(3*time.Hour))
	createTestModification(t, db, repoID, blockC, "commit4", "charlie@example.com", baseTime.Add(3*time.Hour))

	// Create coupling calculator (without LLM for testing)
	calc := NewCouplingCalculator(db, nil, repoID)

	// Calculate co-changes
	edgesCreated, err := calc.CalculateCoChanges(ctx)
	require.NoError(t, err)

	// Should create 2 edges (A-B with 75% and A-C with 50%)
	// B-C has only 33% so it should not be created
	assert.Equal(t, 2, edgesCreated, "should create 2 co-change edges")

	// Verify the edges exist in the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM code_block_co_changes WHERE repo_id = $1", repoID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should have 2 co-change edges in database")

	// Verify A-B edge (75% co-change rate)
	var coChangeRate float64
	var coChangeCount int
	err = db.QueryRow(`
		SELECT co_change_rate, co_change_count
		FROM code_block_co_changes
		WHERE repo_id = $1 AND block_a_id = $2 AND block_b_id = $3
	`, repoID, blockA, blockB).Scan(&coChangeRate, &coChangeCount)
	require.NoError(t, err)
	assert.Equal(t, 0.75, coChangeRate, "A-B co-change rate should be 75%")
	assert.Equal(t, 3, coChangeCount, "A-B should have 3 co-changes")

	// Verify A-C edge (50% co-change rate)
	err = db.QueryRow(`
		SELECT co_change_rate, co_change_count
		FROM code_block_co_changes
		WHERE repo_id = $1 AND block_a_id = $2 AND block_b_id = $3
	`, repoID, blockA, blockC).Scan(&coChangeRate, &coChangeCount)
	require.NoError(t, err)
	assert.Equal(t, 0.50, coChangeRate, "A-C co-change rate should be 50%")
	assert.Equal(t, 2, coChangeCount, "A-C should have 2 co-changes")

	// Verify B-C edge does NOT exist (33% is below threshold)
	err = db.QueryRow(`
		SELECT co_change_rate
		FROM code_block_co_changes
		WHERE repo_id = $1 AND ((block_a_id = $2 AND block_b_id = $3) OR (block_a_id = $3 AND block_b_id = $2))
	`, repoID, blockB, blockC).Scan(&coChangeRate)
	assert.Equal(t, sql.ErrNoRows, err, "B-C edge should not exist (below 50% threshold)")
}

func TestCouplingCalculator_GetCoChangeStatistics(t *testing.T) {
	db, repoID := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db, repoID)

	ctx := context.Background()

	// Create test blocks and modifications
	blockA := createTestCodeBlock(t, db, repoID, "src/a.go", "funcA", "function")
	blockB := createTestCodeBlock(t, db, repoID, "src/b.go", "funcB", "function")
	blockC := createTestCodeBlock(t, db, repoID, "src/c.go", "funcC", "function")

	baseTime := time.Now().Add(-24 * time.Hour)

	// Create strong coupling: A-B (100%)
	createTestModification(t, db, repoID, blockA, "commit1", "alice@example.com", baseTime)
	createTestModification(t, db, repoID, blockB, "commit1", "alice@example.com", baseTime)

	// Create medium coupling: A-C (50%)
	createTestModification(t, db, repoID, blockA, "commit2", "bob@example.com", baseTime.Add(1*time.Hour))
	createTestModification(t, db, repoID, blockC, "commit2", "bob@example.com", baseTime.Add(1*time.Hour))

	// Calculate co-changes
	calc := NewCouplingCalculator(db, nil, repoID)
	_, err := calc.CalculateCoChanges(ctx)
	require.NoError(t, err)

	// Get statistics
	stats, err := calc.GetCoChangeStatistics(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, stats["total_edges"], "should have 2 total edges")
	assert.Equal(t, 0.5, stats["min_rate"], "min rate should be 0.5")
	assert.Equal(t, 1.0, stats["max_rate"], "max rate should be 1.0")
	assert.Equal(t, 1, stats["high_coupling_count"], "should have 1 high coupling edge (>= 0.75)")
	assert.Equal(t, 1, stats["medium_coupling_count"], "should have 1 medium coupling edge (0.5-0.75)")
}

func TestCouplingCalculator_GetTopCoupledBlocks(t *testing.T) {
	db, repoID := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db, repoID)

	ctx := context.Background()

	// Create test blocks
	blockA := createTestCodeBlock(t, db, repoID, "src/a.go", "funcA", "function")
	blockB := createTestCodeBlock(t, db, repoID, "src/b.go", "funcB", "function")
	blockC := createTestCodeBlock(t, db, repoID, "src/c.go", "funcC", "function")
	blockD := createTestCodeBlock(t, db, repoID, "src/d.go", "funcD", "function")

	baseTime := time.Now().Add(-24 * time.Hour)

	// Create coupling pattern where A is coupled with B, C, and D
	// A has the most couplings
	for i := 0; i < 3; i++ {
		commitSHA := fmt.Sprintf("commit-a-b-%d", i)
		createTestModification(t, db, repoID, blockA, commitSHA, "alice@example.com", baseTime.Add(time.Duration(i)*time.Hour))
		createTestModification(t, db, repoID, blockB, commitSHA, "alice@example.com", baseTime.Add(time.Duration(i)*time.Hour))
	}

	for i := 0; i < 2; i++ {
		commitSHA := fmt.Sprintf("commit-a-c-%d", i)
		createTestModification(t, db, repoID, blockA, commitSHA, "bob@example.com", baseTime.Add(time.Duration(i+3)*time.Hour))
		createTestModification(t, db, repoID, blockC, commitSHA, "bob@example.com", baseTime.Add(time.Duration(i+3)*time.Hour))
	}

	for i := 0; i < 2; i++ {
		commitSHA := fmt.Sprintf("commit-a-d-%d", i)
		createTestModification(t, db, repoID, blockA, commitSHA, "charlie@example.com", baseTime.Add(time.Duration(i+5)*time.Hour))
		createTestModification(t, db, repoID, blockD, commitSHA, "charlie@example.com", baseTime.Add(time.Duration(i+5)*time.Hour))
	}

	// Calculate co-changes
	calc := NewCouplingCalculator(db, nil, repoID)
	_, err := calc.CalculateCoChanges(ctx)
	require.NoError(t, err)

	// Get top coupled blocks
	topBlocks, err := calc.GetTopCoupledBlocks(ctx, 5)
	require.NoError(t, err)

	// Block A should be at the top with 3 couplings
	assert.GreaterOrEqual(t, len(topBlocks), 1, "should have at least 1 block")
	assert.Equal(t, blockA, topBlocks[0]["id"].(int64), "block A should be the most coupled")
	assert.Equal(t, 3, topBlocks[0]["total_couplings"].(int), "block A should have 3 couplings")
}

func TestCouplingCalculator_ExplainCoupling(t *testing.T) {
	// This test requires a working LLM client
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		t.Skip("Skipping LLM test: GEMINI_API_KEY not set")
	}

	os.Setenv("PHASE2_ENABLED", "true")
	os.Setenv("LLM_PROVIDER", "gemini")

	ctx := context.Background()
	cfg := &config.Config{
		API: config.APIConfig{
			GeminiKey: geminiKey,
		},
	}

	llmClient, err := llm.NewClient(ctx, cfg)
	require.NoError(t, err)

	calc := NewCouplingCalculator(nil, llmClient, 1)

	pair := CoChangePair{
		BlockA: CodeBlockInfo{
			ID:       1,
			FilePath: "src/parser.go",
			Name:     "parseExpression",
			Type:     "function",
			Language: "go",
		},
		BlockB: CodeBlockInfo{
			ID:       2,
			FilePath: "src/validator.go",
			Name:     "validateExpression",
			Type:     "function",
			Language: "go",
		},
		CoChangeRate:  0.75,
		CoChangeCount: 3,
		TotalChangesA: 4,
	}

	explanation, err := calc.ExplainCoupling(ctx, pair)
	require.NoError(t, err)
	assert.NotEmpty(t, explanation, "explanation should not be empty")
	t.Logf("LLM Explanation: %s", explanation)
}

func TestCouplingCalculator_EdgeCases(t *testing.T) {
	db, repoID := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db, repoID)

	ctx := context.Background()

	t.Run("no code blocks", func(t *testing.T) {
		calc := NewCouplingCalculator(db, nil, repoID)
		edgesCreated, err := calc.CalculateCoChanges(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, edgesCreated, "should create 0 edges when no blocks exist")
	})

	t.Run("single block", func(t *testing.T) {
		blockA := createTestCodeBlock(t, db, repoID, "src/single.go", "funcSingle", "function")
		createTestModification(t, db, repoID, blockA, "commit1", "alice@example.com", time.Now())

		calc := NewCouplingCalculator(db, nil, repoID)
		edgesCreated, err := calc.CalculateCoChanges(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, edgesCreated, "should create 0 edges with only 1 block")
	})

	t.Run("blocks in same file", func(t *testing.T) {
		blockA := createTestCodeBlock(t, db, repoID, "src/same.go", "funcA", "function")
		blockB := createTestCodeBlock(t, db, repoID, "src/same.go", "funcB", "function")

		// Both change together
		baseTime := time.Now()
		createTestModification(t, db, repoID, blockA, "commit-same-1", "alice@example.com", baseTime)
		createTestModification(t, db, repoID, blockB, "commit-same-1", "alice@example.com", baseTime)

		calc := NewCouplingCalculator(db, nil, repoID)
		edgesCreated, err := calc.CalculateCoChanges(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, edgesCreated, "should create edge even for blocks in same file")
	})
}
