package types

import "time"

// DetectionMethod represents how a link was discovered
type DetectionMethod string

const (
	DetectionGitHubTimeline    DetectionMethod = "github_timeline_verified"
	DetectionExplicit          DetectionMethod = "explicit"
	DetectionExplicitBidir     DetectionMethod = "explicit_bidirectional"
	DetectionDeepLinkFinder    DetectionMethod = "deep_link_finder"
)

// LinkQuality represents the quality tier of a link
type LinkQuality string

const (
	QualityHigh   LinkQuality = "high"   // >= 0.85
	QualityMedium LinkQuality = "medium" // 0.70-0.84
	QualityLow    LinkQuality = "low"    // < 0.70
)

// ReferenceType represents the type of reference
type ReferenceType string

const (
	RefFixes     ReferenceType = "fixes"
	RefCloses    ReferenceType = "closes"
	RefResolves  ReferenceType = "resolves"
	RefAddresses ReferenceType = "addresses"
	RefFor       ReferenceType = "for"
	RefMentions  ReferenceType = "mentions"
	RefOther     ReferenceType = "other"
)

// ReferenceLocation represents where a reference was found
type ReferenceLocation string

const (
	LocPRTitle       ReferenceLocation = "pr_title"
	LocPRDescription ReferenceLocation = "pr_description"
	LocPRComment     ReferenceLocation = "pr_comment"
	LocTimelineAPI   ReferenceLocation = "timeline_api"
	LocIssueBody     ReferenceLocation = "issue_description"
	LocIssueComment  ReferenceLocation = "issue_comment"
)

// TemporalPattern represents the temporal relationship pattern
type TemporalPattern string

const (
	PatternNormal       TemporalPattern = "normal"
	PatternReverse      TemporalPattern = "reverse"
	PatternSimultaneous TemporalPattern = "simultaneous"
	PatternDelayed      TemporalPattern = "delayed"
)

// ClosureClassification represents the issue closure type
type ClosureClassification string

const (
	ClassFixedWithCode        ClosureClassification = "fixed_with_code"
	ClassNotABug              ClosureClassification = "not_a_bug"
	ClassDuplicate            ClosureClassification = "duplicate"
	ClassWontFix              ClosureClassification = "wontfix"
	ClassUserActionRequired   ClosureClassification = "user_action_required"
	ClassUnclear              ClosureClassification = "unclear"
)

// DORAMetrics contains repository-level DORA metrics
type DORAMetrics struct {
	MedianLeadTimeHours      float64   `json:"median_lead_time_hours"`
	MedianPRLifespanHours    float64   `json:"median_pr_lifespan_hours"`
	ComputedAt               time.Time `json:"computed_at"`
	SampleSize               int       `json:"sample_size"`
	InsufficientHistory      bool      `json:"insufficient_history"`
	TimelineEventsFetched    int       `json:"timeline_events_fetched"`
	CrossReferenceLinksFound int       `json:"cross_reference_links_found"`
}

// ExplicitReference represents a reference extracted from PR content
type ExplicitReference struct {
	IssueNumber       int               `json:"issue_number"`
	PRNumber          int               `json:"pr_number"` // Added: source PR number
	ReferenceType     ReferenceType     `json:"reference_type"`
	ReferenceLocation ReferenceLocation `json:"reference_location"`
	ExtractedText     string            `json:"extracted_text"`
	BaseConfidence    float64           `json:"base_confidence"`
	DetectionMethod   DetectionMethod   `json:"detection_method"`
	ExternalRepo      bool              `json:"external_repo"`
}

// SemanticScores contains semantic similarity analysis results
type SemanticScores struct {
	TitleScore          float64  `json:"title_score"`
	BodyScore           float64  `json:"body_score"`
	CommentScore        float64  `json:"comment_score"`
	CrossContentScore   float64  `json:"cross_content_score"`
	TitleKeywords       []string `json:"title_keywords"`
	BodyKeywords        []string `json:"body_keywords"`
	CommentKeywords     []string `json:"comment_keywords"`
	TitleRationale      string   `json:"title_rationale"`
	BodyRationale       string   `json:"body_rationale"`
	CrossContentRationale string `json:"cross_content_rationale"`
}

// TemporalAnalysis contains temporal correlation analysis
type TemporalAnalysis struct {
	IssueClosedAt       time.Time       `json:"issue_closed_at"`
	PRMergedAt          time.Time       `json:"pr_merged_at"`
	TemporalDeltaSeconds int64          `json:"temporal_delta_seconds"`
	TemporalPattern     TemporalPattern `json:"temporal_pattern"`
	TemporalDirection   string          `json:"temporal_direction"` // "normal" or "reverse"
}

// ConfidenceBreakdown contains all confidence components
type ConfidenceBreakdown struct {
	BaseConfidence       float64 `json:"base_confidence"`
	BidirectionalBoost   float64 `json:"bidirectional_boost"`
	SemanticBoost        float64 `json:"semantic_boost"`
	TemporalBoost        float64 `json:"temporal_boost"`
	FileContextBoost     float64 `json:"file_context_boost"`
	SharedPRPenalty      float64 `json:"shared_pr_penalty"`
	NegativeSignalPenalty float64 `json:"negative_signal_penalty"`
}

// LinkOutput represents a validated issue-PR link
type LinkOutput struct {
	IssueNumber           int                  `json:"issue_number"`
	PRNumber              int                  `json:"pr_number"`
	DetectionMethod       DetectionMethod      `json:"detection_method"`
	FinalConfidence       float64              `json:"final_confidence"`
	LinkQuality           LinkQuality          `json:"link_quality"`
	ConfidenceBreakdown   ConfidenceBreakdown  `json:"confidence_breakdown"`
	EvidenceSources       []string             `json:"evidence_sources"`
	ComprehensiveRationale string              `json:"comprehensive_rationale"`
	SemanticAnalysis      *SemanticScores      `json:"semantic_analysis,omitempty"`
	TemporalAnalysis      *TemporalAnalysis    `json:"temporal_analysis,omitempty"`
	Flags                 LinkFlags            `json:"flags"`
	Metadata              LinkMetadata         `json:"metadata"`
}

