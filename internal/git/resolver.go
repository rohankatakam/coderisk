package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// FileResolver bridges current file paths to historical graph data
// Implements 2-level resolution strategy:
//   Level 1: Exact match (file path unchanged)
//   Level 2: Git log --follow (file was renamed/moved)
type FileResolver struct {
	repoPath    string
	graphClient GraphQueryer
}

// GraphQueryer interface for querying Neo4j
type GraphQueryer interface {
	ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
}

// FileMatch represents a resolved historical path
type FileMatch struct {
	HistoricalPath string  // Path as stored in graph
	Confidence     float64 // Confidence score (1.0 = exact, 0.95 = git follow)
	Method         string  // Resolution method: "exact", "git-follow"
}

// NewFileResolver creates a new file resolver
func NewFileResolver(repoPath string, graphClient GraphQueryer) *FileResolver {
	return &FileResolver{
		repoPath:    repoPath,
		graphClient: graphClient,
	}
}

// Resolve finds historical paths for a current file path
// Returns array of matches sorted by confidence (highest first)
// CRITICAL FIX: Always check BOTH exact AND git-follow to capture rename history
func (r *FileResolver) Resolve(ctx context.Context, currentPath string) ([]FileMatch, error) {
	var allMatches []FileMatch

	// Level 1: Exact match
	exactMatches, err := r.exactMatch(ctx, currentPath)
	if err == nil && len(exactMatches) > 0 {
		allMatches = append(allMatches, exactMatches...)
	}

	// Level 2: Git log --follow (ALWAYS check, even if exact match found)
	// This captures historical paths before renames/moves
	gitFollowMatches, err := r.gitFollowMatch(ctx, currentPath)
	if err == nil && len(gitFollowMatches) > 0 {
		// Deduplicate: only add paths not already in allMatches
		for _, gitMatch := range gitFollowMatches {
			isDuplicate := false
			for _, existing := range allMatches {
				if existing.HistoricalPath == gitMatch.HistoricalPath {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				allMatches = append(allMatches, gitMatch)
			}
		}
	}

	// Return all unique historical paths (current + renamed)
	return allMatches, nil
}

// BatchResolve resolves multiple files in parallel
// Returns map of currentPath -> []FileMatch
func (r *FileResolver) BatchResolve(ctx context.Context, currentPaths []string) (map[string][]FileMatch, error) {
	results := make(map[string][]FileMatch)
	mu := sync.Mutex{}

	var wg sync.WaitGroup
	for _, path := range currentPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			matches, err := r.Resolve(ctx, p)
			if err == nil {
				mu.Lock()
				results[p] = matches
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()
	return results, nil
}

// exactMatch checks if current path exists in graph (Level 1: 100% confidence)
func (r *FileResolver) exactMatch(ctx context.Context, currentPath string) ([]FileMatch, error) {
	query := `
		MATCH (f:File)
		WHERE f.path = $path
		RETURN f.path as path
		LIMIT 1
	`

	results, err := r.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"path": currentPath,
	})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil // No exact match
	}

	path, ok := results[0]["path"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid path type in graph result")
	}

	return []FileMatch{{
		HistoricalPath: path,
		Confidence:     1.0,
		Method:         "exact",
	}}, nil
}

// gitFollowMatch uses git log --follow to find historical paths (Level 2: 95% confidence)
func (r *FileResolver) gitFollowMatch(ctx context.Context, currentPath string) ([]FileMatch, error) {
	// Execute git log --follow to get all historical paths for this file
	cmd := exec.Command("git", "log", "--follow", "--name-only", "--pretty=format:", "--", currentPath)
	cmd.Dir = r.repoPath

	output, err := cmd.Output()
	if err != nil {
		// Git command failed (file may not exist yet)
		return nil, fmt.Errorf("git log --follow failed: %w", err)
	}

	// Parse unique file paths from output
	historicalPaths := r.parseUniquePaths(output)
	if len(historicalPaths) == 0 {
		return nil, nil
	}

	// Check which historical paths exist in our graph
	query := `
		MATCH (f:File)
		WHERE f.path IN $paths
		RETURN f.path as path
	`

	results, err := r.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths": historicalPaths,
	})
	if err != nil {
		return nil, fmt.Errorf("graph query failed: %w", err)
	}

	// Build matches from results
	var matches []FileMatch
	for _, row := range results {
		if path, ok := row["path"].(string); ok {
			matches = append(matches, FileMatch{
				HistoricalPath: path,
				Confidence:     0.95, // Git's rename detection is very accurate
				Method:         "git-follow",
			})
		}
	}

	return matches, nil
}

// parseUniquePaths extracts unique file paths from git log output
func (r *FileResolver) parseUniquePaths(output []byte) []string {
	lines := strings.Split(string(output), "\n")
	seen := make(map[string]bool)
	var unique []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			seen[line] = true
			unique = append(unique, line)
		}
	}

	return unique
}

// ResolveToSinglePath returns the best historical path for a file
// Useful when you need exactly one path for queries
func (r *FileResolver) ResolveToSinglePath(ctx context.Context, currentPath string) (string, float64, error) {
	matches, err := r.Resolve(ctx, currentPath)
	if err != nil {
		return "", 0, err
	}

	if len(matches) == 0 {
		// No match found - return current path with low confidence
		return currentPath, 0.3, nil
	}

	// Return highest confidence match (first in array)
	return matches[0].HistoricalPath, matches[0].Confidence, nil
}

// ResolveToAllPaths returns all historical paths for a file (including current)
// Useful for queries that should search across rename history
func (r *FileResolver) ResolveToAllPaths(ctx context.Context, currentPath string) ([]string, error) {
	matches, err := r.Resolve(ctx, currentPath)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		// No historical data - return just the current path
		return []string{currentPath}, nil
	}

	// Extract all historical paths
	paths := make([]string, len(matches))
	for i, match := range matches {
		paths[i] = match.HistoricalPath
	}

	return paths, nil
}
