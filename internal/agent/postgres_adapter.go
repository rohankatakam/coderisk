package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
)

// PostgresAdapter adapts database.StagingClient to PostgresQueryExecutor interface
type PostgresAdapter struct {
	client *database.StagingClient
}

// NewPostgresAdapter creates a new Postgres adapter
func NewPostgresAdapter(client *database.StagingClient) *PostgresAdapter {
	return &PostgresAdapter{client: client}
}

// GitHubFile represents a file change from GitHub API
type GitHubFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch"`
}

// CommitData holds commit metadata and file changes
type CommitData struct {
	SHA        string
	Message    string
	AuthorName string
	AuthorDate time.Time
	Additions  int
	Deletions  int
	Files      []GitHubFile
}

// GetCommitPatch retrieves the patch (diff) for a specific commit from PostgreSQL
func (a *PostgresAdapter) GetCommitPatch(ctx context.Context, commitSHA string) (string, error) {
	query := `
		SELECT
			sha,
			message,
			author_name,
			author_date,
			additions,
			deletions,
			raw_data->'files' as files
		FROM github_commits
		WHERE sha = $1
		LIMIT 1
	`

	var (
		sha        string
		message    string
		authorName string
		authorDate time.Time
		additions  int
		deletions  int
		filesJSON  []byte
	)

	err := a.client.QueryRow(ctx, query, commitSHA).Scan(
		&sha,
		&message,
		&authorName,
		&authorDate,
		&additions,
		&deletions,
		&filesJSON,
	)
	if err != nil {
		return "", fmt.Errorf("commit %s not found: %w", commitSHA, err)
	}

	// Parse files array from GitHub API format
	var files []GitHubFile
	if err := json.Unmarshal(filesJSON, &files); err != nil {
		return "", fmt.Errorf("failed to parse file changes: %w", err)
	}

	commitData := CommitData{
		SHA:        sha,
		Message:    message,
		AuthorName: authorName,
		AuthorDate: authorDate,
		Additions:  additions,
		Deletions:  deletions,
		Files:      files,
	}

	return formatPatchOutput(commitData), nil
}

// formatPatchOutput formats commit data as readable text
func formatPatchOutput(data CommitData) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("Commit: %s\n", data.SHA))
	b.WriteString(fmt.Sprintf("Author: %s\n", data.AuthorName))
	b.WriteString(fmt.Sprintf("Date: %s\n", data.AuthorDate.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("\n%s\n\n", data.Message))

	// Stats summary
	b.WriteString(fmt.Sprintf("Changed %d files: +%d additions, -%d deletions\n\n",
		len(data.Files), data.Additions, data.Deletions))

	// File-by-file changes
	for _, file := range data.Files {
		b.WriteString(fmt.Sprintf("--- %s (%s)\n", file.Filename, file.Status))
		b.WriteString(fmt.Sprintf("+%d -%d changes\n", file.Additions, file.Deletions))

		if file.Patch != "" {
			b.WriteString("\n")
			b.WriteString(file.Patch)
			b.WriteString("\n\n")
		} else {
			b.WriteString("(Binary file or no diff available)\n\n")
		}
	}

	return b.String()
}
