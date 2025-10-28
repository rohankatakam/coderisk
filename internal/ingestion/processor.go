package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/treesitter"
)

// ProcessorConfig holds configuration for repository processing
type ProcessorConfig struct {
	Workers           int           // Number of concurrent parsers (default: 20)
	Timeout           time.Duration // Per-file parsing timeout (default: 30s)
	GraphBatch        int           // Batch size for graph writes (default: 100)
	EnableTemporal    bool          // Enable Layer 2 (Temporal) analysis (default: true for init-local, false for init)
}

// DefaultProcessorConfig returns default configuration
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		Workers:        20,
		Timeout:        30 * time.Second,
		GraphBatch:     100,
		EnableTemporal: true, // Default: enabled for backward compatibility with init-local
	}
}

// Processor orchestrates: clone → parse → graph construction
type Processor struct {
	config       *ProcessorConfig
	graphClient  graph.Backend
	graphBuilder *graph.Builder
}

// NewProcessor creates a new repository processor
func NewProcessor(config *ProcessorConfig, graphClient graph.Backend, graphBuilder *graph.Builder) *Processor {
	if config == nil {
		config = DefaultProcessorConfig()
	}
	return &Processor{
		config:       config,
		graphClient:  graphClient,
		graphBuilder: graphBuilder,
	}
}

// ProcessResult holds results from processing a repository
type ProcessResult struct {
	RepoPath      string
	FilesTotal    int
	FilesParsed   int
	FilesFailed   int
	EntitiesTotal int
	Functions     int
	Classes       int
	Imports       int
	Duration      time.Duration
	Errors        []error
}

// ProcessRepository performs full Layer 1 processing
// Steps:
// 1. Clone repository (if not already cloned)
// 2. Walk file tree and discover source files
// 3. Parse files concurrently using worker pool
// 4. Extract entities (functions, classes, imports)
// 5. Build graph (File, Function, Class nodes + CALLS, IMPORTS edges)
func (p *Processor) ProcessRepository(ctx context.Context, repoURL string) (*ProcessResult, error) {
	// Step 1: Clone repository
	repoPath, err := CloneRepository(ctx, repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Info("repository cloned", "path", repoPath)

	// Process using the cloned path
	return p.ProcessRepositoryFromPath(ctx, repoPath)
}

// ProcessRepositoryFromPath processes an already-cloned repository
// This is used by `crisk init` to avoid double-cloning (Layer 1 uses the same clone as Layer 2)
func (p *Processor) ProcessRepositoryFromPath(ctx context.Context, repoPath string) (*ProcessResult, error) {
	startTime := time.Now()

	slog.Info("starting repository processing",
		"path", repoPath,
		"workers", p.config.Workers,
	)

	result := &ProcessResult{
		Errors:   []error{},
		RepoPath: repoPath,
	}

	// Step 2: Walk file tree
	files, err := WalkSourceFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to walk files: %w", err)
	}

	// Step 3: Parse files concurrently
	entities, parseErrors := p.parseFilesParallel(ctx, files)
	result.FilesParsed = len(entities)
	result.FilesFailed = len(parseErrors)
	result.Errors = parseErrors

	slog.Info("parsing complete",
		"parsed", result.FilesParsed,
		"failed", result.FilesFailed,
	)

	// Step 4: Flatten entities
	allEntities := []treesitter.CodeEntity{}
	for _, parseResult := range entities {
		allEntities = append(allEntities, parseResult.Entities...)
	}
	result.EntitiesTotal = len(allEntities)

	// Count by type
	for _, entity := range allEntities {
		switch entity.Type {
		case "function":
			result.Functions++
		case "class":
			result.Classes++
		case "import":
			result.Imports++
		}
	}

	slog.Info("entities extracted",
		"total", result.EntitiesTotal,
		"functions", result.Functions,
		"classes", result.Classes,
		"imports", result.Imports,
	)

	// Step 5: Build graph (File nodes + DEPENDS_ON edges only)
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - only File nodes from treesitter parsing
	if p.graphClient != nil {
		if err := p.buildGraph(ctx, repoPath, allEntities); err != nil {
			return nil, fmt.Errorf("failed to build graph: %w", err)
		}
		slog.Info("graph construction complete")

		// NOTE: Layer 2 (Temporal Analysis) removed - now comes from GitHub API
		// Per IMPLEMENTATION_GAP_ANALYSIS.md:
		// "CO_CHANGED edges are now computed dynamically via queries"
		// The old local git history parsing for co-changes has been removed
		// All temporal data (commits, developers, ownership, co-changes) now comes from GitHub API
		if p.config.EnableTemporal {
			slog.Warn("temporal analysis via local git is deprecated",
				"reason", "Layer 2 & 3 now come from GitHub API",
				"recommendation", "use 'crisk init' instead of 'crisk init-local'")
		}
	}

	result.Duration = time.Since(startTime)
	result.FilesTotal = result.FilesParsed + result.FilesFailed

	slog.Info("repository processing complete",
		"duration", result.Duration,
		"files", result.FilesTotal,
		"entities", result.EntitiesTotal,
	)

	return result, nil
}

