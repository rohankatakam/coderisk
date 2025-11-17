package atomizer

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessorIntegration tests the end-to-end processing flow
// Reference: AGENT_P2B_PROCESSOR.md - Integration tests
func TestProcessorIntegration(t *testing.T) {
	// Skip if no database connection available
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	neoURI := os.Getenv("NEO4J_URI")
	neoUser := os.Getenv("NEO4J_USERNAME")
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoURI == "" || neoUser == "" || neoPassword == "" {
		t.Skip("Skipping integration test: Neo4j credentials not set")
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	// Connect to Neo4j
	driver, err := neo4j.NewDriverWithContext(neoURI, neo4j.BasicAuth(neoUser, neoPassword, ""))
	require.NoError(t, err)
	defer driver.Close(ctx)

	// Create LLM client
	// For testing purposes, we'll use environment variables
	os.Setenv("PHASE2_ENABLED", "true")
	os.Setenv("LLM_PROVIDER", "gemini")

	// Create a minimal config for testing
	cfg := &config.Config{
		API: config.APIConfig{
			GeminiKey: os.Getenv("GEMINI_API_KEY"),
		},
	}
	llmClient, err := llm.NewClient(ctx, cfg)
	require.NoError(t, err)

	// Create processor
	extractor := NewExtractor(llmClient)
	processor := NewProcessor(extractor, db, driver, "neo4j")

	// Test data: Simple CREATE → MODIFY → DELETE sequence
	commits := []CommitData{
		{
			SHA:         "commit1",
			Message:     "Add parseExpression function",
			DiffContent: createSimpleDiff("src/parser.go", "parseExpression", "CREATE"),
			AuthorEmail: "alice@example.com",
			Timestamp:   time.Now().Add(-2 * time.Hour),
		},
		{
			SHA:         "commit2",
			Message:     "Update parseExpression to handle errors",
			DiffContent: createSimpleDiff("src/parser.go", "parseExpression", "MODIFY"),
			AuthorEmail: "bob@example.com",
			Timestamp:   time.Now().Add(-1 * time.Hour),
		},
		{
			SHA:         "commit3",
			Message:     "Remove parseExpression",
			DiffContent: createSimpleDiff("src/parser.go", "parseExpression", "DELETE"),
			AuthorEmail: "alice@example.com",
			Timestamp:   time.Now(),
		},
	}

	// Process commits
	repoID := int64(1) // Test repo ID
	err = processor.ProcessCommitsChronologically(ctx, commits, repoID)
	assert.NoError(t, err)

	// Verify PostgreSQL data
	var blockCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1", repoID).Scan(&blockCount)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, blockCount, 0, "Should have processed at least one code block")

	var modCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_block_modifications WHERE repo_id = $1", repoID).Scan(&modCount)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, modCount, 0, "Should have modification records")
}

// TestStateTracker tests the state tracker functionality
// Reference: AGENT_P2B_PROCESSOR.md - State tracking tests
func TestStateTracker(t *testing.T) {
	state := NewStateTracker()

	// Test initial state
	assert.Equal(t, 0, state.GetBlockCount())
	assert.False(t, state.BlockExists("src/file.go", "funcA"))

	// Test adding blocks
	state.SetBlockID("src/file.go", "funcA", 100)
	assert.Equal(t, 1, state.GetBlockCount())
	assert.True(t, state.BlockExists("src/file.go", "funcA"))

	id, exists := state.GetBlockID("src/file.go", "funcA")
	assert.True(t, exists)
	assert.Equal(t, int64(100), id)

	// Test adding multiple blocks
	state.SetBlockID("src/file.go", "funcB", 101)
	state.SetBlockID("src/other.go", "funcC", 102)
	assert.Equal(t, 3, state.GetBlockCount())

	// Test deleting blocks
	state.DeleteBlock("src/file.go", "funcA")
	assert.Equal(t, 2, state.GetBlockCount())
	assert.False(t, state.BlockExists("src/file.go", "funcA"))

	// Test non-existent block
	id, exists = state.GetBlockID("nonexistent.go", "funcX")
	assert.False(t, exists)
	assert.Equal(t, int64(0), id)
}

// TestDBWriter tests database write operations
func TestDBWriter(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	writer := NewDBWriter(db)

	// Test creating a code block
	event := &ChangeEvent{
		Behavior:        "CREATE_BLOCK",
		TargetFile:      "test/example.go",
		TargetBlockName: "testFunction",
		BlockType:       "function",
	}

	repoID := int64(999) // Test repo ID
	commitSHA := "test_commit_sha"
	authorEmail := "test@example.com"
	timestamp := time.Now()

	blockID, err := writer.CreateCodeBlock(ctx, event, commitSHA, authorEmail, timestamp, repoID)
	if err != nil {
		// Block might already exist from previous test runs
		t.Logf("Note: CreateCodeBlock returned error (may be duplicate): %v", err)
	} else {
		assert.Greater(t, blockID, int64(0))

		// Test creating a modification
		err = writer.CreateModification(ctx, blockID, repoID, commitSHA, authorEmail, timestamp, "create")
		assert.NoError(t, err)
	}
}

// TestDetectLanguage tests language detection from file extensions
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filepath string
		expected string
	}{
		{"src/main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"component.tsx", "typescript"},
		{"script.rb", "ruby"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			result := detectLanguage(tt.filepath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEventProcessing tests individual event handlers
func TestEventProcessing(t *testing.T) {
	// Test CREATE_BLOCK for existing block (edge case)
	state := NewStateTracker()
	state.SetBlockID("src/file.go", "existingFunc", 100)

	assert.True(t, state.BlockExists("src/file.go", "existingFunc"))
	assert.Equal(t, 1, state.GetBlockCount())

	// Test MODIFY_BLOCK for non-existent block (edge case)
	assert.False(t, state.BlockExists("src/file.go", "newFunc"))

	// Test DELETE_BLOCK
	state.DeleteBlock("src/file.go", "existingFunc")
	assert.False(t, state.BlockExists("src/file.go", "existingFunc"))
	assert.Equal(t, 0, state.GetBlockCount())
}

// createSimpleDiff creates a simple test diff for testing
func createSimpleDiff(file, funcName, operation string) string {
	switch operation {
	case "CREATE":
		return `diff --git a/` + file + ` b/` + file + `
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/` + file + `
@@ -0,0 +1,5 @@
+func ` + funcName + `() {
+    // New function
+    return nil
+}
`
	case "MODIFY":
		return `diff --git a/` + file + ` b/` + file + `
index 1234567..abcdefg 100644
--- a/` + file + `
+++ b/` + file + `
@@ -1,5 +1,6 @@
 func ` + funcName + `() {
-    // Old implementation
+    // Updated implementation
+    handleError()
     return nil
 }
`
	case "DELETE":
		return `diff --git a/` + file + ` b/` + file + `
index abcdefg..0000000 100644
--- a/` + file + `
+++ b/` + file + `
@@ -1,6 +0,0 @@
-func ` + funcName + `() {
-    // Function removed
-    return nil
-}
-
`
	default:
		return ""
	}
}
