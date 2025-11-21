package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
)

// EnsureRepoInitialized checks if the repository has been initialized
func EnsureRepoInitialized(ctx context.Context, db *sql.DB, repoID int64) error {
	// Check if repo exists in database
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM github_repositories WHERE id = $1)`
	err := db.QueryRowContext(ctx, query, repoID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check repository: %w", err)
	}

	if !exists {
		return fmt.Errorf("repository not initialized. Run 'crisk init' first")
	}

	// Check if code blocks exist
	var blockCount int
	query = `SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1`
	err = db.QueryRowContext(ctx, query, repoID).Scan(&blockCount)
	if err != nil {
		return fmt.Errorf("failed to check code blocks: %w", err)
	}

	if blockCount == 0 {
		return fmt.Errorf("no code blocks found. Repository may not be fully indexed. Run 'crisk init' to complete initialization")
	}

	return nil
}

// CheckGraphFreshness checks if the graph data is up to date with the repository
func CheckGraphFreshness(ctx context.Context, db *sql.DB, repoID int64) (warning string, err error) {
	// Get latest commit SHA from database
	var latestDBSHA string
	query := `
		SELECT sha
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY topological_index DESC
		LIMIT 1
	`
	err = db.QueryRowContext(ctx, query, repoID).Scan(&latestDBSHA)
	if err == sql.ErrNoRows {
		return "⚠️  No commits found in database. Run 'crisk init' to index the repository", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit from database: %w", err)
	}

	// Get current HEAD SHA from git
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current HEAD: %w", err)
	}
	currentSHA := strings.TrimSpace(string(output))

	// Compare
	if currentSHA != latestDBSHA {
		// Count commits behind
		cmd = exec.Command("git", "rev-list", "--count", latestDBSHA+".."+currentSHA)
		output, err := cmd.Output()
		if err == nil {
			commitsBehind := strings.TrimSpace(string(output))
			return fmt.Sprintf("⚠️  Graph data is %s commits behind. Run 'crisk init' to update", commitsBehind), nil
		}
		return "⚠️  Graph data may be outdated. Run 'crisk init' to update", nil
	}

	return "", nil
}

// HandleBlockNotFound provides helpful error message with suggestions
func HandleBlockNotFound(ctx context.Context, db *sql.DB, repoID int64, blockName, filePath string) error {
	// Try to find similar blocks
	similar, err := FindSimilarBlocks(ctx, db, repoID, blockName, filePath)
	if err != nil {
		return fmt.Errorf("function %q not found in %s", blockName, filePath)
	}

	if len(similar) > 0 {
		suggestions := make([]string, 0, len(similar))
		for _, block := range similar {
			suggestions = append(suggestions, fmt.Sprintf("  - %s (%s)", block.BlockName, block.Signature))
		}
		return fmt.Errorf("function %q not found in %s. Did you mean:\n%s",
			blockName, filePath, strings.Join(suggestions, "\n"))
	}

	return fmt.Errorf("function %q not found in %s. The file may not have been analyzed yet", blockName, filePath)
}

// FormatRepoNotFoundError provides a helpful message when repo is not in database
func FormatRepoNotFoundError(owner, repo string) error {
	return fmt.Errorf(`Repository %s/%s not found in database.

To initialize this repository:
  1. Make sure you're in the repository directory
  2. Run: crisk init

This will index the repository and make it available for analysis.`, owner, repo)
}
