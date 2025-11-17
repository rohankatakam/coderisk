package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	bolt "go.etcd.io/bbolt"
)

const bucketName = "file_renames"

// IdentityResolver resolves historical file paths using git log --follow
type IdentityResolver struct {
	cacheDB  *bolt.DB
	repoRoot string // Repository root directory for path normalization
}

// NewIdentityResolver creates a new identity resolver
func NewIdentityResolver(cacheDB *bolt.DB) *IdentityResolver {
	// Try to detect repo root
	repoRoot, _ := detectRepoRoot()
	return &IdentityResolver{
		cacheDB:  cacheDB,
		repoRoot: repoRoot,
	}
}

// detectRepoRoot finds the git repository root by looking for .git directory
func detectRepoRoot() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for .git
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .git
			return "", nil
		}
		dir = parent
	}
}


// ResolveHistoricalPaths returns all historical paths for a file
// Uses the repo root from constructor
func (r *IdentityResolver) ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error) {
	return r.ResolveHistoricalPathsWithRoot(ctx, currentPath, r.repoRoot)
}

// ResolveHistoricalPathsWithRoot returns all historical paths for a file
// Accepts repo root as parameter for dynamic path resolution from Claude Code session
func (r *IdentityResolver) ResolveHistoricalPathsWithRoot(ctx context.Context, currentPath string, repoRoot string) ([]string, error) {
	// Normalize path to relative if it's absolute
	normalizedPath := currentPath

	if repoRoot != "" && filepath.IsAbs(currentPath) {
		normalizedPath = NormalizeToRelativePath(currentPath, repoRoot)
	}

	// 1. Check cache first
	cached, err := r.getCached(normalizedPath)
	if err == nil {
		return cached, nil
	}

	// 2. Run git log --follow
	cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--name-only", "--format=%H", normalizedPath)
	// Use provided repoRoot or fall back to instance repoRoot
	if repoRoot != "" {
		cmd.Dir = repoRoot
	} else if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// 3. Parse output
	lines := strings.Split(string(output), "\n")
	paths := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "/") || strings.Contains(line, ".")) {
			paths[line] = true
		}
	}

	// 4. Cache result
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	r.setCached(normalizedPath, result)

	return result, nil
}

// getCached retrieves cached paths from bbolt
func (r *IdentityResolver) getCached(filePath string) ([]string, error) {
	var result []string
	err := r.cacheDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		data := bucket.Get([]byte(filePath))
		if data == nil {
			return bolt.ErrBucketNotFound
		}
		return json.Unmarshal(data, &result)
	})
	return result, err
}

// setCached stores paths in bbolt cache
func (r *IdentityResolver) setCached(filePath string, paths []string) error {
	return r.cacheDB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		data, err := json.Marshal(paths)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(filePath), data)
	})
}
