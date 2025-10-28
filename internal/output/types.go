package output

import (
	"time"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/metrics"
	"github.com/rohankatakam/coderisk/internal/types"
)

// AIJSONOutput is the complete schema for --ai-mode
// Reference: dev_docs/00-product/developer_experience.md lines 290-575
// 12-factor: Factor 4 - Tools are structured outputs
type AIJSONOutput struct {
	AIAssistantActions []AIAssistantAction `json:"ai_assistant_actions"`
	BlockReason        string              `json:"block_reason,omitempty"`
	Branch             string              `json:"branch"`
	ContextualInsights ContextualInsights  `json:"contextual_insights"`
	Files              []FileAnalysis      `json:"files"`
	GraphAnalysis      GraphAnalysis       `json:"graph_analysis"`
	InvestigationTrace []InvestigationHop  `json:"investigation_trace"`
	OverallRisk        string              `json:"overall_risk"`
	Recommendations    Recommendations     `json:"recommendations"`
	Repository         string              `json:"repository"`
	Timestamp          time.Time           `json:"timestamp"`
}

// AIAssistantAction represents an auto-fixable action for AI coding assistants
// 12-factor: Factor 4 - Tools are structured outputs
type AIAssistantAction struct {
	ActionType     string  `json:"action_type"`     // "add_test", "refactor", "add_error_handling", etc.
	Confidence     float64 `json:"confidence"`      // 0.0-1.0
	Description    string  `json:"description"`     // Human-readable description
	EstimatedLines int     `json:"estimated_lines"` // Code change size estimate
	FilePath       string  `json:"file_path"`
	Function       string  `json:"function,omitempty"`
	LineNumber     int     `json:"line_number,omitempty"`
	Prompt         string  `json:"prompt"`           // Ready-to-execute prompt for AI
	ReadyToExecute bool    `json:"ready_to_execute"` // Can be run without human review
	RiskReduction  float64 `json:"risk_reduction"`   // Expected risk score improvement
}

// FileAnalysis provides detailed analysis for a single file
type FileAnalysis struct {
	Path         string                 `json:"path"`
	Language     string                 `json:"language"`
	LinesChanged int                    `json:"lines_changed"`
	RiskScore    float64                `json:"risk_score"`
	Metrics      map[string]interface{} `json:"metrics"`
	Issues       []Issue                `json:"issues"`
}

// Issue represents a detected risk issue in a file
type Issue struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

// GraphAnalysis provides graph-based risk insights
type GraphAnalysis struct {
	BlastRadius      BlastRadius            `json:"blast_radius"`
	Hotspots         []Hotspot              `json:"hotspots"`
	TemporalCoupling []TemporalCouplingPair `json:"temporal_coupling"`
}

// BlastRadius shows impact of changes
type BlastRadius struct {
	DirectDependents     int      `json:"direct_dependents"`
	TotalAffectedFiles   int      `json:"total_affected_files"`
	CriticalPaths        []string `json:"critical_paths,omitempty"` // Files on critical execution paths
	TransitiveDependents int      `json:"transitive_dependents"`
}

// Hotspot identifies risky areas in codebase
type Hotspot struct {
	File          string  `json:"file"`
	Score         float64 `json:"score"`         // 0.0-1.0 risk score
	Reason        string  `json:"reason"`        // "high_churn_low_coverage", "incident_prone", etc.
	ChurnRate     float64 `json:"churn_rate"`    // Changes per week
	TestCoverage  float64 `json:"test_coverage"` // 0.0-1.0
	IncidentCount int     `json:"incident_count"`
}

// TemporalCouplingPair shows files that change together
type TemporalCouplingPair struct {
	FileA      string  `json:"file_a"`
	FileB      string  `json:"file_b"`
	Frequency  float64 `json:"frequency"`   // 0.0-1.0
	CoChanges  int     `json:"co_changes"`  // Number of times changed together
	WindowDays int     `json:"window_days"` // Analysis window
}

// ContextualInsights provides historical context
type ContextualInsights struct {
	FileReputation     map[string]float64 `json:"file_reputation"` // File path → stability score
	SimilarPastChanges []SimilarChange    `json:"similar_past_changes"`
	TeamPatterns       *TeamPattern       `json:"team_patterns,omitempty"`
}

// SimilarChange represents historically similar code changes
type SimilarChange struct {
	CommitSHA    string    `json:"commit_sha"`
	FilesChanged []string  `json:"files_changed"`
	Similarity   float64   `json:"similarity"` // 0.0-1.0
	Outcome      string    `json:"outcome"`    // "success", "incident", "reverted"
	Date         time.Time `json:"date"`
}

// TeamPattern shows team-specific risk patterns
type TeamPattern struct {
	PeakRiskHours  []int    `json:"peak_risk_hours"`  // Hours of day (0-23)
	SafeReviewers  []string `json:"safe_reviewers"`   // Developers with low FP rate
	RiskyFileTypes []string `json:"risky_file_types"` // Extensions with high incident rate
}

// Recommendations provides actionable next steps
type Recommendations struct {
	Critical []Recommendation `json:"critical"` // Must do before merge
	High     []Recommendation `json:"high"`     // Should do before merge
	Medium   []Recommendation `json:"medium"`   // Consider doing
	Future   []Recommendation `json:"future"`   // Technical debt / improvements
}

