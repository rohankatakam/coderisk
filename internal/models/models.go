package models

import (
	"time"
)

// Repository represents a GitHub repository
type Repository struct {
	ID            string    `json:"id" db:"id"`
	Owner         string    `json:"owner" db:"owner"`
	Name          string    `json:"name" db:"name"`
	FullName      string    `json:"full_name" db:"full_name"`
	URL           string    `json:"url" db:"url"`
	DefaultBranch string    `json:"default_branch" db:"default_branch"`
	Language      string    `json:"language" db:"language"`
	Size          int64     `json:"size" db:"size"`
	StarCount     int       `json:"star_count" db:"star_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	LastIngested  time.Time `json:"last_ingested" db:"last_ingested"`
}

// Commit represents a git commit
type Commit struct {
	SHA         string    `json:"sha" db:"sha"`
	RepoID      string    `json:"repo_id" db:"repo_id"`
	Author      string    `json:"author" db:"author"`
	AuthorEmail string    `json:"author_email" db:"author_email"`
	Message     string    `json:"message" db:"message"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	ParentSHAs  []string  `json:"parent_shas"`
	Files       []*File   `json:"files"`
}

// File represents a file in the repository
type File struct {
	ID           string    `json:"id" db:"id"`
	RepoID       string    `json:"repo_id" db:"repo_id"`
	Path         string    `json:"path" db:"path"`
	Content      string    `json:"content,omitempty" db:"content"`
	Size         int64     `json:"size" db:"size"`
	Language     string    `json:"language" db:"language"`
	SHA          string    `json:"sha" db:"sha"`
	LastModified time.Time `json:"last_modified" db:"last_modified"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID          int        `json:"id" db:"id"`
	RepoID      string     `json:"repo_id" db:"repo_id"`
	Number      int        `json:"number" db:"number"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	State       string     `json:"state" db:"state"`
	Author      string     `json:"author" db:"author"`
	BaseBranch  string     `json:"base_branch" db:"base_branch"`
	HeadBranch  string     `json:"head_branch" db:"head_branch"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	MergedAt    *time.Time `json:"merged_at" db:"merged_at"`
	ClosedAt    *time.Time `json:"closed_at" db:"closed_at"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID         int        `json:"id" db:"id"`
	RepoID     string     `json:"repo_id" db:"repo_id"`
	Number     int        `json:"number" db:"number"`
	Title      string     `json:"title" db:"title"`
	Body       string     `json:"body" db:"body"`
	State      string     `json:"state" db:"state"`
	Author     string     `json:"author" db:"author"`
	Labels     []string   `json:"labels"`
	IsIncident bool       `json:"is_incident" db:"is_incident"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ClosedAt   *time.Time `json:"closed_at" db:"closed_at"`
}

// RiskLevel represents the risk severity
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// RiskAssessment represents a risk analysis result
type RiskAssessment struct {
	ID           string       `json:"id" db:"id"`
	RepoID       string       `json:"repo_id" db:"repo_id"`
	CommitSHA    string       `json:"commit_sha" db:"commit_sha"`
	Level        RiskLevel    `json:"level" db:"level"`
	Score        float64      `json:"score" db:"score"`
	Factors      []RiskFactor `json:"factors"`
	Suggestions  []string     `json:"suggestions"`
	BlastRadius  float64      `json:"blast_radius" db:"blast_radius"`
	TestCoverage float64      `json:"test_coverage" db:"test_coverage"`
	Timestamp    time.Time    `json:"timestamp" db:"timestamp"`
	CacheVersion string       `json:"cache_version" db:"cache_version"`
}

// RiskFactor represents a contributing factor to risk
type RiskFactor struct {
	Signal   string  `json:"signal"`
	Impact   string  `json:"impact"`
	Score    float64 `json:"score"`
	Detail   string  `json:"detail"`
	Evidence string  `json:"evidence,omitempty"`
}

// RiskSketch represents pre-computed risk signals
type RiskSketch struct {
	FileID            string             `json:"file_id" db:"file_id"`
	RepoID            string             `json:"repo_id" db:"repo_id"`
	PPRVector         []float64          `json:"ppr_vector"`
	CentralityScore   float64            `json:"centrality_score" db:"centrality_score"`
	CoChangeFrequency map[string]float64 `json:"co_change_frequency"`
	OwnershipScore    float64            `json:"ownership_score" db:"ownership_score"`
	TestCoverage      float64            `json:"test_coverage" db:"test_coverage"`
	IncidentHistory   []string           `json:"incident_history"`
	LastUpdated       time.Time          `json:"last_updated" db:"last_updated"`
}

// CacheMetadata represents cache version information
type CacheMetadata struct {
	Version     string    `json:"version" db:"version"`
	RepoID      string    `json:"repo_id" db:"repo_id"`
	LastCommit  string    `json:"last_commit" db:"last_commit"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
	Size        int64     `json:"size" db:"size"`
	FileCount   int       `json:"file_count" db:"file_count"`
	SketchCount int       `json:"sketch_count" db:"sketch_count"`
}

