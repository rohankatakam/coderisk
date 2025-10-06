package incidents

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create tables (SQLite compatible schema)
	schema := `
		CREATE TABLE incidents (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			severity TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
			occurred_at DATETIME NOT NULL,
			resolved_at DATETIME,
			root_cause TEXT,
			impact TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE incident_files (
			incident_id TEXT NOT NULL,
			file_path TEXT NOT NULL,
			line_number INTEGER DEFAULT 0,
			blamed_function TEXT DEFAULT '',
			confidence REAL DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
			PRIMARY KEY (incident_id, file_path),
			FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
		);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func TestCreateIncident(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	tests := []struct {
		name      string
		incident  *Incident
		wantError bool
	}{
		{
			name: "valid incident",
			incident: &Incident{
				Title:       "Payment processor timeout",
				Description: "Users unable to complete checkout due to 30s timeout",
				Severity:    SeverityCritical,
				OccurredAt:  time.Now(),
				RootCause:   "payment_processor.py missing connection pooling",
			},
			wantError: false,
		},
		{
			name: "with resolved date",
			incident: &Incident{
				Title:       "Database deadlock",
				Description: "Deadlock in user table",
				Severity:    SeverityHigh,
				OccurredAt:  time.Now().Add(-24 * time.Hour),
				ResolvedAt:  timePtr(time.Now()),
			},
			wantError: false,
		},
		{
			name:      "nil incident",
			incident:  nil,
			wantError: true,
		},
		{
			name: "invalid severity",
			incident: &Incident{
				Title:       "Test",
				Description: "Test",
				Severity:    Severity("invalid"),
				OccurredAt:  time.Now(),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := incDB.CreateIncident(ctx, tt.incident)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.incident.ID)
				assert.NotZero(t, tt.incident.CreatedAt)
				assert.NotZero(t, tt.incident.UpdatedAt)
			}
		})
	}
}

func TestGetIncident(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create a test incident
	incident := &Incident{
		Title:       "Test incident",
		Description: "Test description",
		Severity:    SeverityMedium,
		OccurredAt:  time.Now(),
		RootCause:   "Test root cause",
	}
	err := incDB.CreateIncident(ctx, incident)
	require.NoError(t, err)

	// Get the incident
	retrieved, err := incDB.GetIncident(ctx, incident.ID)
	assert.NoError(t, err)
	assert.Equal(t, incident.ID, retrieved.ID)
	assert.Equal(t, incident.Title, retrieved.Title)
	assert.Equal(t, incident.Severity, retrieved.Severity)

	// Test non-existent incident
	_, err = incDB.GetIncident(ctx, uuid.New())
	assert.Error(t, err)
}

func TestUpdateIncident(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create incident
	incident := &Incident{
		Title:       "Original title",
		Description: "Original description",
		Severity:    SeverityLow,
		OccurredAt:  time.Now(),
	}
	err := incDB.CreateIncident(ctx, incident)
	require.NoError(t, err)

	// Update incident
	incident.Title = "Updated title"
	incident.Severity = SeverityHigh
	now := time.Now()
	incident.ResolvedAt = &now

	err = incDB.UpdateIncident(ctx, incident)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := incDB.GetIncident(ctx, incident.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated title", retrieved.Title)
	assert.Equal(t, SeverityHigh, retrieved.Severity)
	assert.NotNil(t, retrieved.ResolvedAt)
}

func TestDeleteIncident(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create incident
	incident := &Incident{
		Title:       "To be deleted",
		Description: "This will be deleted",
		Severity:    SeverityLow,
		OccurredAt:  time.Now(),
	}
	err := incDB.CreateIncident(ctx, incident)
	require.NoError(t, err)

	// Delete incident
	err = incDB.DeleteIncident(ctx, incident.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = incDB.GetIncident(ctx, incident.ID)
	assert.Error(t, err)

	// Test deleting non-existent incident
	err = incDB.DeleteIncident(ctx, uuid.New())
	assert.Error(t, err)
}

func TestLinkIncidentToFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create incident
	incident := &Incident{
		Title:       "Test incident",
		Description: "Test",
		Severity:    SeverityMedium,
		OccurredAt:  time.Now(),
	}
	err := incDB.CreateIncident(ctx, incident)
	require.NoError(t, err)

	tests := []struct {
		name      string
		link      *IncidentFile
		wantError bool
	}{
		{
			name: "valid link",
			link: &IncidentFile{
				IncidentID:     incident.ID,
				FilePath:       "src/payment_processor.py",
				LineNumber:     142,
				BlamedFunction: "process_payment",
				Confidence:     1.0,
			},
			wantError: false,
		},
		{
			name: "entire file link",
			link: &IncidentFile{
				IncidentID: incident.ID,
				FilePath:   "src/database.py",
				Confidence: 0.8,
			},
			wantError: false,
		},
		{
			name:      "nil link",
			link:      nil,
			wantError: true,
		},
		{
			name: "invalid confidence",
			link: &IncidentFile{
				IncidentID: incident.ID,
				FilePath:   "test.py",
				Confidence: 1.5,
			},
			wantError: true,
		},
		{
			name: "non-existent incident",
			link: &IncidentFile{
				IncidentID: uuid.New(),
				FilePath:   "test.py",
				Confidence: 1.0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := incDB.LinkIncidentToFile(ctx, tt.link)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetIncidentsByFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	filePath := "src/payment_processor.py"

	// Create incidents
	incident1 := &Incident{
		Title:       "Timeout issue",
		Description: "Payment timeout",
		Severity:    SeverityCritical,
		OccurredAt:  time.Now(),
	}
	err := incDB.CreateIncident(ctx, incident1)
	require.NoError(t, err)

	incident2 := &Incident{
		Title:       "Memory leak",
		Description: "Memory leak in payment processing",
		Severity:    SeverityHigh,
		OccurredAt:  time.Now().Add(-24 * time.Hour),
	}
	err = incDB.CreateIncident(ctx, incident2)
	require.NoError(t, err)

	// Link incidents to file
	err = incDB.LinkIncidentToFile(ctx, &IncidentFile{
		IncidentID: incident1.ID,
		FilePath:   filePath,
		Confidence: 1.0,
	})
	require.NoError(t, err)

	err = incDB.LinkIncidentToFile(ctx, &IncidentFile{
		IncidentID: incident2.ID,
		FilePath:   filePath,
		Confidence: 1.0,
	})
	require.NoError(t, err)

	// Get incidents by file
	incidents, err := incDB.GetIncidentsByFile(ctx, filePath)
	assert.NoError(t, err)
	assert.Len(t, incidents, 2)

	// Should be ordered by occurred_at DESC
	assert.Equal(t, incident1.ID, incidents[0].ID)
	assert.Equal(t, incident2.ID, incidents[1].ID)

	// Test file with no incidents
	incidents, err = incDB.GetIncidentsByFile(ctx, "non_existent.py")
	assert.NoError(t, err)
	assert.Len(t, incidents, 0)
}

func TestGetIncidentStats(t *testing.T) {
	t.Skip("SQLite UUID handling differs from PostgreSQL - run integration test instead")

	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	filePath := "src/payment_processor.py"

	// Test file with no incidents
	stats, err := incDB.GetIncidentStats(ctx, filePath)
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.TotalIncidents)
}

func TestListIncidents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create incidents with different severities
	severities := []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
	for i, sev := range severities {
		incident := &Incident{
			Title:       "Incident " + string(sev),
			Description: "Test",
			Severity:    sev,
			OccurredAt:  time.Now().Add(-time.Duration(i) * time.Hour),
		}
		err := incDB.CreateIncident(ctx, incident)
		require.NoError(t, err)
	}

	// List all incidents
	incidents, err := incDB.ListIncidents(ctx, "", 10)
	assert.NoError(t, err)
	assert.Len(t, incidents, 4)

	// List by severity
	incidents, err = incDB.ListIncidents(ctx, SeverityCritical, 10)
	assert.NoError(t, err)
	assert.Len(t, incidents, 1)
	assert.Equal(t, SeverityCritical, incidents[0].Severity)

	// Test limit
	incidents, err = incDB.ListIncidents(ctx, "", 2)
	assert.NoError(t, err)
	assert.Len(t, incidents, 2)
}

func TestUnlinkIncidentFromFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	incDB := NewDatabase(db)
	ctx := context.Background()

	// Create incident and link
	incident := &Incident{
		Title:       "Test",
		Description: "Test",
		Severity:    SeverityMedium,
		OccurredAt:  time.Now(),
	}
	err := incDB.CreateIncident(ctx, incident)
	require.NoError(t, err)

	filePath := "test.py"
	err = incDB.LinkIncidentToFile(ctx, &IncidentFile{
		IncidentID: incident.ID,
		FilePath:   filePath,
		Confidence: 1.0,
	})
	require.NoError(t, err)

	// Unlink
	err = incDB.UnlinkIncidentFromFile(ctx, incident.ID, filePath)
	assert.NoError(t, err)

	// Verify unlinked
	incidents, err := incDB.GetIncidentsByFile(ctx, filePath)
	assert.NoError(t, err)
	assert.Len(t, incidents, 0)

	// Test unlinking non-existent link
	err = incDB.UnlinkIncidentFromFile(ctx, incident.ID, "non_existent.py")
	assert.Error(t, err)
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}
