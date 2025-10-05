package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coderisk/coderisk-go/internal/graph"
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
	config      *ProcessorConfig
	graphClient graph.Backend
}

// NewProcessor creates a new repository processor
func NewProcessor(config *ProcessorConfig, graphClient graph.Backend) *Processor {
	if config == nil {
		config = DefaultProcessorConfig()
	}
	return &Processor{
		config:      config,
		graphClient: graphClient,
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

	return nil
}

// entityToGraphNode converts CodeEntity to graph node
func entityToGraphNode(entity treesitter.CodeEntity) graph.GraphNode {
	properties := make(map[string]interface{})

	properties["name"] = entity.Name
	properties["file_path"] = entity.FilePath
	properties["language"] = entity.Language

	// Add composite unique ID for graph operations
	// Format: "filepath:name:line" for true uniqueness (handles multiple same-named functions in file)
	properties["unique_id"] = fmt.Sprintf("%s:%s:%d", entity.FilePath, entity.Name, entity.StartLine)

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

	// Determine label
	label := "File"
	switch entity.Type {
	case "function":
		label = "Function"
	case "class":
		label = "Class"
	case "import":
		label = "Import"
	}

	return graph.GraphNode{
		ID:         properties["unique_id"].(string),
		Label:      label,
		Properties: properties,
	}
}
