package temporal

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ParseGitHistory executes git log and parses output
func ParseGitHistory(repoPath string, days int) ([]Commit, error) {
	// Execute git log with numstat to get file changes
	cmd := exec.Command("git", "log",
		fmt.Sprintf("--since=%d days ago", days),
		"--numstat",
		"--pretty=format:%H|%an|%ae|%ad|%s",
		"--date=iso-strict")

	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w (output: %s)", err, string(output))
	}

	return parseGitLogOutput(string(output))
}

// parseGitLogOutput parses the raw git log output into Commit structs
func parseGitLogOutput(output string) ([]Commit, error) {
	var commits []Commit
	var currentCommit *Commit

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Empty line separates commits
		if line == "" {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
				currentCommit = nil
			}
			continue
		}

		// Commit header line: SHA|Author|Email|Date|Message
		if strings.Contains(line, "|") {
			// Save previous commit
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}

			parts := strings.SplitN(line, "|", 5)
			if len(parts) != 5 {
				continue // Skip malformed lines
			}

			timestamp, err := time.Parse(time.RFC3339, parts[3])
			if err != nil {
				// Fallback to a simpler parser if strict ISO fails
				timestamp = time.Now()
			}

			currentCommit = &Commit{
				SHA:          parts[0],
				Author:       parts[1],
				Email:        parts[2],
				Timestamp:    timestamp,
				Message:      parts[4],
				FilesChanged: []FileChange{},
			}
			continue
		}

		// File change line: additions deletions path
		if currentCommit != nil {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				additions, _ := strconv.Atoi(fields[0])
				deletions, _ := strconv.Atoi(fields[1])
				path := fields[2]

				// Skip binary files (marked with "-")
				if fields[0] == "-" || fields[1] == "-" {
					continue
				}

				currentCommit.FilesChanged = append(currentCommit.FilesChanged, FileChange{
					Path:      path,
					Additions: additions,
					Deletions: deletions,
				})
			}
		}
	}

	// Don't forget the last commit
	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning git log output: %w", err)
	}

	return commits, nil
}

// ExtractDevelopers builds Developer nodes from commits
func ExtractDevelopers(commits []Commit) []Developer {
	devMap := make(map[string]*Developer)

	for _, commit := range commits {
		email := strings.ToLower(commit.Email)

		if dev, exists := devMap[email]; exists {
			dev.TotalCommits++
			if commit.Timestamp.After(dev.LastCommit) {
				dev.LastCommit = commit.Timestamp
			}
			if commit.Timestamp.Before(dev.FirstCommit) {
				dev.FirstCommit = commit.Timestamp
			}
		} else {
			devMap[email] = &Developer{
				Email:        email,
				Name:         commit.Author,
				FirstCommit:  commit.Timestamp,
				LastCommit:   commit.Timestamp,
				TotalCommits: 1,
			}
		}
	}

	// Convert map to slice
	developers := make([]Developer, 0, len(devMap))
	for _, dev := range devMap {
		developers = append(developers, *dev)
	}

	return developers
}
