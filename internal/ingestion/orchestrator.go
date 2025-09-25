package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/coderisk/coderisk-go/internal/github"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/storage"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Orchestrator coordinates the ingestion process
type Orchestrator struct {
	extractor *github.Extractor
	store     storage.Store
	logger    *logrus.Logger
	config    *config.Config
}

// NewOrchestrator creates a new ingestion orchestrator
func NewOrchestrator(
	extractor *github.Extractor,
	store storage.Store,
	logger *logrus.Logger,
	config *config.Config,
) *Orchestrator {
	return &Orchestrator{
		extractor: extractor,
		store:     store,
		logger:    logger,
		config:    config,
	}
}

// IngestionResult contains the results of an ingestion
type IngestionResult struct {
	RepoID      string
	FileCount   int
	CommitCount int
	PRCount     int
	IssueCount  int
	SketchCount int
	LastCommit  string
	Cost        float64
	Duration    time.Duration
}

// IngestRepository performs complete repository ingestion
func (o *Orchestrator) IngestRepository(ctx context.Context, owner, name string) (*IngestionResult, error) {
	startTime := time.Now()
	o.logger.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
	}).Info("Starting repository ingestion")

	result := &IngestionResult{
		RepoID: fmt.Sprintf("%s/%s", owner, name),
	}

	// Phase 1: Extract GitHub data
	extractResult, err := o.extractor.ExtractRepository(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	result.FileCount = len(extractResult.Files)
	result.CommitCount = len(extractResult.Commits)
	result.PRCount = len(extractResult.PullRequests)
	result.IssueCount = len(extractResult.Issues)

	// Phase 2: Store raw data
	if err := o.storeRawData(ctx, extractResult); err != nil {
		return nil, fmt.Errorf("failed to store raw data: %w", err)
	}

	// Phase 3: Generate risk sketches
	sketches, err := o.generateRiskSketches(ctx, extractResult)
	if err != nil {
		return nil, fmt.Errorf("failed to generate risk sketches: %w", err)
	}
	result.SketchCount = len(sketches)

	// Phase 4: Store risk sketches
	if err := o.store.SaveRiskSketches(ctx, sketches); err != nil {
		return nil, fmt.Errorf("failed to save risk sketches: %w", err)
	}

	// Calculate cost (simplified)
	result.Cost = o.calculateCost(extractResult)

	// Get last commit
	if len(extractResult.Commits) > 0 {
		result.LastCommit = extractResult.Commits[0].SHA
	}

	result.Duration = time.Since(startTime)

	o.logger.WithFields(logrus.Fields{
		"duration": result.Duration.String(),
		"files":    result.FileCount,
		"commits":  result.CommitCount,
		"sketches": result.SketchCount,
		"cost":     result.Cost,
	}).Info("Repository ingestion completed")

	return result, nil
}

// IncrementalIngest performs incremental ingestion
func (o *Orchestrator) IncrementalIngest(ctx context.Context, owner, name string, since time.Time) (*IngestionResult, error) {
	o.logger.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
		"since": since,
	}).Info("Starting incremental ingestion")

	result := &IngestionResult{
		RepoID: fmt.Sprintf("%s/%s", owner, name),
	}

	// Extract only new data
	extractResult, err := o.extractor.IncrementalExtract(ctx, owner, name, since)
	if err != nil {
		return nil, fmt.Errorf("incremental extraction failed: %w", err)
	}

	result.CommitCount = len(extractResult.Commits)
	result.PRCount = len(extractResult.PullRequests)
	result.IssueCount = len(extractResult.Issues)

	// Update risk sketches for changed files
	if len(extractResult.Commits) > 0 {
		changedFiles := o.getChangedFiles(extractResult.Commits)
		sketches, err := o.updateRiskSketches(ctx, changedFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to update risk sketches: %w", err)
		}
		result.SketchCount = len(sketches)

		if err := o.store.SaveRiskSketches(ctx, sketches); err != nil {
			return nil, fmt.Errorf("failed to save updated sketches: %w", err)
		}
	}

	// Store new commits
	if err := o.store.SaveCommits(ctx, extractResult.Commits); err != nil {
		return nil, fmt.Errorf("failed to save commits: %w", err)
	}

	result.Cost = o.calculateIncrementalCost(extractResult)

	o.logger.WithFields(logrus.Fields{
		"commits":  result.CommitCount,
		"prs":      result.PRCount,
		"issues":   result.IssueCount,
		"sketches": result.SketchCount,
		"cost":     result.Cost,
	}).Info("Incremental ingestion completed")

	return result, nil
}

