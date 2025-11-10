package agent

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// ConversationPhase represents the current phase of the investigation
type ConversationPhase string

const (
	PhaseInitialized          ConversationPhase = "INITIALIZED"
	PhasePhase1Running        ConversationPhase = "PHASE_1_RUNNING"
	PhasePhase2Investigating  ConversationPhase = "PHASE_2_INVESTIGATING"
	PhaseAwaitingHuman        ConversationPhase = "AWAITING_HUMAN"
	PhaseAssessmentComplete   ConversationPhase = "ASSESSMENT_COMPLETE"
)

// TerminalState represents the final outcome of an investigation
type TerminalState string

const (
	SafeToCommit            TerminalState = "SAFE_TO_COMMIT"
	RisksUnresolved         TerminalState = "RISKS_UNRESOLVED"
	BlockedWaiting          TerminalState = "BLOCKED_WAITING"
	InvestigationIncomplete TerminalState = "INVESTIGATION_INCOMPLETE"
	InvestigationAborted    TerminalState = "INVESTIGATION_ABORTED"
)

// DirectiveInvestigation represents the complete state of a directive investigation
// This can be checkpointed and resumed across CLI sessions
// Distinct from Investigation (types.go) which is for the RiskInvestigator agent
type DirectiveInvestigation struct {
	ID            string            `json:"id"`
	Phase         ConversationPhase `json:"phase"`
	TerminalState *TerminalState    `json:"terminal_state,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`

	// Context
	FilePaths     []string       `json:"file_paths"`
	RepositoryID  int64          `json:"repository_id,omitempty"`
	Phase1Results *Phase1Result  `json:"phase1_results,omitempty"`

	// Decision tracking
	Decisions     []Decision     `json:"decisions"`

	// Resumption state
	CanResume     bool           `json:"can_resume"`
	ResumeData    map[string]interface{} `json:"resume_data,omitempty"`

	// Final assessment
	FinalRiskLevel   string  `json:"final_risk_level,omitempty"`
	FinalConfidence  float64 `json:"final_confidence,omitempty"`
	FinalSummary     string  `json:"final_summary,omitempty"`
}

// Decision represents a user decision at a directive decision point
type Decision struct {
	DecisionNum  int               `json:"decision_num"`
	Directive    *DirectiveMessage `json:"directive"`
	UserChoice   string            `json:"user_choice"`   // Which option they chose
	UserResponse string            `json:"user_response,omitempty"` // Text response (for manual contacts)
	Timestamp    time.Time         `json:"timestamp"`
}

// Phase1Result represents the results of Phase 1 baseline analysis
type Phase1Result struct {
	RiskLevel         string  `json:"risk_level"` // LOW, MEDIUM, HIGH
	CouplingScore     float64 `json:"coupling_score"`
	CoChangeScore     float64 `json:"co_change_score"`
	TestRatio         float64 `json:"test_ratio"`
	IncidentCount     int     `json:"incident_count"`
	RecommendPhase2   bool    `json:"recommend_phase2"`
}

// NewDirectiveInvestigation creates a new directive investigation instance
func NewDirectiveInvestigation(files []string, repoID int64) *DirectiveInvestigation {
	return &DirectiveInvestigation{
		ID:           generateID(),
		Phase:        PhaseInitialized,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		FilePaths:    files,
		RepositoryID: repoID,
		Decisions:    []Decision{},
		CanResume:    true,
		ResumeData:   make(map[string]interface{}),
	}
}

// AddDecision appends a decision to the investigation history
func (inv *DirectiveInvestigation) AddDecision(directive *DirectiveMessage, userChoice string, userResponse string) {
	decisionNum := len(inv.Decisions) + 1
	inv.Decisions = append(inv.Decisions, Decision{
		DecisionNum:  decisionNum,
		Directive:    directive,
		UserChoice:   userChoice,
		UserResponse: userResponse,
		Timestamp:    time.Now(),
	})
	inv.UpdatedAt = time.Now()
}

// UpdatePhase transitions the investigation to a new phase
func (inv *DirectiveInvestigation) UpdatePhase(phase ConversationPhase) {
	inv.Phase = phase
	inv.UpdatedAt = time.Now()
}

// SetPhase1Results stores Phase 1 analysis results
func (inv *DirectiveInvestigation) SetPhase1Results(results *Phase1Result) {
	inv.Phase1Results = results
	inv.UpdatedAt = time.Now()
}

// SetTerminalState marks the investigation as complete with a final state
func (inv *DirectiveInvestigation) SetTerminalState(state TerminalState, riskLevel string, confidence float64, summary string) {
	inv.TerminalState = &state
	inv.FinalRiskLevel = riskLevel
	inv.FinalConfidence = confidence
	inv.FinalSummary = summary
	inv.CanResume = false
	inv.UpdatedAt = time.Now()
}

// SetResumeData stores arbitrary data for resumption
func (inv *DirectiveInvestigation) SetResumeData(key string, value interface{}) {
	inv.ResumeData[key] = value
	inv.UpdatedAt = time.Now()
}

// GetResumeData retrieves resume data by key
func (inv *DirectiveInvestigation) GetResumeData(key string) (interface{}, bool) {
	val, ok := inv.ResumeData[key]
	return val, ok
}

// IsComplete returns true if the investigation has reached a terminal state
func (inv *DirectiveInvestigation) IsComplete() bool {
	return inv.TerminalState != nil
}

// generateID creates a unique investigation ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
