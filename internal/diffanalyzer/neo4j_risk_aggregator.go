package diffanalyzer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jRiskAggregator queries Neo4j for all risk dimensions using graph traversals
type Neo4jRiskAggregator struct {
	driver neo4j.DriverWithContext
	logger *log.Logger
}

// NewNeo4jRiskAggregator creates a new Neo4j risk aggregator
func NewNeo4jRiskAggregator(driver neo4j.DriverWithContext, logger *log.Logger) *Neo4jRiskAggregator {
	return &Neo4jRiskAggregator{
		driver: driver,
		logger: logger,
	}
}

// QueryTemporal finds ALL incidents for a function across rename history
// Uses [:RENAMED_FROM*0..5] to traverse rename chains
func (a *Neo4jRiskAggregator) QueryTemporal(ctx context.Context, blockID int64) (*TemporalRisk, error) {
	a.logger.Printf("[Neo4jRiskAggregator] Querying temporal risk for block_id=%d", blockID)

	session := a.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock)-[:RENAMED_FROM*0..5]->(historical:CodeBlock)
		WHERE cb.id = $blockId
		WITH collect(cb) + collect(historical) as all_versions
		UNWIND all_versions as version
		MATCH (issue:Issue)-[:FIXED_BY_BLOCK]->(version)
		RETURN DISTINCT
			issue.number AS number,
			issue.title AS title,
			issue.state AS state,
			issue.created_at AS created_at,
			version.block_name AS version_affected,
			issue.labels AS labels
		ORDER BY issue.created_at DESC
		LIMIT 10
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"blockId": blockID,
	})
	if err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Temporal query failed: %v", err)
		return nil, fmt.Errorf("temporal risk query failed: %w", err)
	}

	var linkedIssues []LinkedIssue
	var affectedVersions []string
	affectedVersionsMap := make(map[string]bool)

	for result.Next(ctx) {
		record := result.Record()

		createdAtEpoch, _ := record.Get("created_at")
		var createdAt time.Time
		if epochInt, ok := createdAtEpoch.(int64); ok {
			createdAt = time.Unix(epochInt, 0)
		}

		versionAffected, _ := record.Get("version_affected")
		versionStr := versionAffected.(string)

		issue := LinkedIssue{
			Number:          int(record.Values[0].(int64)),
			Title:           record.Values[1].(string),
			State:           record.Values[2].(string),
			CreatedAt:       createdAt,
			VersionAffected: versionStr,
		}
		linkedIssues = append(linkedIssues, issue)

		// Track unique affected versions
		if !affectedVersionsMap[versionStr] {
			affectedVersions = append(affectedVersions, versionStr)
			affectedVersionsMap[versionStr] = true
		}
	}

	if err := result.Err(); err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Result iteration failed: %v", err)
		return nil, fmt.Errorf("result iteration failed: %w", err)
	}

	incidentCount := len(linkedIssues)
	var lastIncidentDate *time.Time
	if incidentCount > 0 {
		lastIncidentDate = &linkedIssues[0].CreatedAt
	}

	a.logger.Printf("[Neo4jRiskAggregator] Temporal risk: %d incidents across %d versions",
		incidentCount, len(affectedVersions))

	return &TemporalRisk{
		IncidentCount:    incidentCount,
		LastIncidentDate: lastIncidentDate,
		LinkedIssues:     linkedIssues,
		Summary:          fmt.Sprintf("%d incidents found across %d historical versions", incidentCount, len(affectedVersions)),
		AffectedVersions: affectedVersions,
	}, nil
}