// REMOVED: convertCoChangePaths and verifyCoChangedEdges
// These functions supported the deprecated CO_CHANGED edge creation
// CO_CHANGED is now computed dynamically via queries (see internal/risk/queries.go)

// parseFilesParallel parses files using worker pool pattern
func (p *Processor) parseFilesParallel(ctx context.Context, files <-chan string) ([]*treesitter.ParseResult, []error) {
	results := make(chan *treesitter.ParseResult, p.config.Workers)
	errors := make(chan error, p.config.Workers)

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < p.config.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for filePath := range files {
				// Parse with timeout
				parseCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
				result := p.parseFileWithTimeout(parseCtx, filePath)
				cancel()

				if result.Error != nil {
					errors <- fmt.Errorf("%s: %w", filePath, result.Error)
				} else {
					results <- result
				}

				// Check if context cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}()
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	var parseResults []*treesitter.ParseResult
	var parseErrors []error

	for results != nil || errors != nil {
		select {
		case result, ok := <-results:
			if !ok {
				results = nil
			} else {
				parseResults = append(parseResults, result)
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
			} else {
				parseErrors = append(parseErrors, err)
			}
		}
	}

	return parseResults, parseErrors
}

// parseFileWithTimeout parses a single file
func (p *Processor) parseFileWithTimeout(ctx context.Context, filePath string) *treesitter.ParseResult {
	// Parse file
	result, err := treesitter.ParseFile(filePath)
	if err != nil {
		return &treesitter.ParseResult{
			FilePath: filePath,
			Error:    err,
		}
	}

	return result
}

// buildGraph creates graph nodes and edges from entities
// Schema: PRE_COMMIT_GRAPH_SPEC.md - File nodes + DEPENDS_ON edges only
// repoPath is the absolute path to the repository root (for converting absolute -> relative paths)
func (p *Processor) buildGraph(ctx context.Context, repoPath string, entities []treesitter.CodeEntity) error {
	// Batch entities for efficient graph writes
	batchSize := p.config.GraphBatch

	// Only create File nodes (no Function, Class nodes in new schema)
	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]

		// Create nodes - only File nodes
		for _, entity := range batch {
			node := entityToGraphNode(entity, repoPath)
			// Skip empty nodes and non-File nodes
			if node.Label == "" || node.ID == "" || node.Label != "File" {
				continue
			}
			if _, err := p.graphClient.CreateNode(ctx, node); err != nil {
				slog.Warn("failed to create node",
					"entity", entity.Name,
					"type", entity.Type,
					"error", err,
				)
			}
		}

		slog.Debug("batch processed",
			"batch", i/batchSize+1,
			"size", len(batch),
		)
	}

	// Step 2: Create DEPENDS_ON edges (File → File imports)
	slog.Info("creating DEPENDS_ON edges", "total_entities", len(entities))
	if err := p.createDependencyEdges(ctx, repoPath, entities); err != nil {
		return fmt.Errorf("failed to create dependency edges: %w", err)
	}

	// NOTE: CONTAINS, TESTS edges removed - not in PRE_COMMIT_GRAPH_SPEC.md

	return nil
}

// createTestRelationships creates TESTS edges between test files and source files
// Reference: PHASE2_INVESTIGATION_ROADMAP.md Task 5 - Test coverage support
// Detects test files by naming conventions and creates TESTS relationships
func (p *Processor) createTestRelationships(ctx context.Context, entities []treesitter.CodeEntity) error {
	// Extract all file entities
	files := make(map[string]bool)
	for _, entity := range entities {
		if entity.Type == "file" || entity.FilePath != "" {
			files[entity.FilePath] = true
		}
	}

	// Identify test files and their corresponding source files
	var testEdges []graph.GraphEdge

	for filePath := range files {
		if !isTestFileByConvention(filePath) {
			continue
		}

		// Find the source file this test targets
		sourceFile := inferSourceFileFromTest(filePath, files)
		if sourceFile == "" {
			slog.Debug("could not infer source file for test",
				"test_file", filePath)
			continue
		}

		// Create TESTS relationship
		testEdge := graph.GraphEdge{
			From:  fmt.Sprintf("file:%s", filePath),
			To:    fmt.Sprintf("file:%s", sourceFile),
			Label: "TESTS",
			Properties: map[string]interface{}{
				"test_type": detectTestFramework(filePath),
			},
		}
		testEdges = append(testEdges, testEdge)
	}

	// Batch create test edges
	if len(testEdges) > 0 {
		slog.Info("creating TESTS edges", "count", len(testEdges))
		if err := p.graphClient.CreateEdges(ctx, testEdges); err != nil {
			return fmt.Errorf("failed to create TESTS edges: %w", err)
		}
		slog.Info("TESTS relationships created successfully", "count", len(testEdges))
	} else {
		slog.Info("no test files found to link")
	}

	return nil
}

