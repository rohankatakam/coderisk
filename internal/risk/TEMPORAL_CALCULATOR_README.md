# Temporal Risk Calculator

**AGENT-P3C Implementation**

## Overview

The Temporal Risk Calculator links incidents (GitHub Issues) to CodeBlocks based on commit history, calculates incident counts, and generates LLM-powered temporal summaries to identify high-risk code areas.

## Quick Start

```go
import (
    "context"
    "database/sql"
    "github.com/rohankatakam/coderisk/internal/risk"
    "github.com/rohankatakam/coderisk/internal/llm"
)

// Initialize calculator
calculator := risk.NewTemporalCalculator(db, neo4jBackend, llmClient, repoID)

// Step 1: Link issues to code blocks
linksCreated, err := calculator.LinkIssuesViaCommits(ctx)
fmt.Printf("Created %d incident links\n", linksCreated)

// Step 2: Calculate incident counts
blocksUpdated, err := calculator.CalculateIncidentCounts(ctx)
fmt.Printf("Updated %d blocks\n", blocksUpdated)

// Step 3: Generate temporal summaries (requires LLM)
summaries, err := calculator.GenerateTemporalSummaries(ctx)
fmt.Printf("Generated %d summaries\n", summaries)

// Get statistics
stats, err := calculator.GetIncidentStatistics(ctx)
fmt.Printf("Blocks with incidents: %d\n", stats["blocks_with_incidents"])

// Get top hotspots
hotspots, err := calculator.GetTopIncidentBlocks(ctx, 10)
for _, block := range hotspots {
    fmt.Printf("- %s: %d incidents\n",
        block["block_name"],
        block["incident_count"])
}
```

## How It Works

### 1. Incident Linking

Links Issues to CodeBlocks through this relationship chain:

```
Issue -> (closed by) -> Commit -> (modified) -> CodeBlock
```

Process:
1. Queries `timeline_events` table for closed issues
2. Finds commits that closed each issue
3. Joins with `code_block_modifications` to find affected blocks
4. Creates entries in `code_block_incidents` table

**Confidence**: 0.80 (timeline-based evidence)

### 2. Incident Count Calculation

Updates `code_blocks.incident_count` field:
- Counts incidents per block from `code_block_incidents`
- Sets count to 0 for blocks without incidents
- Ensures no NULL values

### 3. Temporal Summaries

Generates LLM-powered summaries for high-incident blocks:
- Processes top 50 blocks by incident count
- Summarizes incident patterns and themes
- Provides actionable insights

## Database Schema

### code_block_incidents

```sql
CREATE TABLE code_block_incidents (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    code_block_id BIGINT NOT NULL REFERENCES code_blocks(id),
    issue_id BIGINT NOT NULL REFERENCES github_issues(id),

    -- Evidence
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0.70),
    evidence_source TEXT NOT NULL,
    evidence_text TEXT,

    -- Fix tracking
    fix_commit_sha VARCHAR(40),
    fixed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_issue UNIQUE(code_block_id, issue_id)
);
```

## API Reference

### NewTemporalCalculator

```go
func NewTemporalCalculator(
    db *sql.DB,
    neo4j *graph.Neo4jBackend,
    llmClient *llm.Client,
    repoID int64,
) *TemporalCalculator
```

Creates a new temporal calculator instance.

### LinkIssuesViaCommits

```go
func (t *TemporalCalculator) LinkIssuesViaCommits(ctx context.Context) (int, error)
```

Links issues to code blocks through commits. Returns number of links created.

**Dependencies**:
- `timeline_events` table populated
- `code_block_modifications` table populated

### CalculateIncidentCounts

```go
func (t *TemporalCalculator) CalculateIncidentCounts(ctx context.Context) (int, error)
```

Updates incident counts on all code blocks. Returns number of blocks updated.

**Side effects**: Updates `code_blocks.incident_count`

### GenerateTemporalSummaries

```go
func (t *TemporalCalculator) GenerateTemporalSummaries(ctx context.Context) (int, error)
```

Generates LLM summaries for top 50 high-incident blocks. Returns number of summaries generated.

**Requirements**:
- LLM client enabled (`PHASE2_ENABLED=true`)
- API key configured

### GetIncidentStatistics

```go
func (t *TemporalCalculator) GetIncidentStatistics(ctx context.Context) (map[string]interface{}, error)
```

Returns summary statistics:
- `total_blocks`: Total code blocks
- `blocks_with_incidents`: Blocks with incidents
- `blocks_without_incidents`: Blocks with no incidents
- `total_unique_issues`: Unique issues linked
- `total_incident_links`: Total links created
- `avg_incidents_per_block`: Average incidents
- `max_incidents_per_block`: Maximum incidents

### GetTopIncidentBlocks

```go
func (t *TemporalCalculator) GetTopIncidentBlocks(ctx context.Context, limit int) ([]map[string]interface{}, error)
```

Returns top N blocks by incident count. Each block includes:
- `id`: Block ID
- `file_path`: File path
- `block_name`: Block name
- `block_type`: Block type
- `incident_count`: Number of incidents
- `last_modified_at`: Last modification time
- `last_modifier_email`: Last modifier