// QueryOwnership finds all developers who edited a function across rename history
func (a *Neo4jRiskAggregator) QueryOwnership(ctx context.Context, blockID int64) (*OwnershipRisk, error) {
	a.logger.Printf("[Neo4jRiskAggregator] Querying ownership risk for block_id=%d", blockID)

	session := a.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock)-[:RENAMED_FROM*0..5]->(historical:CodeBlock)
		WHERE cb.id = $blockId
		WITH collect(cb) + collect(historical) as all_versions
		UNWIND all_versions as version
		MATCH (dev:Developer)-[:AUTHORED]->(commit:Commit)-[:MODIFIED_BLOCK]->(version)
		WITH dev,
		     count(commit) as modification_count,
		     max(commit.committed_at) as last_edit,
		     collect(DISTINCT version.block_name) as versions_touched
		RETURN
			dev.email AS email,
			dev.name AS name,
			modification_count,
			last_edit,
			duration.between(datetime({epochSeconds: last_edit}), datetime()).days AS staleness_days,
			versions_touched
		ORDER BY modification_count DESC
		LIMIT 10
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"blockId": blockID,
	})
	if err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Ownership query failed: %v", err)
		return nil, fmt.Errorf("ownership risk query failed: %w", err)
	}

	var topContributors []DeveloperContribution
	var originalAuthor, lastModifier string
	var daysSinceModified int

	for result.Next(ctx) {
		record := result.Record()

		lastEditEpoch, _ := record.Get("last_edit")
		var lastEdit time.Time
		if epochInt, ok := lastEditEpoch.(int64); ok {
			lastEdit = time.Unix(epochInt, 0)
		}

		stalenessDaysVal, _ := record.Get("staleness_days")
		var stalenessDays int
		if staleInt, ok := stalenessDaysVal.(int64); ok {
			stalenessDays = int(staleInt)
		}

		versionsTouchedVal, _ := record.Get("versions_touched")
		var versionsTouched []string
		if versions, ok := versionsTouchedVal.([]interface{}); ok {
			for _, v := range versions {
				if vStr, ok := v.(string); ok {
					versionsTouched = append(versionsTouched, vStr)
				}
			}
		}

		contrib := DeveloperContribution{
			Email:             record.Values[0].(string),
			Name:              record.Values[1].(string),
			ModificationCount: int(record.Values[2].(int64)),
			LastEdit:          lastEdit,
			VersionsTouched:   versionsTouched,
		}
		topContributors = append(topContributors, contrib)

		// First contributor is the most active (after sorting by mod count)
		if len(topContributors) == 1 {
			lastModifier = contrib.Email
			daysSinceModified = stalenessDays
		}
		// Last contributor in the list is the original author (least modifications, earliest)
		originalAuthor = contrib.Email
	}

	if err := result.Err(); err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Result iteration failed: %v", err)
		return nil, fmt.Errorf("result iteration failed: %w", err)
	}

	// Determine status based on staleness
	status := "ACTIVE"
	if daysSinceModified > 365 {
		status = "STALE"
	}

	// Bus factor warning if only 1-2 contributors
	busFactorWarning := len(topContributors) <= 2

	a.logger.Printf("[Neo4jRiskAggregator] Ownership risk: %d contributors, staleness=%d days, status=%s",
		len(topContributors), daysSinceModified, status)

	return &OwnershipRisk{
		Status:            status,
		OriginalAuthor:    originalAuthor,
		LastModifier:      lastModifier,
		DaysSinceModified: daysSinceModified,
		BusFactorWarning:  busFactorWarning,
		TopContributors:   topContributors,
	}, nil
}

