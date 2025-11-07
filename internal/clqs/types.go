package clqs

import "time"

// CLQS component weights (pragmatically adjusted from CLQS spec v2.1)
// Adjusted weights to better reflect reality:
// - Deep link finder provides high-quality links with strong temporal+semantic
// - Reduced penalty for repos with fewer evidence types (deep links = 2/6 types)
const (
	WeightLinkCoverage      = 0.40 // Increased: covering more issues is most important
	WeightConfidenceQuality = 0.35 // Increased: confidence matters most for risk scoring
	WeightEvidenceDiversity = 0.15 // Decreased: don't over-penalize deep-link-heavy repos
	WeightTemporalPrecision = 0.05 // Decreased: nice-to-have but not critical
	WeightSemanticStrength  = 0.05 // Unchanged: still valuable but lightweight
)

// Evidence type categories (from linking system)
const (
	EvidenceExplicit              = "explicit"
	EvidenceGitHubTimelineVerified = "github_timeline_verified"
	EvidenceBidirectional         = "bidirectional"
	EvidenceSemantic              = "semantic" // ANY of semantic_title, semantic_body, semantic_comment
	EvidenceTemporal              = "temporal"
	EvidenceFileContext           = "file_context"
)

// CLQSScore represents the complete CLQS calculation result
type CLQSScore struct {
	RepoID               int64
	CLQS                 float64
	Grade                string
	Rank                 string
	ConfidenceMultiplier float64

	// Component scores (0-100)
	LinkCoverage      float64
	ConfidenceQuality float64
	EvidenceDiversity float64
	TemporalPrecision float64
	SemanticStrength  float64

	// Component contributions (weighted)
	LinkCoverageContribution      float64
	ConfidenceQualityContribution float64
	EvidenceDiversityContribution float64
	TemporalPrecisionContribution float64
	SemanticStrengthContribution  float64

	// Statistics
	TotalClosedIssues int
	EligibleIssues    int
	LinkedIssues      int
	TotalLinks        int
	AvgConfidence     float64

	ComputedAt time.Time
}

// CLQSReport represents the complete CLQS report for output
type CLQSReport struct {
	Repository            RepositoryInfo
	Overall               OverallScore
	Components            []*ComponentBreakdown
	Statistics            Statistics
	Recommendations       []Recommendation
	LabelingOpportunities LabelingOpportunities
}

// RepositoryInfo contains repository metadata
type RepositoryInfo struct {
	ID         int64     `json:"id"`
	FullName   string    `json:"full_name"`
	AnalyzedAt time.Time `json:"analyzed_at"`
}

// OverallScore contains the overall CLQS score
type OverallScore struct {
	CLQS                 float64 `json:"clqs"`
	Grade                string  `json:"grade"`
	Rank                 string  `json:"rank"`
	ConfidenceMultiplier float64 `json:"confidence_multiplier"`
}

// ComponentBreakdown represents a single CLQS component
type ComponentBreakdown struct {
	Name         string      `json:"name"`
	Score        float64     `json:"score"`
	Weight       float64     `json:"weight"`
	Contribution float64     `json:"contribution"`
	Status       string      `json:"status"` // "Excellent", "Good", "Fair", "Poor"
	Details      interface{} `json:"details"`
}

// Statistics contains overall linking statistics
type Statistics struct {
	TotalClosedIssues int     `json:"total_closed_issues"`
	EligibleIssues    int     `json:"eligible_issues"`
	LinkedIssues      int     `json:"linked_issues"`
	TotalLinks        int     `json:"total_links"`
	AvgConfidence     float64 `json:"avg_confidence"`
}

// Recommendation represents an improvement recommendation
type Recommendation struct {
	Priority    int    `json:"priority"`
	Category    string `json:"category"`
	Message     string `json:"message"`
	Action      string `json:"action"`
	ImpactLevel string `json:"impact_level"` // "high", "medium", "low"
}

// LabelingOpportunities identifies potential CLQS improvements
type LabelingOpportunities struct {
	LowConfidenceCount int     `json:"low_confidence_count"`
	AmbiguousCount     int     `json:"ambiguous_count"`
	PotentialCLQSGain  float64 `json:"potential_clqs_gain"`
}

