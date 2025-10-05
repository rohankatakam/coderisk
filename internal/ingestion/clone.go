package ingestion

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRepository performs shallow clone for Layer 1 parsing
// Stores repos in ~/.coderisk/repos/<repo-hash>/ for persistence
// Returns path to cloned repository
func CloneRepository(ctx context.Context, url string) (string, error) {
	// Generate unique hash for storage
	hash := generateRepoHash(url)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	repoPath := filepath.Join(homeDir, ".coderisk", "repos", hash)

	// Check if already cloned
	if _, err := os.Stat(repoPath); err == nil {
		// Repository already exists, verify it's valid
		if isValidGitRepo(repoPath) {
			return repoPath, nil
		}
		// Invalid repo, remove and re-clone
		os.RemoveAll(repoPath)
	}

	// Create parent directory
	parentDir := filepath.Dir(repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Shallow clone (--depth 1) for speed
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1",
		"--single-branch",
		url,
		repoPath,
	)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	return repoPath, nil
}

// CloneRepositoryWithBranch clones a specific branch
func CloneRepositoryWithBranch(ctx context.Context, url string, branch string) (string, error) {
	hash := generateRepoHash(url + "#" + branch)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	repoPath := filepath.Join(homeDir, ".coderisk", "repos", hash)

	// Check if already cloned
	if _, err := os.Stat(repoPath); err == nil {
		if isValidGitRepo(repoPath) {
			return repoPath, nil
		}
		os.RemoveAll(repoPath)
	}

	// Create parent directory
	parentDir := filepath.Dir(repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Clone specific branch
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1",
		"--single-branch",
		"--branch", branch,
		url,
		repoPath,
	)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	return repoPath, nil
}

// generateRepoHash creates a unique hash from repository URL
func generateRepoHash(url string) string {
	// Normalize URL (remove trailing .git, etc.)
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	// Generate SHA256 hash
	h := sha256.New()
	h.Write([]byte(url))
	hashBytes := h.Sum(nil)

	// Use first 16 characters of hex
	return fmt.Sprintf("%x", hashBytes)[:16]
}

// isValidGitRepo checks if directory is a valid git repository
func isValidGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ParseRepoURL extracts org/repo from GitHub URL
// Supports:
// - https://github.com/org/repo
// - git@github.com:org/repo.git
// - org/repo (shorthand)
func ParseRepoURL(url string) (org string, repo string, err error) {
	url = strings.TrimSpace(url)

	// Handle git@ format
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
	}

	// Handle https format
	if strings.HasPrefix(url, "https://github.com/") {
		url = strings.TrimPrefix(url, "https://github.com/")
	}

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Split org/repo
	parts := strings.Split(url, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected org/repo)", url)
	}

	return parts[0], parts[1], nil
}

// BuildGitHubURL converts org/repo to full GitHub URL
func BuildGitHubURL(org, repo string) string {
	return fmt.Sprintf("https://github.com/%s/%s", org, repo)
}