// QueryCoupling finds blocks that co-change with the target function
// Uses dynamic coupling score calculation with recency multiplier
func (a *Neo4jRiskAggregator) QueryCoupling(ctx context.Context, blockID int64) (*CouplingRisk, error) {
	a.logger.Printf("[Neo4jRiskAggregator] Querying coupling risk for block_id=%d", blockID)

	session := a.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock)-[r:CO_CHANGES_WITH]-(coupled:CodeBlock)
		WHERE cb.id = $blockId
		  AND r.coupling_rate >= 0.75
		  AND coupled.incident_count >= 1
		WITH cb, coupled, r,
		     r.co_change_percentage *
		     (cb.incident_count + coupled.incident_count) *
		     CASE
		         WHEN duration.between(datetime({epochSeconds: r.last_co_change}), datetime()).days < 90 THEN 1.5
		         WHEN duration.between(datetime({epochSeconds: r.last_co_change}), datetime()).days < 180 THEN 1.0
		         ELSE 0.5
		     END as coupling_score
		RETURN
			coupled.block_name AS name,
			coupled.canonical_file_path AS file,
			r.co_change_count AS co_change_count,
			r.coupling_rate AS coupling_rate,
			coupled.incident_count AS incident_count,
			coupling_score
		ORDER BY coupling_score DESC
		LIMIT 3
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"blockId": blockID,
	})
	if err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Coupling query failed: %v", err)
		return nil, fmt.Errorf("coupling risk query failed: %w", err)
	}

	var coupledBlocks []CoupledBlock
	var totalScore float64

	for result.Next(ctx) {
		record := result.Record()

		couplingScore, _ := record.Get("coupling_score")
		var score float64
		if scoreFloat, ok := couplingScore.(float64); ok {
			score = scoreFloat
		}

		block := CoupledBlock{
			Name:                 record.Values[0].(string),
			File:                 record.Values[1].(string),
			CoChangeCount:        int(record.Values[2].(int64)),
			CouplingRate:         record.Values[3].(float64),
			IncidentCount:        int(record.Values[4].(int64)),
			DynamicCouplingScore: score,
		}
		coupledBlocks = append(coupledBlocks, block)
		totalScore += score
	}

	if err := result.Err(); err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Result iteration failed: %v", err)
		return nil, fmt.Errorf("result iteration failed: %w", err)
	}

	a.logger.Printf("[Neo4jRiskAggregator] Coupling risk: %d coupled blocks, total_score=%.2f",
		len(coupledBlocks), totalScore)

	return &CouplingRisk{
		Score:         totalScore,
		CoupledBlocks: coupledBlocks,
	}, nil
}

// QueryHistory finds recent modifications to a function across rename history
func (a *Neo4jRiskAggregator) QueryHistory(ctx context.Context, blockID int64) (*ChangeHistory, error) {
	a.logger.Printf("[Neo4jRiskAggregator] Querying change history for block_id=%d", blockID)

	session := a.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock)-[:RENAMED_FROM*0..5]->(historical:CodeBlock)
		WHERE cb.id = $blockId
		WITH collect(cb) + collect(historical) as all_versions
		UNWIND all_versions as version
		MATCH (dev:Developer)-[:AUTHORED]->(commit:Commit)-[mod:MODIFIED_BLOCK]->(version)
		RETURN
			commit.sha AS sha,
			commit.message AS message,
			commit.committed_at AS committed_at,
			dev.email AS author,
			mod.change_type AS change_type,
			mod.lines_added AS lines_added,
			mod.lines_deleted AS lines_deleted,
			mod.change_summary AS summary,
			version.block_name AS version_name
		ORDER BY commit.committed_at DESC
		LIMIT 20
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"blockId": blockID,
	})
	if err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: History query failed: %v", err)
		return nil, fmt.Errorf("change history query failed: %w", err)
	}

	var recentChanges []BlockChange

	for result.Next(ctx) {
		record := result.Record()

		committedAtEpoch, _ := record.Get("committed_at")
		var committedAt time.Time
		if epochInt, ok := committedAtEpoch.(int64); ok {
			committedAt = time.Unix(epochInt, 0)
		}

		change := BlockChange{
			SHA:          record.Values[0].(string),
			Message:      record.Values[1].(string),
			CommittedAt:  committedAt,
			Author:       record.Values[3].(string),
			ChangeType:   record.Values[4].(string),
			LinesAdded:   int(record.Values[5].(int64)),
			LinesDeleted: int(record.Values[6].(int64)),
			Summary:      record.Values[7].(string),
			VersionName:  record.Values[8].(string),
		}
		recentChanges = append(recentChanges, change)
	}

	if err := result.Err(); err != nil {
		a.logger.Printf("[Neo4jRiskAggregator] ERROR: Result iteration failed: %v", err)
		return nil, fmt.Errorf("result iteration failed: %w", err)
	}

	a.logger.Printf("[Neo4jRiskAggregator] Change history: %d recent modifications found", len(recentChanges))

	return &ChangeHistory{
		RecentChanges: recentChanges,
	}, nil
}
