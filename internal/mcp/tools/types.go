package tools

// CodeBlock represents a code block from Neo4j with ownership inline
type CodeBlock struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Type          string        `json:"block_type"`
	Path          string        `json:"file_path"`
	OwnershipData OwnershipData `json:"ownership"`
}

// OwnershipData from Neo4j CodeBlock node properties
type OwnershipData struct {
	OriginalAuthor     string `json:"original_author"`
	LastModifier       string `json:"last_modifier"`
	StaleDays          int    `json:"staleness_days"`
	FamiliarityMap     string `json:"familiarity_map"`     // JSON string
	SemanticImportance string `json:"semantic_importance"` // P0, P1, P2
}

// CoupledBlock represents a block with coupling relationship
type CoupledBlock struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Path          string  `json:"file_path"`
	Rate          float64 `json:"coupling_rate"`
	CoChangeCount int     `json:"co_change_count"`
}

// CouplingData from Neo4j CO_CHANGES_WITH edges
type CouplingData struct {
	CoupledWith []CoupledBlock `json:"coupled_blocks"`
}

// TemporalIncident represents a linked issue/PR
type TemporalIncident struct {
	IssueID         int     `json:"issue_id,omitempty"`
	IssueTitle      string  `json:"issue_title,omitempty"`
	IssueState      string  `json:"issue_state,omitempty"`
	PRID            int     `json:"pr_id,omitempty"`
	PRTitle         string  `json:"pr_title,omitempty"`
	PRState         string  `json:"pr_state,omitempty"`
	ConfidenceScore float64 `json:"confidence_score"`
}

// TemporalData from PostgreSQL code_block_incidents
type TemporalData struct {
	IncidentCount int                `json:"incident_count"`
	Incidents     []TemporalIncident `json:"incidents"`
}

// BlockEvidence is the final output per block
type BlockEvidence struct {
	BlockName          string             `json:"block_name"`
	BlockType          string             `json:"block_type"`
	FilePath           string             `json:"file_path"`
	// Ownership
	OriginalAuthor     string             `json:"original_author"`
	LastModifier       string             `json:"last_modifier"`
	StaleDays          int                `json:"staleness_days"`
	FamiliarityMap     string             `json:"familiarity_map"` // JSON string
	SemanticImportance string             `json:"semantic_importance"`
	// Coupling
	CoupledBlocks      []CoupledBlock     `json:"coupled_blocks"`
	// Temporal
	IncidentCount      int                `json:"incident_count"`
	Incidents          []TemporalIncident `json:"incidents"`
	// Risk Score (optional, for debugging/verification)
	RiskScore          *float64           `json:"risk_score,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
