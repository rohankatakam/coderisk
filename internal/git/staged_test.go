package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetStagedFiles(t *testing.T) {
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

	// Test 1: No staged files
	files, err := GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 staged files, got %d", len(files))
	}

	// Test 2: Create and stage a file
	testFile := "test.go"
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "add", testFile).Run(); err != nil {
		t.Fatal(err)
	}

	files, err = GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 staged file, got %d", len(files))
	}
	if len(files) > 0 && files[0] != testFile {
		t.Errorf("Expected file %s, got %s", testFile, files[0])
	}

	// Test 3: Multiple staged files
	testFile2 := "test2.go"
	if err := os.WriteFile(testFile2, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "add", testFile2).Run(); err != nil {
		t.Fatal(err)
	}

	files, err = GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 staged files, got %d", len(files))
	}
}

func TestFindGitRoot(t *testing.T) {
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

	// Test 1: Not in a git repo
	_, err = FindGitRoot()
	if err == nil {
		t.Error("Expected error when not in git repo")
	}

	// Test 2: In a git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("git not available")
	}

	root, err := FindGitRoot()
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}

	// Resolve both paths to compare (use EvalSymlinks for macOS /var -> /private/var)
	expectedRoot, _ := filepath.EvalSymlinks(tmpDir)
	actualRoot, _ := filepath.EvalSymlinks(root)
	if actualRoot != expectedRoot {
		t.Errorf("Expected root %s, got %s", expectedRoot, actualRoot)
	}

	// Test 3: In subdirectory of git repo
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}

	root, err = FindGitRoot()
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}

	actualRoot, _ = filepath.EvalSymlinks(root)
	if actualRoot != expectedRoot {
		t.Errorf("From subdirectory, expected root %s, got %s", expectedRoot, actualRoot)
	}
}
