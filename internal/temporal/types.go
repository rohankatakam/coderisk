package temporal

import "time"

// Commit represents a git commit
type Commit struct {
	SHA          string
	Author       string
	Email        string
	Timestamp    time.Time
	Message      string
	FilesChanged []FileChange
}

// FileChange represents file modifications in a commit
type FileChange struct {
	Path      string
	Additions int
	Deletions int
}

// Developer represents a code contributor
type Developer struct {
	Email        string
	Name         string
	FirstCommit  time.Time
	LastCommit   time.Time
	TotalCommits int
}

// CoChangeResult represents files that change together
type CoChangeResult struct {
	FileA      string
	FileB      string
	Frequency  float64 // 0.0 to 1.0 (how often they change together)
	CoChanges  int     // absolute count
	WindowDays int     // 90
}

// OwnershipHistory tracks who owns a file
type OwnershipHistory struct {
	FilePath       string
	CurrentOwner   string // email with most commits
	PreviousOwner  string // previous primary contributor
	TransitionDate time.Time
	DaysSince      int
}
