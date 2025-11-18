package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileIdentity represents a file's canonical identity and rename history
type FileIdentity struct {
	CanonicalPath        string    // Current path at HEAD
	HistoricalPaths      []string  // All historical paths (including current)
	FirstSeenCommitSHA   string    // First commit where file appeared
	LastModifiedCommitSHA string   // Most recent commit
	LastModifiedAt       time.Time // When last modified
	Language             string    // Detected language
	FileType             string    // 'source', 'test', 'config', 'docs'
	Status               string    // 'active', 'deleted', 'renamed'
}

// FileIdentityMapper builds canonical file identity mappings
type FileIdentityMapper struct {
	repoPath string
	repoID   int64
}

// NewFileIdentityMapper creates a new file identity mapper
func NewFileIdentityMapper(repoPath string, repoID int64) *FileIdentityMapper {
	return &FileIdentityMapper{
		repoPath: repoPath,
		repoID:   repoID,
	}
}

// BuildIdentityMap discovers all files at HEAD and traces their rename history
// Returns a map of canonical_path -> FileIdentity
func (m *FileIdentityMapper) BuildIdentityMap(ctx context.Context) (map[string]*FileIdentity, error) {
	// Step 1: Discover all source files at HEAD
	files, err := m.discoverSourceFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover source files: %w", err)
	}

	fmt.Printf("📁 Discovered %d source files at HEAD\n", len(files))

	// Step 2: Trace rename history for each file in parallel
	identityMap := make(map[string]*FileIdentity)
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit to 50 concurrent git operations

	errors := make([]error, 0)
	var errorMu sync.Mutex

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Trace file history
			identity, err := m.traceFileHistory(ctx, filePath)
			if err != nil {
				errorMu.Lock()
				errors = append(errors, fmt.Errorf("failed to trace %s: %w", filePath, err))
				errorMu.Unlock()
				return
			}

			// Store in map
			mu.Lock()
			identityMap[filePath] = identity
			mu.Unlock()
		}(file)
	}

	wg.Wait()

	if len(errors) > 0 {
		// Return first few errors for debugging
		errMsg := "errors during file history tracing:\n"
		for i, err := range errors {
			if i >= 5 {
				errMsg += fmt.Sprintf("... and %d more errors\n", len(errors)-5)
				break
			}
			errMsg += fmt.Sprintf("  - %v\n", err)
		}
		return nil, fmt.Errorf(errMsg)
	}

	fmt.Printf("✅ Successfully traced rename history for %d files\n", len(identityMap))
	return identityMap, nil
}

// discoverSourceFiles finds all source files at HEAD
// Filters out binary files, vendor directories, node_modules, etc.
func (m *FileIdentityMapper) discoverSourceFiles(ctx context.Context) ([]string, error) {
	// Use git ls-files to get all tracked files at HEAD
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = m.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	allFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	sourceFiles := make([]string, 0, len(allFiles))

	for _, file := range allFiles {
		if file == "" {
			continue
		}

		// Filter out non-source files
		if m.isSourceFile(file) {
			sourceFiles = append(sourceFiles, file)
		}
	}

	return sourceFiles, nil
}

