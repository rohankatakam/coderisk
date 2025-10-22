package risk

import (
	"time"

	"github.com/rohankatakam/coderisk/internal/types"
)

// PhaseResult represents the output of a single phase in the sequential analysis chain
type PhaseResult struct {
	PhaseName   string                 `json:"phase_name"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Confidence  float64                `json:"confidence"`
	NextPhase   string                 `json:"next_phase,omitempty"`
	ShouldSkip  bool                   `json:"should_skip"`
	SkipReason  string                 `json:"skip_reason,omitempty"`
}

// HeuristicFilterResult represents the output of Tier 0 heuristic filtering
type HeuristicFilterResult struct {
	IsTrivial     bool     `json:"is_trivial"`
	Reason        string   `json:"reason"`
	Confidence    float64  `json:"confidence"`
	MatchedRules  []string `json:"matched_rules"`
	LinesChanged  int      `json:"lines_changed"`
	FilesChanged  int      `json:"files_changed"`
	ChangeType    string   `json:"change_type"` // "whitespace", "comment", "documentation", etc.
	DurationMS    int64    `json:"duration_ms"`
}

// Phase1Data represents the collected data from Phase 1 (7 queries)
type Phase1Data struct {
	// Query 1: Blast radius (structural coupling)
	BlastRadius      int      `json:"blast_radius"`
	DependentFiles   []string `json:"dependent_files"`

	// Query 2: Co-change patterns
	CoChangePartners []CoChangePartner `json:"co_change_partners"`

	// Query 3: Ownership and churn
	FileOwner        string    `json:"file_owner"`
	OwnerEmail       string    `json:"owner_email"`
	ChurnScore       float64   `json:"churn_score"`
	LastModified     time.Time `json:"last_modified"`
	LastModifier     string    `json:"last_modifier"`
	CommitCount      int       `json:"commit_count"`

	// Query 4: Test coverage
	TestRatio        float64  `json:"test_ratio"`
	TestFiles        []string `json:"test_files"`
	HasTests         bool     `json:"has_tests"`

	// Query 5: Incident history
	IncidentCount    int      `json:"incident_count"`
	Incidents        []string `json:"incidents"`
	IncidentDensity  float64  `json:"incident_density"`

	// Query 6: Change complexity
	LinesAdded       int      `json:"lines_added"`
	LinesDeleted     int      `json:"lines_deleted"`
	LinesModified    int      `json:"lines_modified"`
	ComplexityScore  float64  `json:"complexity_score"`

	// Query 7: Developer patterns
	DeveloperHistory []DeveloperActivity `json:"developer_history"`
	TeamSize         int                 `json:"team_size"`

	// Metadata
	QueryDurations   map[string]time.Duration `json:"query_durations"`
	CollectionTime   time.Time                `json:"collection_time"`
}

// CoChangePartner represents a file that frequently changes with the target file
type CoChangePartner struct {
	FilePath       string    `json:"file_path"`
	Frequency      float64   `json:"frequency"`
	LastCoChange   time.Time `json:"last_co_change"`
	TotalCoChanges int       `json:"total_co_changes"`
	IsUpdated      bool      `json:"is_updated"` // Whether this file is updated in current change
}

// DeveloperActivity represents a developer's activity pattern
type DeveloperActivity struct {
	Developer    string    `json:"developer"`
	Email        string    `json:"email"`
	CommitCount  int       `json:"commit_count"`
	LinesChanged int       `json:"lines_changed"`
	LastCommit   time.Time `json:"last_commit"`
	IsActiveNow  bool      `json:"is_active_now"`
}

// AgentContext holds shared context passed between agents in the sequential chain
type AgentContext struct {
	// Input
	FilePath      string                 `json:"file_path"`
	RepoID        string                 `json:"repo_id"`
	CommitSHA     string                 `json:"commit_sha"`
	GitDiff       string                 `json:"git_diff"`
	Branch        string                 `json:"branch"`

	// Phase 0: Heuristic filter
	HeuristicResult *HeuristicFilterResult `json:"heuristic_result,omitempty"`

	// Phase 1: Data collection
	Phase1Data      *Phase1Data            `json:"phase1_data,omitempty"`

	// Phase 2-5: Agent results
	AgentResults    map[string]interface{} `json:"agent_results"`

	// Accumulated risk signals
	RiskSignals     []RiskSignal           `json:"risk_signals"`
	OverallRiskScore float64               `json:"overall_risk_score"`

	// Metadata
	StartTime       time.Time              `json:"start_time"`
	PhaseDurations  map[string]time.Duration `json:"phase_durations"`
	CurrentPhase    string                 `json:"current_phase"`
}

// RiskSignal represents a specific risk indicator identified by an agent
type RiskSignal struct {
	AgentName    string                 `json:"agent_name"`
	SignalType   string                 `json:"signal_type"` // "incident", "coordination", "forgotten_update", etc.
	Severity     string                 `json:"severity"`    // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Confidence   float64                `json:"confidence"`
	Description  string                 `json:"description"`
	Evidence     []string               `json:"evidence"`
	Impact       string                 `json:"impact"`
	Mitigation   string                 `json:"mitigation,omitempty"`
	Score        float64                `json:"score"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ChainOrchestratorResult represents the final output of the sequential analysis chain
type ChainOrchestratorResult struct {
	// Final assessment
	RiskLevel        types.RiskLevel `json:"risk_level"`
	RiskScore        float64          `json:"risk_score"`
	Confidence       float64          `json:"confidence"`

	// Detailed results
	Summary          string                 `json:"summary"`
	RiskSignals      []RiskSignal           `json:"risk_signals"`
	Recommendations  []string               `json:"recommendations"`
	CoordinationNeeded []CoordinationInfo   `json:"coordination_needed"`
	ForgottenUpdates []ForgottenUpdate      `json:"forgotten_updates"`

	// Phase results
	PhaseResults     []PhaseResult          `json:"phase_results"`
	HeuristicResult  *HeuristicFilterResult `json:"heuristic_result,omitempty"`
	Phase1Data       *Phase1Data            `json:"phase1_data,omitempty"`

	// Performance
	TotalDuration    time.Duration          `json:"total_duration"`
	CacheHit         bool                   `json:"cache_hit"`
	FastPathTaken    bool                   `json:"fast_path_taken"`

	// Metadata
	Timestamp        time.Time              `json:"timestamp"`
	FilePath         string                 `json:"file_path"`
	CommitSHA        string                 `json:"commit_sha"`
	Branch           string                 `json:"branch"`
}

// CoordinationInfo represents coordination requirements identified by agents
type CoordinationInfo struct {
	TeamMember   string   `json:"team_member"`
	Reason       string   `json:"reason"`
	Files        []string `json:"files"`
	Urgency      string   `json:"urgency"` // "OPTIONAL", "RECOMMENDED", "REQUIRED"
	ContactInfo  string   `json:"contact_info,omitempty"`
}

// ForgottenUpdate represents a potentially forgotten co-change update
type ForgottenUpdate struct {
	FilePath     string  `json:"file_path"`
	Reason       string  `json:"reason"`
	CoChangeFreq float64 `json:"co_change_freq"`
	LastCoChange string  `json:"last_co_change"`
	Confidence   float64 `json:"confidence"`
	Impact       string  `json:"impact"`
}

// AnalysisRequest represents a request to analyze risk for a change
type AnalysisRequest struct {
	RepoID       string            `json:"repo_id"`
	CommitSHA    string            `json:"commit_sha"`
	Branch       string            `json:"branch"`
	FilePaths    []string          `json:"file_paths"`
	GitDiff      string            `json:"git_diff"`
	UseCache     bool              `json:"use_cache"`
	Verbosity    string            `json:"verbosity"` // "quiet", "standard", "explain", "ai"
	Timeout      time.Duration     `json:"timeout"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
