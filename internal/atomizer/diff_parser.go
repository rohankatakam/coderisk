package atomizer

import (
	"regexp"
	"strconv"
	"strings"
)

// DiffFileChange represents a parsed file change from a git diff
type DiffFileChange struct {
	FilePath   string          // Canonical file path (b/ side for new files, a/ side for deletions)
	ChangeType string          // "added", "modified", "deleted", "renamed"
	OldPath    string          // For renames, the old path
	Hunks      []DiffHunk      // Parsed diff hunks with line numbers
}

// DiffHunk represents a single diff hunk with line number information
type DiffHunk struct {
	OldStart int    // Starting line number in old file
	OldCount int    // Number of lines in old file
	NewStart int    // Starting line number in new file
	NewCount int    // Number of lines in new file
	Content  string // Actual diff content
}

// Regex patterns for parsing git diffs
var (
	// Match: diff --git a/path/to/file.ts b/path/to/file.ts
	diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+?) b/(.+?)$`)

	// Match: @@ -42,10 +42,15 @@ function name
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	// Match: new file mode 100644
	newFileRegex = regexp.MustCompile(`^new file mode`)

	// Match: deleted file mode 100644
	deletedFileRegex = regexp.MustCompile(`^deleted file mode`)

	// Match: rename from old/path.ts
	renameFromRegex = regexp.MustCompile(`^rename from (.+)$`)

	// Match: rename to new/path.ts
	renameToRegex = regexp.MustCompile(`^rename to (.+)$`)
)

// ParseDiff parses a git diff and extracts file paths and line number ranges
// Returns a map of file paths to their parsed change information
// Reference: YC_DEMO_GAP_ANALYSIS.md - Parse structured data instead of asking LLM
func ParseDiff(diffContent string) map[string]*DiffFileChange {
	files := make(map[string]*DiffFileChange)

	lines := strings.Split(diffContent, "\n")
	var currentFile *DiffFileChange
	var currentHunk *DiffHunk
	var inHunk bool

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Parse diff header: diff --git a/file.ts b/file.ts
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous file if exists
			if currentFile != nil && currentFile.FilePath != "" {
				files[currentFile.FilePath] = currentFile
			}

			// Start new file
			oldPath := matches[1]
			newPath := matches[2]

			currentFile = &DiffFileChange{
				FilePath:   newPath, // Default to new path
				ChangeType: "modified",
				OldPath:    oldPath,
				Hunks:      []DiffHunk{},
			}
			inHunk = false
			continue
		}

		if currentFile == nil {
			continue
		}

		// Detect new file
		if newFileRegex.MatchString(line) {
			currentFile.ChangeType = "added"
			continue
		}

		// Detect deleted file
		if deletedFileRegex.MatchString(line) {
			currentFile.ChangeType = "deleted"
			currentFile.FilePath = currentFile.OldPath // Use old path for deletions
			continue
		}

		// Detect rename
		if matches := renameFromRegex.FindStringSubmatch(line); matches != nil {
			currentFile.ChangeType = "renamed"
			currentFile.OldPath = matches[1]
			continue
		}

		if matches := renameToRegex.FindStringSubmatch(line); matches != nil {
			currentFile.FilePath = matches[1]
			continue
		}

		// Parse hunk header: @@ -42,10 +42,15 @@
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			// Parse line numbers
			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}

			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &DiffHunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Content:  "",
			}
			inHunk = true
			continue
		}

		// Collect hunk content
		if inHunk && currentHunk != nil {
			// Stop collecting if we hit another diff header
			if strings.HasPrefix(line, "diff --git") {
				i-- // Reprocess this line
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
					currentHunk = nil
				}
				inHunk = false
				continue
			}

			currentHunk.Content += line + "\n"
		}
	}

	// Save last hunk and file
	if currentHunk != nil && currentFile != nil {
		currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
	}
	if currentFile != nil && currentFile.FilePath != "" {
		files[currentFile.FilePath] = currentFile
	}

	return files
}

// ExtractFilePaths extracts just the file paths from a diff (fast path)
// Returns a slice of file paths in the order they appear
func ExtractFilePaths(diffContent string) []string {
	var paths []string
	seen := make(map[string]bool)

	lines := strings.Split(diffContent, "\n")

	for _, line := range lines {
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			newPath := matches[2]
			if !seen[newPath] {
				paths = append(paths, newPath)
				seen[newPath] = true
			}
		}
	}

	return paths
}

// GetLineRangeForFile returns the line number ranges affected in a file
// Returns (startLine, endLine) for the new version of the file
// If multiple hunks exist, returns the range spanning all hunks
func GetLineRangeForFile(fileChange *DiffFileChange) (int, int) {
	if len(fileChange.Hunks) == 0 {
		return 0, 0
	}

	minStart := fileChange.Hunks[0].NewStart
	maxEnd := fileChange.Hunks[0].NewStart + fileChange.Hunks[0].NewCount - 1

	for _, hunk := range fileChange.Hunks {
		if hunk.NewStart < minStart {
			minStart = hunk.NewStart
		}
		hunkEnd := hunk.NewStart + hunk.NewCount - 1
		if hunkEnd > maxEnd {
			maxEnd = hunkEnd
		}
	}

	return minStart, maxEnd
}

// ValidateFilePath checks if a file path from LLM matches the parsed diff
// Returns the canonical file path if valid, empty string if invalid
func ValidateFilePath(llmPath string, parsedFiles map[string]*DiffFileChange) string {
	// Exact match
	if _, exists := parsedFiles[llmPath]; exists {
		return llmPath
	}

	// Try without leading slash
	trimmedPath := strings.TrimPrefix(llmPath, "/")
	if _, exists := parsedFiles[trimmedPath]; exists {
		return trimmedPath
	}

	// Try with leading slash
	slashPath := "/" + strings.TrimPrefix(llmPath, "/")
	if _, exists := parsedFiles[slashPath]; exists {
		return slashPath
	}

	// Check if it's a suffix match (LLM might omit leading directories)
	for path := range parsedFiles {
		if strings.HasSuffix(path, llmPath) {
			return path
		}
	}

	return ""
}
