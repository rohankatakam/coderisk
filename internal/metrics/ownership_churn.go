package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/graph"
)

// OwnershipChurnResult represents the ownership churn metric result (Tier 2)
// Reference: risk_assessment_methodology.md §3.1 - Ownership Churn
// Reference: PHASE2_INVESTIGATION_ROADMAP.md Task 1 - MODIFIES edge analysis
type OwnershipChurnResult struct {
	FilePath            string    `json:"file_path"`
	ModifyCount         int       `json:"modify_count"`          // Total commits in window
	WindowDays          int       `json:"window_days"`           // Analysis window (90 days)
	CurrentOwner        string    `json:"current_owner"`         // Most commits in last 30 days
	PreviousOwner       string    `json:"previous_owner"`        // Most commits in days 31-90
	DaysSinceTransition int       `json:"days_since_transition"` // Days since ownership changed
	RiskLevel           RiskLevel `json:"risk_level"`            // LOW, MEDIUM, HIGH
	Developers          []string  `json:"developers"`            // All developers who modified
}

// DeveloperCommitCount tracks commits per developer
type DeveloperCommitCount struct {
	Email       string
	CommitCount int
}

// CalculateOwnershipChurn computes ownership churn for a file (Tier 2 metric)
// Reference: risk_assessment_methodology.md §3.1
// Formula: Queries MODIFIES edges for last 90 days, identifies ownership transitions
// 12-factor: Factor 8 - Own your control flow (LLM-requested metric, on-demand only)
func CalculateOwnershipChurn(ctx context.Context, neo4j *graph.Client, redis *cache.Client, repoID, filePath string) (*OwnershipChurnResult, error) {
	// Try cache first (15-min TTL per risk_assessment_methodology.md §3.1)
	cacheKey := cache.CacheKey("ownership_churn", repoID, filePath)
	var cached OwnershipChurnResult
	hit, err := redis.Get(ctx, cacheKey, &cached)
	if err != nil {
		fmt.Printf("cache error (non-fatal): %v\n", err)
	} else if hit {
		return &cached, nil
	}

	// Query Neo4j for MODIFIES edges in last 90 days
	// Cypher: MATCH (f:File {file_path: $path})<-[:MODIFIES]-(c:Commit)-[:AUTHORED]->(d:Developer)
	//         WHERE c.author_date > timestamp() - duration({days: 90})
	//         RETURN d.email, count(c) as commit_count
	windowDays := 90
	developers, err := queryModifiesEdges(ctx, neo4j, filePath, windowDays)
	if err != nil {
		return nil, fmt.Errorf("failed to query MODIFIES edges for %s: %w", filePath, err)
	}

	// If no commits found, return LOW risk
	if len(developers) == 0 {
		result := &OwnershipChurnResult{
			FilePath:    filePath,
			ModifyCount: 0,
			WindowDays:  windowDays,
			RiskLevel:   RiskLevelLow,
		}
		// Cache result
		if err := redis.Set(ctx, cacheKey, result); err != nil {
			fmt.Printf("failed to cache ownership_churn result: %v\n", err)
		}
		return result, nil
	}

	// Calculate total modify count
	modifyCount := 0
	developerList := []string{}
	for _, dev := range developers {
		modifyCount += dev.CommitCount
		developerList = append(developerList, dev.Email)
	}

	// Identify current owner (most commits in last 30 days)
	// Identify previous owner (most commits in days 31-90)
	currentOwner, previousOwner := identifyOwners(ctx, neo4j, filePath, developers)

	// Calculate days since ownership transition
	daysSinceTransition := calculateDaysSinceTransition(ctx, neo4j, filePath, currentOwner)

	// Determine risk level using thresholds from risk_assessment_methodology.md §3.1
	// modify_count > 10: HIGH churn
	// 5 < modify_count <= 10: MEDIUM churn
	// modify_count <= 5: LOW churn
	// days_since_transition < 30: Elevate risk (ownership instability)
	riskLevel := classifyChurnRisk(modifyCount, daysSinceTransition)

	result := &OwnershipChurnResult{
		FilePath:            filePath,
		ModifyCount:         modifyCount,
		WindowDays:          windowDays,
		CurrentOwner:        currentOwner,
		PreviousOwner:       previousOwner,
		DaysSinceTransition: daysSinceTransition,
		RiskLevel:           riskLevel,
		Developers:          developerList,
	}

	// Cache result (15-min TTL)
	if err := redis.Set(ctx, cacheKey, result); err != nil {
		fmt.Printf("failed to cache ownership_churn result: %v\n", err)
	}

	return result, nil
}

// queryModifiesEdges queries Neo4j for all developers who modified the file
func queryModifiesEdges(ctx context.Context, neo4j *graph.Client, filePath string, windowDays int) ([]DeveloperCommitCount, error) {
	// Cypher query to get all MODIFIES edges for the file
	query := `
		MATCH (c:Commit)-[:MODIFIES]->(f:File {file_path: $file_path})
		WHERE c.author_date > timestamp() - duration({days: $window_days}) * 1000
		MATCH (c)<-[:AUTHORED]-(d:Developer)
		RETURN d.email as email, count(c) as commit_count
		ORDER BY commit_count DESC
	`

	params := map[string]interface{}{
		"file_path":   filePath,
		"window_days": windowDays,
	}

	records, err := neo4j.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("cypher query failed: %w", err)
	}

	// Parse result
	var developers []DeveloperCommitCount
	for _, record := range records {
		email, _ := record["email"].(string)
		commitCount, _ := record["commit_count"].(int64)
		developers = append(developers, DeveloperCommitCount{
			Email:       email,
			CommitCount: int(commitCount),
		})
	}

	return developers, nil
}