// LinkFlags contains boolean flags for link characteristics
type LinkFlags struct {
	NeedsManualReview bool   `json:"needs_manual_review"`
	AmbiguityReason   string `json:"ambiguity_reason,omitempty"`
	SharedPR          bool   `json:"shared_pr"`
	ReverseTemporal   bool   `json:"reverse_temporal"`
}

// LinkMetadata contains metadata about link creation
type LinkMetadata struct {
	LinkCreatedAt time.Time     `json:"link_created_at"`
	PhaseTimings  PhaseTimings  `json:"phase_timings"`
}

// PhaseTimings tracks execution time for each phase
type PhaseTimings struct {
	Phase0TimelineMS int64 `json:"phase_0_timeline_ms"`
	Phase1MS         int64 `json:"phase_1_ms"`
	Phase2MS         int64 `json:"phase_2_ms"`
}

// NoLinkReason represents why no link was created
type NoLinkReason string

const (
	NoLinkNotABug                    NoLinkReason = "not_a_bug"
	NoLinkDuplicate                  NoLinkReason = "duplicate"
	NoLinkWontFix                    NoLinkReason = "wontfix"
	NoLinkUserAction                 NoLinkReason = "user_action_required"
	NoLinkTemporalCoincidence        NoLinkReason = "temporal_coincidence_rejected"
	NoLinkAmbiguousClassificationWeak NoLinkReason = "ambiguous_classification_weak_match"
	NoLinkNoTemporalMatches          NoLinkReason = "no_temporal_matches"
	NoLinkNoSemanticMatches          NoLinkReason = "no_semantic_matches"
)

// NoLinkOutput represents an issue with no PR links found
type NoLinkOutput struct {
	IssueNumber              int                    `json:"issue_number"`
	NoLinksReason            NoLinkReason           `json:"no_links_reason"`
	Classification           ClosureClassification  `json:"classification"`
	ClassificationConfidence float64                `json:"classification_confidence"`
	ClassificationRationale  string                 `json:"classification_rationale"`
	ConversationSummary      string                 `json:"conversation_summary"`
	CandidatesEvaluated      int                    `json:"candidates_evaluated,omitempty"`
	BestCandidateScore       float64                `json:"best_candidate_score,omitempty"`
	SafetyBrakeReason        string                 `json:"safety_brake_reason,omitempty"`
	IssueClosedAt            time.Time              `json:"issue_closed_at"`
	AnalyzedAt               time.Time              `json:"analyzed_at"`
}

// BugClassificationResult contains bug classification analysis
type BugClassificationResult struct {
	IssueNumber              int                    `json:"issue_number"`
	ClosureClassification    ClosureClassification  `json:"closure_classification"`
	ClassificationConfidence float64                `json:"classification_confidence"`
	ClassificationRationale  string                 `json:"classification_rationale"`
	ConversationSummary      string                 `json:"conversation_summary"`
	KeyDecisionSnippets      []string               `json:"key_decision_snippets"`
	ClosingCommentAuthor     string                 `json:"closing_comment_author"`
	ClosingCommentTimestamp  time.Time              `json:"closing_comment_timestamp"`
	LowClassificationConfidence bool                `json:"low_classification_confidence"`
}

// CandidatePR represents a candidate PR for deep linking
type CandidatePR struct {
	PRNumber              int              `json:"pr_number"`
	PRTitle               string           `json:"pr_title"`
	PRDescription         string           `json:"pr_description"`
	PRComments            []string         `json:"pr_comments"`
	PRMergedAt            time.Time        `json:"pr_merged_at"`
	TemporalDeltaSeconds  int64            `json:"temporal_delta_seconds"`
	TemporalDirection     string           `json:"temporal_direction"`
	WindowUsedDays        float64          `json:"window_used_days"`
	WeakTemporalSignal    bool             `json:"weak_temporal_signal"`
	RankingScore          float64          `json:"ranking_score"`
	SemanticScores        SemanticScores   `json:"semantic_scores"`
	FileChangeSummary     string           `json:"file_change_summary"`
	SharedPRContext       string           `json:"shared_pr_context,omitempty"`
	ComprehensiveRationale string          `json:"comprehensive_rationale"`
}

// TimelineLink represents a GitHub-verified timeline link
type TimelineLink struct {
	IssueNumber    int
	PRNumber       int
	ReferenceType  ReferenceType
	ExtractedText  string
	BaseConfidence float64
}

// IssueData represents issue data for processing
type IssueData struct {
	IssueNumber int
	Title       string
	Body        string
	State       string
	Labels      []string
	CreatedAt   time.Time
	ClosedAt    *time.Time
	Comments    []CommentData
}

// CommentData represents a comment with metadata
type CommentData struct {
	Author        string
	CreatedAt     time.Time
	Body          string
	WasTruncated  bool
	AuthorRole    string // "OWNER", "MEMBER", "CONTRIBUTOR", "NONE"
}

// PRData represents PR data for processing
type PRData struct {
	PRNumber       int
	Title          string
	Body           string
	State          string
	Merged         bool
	MergedAt       *time.Time
	CreatedAt      time.Time
	Comments       []CommentData
	MergeCommitSHA *string
	Files          []PRFileData
}

// PRFileData represents a file changed in a PR
type PRFileData struct {
	Filename         string
	Status           string
	Additions        int
	Deletions        int
	PreviousFilename *string
}
