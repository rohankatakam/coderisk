package git

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
)

// TopologicalSorter handles git topological ordering operations
type TopologicalSorter struct {
	repoPath string
}

// NewTopologicalSorter creates a new topological sorter for the given repo
func NewTopologicalSorter(repoPath string) *TopologicalSorter {
	return &TopologicalSorter{
		repoPath: repoPath,
	}
}

// ComputeTopologicalOrder computes topological ordering for all commits
// Returns map of commit SHA -> topological_index (0-based)
// Parents always have lower index than children
func (ts *TopologicalSorter) ComputeTopologicalOrder(ctx context.Context) (map[string]int, error) {
	// Execute: git rev-list --topo-order --reverse HEAD
	// This gives us commits in topological order: parents before children
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--topo-order", "--reverse", "HEAD")
	cmd.Dir = ts.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list failed: %w", err)
	}

	// Parse output and build index map
	result := make(map[string]int)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	index := 0

	for scanner.Scan() {
		sha := strings.TrimSpace(scanner.Text())
		if sha != "" {
			result[sha] = index
			index++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse git output: %w", err)
	}

	return result, nil
}

// ComputeParentSHAsHash computes SHA256 hash of all commit parent relationships
// Used for force-push detection
func (ts *TopologicalSorter) ComputeParentSHAsHash(ctx context.Context) (string, error) {
	// Execute: git log --format=%H:%P (commit:parent1 parent2...)
	cmd := exec.CommandContext(ctx, "git", "log", "--format=%H:%P", "HEAD")
	cmd.Dir = ts.repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	// Compute SHA256 of the parent relationships
	hash := sha256.Sum256(output)
	return fmt.Sprintf("%x", hash), nil
}

// GetCommitParents retrieves parent SHAs for a given commit
func (ts *TopologicalSorter) GetCommitParents(ctx context.Context, commitSHA string) ([]string, error) {
	// Execute: git log -1 --format=%P <sha>
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%P", commitSHA)
	cmd.Dir = ts.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed for %s: %w", commitSHA, err)
	}

	parentString := strings.TrimSpace(string(output))
	if parentString == "" {
		// No parents (root commit)
		return []string{}, nil
	}

	parents := strings.Fields(parentString)
	return parents, nil
}

// ValidateTopologicalOrdering checks if topological ordering is still valid
// Returns true if ordering is valid, false if recomputation needed
func (ts *TopologicalSorter) ValidateTopologicalOrdering(ctx context.Context, existingHash string) (bool, string, error) {
	currentHash, err := ts.ComputeParentSHAsHash(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to compute current hash: %w", err)
	}

	if existingHash == "" {
		// No existing hash - first time computing
		return true, currentHash, nil
	}

	// Compare hashes
	if currentHash != existingHash {
		// Hash mismatch - force-push detected
		return false, currentHash, nil
	}

	return true, currentHash, nil
}

// BatchGetCommitParents retrieves parents for multiple commits efficiently
func (ts *TopologicalSorter) BatchGetCommitParents(ctx context.Context, commitSHAs []string) (map[string][]string, error) {
	if len(commitSHAs) == 0 {
		return make(map[string][]string), nil
	}

	// For now, we'll do this serially. In production, could optimize with git cat-file --batch
	result := make(map[string][]string)

	for _, sha := range commitSHAs {
		parents, err := ts.GetCommitParents(ctx, sha)
		if err != nil {
			// Log error but continue with empty parents
			result[sha] = []string{}
			continue
		}
		result[sha] = parents
	}

	return result, nil
}
