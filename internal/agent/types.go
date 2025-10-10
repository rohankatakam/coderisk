package agent

import (
	"time"

	"github.com/google/uuid"
)

// InvestigationRequest represents a Phase 2 escalation
type InvestigationRequest struct {
	RequestID   uuid.UUID
	FilePath    string
	ChangeType  string // "modify" | "create" | "delete"
	DiffPreview string // First 50 lines of git diff
	Baseline    BaselineMetrics
	StartedAt   time.Time
}

// BaselineMetrics from Phase 1 (provided by ingestion package)
type BaselineMetrics struct {
	CouplingScore     float64 // 0.0-1.0
	CoChangeFrequency float64 // 0.0-1.0
	IncidentCount     int
	TestCoverage      float64 // 0.0-1.0 (future)
	OwnershipDays     int     // Days since ownership transition
}

// Investigation represents the full investigation state
type Investigation struct {
	Request     InvestigationRequest
	Hops        []HopResult
	Evidence    []Evidence
	RiskScore   float64 // 0.0-1.0
	Confidence  float64 // 0.0-1.0
	Summary     string
	CompletedAt time.Time
	TotalTokens int
}

// HopResult represents the result of a single hop
type HopResult struct {
	HopNumber      int
	Query          string   // LLM query sent
	Response       string   // LLM response
	NodesVisited   []string // Node IDs visited
	EdgesTraversed []string // Edge types traversed
	TokensUsed     int
	Duration       time.Duration
}

// Evidence represents a single piece of risk evidence
type Evidence struct {
	Type        EvidenceType
	Description string
	Severity    float64 // 0.0-1.0
	Source      string  // "temporal" | "incidents" | "structure"
	FilePath    string
}

// EvidenceType categorizes evidence
type EvidenceType string

const (
	EvidenceCoChange     EvidenceType = "co_change"     // Files change together
	EvidenceIncident     EvidenceType = "incident"      // Past production incidents
	EvidenceOwnership    EvidenceType = "ownership"     // Ownership transition
	EvidenceCoupling     EvidenceType = "coupling"      // High structural coupling
	EvidenceComplexity   EvidenceType = "complexity"    // High cyclomatic complexity
	EvidenceMissingTests EvidenceType = "missing_tests" // No test coverage
)

// RiskAssessment is the final output
type RiskAssessment struct {
	FilePath      string
	RiskLevel     RiskLevel
	RiskScore     float64
	Confidence    float64
	Summary       string
	Evidence      []Evidence
	Investigation *Investigation // Full details
}

// RiskLevel categorizes risk
type RiskLevel string

const (
	RiskCritical RiskLevel = "CRITICAL" // >0.8
	RiskHigh     RiskLevel = "HIGH"     // 0.6-0.8
	RiskMedium   RiskLevel = "MEDIUM"   // 0.4-0.6
	RiskLow      RiskLevel = "LOW"      // 0.2-0.4
	RiskMinimal  RiskLevel = "MINIMAL"  // <0.2
)
