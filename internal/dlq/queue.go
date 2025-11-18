package dlq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Entry represents a dead letter queue entry
type Entry struct {
	ID           int64
	RepoID       int64
	CommitSHA    string
	ErrorMessage string
	ErrorStack   string
	RetryCount   int
	LastRetryAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]interface{}
}

// Queue manages failed commit processing
type Queue struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewQueue creates a new DLQ manager
func NewQueue(db *sql.DB) *Queue {
	return &Queue{
		db:     db,
		logger: slog.Default().With("component", "dlq"),
	}
}

// Enqueue adds a failed commit to the DLQ
// If the commit already exists, increments retry_count
func (q *Queue) Enqueue(ctx context.Context, repoID int64, commitSHA string, err error, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Extract error message and stack
	errorMsg := err.Error()
	errorStack := fmt.Sprintf("%+v", err) // Try to get stack trace if available

	_, dbErr := q.db.ExecContext(ctx, `
		INSERT INTO dead_letter_queue (repo_id, commit_sha, error_message, error_stack, retry_count, metadata)
		VALUES ($1, $2, $3, $4, 0, $5)
		ON CONFLICT (repo_id, commit_sha) DO UPDATE
		SET retry_count = dead_letter_queue.retry_count + 1,
		    error_message = $3,
		    error_stack = $4,
		    updated_at = NOW(),
		    last_retry_at = NOW(),
		    metadata = $5
	`, repoID, commitSHA, errorMsg, errorStack, metadataJSON)

	if dbErr != nil {
		return fmt.Errorf("failed to enqueue commit to DLQ: %w", dbErr)
	}

	q.logger.Warn("commit enqueued to DLQ",
		"repo_id", repoID,
		"commit_sha", commitSHA[:8],
		"error", errorMsg,
	)

	return nil
}

// GetPendingRetries returns commits ready for retry (retry_count < max)
func (q *Queue) GetPendingRetries(ctx context.Context, repoID int64, maxRetries int) ([]Entry, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, repo_id, commit_sha, error_message, error_stack, retry_count, last_retry_at, created_at, updated_at, metadata
		FROM dead_letter_queue
		WHERE repo_id = $1 AND retry_count < $2
		ORDER BY created_at ASC
	`, repoID, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to query DLQ: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var metadataJSON []byte
		var lastRetryAt sql.NullTime

		err := rows.Scan(&e.ID, &e.RepoID, &e.CommitSHA, &e.ErrorMessage, &e.ErrorStack,
			&e.RetryCount, &lastRetryAt, &e.CreatedAt, &e.UpdatedAt, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DLQ entry: %w", err)
		}

		if lastRetryAt.Valid {
			e.LastRetryAt = &lastRetryAt.Time
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				q.logger.Warn("failed to unmarshal metadata", "entry_id", e.ID, "error", err)
				e.Metadata = make(map[string]interface{})
			}
		} else {
			e.Metadata = make(map[string]interface{})
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// MarkResolved removes a commit from the DLQ after successful retry
func (q *Queue) MarkResolved(ctx context.Context, repoID int64, commitSHA string) error {
	result, err := q.db.ExecContext(ctx, `
		DELETE FROM dead_letter_queue
		WHERE repo_id = $1 AND commit_sha = $2
	`, repoID, commitSHA)
	if err != nil {
		return fmt.Errorf("failed to delete DLQ entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		q.logger.Info("commit resolved and removed from DLQ",
			"repo_id", repoID,
			"commit_sha", commitSHA[:8],
		)
	}

	return nil
}

// GetStats returns DLQ statistics for a repository
func (q *Queue) GetStats(ctx context.Context, repoID int64) (*Stats, error) {
	var stats Stats

	// Count total entries
	err := q.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE retry_count >= 5) as exhausted,
			COUNT(*) FILTER (WHERE retry_count < 5) as retryable
		FROM dead_letter_queue
		WHERE repo_id = $1
	`, repoID).Scan(&stats.TotalEntries, &stats.ExhaustedRetries, &stats.RetryableEntries)

	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ stats: %w", err)
	}

	stats.RepoID = repoID

	return &stats, nil
}

// Stats contains DLQ statistics
type Stats struct {
	RepoID            int64
	TotalEntries      int
	RetryableEntries  int
	ExhaustedRetries  int
}

// GetRecentFailures returns the N most recent failures for review
func (q *Queue) GetRecentFailures(ctx context.Context, repoID int64, limit int) ([]Entry, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, repo_id, commit_sha, error_message, error_stack, retry_count, last_retry_at, created_at, updated_at, metadata
		FROM dead_letter_queue
		WHERE repo_id = $1
		ORDER BY updated_at DESC
		LIMIT $2
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent failures: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var metadataJSON []byte
		var lastRetryAt sql.NullTime

		err := rows.Scan(&e.ID, &e.RepoID, &e.CommitSHA, &e.ErrorMessage, &e.ErrorStack,
			&e.RetryCount, &lastRetryAt, &e.CreatedAt, &e.UpdatedAt, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DLQ entry: %w", err)
		}

		if lastRetryAt.Valid {
			e.LastRetryAt = &lastRetryAt.Time
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// PurgeOld removes DLQ entries older than the specified duration
func (q *Queue) PurgeOld(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)

	result, err := q.db.ExecContext(ctx, `
		DELETE FROM dead_letter_queue
		WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old DLQ entries: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		q.logger.Info("purged old DLQ entries",
			"count", rows,
			"older_than", olderThan,
		)
	}

	return int(rows), nil
}
