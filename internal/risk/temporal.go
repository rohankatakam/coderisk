package risk

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// TemporalCalculator links incidents (Issues) to CodeBlocks and calculates R_temporal properties
// Reference: AGENT-P3C Temporal Risk Calculator
// Implements R_temporal calculation: incident counts and temporal summaries
type TemporalCalculator struct {
	db      *sql.DB
	neo4j   *graph.Neo4jBackend
	llm     *llm.Client
	repoID  int64
}

// NewTemporalCalculator creates a new temporal calculator instance
func NewTemporalCalculator(db *sql.DB, neo4j *graph.Neo4jBackend, llmClient *llm.Client, repoID int64) *TemporalCalculator {
	return &TemporalCalculator{
		db:     db,
		neo4j:  neo4j,
		llm:    llmClient,
		repoID: repoID,
	}
}

// IssueInfo represents an issue with its metadata
type IssueInfo struct {
	ID        int64
	GitHubID  int64
	Number    int
	Title     string
	Body      string
	State     string
	CreatedAt time.Time
	ClosedAt  sql.NullTime
}

// CodeBlockIncident represents a link between a code block and an issue
type CodeBlockIncident struct {
	CodeBlockID    int64
	IssueID        int64
	Confidence     float64
	EvidenceSource string
	EvidenceText   string
	FixCommitSHA   string
	FixedAt        sql.NullTime
}

// LinkIssuesViaCommits links issues to code blocks through commits that fixed them
// Reference: AGENT-P3C §1 - Link Issues to CodeBlocks
// Strategy: Find issues that were CLOSED_BY or REFERENCED by commits that MODIFIED_BLOCK specific blocks
func (t *TemporalCalculator) LinkIssuesViaCommits(ctx context.Context) (int, error) {
	// Query to find issues linked to commits that modified code blocks
	// We use the github_issue_timeline table which tracks both closed and referenced events
	query := `
		WITH issue_related_commits AS (
			-- Find commits that closed OR referenced issues via timeline events
			SELECT DISTINCT
				git.issue_id,
				git.source_sha AS commit_sha,
				git.created_at AS event_at,
				git.event_type
			FROM github_issue_timeline git
			JOIN github_issues gi ON gi.id = git.issue_id
			WHERE git.event_type IN ('closed', 'referenced')
			  AND git.source_sha IS NOT NULL
			  AND gi.repo_id = $1
		),
		issue_fix_blocks AS (
			-- Join with code block modifications to find which blocks were modified by related commits
			SELECT DISTINCT
				irc.issue_id,
				cbm.code_block_id,
				irc.commit_sha AS fix_commit_sha,
				irc.event_at AS fixed_at,
				irc.event_type
			FROM issue_related_commits irc
			JOIN code_block_modifications cbm ON cbm.commit_sha = irc.commit_sha
			JOIN code_blocks cb ON cb.id = cbm.code_block_id
			WHERE cb.repo_id = $1
		)
		SELECT
			ifb.code_block_id,
			ifb.issue_id,
			ifb.fix_commit_sha,
			ifb.fixed_at,
			ifb.event_type,
			gi.number,
			gi.title
		FROM issue_fix_blocks ifb
		JOIN github_issues gi ON gi.id = ifb.issue_id
		WHERE gi.repo_id = $1
		ORDER BY ifb.fixed_at DESC
	`

	rows, err := t.db.QueryContext(ctx, query, t.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to query issue-commit-block links: %w", err)
	}
	defer rows.Close()

	linksCreated := 0

	for rows.Next() {
		var codeBlockID, issueID int64
		var fixCommitSHA string
		var fixedAt sql.NullTime
		var eventType string
		var issueNumber int
		var issueTitle string

		if err := rows.Scan(&codeBlockID, &issueID, &fixCommitSHA, &fixedAt, &eventType, &issueNumber, &issueTitle); err != nil {
			return linksCreated, fmt.Errorf("failed to scan row: %w", err)
		}

		// Fetch issue metadata for new schema columns
		var issueCreatedAt time.Time
		var issueClosedAt sql.NullTime
		var issueLabels []string
		issueQuery := `
			SELECT created_at, closed_at, labels
			FROM github_issues
			WHERE id = $1
		`
		err = t.db.QueryRowContext(ctx, issueQuery, issueID).Scan(&issueCreatedAt, &issueClosedAt, &issueLabels)
		if err != nil {
			return linksCreated, fmt.Errorf("failed to fetch issue metadata for issue_id=%d: %w", issueID, err)
		}

		// Determine incident_type from labels
		incidentType := "unknown"
		if len(issueLabels) > 0 {
			// Prioritize critical labels
			for _, label := range issueLabels {
				labelLower := strings.ToLower(label)
				if strings.Contains(labelLower, "bug") {
					incidentType = "bug"
					break
				} else if strings.Contains(labelLower, "security") {
					incidentType = "security"
					break
				} else if strings.Contains(labelLower, "critical") {
					incidentType = "critical"
					break
				}
			}
			// If no priority label found, use first label
			if incidentType == "unknown" && len(issueLabels) > 0 {
				incidentType = issueLabels[0]
			}
		}

		// Insert into code_block_incidents table using NEW schema
		insertQuery := `
			INSERT INTO code_block_incidents (
				repo_id, block_id, issue_id,
				confidence, evidence_source, evidence_text,
				commit_sha, incident_date, resolution_date, incident_type,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
			ON CONFLICT (block_id, issue_id)
			DO UPDATE SET
				commit_sha = EXCLUDED.commit_sha,
				incident_date = EXCLUDED.incident_date,
				resolution_date = EXCLUDED.resolution_date,
				incident_type = EXCLUDED.incident_type,
				confidence = EXCLUDED.confidence,
				evidence_source = EXCLUDED.evidence_source,
				evidence_text = EXCLUDED.evidence_text
		`

		// Create evidence text based on event type
		var evidenceText string
		if eventType == "closed" {
			evidenceText = fmt.Sprintf("Issue #%d '%s' was closed by commit %s which modified this code block",
				issueNumber, issueTitle, fixCommitSHA[:7])
		} else {
			evidenceText = fmt.Sprintf("Issue #%d '%s' was referenced by commit %s which modified this code block",
				issueNumber, issueTitle, fixCommitSHA[:7])
		}

		// Use issue closed_at as resolution_date, or fixed_at as fallback
		var resolutionDate interface{}
		if issueClosedAt.Valid {
			resolutionDate = issueClosedAt.Time
		} else if fixedAt.Valid {
			resolutionDate = fixedAt.Time
		} else {
			resolutionDate = nil
		}

		_, err = t.db.ExecContext(ctx, insertQuery,
			t.repoID, codeBlockID, issueID,
			0.80, // Confidence: 0.80 for timeline-based linking (referenced events are 100% reliable)
			"timeline_event",
			evidenceText,
			fixCommitSHA,
			issueCreatedAt,    // incident_date: when issue was created
			resolutionDate,    // resolution_date: when issue was closed
			incidentType,      // incident_type: derived from labels
		)
		if err != nil {
			return linksCreated, fmt.Errorf("failed to insert incident link: %w", err)
		}

		linksCreated++
	}

	if err := rows.Err(); err != nil {
		return linksCreated, fmt.Errorf("error iterating rows: %w", err)
	}

	return linksCreated, nil
}

