// ========================================
// CodeRisk Neo4j Index Strategy
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 1
// ========================================
// Run with: cat scripts/schema/neo4j_indexes.cypher | docker exec -i coderisk-neo4j cypher-shell -u neo4j -p <password>

// ========================================
// Layer 1: Structure Indexes
// ========================================

// File.path - Primary lookup key for Layer 1
// Used in: neo4j_client.go QueryCoupling(), QueryCoChange()
CREATE CONSTRAINT file_path_unique IF NOT EXISTS
FOR (f:File) REQUIRE f.path IS UNIQUE;

// File.file_path - Primary lookup key for Layer 2 integration
// Used in: ownership_churn.go, graph builder
CREATE INDEX file_file_path_idx IF NOT EXISTS
FOR (f:File) ON (f.file_path);

// Function.unique_id - Unique identifier (filepath:name:line)
// Used in: function-level risk analysis
CREATE CONSTRAINT function_unique_id_unique IF NOT EXISTS
FOR (f:Function) REQUIRE f.unique_id IS UNIQUE;

// Class.unique_id - Unique identifier
// Used in: class-level dependency analysis
CREATE CONSTRAINT class_unique_id_unique IF NOT EXISTS
FOR (c:Class) REQUIRE c.unique_id IS UNIQUE;

// ========================================
// Layer 2: Temporal Indexes
// ========================================

// Commit.sha - Primary lookup key
// Used in: git history traversal, incident linking
CREATE CONSTRAINT commit_sha_unique IF NOT EXISTS
FOR (c:Commit) REQUIRE c.sha IS UNIQUE;

// Developer.email - Primary lookup key
// Used in: ownership_churn.go, developer metrics
CREATE CONSTRAINT developer_email_unique IF NOT EXISTS
FOR (d:Developer) REQUIRE d.email IS UNIQUE;

// Commit.author_date - Used in temporal queries
// Used in: ownership_churn.go (90-day window queries)
CREATE INDEX commit_date_idx IF NOT EXISTS
FOR (c:Commit) ON (c.author_date);

// PullRequest.number - Primary lookup key
// Used in: PR impact analysis
CREATE CONSTRAINT pr_number_unique IF NOT EXISTS
FOR (pr:PullRequest) REQUIRE pr.number IS UNIQUE;

// ========================================
// Layer 3: Incident Indexes
// ========================================

// Incident.id - Primary lookup key (UUID)
// Used in: incident linking, similarity queries
CREATE CONSTRAINT incident_id_unique IF NOT EXISTS
FOR (i:Incident) REQUIRE i.id IS UNIQUE;

// Issue.number - Primary lookup key
// Used in: GitHub issue integration
CREATE CONSTRAINT issue_number_unique IF NOT EXISTS
FOR (i:Issue) REQUIRE i.number IS UNIQUE;

// Incident.severity - Used in filtering
// Used in: high-severity incident queries
CREATE INDEX incident_severity_idx IF NOT EXISTS
FOR (i:Incident) ON (i.severity);

// ========================================
// Composite Indexes (Advanced)
// ========================================

// File branch + git_sha - For branch-aware queries (future)
// CREATE INDEX file_branch_sha_idx IF NOT EXISTS
// FOR (f:File) ON (f.branch, f.git_sha);

// ========================================
// Verification
// ========================================

SHOW CONSTRAINTS;
SHOW INDEXES;
