package incidents

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Database handles PostgreSQL operations for incidents
type Database struct {
	db *sqlx.DB
}

// NewDatabase creates a new incident database client
func NewDatabase(db *sqlx.DB) *Database {
	return &Database{db: db}
}

// CreateIncident inserts a new incident
func (d *Database) CreateIncident(ctx context.Context, inc *Incident) error {
	if inc == nil {
		return fmt.Errorf("incident cannot be nil")
	}

	// Validate severity
	if !inc.Severity.Validate() {
		return fmt.Errorf("invalid severity: %s", inc.Severity)
	}

	// Generate UUID if not set
	if inc.ID == uuid.Nil {
		inc.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = now
	}
	if inc.UpdatedAt.IsZero() {
		inc.UpdatedAt = now
	}

	query := `
		INSERT INTO incidents (id, title, description, severity, occurred_at, resolved_at, root_cause, impact, created_at, updated_at)
		VALUES (:id, :title, :description, :severity, :occurred_at, :resolved_at, :root_cause, :impact, :created_at, :updated_at)
	`

	_, err := d.db.NamedExecContext(ctx, query, inc)
	if err != nil {
		return fmt.Errorf("insert incident: %w", err)
	}

	return nil
}

// GetIncident retrieves incident by ID with linked files
func (d *Database) GetIncident(ctx context.Context, id uuid.UUID) (*Incident, error) {
	var incident Incident

	// Query incident
	query := `SELECT * FROM incidents WHERE id = $1`
	err := d.db.GetContext(ctx, &incident, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("incident not found: %s", id)
		}
		return nil, fmt.Errorf("get incident: %w", err)
	}

	// Query linked files
	filesQuery := `SELECT * FROM incident_files WHERE incident_id = $1`
	var files []IncidentFile
	err = d.db.SelectContext(ctx, &files, filesQuery, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get incident files: %w", err)
	}

	incident.LinkedFiles = files

	return &incident, nil
}

// UpdateIncident updates an existing incident
func (d *Database) UpdateIncident(ctx context.Context, inc *Incident) error {
	if inc == nil {
		return fmt.Errorf("incident cannot be nil")
	}

	if !inc.Severity.Validate() {
		return fmt.Errorf("invalid severity: %s", inc.Severity)
	}

	// Update timestamp
	inc.UpdatedAt = time.Now()

	query := `
		UPDATE incidents
		SET title = :title,
		    description = :description,
		    severity = :severity,
		    occurred_at = :occurred_at,
		    resolved_at = :resolved_at,
		    root_cause = :root_cause,
		    impact = :impact,
		    updated_at = :updated_at
		WHERE id = :id
	`

	result, err := d.db.NamedExecContext(ctx, query, inc)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("incident not found: %s", inc.ID)
	}

	return nil
}