// Recommendation is a single suggested action
type Recommendation struct {
	Action        string `json:"action"`
	Reason        string `json:"reason"`
	EstimatedTime int    `json:"estimated_time"` // Minutes
	AutoFixable   bool   `json:"auto_fixable"`
	Priority      string `json:"priority"` // "critical", "high", "medium", "low"
}

// InvestigationHop represents one step in Phase 2 investigation
// (Populated when Phase 2 runs)
type InvestigationHop struct {
	Hop               int            `json:"hop"`
	NodeType          string         `json:"node_type"`
	NodeID            string         `json:"node_id"`
	Action            string         `json:"action"`
	MetricsCalculated []MetricResult `json:"metrics_calculated"`
	Decision          string         `json:"decision"`
	Reasoning         string         `json:"reasoning"`
	Confidence        float64        `json:"confidence"`
	DurationMS        int64          `json:"duration_ms"`
}

// MetricResult represents a calculated metric from investigation
type MetricResult struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// === Merged from converter.go ===

// ConvertPhase1ToRiskResult converts Phase1Result to RiskResult for formatting
func ConvertPhase1ToRiskResult(phase1 *metrics.Phase1Result) *types.RiskResult {
	// Get current branch dynamically
	branch, err := git.GetCurrentBranch()
	if err != nil {
		branch = "main" // Fallback if not in git repo
	}

	result := &types.RiskResult{
		Branch:       branch,
		FilesChanged: 1,
		RiskLevel:    string(phase1.OverallRisk),
		RiskScore:    calculateRiskScore(phase1),
		Confidence:   0.8, // Phase 1 has fixed confidence
		StartTime:    time.Now().Add(-time.Duration(phase1.DurationMS) * time.Millisecond),
		EndTime:      time.Now(),
		Duration:     time.Duration(phase1.DurationMS) * time.Millisecond,
		CacheHit:     false,
	}

	// Detect language from file extension
	language := git.DetectLanguage(phase1.FilePath)

	// Convert to FileRisk
	result.Files = []types.FileRisk{
		{
			Path:         phase1.FilePath,
			Language:     language,
			LinesChanged: 0, // TODO: Get from git diff (requires more complex git integration)
			RiskScore:    calculateRiskScore(phase1),
			Metrics:      convertMetrics(phase1),
		},
	}

	// Convert to issues
	result.Issues = convertToIssues(phase1)

	// Generate recommendations
	result.Recommendations = generateRecommendations(phase1)

	return result
}

func calculateRiskScore(phase1 *metrics.Phase1Result) float64 {
	switch phase1.OverallRisk {
	case metrics.RiskLevelHigh:
		return 8.0
	case metrics.RiskLevelMedium:
		return 5.0
	case metrics.RiskLevelLow:
		return 2.0
	default:
		return 0.0
	}
}

func convertMetrics(phase1 *metrics.Phase1Result) map[string]types.Metric {
	metrics := make(map[string]types.Metric)

	if phase1.Coupling != nil {
		threshold := 10.0
		metrics["coupling"] = types.Metric{
			Name:      "Structural Coupling",
			Value:     float64(phase1.Coupling.Count),
			Threshold: &threshold,
		}
	}

	if phase1.CoChange != nil {
		threshold := 0.7
		metrics["co_change"] = types.Metric{
			Name:      "Temporal Co-Change",
			Value:     phase1.CoChange.MaxFrequency,
			Threshold: &threshold,
		}
	}

	if phase1.TestRatio != nil {
		threshold := 0.3
		metrics["test_coverage"] = types.Metric{
			Name:      "Test Coverage",
			Value:     phase1.TestRatio.Ratio,
			Threshold: &threshold,
		}
	}

	return metrics
}

func convertToIssues(phase1 *metrics.Phase1Result) []types.RiskIssue {
	var issues []types.RiskIssue

	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		issues = append(issues, types.RiskIssue{
			ID:       "COUPLING_HIGH",
			Severity: string(phase1.Coupling.RiskLevel),
			Category: "coupling",
			File:     phase1.FilePath,
			Message:  phase1.Coupling.FormatEvidence(),
		})
	}

	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		issues = append(issues, types.RiskIssue{
			ID:       "COCHANGE_HIGH",
			Severity: string(phase1.CoChange.RiskLevel),
			Category: "temporal",
			File:     phase1.FilePath,
			Message:  phase1.CoChange.FormatEvidence(),
		})
	}

	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		issues = append(issues, types.RiskIssue{
			ID:       "TEST_COVERAGE_LOW",
			Severity: string(phase1.TestRatio.RiskLevel),
			Category: "quality",
			File:     phase1.FilePath,
			Message:  phase1.TestRatio.FormatEvidence(),
		})
	}

	return issues
}

func generateRecommendations(phase1 *metrics.Phase1Result) []string {
	var recs []string

	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		recs = append(recs, "Review and reduce structural coupling")
	}

	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		recs = append(recs, "Investigate temporal coupling patterns")
	}

	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		recs = append(recs, "Add test coverage for this file")
	}

	return recs
}
