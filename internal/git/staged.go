package git

import (
	"os/exec"
	"strings"
)

// GetStagedFiles returns list of files staged for commit
// Uses git diff --cached to detect files in staging area
// Reference: ux_pre_commit_hook.md - Staged file detection
func GetStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter out empty strings
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}

	return result, nil
}

// FindGitRoot returns the root directory of the git repository
// Uses git rev-parse --show-toplevel to find repo root
func FindGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
