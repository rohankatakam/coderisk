package llm

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// DualPipeline implements the two-stage LLM processing strategy
// Stage 1 (Pre-filter): Fast metadata-based file selection (80-95% reduction)
// Stage 2 (Primary Parser): Deep semantic analysis on selected files only
// Fallback: Heuristic-based selection when LLM unavailable
type DualPipeline struct {
	preFilterClient *Client // Fast, cheap model (gemini-2.0-flash)
	primaryClient   *Client // Accurate model (gemini-2.0-flash or gemini-1.5-pro)
	logger          *slog.Logger
}

// NewDualPipeline creates a new dual-LLM pipeline
func NewDualPipeline(ctx context.Context, preFilterClient, primaryClient *Client) *DualPipeline {
	return &DualPipeline{
		preFilterClient: preFilterClient,
		primaryClient:   primaryClient,
		logger:          slog.Default().With("component", "dual_pipeline"),
	}
}

// FileMetadata contains metadata for pre-filter stage
type FileMetadata struct {
	Path          string
	Extension     string
	SizeBytes     int64
	LinesAdded    int
	LinesDeleted  int
	ChangeType    string // "added", "modified", "deleted", "renamed"
	IsNewFile     bool
	IsDeletedFile bool
}

// PreFilterResult contains the result of Stage 1 filtering
type PreFilterResult struct {
	SelectedFiles []string
	SkippedFiles  []string
	UsedHeuristic bool
	Reason        string
}

// FilterFiles performs Stage 1: Pre-filter LLM selection
// Batches up to 100 files per LLM call for efficiency
func (dp *DualPipeline) FilterFiles(ctx context.Context, files []FileMetadata, commitSummary string) (*PreFilterResult, error) {
	// Edge case: Auto-skip commits with >1000 files (mass reformats/dependency updates)
	if len(files) > 1000 {
		dp.logger.Warn("commit has >1000 files, auto-skipping semantic analysis",
			"file_count", len(files),
		)
		return &PreFilterResult{
			SelectedFiles: []string{},
			SkippedFiles:  extractPaths(files),
			UsedHeuristic: true,
			Reason:        fmt.Sprintf("Auto-skipped: commit has %d files (>1000 threshold for mass reformats)", len(files)),
		}, nil
	}

	// If LLM disabled or pre-filter client unavailable, use heuristic fallback
	if dp.preFilterClient == nil || !dp.preFilterClient.enabled {
		dp.logger.Info("pre-filter LLM unavailable, using heuristic fallback")
		return dp.heuristicFilter(files)
	}

	// Try LLM pre-filter
	result, err := dp.llmPreFilter(ctx, files, commitSummary)
	if err != nil {
		dp.logger.Warn("pre-filter LLM failed, falling back to heuristics",
			"error", err,
		)
		return dp.heuristicFilter(files)
	}

	return result, nil
}

// llmPreFilter uses LLM to select files for semantic analysis
func (dp *DualPipeline) llmPreFilter(ctx context.Context, files []FileMetadata, commitSummary string) (*PreFilterResult, error) {
	// Build file list for LLM
	var fileList strings.Builder
	for i, f := range files {
		fileList.WriteString(fmt.Sprintf("%d. %s (%s, %d lines changed)\n",
			i+1, f.Path, f.Extension, f.LinesAdded+f.LinesDeleted))
	}

	// Construct prompt for binary classification
	systemPrompt := `You are a code analysis pre-filter. Your job is to identify which files likely contain semantic code changes (functions, classes, methods) that should be analyzed.

SKIP these files:
- Configuration files (package.json, .yml, .toml, etc.)
- Lock files (package-lock.json, yarn.lock, go.sum, etc.)
- Generated code (*.pb.go, *_generated.go, *.min.js, etc.)
- Documentation (*.md, *.txt, LICENSE, etc.)
- Binary files
- Large files (>50KB) unless clearly important
- Test fixtures and mock data

PARSE these files:
- Source code files (.ts, .tsx, .js, .jsx, .py, .go, .rs, .java, etc.)
- Files with significant logic changes
- New files that define functions/classes
- Files mentioned in commit message

Return ONLY the file paths that should be parsed, one per line. No explanations.`

	userPrompt := fmt.Sprintf(`Commit summary: %s

Files (%d total):
%s

Which files should be semantically analyzed?`, commitSummary, len(files), fileList.String())

	// Call pre-filter LLM
	response, err := dp.preFilterClient.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("pre-filter LLM call failed: %w", err)
	}

	// Parse response (line-separated file paths)
	selectedPaths := parseFileListResponse(response)

	// Build result
	pathSet := make(map[string]bool)
	for _, path := range selectedPaths {
		pathSet[path] = true
	}

	var selected, skipped []string
	for _, f := range files {
		if pathSet[f.Path] {
			selected = append(selected, f.Path)
		} else {
			skipped = append(skipped, f.Path)
		}
	}

	reductionPercent := float64(len(skipped)) / float64(len(files)) * 100.0
	dp.logger.Info("pre-filter LLM completed",
		"total_files", len(files),
		"selected", len(selected),
		"skipped", len(skipped),
		"reduction_percent", fmt.Sprintf("%.1f%%", reductionPercent),
	)

	return &PreFilterResult{
		SelectedFiles: selected,
		SkippedFiles:  skipped,
		UsedHeuristic: false,
		Reason:        "LLM pre-filter",
	}, nil
}