// CalculateIncidentCounts updates the incident_count field for all code blocks
// Reference: AGENT-P3C §2 - Count Incidents Per Block
func (t *TemporalCalculator) CalculateIncidentCounts(ctx context.Context) (int, error) {
	// Update incident counts for blocks with incidents
	updateQuery := `
		UPDATE code_blocks cb
		SET incident_count = (
			SELECT COUNT(*)
			FROM code_block_incidents cbi
			WHERE cbi.code_block_id = cb.id
		)
		WHERE cb.repo_id = $1
		  AND EXISTS (
			SELECT 1
			FROM code_block_incidents cbi
			WHERE cbi.code_block_id = cb.id
		  )
	`

	result, err := t.db.ExecContext(ctx, updateQuery, t.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to update incident counts: %w", err)
	}

	blocksUpdated, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Set zero incidents for blocks without incidents
	zeroQuery := `
		UPDATE code_blocks
		SET incident_count = 0
		WHERE repo_id = $1
		  AND incident_count IS NULL
		  AND NOT EXISTS (
			SELECT 1
			FROM code_block_incidents cbi
			WHERE cbi.code_block_id = code_blocks.id
		  )
	`

	zeroResult, err := t.db.ExecContext(ctx, zeroQuery, t.repoID)
	if err != nil {
		return int(blocksUpdated), fmt.Errorf("failed to set zero incident counts: %w", err)
	}

	zeroBlocks, err := zeroResult.RowsAffected()
	if err != nil {
		return int(blocksUpdated), fmt.Errorf("failed to get zero rows affected: %w", err)
	}

	return int(blocksUpdated + zeroBlocks), nil
}

