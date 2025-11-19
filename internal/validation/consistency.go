package validation

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ValidationResult contains the results of a consistency check
type ValidationResult struct {
	EntityType      string
	PostgresCount   int64
	Neo4jCount      int64
	VariancePercent float64
	PassedThreshold bool
}

// ConsistencyValidator validates data consistency between Postgres and Neo4j
type ConsistencyValidator struct {
	db     *sql.DB
	driver neo4j.DriverWithContext
	logger *slog.Logger
}

// NewConsistencyValidator creates a new consistency validator
func NewConsistencyValidator(db *sql.DB, driver neo4j.DriverWithContext) *ConsistencyValidator {
	return &ConsistencyValidator{
		db:     db,
		driver: driver,
		logger: slog.Default().With("component", "validation"),
	}
}

// ValidateAfterIngest validates entity counts after crisk-ingest
func (v *ConsistencyValidator) ValidateAfterIngest(ctx context.Context, repoID int64) ([]ValidationResult, error) {
	var results []ValidationResult

	// Validate Commits
	commitResult, err := v.validateCommits(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate commits: %w", err)
	}
	results = append(results, commitResult)

	// Validate Files
	fileResult, err := v.validateFiles(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate files: %w", err)
	}
	results = append(results, fileResult)

	// Validate Issues
	issueResult, err := v.validateIssues(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate issues: %w", err)
	}
	results = append(results, issueResult)

	// Validate CodeBlocks
	codeBlockResult, err := v.validateCodeBlocks(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate code blocks: %w", err)
	}
	results = append(results, codeBlockResult)

	return results, nil
}

// ValidateAfterAtomize validates entity counts after crisk-atomize
func (v *ConsistencyValidator) ValidateAfterAtomize(ctx context.Context, repoID int64) ([]ValidationResult, error) {
	var results []ValidationResult

	// Validate CodeBlocks
	blockResult, err := v.validateCodeBlocks(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate code blocks: %w", err)
	}
	results = append(results, blockResult)

	return results, nil
}

// validateCommits checks commit counts
func (v *ConsistencyValidator) validateCommits(ctx context.Context, repoID int64) (ValidationResult, error) {
	// Count in Postgres
	var pgCount int64
	err := v.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM github_commits WHERE repo_id = $1",
		repoID,
	).Scan(&pgCount)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("postgres query failed: %w", err)
	}

	// Count in Neo4j
	neoCount, err := v.countNeo4jNodes(ctx, "Commit", repoID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("neo4j query failed: %w", err)
	}

	variance := 0.0
	if pgCount > 0 {
		variance = float64(neoCount) / float64(pgCount) * 100.0
	}
	passed := variance >= 95.0

	return ValidationResult{
		EntityType:      "Commits",
		PostgresCount:   pgCount,
		Neo4jCount:      neoCount,
		VariancePercent: variance,
		PassedThreshold: passed,
	}, nil
}

// validateFiles checks file counts
func (v *ConsistencyValidator) validateFiles(ctx context.Context, repoID int64) (ValidationResult, error) {
	// Count in Postgres
	var pgCount int64
	err := v.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM file_identity_map WHERE repo_id = $1",
		repoID,
	).Scan(&pgCount)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("postgres query failed: %w", err)
	}

	// Count in Neo4j
	neoCount, err := v.countNeo4jNodes(ctx, "File", repoID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("neo4j query failed: %w", err)
	}

	variance := 0.0
	if pgCount > 0 {
		variance = float64(neoCount) / float64(pgCount) * 100.0
	}
	passed := variance >= 95.0

	return ValidationResult{
		EntityType:      "Files",
		PostgresCount:   pgCount,
		Neo4jCount:      neoCount,
		VariancePercent: variance,
		PassedThreshold: passed,
	}, nil
}

// validateIssues checks issue counts
func (v *ConsistencyValidator) validateIssues(ctx context.Context, repoID int64) (ValidationResult, error) {
	// Count in Postgres
	var pgCount int64
	err := v.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM github_issues WHERE repo_id = $1",
		repoID,
	).Scan(&pgCount)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("postgres query failed: %w", err)
	}

	// Count in Neo4j
	neoCount, err := v.countNeo4jNodes(ctx, "Issue", repoID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("neo4j query failed: %w", err)
	}

	variance := 0.0
	if pgCount > 0 {
		variance = float64(neoCount) / float64(pgCount) * 100.0
	}
	passed := variance >= 95.0

	return ValidationResult{
		EntityType:      "Issues",
		PostgresCount:   pgCount,
		Neo4jCount:      neoCount,
		VariancePercent: variance,
		PassedThreshold: passed,
	}, nil
}

// validateCodeBlocks checks code block counts
func (v *ConsistencyValidator) validateCodeBlocks(ctx context.Context, repoID int64) (ValidationResult, error) {
	// Count in Postgres
	var pgCount int64
	err := v.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1",
		repoID,
	).Scan(&pgCount)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("postgres query failed: %w", err)
	}

	// Count in Neo4j
	neoCount, err := v.countNeo4jNodes(ctx, "CodeBlock", repoID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("neo4j query failed: %w", err)
	}

	variance := 0.0
	if pgCount > 0 {
		variance = float64(neoCount) / float64(pgCount) * 100.0
	}
	passed := variance >= 95.0

	return ValidationResult{
		EntityType:      "CodeBlocks",
		PostgresCount:   pgCount,
		Neo4jCount:      neoCount,
		VariancePercent: variance,
		PassedThreshold: passed,
	}, nil
}

// countNeo4jNodes is a helper to count nodes of a specific label
func (v *ConsistencyValidator) countNeo4jNodes(ctx context.Context, label string, repoID int64) (int64, error) {
	session := v.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := fmt.Sprintf("MATCH (n:%s {repo_id: $repoID}) RETURN count(n) as count", label)

	result, err := session.Run(ctx, query, map[string]any{"repoID": repoID})
	if err != nil {
		return 0, fmt.Errorf("neo4j query failed: %w", err)
	}

	var count int64 = 0
	if result.Next(ctx) {
		record := result.Record()
		if countVal, ok := record.Get("count"); ok {
			count = countVal.(int64)
		}
	}

	return count, nil
}

// LogResults logs validation results in a formatted way
func LogResults(results []ValidationResult) {
	logger := slog.Default()

	logger.Info("═══ VALIDATION METRICS ═══")

	allPassed := true
	for _, r := range results {
		logger.Info(fmt.Sprintf("%-12s Postgres=%d, Neo4j=%d, Sync=%.1f%%",
			r.EntityType+":", r.PostgresCount, r.Neo4jCount, r.VariancePercent))

		if !r.PassedThreshold {
			allPassed = false
		}
	}

	if allPassed {
		logger.Info("✓ All entities within acceptable variance (≥95%)")
	} else {
		logger.Warn("⚠️  WARNING: Sync variance below 95% threshold - manual review required")
	}
}

// UpdateRepositoryStatus updates the repository status in Postgres based on validation
func UpdateRepositoryStatus(ctx context.Context, db *sql.DB, repoID int64, results []ValidationResult) error {
	allPassed := true
	for _, r := range results {
		if !r.PassedThreshold {
			allPassed = false
			break
		}
	}

	status := "completed"
	if !allPassed {
		status = "needs_sync"
	}

	_, err := db.ExecContext(ctx,
		"UPDATE github_repositories SET ingestion_status = $1 WHERE id = $2",
		status, repoID,
	)

	return err
}
