package risk

// Fixed Query Library (FR-7)
// This file contains the 7 core Cypher queries used in Phase 1 data collection
// Reference: mvp_development_plan.md FR-7

const (
	// Query 1: Blast Radius (Structural Coupling)
	// Returns the number of files that depend on or are depended upon by the target file
	QueryBlastRadius = `
		MATCH (target:File {path: $filePath, repo_id: $repoID})
		OPTIONAL MATCH (target)-[:IMPORTS|CALLS|DEPENDS_ON]-(dependent:File)
		WITH target, count(DISTINCT dependent) as coupling_count, collect(DISTINCT dependent.path) as dependent_files
		RETURN coupling_count, dependent_files
	`

	// Query 2: Co-Change Patterns
	// Returns files that frequently change together with the target file
	QueryCoChangePartners = `
		MATCH (target:File {path: $filePath, repo_id: $repoID})
		MATCH (target)<-[:MODIFIES]-(c:Commit)-[:MODIFIES]->(partner:File)
		WHERE partner.path <> target.path
		WITH partner, count(DISTINCT c) as co_change_count
		WHERE co_change_count > 2
		MATCH (partner)<-[:MODIFIES]-(latest:Commit)
		WITH partner, co_change_count, max(latest.timestamp) as last_co_change
		RETURN partner.path as file_path, co_change_count, last_co_change
		ORDER BY co_change_count DESC
		LIMIT 10
	`

	// Query 3: Ownership and Churn
	// Returns the file owner and commit activity metrics
	QueryOwnershipChurn = `
		MATCH (f:File {path: $filePath, repo_id: $repoID})
		OPTIONAL MATCH (f)<-[:MODIFIES]-(c:Commit)
		WITH f, c, c.author as author, c.author_email as email, c.timestamp as timestamp
		ORDER BY timestamp DESC
		WITH f, collect({author: author, email: email, timestamp: timestamp}) as commits
		WITH f, commits,
			 commits[0].author as last_modifier,
			 commits[0].timestamp as last_modified,
			 size(commits) as commit_count
		// Calculate ownership (most frequent contributor)
		UNWIND commits as commit
		WITH f, commit.author as author, commit.email as email, count(*) as author_commits,
			 last_modifier, last_modified, commit_count
		ORDER BY author_commits DESC
		LIMIT 1
		RETURN author as file_owner, email as owner_email, author_commits,
			   last_modifier, last_modified, commit_count
	`

	// Query 4: Test Coverage
	// Returns test files and test ratio for the target file
	QueryTestCoverage = `
		MATCH (target:File {path: $filePath, repo_id: $repoID})
		OPTIONAL MATCH (target)<-[:TESTS]-(test:File)
		WITH target, collect(test.path) as test_files, count(test) as test_count
		RETURN test_files, test_count, 
			   CASE WHEN test_count > 0 THEN 1.0 ELSE 0.0 END as test_ratio
	`

	// Query 5: Incident History
	// Returns historical incidents linked to the target file
	QueryIncidentHistory = `
		MATCH (f:File {path: $filePath, repo_id: $repoID})
		OPTIONAL MATCH (f)-[:RELATED_TO]->(incident:Issue)
		WHERE incident.is_incident = true
		WITH f, collect(incident.title) as incidents, count(incident) as incident_count
		RETURN incidents, incident_count
	`

	// Query 6: Change Complexity
	// Note: This query cannot be purely Cypher-based as it requires git diff analysis
	// This is a placeholder - actual implementation will parse git diff
	QueryChangeComplexity = `
		// This query is implemented in collector.go via git diff parsing
		// Returns: lines_added, lines_deleted, lines_modified, complexity_score
	`

	// Query 7: Developer Patterns
	// Returns recent developer activity patterns for the file
	QueryDeveloperPatterns = `
		MATCH (f:File {path: $filePath, repo_id: $repoID})
		OPTIONAL MATCH (f)<-[:MODIFIES]-(c:Commit)
		WHERE c.timestamp > datetime() - duration({days: 90})
		WITH f, c.author as developer, c.author_email as email, c.timestamp as timestamp,
			 count(*) as commit_count
		ORDER BY timestamp DESC
		WITH developer, email, commit_count, max(timestamp) as last_commit
		RETURN collect({
			developer: developer,
			email: email,
			commit_count: commit_count,
			last_commit: last_commit
		}) as developer_history, count(DISTINCT developer) as team_size
	`
)

// QueryNames provides human-readable names for queries (for logging/debugging)
var QueryNames = map[string]string{
	"blast_radius":      "Blast Radius (Structural Coupling)",
	"co_change":         "Co-Change Patterns",
	"ownership_churn":   "Ownership and Churn",
	"test_coverage":     "Test Coverage",
	"incident_history":  "Incident History",
	"change_complexity": "Change Complexity",
	"developer_patterns": "Developer Patterns",
}

// QueryTimeout defines the maximum execution time for each query (in seconds)
var QueryTimeout = map[string]int{
	"blast_radius":      5,
	"co_change":         10,
	"ownership_churn":   5,
	"test_coverage":     3,
	"incident_history":  5,
	"change_complexity": 2,  // Fast (git diff parsing)
	"developer_patterns": 7,
}
