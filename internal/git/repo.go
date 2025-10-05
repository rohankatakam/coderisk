package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// DetectGitRepo checks if current directory is a git repository
// Uses git rev-parse to verify we're inside a working tree
// Reference: NEXT_STEPS.md - Task 1 (Git Integration Functions)
func DetectGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	return nil
}

// ParseRepoURL extracts org and repo name from git remote URL
// Supports multiple URL formats:
//   - HTTPS: https://github.com/owner/repo.git
//   - SSH: git@github.com:owner/repo.git
//   - Git protocol: git://github.com/owner/repo.git
//
// Reference: NEXT_STEPS.md - Task 1 (Git Integration Functions)
func ParseRepoURL(remoteURL string) (org, repo string, err error) {
	// Remove .git suffix if present
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Try HTTPS format: https://github.com/owner/repo or http://...
	httpsRegex := regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)`)
	if matches := httpsRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// Try SSH format: git@github.com:owner/repo
	sshRegex := regexp.MustCompile(`git@[^:]+:([^/]+)/([^/]+)`)
	if matches := sshRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// Try git protocol: git://github.com/owner/repo
	gitRegex := regexp.MustCompile(`git://[^/]+/([^/]+)/([^/]+)`)
	if matches := gitRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("unrecognized git URL format: %s", remoteURL)
}

// GetChangedFiles returns list of files changed in working directory
// Uses git diff to find modified files compared to HEAD
// Reference: NEXT_STEPS.md - Task 1 (Git Integration Functions)
func GetChangedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the URL of the git remote (typically 'origin')
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentCommitSHA returns the SHA of the current commit
func GetCurrentCommitSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetAuthorEmail returns the configured git user email
func GetAuthorEmail() (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
