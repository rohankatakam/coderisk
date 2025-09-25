package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// SQLiteStore implements storage using SQLite (for local/development)
type SQLiteStore struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewSQLiteStore creates a new SQLite storage
func NewSQLiteStore(path string, logger *logrus.Logger) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("connect to sqlite: %w", err)
	}

	// Enable foreign keys and WAL mode for better concurrency
	db.Exec("PRAGMA foreign_keys = ON")
	db.Exec("PRAGMA journal_mode = WAL")

	store := &SQLiteStore{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS repositories (
		id TEXT PRIMARY KEY,
		owner TEXT NOT NULL,
		name TEXT NOT NULL,
		full_name TEXT NOT NULL,
		url TEXT,
		default_branch TEXT,
		language TEXT,
		size INTEGER,
		star_count INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		last_ingested DATETIME
	);

	CREATE TABLE IF NOT EXISTS commits (
		sha TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		author TEXT,
		author_email TEXT,
		message TEXT,
		timestamp DATETIME,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		path TEXT NOT NULL,
		content TEXT,
		size INTEGER,
		language TEXT,
		sha TEXT,
		last_modified DATETIME,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE TABLE IF NOT EXISTS risk_sketches (
		file_id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		centrality_score REAL,
		ownership_score REAL,
		test_coverage REAL,
		last_updated DATETIME,
		FOREIGN KEY (repo_id) REFERENCES repositories(id),
		FOREIGN KEY (file_id) REFERENCES files(id)
	);

	CREATE TABLE IF NOT EXISTS risk_assessments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		commit_sha TEXT,
		level TEXT,
		score REAL,
		blast_radius REAL,
		test_coverage REAL,
		timestamp DATETIME,
		cache_version TEXT,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE TABLE IF NOT EXISTS cache_metadata (
		repo_id TEXT PRIMARY KEY,
		version TEXT,
		last_commit TEXT,
		last_updated DATETIME,
		size INTEGER,
		file_count INTEGER,
		sketch_count INTEGER,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE TABLE IF NOT EXISTS pull_requests (
		id INTEGER PRIMARY KEY,
		repo_id TEXT NOT NULL,
		number INTEGER,
		title TEXT,
		description TEXT,
		state TEXT,
		author TEXT,
		base_branch TEXT,
		head_branch TEXT,
		created_at DATETIME,
		merged_at DATETIME,
		closed_at DATETIME,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE TABLE IF NOT EXISTS issues (
		id INTEGER PRIMARY KEY,
		repo_id TEXT NOT NULL,
		number INTEGER,
		title TEXT,
		body TEXT,
		state TEXT,
		author TEXT,
		is_incident INTEGER,
		created_at DATETIME,
		closed_at DATETIME,
		FOREIGN KEY (repo_id) REFERENCES repositories(id)
	);

	CREATE INDEX IF NOT EXISTS idx_commits_repo ON commits(repo_id);
	CREATE INDEX IF NOT EXISTS idx_files_repo ON files(repo_id);
	CREATE INDEX IF NOT EXISTS idx_sketches_repo ON risk_sketches(repo_id);
	CREATE INDEX IF NOT EXISTS idx_assessments_repo ON risk_assessments(repo_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Repository operations
func (s *SQLiteStore) SaveRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT OR REPLACE INTO repositories
		(id, owner, name, full_name, url, default_branch,
		 language, size, star_count, created_at, updated_at, last_ingested)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		repo.ID, repo.Owner, repo.Name, repo.FullName, repo.URL,
		repo.DefaultBranch, repo.Language, repo.Size, repo.StarCount,
		repo.CreatedAt, repo.UpdatedAt, repo.LastIngested)

	return err
}

func (s *SQLiteStore) GetRepository(ctx context.Context, repoID string) (*models.Repository, error) {
	var repo models.Repository
	query := `SELECT * FROM repositories WHERE id = ?`

	err := s.db.GetContext(ctx, &repo, query, repoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &repo, nil
}

// Commit operations
func (s *SQLiteStore) SaveCommits(ctx context.Context, commits []*models.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR IGNORE INTO commits
		(sha, repo_id, author, author_email, message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	for _, commit := range commits {
		_, err := tx.ExecContext(ctx, query,
			commit.SHA, commit.RepoID, commit.Author,
			commit.AuthorEmail, commit.Message, commit.Timestamp)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetCommits(ctx context.Context, repoID string, limit int) ([]*models.Commit, error) {
	var commits []*models.Commit
	query := `SELECT * FROM commits WHERE repo_id = ? ORDER BY timestamp DESC LIMIT ?`

	err := s.db.SelectContext(ctx, &commits, query, repoID, limit)
	if err != nil {
		return nil, err
	}

	return commits, nil
}

// File operations
func (s *SQLiteStore) SaveFiles(ctx context.Context, files []*models.File) error {
	if len(files) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO files
		(id, repo_id, path, content, size, language, sha, last_modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, file := range files {
		_, err := tx.ExecContext(ctx, query,
			file.ID, file.RepoID, file.Path, file.Content,
			file.Size, file.Language, file.SHA, file.LastModified)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetFiles(ctx context.Context, repoID string) ([]*models.File, error) {
	var files []*models.File
	query := `SELECT * FROM files WHERE repo_id = ?`

	err := s.db.SelectContext(ctx, &files, query, repoID)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Risk Sketch operations
func (s *SQLiteStore) SaveRiskSketches(ctx context.Context, sketches []*models.RiskSketch) error {
	if len(sketches) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO risk_sketches
		(file_id, repo_id, centrality_score, ownership_score, test_coverage, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	for _, sketch := range sketches {
		_, err := tx.ExecContext(ctx, query,
			sketch.FileID, sketch.RepoID, sketch.CentralityScore,
			sketch.OwnershipScore, sketch.TestCoverage, sketch.LastUpdated)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetRiskSketches(ctx context.Context, repoID string) ([]*models.RiskSketch, error) {
	var sketches []*models.RiskSketch

	query := `SELECT * FROM risk_sketches`
	args := []interface{}{}

	if repoID != "" {
		query += ` WHERE repo_id = ?`
		args = append(args, repoID)
	}

	err := s.db.SelectContext(ctx, &sketches, query, args...)
	if err != nil {
		return nil, err
	}

	return sketches, nil
}

// Additional operations for SQLite
func (s *SQLiteStore) SavePullRequests(ctx context.Context, prs []*models.PullRequest) error {
	if len(prs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO pull_requests
		(id, repo_id, number, title, description, state, author,
		 base_branch, head_branch, created_at, merged_at, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, pr := range prs {
		_, err := tx.ExecContext(ctx, query,
			pr.ID, pr.RepoID, pr.Number, pr.Title, pr.Description,
			pr.State, pr.Author, pr.BaseBranch, pr.HeadBranch,
			pr.CreatedAt, pr.MergedAt, pr.ClosedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) SaveIssues(ctx context.Context, issues []*models.Issue) error {
	if len(issues) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO issues
		(id, repo_id, number, title, body, state, author, is_incident, created_at, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, issue := range issues {
		_, err := tx.ExecContext(ctx, query,
			issue.ID, issue.RepoID, issue.Number, issue.Title, issue.Body,
			issue.State, issue.Author, issue.IsIncident, issue.CreatedAt, issue.ClosedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Implement remaining interface methods...
func (s *SQLiteStore) SaveRiskAssessment(ctx context.Context, assessment *models.RiskAssessment) error {
	query := `
		INSERT INTO risk_assessments
		(id, repo_id, commit_sha, level, score, blast_radius, test_coverage, timestamp, cache_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		assessment.ID, assessment.RepoID, assessment.CommitSHA, assessment.Level,
		assessment.Score, assessment.BlastRadius, assessment.TestCoverage,
		assessment.Timestamp, assessment.CacheVersion)

	return err
}

func (s *SQLiteStore) GetRiskAssessment(ctx context.Context, id string) (*models.RiskAssessment, error) {
	var assessment models.RiskAssessment
	query := `SELECT * FROM risk_assessments WHERE id = ?`

	err := s.db.GetContext(ctx, &assessment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &assessment, nil
}

func (s *SQLiteStore) SaveCacheMetadata(ctx context.Context, metadata *models.CacheMetadata) error {
	query := `
		INSERT OR REPLACE INTO cache_metadata
		(repo_id, version, last_commit, last_updated, size, file_count, sketch_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		metadata.RepoID, metadata.Version, metadata.LastCommit,
		metadata.LastUpdated, metadata.Size, metadata.FileCount, metadata.SketchCount)

	return err
}

func (s *SQLiteStore) GetCacheMetadata(ctx context.Context, repoID string) (*models.CacheMetadata, error) {
	var metadata models.CacheMetadata
	query := `SELECT * FROM cache_metadata WHERE repo_id = ?`

	err := s.db.GetContext(ctx, &metadata, query, repoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &metadata, nil
}
