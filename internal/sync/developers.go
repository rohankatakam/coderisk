package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// DeveloperSyncer syncs Developer nodes from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 484-499 - Developer node spec
type DeveloperSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewDeveloperSyncer creates a new developer syncer
func NewDeveloperSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *DeveloperSyncer {
	return &DeveloperSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// Developer represents a developer from PostgreSQL
type Developer struct {
	Email            string
	Name             string
	RepoID           int64
	CommitCount      int
	FirstCommitDate  time.Time
	LastCommitDate   time.Time
}

// SyncMissingDevelopers syncs Developer nodes from PostgreSQL to Neo4j (incremental)
// Returns: (nodes created, error)
func (s *DeveloperSyncer) SyncMissingDevelopers(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  👥 Syncing Developer nodes from PostgreSQL → Neo4j...")

	// Step 1: Get all developers from PostgreSQL (aggregated from commits)
	developers, err := s.getAllDevelopers(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres developers: %w", err)
	}
	log.Printf("     PostgreSQL: %d Developers", len(developers))

	// Step 2: Get existing developer emails from Neo4j
	neo4jEmails, err := s.getNeo4jDeveloperEmails(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j developers: %w", err)
	}
	log.Printf("     Neo4j: %d Developers", len(neo4jEmails))

	// Step 3: Compute delta (developers not in Neo4j)
	neo4jEmailSet := make(map[string]bool, len(neo4jEmails))
	for _, email := range neo4jEmails {
		neo4jEmailSet[email] = true
	}

	var missingDevelopers []Developer
	for _, dev := range developers {
		if !neo4jEmailSet[dev.Email] {
			missingDevelopers = append(missingDevelopers, dev)
		}
	}

	if len(missingDevelopers) == 0 {
		log.Printf("     ✓ All Developers already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d Developers need syncing", len(missingDevelopers))

	// Step 4: Create missing Developer nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, dev := range missingDevelopers {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create Developer node with MERGE (idempotent)
			// Reference: DATA_SCHEMA_REFERENCE.md line 857 - UNIQUE(Developer.email)
			query := `
				MERGE (d:Developer {email: $email})
				SET d.name = $name,
				    d.repo_id = $repoID,
				    d.commit_count = $commitCount,
				    d.first_commit_date = datetime($firstCommitDate),
				    d.last_commit_date = datetime($lastCommitDate)
				RETURN d.email as email`

			params := map[string]any{
				"email":           dev.Email,
				"name":            dev.Name,
				"repoID":          dev.RepoID,
				"commitCount":     dev.CommitCount,
				"firstCommitDate": dev.FirstCommitDate.Format(time.RFC3339),
				"lastCommitDate":  dev.LastCommitDate.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync developer %s: %v", dev.Email, err)
			continue
		}
		synced++

		// Log progress every 50 developers
		if synced%50 == 0 {
			log.Printf("     → Synced %d/%d developers", synced, len(missingDevelopers))
		}
	}

	log.Printf("     ✓ Synced %d Developers to Neo4j", synced)
	return synced, nil
}

// getAllDevelopers returns all developers from PostgreSQL (aggregated from commits)
// Reference: microservice_arch.md line 832-834 - crisk-ingest creates Developer nodes
func (s *DeveloperSyncer) getAllDevelopers(ctx context.Context, repoID int64) ([]Developer, error) {
	query := `
		SELECT
			author_email,
			author_name,
			repo_id,
			COUNT(*) as commit_count,
			MIN(author_date) as first_commit_date,
			MAX(author_date) as last_commit_date
		FROM github_commits
		WHERE repo_id = $1
		GROUP BY author_email, author_name, repo_id
		ORDER BY commit_count DESC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var developers []Developer
	for rows.Next() {
		var d Developer
		err := rows.Scan(
			&d.Email,
			&d.Name,
			&d.RepoID,
			&d.CommitCount,
			&d.FirstCommitDate,
			&d.LastCommitDate,
		)
		if err != nil {
			return nil, err
		}
		developers = append(developers, d)
	}

	return developers, rows.Err()
}

// getNeo4jDeveloperEmails returns existing developer emails from Neo4j
func (s *DeveloperSyncer) getNeo4jDeveloperEmails(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (d:Developer {repo_id: $repoID}) RETURN d.email as email`
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
	var emails []string
	for _, record := range records {
		email, ok := record.Get("email")
		if !ok {
			continue
		}
		emails = append(emails, email.(string))
	}

	return emails, nil
}
