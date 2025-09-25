package models

import (
	"time"
)

// Repository represents a GitHub repository
type Repository struct {
	ID            string    `json:"id" db:"id"`
	Owner         string    `json:"owner" db:"owner"`
	Name          string    `json:"name" db:"name"`
	FullName      string    `json:"full_name" db:"full_name"`
	URL           string    `json:"url" db:"url"`
	DefaultBranch string    `json:"default_branch" db:"default_branch"`
	Language      string    `json:"language" db:"language"`
	Size          int64     `json:"size" db:"size"`
	StarCount     int       `json:"star_count" db:"star_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	LastIngested  time.Time `json:"last_ingested" db:"last_ingested"`
}

// Commit represents a git commit
type Commit struct {
	SHA         string    `json:"sha" db:"sha"`
	RepoID      string    `json:"repo_id" db:"repo_id"`
	Author      string    `json:"author" db:"author"`
	AuthorEmail string    `json:"author_email" db:"author_email"`
	Message     string    `json:"message" db:"message"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	ParentSHAs  []string  `json:"parent_shas"`
	Files       []*File   `json:"files"`
}

// File represents a file in the repository
type File struct {
	ID           string    `json:"id" db:"id"`
	RepoID       string    `json:"repo_id" db:"repo_id"`
	Path         string    `json:"path" db:"path"`
	Content      string    `json:"content,omitempty" db:"content"`
	Size         int64     `json:"size" db:"size"`
	Language     string    `json:"language" db:"language"`
	SHA          string    `json:"sha" db:"sha"`
	LastModified time.Time `json:"last_modified" db:"last_modified"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID          int        `json:"id" db:"id"`
	RepoID      string     `json:"repo_id" db:"repo_id"`
	Number      int        `json:"number" db:"number"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	State       string     `json:"state" db:"state"`
	Author      string     `json:"author" db:"author"`
	BaseBranch  string     `json:"base_branch" db:"base_branch"`
	HeadBranch  string     `json:"head_branch" db:"head_branch"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	MergedAt    *time.Time `json:"merged_at" db:"merged_at"`
	ClosedAt    *time.Time `json:"closed_at" db:"closed_at"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID         int        `json:"id" db:"id"`
	RepoID     string     `json:"repo_id" db:"repo_id"`
	Number     int        `json:"number" db:"number"`
	Title      string     `json:"title" db:"title"`
	Body       string     `json:"body" db:"body"`
	State      string     `json:"state" db:"state"`
	Author     string     `json:"author" db:"author"`
	Labels     []string   `json:"labels"`
	IsIncident bool       `json:"is_incident" db:"is_incident"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ClosedAt   *time.Time `json:"closed_at" db:"closed_at"`
}

// RiskLevel represents the risk severity
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// RiskAssessment represents a risk analysis result
type RiskAssessment struct {
	ID           string       `json:"id" db:"id"`
	RepoID       string       `json:"repo_id" db:"repo_id"`
	CommitSHA    string       `json:"commit_sha" db:"commit_sha"`
	Level        RiskLevel    `json:"level" db:"level"`
	Score        float64      `json:"score" db:"score"`
	Factors      []RiskFactor `json:"factors"`
	Suggestions  []string     `json:"suggestions"`
	BlastRadius  float64      `json:"blast_radius" db:"blast_radius"`
	TestCoverage float64      `json:"test_coverage" db:"test_coverage"`
	Timestamp    time.Time    `json:"timestamp" db:"timestamp"`
	CacheVersion string       `json:"cache_version" db:"cache_version"`
}

// RiskFactor represents a contributing factor to risk
type RiskFactor struct {
	Signal   string  `json:"signal"`
	Impact   string  `json:"impact"`
	Score    float64 `json:"score"`
	Detail   string  `json:"detail"`
	Evidence string  `json:"evidence,omitempty"`
}

// RiskSketch represents pre-computed risk signals
type RiskSketch struct {
	FileID            string             `json:"file_id" db:"file_id"`
	RepoID            string             `json:"repo_id" db:"repo_id"`
	PPRVector         []float64          `json:"ppr_vector"`
	CentralityScore   float64            `json:"centrality_score" db:"centrality_score"`
	CoChangeFrequency map[string]float64 `json:"co_change_frequency"`
	OwnershipScore    float64            `json:"ownership_score" db:"ownership_score"`
	TestCoverage      float64            `json:"test_coverage" db:"test_coverage"`
	IncidentHistory   []string           `json:"incident_history"`
	LastUpdated       time.Time          `json:"last_updated" db:"last_updated"`
}

// CacheMetadata represents cache version information
type CacheMetadata struct {
	Version     string    `json:"version" db:"version"`
	RepoID      string    `json:"repo_id" db:"repo_id"`
	LastCommit  string    `json:"last_commit" db:"last_commit"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
	Size        int64     `json:"size" db:"size"`
	FileCount   int       `json:"file_count" db:"file_count"`
	SketchCount int       `json:"sketch_count" db:"sketch_count"`
}
