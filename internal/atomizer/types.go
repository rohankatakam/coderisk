package atomizer

import (
	"fmt"
	"log"
	"time"
)

// LLMExtractionResponse represents ONLY the semantic data that LLM should extract
// This struct is used for server-side schema validation with Google's ResponseSchema
// Reference: YC_DEMO_GAP_ANALYSIS.md - Fix hallucination by separating metadata and structured data
type LLMExtractionResponse struct {
	LLMIntentSummary string             `json:"llm_intent_summary"` // One sentence summary of the change intent
	MentionedIssues  []string           `json:"mentioned_issues_in_msg"` // Issue numbers mentioned in commit message (e.g., "#123")
	ChangeEvents     []LLMChangeEvent   `json:"change_events"` // List of code block changes (without file paths)
}

// LLMChangeEvent represents what the LLM extracts (WITHOUT file paths or line numbers)
// File paths and line numbers are parsed from diff headers to prevent hallucination
type LLMChangeEvent struct {
	Behavior        string `json:"behavior"`                  // CREATE_BLOCK, MODIFY_BLOCK, DELETE_BLOCK, RENAME_BLOCK, ADD_IMPORT, REMOVE_IMPORT
	TargetBlockName string `json:"target_block_name,omitempty"` // Name of function/method/class
	OldBlockName    string `json:"old_block_name,omitempty"`  // For RENAME_BLOCK: old function name
	Signature       string `json:"signature,omitempty"`       // Function signature with parameter types and return type
	BlockType       string `json:"block_type,omitempty"`      // function, method, class, component
	DependencyPath  string `json:"dependency_path,omitempty"` // For imports: package/module path
	OldVersion      string `json:"old_version,omitempty"`     // Old code snippet (for modifications)
	NewVersion      string `json:"new_version,omitempty"`     // New code snippet (for modifications)
}

// CommitChangeEventLog represents the FULL commit event log including metadata
// Metadata (SHA, timestamp, author) is appended AFTER LLM extraction to prevent hallucination
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Phase 1 Pipeline 2
type CommitChangeEventLog struct {
	CommitSHA        string        `json:"commit_sha"` // Appended from CommitData, NOT from LLM
	AuthorEmail      string        `json:"author_email"` // Appended from CommitData, NOT from LLM
	Timestamp        time.Time     `json:"timestamp"` // Appended from CommitData, NOT from LLM
	LLMIntentSummary string        `json:"llm_intent_summary"` // From LLM
	MentionedIssues  []string      `json:"mentioned_issues_in_msg"` // From LLM
	ChangeEvents     []ChangeEvent `json:"change_events"` // From LLM
}

// ChangeEvent represents a single code block modification
type ChangeEvent struct {
	Behavior        string `json:"behavior"`                  // CREATE_BLOCK, MODIFY_BLOCK, DELETE_BLOCK, RENAME_BLOCK, ADD_IMPORT, REMOVE_IMPORT
	TargetFile      string `json:"target_file"`               // Path to the file
	TargetBlockName string `json:"target_block_name,omitempty"` // Name of function/method/class
	OldBlockName    string `json:"old_block_name,omitempty"`  // For RENAME_BLOCK: old function name
	Signature       string `json:"signature,omitempty"`       // Function signature with parameter types and return type
	BlockType       string `json:"block_type,omitempty"`      // function, method, class, component
	StartLine       int    `json:"start_line,omitempty"`      // Starting line number of the code block
	EndLine         int    `json:"end_line,omitempty"`        // Ending line number of the code block
	DependencyPath  string `json:"dependency_path,omitempty"` // For imports: package/module path
	OldVersion      string `json:"old_version,omitempty"`     // Old code snippet (for modifications)
	NewVersion      string `json:"new_version,omitempty"`     // New code snippet (for modifications)
}

// CommitData represents the input data for atomization
// Aligned with database.CommitData structure
type CommitData struct {
	SHA         string
	Message     string
	DiffContent string // Git diff for this commit
	AuthorEmail string
	Timestamp   time.Time
	RepoID      int64     // Repository ID for database operations
	AuthorDate  time.Time // Author date for rename tracking
}

// ValidateEvent checks if a ChangeEvent is valid
func (e *ChangeEvent) ValidateEvent() error {
	switch e.Behavior {
	case "RENAME_BLOCK":
		if e.OldBlockName == "" {
			return fmt.Errorf("RENAME_BLOCK requires old_block_name")
		}
		if e.TargetBlockName == "" {
			return fmt.Errorf("RENAME_BLOCK requires target_block_name")
		}
		if e.OldBlockName == e.TargetBlockName {
			return fmt.Errorf("RENAME_BLOCK old and new names cannot be same")
		}
		if e.Signature == "" {
			log.Printf("⚠️  WARNING: RENAME_BLOCK missing signature for block %s", e.TargetBlockName)
		}
	case "CREATE_BLOCK", "MODIFY_BLOCK":
		if e.Signature == "" {
			log.Printf("⚠️  WARNING: %s missing signature for block %s", e.Behavior, e.TargetBlockName)
		}
	case "DELETE_BLOCK":
		// Signature not required
	case "ADD_IMPORT", "REMOVE_IMPORT":
		// Signature not applicable
	default:
		return fmt.Errorf("unknown behavior: %s", e.Behavior)
	}
	return nil
}
