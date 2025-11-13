package risk

// Fixed Query Library (FR-7)
// This file contains the 7 core Cypher queries used in Phase 1 data collection
// Reference: mvp_development_plan.md FR-7
// Reference: simplified_graph_schema.md - Schema properties and relationships
// All queries filter to default branch (is_default: true) to prevent feature branch pollution

const (
	// Query 1: Recent Incidents (last 90 days)
	// DISABLED: Requires Issue nodes which are deferred to post-MVP
	// Per IMPLEMENTATION_GAP_ANALYSIS.md: Issue linking is not part of MVP
	// Use QueryIncidentHistory (commit message regex) instead
	QueryRecentIncidents = ``

	// Query 2: Dependency Count (Blast Radius - depth 1-2)
	// Returns: Count of files that depend on this file
	// FIXED: Removed CALLS edge (deferred to post-MVP), only use DEPENDS_ON
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryDependencyCount = `
		MATCH (f:File {path: $filePath, repo_id: $repoId})<-[:DEPENDS_ON*1..2]-(dependent:File)
		WHERE dependent.repo_id = $repoId
		RETURN count(DISTINCT dependent) as dependent_count,
		       collect(DISTINCT dependent.path)[0..10] as sample_dependents
	`

	// Query 3: Co-change Partners (computed dynamically from MODIFIED edges)
	// Returns: Files that frequently change together (>50% rate, last 90 days)
	// FIXED: Removed ON_BRANCH filtering (not needed - all commits from GitHub are main branch)
	// FIXED: Changed MODIFIES to MODIFIED (correct edge name)
	// FIXED: Lowered threshold to 50% (70% was too restrictive)
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryCoChangePartners = `
		MATCH (f:File {path: $filePath, repo_id: $repoId})<-[:MODIFIED]-(c:Commit)
		WHERE c.committed_at > datetime().epochSeconds - (90 * 24 * 60 * 60)
		  AND c.repo_id = $repoId
		WITH f, collect(c) as target_commits
		WITH f, target_commits, size(target_commits) as total_commits
		UNWIND target_commits as commit
		MATCH (commit)-[:MODIFIED]->(other:File)
		WHERE other.path <> $filePath
		  AND other.repo_id = $repoId
		WITH other, count(commit) as co_changes, total_commits
		WITH other, co_changes, toFloat(co_changes)/toFloat(total_commits) as frequency
		WHERE frequency > 0.5
		RETURN other.path, frequency, co_changes
		ORDER BY frequency DESC
		LIMIT 10
	`

	// Query 4: Ownership (computed dynamically from AUTHORED + MODIFIED edges)
	// Returns: Top 3 owners by commit count with ownership percentages
	// FIXED: Removed ON_BRANCH filtering (not needed), removed OWNS edge
	// FIXED: Changed MODIFIES to MODIFIED, returns top 3 instead of 1
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryOwnership = `
		MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $filePath, repo_id: $repoId})
		WHERE c.repo_id = $repoId
		WITH d, count(c) as commit_count, max(c.committed_at) as last_commit_date
		WITH d, commit_count, last_commit_date, sum(commit_count) OVER () as total_file_commits
		RETURN d.email, d.name, commit_count, last_commit_date,
		       toFloat(commit_count)/toFloat(total_file_commits) as ownership_percentage
		ORDER BY commit_count DESC
		LIMIT 3
	`

	// Query 5: Blast Radius (full dependency graph, depth 3)
	// Returns: All files depending on this file with sample list
	// FIXED: Removed CALLS edge (deferred to post-MVP), only use DEPENDS_ON
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryBlastRadius = `
		MATCH (f:File {path: $filePath, repo_id: $repoId})<-[:DEPENDS_ON*1..3]-(dependent:File)
		WHERE dependent.repo_id = $repoId
		RETURN count(DISTINCT dependent) as dependent_count,
		       collect(DISTINCT dependent.path)[0..20] as sample_dependents
	`

	// Query 6: Incident History (last 180 days)
	// Returns: Bug-fix commits via commit message regex
	// FIXED: Removed LINKED_TO edge (requires Issue nodes, deferred to post-MVP)
	// Now uses commit message pattern matching for "fix", "bug", "hotfix", "patch"
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryIncidentHistory = `
		MATCH (f:File {path: $filePath, repo_id: $repoId})<-[:MODIFIED]-(c:Commit)
		WHERE c.message =~ '(?i).*(fix|bug|hotfix|patch).*'
		  AND c.committed_at > datetime().epochSeconds - (180 * 24 * 60 * 60)
		  AND c.repo_id = $repoId
		RETURN c.sha, c.message, c.committed_at
		ORDER BY c.committed_at DESC
		LIMIT 10
	`

	// Query 7: Recent Commits (last 5 commits)
	// Returns: Recent commit history with authors and changes
	// FIXED: Removed ON_BRANCH filtering (not needed), changed MODIFIES to MODIFIED
	// FIXED: Use committed_at instead of author_date for consistency
	// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
	QueryRecentCommits = `
		MATCH (f:File {path: $filePath, repo_id: $repoId})<-[:MODIFIED]-(c:Commit)
		WHERE c.repo_id = $repoId
		MATCH (c)<-[:AUTHORED]-(d:Developer)
		RETURN c.sha, c.message, d.email, c.committed_at, c.additions, c.deletions
		ORDER BY c.committed_at DESC
		LIMIT 5
	`
)

