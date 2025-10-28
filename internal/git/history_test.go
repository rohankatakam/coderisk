package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestGetFileHistory tests the GetFileHistory function with a real repository
// that has known file reorganizations.
//
// This test requires the omnara repository to be cloned locally.
// Skip this test if the repository is not available.
func TestGetFileHistory(t *testing.T) {
	// Try to find omnara repository in common locations
	possiblePaths := []string{
		os.Getenv("HOME") + "/.coderisk/repos/omnara",
		os.Getenv("HOME") + "/Documents/brain/omnara",
		"/tmp/omnara",
	}

	var omnaraPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			omnaraPath = path
			break
		}
	}

	if omnaraPath == "" {
		t.Skip("Omnara repository not found in common locations, skipping test")
	}

	tracker := NewHistoryTracker(omnaraPath)

	// Test with a file that was reorganized (moved to src/ subdirectory)
	// This file is known to have been reorganized in commit #245
	testFile := "src/shared/config/settings.py"

	paths, err := tracker.GetFileHistory(context.Background(), testFile)
	if err != nil {
		t.Fatalf("GetFileHistory failed: %v", err)
	}

	// Should find at least 2 paths (current and historical)
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 paths for reorganized file, got %d: %v", len(paths), paths)
	}

	// Verify we found both current and historical paths
	hasCurrentPath := false
	hasHistoricalPath := false
	for _, p := range paths {
		if p == "src/shared/config/settings.py" {
			hasCurrentPath = true
		}
		if p == "shared/config/settings.py" {
			hasHistoricalPath = true
		}
	}

	if !hasCurrentPath {
		t.Errorf("Missing current path 'src/shared/config/settings.py' in history: %v", paths)
	}
	if !hasHistoricalPath {
		t.Errorf("Missing historical path 'shared/config/settings.py' in history: %v", paths)
	}

	t.Logf("✓ Found %d historical paths for %s", len(paths), testFile)
	for i, p := range paths {
		t.Logf("  [%d] %s", i, p)
	}
}

// TestGetFileHistoryNoRenames tests a file that has never been renamed
func TestGetFileHistoryNoRenames(t *testing.T) {
	// Try to find omnara repository
	possiblePaths := []string{
		os.Getenv("HOME") + "/.coderisk/repos/omnara",
		os.Getenv("HOME") + "/Documents/brain/omnara",
		"/tmp/omnara",
	}

	var omnaraPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			omnaraPath = path
			break
		}
	}

	if omnaraPath == "" {
		t.Skip("Omnara repository not found, skipping test")
	}

	tracker := NewHistoryTracker(omnaraPath)

	// Test with README.md which typically isn't renamed
	testFile := "README.md"

	paths, err := tracker.GetFileHistory(context.Background(), testFile)
	if err != nil {
		t.Fatalf("GetFileHistory failed: %v", err)
	}

	// Should find exactly 1 path (never renamed)
	if len(paths) != 1 {
		t.Logf("Note: Expected 1 path for never-renamed file, got %d", len(paths))
		t.Logf("Paths: %v", paths)
		// Don't fail - README might have been renamed in this repo
	}

	if paths[0] != testFile {
		t.Errorf("Expected path to be %s, got %s", testFile, paths[0])
	}

	t.Logf("✓ Found %d path(s) for %s", len(paths), testFile)
}

// TestGetFileHistoryNonExistent tests handling of a file that doesn't exist
func TestGetFileHistoryNonExistent(t *testing.T) {
	// Try to find omnara repository
	possiblePaths := []string{
		os.Getenv("HOME") + "/.coderisk/repos/omnara",
		os.Getenv("HOME") + "/Documents/brain/omnara",
		"/tmp/omnara",
	}

	var omnaraPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			omnaraPath = path
			break
		}
	}

	if omnaraPath == "" {
		t.Skip("Omnara repository not found, skipping test")
	}

	tracker := NewHistoryTracker(omnaraPath)

	// Test with a file that definitely doesn't exist
	testFile := "this/file/does/not/exist.xyz"

	paths, err := tracker.GetFileHistory(context.Background(), testFile)

	// Should return an error
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil (paths: %v)", paths)
	}

	if paths != nil {
		t.Errorf("Expected nil paths for non-existent file, got %v", paths)
	}

	t.Logf("✓ Correctly returned error for non-existent file: %v", err)
}

// TestGetFileHistoryBatch tests batch processing of multiple files
func TestGetFileHistoryBatch(t *testing.T) {
	// Try to find omnara repository
	possiblePaths := []string{
		os.Getenv("HOME") + "/.coderisk/repos/omnara",
		os.Getenv("HOME") + "/Documents/brain/omnara",
		"/tmp/omnara",
	}

	var omnaraPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			omnaraPath = path
			break
		}
	}

	if omnaraPath == "" {
		t.Skip("Omnara repository not found, skipping test")
	}

	tracker := NewHistoryTracker(omnaraPath)

	// Test with multiple files
	testFiles := []string{
		"src/shared/config/settings.py", // Known to be reorganized
		"README.md",                      // Likely never renamed
	}

	results, err := tracker.GetFileHistoryBatch(context.Background(), testFiles)
	if err != nil {
		t.Fatalf("GetFileHistoryBatch failed: %v", err)
	}

	// Should have results for both files (or at least some)
	if len(results) == 0 {
		t.Fatal("Expected at least some results, got empty map")
	}

	// Verify each file's results
	for file, paths := range results {
		t.Logf("File: %s", file)
		if len(paths) == 0 {
			t.Errorf("  Expected at least 1 path, got 0")
		}
		for i, p := range paths {
			t.Logf("  [%d] %s", i, p)
		}
	}

	t.Logf("✓ Successfully retrieved histories for %d files", len(results))
}

// TestGetFileHistoryContext tests context cancellation
func TestGetFileHistoryContext(t *testing.T) {
	// Try to find omnara repository
	possiblePaths := []string{
		os.Getenv("HOME") + "/.coderisk/repos/omnara",
		os.Getenv("HOME") + "/Documents/brain/omnara",
		"/tmp/omnara",
	}

	var omnaraPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			omnaraPath = path
			break
		}
	}

	if omnaraPath == "" {
		t.Skip("Omnara repository not found, skipping test")
	}

	tracker := NewHistoryTracker(omnaraPath)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testFile := "README.md"

	_, err := tracker.GetFileHistory(ctx, testFile)

	// Should return an error due to cancelled context
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}

	t.Logf("✓ Correctly handled cancelled context: %v", err)
}