// Component-specific detail structures

// LinkCoverageDetails contains Link Coverage component details
type LinkCoverageDetails struct {
	TotalClosedIssues int     `json:"total_closed_issues"`
	ExcludedIssues    int     `json:"excluded_issues"`
	EligibleIssues    int     `json:"eligible_issues"`
	LinkedIssues      int     `json:"linked_issues"`
	UnlinkedIssues    int     `json:"unlinked_issues"`
	CoverageRate      float64 `json:"coverage_rate"`
}

// ConfidenceQualityDetails contains Confidence Quality component details
type ConfidenceQualityDetails struct {
	HighConfidenceLinks   ConfidenceTier `json:"high_confidence_links"`
	MediumConfidenceLinks ConfidenceTier `json:"medium_confidence_links"`
	LowConfidenceLinks    ConfidenceTier `json:"low_confidence_links"`
	WeightedAverage       float64        `json:"weighted_average"`
}

// ConfidenceTier represents a confidence tier (high/medium/low)
type ConfidenceTier struct {
	Count         int     `json:"count"`
	Percentage    float64 `json:"percentage"`
	AvgConfidence float64 `json:"avg_confidence"`
}

// EvidenceDiversityDetails contains Evidence Diversity component details
type EvidenceDiversityDetails struct {
	AvgEvidenceTypes  float64        `json:"avg_evidence_types"`
	MaxPossible       int            `json:"max_possible"`
	Distribution      map[string]int `json:"distribution"`       // "6_types": count, "5_types": count, etc.
	EvidenceTypeUsage map[string]int `json:"evidence_type_usage"` // "explicit": count, "bidirectional": count, etc.
}

// TemporalPrecisionDetails contains Temporal Precision component details
type TemporalPrecisionDetails struct {
	TightTemporalLinks int            `json:"tight_temporal_links"`
	TotalLinks         int            `json:"total_links"`
	PrecisionRate      float64        `json:"precision_rate"`
	Distribution       map[string]int `json:"distribution"` // "under_5min", "5min_to_1hr", etc.
}

// SemanticStrengthDetails contains Semantic Strength component details
type SemanticStrengthDetails struct {
	AvgSemanticScore float64            `json:"avg_semantic_score"`
	Distribution     map[string]int     `json:"distribution"` // "high_semantic", "medium_semantic", "low_semantic"
	AvgByType        map[string]float64 `json:"avg_by_type"`  // "title", "body", "comment"
}

// Database query result structures

// LinkCoverageStats contains raw statistics for Link Coverage calculation
type LinkCoverageStats struct {
	TotalClosed    int
	EligibleIssues int
	LinkedIssues   int
	UnlinkedIssues int
}

// ConfidenceStats contains raw statistics for Confidence Quality calculation
type ConfidenceStats struct {
	HighCount   int
	HighAvg     float64
	MediumCount int
	MediumAvg   float64
	LowCount    int
	LowAvg      float64
	TotalLinks  int
}

// EvidenceDiversityStats contains raw statistics for Evidence Diversity calculation
type EvidenceDiversityStats struct {
	AvgEvidenceTypes    float64
	SixTypes            int
	FiveTypes           int
	FourTypes           int
	ThreeTypes          int
	TwoTypes            int
	OneType             int
	ExplicitCount       int
	TimelineCount       int
	BidirectionalCount  int
	SemanticCount       int
	TemporalCount       int
	FileContextCount    int
	TotalLinks          int
}

// TemporalPrecisionStats contains raw statistics for Temporal Precision calculation
type TemporalPrecisionStats struct {
	TotalLinks         int
	TightTemporalLinks int
	Under5Min          int
	FiveMinTo1Hr       int
	OneHrTo24Hr        int
	Over24Hr           int
}

// SemanticStrengthStats contains raw statistics for Semantic Strength calculation
type SemanticStrengthStats struct {
	AvgMaxSemantic float64
	HighSemantic   int
	MediumSemantic int
	LowSemantic    int
	AvgTitle       float64
	AvgBody        float64
	AvgComment     float64
	TotalLinks     int
}
