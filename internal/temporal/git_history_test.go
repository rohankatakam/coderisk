package temporal

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseGitLogOutput(t *testing.T) {
	// Test parsing git log output
	gitLogOutput := `abc123|John Doe|john@example.com|2025-09-15T10:00:00Z|Fix auth bug
10	5	src/auth.ts
3	1	src/database.ts

def456|Jane Smith|jane@example.com|2025-09-16T14:30:00Z|Add caching
25	0	src/cache.ts
5	2	src/database.ts
`

	commits, err := parseGitLogOutput(gitLogOutput)
	if err != nil {
		t.Fatalf("parseGitLogOutput failed: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}

	// Verify first commit
	commit1 := commits[0]
	if commit1.SHA != "abc123" {
		t.Errorf("expected SHA abc123, got %s", commit1.SHA)
	}
	if commit1.Author != "John Doe" {
		t.Errorf("expected author John Doe, got %s", commit1.Author)
	}
	if commit1.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %s", commit1.Email)
	}
	if commit1.Message != "Fix auth bug" {
		t.Errorf("expected message 'Fix auth bug', got '%s'", commit1.Message)
	}
	if len(commit1.FilesChanged) != 2 {
		t.Fatalf("expected 2 files changed, got %d", len(commit1.FilesChanged))
	}
	if commit1.FilesChanged[0].Path != "src/auth.ts" {
		t.Errorf("expected path src/auth.ts, got %s", commit1.FilesChanged[0].Path)
	}
	if commit1.FilesChanged[0].Additions != 10 {
		t.Errorf("expected 10 additions, got %d", commit1.FilesChanged[0].Additions)
	}
	if commit1.FilesChanged[0].Deletions != 5 {
		t.Errorf("expected 5 deletions, got %d", commit1.FilesChanged[0].Deletions)
	}

	// Verify second commit
	commit2 := commits[1]
	if commit2.SHA != "def456" {
		t.Errorf("expected SHA def456, got %s", commit2.SHA)
	}
	if len(commit2.FilesChanged) != 2 {
		t.Fatalf("expected 2 files changed, got %d", len(commit2.FilesChanged))
	}
}

func TestExtractDevelopers(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	commits := []Commit{
		{
			SHA:       "abc123",
			Author:    "John Doe",
			Email:     "john@example.com",
			Timestamp: yesterday,
		},
		{
			SHA:       "def456",
			Author:    "John Doe",
			Email:     "john@example.com",
			Timestamp: now,
		},
		{
			SHA:       "ghi789",
			Author:    "Jane Smith",
			Email:     "jane@example.com",
			Timestamp: now,
		},
	}

	developers := ExtractDevelopers(commits)

	if len(developers) != 2 {
		t.Fatalf("expected 2 developers, got %d", len(developers))
	}

	// Find John's developer entry
	var johnDev *Developer
	for i := range developers {
		if developers[i].Email == "john@example.com" {
			johnDev = &developers[i]
			break
		}
	}

	if johnDev == nil {
		t.Fatal("John Doe not found in developers")
	}

	if johnDev.TotalCommits != 2 {
		t.Errorf("expected 2 commits for John, got %d", johnDev.TotalCommits)
	}
	if !johnDev.FirstCommit.Equal(yesterday) {
		t.Errorf("expected first commit yesterday, got %v", johnDev.FirstCommit)
	}
	if !johnDev.LastCommit.Equal(now) {
		t.Errorf("expected last commit now, got %v", johnDev.LastCommit)
	}
}

func TestParseGitHistory(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tmpDir
	if err := configEmail.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tmpDir
	if err := configName.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Add and commit
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = tmpDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tmpDir
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit failed: %v, output: %s", err, string(output))
	}

	// Parse git history
	commits, err := ParseGitHistory(tmpDir, 90)
	if err != nil {
		t.Fatalf("ParseGitHistory failed: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("expected at least 1 commit")
	}

	commit := commits[0]
	if commit.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", commit.Email)
	}
	if commit.Message != "Initial commit" {
		t.Errorf("expected message 'Initial commit', got '%s'", commit.Message)
	}
}

func TestParseGitLogOutput_BinaryFiles(t *testing.T) {
	// Binary files should be skipped (marked with "-")
	gitLogOutput := `abc123|John Doe|john@example.com|2025-09-15T10:00:00Z|Add image
-	-	assets/logo.png
10	5	src/auth.ts
`

	commits, err := parseGitLogOutput(gitLogOutput)
	if err != nil {
		t.Fatalf("parseGitLogOutput failed: %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}

	// Should only have auth.ts, not logo.png
	if len(commits[0].FilesChanged) != 1 {
		t.Fatalf("expected 1 file change, got %d", len(commits[0].FilesChanged))
	}
	if commits[0].FilesChanged[0].Path != "src/auth.ts" {
		t.Errorf("expected src/auth.ts, got %s", commits[0].FilesChanged[0].Path)
	}
}
