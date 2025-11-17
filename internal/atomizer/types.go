package atomizer

import "time"

// CommitChangeEventLog represents LLM-extracted semantic events from a commit
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Phase 1 Pipeline 2
type CommitChangeEventLog struct {
	CommitSHA        string        `json:"commit_sha"`
	AuthorEmail      string        `json:"author_email"`
	Timestamp        time.Time     `json:"timestamp"`
	LLMIntentSummary string        `json:"llm_intent_summary"`
	MentionedIssues  []string      `json:"mentioned_issues_in_msg"`
	ChangeEvents     []ChangeEvent `json:"change_events"`
}

// ChangeEvent represents a single code block modification
type ChangeEvent struct {
	Behavior        string `json:"behavior"`                  // CREATE_BLOCK, MODIFY_BLOCK, DELETE_BLOCK, ADD_IMPORT, REMOVE_IMPORT
	TargetFile      string `json:"target_file"`               // Path to the file
	TargetBlockName string `json:"target_block_name,omitempty"` // Name of function/method/class
	BlockType       string `json:"block_type,omitempty"`      // function, method, class, component
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
}