func (o *Orchestrator) storeRawData(ctx context.Context, data *github.ExtractResult) error {
	g, ctx := errgroup.WithContext(ctx)

	// Store repository metadata
	g.Go(func() error {
		data.Repository.LastIngested = time.Now()
		return o.store.SaveRepository(ctx, data.Repository)
	})

	// Store commits
	g.Go(func() error {
		return o.store.SaveCommits(ctx, data.Commits)
	})

	// Store files
	g.Go(func() error {
		return o.store.SaveFiles(ctx, data.Files)
	})

	// Store pull requests
	g.Go(func() error {
		return o.store.SavePullRequests(ctx, data.PullRequests)
	})

	// Store issues
	g.Go(func() error {
		return o.store.SaveIssues(ctx, data.Issues)
	})

	return g.Wait()
}

func (o *Orchestrator) generateRiskSketches(ctx context.Context, data *github.ExtractResult) ([]*models.RiskSketch, error) {
	o.logger.Info("Generating risk sketches")

	sketches := make([]*models.RiskSketch, 0, len(data.Files))

	for _, file := range data.Files {
		sketch := &models.RiskSketch{
			FileID:      file.ID,
			RepoID:      file.RepoID,
			LastUpdated: time.Now(),
		}

		// Calculate centrality (simplified)
		sketch.CentralityScore = o.calculateCentrality(file, data.Files)

		// Calculate ownership score (simplified)
		sketch.OwnershipScore = o.calculateOwnership(file, data.Commits)

		// Calculate test coverage (simplified)
		sketch.TestCoverage = o.estimateTestCoverage(file)

		// Extract incident history
		sketch.IncidentHistory = o.extractIncidentHistory(file, data.Issues)

		sketches = append(sketches, sketch)
	}

	return sketches, nil
}

func (o *Orchestrator) updateRiskSketches(ctx context.Context, changedFiles []string) ([]*models.RiskSketch, error) {
	// Load existing sketches
	sketches, err := o.store.GetRiskSketches(ctx, "")
	if err != nil {
		return nil, err
	}

	// Update sketches for changed files
	changeMap := make(map[string]bool)
	for _, file := range changedFiles {
		changeMap[file] = true
	}

	updatedSketches := []*models.RiskSketch{}
	for _, sketch := range sketches {
		if changeMap[sketch.FileID] {
			// Update sketch metrics
			sketch.LastUpdated = time.Now()
			// Recalculate metrics...
			updatedSketches = append(updatedSketches, sketch)
		}
	}

	return updatedSketches, nil
}

func (o *Orchestrator) getChangedFiles(commits []*models.Commit) []string {
	fileSet := make(map[string]bool)
	for _, commit := range commits {
		for _, file := range commit.Files {
			fileSet[file.Path] = true
		}
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

func (o *Orchestrator) calculateCost(data *github.ExtractResult) float64 {
	// Simplified cost calculation
	// $0.01 per 100 files for embeddings
	// $0.05 per 1000 commits for analysis
	// $0.02 per 100 PRs/issues

	fileCost := float64(len(data.Files)) / 100 * 0.01
	commitCost := float64(len(data.Commits)) / 1000 * 0.05
	prIssueCost := float64(len(data.PullRequests)+len(data.Issues)) / 100 * 0.02

	return fileCost + commitCost + prIssueCost + 15.0 // Base cost
}

func (o *Orchestrator) calculateIncrementalCost(data *github.ExtractResult) float64 {
	// Lower cost for incremental updates
	commitCost := float64(len(data.Commits)) / 1000 * 0.02
	prIssueCost := float64(len(data.PullRequests)+len(data.Issues)) / 100 * 0.01

	return commitCost + prIssueCost
}

// Simplified metric calculations (these would be more sophisticated in production)

func (o *Orchestrator) calculateCentrality(file *models.File, allFiles []*models.File) float64 {
	// Simplified: based on file path depth and common directories
	return 0.5
}

func (o *Orchestrator) calculateOwnership(file *models.File, commits []*models.Commit) float64 {
	// Simplified: would analyze commit history for file
	return 0.7
}

func (o *Orchestrator) estimateTestCoverage(file *models.File) float64 {
	// Simplified: would analyze test files and coverage reports
	if isTestFile(file.Path) {
		return 1.0
	}
	return 0.3
}

func (o *Orchestrator) extractIncidentHistory(file *models.File, issues []*models.Issue) []string {
	// Simplified: would correlate issues with file changes
	return []string{}
}

func isTestFile(path string) bool {
	return len(path) > 8 && path[len(path)-8:] == "_test.go"
}
