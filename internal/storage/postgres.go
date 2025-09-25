package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// PostgresStore implements storage using PostgreSQL
type PostgresStore struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewPostgresStore creates a new PostgreSQL storage
func NewPostgresStore(dsn string, logger *logrus.Logger) (*PostgresStore, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStore{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// Repository operations

func (s *PostgresStore) SaveRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT INTO repositories (id, owner, name, full_name, url, default_branch,
			language, size, star_count, created_at, updated_at, last_ingested)
		VALUES (:id, :owner, :name, :full_name, :url, :default_branch,
			:language, :size, :star_count, :created_at, :updated_at, :last_ingested)
		ON CONFLICT (id) DO UPDATE SET
			default_branch = EXCLUDED.default_branch,
			language = EXCLUDED.language,
			size = EXCLUDED.size,
			star_count = EXCLUDED.star_count,
			updated_at = EXCLUDED.updated_at,
			last_ingested = EXCLUDED.last_ingested
	`

	_, err := s.db.NamedExecContext(ctx, query, repo)
	if err != nil {
		return fmt.Errorf("save repository: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetRepository(ctx context.Context, repoID string) (*models.Repository, error) {
	var repo models.Repository
	query := `SELECT * FROM repositories WHERE id = $1`

	err := s.db.GetContext(ctx, &repo, query, repoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get repository: %w", err)
	}

	return &repo, nil
}

// Commit operations

func (s *PostgresStore) SaveCommits(ctx context.Context, commits []*models.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO commits (sha, repo_id, author, author_email, message, timestamp)
		VALUES (:sha, :repo_id, :author, :author_email, :message, :timestamp)
		ON CONFLICT (sha) DO NOTHING
	`

	for _, commit := range commits {
		_, err := tx.NamedExecContext(ctx, query, commit)
		if err != nil {
			return fmt.Errorf("save commit: %w", err)
		}
	}

	return tx.Commit()
}

// Risk Sketch operations

func (s *PostgresStore) SaveRiskSketches(ctx context.Context, sketches []*models.RiskSketch) error {
	if len(sketches) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO risk_sketches (
			file_id, repo_id, centrality_score, ownership_score,
			test_coverage, last_updated
		) VALUES (
			:file_id, :repo_id, :centrality_score, :ownership_score,
			:test_coverage, :last_updated
		) ON CONFLICT (file_id) DO UPDATE SET
			centrality_score = EXCLUDED.centrality_score,
			ownership_score = EXCLUDED.ownership_score,
			test_coverage = EXCLUDED.test_coverage,
			last_updated = EXCLUDED.last_updated
	`

	for _, sketch := range sketches {
		_, err := tx.NamedExecContext(ctx, query, sketch)
		if err != nil {
			return fmt.Errorf("save risk sketch: %w", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetRiskSketches(ctx context.Context, repoID string) ([]*models.RiskSketch, error) {
	var sketches []*models.RiskSketch
	query := `SELECT * FROM risk_sketches WHERE repo_id = $1`

	err := s.db.SelectContext(ctx, &sketches, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("get risk sketches: %w", err)
	}

	return sketches, nil
}

// Risk Assessment operations

func (s *PostgresStore) SaveRiskAssessment(ctx context.Context, assessment *models.RiskAssessment) error {
	query := `
		INSERT INTO risk_assessments (
			id, repo_id, commit_sha, level, score,
			blast_radius, test_coverage, timestamp, cache_version
		) VALUES (
			:id, :repo_id, :commit_sha, :level, :score,
			:blast_radius, :test_coverage, :timestamp, :cache_version
		)
	`

	_, err := s.db.NamedExecContext(ctx, query, assessment)
	if err != nil {
		return fmt.Errorf("save risk assessment: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetRiskAssessment(ctx context.Context, id string) (*models.RiskAssessment, error) {
	var assessment models.RiskAssessment
	query := `SELECT * FROM risk_assessments WHERE id = $1`

	err := s.db.GetContext(ctx, &assessment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get risk assessment: %w", err)
	}

	return &assessment, nil
}

// Cache Metadata operations

func (s *PostgresStore) SaveCacheMetadata(ctx context.Context, metadata *models.CacheMetadata) error {
	query := `
		INSERT INTO cache_metadata (
			version, repo_id, last_commit, last_updated,
			size, file_count, sketch_count
		) VALUES (
			:version, :repo_id, :last_commit, :last_updated,
			:size, :file_count, :sketch_count
		) ON CONFLICT (repo_id) DO UPDATE SET
			version = EXCLUDED.version,
			last_commit = EXCLUDED.last_commit,
			last_updated = EXCLUDED.last_updated,
			size = EXCLUDED.size,
			file_count = EXCLUDED.file_count,
			sketch_count = EXCLUDED.sketch_count
	`

	_, err := s.db.NamedExecContext(ctx, query, metadata)
	if err != nil {
		return fmt.Errorf("save cache metadata: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetCacheMetadata(ctx context.Context, repoID string) (*models.CacheMetadata, error) {
	var metadata models.CacheMetadata
	query := `SELECT * FROM cache_metadata WHERE repo_id = $1`

	err := s.db.GetContext(ctx, &metadata, query, repoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get cache metadata: %w", err)
	}

	return &metadata, nil
}

// File operations
func (s *PostgresStore) SaveFiles(ctx context.Context, files []*models.File) error {
	if len(files) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO files (id, repo_id, path, content, size, language, sha, last_modified)
		VALUES (:id, :repo_id, :path, :content, :size, :language, :sha, :last_modified)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			size = EXCLUDED.size,
			sha = EXCLUDED.sha,
			last_modified = EXCLUDED.last_modified
	`

	for _, file := range files {
		_, err := tx.NamedExecContext(ctx, query, file)
		if err != nil {
			return fmt.Errorf("save file: %w", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetFiles(ctx context.Context, repoID string) ([]*models.File, error) {
	var files []*models.File
	query := `SELECT * FROM files WHERE repo_id = $1`

	err := s.db.SelectContext(ctx, &files, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	return files, nil
}

func (s *PostgresStore) GetCommits(ctx context.Context, repoID string, limit int) ([]*models.Commit, error) {
	var commits []*models.Commit
	query := `SELECT * FROM commits WHERE repo_id = $1 ORDER BY timestamp DESC LIMIT $2`

	err := s.db.SelectContext(ctx, &commits, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("get commits: %w", err)
	}

	return commits, nil
}

// Pull Request operations
func (s *PostgresStore) SavePullRequests(ctx context.Context, prs []*models.PullRequest) error {
	if len(prs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO pull_requests (
			id, repo_id, number, title, description, state, author,
			base_branch, head_branch, created_at, merged_at, closed_at
		) VALUES (
			:id, :repo_id, :number, :title, :description, :state, :author,
			:base_branch, :head_branch, :created_at, :merged_at, :closed_at
		) ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			merged_at = EXCLUDED.merged_at,
			closed_at = EXCLUDED.closed_at
	`

	for _, pr := range prs {
		_, err := tx.NamedExecContext(ctx, query, pr)
		if err != nil {
			return fmt.Errorf("save pull request: %w", err)
		}
	}

	return tx.Commit()
}

// Issue operations
func (s *PostgresStore) SaveIssues(ctx context.Context, issues []*models.Issue) error {
	if len(issues) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO issues (
			id, repo_id, number, title, body, state, author,
			is_incident, created_at, closed_at
		) VALUES (
			:id, :repo_id, :number, :title, :body, :state, :author,
			:is_incident, :created_at, :closed_at
		) ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			closed_at = EXCLUDED.closed_at
	`

	for _, issue := range issues {
		_, err := tx.NamedExecContext(ctx, query, issue)
		if err != nil {
			return fmt.Errorf("save issue: %w", err)
		}
	}

	return tx.Commit()
}
