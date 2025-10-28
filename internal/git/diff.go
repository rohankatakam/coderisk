package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetFileDiff returns the git diff for a specific file
// If the file is staged, it shows the staged diff
// Otherwise, it shows the working directory diff
func GetFileDiff(filePath string) (string, error) {
	// Try staged changes first
	cmd := exec.Command("git", "diff", "--cached", "--", filePath)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output), nil
	}

	// Fall back to working directory changes
	cmd = exec.Command("git", "diff", "--", filePath)
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return string(output), nil
}

// CountDiffLines counts the added and deleted lines in a git diff
// Returns (linesAdded, linesDeleted)
func CountDiffLines(diff string) (int, int) {
	if diff == "" {
		return 0, 0
	}

	lines := strings.Split(diff, "\n")
	linesAdded := 0
	linesDeleted := 0

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case '+':
			// Ignore +++ header lines
			if !strings.HasPrefix(line, "+++") {
				linesAdded++
			}
		case '-':
			// Ignore --- header lines
			if !strings.HasPrefix(line, "---") {
				linesDeleted++
			}
		}
	}

	return linesAdded, linesDeleted
}

// DetectChangeStatus determines the file's change status
// Returns one of: MODIFIED, ADDED, DELETED, RENAMED, COPIED
func DetectChangeStatus(filePath string) (string, error) {
	// Try staged status first
	cmd := exec.Command("git", "status", "--porcelain", "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}

	status := strings.TrimSpace(string(output))
	if status == "" {
		return "UNMODIFIED", nil
	}

	// Parse status code (first two characters)
	// Format: XY where X is staged, Y is unstaged
	// A = added, M = modified, D = deleted, R = renamed, C = copied
	if len(status) < 2 {
		return "UNKNOWN", nil
	}

	statusCode := status[0:2]

	// Check staged status (first character)
	switch statusCode[0] {
	case 'A':
		return "ADDED", nil
	case 'M':
		return "MODIFIED", nil
	case 'D':
		return "DELETED", nil
	case 'R':
		return "RENAMED", nil
	case 'C':
		return "COPIED", nil
	case '?':
		return "UNTRACKED", nil
	}

	// Check unstaged status (second character)
	switch statusCode[1] {
	case 'M':
		return "MODIFIED", nil
	case 'D':
		return "DELETED", nil
	}

	return "MODIFIED", nil // Default to modified
}

// TruncateDiffForPrompt truncates a diff to a reasonable size for LLM prompts
// Keeps the most important parts: first few hunks and function signatures
func TruncateDiffForPrompt(diff string, maxLines int) string {
	if diff == "" {
		return ""
	}

	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}

	// Strategy: Keep header + first N hunks
	var result []string
	hunkCount := 0
	maxHunks := 3
	inHunk := false

	for _, line := range lines {
		// Always include diff headers
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") {
			result = append(result, line)
			continue
		}

		// Track hunks (@@ markers)
		if strings.HasPrefix(line, "@@") {
			hunkCount++
			if hunkCount > maxHunks {
				result = append(result, "... (remaining hunks truncated)")
				break
			}
			inHunk = true
			result = append(result, line)
			continue
		}

		// Include lines from first N hunks
		if inHunk && hunkCount <= maxHunks {
			result = append(result, line)
		}

		// Stop if we've hit max lines
		if len(result) >= maxLines {
			result = append(result, "... (diff truncated)")
			break
		}
	}

	return strings.Join(result, "\n")
}
