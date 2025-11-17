package mcp

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	bolt "go.etcd.io/bbolt"
)

const bucketName = "file_renames"

// IdentityResolver resolves historical file paths using git log --follow
type IdentityResolver struct {
	cacheDB *bolt.DB
}

// NewIdentityResolver creates a new identity resolver
func NewIdentityResolver(cacheDB *bolt.DB) *IdentityResolver {
	return &IdentityResolver{
		cacheDB: cacheDB,
	}
}

// ResolveHistoricalPaths returns all historical paths for a file
func (r *IdentityResolver) ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error) {
	// 1. Check cache first
	cached, err := r.getCached(currentPath)
	if err == nil {
		return cached, nil
	}

	// 2. Run git log --follow
	cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--name-only", "--format=%H", currentPath)
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
	r.setCached(currentPath, result)

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