// RiskResult represents a complete risk analysis result for output formatting
type RiskResult struct {
	// Metadata
	Branch       string        `json:"branch"`
	FilesChanged int           `json:"files_changed"`
	RiskLevel    string        `json:"risk_level"`
	RiskScore    float64       `json:"risk_score"`
	Confidence   float64       `json:"confidence"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	CacheHit     bool          `json:"cache_hit"`

	// Analysis results
	Files               []FileRisk          `json:"files"`
	Issues              []RiskIssue         `json:"issues"`
	Recommendations     []string            `json:"recommendations"`
	Evidence            []string            `json:"evidence"`
	NextSteps           []string            `json:"next_steps"`
	InvestigationTrace  []InvestigationHop  `json:"investigation_trace"`

	// Graph metrics
	BlastRadius      int               `json:"blast_radius"`
	TemporalCoupling []CouplingPair    `json:"temporal_coupling"`
	Hotspots         []Hotspot         `json:"hotspots"`

	// Commit control (Session 3 - AI Mode)
	ShouldBlock                   bool   `json:"should_block_commit,omitempty"`
	BlockReason                   string `json:"block_reason,omitempty"`
	OverrideAllowed               bool   `json:"override_allowed,omitempty"`
	OverrideRequiresJustification bool   `json:"override_requires_justification,omitempty"`

	// Performance metrics (Session 3 - AI Mode)
	Performance Performance `json:"performance,omitempty"`

	// Contextual insights (Session 3 - AI Mode)
	SimilarPastChanges []SimilarChange      `json:"similar_past_changes,omitempty"`
	TeamPatterns       *TeamPatterns        `json:"team_patterns,omitempty"`
	FileReputation     map[string]Reputation `json:"file_reputation,omitempty"`
}

// Performance represents execution metrics for AI Mode
type Performance struct {
	TotalDurationMS int                    `json:"total_duration_ms"`
	Breakdown       map[string]int         `json:"breakdown"`
	CacheEfficiency map[string]interface{} `json:"cache_efficiency"`
}

// SimilarChange represents a similar past change for context
type SimilarChange struct {
	CommitSHA    string   `json:"commit_sha"`
	Date         string   `json:"date"`
	Author       string   `json:"author"`
	FilesChanged []string `json:"files_changed"`
	Outcome      string   `json:"outcome"`
	Lesson       string   `json:"lesson"`
}

// TeamPatterns represents team coding patterns
type TeamPatterns struct {
	AvgTestCoverage float64 `json:"avg_test_coverage"`
	YourCoverage    float64 `json:"your_coverage"`
	Percentile      int     `json:"percentile"`
	TeamAvgCoupling float64 `json:"team_avg_coupling"`
	YourCoupling    float64 `json:"your_coupling"`
	Recommendation  string  `json:"recommendation"`
}

// Reputation represents file reputation metrics
type Reputation struct {
	IncidentDensity        float64 `json:"incident_density"`
	TeamAvg                float64 `json:"team_avg"`
	Classification         string  `json:"classification"`
	ExtraReviewRecommended bool    `json:"extra_review_recommended"`
}

// FileRisk represents risk information for a single file
type FileRisk struct {
	Path         string            `json:"path"`
	Language     string            `json:"language"`
	LinesChanged int               `json:"lines_changed"`
	RiskScore    float64           `json:"risk_score"`
	Metrics      map[string]Metric `json:"metrics"`
}

// Metric represents a calculated metric with threshold
type Metric struct {
	Name      string   `json:"name"`
	Value     float64  `json:"value"`
	Threshold *float64 `json:"threshold,omitempty"`
	Warning   *float64 `json:"warning,omitempty"`
}

// RiskIssue represents a detected risk issue
type RiskIssue struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity"`
	Category string   `json:"category"`
	File     string   `json:"file"`
	Message  string   `json:"message"`
	LineStart int     `json:"line_start,omitempty"`
	LineEnd   int     `json:"line_end,omitempty"`
	Function  string  `json:"function,omitempty"`

	// AI Mode specific fields (Session 3)
	ImpactScore         float64  `json:"impact_score,omitempty"`
	FixPriority         int      `json:"fix_priority,omitempty"`
	EstimatedFixTimeMin int      `json:"estimated_fix_time_min,omitempty"`
	AutoFixable         bool     `json:"auto_fixable,omitempty"`
	FixType             string   `json:"fix_type,omitempty"` // "generate_tests", "add_error_handling", etc.
	FixConfidence       float64  `json:"fix_confidence,omitempty"`
	AIPromptTemplate    string   `json:"ai_prompt_template,omitempty"`
	ExpectedFiles       []string `json:"expected_files,omitempty"`
	EstimatedLines      int      `json:"estimated_lines,omitempty"`
	FixCommand          string   `json:"fix_command,omitempty"`
}

// InvestigationHop represents one step in the agent investigation
type InvestigationHop struct {
	NodeID          string            `json:"node_id"`
	NodeType        string            `json:"node_type"`
	ChangedEntities []ChangedEntity   `json:"changed_entities"`
	Metrics         []Metric          `json:"metrics"`
	Decision        string            `json:"decision"`
	Reasoning       string            `json:"reasoning"`
	Confidence      float64           `json:"confidence"`
	DurationMS      int64             `json:"duration_ms"`
}

// ChangedEntity represents a changed function or class
type ChangedEntity struct {
	Name      string `json:"name"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// CouplingPair represents temporal coupling between two files
type CouplingPair struct {
	FileA       string    `json:"file_a"`
	FileB       string    `json:"file_b"`
	Strength    float64   `json:"strength"`
	Commits     int       `json:"commits"`
	TotalCommits int      `json:"total_commits"`
	WindowDays  int       `json:"window_days"`
	LastCoChange time.Time `json:"last_co_change"`
}

// Hotspot represents a high-risk file hotspot
type Hotspot struct {
	File      string  `json:"file"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
	Churn     float64 `json:"churn"`
	Coverage  float64 `json:"coverage"`
	Incidents int     `json:"incidents"`
}