### GetBlockIncidents

```go
func (t *TemporalCalculator) GetBlockIncidents(ctx context.Context, blockID int64) ([]map[string]interface{}, error)
```

Returns all incidents for a specific block. Each incident includes:
- `number`: Issue number
- `title`: Issue title
- `state`: Issue state
- `created_at`: Issue creation time
- `closed_at`: Issue close time
- `confidence`: Link confidence score
- `evidence_source`: Evidence source
- `evidence_text`: Evidence description
- `fix_commit_sha`: Fix commit SHA
- `fixed_at`: Fix timestamp

## Verification

Run verification queries:

```bash
psql $DATABASE_URL -f scripts/verify_temporal_calculator.sql
```

This checks:
1. Total incident links
2. Count distribution
3. Overall statistics
4. Count accuracy
5. Top hotspots
6. Recent links
7. Evidence quality
8. High-risk blocks
9. Coverage metrics
10. Validation status

## Edge Cases

| Case | Behavior |
|------|----------|
| Block with 0 incidents | `incident_count = 0` |
| Issue linked to 10+ blocks | All blocks get the incident |
| Issue with no commits | Skipped (no link created) |
| Multiple issues in same commit | All linked to modified blocks |
| Closed issue without timeline event | Not linked (requires timeline event) |

## Performance Considerations

- Incident linking: O(issues × commits × blocks)
- Count calculation: O(blocks)
- Summary generation: O(50 × LLM latency) for top 50 blocks
- Recommended: Run during off-peak hours for large repos

## Integration Example

```go
// Pipeline 3C integration
func RunPipeline3C(ctx context.Context, repoID int64) error {
    db, err := connectDB()
    if err != nil {
        return err
    }

    neo4j, err := connectNeo4j()
    if err != nil {
        return err
    }

    llm, err := llm.NewClient(ctx, cfg)
    if err != nil {
        return err
    }

    calc := risk.NewTemporalCalculator(db, neo4j, llm, repoID)

    // Step 1: Link incidents
    links, err := calc.LinkIssuesViaCommits(ctx)
    if err != nil {
        return fmt.Errorf("linking failed: %w", err)
    }
    log.Printf("Created %d incident links", links)

    // Step 2: Calculate counts
    updated, err := calc.CalculateIncidentCounts(ctx)
    if err != nil {
        return fmt.Errorf("count calculation failed: %w", err)
    }
    log.Printf("Updated %d blocks", updated)

    // Step 3: Generate summaries (optional)
    if llm.IsEnabled() {
        summaries, err := calc.GenerateTemporalSummaries(ctx)
        if err != nil {
            log.Printf("Warning: summary generation failed: %v", err)
        } else {
            log.Printf("Generated %d summaries", summaries)
        }
    }

    // Get final stats
    stats, err := calc.GetIncidentStatistics(ctx)
    if err != nil {
        return fmt.Errorf("stats query failed: %w", err)
    }

    log.Printf("Temporal analysis complete:")
    log.Printf("  - Blocks analyzed: %d", stats["total_blocks"])
    log.Printf("  - Blocks with incidents: %d", stats["blocks_with_incidents"])
    log.Printf("  - Average incidents: %.2f", stats["avg_incidents_per_block"])

    return nil
}
```

## Testing

Run tests:

```bash
# Set up test database
export DATABASE_URL="postgresql://user:pass@localhost:5432/coderisk_test"

# Run temporal tests
go test -v ./internal/risk -run TestTemporal

# Run all risk tests
go test -v ./internal/risk
```

## Troubleshooting

### No incidents linked

**Check**:
1. Are there closed issues? `SELECT COUNT(*) FROM github_issues WHERE state = 'closed'`
2. Are there timeline events? `SELECT COUNT(*) FROM timeline_events WHERE event_type = 'closed'`
3. Are commits linked to timeline events? `SELECT COUNT(*) FROM timeline_events WHERE commit_sha IS NOT NULL`
4. Are there code block modifications? `SELECT COUNT(*) FROM code_block_modifications`

### Count mismatches

Run validation:
```sql
SELECT cb.id, cb.incident_count AS stored, COUNT(cbi.id) AS actual
FROM code_blocks cb
LEFT JOIN code_block_incidents cbi ON cbi.code_block_id = cb.id
WHERE cb.repo_id = 1
GROUP BY cb.id, cb.incident_count
HAVING cb.incident_count != COUNT(cbi.id);
```

If rows returned, re-run `CalculateIncidentCounts()`.

### LLM summaries failing

**Check**:
1. Is `PHASE2_ENABLED=true`?
2. Is LLM API key configured?
3. Check API rate limits
4. Review LLM client logs

## References

- **Spec**: `/docs/AGENT_P3C_TEMPORAL.md`
- **Schema**: `migrations/001_code_block_schema.sql`
- **Tests**: `internal/risk/temporal_test.go`
- **Verification**: `scripts/verify_temporal_calculator.sql`

## License

Same as parent project
