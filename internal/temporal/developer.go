package temporal

import (
	"context"
	"sort"
	"strings"
	"time"
)

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
		Time  time.Time
		IsDevA bool
	}, 0, len(devA.Commits)+len(devB.Commits))

	for _, t := range devA.Commits {
		allCommits = append(allCommits, struct {
			Time  time.Time
			IsDevA bool
		}{t, true})
	}

	for _, t := range devB.Commits {
		allCommits = append(allCommits, struct {
			Time  time.Time
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
