package audit

import (
	"encoding/json"
	"os"
	"time"
)

// OverrideEvent represents a logged override when --no-verify is used
// Reference: ux_pre_commit_hook.md - Override tracking
type OverrideEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Author     string    `json:"author"`
	CommitSHA  string    `json:"commit_sha,omitempty"`
	RiskLevel  string    `json:"risk_level"`
	FilesCount int       `json:"files_count"`
	Issues     []string  `json:"issues"`
	Reason     string    `json:"reason,omitempty"`
}

// LogOverride records when --no-verify is used to bypass risk check
// Logs are written to .coderisk/hook_log.jsonl in JSONL format
func LogOverride(event OverrideEvent) error {
	logFile := ".coderisk/hook_log.jsonl"

	// Create directory if needed
	if err := os.MkdirAll(".coderisk", 0755); err != nil {
		return err
	}

	// Open file in append mode (create if doesn't exist)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode as JSON and append to file
	encoder := json.NewEncoder(f)
	return encoder.Encode(event)
}