// isTestFileByConvention determines if a file is a test file based on naming conventions
// Reference: risk_assessment_methodology.md §2.3 - Test file discovery
func isTestFileByConvention(filePath string) bool {
	// Python: test_*.py, *_test.py, files in tests/ or __tests__/
	if strings.HasSuffix(filePath, "_test.py") || strings.HasPrefix(filepath.Base(filePath), "test_") {
		return true
	}
	if strings.Contains(filePath, "/tests/") || strings.Contains(filePath, "/__tests__/") {
		if strings.HasSuffix(filePath, ".py") {
			return true
		}
	}

	// JavaScript/TypeScript: *.test.js, *.spec.js, *.test.ts, *.spec.tsx
	if strings.Contains(filePath, ".test.") || strings.Contains(filePath, ".spec.") {
		return true
	}
	if strings.Contains(filePath, "/__tests__/") {
		if strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx") ||
			strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") {
			return true
		}
	}

	// Go: *_test.go
	if strings.HasSuffix(filePath, "_test.go") {
		return true
	}

	return false
}

// detectTestFramework returns the test framework type based on file path
func detectTestFramework(filePath string) string {
	if strings.HasSuffix(filePath, ".py") {
		return "python_unittest" // Could be pytest, unittest, etc.
	}
	if strings.HasSuffix(filePath, ".test.js") || strings.HasSuffix(filePath, ".test.jsx") {
		return "jest" // Common for JS/JSX
	}
	if strings.HasSuffix(filePath, ".test.ts") || strings.HasSuffix(filePath, ".test.tsx") {
		return "jest" // Common for TS/TSX
	}
	if strings.HasSuffix(filePath, ".spec.ts") || strings.HasSuffix(filePath, ".spec.tsx") {
		return "jasmine_or_mocha" // Common for spec files
	}
	if strings.HasSuffix(filePath, "_test.go") {
		return "go_test"
	}
	return "unknown"
}

