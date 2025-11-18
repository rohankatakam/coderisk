package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RepoResolver resolves repository ID from git remote URL or absolute path
type RepoResolver struct {
	pgPool *pgxpool.Pool
}

// NewRepoResolver creates a new repository resolver
func NewRepoResolver(pgPool *pgxpool.Pool) *RepoResolver {
	return &RepoResolver{pgPool: pgPool}
}

// ResolveRepoID resolves the repository ID from the given repo_root directory
// Uses git remote to extract owner/repo, then looks up in github_repositories table
// Falls back to absolute_path matching if git remote fails
func (r *RepoResolver) ResolveRepoID(ctx context.Context, repoRoot string) (int, error) {
	if repoRoot == "" {
		return 0, fmt.Errorf("repo_root is required")
	}

	// Strategy 1: Try to get git remote URL
	repoID, err := r.resolveFromGitRemote(ctx, repoRoot)
	if err == nil {
		return repoID, nil
	}

	// Strategy 2: Fall back to absolute path matching
	repoID, err = r.resolveFromAbsolutePath(ctx, repoRoot)
	if err == nil {
		return repoID, nil
	}

	return 0, fmt.Errorf("failed to resolve repo_id for %s: not found in database", repoRoot)
}

// resolveFromGitRemote extracts owner/repo from git remote and looks up in database
func (r *RepoResolver) resolveFromGitRemote(ctx context.Context, repoRoot string) (int, error) {
	// Get git remote origin URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("git remote failed: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return 0, fmt.Errorf("no git remote found")
	}

	// Extract owner/repo from URL
	// Handles:
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git
	// - https://github.com/owner/repo
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return 0, err
	}

	// Look up in database by owner/name
	query := `
		SELECT id
		FROM github_repositories
		WHERE owner = $1 AND name = $2
		LIMIT 1
	`

	var repoID int
	err = r.pgPool.QueryRow(ctx, query, owner, repo).Scan(&repoID)
	if err != nil {
		return 0, fmt.Errorf("repository %s/%s not found in database: %w", owner, repo, err)
	}

	return repoID, nil
}

// resolveFromAbsolutePath looks up repository by absolute_path in database
func (r *RepoResolver) resolveFromAbsolutePath(ctx context.Context, repoRoot string) (int, error) {
	query := `
		SELECT id
		FROM github_repositories
		WHERE absolute_path = $1
		LIMIT 1
	`

	var repoID int
	err := r.pgPool.QueryRow(ctx, query, repoRoot).Scan(&repoID)
	if err != nil {
		return 0, fmt.Errorf("repository at path %s not found in database: %w", repoRoot, err)
	}

	return repoID, nil
}

// parseGitHubURL extracts owner and repo name from a GitHub URL
func parseGitHubURL(url string) (owner, repo string, err error) {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Pattern for HTTPS: https://github.com/owner/repo
	httpsPattern := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// Pattern for SSH: git@github.com:owner/repo
	sshPattern := regexp.MustCompile(`git@github\.com:([^/]+)/(.+)`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("unable to parse GitHub URL: %s", url)
}