// isSourceFile determines if a file is a source code file worth tracking
func (m *FileIdentityMapper) isSourceFile(path string) bool {
	// Skip common non-source directories
	excludeDirs := []string{
		"vendor/", "node_modules/", ".git/", "build/", "dist/",
		"__pycache__/", ".pytest_cache/", "coverage/", ".next/",
		"target/", "bin/", "obj/", ".gradle/", ".idea/", ".vscode/",
	}

	for _, dir := range excludeDirs {
		if strings.Contains(path, dir) {
			return false
		}
	}

	// Include common source file extensions
	sourceExts := []string{
		".go", ".py", ".js", ".ts", ".tsx", ".jsx",
		".java", ".c", ".cpp", ".cc", ".h", ".hpp",
		".rs", ".rb", ".php", ".swift", ".kt",
		".scala", ".clj", ".sh", ".sql",
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, srcExt := range sourceExts {
		if ext == srcExt {
			return true
		}
	}

	return false
}

// traceFileHistory runs git log --follow to get complete rename history
func (m *FileIdentityMapper) traceFileHistory(ctx context.Context, filePath string) (*FileIdentity, error) {
	// Run: git log --follow --name-only --pretty=format:"%H|%aI" -- <file>
	// This gives us: commit_sha|timestamp\nfilename\n
	cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--name-only", "--pretty=format:%H|%aI", "--", filePath)
	cmd.Dir = m.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log --follow failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		// File exists but has no history (shouldn't happen, but handle gracefully)
		return &FileIdentity{
			CanonicalPath:   filePath,
			HistoricalPaths: []string{filePath},
			Language:        m.detectLanguage(filePath),
			FileType:        m.detectFileType(filePath),
			Status:          "active",
		}, nil
	}

	// Parse output to extract history
	var historicalPaths []string
	var firstCommitSHA, lastCommitSHA string
	var lastCommitTime time.Time

	seenPaths := make(map[string]bool)
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Parse commit line: "sha|timestamp"
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				commitSHA := parts[0]
				timestamp := parts[1]

				// Parse timestamp
				commitTime, err := time.Parse(time.RFC3339, timestamp)
				if err == nil {
					if lastCommitSHA == "" {
						// First commit in log (most recent)
						lastCommitSHA = commitSHA
						lastCommitTime = commitTime
					}
					// Keep updating to get oldest commit
					firstCommitSHA = commitSHA
				}
			}
			i++
			continue
		}

		// Parse file path line
		if line != "" && !strings.Contains(line, "|") {
			// This is a file path
			if !seenPaths[line] {
				historicalPaths = append(historicalPaths, line)
				seenPaths[line] = true
			}
		}

		i++
	}

	// If we didn't find any paths in the log output, use the current path
	if len(historicalPaths) == 0 {
		historicalPaths = []string{filePath}
	}

	// Reverse historical paths to get chronological order (oldest first)
	reversedPaths := make([]string, len(historicalPaths))
	for i := range historicalPaths {
		reversedPaths[i] = historicalPaths[len(historicalPaths)-1-i]
	}

	return &FileIdentity{
		CanonicalPath:         filePath,
		HistoricalPaths:       reversedPaths,
		FirstSeenCommitSHA:    firstCommitSHA,
		LastModifiedCommitSHA: lastCommitSHA,
		LastModifiedAt:        lastCommitTime,
		Language:              m.detectLanguage(filePath),
		FileType:              m.detectFileType(filePath),
		Status:                "active",
	}, nil
}

// detectLanguage infers language from file extension
func (m *FileIdentityMapper) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	langMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".jsx":   "javascript",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cc":    "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rs":    "rust",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".clj":   "clojure",
		".sh":    "shell",
		".sql":   "sql",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "unknown"
}

// detectFileType categorizes file as source/test/config/docs
func (m *FileIdentityMapper) detectFileType(path string) string {
	lowerPath := strings.ToLower(path)

	// Test files
	if strings.Contains(lowerPath, "test") || strings.Contains(lowerPath, "_test.") || strings.Contains(lowerPath, ".test.") {
		return "test"
	}

	// Config files
	if strings.Contains(lowerPath, "config") || strings.Contains(lowerPath, ".config.") {
		return "config"
	}

	// Documentation
	if strings.HasSuffix(lowerPath, ".md") || strings.HasSuffix(lowerPath, ".txt") || strings.Contains(lowerPath, "doc") {
		return "docs"
	}

	return "source"
}

// HistoricalPathsToJSON converts historical paths to JSONB format
func HistoricalPathsToJSON(paths []string) (json.RawMessage, error) {
	data, err := json.Marshal(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal historical paths: %w", err)
	}
	return json.RawMessage(data), nil
}
