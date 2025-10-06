package incidents

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// SearchIncidents performs BM25-style full-text search using PostgreSQL tsvector
func (d *Database) SearchIncidents(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	// Convert query to tsquery format (handle multiple words)
	// Replace spaces with & for AND logic: "payment timeout" -> "payment & timeout"
	tsQuery := strings.TrimSpace(query)
	tsQuery = strings.ReplaceAll(tsQuery, " ", " & ")

	// Use ts_rank_cd for BM25-style ranking (considers document length and term frequency)
	sqlQuery := `
		SELECT
		    i.*,
		    ts_rank_cd(i.search_vector, query) AS rank
		FROM incidents i, to_tsquery('english', $1) query
		WHERE i.search_vector @@ query
		ORDER BY rank DESC
		LIMIT $2
	`

	rows, err := d.db.QueryContext(ctx, sqlQuery, tsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search incidents: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	for rows.Next() {
		var incident Incident
		var rank float64

		err := rows.Scan(
			&incident.ID,
			&incident.Title,
			&incident.Description,
			&incident.Severity,
			&incident.OccurredAt,
			&incident.ResolvedAt,
			&incident.RootCause,
			&incident.Impact,
			&incident.SearchVectorPlaceholder, // Generated column, we don't need the value
			&incident.CreatedAt,
			&incident.UpdatedAt,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}

		// Map rank to relevance level
		relevance := "low"
		if rank > 0.5 {
			relevance = "high"
		} else if rank > 0.2 {
			relevance = "medium"
		}

		results = append(results, SearchResult{
			Incident:  incident,
			Rank:      rank,
			Relevance: relevance,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return results, nil
}

// FindSimilarIncidents finds incidents similar to a given incident
func (d *Database) FindSimilarIncidents(ctx context.Context, incidentID uuid.UUID, limit int) ([]SearchResult, error) {
	// Get source incident
	incident, err := d.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("get source incident: %w", err)
	}

	// Build search query from title + description
	query := incident.Title + " " + incident.Description

	// Search for similar incidents
	results, err := d.SearchIncidents(ctx, query, limit+1) // +1 to account for source incident
	if err != nil {
		return nil, err
	}

	// Filter out the source incident
	filtered := make([]SearchResult, 0, len(results))
	for _, result := range results {
		if result.Incident.ID != incidentID {
			filtered = append(filtered, result)
		}
	}

	// Trim to limit
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// SearchByFile finds incidents mentioning a specific file path in their description or root cause
func (d *Database) SearchByFile(ctx context.Context, filePath string) ([]SearchResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Extract just the filename for better matching
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]

	// Search using the file name
	return d.SearchIncidents(ctx, fileName, 50)
}

// SearchByTimeRange searches incidents within a time range with optional text query
func (d *Database) SearchByTimeRange(ctx context.Context, query string, startTime, endTime *sql.NullTime, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	var sqlQuery string
	var args []interface{}

	if query != "" {
		// With text search
		tsQuery := strings.TrimSpace(query)
		tsQuery = strings.ReplaceAll(tsQuery, " ", " & ")

		if startTime != nil && startTime.Valid && endTime != nil && endTime.Valid {
			sqlQuery = `
				SELECT
				    i.*,
				    ts_rank_cd(i.search_vector, q) AS rank
				FROM incidents i, to_tsquery('english', $1) q
				WHERE i.search_vector @@ q
				    AND i.occurred_at >= $2
				    AND i.occurred_at <= $3
				ORDER BY rank DESC, i.occurred_at DESC
				LIMIT $4
			`
			args = []interface{}{tsQuery, startTime.Time, endTime.Time, limit}
		} else if startTime != nil && startTime.Valid {
			sqlQuery = `
				SELECT
				    i.*,
				    ts_rank_cd(i.search_vector, q) AS rank
				FROM incidents i, to_tsquery('english', $1) q
				WHERE i.search_vector @@ q
				    AND i.occurred_at >= $2
				ORDER BY rank DESC, i.occurred_at DESC
				LIMIT $3
			`
			args = []interface{}{tsQuery, startTime.Time, limit}
		} else {
			return d.SearchIncidents(ctx, query, limit)
		}
	} else {
		// No text search, just time range
		if startTime != nil && startTime.Valid && endTime != nil && endTime.Valid {
			sqlQuery = `
				SELECT
				    i.*,
				    0.5 AS rank
				FROM incidents i
				WHERE i.occurred_at >= $1
				    AND i.occurred_at <= $2
				ORDER BY i.occurred_at DESC
				LIMIT $3
			`
			args = []interface{}{startTime.Time, endTime.Time, limit}
		} else if startTime != nil && startTime.Valid {
			sqlQuery = `
				SELECT
				    i.*,
				    0.5 AS rank
				FROM incidents i
				WHERE i.occurred_at >= $1
				ORDER BY i.occurred_at DESC
				LIMIT $2
			`
			args = []interface{}{startTime.Time, limit}
		} else {
			return nil, fmt.Errorf("either query or time range must be provided")
		}
	}

	rows, err := d.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search by time range: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	for rows.Next() {
		var incident Incident
		var rank float64

		err := rows.Scan(
			&incident.ID,
			&incident.Title,
			&incident.Description,
			&incident.Severity,
			&incident.OccurredAt,
			&incident.ResolvedAt,
			&incident.RootCause,
			&incident.Impact,
			&incident.SearchVectorPlaceholder,
			&incident.CreatedAt,
			&incident.UpdatedAt,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}

		relevance := "low"
		if rank > 0.5 {
			relevance = "high"
		} else if rank > 0.2 {
			relevance = "medium"
		}

		results = append(results, SearchResult{
			Incident:  incident,
			Rank:      rank,
			Relevance: relevance,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return results, nil
}