// GenerateTemporalSummaries creates LLM-generated summaries for blocks with incidents
// Reference: AGENT-P3C §4 - LLM Temporal Summary
func (t *TemporalCalculator) GenerateTemporalSummaries(ctx context.Context) (int, error) {
	if t.llm == nil || !t.llm.IsEnabled() {
		return 0, fmt.Errorf("llm client not enabled")
	}

	// Find blocks with incidents that don't have summaries yet
	query := `
		SELECT DISTINCT
			cb.id,
			cb.block_name,
			cb.file_path,
			cb.block_type,
			cb.incident_count
		FROM code_blocks cb
		WHERE cb.repo_id = $1
		  AND cb.incident_count > 0
		ORDER BY cb.incident_count DESC
		LIMIT 50  -- Process top 50 high-incident blocks
	`

	rows, err := t.db.QueryContext(ctx, query, t.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to query blocks for summaries: %w", err)
	}
	defer rows.Close()

	summariesGenerated := 0

	for rows.Next() {
		var blockID int64
		var blockName, filePath, blockType string
		var incidentCount int

		if err := rows.Scan(&blockID, &blockName, &filePath, &blockType, &incidentCount); err != nil {
			return summariesGenerated, fmt.Errorf("failed to scan block row: %w", err)
		}

		// Get issues linked to this block
		issuesQuery := `
			SELECT
				gi.number,
				gi.title,
				gi.body,
				gi.created_at,
				gi.closed_at
			FROM code_block_incidents cbi
			JOIN github_issues gi ON gi.id = cbi.issue_id
			WHERE cbi.code_block_id = $1
			ORDER BY gi.created_at DESC
		`

		issueRows, err := t.db.QueryContext(ctx, issuesQuery, blockID)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch issues for block %d: %v\n", blockID, err)
			continue
		}

		var issues []IssueInfo
		for issueRows.Next() {
			var issue IssueInfo
			var body sql.NullString
			if err := issueRows.Scan(&issue.Number, &issue.Title, &body, &issue.CreatedAt, &issue.ClosedAt); err != nil {
				issueRows.Close()
				return summariesGenerated, fmt.Errorf("failed to scan issue row: %w", err)
			}
			if body.Valid {
				issue.Body = body.String
			}
			issues = append(issues, issue)
		}
		issueRows.Close()

		if len(issues) == 0 {
			continue
		}

		// Generate temporal summary using LLM
		summary, err := t.generateSummaryForBlock(ctx, blockName, filePath, blockType, issues)
		if err != nil {
			fmt.Printf("Warning: Failed to generate summary for block %d: %v\n", blockID, err)
			continue
		}

		// Store summary in a new field (we'll need to add this to the schema if it doesn't exist)
		// For now, we'll just print it
		fmt.Printf("Temporal summary for %s (%s): %s\n", blockName, filePath, summary)
		summariesGenerated++
	}

	if err := rows.Err(); err != nil {
		return summariesGenerated, fmt.Errorf("error iterating block rows: %w", err)
	}

	return summariesGenerated, nil
}

