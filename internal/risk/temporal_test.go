package risk

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestLinkIssuesViaCommits tests the incident linking functionality
func TestLinkIssuesViaCommits(t *testing.T) {
	db, repoID := setupTemporalTestDB(t)
	defer db.Close()
	defer cleanupTemporalTestData(t, db, repoID)

	ctx := context.Background()

	// Create test data
	issueID := createTestIssue(t, db, repoID, 1, "Test bug", "closed")
	commitSHA := "abc123def456"
	createTestCommit(t, db, repoID, commitSHA, "Fix test bug", "test@example.com")

	blockID := createTemporalTestCodeBlock(t, db, repoID, "test.go", "TestFunction", "function")
	createTemporalTestBlockModification(t, db, repoID, blockID, commitSHA, "test@example.com")

	// Create timeline event linking issue to commit
	createTestTimelineEvent(t, db, repoID, issueID, commitSHA, "closed")

	// Run the linking
	calculator := NewTemporalCalculator(db, nil, nil, repoID)
	linksCreated, err := calculator.LinkIssuesViaCommits(ctx)
	if err != nil {
		t.Fatalf("LinkIssuesViaCommits failed: %v", err)
	}

	if linksCreated == 0 {
		t.Error("Expected at least one link to be created")
	}

	// Verify the link was created
	var count int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM code_block_incidents
		WHERE code_block_id = $1 AND issue_id = $2
	`, blockID, issueID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query incident link: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 incident link, got %d", count)
	}
}

// TestCalculateIncidentCounts tests the incident count calculation
func TestCalculateIncidentCounts(t *testing.T) {
	db, repoID := setupTemporalTestDB(t)
	defer db.Close()
	defer cleanupTemporalTestData(t, db, repoID)

	ctx := context.Background()

	// Create blocks with different incident counts
	block1 := createTemporalTestCodeBlock(t, db, repoID, "file1.go", "Function1", "function")
	block2 := createTemporalTestCodeBlock(t, db, repoID, "file2.go", "Function2", "function")
	block3 := createTemporalTestCodeBlock(t, db, repoID, "file3.go", "Function3", "function")

	// Create issues
	issue1 := createTestIssue(t, db, repoID, 1, "Bug 1", "closed")
	issue2 := createTestIssue(t, db, repoID, 2, "Bug 2", "closed")

	// Link block1 to both issues
	createTestIncident(t, db, repoID, block1, issue1)
	createTestIncident(t, db, repoID, block1, issue2)

	// Link block2 to one issue
	createTestIncident(t, db, repoID, block2, issue1)

	// block3 has no incidents

	// Calculate counts
	calculator := NewTemporalCalculator(db, nil, nil, repoID)
	updated, err := calculator.CalculateIncidentCounts(ctx)
	if err != nil {
		t.Fatalf("CalculateIncidentCounts failed: %v", err)
	}

	if updated == 0 {
		t.Error("Expected some blocks to be updated")
	}

	// Verify counts
	verifyIncidentCount(t, db, block1, 2)
	verifyIncidentCount(t, db, block2, 1)
	verifyIncidentCount(t, db, block3, 0)
}

// TestGetIncidentStatistics tests the statistics query
func TestGetIncidentStatistics(t *testing.T) {
	db, repoID := setupTemporalTestDB(t)
	defer db.Close()
	defer cleanupTemporalTestData(t, db, repoID)

	ctx := context.Background()

	// Create test data
	block1 := createTemporalTestCodeBlock(t, db, repoID, "file1.go", "Function1", "function")
	block2 := createTemporalTestCodeBlock(t, db, repoID, "file2.go", "Function2", "function")
	issue1 := createTestIssue(t, db, repoID, 1, "Bug 1", "closed")
	issue2 := createTestIssue(t, db, repoID, 2, "Bug 2", "closed")

	createTestIncident(t, db, repoID, block1, issue1)
	createTestIncident(t, db, repoID, block1, issue2)
	createTestIncident(t, db, repoID, block2, issue1)

	// Get statistics
	calculator := NewTemporalCalculator(db, nil, nil, repoID)
	stats, err := calculator.GetIncidentStatistics(ctx)
	if err != nil {
		t.Fatalf("GetIncidentStatistics failed: %v", err)
	}

	if stats["blocks_with_incidents"].(int) != 2 {
		t.Errorf("Expected 2 blocks with incidents, got %v", stats["blocks_with_incidents"])
	}

	if stats["total_unique_issues"].(int) != 2 {
		t.Errorf("Expected 2 unique issues, got %v", stats["total_unique_issues"])
	}

	if stats["total_incident_links"].(int) != 3 {
		t.Errorf("Expected 3 incident links, got %v", stats["total_incident_links"])
	}
}

// TestGetTopIncidentBlocks tests the top incident blocks query
func TestGetTopIncidentBlocks(t *testing.T) {
	db, repoID := setupTemporalTestDB(t)
	defer db.Close()
	defer cleanupTemporalTestData(t, db, repoID)

	ctx := context.Background()

	// Create blocks with different incident counts
	block1 := createTemporalTestCodeBlock(t, db, repoID, "hotspot.go", "HotFunction", "function")
	block2 := createTemporalTestCodeBlock(t, db, repoID, "normal.go", "NormalFunction", "function")

	issue1 := createTestIssue(t, db, repoID, 1, "Bug 1", "closed")
	issue2 := createTestIssue(t, db, repoID, 2, "Bug 2", "closed")
	issue3 := createTestIssue(t, db, repoID, 3, "Bug 3", "closed")

	// Block1 has 3 incidents (hotspot)
	createTestIncident(t, db, repoID, block1, issue1)
	createTestIncident(t, db, repoID, block1, issue2)
	createTestIncident(t, db, repoID, block1, issue3)

	// Block2 has 1 incident
	createTestIncident(t, db, repoID, block2, issue1)

	// Update counts
	calculator := NewTemporalCalculator(db, nil, nil, repoID)
	_, err := calculator.CalculateIncidentCounts(ctx)
	if err != nil {
		t.Fatalf("CalculateIncidentCounts failed: %v", err)
	}

	// Get top blocks
	topBlocks, err := calculator.GetTopIncidentBlocks(ctx, 10)
	if err != nil {
		t.Fatalf("GetTopIncidentBlocks failed: %v", err)
	}

	if len(topBlocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(topBlocks))
	}

	// First block should be the hotspot
	if topBlocks[0]["block_name"] != "HotFunction" {
		t.Errorf("Expected HotFunction first, got %v", topBlocks[0]["block_name"])
	}

	if topBlocks[0]["incident_count"].(int) != 3 {
		t.Errorf("Expected 3 incidents, got %v", topBlocks[0]["incident_count"])
	}
}

// Helper functions specific to temporal tests

func setupTemporalTestDB(t *testing.T) (*sql.DB, int64) {
	// Try to connect to test database
	dbURL := "postgresql://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping test: cannot connect to database")
	}

	if err := db.Ping(); err != nil {
		t.Skip("Skipping test: database not available")
	}

	// Create test repo
	repoID := createTemporalTestRepo(t, db, "test-owner", "test-repo-temporal")
	return db, repoID
}

func createTemporalTestRepo(t *testing.T, db *sql.DB, owner, name string) int64 {
	var id int64
	err := db.QueryRow(`
		INSERT INTO github_repositories (github_id, owner, name, full_name, url, default_branch, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'main', NOW(), NOW())
		ON CONFLICT (owner, name) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, time.Now().Unix(), owner, name, owner+"/"+name, "https://github.com/"+owner+"/"+name).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}
	return id
}

