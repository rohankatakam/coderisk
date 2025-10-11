package phase0

import (
	"strings"
)

// DocumentationExtensions contains file extensions that indicate documentation files
// 12-factor: Factor 3 - Own your context window (skip analysis for zero-impact changes)
var DocumentationExtensions = []string{
	".md",    // Markdown
	".txt",   // Plain text
	".rst",   // reStructuredText
	".adoc",  // AsciiDoc
	".markdown", // Markdown variant
}

// DocumentationDetectionResult contains the results of documentation-only detection
type DocumentationDetectionResult struct {
	IsDocumentationOnly bool   // True if all changes are documentation
	SkipAnalysis        bool   // True if we should skip Phase 1/2 analysis
	Reason              string // Human-readable explanation
	DocumentationType   string // Type of documentation (markdown, text, etc.)
}

// IsDocumentationOnly checks if a file is a documentation file with no code impact
// Returns true if the file is documentation-only (README.md, comments, etc.)
// 12-factor: Factor 8 - Own your control flow (explicit skip criteria)
func IsDocumentationOnly(filePath string) DocumentationDetectionResult {
	result := DocumentationDetectionResult{
		IsDocumentationOnly: false,
		SkipAnalysis:        false,
	}

	// Check if file extension matches documentation types
	filePathLower := strings.ToLower(filePath)

	for _, ext := range DocumentationExtensions {
		if strings.HasSuffix(filePathLower, ext) {
			result.IsDocumentationOnly = true
			result.SkipAnalysis = true
			result.DocumentationType = strings.TrimPrefix(ext, ".")
			result.Reason = "Documentation file (zero runtime impact)"
			return result
		}
	}

	// Check for common documentation file names (without extensions)
	docFileNames := []string{
		"readme",
		"changelog",
		"contributing",
		"license",
		"authors",
		"contributors",
		"code_of_conduct",
		"security",
		"support",
	}

	fileNameLower := strings.ToLower(getFileName(filePath))
	for _, docName := range docFileNames {
		if strings.HasPrefix(fileNameLower, docName) {
			result.IsDocumentationOnly = true
			result.SkipAnalysis = true
			result.DocumentationType = "documentation"
			result.Reason = "Documentation file (zero runtime impact)"
			return result
		}
	}

	return result
}

// IsCommentOnlyChange analyzes diff content to determine if changes are only comments
// This is more complex and requires parsing diff hunks
func IsCommentOnlyChange(diff string) bool {
	if diff == "" {
		return false // No change
	}

	// Parse diff lines
	lines := strings.Split(diff, "\n")
	hasCodeChange := false
	hasAnyChange := false // Track if we found any actual +/- lines

	for _, line := range lines {
		// Skip diff metadata lines
		if strings.HasPrefix(line, "@@") ||
			strings.HasPrefix(line, "diff") ||
			strings.HasPrefix(line, "index") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") {
			continue
		}

		// Check added or removed lines
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			hasAnyChange = true // Found at least one change line

			// Remove the +/- prefix
			content := strings.TrimSpace(line[1:])

			// Skip empty lines
			if content == "" {
				continue
			}

			// Check if line is a comment
			if !isCommentLine(content) {
				hasCodeChange = true
				break
			}
		}
	}

	// Only return true if we found changes and they were all comments
	return hasAnyChange && !hasCodeChange
}

// isCommentLine checks if a line is a comment in common languages
func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Empty line
	if trimmed == "" {
		return true
	}

	// Common comment patterns
	commentPrefixes := []string{
		"//",   // Go, C, C++, Java, JavaScript, TypeScript
		"#",    // Python, Ruby, Shell, YAML
		"/*",   // Block comment start (Go, C, Java, etc.)
		"*/",   // Block comment end
		"*",    // Block comment continuation
		"<!--", // HTML, Markdown
		"-->",  // HTML, Markdown
		"--",   // SQL, Lua
		";",    // Lisp, Assembly
		"\"\"\"", // Python docstring
		"'''",  // Python docstring
	}

	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	return false
}

// getFileName extracts the file name from a path
func getFileName(path string) string {
	// Handle both Unix and Windows paths
	path = strings.ReplaceAll(path, "\\", "/")

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return path
	}

	return path[lastSlash+1:]
}

// IsDocumentationFile is a simplified check for documentation file extensions only
// Use this for quick checks without full analysis
func IsDocumentationFile(filePath string) bool {
	result := IsDocumentationOnly(filePath)
	return result.IsDocumentationOnly
}
