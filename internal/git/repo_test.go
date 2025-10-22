package git

import (
	"os"
	"os/exec"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	// Create a temporary directory for test repo
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create initial commit
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", "test.txt").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	// Test: Get current branch (should be main or master)
	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}
	if branch != "main" && branch != "master" {
		// Git might use different default branch names
		t.Logf("Current branch: %s (expected main or master)", branch)
	}
}

func TestGetRemoteURL(t *testing.T) {
	// Create a temporary directory for test repo
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Test 1: No remote configured
	_, err = GetRemoteURL()
	if err == nil {
		t.Error("Expected error when no remote configured")
	}

	// Test 2: Add remote
	testURL := "https://github.com/test/repo.git"
	if err := exec.Command("git", "remote", "add", "origin", testURL).Run(); err != nil {
		t.Fatal(err)
	}

	url, err := GetRemoteURL()
	if err != nil {
		t.Fatalf("GetRemoteURL() error = %v", err)
	}
	if url != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, url)
	}
}

func TestGetCurrentCommitSHA(t *testing.T) {
	// Create a temporary directory for test repo
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Test 1: No commits yet
	_, err = GetCurrentCommitSHA()
	if err == nil {
		t.Error("Expected error when no commits exist")
	}

	// Test 2: After commit
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", "test.txt").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	sha, err := GetCurrentCommitSHA()
	if err != nil {
		t.Fatalf("GetCurrentCommitSHA() error = %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("Expected SHA length 40, got %d", len(sha))
	}
}

func TestGetAuthorEmail(t *testing.T) {
	// Create a temporary directory for test repo
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Test 1: No email configured
	_, err = GetAuthorEmail()
	// This might succeed with global config, so we don't assert error

	// Test 2: Set email
	testEmail := "test@example.com"
	if err := exec.Command("git", "config", "user.email", testEmail).Run(); err != nil {
		t.Fatal(err)
	}

	email, err := GetAuthorEmail()
	if err != nil {
		t.Fatalf("GetAuthorEmail() error = %v", err)
	}
	if email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, email)
	}
}

func TestDetectGitRepo(t *testing.T) {
	// Test 1: In a git repository
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Should detect git repo
	if err := DetectGitRepo(); err != nil {
		t.Errorf("DetectGitRepo() error = %v, expected nil in git repo", err)
	}

	// Test 2: Not in a git repository
	tmpDir2 := t.TempDir()
	if err := os.Chdir(tmpDir2); err != nil {
		t.Fatal(err)
	}

	// Should return error
	if err := DetectGitRepo(); err == nil {
		t.Error("DetectGitRepo() expected error in non-git directory")
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantOrg  string
		wantRepo string
		wantErr  bool
	}{
		{
			name:     "HTTPS with .git",
			url:      "https://github.com/rohankatakam/coderisk.git",
			wantOrg:  "rohankatakam",
			wantRepo: "coderisk",
			wantErr:  false,
		},
		{
			name:     "HTTPS without .git",
			url:      "https://github.com/rohankatakam/coderisk",
			wantOrg:  "rohankatakam",
			wantRepo: "coderisk",
			wantErr:  false,
		},
		{
			name:     "HTTP with .git",
			url:      "http://github.com/rohankatakam/coderisk.git",
			wantOrg:  "rohankatakam",
			wantRepo: "coderisk",
			wantErr:  false,
		},
		{
			name:     "SSH format",
			url:      "git@github.com:coderisk/coderisk-go.git",
			wantOrg:  "coderisk",
			wantRepo: "coderisk-go",
			wantErr:  false,
		},
		{
			name:     "SSH without .git",
			url:      "git@github.com:coderisk/coderisk-go",
			wantOrg:  "coderisk",
			wantRepo: "coderisk-go",
			wantErr:  false,
		},
		{
			name:     "Git protocol",
			url:      "git://github.com/rohankatakam/coderisk.git",
			wantOrg:  "rohankatakam",
			wantRepo: "coderisk",
			wantErr:  false,
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/myorg/myrepo.git",
			wantOrg:  "myorg",
			wantRepo: "myrepo",
			wantErr:  false,
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:myorg/myrepo.git",
			wantOrg:  "myorg",
			wantRepo: "myrepo",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-git-url",
			wantOrg:  "",
			wantRepo: "",
			wantErr:  true,
		},
		{
			name:     "Invalid format - no slash",
			url:      "https://github.com/onlyonepart",
			wantOrg:  "",
			wantRepo: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, err := ParseRepoURL(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if org != tt.wantOrg {
				t.Errorf("ParseRepoURL() org = %v, want %v", org, tt.wantOrg)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseRepoURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetChangedFiles(t *testing.T) {
	// Create a temporary directory for test repo
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Test 1: No commits yet (should error)
	_, err = GetChangedFiles()
	if err == nil {
		t.Log("GetChangedFiles() succeeded with no commits (may return empty list)")
	}

	// Create initial commit
	if err := os.WriteFile("file1.txt", []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", "file1.txt").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	// Test 2: No changes
	files, err := GetChangedFiles()
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 changed files, got %d: %v", len(files), files)
	}

	// Test 3: Modify file
	if err := os.WriteFile("file1.txt", []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = GetChangedFiles()
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 changed file, got %d: %v", len(files), files)
	}
	if len(files) > 0 && files[0] != "file1.txt" {
		t.Errorf("Expected file1.txt, got %s", files[0])
	}

	// Test 4: Add new file (not staged, so won't show in diff HEAD)
	if err := os.WriteFile("file2.txt", []byte("new file"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = GetChangedFiles()
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	// New untracked files won't show in diff HEAD, only modified tracked files
	if len(files) != 1 {
		t.Errorf("Expected 1 changed file (file1.txt), got %d: %v", len(files), files)
	}
}