// generateSummaryForBlock generates an LLM summary of incident patterns
func (t *TemporalCalculator) generateSummaryForBlock(ctx context.Context, blockName, filePath, blockType string, issues []IssueInfo) (string, error) {
	issueTitles := make([]string, len(issues))
	for i, issue := range issues {
		issueTitles[i] = fmt.Sprintf("#%d: %s", issue.Number, issue.Title)
	}

	systemPrompt := "You are a code analysis expert. Summarize incident patterns to help developers understand historical issues."

	userPrompt := fmt.Sprintf(`These issues were linked to the %s "%s" in file "%s":

%s

Summarize the incident pattern in 1-2 sentences. Focus on:
- What types of issues occurred (bugs, performance, features)?
- Any common themes or patterns?
- The overall reliability trend

Keep it concise and actionable.`,
		blockType,
		blockName,
		filePath,
		strings.Join(issueTitles, "\n"))

	response, err := t.llm.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("llm completion failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// CreateNeo4jEdges creates WAS_ROOT_CAUSE_IN edges in Neo4j for graph queries
// Reference: AGENT-P3C §1 - Neo4j Edge Creation
// Note: This is optional for MVP but useful for fast graph traversal
func (t *TemporalCalculator) CreateNeo4jEdges(ctx context.Context) (int, error) {
	if t.neo4j == nil {
		return 0, fmt.Errorf("neo4j backend not configured")
	}

	// For MVP, we'll skip Neo4j edge creation since the primary storage is PostgreSQL
	// In production, we would create WAS_ROOT_CAUSE_IN edges in Neo4j for fast graph queries
	// This would involve querying code_block_incidents table and creating corresponding edges

	return 0, nil
}

// GetIncidentStatistics returns summary statistics about incidents
func (t *TemporalCalculator) GetIncidentStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(DISTINCT cbi.code_block_id) AS blocks_with_incidents,
			COUNT(DISTINCT cbi.issue_id) AS total_unique_issues,
			COUNT(*) AS total_incident_links,
			AVG(cb.incident_count) AS avg_incidents_per_block,
			MAX(cb.incident_count) AS max_incidents_per_block
		FROM code_block_incidents cbi
		JOIN code_blocks cb ON cb.id = cbi.code_block_id
		WHERE cbi.repo_id = $1
	`

	var blocksWithIncidents, totalIssues, totalLinks int
	var avgIncidents, maxIncidents sql.NullFloat64

	err := t.db.QueryRowContext(ctx, query, t.repoID).Scan(
		&blocksWithIncidents,
		&totalIssues,
		&totalLinks,
		&avgIncidents,
		&maxIncidents,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query statistics: %w", err)
	}

	// Get total blocks count
	var totalBlocks int
	countQuery := `SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1`
	if err := t.db.QueryRowContext(ctx, countQuery, t.repoID).Scan(&totalBlocks); err != nil {
		return nil, fmt.Errorf("failed to query total blocks: %w", err)
	}

	stats := map[string]interface{}{
		"total_blocks":           totalBlocks,
		"blocks_with_incidents":  blocksWithIncidents,
		"blocks_without_incidents": totalBlocks - blocksWithIncidents,
		"total_unique_issues":    totalIssues,
		"total_incident_links":   totalLinks,
		"avg_incidents_per_block": avgIncidents.Float64,
		"max_incidents_per_block": maxIncidents.Float64,
	}

	return stats, nil
}

// GetTopIncidentBlocks returns the code blocks with the most incidents
func (t *TemporalCalculator) GetTopIncidentBlocks(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			cb.id,
			cb.file_path,
			cb.block_name,
			cb.block_type,
			cb.incident_count,
			cb.updated_at
		FROM code_blocks cb
		WHERE cb.repo_id = $1
		  AND cb.incident_count > 0
		ORDER BY cb.incident_count DESC, cb.updated_at DESC
		LIMIT $2
	`

	rows, err := t.db.QueryContext(ctx, query, t.repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top incident blocks: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var filePath, blockName, blockType string
		var incidentCount int
		var updatedAt sql.NullTime

		if err := rows.Scan(&id, &filePath, &blockName, &blockType, &incidentCount, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result := map[string]interface{}{
			"id":             id,
			"file_path":      filePath,
			"block_name":     blockName,
			"block_type":     blockType,
			"incident_count": incidentCount,
		}

		if updatedAt.Valid {
			result["updated_at"] = updatedAt.Time
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// GetBlockIncidents returns all incidents for a specific code block
func (t *TemporalCalculator) GetBlockIncidents(ctx context.Context, blockID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT
			gi.number,
			gi.title,
			gi.state,
			gi.created_at,
			gi.closed_at,
			cbi.confidence,
			cbi.evidence_source,
			cbi.evidence_text,
			cbi.fix_commit_sha,
			cbi.fixed_at
		FROM code_block_incidents cbi
		JOIN github_issues gi ON gi.id = cbi.issue_id
		WHERE cbi.code_block_id = $1
		ORDER BY gi.created_at DESC
	`

	rows, err := t.db.QueryContext(ctx, query, blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to query block incidents: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var number int
		var title, state, evidenceSource, evidenceText, fixCommitSHA string
		var createdAt time.Time
		var closedAt, fixedAt sql.NullTime
		var confidence float64

		if err := rows.Scan(&number, &title, &state, &createdAt, &closedAt, &confidence, &evidenceSource, &evidenceText, &fixCommitSHA, &fixedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result := map[string]interface{}{
			"number":          number,
			"title":           title,
			"state":           state,
			"created_at":      createdAt,
			"confidence":      confidence,
			"evidence_source": evidenceSource,
			"evidence_text":   evidenceText,
			"fix_commit_sha":  fixCommitSHA,
		}

		if closedAt.Valid {
			result["closed_at"] = closedAt.Time
		}
		if fixedAt.Valid {
			result["fixed_at"] = fixedAt.Time
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}
