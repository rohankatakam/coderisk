package diffanalyzer

import "time"

// RiskEvidenceJSON is the main output structure matching the API spec
type RiskEvidenceJSON struct {
	RiskSummary string      `json:"risk_summary"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Blocks      []BlockRisk `json:"blocks"`
}

// BlockRisk represents risk assessment for a single code block
type BlockRisk struct {
	Name       string         `json:"name"`
	File       string         `json:"file"`
	ChangeType string         `json:"change_type"` // "MODIFIED", "CREATED", "DELETED"
	MatchType  string         `json:"match_type"`  // "exact", "fuzzy_signature", "historical_name", "new_function"
	Risks      RiskDimensions `json:"risks"`
}

// RiskDimensions contains all three risk dimensions
type RiskDimensions struct {
	Temporal      *TemporalRisk  `json:"temporal,omitempty"`
	Ownership     *OwnershipRisk `json:"ownership,omitempty"`
	Coupling      *CouplingRisk  `json:"coupling,omitempty"`
	ChangeHistory *ChangeHistory `json:"change_history,omitempty"`
}

// TemporalRisk represents incident history for a code block
type TemporalRisk struct {
	IncidentCount    int           `json:"incident_count"`
	LastIncidentDate *time.Time    `json:"last_incident_date,omitempty"`
	LinkedIssues     []LinkedIssue `json:"linked_issues"`
	Summary          string        `json:"summary"`
	AffectedVersions []string      `json:"affected_versions"` // Historical function names affected
}

// LinkedIssue represents an issue linked to a code block
type LinkedIssue struct {
	Number          int       `json:"number"`
	Title           string    `json:"title"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	VersionAffected string    `json:"version_affected"` // Which historical name was affected
}

// OwnershipRisk represents ownership and staleness metrics
type OwnershipRisk struct {
	Status            string                  `json:"status"` // "STALE", "ACTIVE", "BUS_FACTOR"
	OriginalAuthor    string                  `json:"original_author"`
	LastModifier      string                  `json:"last_modifier"`
	DaysSinceModified int                     `json:"days_since_modified"` // DYNAMIC
	BusFactorWarning  bool                    `json:"bus_factor_warning"`
	TopContributors   []DeveloperContribution `json:"top_contributors"`
}

// DeveloperContribution represents a developer's contribution to a code block
type DeveloperContribution struct {
	Email             string    `json:"email"`
	Name              string    `json:"name"`
	ModificationCount int       `json:"modification_count"`
	LastEdit          time.Time `json:"last_edit"`
	VersionsTouched   []string  `json:"versions_touched"` // Historical names touched
}

// CouplingRisk represents co-change coupling with other blocks
type CouplingRisk struct {
	Score         float64        `json:"score"` // 0-100
	CoupledBlocks []CoupledBlock `json:"coupled_blocks"`
}

// CoupledBlock represents a block that co-changes with the target
type CoupledBlock struct {
	Name                  string  `json:"name"`
	File                  string  `json:"file"`
	CoChangeCount         int     `json:"co_change_count"`
	CouplingRate          float64 `json:"coupling_rate"`
	IncidentCount         int     `json:"incident_count"`
	DynamicCouplingScore  float64 `json:"dynamic_coupling_score"`
}

// ChangeHistory represents recent modifications to a code block
type ChangeHistory struct {
	RecentChanges []BlockChange `json:"recent_changes"`
}

// BlockChange represents a single modification event
type BlockChange struct {
	SHA          string    `json:"sha"`
	Message      string    `json:"message"`
	CommittedAt  time.Time `json:"committed_at"`
	Author       string    `json:"author"`
	ChangeType   string    `json:"change_type"`
	LinesAdded   int       `json:"lines_added"`
	LinesDeleted int       `json:"lines_deleted"`
	Summary      string    `json:"summary"`
	VersionName  string    `json:"version_name"` // Which historical name was modified
}

// BlockMatch represents a match between a diff block and a graph node
type BlockMatch struct {
	DiffBlock     DiffBlockRef
	GraphBlock    *CodeBlockMatch
	CanonicalPath string
}

// DiffBlockRef represents a code block extracted from a diff
type DiffBlockRef struct {
	FilePath  string
	BlockName string
	BlockType string
	Behavior  string // "CREATE_BLOCK", "MODIFY_BLOCK", "DELETE_BLOCK"
	Signature string
}

// CodeBlockMatch represents a matched code block from the graph
type CodeBlockMatch struct {
	ID                   int64
	BlockName            string
	HistoricalBlockNames []string
	Signature            string
	MatchType            string // "exact", "fuzzy_signature", "historical_name", "new_function"
	Confidence           string // "high", "medium", "low"
}

// BlockRiskData aggregates all risk data for a single block
type BlockRiskData struct {
	Status         string
	Confidence     string
	Temporal       *TemporalRisk
	Ownership      *OwnershipRisk
	Coupling       *CouplingRisk
	ChangeHistory  *ChangeHistory
}
