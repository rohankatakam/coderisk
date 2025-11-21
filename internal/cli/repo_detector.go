package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectRepoID finds the repo_id from the current git directory
func DetectRepoID(ctx context.Context, db *sql.DB) (int64, string, error) {
	// Get git remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return 0, "", fmt.Errorf("not a git repository or no remote configured: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return 0, "", fmt.Errorf("remote.origin.url is empty")
	}

	// Parse owner/repo from URL
	owner, repo, err := parseGitURL(remoteURL)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse git remote URL %q: %w", remoteURL, err)
	}

	// Query database for repo_id
	var repoID int64
	query := `
		SELECT id
		FROM github_repositories
		WHERE owner = $1 AND name = $2
		LIMIT 1
	`
	err = db.QueryRowContext(ctx, query, owner, repo).Scan(&repoID)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("repository %s/%s not found in database. Run 'crisk init' first", owner, repo)
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to query repository: %w", err)
	}

	return repoID, fmt.Sprintf("%s/%s", owner, repo), nil
}

// GetCurrentUserEmail returns the git user email from config
func GetCurrentUserEmail() (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user.email: %w", err)
	}

	email := strings.TrimSpace(string(output))
	if email == "" {
		return "", fmt.Errorf("git user.email is not configured")
	}

	return email, nil
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root: %w", err)
	}

	root := strings.TrimSpace(string(output))
	return root, nil
}

// GetRelativePath returns the path relative to the repo root
func GetRelativePath(path string) (string, error) {
	// If the path is already relative, just return it
	if !filepath.IsAbs(path) {
		return path, nil
	}

	root, err := GetRepoRoot()
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

// parseGitURL extracts owner and repo name from various git URL formats
func parseGitURL(url string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
		}
		path := strings.TrimSuffix(parts[1], ".git")
		pathParts := strings.Split(path, "/")
		if len(pathParts) != 2 {
			return "", "", fmt.Errorf("invalid repository path in SSH URL")
		}
		return pathParts[0], pathParts[1], nil
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		url = strings.TrimSuffix(url, ".git")
		parts := strings.Split(url, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid HTTPS URL format")
		}
		repo = parts[len(parts)-1]
		owner = parts[len(parts)-2]
		return owner, repo, nil
	}

	return "", "", fmt.Errorf("unsupported URL format: %s", url)
}
