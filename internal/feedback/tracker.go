package feedback

import (
	"time"
)

// Tracker manages false positive feedback collection
type Tracker struct {
	// TODO: Add storage backend (PostgreSQL)
}

// NewTracker creates a new feedback tracker
func NewTracker() *Tracker {
	return &Tracker{}
}

// FeedbackEntry represents a single feedback submission
type FeedbackEntry struct {
	ID            string    `json:"id"`
	RepoID        string    `json:"repo_id"`
	CommitSHA     string    `json:"commit_sha"`
	FilePath      string    `json:"file_path"`
	RiskLevel     string    `json:"risk_level"`
	UserFeedback  string    `json:"user_feedback"` // "false_positive", "correct", "too_strict", "too_lenient"
	UserComment   string    `json:"user_comment"`
	Timestamp     time.Time `json:"timestamp"`
}

// RecordFeedback stores user feedback about a risk assessment
func (t *Tracker) RecordFeedback(entry *FeedbackEntry) error {
	// TODO: Implement storage
	return nil
}
