package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CommitSyncer syncs Commit nodes from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 502-523 - Commit node spec
type CommitSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewCommitSyncer creates a new commit syncer
func NewCommitSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *CommitSyncer {
	return &CommitSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// Commit represents a commit from PostgreSQL
type Commit struct {
	SHA              string
	RepoID           int64
	Message          string
	AuthorDate       time.Time
	CommitterDate    *time.Time
	TopologicalIndex int
	FilesChanged     int
	Additions        int
	Deletions        int
}

// SyncMissingCommits syncs Commit nodes from PostgreSQL to Neo4j (incremental)
// Returns: (nodes created, error)
func (s *CommitSyncer) SyncMissingCommits(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  📝 Syncing Commit nodes from PostgreSQL → Neo4j...")

	// Step 1: Get all commits from PostgreSQL
	commits, err := s.getAllCommits(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres commits: %w", err)
	}
	log.Printf("     PostgreSQL: %d Commits", len(commits))

	// Step 2: Get existing commit SHAs from Neo4j
	neo4jSHAs, err := s.getNeo4jCommitSHAs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j commits: %w", err)
	}
	log.Printf("     Neo4j: %d Commits", len(neo4jSHAs))

	// Step 3: Compute delta (commits not in Neo4j)
	neo4jSHASet := make(map[string]bool, len(neo4jSHAs))
	for _, sha := range neo4jSHAs {
		neo4jSHASet[sha] = true
	}

	var missingCommits []Commit
	for _, commit := range commits {
		if !neo4jSHASet[commit.SHA] {
			missingCommits = append(missingCommits, commit)
		}
	}

	if len(missingCommits) == 0 {
		log.Printf("     ✓ All Commits already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d Commits need syncing", len(missingCommits))

	// Step 4: Create missing Commit nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, commit := range missingCommits {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create Commit node with MERGE (idempotent)
			// Reference: DATA_SCHEMA_REFERENCE.md line 858 - UNIQUE(Commit.repo_id, Commit.sha)
			query := `
				MERGE (c:Commit {sha: $sha, repo_id: $repoID})
				SET c.message = $message,
				    c.author_date = datetime($authorDate),
				    c.committer_date = CASE WHEN $committerDate IS NOT NULL THEN datetime($committerDate) ELSE NULL END,
				    c.topological_index = $topologicalIndex,
				    c.files_changed = $filesChanged,
				    c.additions = $additions,
				    c.deletions = $deletions
				RETURN c.sha as sha`

			// Handle NULL committer_date
			var committerDateStr interface{}
			if commit.CommitterDate != nil {
				committerDateStr = commit.CommitterDate.Format(time.RFC3339)
			} else {
				committerDateStr = nil
			}

			params := map[string]any{
				"sha":              commit.SHA,
				"repoID":           commit.RepoID,
				"message":          commit.Message,
				"authorDate":       commit.AuthorDate.Format(time.RFC3339),
				"committerDate":    committerDateStr,
				"topologicalIndex": commit.TopologicalIndex,
				"filesChanged":     commit.FilesChanged,
				"additions":        commit.Additions,
				"deletions":        commit.Deletions,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync commit %s: %v", commit.SHA[:8], err)
			continue
		}
		synced++

		// Log progress every 500 commits
		if synced%500 == 0 {
			log.Printf("     → Synced %d/%d commits", synced, len(missingCommits))
		}
	}

	log.Printf("     ✓ Synced %d Commits to Neo4j", synced)
	return synced, nil
}

// getAllCommits returns all commits from PostgreSQL
func (s *CommitSyncer) getAllCommits(ctx context.Context, repoID int64) ([]Commit, error) {
	query := `
		SELECT
			sha,
			repo_id,
			message,
			author_date,
			committer_date,
			topological_index,
			files_changed,
			additions,
			deletions
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY topological_index ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		err := rows.Scan(
			&c.SHA,
			&c.RepoID,
			&c.Message,
			&c.AuthorDate,
			&c.CommitterDate,
			&c.TopologicalIndex,
			&c.FilesChanged,
			&c.Additions,
			&c.Deletions,
		)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}

	return commits, rows.Err()
}

// getNeo4jCommitSHAs returns existing commit SHAs from Neo4j
func (s *CommitSyncer) getNeo4jCommitSHAs(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (c:Commit {repo_id: $repoID}) RETURN c.sha as sha`
		result, err := tx.Run(ctx, query, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		return result.Collect(ctx)
	})

	if err != nil {
		return nil, err
	}

	records := result.([]*neo4j.Record)
	var shas []string
	for _, record := range records {
		sha, ok := record.Get("sha")
		if !ok {
			continue
		}
		shas = append(shas, sha.(string))
	}

	return shas, nil
}