// inferSourceFileFromTest infers the source file path from a test file path
// Uses naming conventions to determine which source file a test is testing
func inferSourceFileFromTest(testPath string, availableFiles map[string]bool) string {
	dir := filepath.Dir(testPath)
	base := filepath.Base(testPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Python: test_foo.py -> foo.py, foo_test.py -> foo.py
	if strings.HasPrefix(nameWithoutExt, "test_") {
		sourceName := strings.TrimPrefix(nameWithoutExt, "test_") + ext
		// Check same directory
		candidate := filepath.Join(dir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
		// Check parent directory (tests/ subfolder pattern)
		parentDir := filepath.Dir(dir)
		candidate = filepath.Join(parentDir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
	}
	if strings.HasSuffix(nameWithoutExt, "_test") {
		sourceName := strings.TrimSuffix(nameWithoutExt, "_test") + ext
		candidate := filepath.Join(dir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
	}

	// JavaScript/TypeScript: foo.test.ts -> foo.ts, foo.spec.tsx -> foo.tsx
	if strings.Contains(nameWithoutExt, ".test") {
		sourceName := strings.Replace(nameWithoutExt, ".test", "", 1) + ext
		candidate := filepath.Join(dir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
		// Check parent directory (__tests__/ subfolder pattern)
		if strings.Contains(dir, "/__tests__") {
			parentDir := strings.Replace(dir, "/__tests__", "", 1)
			candidate = filepath.Join(parentDir, sourceName)
			if availableFiles[candidate] {
				return candidate
			}
		}
	}
	if strings.Contains(nameWithoutExt, ".spec") {
		sourceName := strings.Replace(nameWithoutExt, ".spec", "", 1) + ext
		candidate := filepath.Join(dir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
	}

	// Go: foo_test.go -> foo.go
	if strings.HasSuffix(nameWithoutExt, "_test") {
		sourceName := strings.TrimSuffix(nameWithoutExt, "_test") + ext
		candidate := filepath.Join(dir, sourceName)
		if availableFiles[candidate] {
			return candidate
		}
	}

	// If no match found, return empty string
	return ""
}

// createDependencyEdges creates DEPENDS_ON edges between files
// Schema: PRE_COMMIT_GRAPH_SPEC.md - DEPENDS_ON edge with import_type property
func (p *Processor) createDependencyEdges(ctx context.Context, repoPath string, entities []treesitter.CodeEntity) error {
	var edges []graph.GraphEdge

	// Group imports by file
	fileToImports := make(map[string][]treesitter.CodeEntity)

	// Build map of available files for import resolution
	// Store RELATIVE paths for matching
	availableFiles := make(map[string]bool)
	for _, entity := range entities {
		if entity.Type == "file" {
			relativePath := makeRelativePath(entity.FilePath, repoPath)
			availableFiles[relativePath] = true
		}
	}

	for _, entity := range entities {
		if entity.Type == "import" {
			fileToImports[entity.FilePath] = append(fileToImports[entity.FilePath], entity)
		}
	}

	// Create DEPENDS_ON edges: File → File (imports)
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - (File)-[:DEPENDS_ON {import_type: STRING}]->(File)
	for filePath, imports := range fileToImports {
		for _, imp := range imports {
			// Resolve import path to actual File node
			targetFile := resolveImportPath(imp.ImportPath, filePath, availableFiles)

			if targetFile != "" {
				// Convert paths to relative for edge creation
				fromPath := makeRelativePath(filePath, repoPath)
				toPath := targetFile // Already relative from resolveImportPath

				// Create File → File edge (only for imports within the repository)
				edges = append(edges, graph.GraphEdge{
					From:  fmt.Sprintf("file:%s", fromPath),
					To:    fmt.Sprintf("file:%s", toPath),
					Label: "DEPENDS_ON",
					Properties: map[string]interface{}{
						"import_type": detectImportType(imp.ImportPath, filePath),
					},
				})
			}
			// Skip external imports (npm packages, stdlib, etc.)
			// Per schema: only link to files in repository
		}
	}

	// Batch create edges
	if len(edges) > 0 {
		slog.Info("creating DEPENDS_ON edges", "count", len(edges))
		if err := p.graphClient.CreateEdges(ctx, edges); err != nil {
			return fmt.Errorf("failed to create DEPENDS_ON edges: %w", err)
		}
	}

	return nil
}

// detectImportType determines the import type based on file extension and import syntax
func detectImportType(importPath, sourceFile string) string {
	// Determine import type: "import", "require", "include", etc.
	ext := filepath.Ext(sourceFile)

	switch ext {
	case ".js", ".jsx", ".ts", ".tsx":
		// JavaScript/TypeScript uses "import" or "require"
		return "import"
	case ".py":
		// Python uses "import"
		return "import"
	case ".go":
		// Go uses "import"
		return "import"
	default:
		return "import"
	}
}

// makeRelativePath converts an absolute file path to a relative path from the repository root
// Example: "/Users/.../omnara/src/main.py" → "src/main.py"
func makeRelativePath(absolutePath, repoRoot string) string {
	// Ensure repoRoot doesn't have trailing slash
	repoRoot = strings.TrimSuffix(repoRoot, "/")

	// Strip the repo root prefix
	relativePath := strings.TrimPrefix(absolutePath, repoRoot+"/")

	// If the path didn't change, it means it wasn't under the repo root
	// In that case, return the absolute path as-is (shouldn't happen in normal usage)
	if relativePath == absolutePath {
		return absolutePath
	}

	return relativePath
}

// entityToGraphNode converts CodeEntity to graph node
// entityToGraphNode converts a TreeSitter entity to a graph node
// repoPath is the absolute path to the repository root for converting absolute -> relative paths
func entityToGraphNode(entity treesitter.CodeEntity, repoPath string) graph.GraphNode {
	properties := make(map[string]interface{})

	properties["name"] = entity.Name
	properties["language"] = entity.Language

	// Determine label first to properly set unique_id
	label := "File"
	switch entity.Type {
	case "function":
		label = "Function"
	case "class":
		label = "Class"
	case "import":
		// Import is NOT a node type per schema spec (simplified_graph_schema.md)
		// Imports are only EDGES between File nodes
		// Skip creating import nodes - they'll be handled as edges only
		return graph.GraphNode{} // Return empty node, will be filtered out
	}

	// Generate unique_id based on entity type
	var uniqueID string
	if label == "File" {
		// For Files: use 'path' property per PRE_COMMIT_GRAPH_SPEC.md
		// Schema: File node with path (PRIMARY KEY), language, loc, last_modified
		// Convert absolute path to relative (matching GitHub API paths)
		relativePath := makeRelativePath(entity.FilePath, repoPath)
		properties["path"] = relativePath
		uniqueID = relativePath

		// Mark as current file (from TreeSitter, represents current codebase structure)
		// Reference: issue_ingestion_implementation_plan.md Phase 1
		properties["current"] = true

		// Add language
		properties["language"] = entity.Language

		// Add LOC (lines of code) by calculating from file
		if loc, err := countLinesInFile(entity.FilePath); err == nil {
			properties["loc"] = loc
		} else {
			properties["loc"] = 0 // Default if we can't read the file
		}

		// Add last_modified timestamp from git history
		// Schema uses last_modified (DATETIME) not last_updated
		if lastModified, err := getFileLastModified(entity.FilePath); err == nil {
			properties["last_modified"] = lastModified
		} else {
			// Fallback to current time if not in git
			properties["last_modified"] = time.Now().Unix()
		}
	} else {
		// For Functions/Classes (deprecated in new schema but keep for backwards compat)
		properties["file_path"] = entity.FilePath
		uniqueID = fmt.Sprintf("%s:%s:%d", entity.FilePath, entity.Name, entity.StartLine)
	}
	properties["unique_id"] = uniqueID

	if entity.StartLine > 0 {
		properties["start_line"] = entity.StartLine
		properties["end_line"] = entity.EndLine
	}

	if entity.Signature != "" {
		properties["signature"] = entity.Signature
	}

	if entity.ImportPath != "" {
		properties["import_path"] = entity.ImportPath
	}

	return graph.GraphNode{
		ID:         uniqueID,
		Label:      label,
		Properties: properties,
	}
}

// countLinesInFile counts the number of lines in a file
// Returns the line count or error if file cannot be read
func countLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return lineCount, nil
}

// getFileLastModified gets the last modified timestamp from git history
// Returns Unix timestamp of the most recent commit that touched this file
func getFileLastModified(filePath string) (int64, error) {
	// Use git log to get the last commit timestamp for this file
	// git log -1 --format=%ct <file> returns Unix timestamp
	cmd := exec.Command("git", "log", "-1", "--format=%ct", filePath)
	output, err := cmd.Output()
	if err != nil {
		// File might not be in git yet, return current time
		return time.Now().Unix(), nil
	}

	timestampStr := strings.TrimSpace(string(output))
	if timestampStr == "" {
		// No commits for this file, return current time
		return time.Now().Unix(), nil
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Now().Unix(), nil
	}

	return timestamp, nil
}

// resolveImportPath attempts to resolve an import path to an actual file in the repository
// Returns empty string if the import is external (npm package, stdlib, etc.)
// Only resolves imports to files within the repository (MVP scope)
func resolveImportPath(importPath string, sourceFile string, availableFiles map[string]bool) string {
	// Skip external imports (npm packages, stdlib, etc.)
	// External imports don't start with './' or '../' or '/'
	if !strings.HasPrefix(importPath, "./") &&
	   !strings.HasPrefix(importPath, "../") &&
	   !strings.HasPrefix(importPath, "/") {
		// This is an external import (e.g., "react", "lodash", "@/components")
		// Skip per MVP scope - only link to files in repository
		return ""
	}

	// For relative imports, resolve to absolute path
	sourceDir := filepath.Dir(sourceFile)
	var candidatePath string

	if strings.HasPrefix(importPath, "/") {
		// Absolute path
		candidatePath = importPath
	} else {
		// Relative path (./ or ../)
		candidatePath = filepath.Join(sourceDir, importPath)
		candidatePath = filepath.Clean(candidatePath)
	}

	// Try exact match first
	if availableFiles[candidatePath] {
		return candidatePath
	}

	// Try with common extensions (TypeScript/JavaScript often omit extensions)
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".py", ".go"}
	for _, ext := range extensions {
		withExt := candidatePath + ext
		if availableFiles[withExt] {
			return withExt
		}
	}

	// Try as directory with index file (common in JS/TS)
	indexFiles := []string{
		filepath.Join(candidatePath, "index.ts"),
		filepath.Join(candidatePath, "index.tsx"),
		filepath.Join(candidatePath, "index.js"),
		filepath.Join(candidatePath, "index.jsx"),
	}
	for _, indexFile := range indexFiles {
		if availableFiles[indexFile] {
			return indexFile
		}
	}

	// If no match found, this is likely an external import or unresolvable
	return ""
}
