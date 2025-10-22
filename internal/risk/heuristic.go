package risk

import (
	"context"
	"strings"
	"time"
)

// HeuristicFilter implements Tier 0 filtering for trivial changes
type HeuristicFilter struct {
	maxLinesForTrivial int
	maxFilesForTrivial int
}

// NewHeuristicFilter creates a new heuristic filter
func NewHeuristicFilter() *HeuristicFilter {
	return &HeuristicFilter{
		maxLinesForTrivial: 10,
		maxFilesForTrivial: 3,
	}
}

// Filter applies heuristic rules to quickly identify trivial changes
func (h *HeuristicFilter) Filter(ctx context.Context, req *AnalysisRequest) (*HeuristicFilterResult, error) {
	startTime := time.Now()
	
	result := &HeuristicFilterResult{
		IsTrivial:    false,
		Confidence:   0.0,
		MatchedRules: []string{},
		FilesChanged: len(req.FilePaths),
	}

	// Count total lines changed in diff
	linesChanged := countLinesInDiff(req.GitDiff)
	result.LinesChanged = linesChanged

	// Rule 1: Whitespace-only changes
	if isWhitespaceOnly(req.GitDiff) {
		result.IsTrivial = true
		result.Reason = "Change contains only whitespace modifications"
		result.ChangeType = "whitespace"
		result.Confidence = 0.95
		result.MatchedRules = append(result.MatchedRules, "whitespace_only")
		result.DurationMS = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Rule 2: Comment-only changes
	if isCommentOnly(req.GitDiff) {
		result.IsTrivial = true
		result.Reason = "Change contains only comment modifications"
		result.ChangeType = "comment"
		result.Confidence = 0.90
		result.MatchedRules = append(result.MatchedRules, "comment_only")
		result.DurationMS = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Rule 3: Documentation files only (.md, .txt, README, etc.)
	if isDocsOnly(req.FilePaths) {
		result.IsTrivial = true
		result.Reason = "Change affects only documentation files"
		result.ChangeType = "documentation"
		result.Confidence = 0.85
		result.MatchedRules = append(result.MatchedRules, "docs_only")
		result.DurationMS = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Rule 4: Very small changes (< 10 lines, < 3 files)
	if linesChanged < h.maxLinesForTrivial && len(req.FilePaths) <= h.maxFilesForTrivial {
		result.IsTrivial = true
		result.Reason = "Very small change (low line/file count)"
		result.ChangeType = "small_change"
		result.Confidence = 0.70
		result.MatchedRules = append(result.MatchedRules, "small_change")
		result.DurationMS = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Not trivial
	result.IsTrivial = false
	result.Reason = "Change requires full analysis"
	result.ChangeType = "complex"
	result.Confidence = 0.80
	result.DurationMS = time.Since(startTime).Milliseconds()

	return result, nil
}

// Helper functions

func countLinesInDiff(diff string) int {
	lines := strings.Split(diff, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Skip diff metadata lines
			if !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				count++
			}
		}
	}
	return count
}

func isWhitespaceOnly(diff string) bool {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Skip diff metadata
			if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
				continue
			}
			// Check if line has non-whitespace content
			trimmed := strings.TrimSpace(line[1:]) // Remove +/- prefix
			if trimmed != "" {
				return false
			}
		}
	}
	return true
}

func isCommentOnly(diff string) bool {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Skip diff metadata
			if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
				continue
			}
			// Check if line is a comment
			trimmed := strings.TrimSpace(line[1:])
			if trimmed == "" {
				continue
			}
			// Basic comment detection (can be expanded)
			if !strings.HasPrefix(trimmed, "//") && 
			   !strings.HasPrefix(trimmed, "#") && 
			   !strings.HasPrefix(trimmed, "/*") && 
			   !strings.HasPrefix(trimmed, "*") &&
			   !strings.HasPrefix(trimmed, "<!--") {
				return false
			}
		}
	}
	return true
}

func isDocsOnly(filePaths []string) bool {
	if len(filePaths) == 0 {
		return false
	}
	
	docExtensions := map[string]bool{
		".md":   true,
		".txt":  true,
		".rst":  true,
		".adoc": true,
	}
	
	docFiles := map[string]bool{
		"README":     true,
		"LICENSE":    true,
		"CHANGELOG":  true,
		"AUTHORS":    true,
		"CONTRIBUTORS": true,
	}
	
	for _, path := range filePaths {
		// Check extension
		isDoc := false
		for ext := range docExtensions {
			if strings.HasSuffix(strings.ToLower(path), ext) {
				isDoc = true
				break
			}
		}
		
		// Check filename
		if !isDoc {
			for docFile := range docFiles {
				if strings.Contains(strings.ToUpper(path), docFile) {
					isDoc = true
					break
				}
			}
		}
		
		// If any file is not a doc file, return false
		if !isDoc {
			return false
		}
	}
	
	return true
}