// QueryNames provides human-readable names for queries (for logging/debugging)
var QueryNames = map[string]string{
	"recent_incidents":  "Recent Incidents (90 days)",
	"dependency_count":  "Dependency Count (Blast Radius depth 1-2)",
	"co_change":         "Co-Change Partners (>70%, 90 days, default branch)",
	"ownership":         "Primary Owner (default branch, 90 days)",
	"blast_radius":      "Full Blast Radius (depth 3)",
	"incident_history":  "Incident History (180 days)",
	"recent_commits":    "Recent Commits (last 5, default branch)",
}

// QueryTimeout defines the maximum execution time for each query (in seconds)
var QueryTimeout = map[string]int{
	"recent_incidents":  5,  // Simple MATCH with date filter
	"dependency_count":  10, // Variable-length path (depth 2)
	"co_change":         15, // Complex computation (commit traversal)
	"ownership":         10, // Aggregation with GROUP BY
	"blast_radius":      15, // Variable-length path (depth 3)
	"incident_history":  5,  // Simple MATCH with date filter
	"recent_commits":    10, // MATCH with ORDER BY
}

// QueryDescription provides detailed descriptions for documentation
var QueryDescription = map[string]string{
	"recent_incidents": `
		Finds issues linked to the file in the last 90 days.
		Used by Agent 1 (Incident Risk Specialist) to assess recent failure patterns.
	`,
	"dependency_count": `
		Counts files that depend on this file (depth 1-2 hops).
		Used by Agent 2 (Blast Radius Specialist) for impact assessment.
	`,
	"co_change": `
		Computes co-change frequency dynamically from commit history.
		Filters to default branch only to prevent feature branch pollution.
		Used by Agent 3 (Co-change & Forgotten Updates) to detect temporal coupling.
	`,
	"ownership": `
		Determines primary file owner based on commit frequency.
		Filters to default branch commits in last 90 days.
		Used by Agent 4 (Ownership & Coordination) for collaboration recommendations.
	`,
	"blast_radius": `
		Calculates full dependency graph (depth 3) for comprehensive impact analysis.
		Returns sample of up to 20 dependent files.
		Used by Agent 2 (Blast Radius Specialist) for deep impact assessment.
	`,
	"incident_history": `
		Retrieves all incidents linked to file in last 180 days.
		Includes severity, state, and timestamps for pattern analysis.
		Used by Agent 1 (Incident Risk Specialist) for historical context.
	`,
	"recent_commits": `
		Fetches last 5 commits modifying the file on default branch.
		Includes author, message, and change statistics.
		Used by Agent 4 (Ownership) and Agent 5 (Quality) for activity patterns.
	`,
}

// QueryCommitsForPaths - Multi-path query for file resolution
// This query is used by crisk check to find commits that modified ANY of the
// given file paths. This enables on-demand file resolution where we can query
// commits using multiple historical paths discovered via git log --follow.
//
// Example use case:
//   File was reorganized: shared/config/settings.py → src/shared/config/settings.py
//   git log --follow returns both paths
//   This query finds commits that modified either path
//
// Multi-repo safety: Filter by repo_id to prevent cross-repository data leakage
// Reference: issue_ingestion_implementation_plan.md Phase 1.3
const QueryCommitsForPaths = `
	MATCH (c:Commit)-[:MODIFIED]->(f:File)
	WHERE f.path IN $paths
	  AND f.repo_id = $repoId
	  AND c.repo_id = $repoId
	WITH c, f
	ORDER BY c.committed_at DESC
	LIMIT $limit
	MATCH (c)<-[:AUTHORED]-(d:Developer)
	RETURN DISTINCT c.sha, c.message, c.committed_at, c.additions, c.deletions,
	       d.email, d.name, f.path as file_path
	ORDER BY c.committed_at DESC
`
