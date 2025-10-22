package temporal

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sort"
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

// === Merged from developer.go ===

// DeveloperCommitCount tracks commits per developer for a file
type DeveloperCommitCount struct {
	Email   string
	Name    string
	Count   int
	Commits []time.Time
}

// CalculateOwnership determines primary owner of each file
func CalculateOwnership(commits []Commit) map[string]*OwnershipHistory {
	// Map: filePath -> developer -> commit count
	fileDevCounts := make(map[string]map[string]*DeveloperCommitCount)

	// Count commits per developer per file
	for _, commit := range commits {
		email := strings.ToLower(commit.Email)

		for _, fileChange := range commit.FilesChanged {
			path := fileChange.Path

			if fileDevCounts[path] == nil {
				fileDevCounts[path] = make(map[string]*DeveloperCommitCount)
			}

			if devCount, exists := fileDevCounts[path][email]; exists {
				devCount.Count++
				devCount.Commits = append(devCount.Commits, commit.Timestamp)
			} else {
				fileDevCounts[path][email] = &DeveloperCommitCount{
					Email:   email,
					Name:    commit.Author,
					Count:   1,
					Commits: []time.Time{commit.Timestamp},
				}
			}
		}
	}

	// Calculate ownership for each file
	ownership := make(map[string]*OwnershipHistory)
	for filePath, devCounts := range fileDevCounts {
		ownership[filePath] = calculateFileOwnership(filePath, devCounts)
	}

	return ownership
}

// calculateFileOwnership determines ownership for a single file
func calculateFileOwnership(filePath string, devCounts map[string]*DeveloperCommitCount) *OwnershipHistory {
	// Convert to sorted slice
	developers := make([]*DeveloperCommitCount, 0, len(devCounts))
	for _, dev := range devCounts {
		developers = append(developers, dev)
	}

	// Sort by commit count descending
	sort.Slice(developers, func(i, j int) bool {
		return developers[i].Count > developers[j].Count
	})

	if len(developers) == 0 {
		return &OwnershipHistory{
			FilePath: filePath,
		}
	}

	history := &OwnershipHistory{
		FilePath:     filePath,
		CurrentOwner: developers[0].Email,
	}

	// Determine previous owner (second most commits, if different)
	if len(developers) > 1 {
		history.PreviousOwner = developers[1].Email

		// Calculate transition date (when current owner overtook previous)
		transitionDate := findTransitionDate(developers[0], developers[1])
		if !transitionDate.IsZero() {
			history.TransitionDate = transitionDate
			history.DaysSince = int(time.Since(transitionDate).Hours() / 24)
		}
	}

	return history
}

// findTransitionDate finds when devA overtook devB in commit count
func findTransitionDate(devA, devB *DeveloperCommitCount) time.Time {
	// Sort commits by timestamp
	allCommits := make([]struct {
		Time   time.Time
		IsDevA bool
	}, 0, len(devA.Commits)+len(devB.Commits))

	for _, t := range devA.Commits {
		allCommits = append(allCommits, struct {
			Time   time.Time
			IsDevA bool
		}{t, true})
	}

	for _, t := range devB.Commits {
		allCommits = append(allCommits, struct {
			Time   time.Time
			IsDevA bool
		}{t, false})
	}

	sort.Slice(allCommits, func(i, j int) bool {
		return allCommits[i].Time.Before(allCommits[j].Time)
	})

	// Find when devA first overtakes devB
	countA, countB := 0, 0
	var transitionTime time.Time

	for _, commit := range allCommits {
		if commit.IsDevA {
			countA++
		} else {
			countB++
		}

		// Track when A overtakes B
		if countA > countB && transitionTime.IsZero() {
			transitionTime = commit.Time
		}
	}

	return transitionTime
}

// GetOwnershipHistory returns ownership for a file (PUBLIC - Session C uses this)
// This function calculates ownership from git commit history
func GetOwnershipHistory(ctx context.Context, filePath string, commits []Commit) (*OwnershipHistory, error) {
	if len(commits) == 0 {
		return &OwnershipHistory{
			FilePath: filePath,
		}, nil
	}

	// Calculate ownership from all commits
	ownershipMap := CalculateOwnership(commits)

	// Return ownership for the specific file
	if ownership, exists := ownershipMap[filePath]; exists {
		return ownership, nil
	}

	// File not found in commits (might be a new file)
	return &OwnershipHistory{
		FilePath: filePath,
	}, nil
}