// heuristicFilter implements deterministic fallback logic
func (dp *DualPipeline) heuristicFilter(files []FileMetadata) (*PreFilterResult, error) {
	var selected, skipped []string
	var warnings []string

	for _, f := range files {
		decision, reason := dp.heuristicDecision(f)

		if decision {
			selected = append(selected, f.Path)
		} else {
			skipped = append(skipped, f.Path)
		}

		// Log heuristic decisions for manual review
		if reason != "" {
			warning := fmt.Sprintf("%s: %s", f.Path, reason)
			warnings = append(warnings, warning)
		}
	}

	// Log all heuristic warnings
	if len(warnings) > 0 {
		dp.logger.Warn("heuristic fallback decisions (manual review recommended)",
			"total_files", len(files),
			"decisions", warnings[:min(10, len(warnings))], // Log first 10
		)
	}

	reductionPercent := float64(len(skipped)) / float64(len(files)) * 100.0
	dp.logger.Info("heuristic filter completed",
		"total_files", len(files),
		"selected", len(selected),
		"skipped", len(skipped),
		"reduction_percent", fmt.Sprintf("%.1f%%", reductionPercent),
	)

	return &PreFilterResult{
		SelectedFiles: selected,
		SkippedFiles:  skipped,
		UsedHeuristic: true,
		Reason:        "Heuristic fallback (LLM unavailable)",
	}, nil
}

// heuristicDecision returns (shouldParse, reason)
func (dp *DualPipeline) heuristicDecision(f FileMetadata) (bool, string) {
	// Rule 1: Skip deleted files
	if f.IsDeletedFile {
		return false, "deleted file"
	}

	// Rule 2: Skip files >50KB (likely generated or data files)
	if f.SizeBytes > 50*1024 {
		return false, fmt.Sprintf("large file (%d KB)", f.SizeBytes/1024)
	}

	// Rule 3: Skip by extension (config/lock/doc files)
	skipExtensions := map[string]bool{
		".json":       true, // package.json, tsconfig.json, etc.
		".lock":       true, // package-lock.json, yarn.lock
		".sum":        true, // go.sum
		".mod":        true, // go.mod (usually)
		".yml":        true, // config files
		".yaml":       true,
		".toml":       true,
		".md":         true, // documentation
		".txt":        true,
		".gitignore":  true,
		".dockerignore": true,
		"Dockerfile":  true,
		"Makefile":    true,
		".sh":         true, // shell scripts (debatable)
		".sql":        true, // migrations (debatable)
	}

	ext := filepath.Ext(f.Path)
	if ext == "" {
		ext = filepath.Base(f.Path) // Handle files like "Dockerfile"
	}

	if skipExtensions[ext] {
		return false, fmt.Sprintf("config/lock file (%s)", ext)
	}

	// Rule 4: Skip generated code patterns
	generatedPatterns := []string{
		"_generated.go",
		".pb.go",       // protobuf
		".min.js",      // minified
		".bundle.js",   // bundled
		"dist/",        // build output
		"build/",
		"node_modules/",
		"vendor/",
		".next/",
		".cache/",
	}

	for _, pattern := range generatedPatterns {
		if strings.Contains(f.Path, pattern) {
			return false, fmt.Sprintf("generated/build artifact (%s)", pattern)
		}
	}

	// Rule 5: Parse code files by extension
	codeExtensions := map[string]bool{
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".py":   true,
		".go":   true,
		".rs":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
		".rb":   true,
		".php":  true,
		".swift": true,
		".kt":   true,
		".scala": true,
	}

	if codeExtensions[ext] {
		return true, "" // No warning needed for standard code files
	}

	// Rule 6: Default to SKIP for unknown extensions (conservative)
	return false, fmt.Sprintf("unknown extension (%s)", ext)
}

// Helper functions

func extractPaths(files []FileMetadata) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths
}

func parseFileListResponse(response string) []string {
	var paths []string
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove leading numbers/bullets (e.g., "1. src/file.ts" -> "src/file.ts")
		line = strings.TrimLeft(line, "0123456789. -•*")
		line = strings.TrimSpace(line)

		if line != "" {
			paths = append(paths, line)
		}
	}

	return paths
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
