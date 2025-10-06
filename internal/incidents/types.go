package incidents

import (
	"time"

	"github.com/google/uuid"
)

// Incident represents a production incident or bug
type Incident struct {
	ID                       uuid.UUID  `db:"id"`
	Title                    string     `db:"title"`
	Description              string     `db:"description"`
	Severity                 Severity   `db:"severity"`
	OccurredAt               time.Time  `db:"occurred_at"`
	ResolvedAt               *time.Time `db:"resolved_at"`
	RootCause                string     `db:"root_cause"`
	Impact                   string     `db:"impact"`
	SearchVectorPlaceholder  string     `db:"search_vector"` // Generated column, placeholder for scanning
	CreatedAt                time.Time  `db:"created_at"`
	UpdatedAt                time.Time  `db:"updated_at"`

	// Linked files (populated on query)
	LinkedFiles []IncidentFile `db:"-"`
}

// Severity levels
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Validate checks if severity is valid
func (s Severity) Validate() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	default:
		return false
	}
}

// IncidentFile represents a file blamed for an incident
type IncidentFile struct {
	IncidentID     uuid.UUID `db:"incident_id"`
	FilePath       string    `db:"file_path"`
	LineNumber     int       `db:"line_number"`      // 0 if entire file
	BlamedFunction string    `db:"blamed_function"`  // empty if entire file
	Confidence     float64   `db:"confidence"`       // 0.0-1.0 (1.0 = manual link, <1.0 = auto-inferred)
}

// SearchResult represents BM25 similarity search result
type SearchResult struct {
	Incident  Incident
	Rank      float64 // BM25 score (higher = more relevant)
	Relevance string  // "high" (>0.5), "medium" (0.2-0.5), "low" (<0.2)
}

// IncidentStats aggregates incident data for risk calculation
type IncidentStats struct {
	FilePath       string
	TotalIncidents int
	Last30Days     int
	Last90Days     int
	CriticalCount  int
	HighCount      int
	LastIncident   *time.Time
	AvgResolution  time.Duration // Average time to resolve
}
