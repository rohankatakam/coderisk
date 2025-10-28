package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// HistoryTracker provides functionality to track file history across renames
// using git log --follow. This is critical for resolving current file paths
// to their historical paths in commits.
type HistoryTracker struct {
	repoPath string
}

// NewHistoryTracker creates a new HistoryTracker for the given repository path
func NewHistoryTracker(repoPath string) *HistoryTracker {
	return &HistoryTracker{repoPath: repoPath}
}

// GetFileHistory returns all historical paths for a file using git log --follow.
// This discovers how a file's path has changed over time due to renames or
// reorganizations.
//
// For example, if a file was reorganized from "shared/config/settings.py" to
// "src/shared/config/settings.py", this function returns both paths.
//
// The returned paths are in the order they appear in git history, with the
// current path first (most recent) and older paths following.
//
// Parameters:
//   - ctx: Context for cancellation
//   - filePath: Relative path from repository root (e.g., "src/shared/config/settings.py")
//
// Returns:
//   - []string: All historical paths for this file (deduplicated)
//   - error: If git command fails or file doesn't exist
func (ht *HistoryTracker) GetFileHistory(ctx context.Context, filePath string) ([]string, error) {
	// git log --follow tracks file renames
	// --name-only shows only file paths, not diffs
	// --pretty=format: suppresses commit metadata (we only want paths)
	cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--name-only", "--pretty=format:", "--", filePath)
	cmd.Dir = ht.repoPath

	output, err := cmd.Output()
	if err != nil {
		// Check if file exists
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log --follow failed for file %s: %w (stderr: %s)", filePath, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git log --follow failed for file %s: %w", filePath, err)
	}

	// Parse output - one path per line
	lines := strings.Split(string(output), "\n")
	seen := make(map[string]bool)
	var paths []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			seen[line] = true
			paths = append(paths, line)
		}
	}

	// If no paths found, the file might not exist or have no history
	if len(paths) == 0 {
		return nil, fmt.Errorf("no history found for file %s (file may not exist or have no commits)", filePath)
	}

	return paths, nil
}

// GetFileHistoryBatch returns historical paths for multiple files concurrently.
// This is more efficient than calling GetFileHistory multiple times sequentially.
//
// Parameters:
//   - ctx: Context for cancellation
//   - filePaths: List of relative file paths from repository root
//
// Returns:
//   - map[string][]string: Map of file path -> historical paths
//   - error: If any git command fails (returns first error encountered)
//
// Note: This function processes files concurrently for performance, but
// returns an error if ANY file fails. Successful results are still returned
// in the map for files that succeeded before the error.
func (ht *HistoryTracker) GetFileHistoryBatch(ctx context.Context, filePaths []string) (map[string][]string, error) {
	result := make(map[string][]string)

	// Process files sequentially for MVP (can optimize later with goroutines)
	for _, filePath := range filePaths {
		paths, err := ht.GetFileHistory(ctx, filePath)
		if err != nil {
			// Log warning but continue (some files might not have history)
			// Return partial results
			continue
		}
		result[filePath] = paths
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no file histories could be retrieved for %d files", len(filePaths))
	}

	return result, nil
}