// identifyOwners identifies current and previous file owners
// Current owner: most commits in last 30 days
// Previous owner: most commits in days 31-90
func identifyOwners(ctx context.Context, neo4j *graph.Client, filePath string, allDevelopers []DeveloperCommitCount) (string, string) {
	// Query for last 30 days (current owner)
	currentQuery := `
		MATCH (c:Commit)-[:MODIFIES]->(f:File {file_path: $file_path})
		WHERE c.author_date > timestamp() - duration({days: 30}) * 1000
		MATCH (c)<-[:AUTHORED]-(d:Developer)
		RETURN d.email as email, count(c) as commit_count
		ORDER BY commit_count DESC
		LIMIT 1
	`

	currentParams := map[string]interface{}{
		"file_path": filePath,
	}

	currentRecords, err := neo4j.ExecuteQuery(ctx, currentQuery, currentParams)
	currentOwner := ""
	if err == nil && len(currentRecords) > 0 {
		currentOwner, _ = currentRecords[0]["email"].(string)
	}

	// Query for days 31-90 (previous owner)
	previousQuery := `
		MATCH (c:Commit)-[:MODIFIES]->(f:File {file_path: $file_path})
		WHERE c.author_date > timestamp() - duration({days: 90}) * 1000
		  AND c.author_date <= timestamp() - duration({days: 30}) * 1000
		MATCH (c)<-[:AUTHORED]-(d:Developer)
		WHERE d.email <> $current_owner
		RETURN d.email as email, count(c) as commit_count
		ORDER BY commit_count DESC
		LIMIT 1
	`

	previousParams := map[string]interface{}{
		"file_path":     filePath,
		"current_owner": currentOwner,
	}

	previousRecords, err := neo4j.ExecuteQuery(ctx, previousQuery, previousParams)
	previousOwner := ""
	if err == nil && len(previousRecords) > 0 {
		previousOwner, _ = previousRecords[0]["email"].(string)
	}

	// Fallback: if no current/previous owner found, use top 2 from all developers
	if currentOwner == "" && len(allDevelopers) > 0 {
		currentOwner = allDevelopers[0].Email
	}
	if previousOwner == "" && len(allDevelopers) > 1 {
		previousOwner = allDevelopers[1].Email
	}

	return currentOwner, previousOwner
}

// calculateDaysSinceTransition calculates days since ownership changed
func calculateDaysSinceTransition(ctx context.Context, neo4j *graph.Client, filePath string, currentOwner string) int {
	if currentOwner == "" {
		return -1 // Unknown
	}

	// Find the most recent commit by the current owner
	query := `
		MATCH (c:Commit)-[:MODIFIES]->(f:File {file_path: $file_path})
		MATCH (c)<-[:AUTHORED]-(d:Developer {email: $current_owner})
		RETURN c.author_date as latest_commit
		ORDER BY c.author_date DESC
		LIMIT 1
	`

	params := map[string]interface{}{
		"file_path":     filePath,
		"current_owner": currentOwner,
	}

	records, err := neo4j.ExecuteQuery(ctx, query, params)
	if err != nil {
		return -1
	}

	if len(records) > 0 {
		if latestCommitUnix, ok := records[0]["latest_commit"].(int64); ok {
			latestCommit := time.Unix(latestCommitUnix, 0)
			daysSince := int(time.Since(latestCommit).Hours() / 24)
			return daysSince
		}
	}

	return -1
}

// classifyChurnRisk applies threshold logic from risk_assessment_methodology.md §3.1
func classifyChurnRisk(modifyCount int, daysSinceTransition int) RiskLevel {
	// High churn: > 10 commits in 90 days
	if modifyCount > 10 {
		return RiskLevelHigh
	}

	// Medium churn: 5-10 commits
	if modifyCount > 5 {
		// Elevate to HIGH if ownership transition is recent (< 30 days)
		if daysSinceTransition >= 0 && daysSinceTransition < 30 {
			return RiskLevelHigh
		}
		return RiskLevelMedium
	}

	// Low churn: <= 5 commits
	return RiskLevelLow
}

// FormatEvidence generates human-readable evidence string
// Reference: risk_assessment_methodology.md §3.1 - Evidence format
func (o *OwnershipChurnResult) FormatEvidence() string {
	if o.ModifyCount == 0 {
		return "No commits in last 90 days (stable file)"
	}

	evidence := fmt.Sprintf("File modified %d times in last %d days (%s churn)",
		o.ModifyCount, o.WindowDays, o.RiskLevel)

	if o.CurrentOwner != "" {
		evidence += fmt.Sprintf(" | Primary owner: %s", o.CurrentOwner)
	}

	if o.PreviousOwner != "" && o.PreviousOwner != o.CurrentOwner {
		evidence += fmt.Sprintf(" | Previous owner: %s", o.PreviousOwner)
		if o.DaysSinceTransition >= 0 {
			evidence += fmt.Sprintf(" (transitioned %d days ago)", o.DaysSinceTransition)
		}
	}

	return evidence
}

// ShouldEscalate returns true if this metric triggers Phase 2
// Reference: risk_assessment_methodology.md §3.1 - Escalation logic
func (o *OwnershipChurnResult) ShouldEscalate() bool {
	// Escalate if HIGH churn (> 10 commits)
	// or if ownership transition is recent (< 30 days)
	return o.ModifyCount > 10 || (o.DaysSinceTransition >= 0 && o.DaysSinceTransition < 30)
}
