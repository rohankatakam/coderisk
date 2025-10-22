package temporal

import (
	"context"
	"sort"
)

// DEPRECATED: CalculateCoChanges is the OLD pre-calculation approach
// This function is being replaced by dynamic co-change queries in the graph
// See: simplified_graph_schema.md - Co-change computed at query time
//
// Current usage:
//   - internal/ingestion/processor.go (old graph builder, being removed)
//   - internal/agent/adapters.go (temporary, will use graph queries)
//
// Replacement: Use graph queries with [:ON_BRANCH] filtering
//   Query: MATCH (f:File)<-[:MODIFIES]-(c:Commit)-[:ON_BRANCH]->(b:Branch {is_default: true})
//
// DO NOT USE for new code - compute co-change dynamically instead
//
// CalculateCoChanges finds files that frequently change together
func CalculateCoChanges(commits []Commit, minFrequency float64) []CoChangeResult {
	// Track pairs of files that change together
	pairCounts := make(map[string]int)
	fileCounts := make(map[string]int)

	// For each commit, get all pairs of files changed
	for _, commit := range commits {
		files := commit.FilesChanged
		if len(files) < 2 {
			continue
		}

		// Count individual file appearances
		for _, file := range files {
			fileCounts[file.Path]++
		}

		// Generate all pairs for this commit
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				fileA, fileB := files[i].Path, files[j].Path

				// Normalize pair order (alphabetical) to avoid duplicates
				if fileA > fileB {
					fileA, fileB = fileB, fileA
				}

				pairKey := fileA + "|" + fileB
				pairCounts[pairKey]++
			}
		}
	}

	// Calculate frequencies
	var results []CoChangeResult
	for pairKey, coChangeCount := range pairCounts {
		// Parse pair key
		parts := splitPairKey(pairKey)
		if len(parts) != 2 {
			continue
		}
		fileA, fileB := parts[0], parts[1]

		// Calculate frequency: how often they appear together vs separately
		// frequency = co_changes / max(fileA_count, fileB_count)
		// This gives us "when fileA changes, how often does fileB change too?"
		maxCount := max(fileCounts[fileA], fileCounts[fileB])
		if maxCount == 0 {
			continue
		}

		frequency := float64(coChangeCount) / float64(maxCount)

		// Filter by minimum frequency
		if frequency >= minFrequency {
			results = append(results, CoChangeResult{
				FileA:      fileA,
				FileB:      fileB,
				Frequency:  frequency,
				CoChanges:  coChangeCount,
				WindowDays: 90, // hardcoded for now, could be parameterized
			})
		}
	}

	// Sort by frequency descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Frequency > results[j].Frequency
	})

	return results
}

// GetCoChangedFiles returns files that change with target (PUBLIC - Session C uses this)
// This function calculates co-changes from commit history
func GetCoChangedFiles(ctx context.Context, filePath string, minFrequency float64, commits []Commit) ([]CoChangeResult, error) {
	if len(commits) == 0 {
		return []CoChangeResult{}, nil
	}

	// Calculate co-changes from all commits
	allResults := CalculateCoChanges(commits, minFrequency)

	// Filter to only results involving the target file
	var filtered []CoChangeResult
	for _, result := range allResults {
		if result.FileA == filePath || result.FileB == filePath {
			// Normalize so the target file is always FileA
			if result.FileB == filePath {
				result.FileA, result.FileB = result.FileB, result.FileA
			}
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// Helper functions

func splitPairKey(key string) []string {
	// Simple split on "|"
	result := make([]string, 0, 2)
	start := 0
	for i, c := range key {
		if c == '|' {
			result = append(result, key[start:i])
			start = i + 1
		}
	}
	result = append(result, key[start:])
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
