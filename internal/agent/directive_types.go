package agent

// DirectiveMessage represents a decision point requiring user interaction
// Used in the directive agent flow for manual human-in-the-loop workflows
type DirectiveMessage struct {
	Action        DirectiveAction       `json:"action"`
	Reasoning     string                `json:"reasoning"`
	Evidence      []DirectiveEvidence   `json:"evidence"`
	Contingencies []ContingencyPlan     `json:"contingencies"`
	UserOptions   []UserOption          `json:"user_options"`
}

// DirectiveAction describes a proposed action for the user to take
type DirectiveAction struct {
	Type        string                 `json:"type"` // "CONTACT_HUMAN", "DEEP_INVESTIGATION", "ESCALATE", etc.
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ContingencyPlan defines what should happen based on different outcomes
type ContingencyPlan struct {
	Trigger     string          `json:"trigger"`      // Human-readable trigger description
	Condition   string          `json:"condition"`    // Machine-parseable condition
	NextAction  DirectiveAction `json:"next_action"`
	Confidence  float64         `json:"confidence"`   // How confident we are in this path (0.0-1.0)
}

// UserOption represents a choice the user can make at a decision point
type UserOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Action      string `json:"action"` // "APPROVE", "MODIFY", "SKIP", "ABORT"
	Shortcut    string `json:"shortcut"`
}

// DirectiveEvidence represents supporting data for a directive
// Distinct from Evidence (types.go) used by RiskInvestigator
type DirectiveEvidence struct {
	Type   string `json:"type"`   // "ownership", "incident", "cochange", etc.
	Data   string `json:"data"`   // Human-readable data
	Source string `json:"source"` // Where this evidence came from
}

// DirectiveType constants for common directive types
const (
	DirectiveTypeContactHuman      = "CONTACT_HUMAN"
	DirectiveTypeDeepInvestigation = "DEEP_INVESTIGATION"
	DirectiveTypeEscalate          = "ESCALATE"
	DirectiveTypeProceedWithCaution = "PROCEED_WITH_CAUTION"
)

// UserActionType constants for user choices
const (
	UserActionApprove = "APPROVE"
	UserActionModify  = "MODIFY"
	UserActionSkip    = "SKIP"
	UserActionAbort   = "ABORT"
)
