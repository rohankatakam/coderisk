package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GraphQueryExecutor is an interface for executing Neo4j queries
// This avoids import cycle with internal/graph package
type GraphQueryExecutor interface {
	ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
}

// HybridClient combines Neo4j graph queries with PostgreSQL details
type HybridClient struct {
	neo4jClient    GraphQueryExecutor
	postgresClient *StagingClient
}

// NewHybridClient creates a new hybrid query client
func NewHybridClient(neo4j GraphQueryExecutor, postgres *StagingClient) *HybridClient {
	return &HybridClient{
		neo4jClient:    neo4j,
		postgresClient: postgres,
	}
}

// IncidentWithContext combines graph and PostgreSQL data for incidents
type IncidentWithContext struct {
	// Graph data
	IssueNumber    int       `json:"issue_number"`
	PRNumber       int       `json:"pr_number"`
	CommitSHA      string    `json:"commit_sha"`
	LinkType       string    `json:"link_type"`        // "FIXED_BY" or "ASSOCIATED_WITH"
	Confidence     float64   `json:"confidence"`
	DetectionMethod string   `json:"detection_method"`
	Evidence       []string  `json:"evidence"`

	// PostgreSQL enrichment
	IssueTitle     string    `json:"issue_title"`
	IssueBody      string    `json:"issue_body"`
	IssueLabels    []string  `json:"issue_labels"`
	CreatedAt      time.Time `json:"created_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	AuthorLogin    string    `json:"author_login"`
	AuthorRole     string    `json:"author_role"`      // from author_association
}

// OwnershipHistory represents developer ownership data
type OwnershipHistory struct {
	Developer       string    `json:"developer"`
	Email           string    `json:"email"`
	CommitCount     int       `json:"commit_count"`
	LastCommitDate  time.Time `json:"last_commit_date"`
	DaysSinceCommit int       `json:"days_since_commit"`
	Role            string    `json:"role"`             // Inferred from PR author_association
	IsActive        bool      `json:"is_active"`        // Has recent activity
}

// CoChangePartnerContext represents co-change patterns with commit context
type CoChangePartnerContext struct {
	PartnerFile    string   `json:"partner_file"`
	CoChangeCount  int      `json:"cochange_count"`
	Frequency      float64  `json:"frequency"`
	SampleCommits  []string `json:"sample_commits"`    // Commit messages explaining why
}

// BlastRadiusFile represents a dependent file with incident history
type BlastRadiusFile struct {
	FilePath       string     `json:"file_path"`
	IncidentCount  int        `json:"incident_count"`
	LastIncident   *time.Time `json:"last_incident,omitempty"`
	DependencyType string     `json:"dependency_type"`  // "import" or "usage"
}

// GetIncidentHistoryForFiles retrieves incidents with full context
func (hc *HybridClient) GetIncidentHistoryForFiles(ctx context.Context, filePaths []string, daysBack int) ([]IncidentWithContext, error) {
	// Step 1: Query Neo4j graph for incident relationships
	query := `
		MATCH (issue:Issue)-[rel]->(pr:PR)<-[:IN_PR]-(c:Commit)-[:MODIFIED]->(f:File)
		WHERE f.path IN $paths
		  AND (type(rel) = 'FIXED_BY' OR type(rel) = 'ASSOCIATED_WITH')
		  AND issue.created_at > datetime() - duration({days: $days_back})
		RETURN
		  issue.number as issue_number,
		  pr.number as pr_number,
		  c.sha as commit_sha,
		  type(rel) as link_type,
		  rel.confidence as confidence,
		  rel.detection_method as detection_method,
		  rel.evidence_sources as evidence
		ORDER BY rel.confidence DESC, issue.created_at DESC
		LIMIT 50
	`

	graphResults, err := hc.neo4jClient.ExecuteQuery(ctx, query, map[string]any{
		"paths":     filePaths,
		"days_back": daysBack,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j query failed: %w", err)
	}

	if len(graphResults) == 0 {
		return []IncidentWithContext{}, nil
	}

	// Step 2: Extract issue numbers for PostgreSQL enrichment
	issueNumbers := make([]int, 0, len(graphResults))
	for _, row := range graphResults {
		if issueNum, ok := row["issue_number"].(int64); ok {
			issueNumbers = append(issueNumbers, int(issueNum))
		}
	}

	// Step 3: Query PostgreSQL for full issue details
	issueDetails, err := hc.getIssueDetails(ctx, issueNumbers)
	if err != nil {
		return nil, fmt.Errorf("postgres query failed: %w", err)
	}

	// Step 4: Merge graph data with PostgreSQL details
	incidents := make([]IncidentWithContext, 0, len(graphResults))
	for _, row := range graphResults {
		issueNum := int(row["issue_number"].(int64))
		details, ok := issueDetails[issueNum]
		if !ok {
			continue // Skip if details not found
		}

		// Extract evidence array
		var evidence []string
		if evidenceRaw, ok := row["evidence"].([]interface{}); ok {
			for _, e := range evidenceRaw {
				if s, ok := e.(string); ok {
					evidence = append(evidence, s)
				}
			}
		}

		incident := IncidentWithContext{
			IssueNumber:     issueNum,
			PRNumber:        int(row["pr_number"].(int64)),
			CommitSHA:       row["commit_sha"].(string),
			LinkType:        row["link_type"].(string),
			Confidence:      row["confidence"].(float64),
			DetectionMethod: row["detection_method"].(string),
			Evidence:        evidence,
			IssueTitle:      details.Title,
			IssueBody:       details.Body,
			IssueLabels:     details.Labels,
			CreatedAt:       details.CreatedAt,
			ClosedAt:        details.ClosedAt,
			AuthorLogin:     details.AuthorLogin,
			AuthorRole:      details.AuthorRole,
		}

		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// issueDetail holds PostgreSQL issue data
type issueDetail struct {
	Title       string
	Body        string
	Labels      []string
	CreatedAt   time.Time
	ClosedAt    *time.Time
	AuthorLogin string
	AuthorRole  string
}

// getIssueDetails fetches full issue details from PostgreSQL
func (hc *HybridClient) getIssueDetails(ctx context.Context, issueNumbers []int) (map[int]issueDetail, error) {
	if len(issueNumbers) == 0 {
		return map[int]issueDetail{}, nil
	}

	query := `
		SELECT
			number,
			title,
			body,
			labels,
			created_at,
			closed_at,
			user_login,
			author_association
		FROM github_issues
		WHERE number = ANY($1)
	`

	rows, err := hc.postgresClient.Query(ctx, query, issueNumbers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	details := make(map[int]issueDetail)
	for rows.Next() {
		var (
			number         int
			title          string
			body           string
			labelsJSON     []byte
			createdAt      time.Time
			closedAt       *time.Time
			authorLogin    string
			authorRole     string
		)

		if err := rows.Scan(&number, &title, &body, &labelsJSON, &createdAt, &closedAt, &authorLogin, &authorRole); err != nil {
			return nil, err
		}

		// Parse labels JSON
		var labelObjects []map[string]interface{}
		var labels []string
		if err := json.Unmarshal(labelsJSON, &labelObjects); err == nil {
			for _, labelObj := range labelObjects {
				if name, ok := labelObj["name"].(string); ok {
					labels = append(labels, name)
				}
			}
		}

		details[number] = issueDetail{
			Title:       title,
			Body:        body,
			Labels:      labels,
			CreatedAt:   createdAt,
			ClosedAt:    closedAt,
			AuthorLogin: authorLogin,
			AuthorRole:  authorRole,
		}
	}

	return details, nil
}

// GetOwnershipHistoryForFiles retrieves developer ownership with activity status
func (hc *HybridClient) GetOwnershipHistoryForFiles(ctx context.Context, filePaths []string) ([]OwnershipHistory, error) {
	// Step 1: Query Neo4j for commit ownership
	query := `
		MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File)
		WHERE f.path IN $paths
		WITH d.email as email, d.name as developer, COUNT(c) as commit_count,
		     MAX(c.committed_at) as last_commit_date
		ORDER BY commit_count DESC
		LIMIT 10
		RETURN email, developer, commit_count, last_commit_date
	`

	results, err := hc.neo4jClient.ExecuteQuery(ctx, query, map[string]any{
		"paths": filePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j query failed: %w", err)
	}

	ownership := make([]OwnershipHistory, 0, len(results))
	now := time.Now()

	for _, row := range results {
		// Safely extract email (handle both string and potential nil)
		email, _ := row["email"].(string)

		// Safely extract developer name (handle multiple types)
		var developer string
		switch v := row["developer"].(type) {
		case string:
			developer = v
		case int64:
			developer = fmt.Sprintf("dev-%d", v)
		default:
			developer = "unknown"
		}

		commitCount := int(row["commit_count"].(int64))

		// Parse last_commit_date (can be Unix timestamp or ISO 8601 string)
		var lastCommitDate time.Time
		switch v := row["last_commit_date"].(type) {
		case int64:
			// Neo4j stores as Unix timestamp
			lastCommitDate = time.Unix(v, 0)
		case string:
			// Try ISO 8601 format
			lastCommitDate, err = time.Parse(time.RFC3339, v)
			if err != nil {
				// Try alternative format
				lastCommitDate, _ = time.Parse("2006-01-02T15:04:05", v)
			}
		default:
			// Fallback to current time if parsing fails
			lastCommitDate = now
		}

		daysSince := int(now.Sub(lastCommitDate).Hours() / 24)
		isActive := daysSince < 90 // Active if committed within 90 days

		// TODO: Query PostgreSQL for author_association from recent PRs to determine role
		// For now, infer from email domain or set as "unknown"
		role := "CONTRIBUTOR"

		ownership = append(ownership, OwnershipHistory{
			Developer:       developer,
			Email:           email,
			CommitCount:     commitCount,
			LastCommitDate:  lastCommitDate,
			DaysSinceCommit: daysSince,
			Role:            role,
			IsActive:        isActive,
		})
	}

	return ownership, nil
}

// GetCoChangePartnersWithContext retrieves co-change files with commit message context
func (hc *HybridClient) GetCoChangePartnersWithContext(ctx context.Context, filePaths []string, threshold float64) ([]CoChangePartnerContext, error) {
	// Step 1: Query Neo4j for co-change patterns
	query := `
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)
		WHERE f1.path IN $paths
		WITH COUNT(DISTINCT c) as total_commits
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)-[:MODIFIED]->(f2:File)
		WHERE f1.path IN $paths AND f1.path <> f2.path
		WITH f2.path as partner_file,
		     COLLECT(DISTINCT c.sha)[0..3] as sample_commits,
		     COUNT(DISTINCT c) as cochange_count,
		     total_commits
		WITH partner_file, sample_commits, cochange_count,
		     (cochange_count * 1.0 / total_commits) as frequency
		WHERE frequency >= $threshold
		ORDER BY frequency DESC
		LIMIT 10
		RETURN partner_file, sample_commits, cochange_count, frequency
	`

	results, err := hc.neo4jClient.ExecuteQuery(ctx, query, map[string]any{
		"paths":     filePaths,
		"threshold": threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j query failed: %w", err)
	}

	partners := make([]CoChangePartnerContext, 0, len(results))

	for _, row := range results {
		partnerFile := row["partner_file"].(string)
		cochangeCount := int(row["cochange_count"].(int64))
		frequency := row["frequency"].(float64)

		// Extract sample commit SHAs
		var sampleSHAs []string
		if sampleRaw, ok := row["sample_commits"].([]interface{}); ok {
			for _, sha := range sampleRaw {
				if s, ok := sha.(string); ok {
					sampleSHAs = append(sampleSHAs, s)
				}
			}
		}

		// Step 2: Get commit messages from PostgreSQL
		var sampleMessages []string
		if len(sampleSHAs) > 0 {
			messages, err := hc.getCommitMessages(ctx, sampleSHAs)
			if err == nil {
				sampleMessages = messages
			}
		}

		partners = append(partners, CoChangePartnerContext{
			PartnerFile:   partnerFile,
			CoChangeCount: cochangeCount,
			Frequency:     frequency,
			SampleCommits: sampleMessages,
		})
	}

	return partners, nil
}

// getCommitMessages retrieves commit messages from PostgreSQL
func (hc *HybridClient) getCommitMessages(ctx context.Context, shas []string) ([]string, error) {
	if len(shas) == 0 {
		return []string{}, nil
	}

	query := `
		SELECT message
		FROM github_commits
		WHERE sha = ANY($1)
		ORDER BY author_date DESC
	`

	rows, err := hc.postgresClient.Query(ctx, query, shas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			continue
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// GetBlastRadiusWithIncidents retrieves downstream dependent files and their incident counts
func (hc *HybridClient) GetBlastRadiusWithIncidents(ctx context.Context, filePath string) ([]BlastRadiusFile, error) {
	// Step 1: Query Neo4j for dependent files
	query := `
		MATCH (f1:File {path: $path})<-[:DEPENDS_ON]-(f2:File)
		WITH f2.path as dependent_file
		OPTIONAL MATCH (issue:Issue)-[:FIXED_BY|ASSOCIATED_WITH]->(pr:PR)<-[:IN_PR]-(c:Commit)-[:MODIFIED]->(f2:File {path: dependent_file})
		WHERE issue.created_at > datetime() - duration({days: 180})
		WITH dependent_file, COUNT(DISTINCT issue) as incident_count, MAX(issue.closed_at) as last_incident
		ORDER BY incident_count DESC
		LIMIT 20
		RETURN dependent_file, incident_count, last_incident
	`

	results, err := hc.neo4jClient.ExecuteQuery(ctx, query, map[string]any{
		"path": filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j query failed: %w", err)
	}

	blastRadius := make([]BlastRadiusFile, 0, len(results))

	for _, row := range results {
		dependentFile := row["dependent_file"].(string)
		incidentCount := 0
		if count, ok := row["incident_count"].(int64); ok {
			incidentCount = int(count)
		}

		var lastIncident *time.Time
		switch v := row["last_incident"].(type) {
		case int64:
			// Neo4j stores as Unix timestamp
			t := time.Unix(v, 0)
			lastIncident = &t
		case string:
			// Try ISO 8601 format
			if v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err == nil {
					lastIncident = &t
				}
			}
		}

		blastRadius = append(blastRadius, BlastRadiusFile{
			FilePath:       dependentFile,
			IncidentCount:  incidentCount,
			LastIncident:   lastIncident,
			DependencyType: "import",
		})
	}

	return blastRadius, nil
}
