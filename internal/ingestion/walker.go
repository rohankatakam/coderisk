package ingestion

import (
	"os"
	"path/filepath"
	"strings"
)

// WalkSourceFiles walks repository and yields source files
// Excludes common non-source directories and generated files
func WalkSourceFiles(repoPath string) (<-chan string, error) {
	files := make(chan string, 100)

	go func() {
		defer close(files)

		filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip excluded directories
			if d.IsDir() && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}

			// Only process files with supported extensions
			if !d.IsDir() && isSupportedFile(path) {
				files <- path
			}

			return nil
		})
	}()

	return files, nil
}

// shouldSkipDir returns true if directory should be excluded from parsing
func shouldSkipDir(name string) bool {
	excludeDirs := []string{
		".git",
		"node_modules",
		"vendor",
		"venv",
		"__pycache__",
		".next",
		".nuxt",
		"dist",
		"build",
		"out",
		"target",
		".cache",
		".parcel-cache",
		"coverage",
		".nyc_output",
		".pytest_cache",
		".tox",
		".venv",
		"env",
		"__mocks__",
		".idea",
		".vscode",
		".DS_Store",
	}

	for _, exclude := range excludeDirs {
		if name == exclude || strings.HasPrefix(name, exclude) {
			return true
		}
	}
	return false
}

// isSupportedFile returns true if file should be parsed
func isSupportedFile(path string) bool {
	// Check extension
	ext := filepath.Ext(path)
	supported := []string{
		".js", ".jsx",
		".ts", ".tsx",
		".mjs", ".cjs",
		".mts", ".cts",
		".py", ".pyi", ".pyw",
	}

	isSupported := false
	for _, s := range supported {
		if ext == s {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return false
	}

	// Exclude generated files
	if isGeneratedFile(path) {
		return false
	}

	// Exclude test fixtures and mock files
	if isTestFixture(path) {
		return false
	}

	return true
}

// isGeneratedFile returns true if file is likely generated
func isGeneratedFile(path string) bool {
	generatedPatterns := []string{
		".min.js",       // Minified JS
		".bundle.js",    // Bundled JS
		".generated.ts", // Generated TypeScript
		".generated.js", // Generated JS
		".pb.js",        // Protocol buffers
		".pb.ts",        // Protocol buffers
		".d.ts",         // TypeScript declarations (optional: could include)
		"_pb.js",        // Protocol buffers
		"_pb.ts",        // Protocol buffers
	}

	for _, pattern := range generatedPatterns {
		if strings.HasSuffix(path, pattern) {
			return true
		}
	}

	// Check if in generated directories
	generatedDirs := []string{
		"/dist/",
		"/build/",
		"/out/",
		"/.next/",
		"/.nuxt/",
	}

	for _, dir := range generatedDirs {
		if strings.Contains(path, dir) {
			return true
		}
	}

	return false
}

// isTestFixture returns true if file is a test fixture or mock
func isTestFixture(path string) bool {
	testDirs := []string{
		"/__tests__/fixtures/",
		"/__mocks__/",
		"/test/fixtures/",
		"/tests/fixtures/",
		"/spec/fixtures/",
	}

	for _, dir := range testDirs {
		if strings.Contains(path, dir) {
			return true
		}
	}

	return false
}

// FileStats holds statistics about discovered files
type FileStats struct {
	Total            int
	JavaScript       int
	TypeScript       int
	Python           int
	Skipped          int
	SkippedGenerated int
	SkippedTest      int
}

// CountFiles walks repository and counts files by type
func CountFiles(repoPath string) (*FileStats, error) {
	stats := &FileStats{}

	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)

		// Count all files
		stats.Total++

		// Count by language
		switch ext {
		case ".js", ".jsx", ".mjs", ".cjs":
			if !isGeneratedFile(path) && !isTestFixture(path) {
				stats.JavaScript++
			}
		case ".ts", ".tsx", ".mts", ".cts":
			if !isGeneratedFile(path) && !isTestFixture(path) {
				stats.TypeScript++
			}
		case ".py", ".pyi", ".pyw":
			if !isGeneratedFile(path) && !isTestFixture(path) {
				stats.Python++
			}
		}

		// Count skipped
		if isGeneratedFile(path) {
			stats.SkippedGenerated++
		}
		if isTestFixture(path) {
			stats.SkippedTest++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	stats.Skipped = stats.SkippedGenerated + stats.SkippedTest

	return stats, nil
}