// DeleteIncident deletes an incident and its links (CASCADE)
func (d *Database) DeleteIncident(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM incidents WHERE id = $1`

	result, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete incident: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("incident not found: %s", id)
	}

	return nil
}

// LinkIncidentToFile creates manual link between incident and file
func (d *Database) LinkIncidentToFile(ctx context.Context, link *IncidentFile) error {
	if link == nil {
		return fmt.Errorf("incident file link cannot be nil")
	}

	// Validate incident exists
	var exists bool
	err := d.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM incidents WHERE id = $1)", link.IncidentID)
	if err != nil {
		return fmt.Errorf("check incident exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("incident not found: %s", link.IncidentID)
	}

	// Validate confidence
	if link.Confidence < 0.0 || link.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got: %f", link.Confidence)
	}

	query := `
		INSERT INTO incident_files (incident_id, file_path, line_number, blamed_function, confidence)
		VALUES (:incident_id, :file_path, :line_number, :blamed_function, :confidence)
		ON CONFLICT (incident_id, file_path) DO UPDATE SET
		    line_number = EXCLUDED.line_number,
		    blamed_function = EXCLUDED.blamed_function,
		    confidence = EXCLUDED.confidence
	`

	_, err = d.db.NamedExecContext(ctx, query, link)
	if err != nil {
		return fmt.Errorf("link incident to file: %w", err)
	}

	return nil
}

// UnlinkIncidentFromFile removes link between incident and file
func (d *Database) UnlinkIncidentFromFile(ctx context.Context, incidentID uuid.UUID, filePath string) error {
	query := `DELETE FROM incident_files WHERE incident_id = $1 AND file_path = $2`

	result, err := d.db.ExecContext(ctx, query, incidentID, filePath)
	if err != nil {
		return fmt.Errorf("unlink incident from file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("link not found: incident %s, file %s", incidentID, filePath)
	}

	return nil
}

// GetIncidentsByFile returns all incidents linked to a file
func (d *Database) GetIncidentsByFile(ctx context.Context, filePath string) ([]Incident, error) {
	query := `
		SELECT i.*
		FROM incidents i
		JOIN incident_files if ON i.id = if.incident_id
		WHERE if.file_path = $1
		ORDER BY i.occurred_at DESC
	`

	var incidents []Incident
	err := d.db.SelectContext(ctx, &incidents, query, filePath)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get incidents by file: %w", err)
	}

	return incidents, nil
}

// GetIncidentStats calculates aggregated stats for a file (PUBLIC - Session C uses this)
func (d *Database) GetIncidentStats(ctx context.Context, filePath string) (*IncidentStats, error) {
	stats := &IncidentStats{
		FilePath: filePath,
	}

	// Calculate time windows
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	// Count total incidents and by severity (compatible with both PostgreSQL and SQLite)
	// Use COALESCE to handle NULL from SUM when COUNT > 0
	query := `
		SELECT
		    COUNT(*) as total,
		    COALESCE(SUM(CASE WHEN i.occurred_at >= $2 THEN 1 ELSE 0 END), 0) as last_30_days,
		    COALESCE(SUM(CASE WHEN i.occurred_at >= $3 THEN 1 ELSE 0 END), 0) as last_90_days,
		    COALESCE(SUM(CASE WHEN i.severity = 'critical' THEN 1 ELSE 0 END), 0) as critical_count,
		    COALESCE(SUM(CASE WHEN i.severity = 'high' THEN 1 ELSE 0 END), 0) as high_count,
		    MAX(i.occurred_at) as last_incident
		FROM incidents i
		JOIN incident_files if ON i.id = if.incident_id
		WHERE if.file_path = $1
	`

	row := d.db.QueryRowContext(ctx, query, filePath, thirtyDaysAgo, ninetyDaysAgo)

	var lastIncident sql.NullTime
	err := row.Scan(
		&stats.TotalIncidents,
		&stats.Last30Days,
		&stats.Last90Days,
		&stats.CriticalCount,
		&stats.HighCount,
		&lastIncident,
	)
	if err != nil {
		return nil, fmt.Errorf("get incident counts: %w", err)
	}

	if lastIncident.Valid {
		stats.LastIncident = &lastIncident.Time
	}

	// Calculate average resolution time (compatible with both PostgreSQL and SQLite)
	resolutionQuery := `
		SELECT AVG(CAST((julianday(i.resolved_at) - julianday(i.occurred_at)) * 86400 AS INTEGER))
		FROM incidents i
		JOIN incident_files if ON i.id = if.incident_id
		WHERE if.file_path = $1 AND i.resolved_at IS NOT NULL
	`

	// Try SQLite-style first, fall back to PostgreSQL-style
	var avgSeconds sql.NullFloat64
	err = d.db.GetContext(ctx, &avgSeconds, resolutionQuery, filePath)
	if err != nil {
		// Try PostgreSQL-style
		resolutionQuery = `
			SELECT AVG(EXTRACT(EPOCH FROM (i.resolved_at - i.occurred_at)))
			FROM incidents i
			JOIN incident_files if ON i.id = if.incident_id
			WHERE if.file_path = $1 AND i.resolved_at IS NOT NULL
		`
		err = d.db.GetContext(ctx, &avgSeconds, resolutionQuery, filePath)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("get average resolution time: %w", err)
		}
	}

	if avgSeconds.Valid {
		stats.AvgResolution = time.Duration(avgSeconds.Float64) * time.Second
	}

	return stats, nil
}

// ListIncidents returns all incidents with optional filters
func (d *Database) ListIncidents(ctx context.Context, severity Severity, limit int) ([]Incident, error) {
	var incidents []Incident
	var query string

	if severity != "" {
		if !severity.Validate() {
			return nil, fmt.Errorf("invalid severity: %s", severity)
		}
		query = `SELECT * FROM incidents WHERE severity = $1 ORDER BY occurred_at DESC LIMIT $2`
		err := d.db.SelectContext(ctx, &incidents, query, severity, limit)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("list incidents by severity: %w", err)
		}
	} else {
		query = `SELECT * FROM incidents ORDER BY occurred_at DESC LIMIT $1`
		err := d.db.SelectContext(ctx, &incidents, query, limit)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("list incidents: %w", err)
		}
	}

	return incidents, nil
}
