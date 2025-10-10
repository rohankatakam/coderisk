package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/temporal"
	"github.com/coderisk/coderisk-go/internal/treesitter"
)

// ProcessorConfig holds configuration for repository processing
type ProcessorConfig struct {
	Workers    int           // Number of concurrent parsers (default: 20)
	Timeout    time.Duration // Per-file parsing timeout (default: 30s)
	GraphBatch int           // Batch size for graph writes (default: 100)
}

// DefaultProcessorConfig returns default configuration
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		Workers:    20,
		Timeout:    30 * time.Second,
		GraphBatch: 100,
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
	startTime := time.Now()

	slog.Info("starting repository processing",
		"repo", repoURL,
		"workers", p.config.Workers,
	)

	result := &ProcessResult{
		Errors: []error{},
	}

	// Step 1: Clone repository
	repoPath, err := CloneRepository(ctx, repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	result.RepoPath = repoPath

	slog.Info("repository cloned", "path", repoPath)

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

	// Step 5: Build graph
	if p.graphClient != nil {
		if err := p.buildGraph(ctx, allEntities); err != nil {
			return nil, fmt.Errorf("failed to build graph: %w", err)
		}
		slog.Info("graph construction complete")

		// Step 6: Add Layer 2 (Temporal Analysis)
		// 12-Factor Principle - Factor 8: Concurrency via process model
		// Use timeout to prevent blocking the entire init-local operation
		slog.Info("starting temporal analysis", "window_days", 90)

		// Add timeout for git history parsing
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		// Parse git history with timeout
		type parseResult struct {
			commits []temporal.Commit
			err     error
		}

		resultChan := make(chan parseResult, 1)
		go func() {
			commits, err := temporal.ParseGitHistory(repoPath, 90)
			resultChan <- parseResult{commits: commits, err: err}
		}()

		var commits []temporal.Commit
		var err error

		select {
		case <-timeoutCtx.Done():
			slog.Warn("temporal analysis timeout",
				"timeout", "3 minutes",
				"recommendation", "reduce window_days or optimize git parsing")
			// Don't fail entire init-local, continue without Layer 2
		case res := <-resultChan:
			commits = res.commits
			err = res.err
		}

		if err != nil {
			slog.Error("temporal analysis failed", "error", err)
			// Continue without Layer 2
		} else if len(commits) == 0 {
			slog.Warn("no commits found in git history", "window_days", 90)
		} else {
			slog.Info("git history parsed",
				"commits", len(commits),
				"window_days", 90)

			// Calculate co-changes and ownership
			developers := temporal.ExtractDevelopers(commits)
			coChanges := temporal.CalculateCoChanges(commits, 0.3) // 30% frequency threshold

			slog.Info("co-changes calculated",
				"total_pairs", len(coChanges),
				"min_frequency", 0.3)

			// Convert relative git paths to absolute paths for graph matching
			// Git history returns paths like "apps/web/src/page.tsx"
			// But File nodes have absolute paths like "/Users/.../apps/web/src/page.tsx"
			slog.Info("converting co-change paths",
				"before_conversion", len(coChanges),
				"repo_path", repoPath)

			// Log sample paths before conversion (debug)
			if len(coChanges) > 0 {
				slog.Debug("sample co-change before conversion",
					"fileA", coChanges[0].FileA,
					"fileB", coChanges[0].FileB)
			}

			coChanges = p.convertCoChangePaths(coChanges, repoPath)

			// Log sample paths after conversion (debug)
			if len(coChanges) > 0 {
				slog.Info("sample co-change after conversion",
					"fileA", coChanges[0].FileA,
					"fileB", coChanges[0].FileB,
					"total_pairs", len(coChanges))
			}

			// Store in Neo4j
			if p.graphBuilder != nil && len(coChanges) > 0 {
				slog.Info("storing CO_CHANGED edges", "count", len(coChanges))

				stats, err := p.graphBuilder.AddLayer2CoChangedEdges(ctx, coChanges)
				if err != nil {
					slog.Error("failed to store CO_CHANGED edges", "error", err, "count", len(coChanges))
					// Continue - don't fail entire init-local
				} else {
					slog.Info("temporal analysis complete",
						"commits", len(commits),
						"developers", len(developers),
						"co_change_edges", stats.Edges)

					// Verify edges were actually created
					verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer verifyCancel()

					verifyCount, verifyErr := p.verifyCoChangedEdges(verifyCtx)
					if verifyErr != nil {
						slog.Error("edge verification failed", "error", verifyErr)
					} else if verifyCount != stats.Edges {
						slog.Warn("edge count mismatch",
							"expected", stats.Edges,
							"actual", verifyCount,
							"possible_transaction_issue", true)
					} else {
						slog.Info("edge verification passed", "count", verifyCount)
					}
				}
			} else if len(coChanges) == 0 {
				slog.Warn("no co-changes found", "min_frequency", 0.3)
			}
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

// convertCoChangePaths converts relative git paths to absolute file paths
// Git history returns paths like "apps/web/src/page.tsx"
// File nodes use absolute paths like "/Users/.../apps/web/src/page.tsx"
func (p *Processor) convertCoChangePaths(coChanges []temporal.CoChangeResult, repoPath string) []temporal.CoChangeResult {
	converted := make([]temporal.CoChangeResult, 0, len(coChanges))

	for _, cc := range coChanges {
		// Convert relative paths to absolute by joining with repo path
		absoluteA := fmt.Sprintf("%s/%s", repoPath, cc.FileA)
		absoluteB := fmt.Sprintf("%s/%s", repoPath, cc.FileB)

		converted = append(converted, temporal.CoChangeResult{
			FileA:      absoluteA,
			FileB:      absoluteB,
			Frequency:  cc.Frequency,
			CoChanges:  cc.CoChanges,
			WindowDays: cc.WindowDays,
		})
	}

	return converted
}

// verifyCoChangedEdges queries Neo4j to confirm edges were created
// 12-Factor Principle - Factor 10: Dev/prod parity
// Verification ensures data consistency across environments
func (p *Processor) verifyCoChangedEdges(ctx context.Context) (int, error) {
	if p.graphClient == nil {
		return 0, fmt.Errorf("graph backend not available")
	}

	// Query Neo4j directly to count CO_CHANGED edges
	query := "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count"

	result, err := p.graphClient.Query(query)
	if err != nil {
		return 0, fmt.Errorf("verification query failed: %w", err)
	}

	// Extract count from result
	// Neo4j returns results as []map[string]interface{}
	if resultSlice, ok := result.([]map[string]interface{}); ok && len(resultSlice) > 0 {
		if count, ok := resultSlice[0]["count"].(int64); ok {
			return int(count), nil
		}
	}

	return 0, fmt.Errorf("unexpected query result format")
}

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
func (p *Processor) buildGraph(ctx context.Context, entities []treesitter.CodeEntity) error {
	// Batch entities for efficient graph writes
	batchSize := p.config.GraphBatch

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]

		// Create nodes
		for _, entity := range batch {
			node := entityToGraphNode(entity)
			if _, err := p.graphClient.CreateNode(node); err != nil {
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

	// Step 2: Create edges (CONTAINS, IMPORTS)
	slog.Info("creating graph edges", "total_entities", len(entities))
	if err := p.createEdges(ctx, entities); err != nil {
		return fmt.Errorf("failed to create edges: %w", err)
	}

	return nil
}

// createEdges creates relationships between entities
func (p *Processor) createEdges(ctx context.Context, entities []treesitter.CodeEntity) error {
	var edges []graph.GraphEdge

	// Group entities by file for efficient edge creation
	fileToFunctions := make(map[string][]treesitter.CodeEntity)
	fileToClasses := make(map[string][]treesitter.CodeEntity)
	fileToImports := make(map[string][]treesitter.CodeEntity)

	for _, entity := range entities {
		switch entity.Type {
		case "function":
			fileToFunctions[entity.FilePath] = append(fileToFunctions[entity.FilePath], entity)
		case "class":
			fileToClasses[entity.FilePath] = append(fileToClasses[entity.FilePath], entity)
		case "import":
			fileToImports[entity.FilePath] = append(fileToImports[entity.FilePath], entity)
		}
	}

	// Create CONTAINS edges: File -> Function, File -> Class
	for filePath, functions := range fileToFunctions {
		for _, fn := range functions {
			edges = append(edges, graph.GraphEdge{
				From:  fmt.Sprintf("file:%s", filePath),
				To:    fmt.Sprintf("function:%s:%s:%d", fn.FilePath, fn.Name, fn.StartLine),
				Label: "CONTAINS",
				Properties: map[string]interface{}{
					"entity_type": "function",
				},
			})
		}
	}

	for filePath, classes := range fileToClasses {
		for _, cls := range classes {
			edges = append(edges, graph.GraphEdge{
				From:  fmt.Sprintf("file:%s", filePath),
				To:    fmt.Sprintf("class:%s:%s:%d", cls.FilePath, cls.Name, cls.StartLine),
				Label: "CONTAINS",
				Properties: map[string]interface{}{
					"entity_type": "class",
				},
			})
		}
	}

	// Create IMPORTS edges: File -> Import
	for filePath, imports := range fileToImports {
		for _, imp := range imports {
			edges = append(edges, graph.GraphEdge{
				From:  fmt.Sprintf("file:%s", filePath),
				To:    fmt.Sprintf("import:%s:%s:%d", imp.FilePath, imp.Name, imp.StartLine),
				Label: "IMPORTS",
				Properties: map[string]interface{}{
					"import_path": imp.ImportPath,
				},
			})
		}
	}

	// Batch create edges
	if len(edges) > 0 {
		slog.Info("creating edges", "count", len(edges))
		if err := p.graphClient.CreateEdges(edges); err != nil {
			return fmt.Errorf("failed to create edges: %w", err)
		}
	}

	return nil
}

// entityToGraphNode converts CodeEntity to graph node
func entityToGraphNode(entity treesitter.CodeEntity) graph.GraphNode {
	properties := make(map[string]interface{})

	properties["name"] = entity.Name
	properties["file_path"] = entity.FilePath
	properties["language"] = entity.Language

	// Determine label first to properly set unique_id
	label := "File"
	switch entity.Type {
	case "function":
		label = "Function"
	case "class":
		label = "Class"
	case "import":
		label = "Import"
	}

	// Generate unique_id based on entity type
	var uniqueID string
	if label == "File" {
		// For Files: unique_id is the file path (no name/line needed)
		uniqueID = entity.FilePath
	} else {
		// For Functions/Classes/Imports: use composite key "filepath:name:line"
		// This handles multiple same-named functions in a file
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