func createTestIssue(t *testing.T, db *sql.DB, repoID int64, number int, title, state string) int64 {
	var id int64
	err := db.QueryRow(`
		INSERT INTO github_issues (repo_id, github_id, number, title, state, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, repoID, number, number, title, state).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}
	return id
}

func createTestCommit(t *testing.T, db *sql.DB, repoID int64, sha, message, authorEmail string) {
	_, err := db.Exec(`
		INSERT INTO github_commits (repo_id, sha, message, author_email, author_date, raw_data)
		VALUES ($1, $2, $3, $4, NOW(), '{}'::jsonb)
		ON CONFLICT (repo_id, sha) DO NOTHING
	`, repoID, sha, message, authorEmail)
	if err != nil {
		t.Fatalf("Failed to create test commit: %v", err)
	}
}

func createTemporalTestCodeBlock(t *testing.T, db *sql.DB, repoID int64, filePath, blockName, blockType string) int64 {
	var id int64
	err := db.QueryRow(`
		INSERT INTO code_blocks (repo_id, file_path, block_name, block_type, language, first_seen_commit_sha, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'go', 'test-sha', NOW(), NOW())
		RETURNING id
	`, repoID, filePath, blockName, blockType).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test code block: %v", err)
	}
	return id
}

func createTemporalTestBlockModification(t *testing.T, db *sql.DB, repoID int64, blockID int64, commitSHA, authorEmail string) {
	_, err := db.Exec(`
		INSERT INTO code_block_modifications (code_block_id, commit_sha, developer_email, change_type, modified_at)
		VALUES ($1, $2, $3, 'modified', NOW())
		ON CONFLICT (code_block_id, commit_sha) DO NOTHING
	`, blockID, commitSHA, authorEmail)
	if err != nil {
		t.Fatalf("Failed to create test block modification: %v", err)
	}
}

func createTestTimelineEvent(t *testing.T, db *sql.DB, repoID int64, issueID int64, commitSHA, eventType string) {
	_, err := db.Exec(`
		INSERT INTO github_issue_timeline (issue_id, event_type, source_sha, created_at)
		VALUES ($1, $2, $3, NOW())
	`, issueID, eventType, commitSHA)
	if err != nil {
		t.Fatalf("Failed to create test timeline event: %v", err)
	}
}

func createTestIncident(t *testing.T, db *sql.DB, repoID int64, blockID int64, issueID int64) {
	_, err := db.Exec(`
		INSERT INTO code_block_incidents (repo_id, code_block_id, issue_id, confidence, evidence_source, evidence_text)
		VALUES ($1, $2, $3, 0.80, 'test', 'test evidence')
		ON CONFLICT (code_block_id, issue_id) DO NOTHING
	`, repoID, blockID, issueID)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}
}

func verifyIncidentCount(t *testing.T, db *sql.DB, blockID int64, expected int) {
	var count int
	err := db.QueryRow(`
		SELECT incident_count FROM code_blocks WHERE id = $1
	`, blockID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query incident count: %v", err)
	}

	if count != expected {
		t.Errorf("Expected incident count %d for block %d, got %d", expected, blockID, count)
	}
}

func cleanupTemporalTestData(t *testing.T, db *sql.DB, repoID int64) {
	// Clean up in reverse order of dependencies
	_, err := db.Exec(`DELETE FROM code_block_incidents WHERE repo_id = $1`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up incidents: %v", err)
	}

	// Delete modifications for blocks belonging to this repo
	_, err = db.Exec(`
		DELETE FROM code_block_modifications
		WHERE code_block_id IN (SELECT id FROM code_blocks WHERE repo_id = $1)
	`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up modifications: %v", err)
	}

	// Note: github_issue_timeline doesn't have repo_id, cleanup via issue_id
	_, err = db.Exec(`
		DELETE FROM github_issue_timeline
		WHERE issue_id IN (SELECT id FROM github_issues WHERE repo_id = $1)
	`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up timeline events: %v", err)
	}

	_, err = db.Exec(`DELETE FROM code_blocks WHERE repo_id = $1`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up code blocks: %v", err)
	}

	_, err = db.Exec(`DELETE FROM github_commits WHERE repo_id = $1`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up commits: %v", err)
	}

	_, err = db.Exec(`DELETE FROM github_issues WHERE repo_id = $1`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up issues: %v", err)
	}

	_, err = db.Exec(`DELETE FROM github_repositories WHERE id = $1`, repoID)
	if err != nil {
		t.Logf("Warning: failed to clean up repository: %v", err)
	}
}
