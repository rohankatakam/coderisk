package mcp

import (
	"path/filepath"
	"strings"
)

// NormalizeToRelativePath converts an absolute file path to a repository-relative path
// Examples:
//   - /Users/user/repo/src/file.ts → src/file.ts
//   - /home/user/project/pkg/main.go → pkg/main.go
//   - src/file.ts → src/file.ts (already relative)
func NormalizeToRelativePath(absPath, repoRoot string) string {
	// If already relative (no leading slash), return as-is
	if !filepath.IsAbs(absPath) {
		return filepath.Clean(absPath)
	}

	// Clean both paths to handle . and .. correctly
	cleanAbs := filepath.Clean(absPath)
	cleanRepo := filepath.Clean(repoRoot)

	// Try to make it relative to repo root
	relPath, err := filepath.Rel(cleanRepo, cleanAbs)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// If we can't make it relative or it goes outside repo, return cleaned absolute path
		return cleanAbs
	}

	return relPath
}

// TryBothPathFormats returns both absolute and relative versions of a path
// This is useful for querying the graph which might have either format
func TryBothPathFormats(path, repoRoot string) []string {
	if repoRoot == "" {
		return []string{path}
	}

	var paths []string

	// Add original path
	paths = append(paths, filepath.Clean(path))

	// If absolute, also add relative version
	if filepath.IsAbs(path) {
		relPath := NormalizeToRelativePath(path, repoRoot)
		if relPath != filepath.Clean(path) {
			paths = append(paths, relPath)
		}
	} else {
		// If relative, also add absolute version
		absPath := filepath.Join(repoRoot, path)
		if absPath != filepath.Clean(path) {
			paths = append(paths, absPath)
		}
	}

	return paths
}

// ExtractFilePathsFromDiff parses a git diff and extracts all modified file paths
// Returns paths in the format they appear in the diff (usually relative)
func ExtractFilePathsFromDiff(diff string) []string {
	lines := strings.Split(diff, "\n")
	var paths []string
	seen := make(map[string]bool)

	for _, line := range lines {
		// Look for diff headers: "diff --git a/path/to/file b/path/to/file"
		if strings.HasPrefix(line, "diff --git ") {
			// Extract the "b/" path (the after state)
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				path := strings.TrimPrefix(parts[3], "b/")
				if path != "" && !seen[path] {
					paths = append(paths, path)
					seen[path] = true
				}
			}
		} else if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			// Also extract from --- a/file and +++ b/file headers
			// Remove prefix (--- a/ or +++ b/)
			path := strings.TrimPrefix(line, "--- ")
			path = strings.TrimPrefix(path, "+++ ")
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")
			path = strings.TrimSpace(path)

			// Ignore /dev/null (file deletions/creations)
			if path != "" && path != "/dev/null" && !seen[path] {
				paths = append(paths, path)
				seen[path] = true
			}
		}
	}

	return paths
}
