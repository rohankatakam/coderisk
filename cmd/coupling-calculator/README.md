# Coupling Risk Calculator

A command-line tool to analyze co-change relationships between code blocks and calculate coupling risk metrics.

## Overview

The Coupling Risk Calculator (AGENT-P3B) analyzes your codebase to identify code blocks that frequently change together, which can indicate architectural coupling and potential risk areas.

## Prerequisites

- PostgreSQL database with ingested code blocks (from AGENT-P2A/P2B)
- Go 1.21 or higher
- Optional: Gemini API key for LLM-powered coupling explanations

## Usage

### Basic Usage

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/coderisk"
go run cmd/coupling-calculator/main.go <repo_id>
```

### With LLM Explanations

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/coderisk"
export GEMINI_API_KEY="your-api-key-here"
export PHASE2_ENABLED="true"
go run cmd/coupling-calculator/main.go <repo_id>
```

### Example Output

```
✓ Connected to database
✓ LLM client enabled (gemini)
✓ Coupling calculator initialized for repo_id=1

═══════════════════════════════════════════════════════
STEP 1: Calculating Co-Change Relationships
═══════════════════════════════════════════════════════

Co-change pair #1 (rate: 85.00%): These blocks likely change together because they implement the request/response cycle for the authentication flow. The parseAuthRequest function extracts credentials, while validateCredentials checks them against the database.

Co-change pair #2 (rate: 75.00%): Both functions are part of the user session management system. When session storage changes, both creation and validation logic need updates to maintain consistency.

✓ Created 42 co-change edges (rate >= 50%)

═══════════════════════════════════════════════════════
STEP 2: Co-Change Statistics
═══════════════════════════════════════════════════════
Total Edges:         42
Min Coupling Rate:   50.0%
Max Coupling Rate:   100.0%
Avg Coupling Rate:   67.3%

Coupling Distribution:
  High (≥75%):       15 edges
  Medium (50-75%):   27 edges

═══════════════════════════════════════════════════════
STEP 3: Top 10 Most Coupled Blocks
═══════════════════════════════════════════════════════

1. handleAuthRequest (function)
   File: src/auth/handler.go
   Couplings: 8 edges
   Avg Rate: 72.5%

2. validateSession (function)
   File: src/auth/validator.go
   Couplings: 6 edges
   Avg Rate: 68.3%

...

═══════════════════════════════════════════════════════
✓ Analysis Complete
═══════════════════════════════════════════════════════
```

## How It Works

### Co-Change Detection

The tool analyzes your repository's commit history to find code blocks (functions, methods, classes) that frequently change together:

1. **Queries modifications**: Finds all commits that modified code blocks
2. **Identifies patterns**: Detects blocks that changed in the same commits
3. **Calculates rates**: Computes co-change rate as `co_changes / total_block_changes`
4. **Filters significant coupling**: Only reports pairs with ≥50% co-change rate

### Coupling Metrics

- **Co-change rate**: Percentage of times blocks changed together (0-100%)
- **Co-change count**: Absolute number of commits affecting both blocks
- **Total couplings**: Number of different blocks a given block couples with
- **Average rate**: Mean coupling rate across all relationships

### Threshold

Only block pairs with **≥50% co-change rate** are considered significantly coupled. This filters out coincidental changes while highlighting true architectural coupling.

## Interpreting Results

### High Coupling (≥75%)

**Interpretation**: Strong architectural coupling
**Action**:
- Review if coupling is intentional (e.g., tightly related features)
- Consider refactoring if it indicates poor separation of concerns
- Ensure comprehensive testing when either block changes

### Medium Coupling (50-75%)

**Interpretation**: Moderate coupling, likely related functionality
**Action**:
- Monitor for increase in coupling over time
- Document the relationship for team awareness
- Consider coordination when making changes

### Edge Cases

- **Blocks in same file**: High co-change rate is expected and not necessarily problematic
- **Utility functions**: May couple with many blocks - this can be acceptable
- **Recently created blocks**: May show artificially high coupling with limited history

## Database Schema

The tool uses the `code_block_co_changes` table:

```sql
CREATE TABLE code_block_co_changes (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    block_a_id BIGINT NOT NULL,
    block_b_id BIGINT NOT NULL,
    co_change_count INTEGER,
    co_change_rate DECIMAL(3,2),
    last_co_changed_at TIMESTAMP,
    last_co_change_commit_sha VARCHAR(40),
    CONSTRAINT unique_block_pair UNIQUE(block_a_id, block_b_id)
);
```

## API Usage

You can also use the coupling calculator programmatically:

```go
import (
    "context"
    "github.com/rohankatakam/coderisk/internal/risk"
)

// Initialize
calc := risk.NewCouplingCalculator(db, llmClient, repoID)

// Calculate co-changes
edgesCreated, err := calc.CalculateCoChanges(ctx)

// Get statistics
stats, err := calc.GetCoChangeStatistics(ctx)

// Find top coupled blocks
topBlocks, err := calc.GetTopCoupledBlocks(ctx, 10)
```

## Troubleshooting

### "DATABASE_URL not set"

Set the environment variable:
```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/coderisk"
```

### "No coupled blocks found"

This can happen if:
- The repository has no code blocks ingested (run AGENT-P2A/P2B first)
- Blocks haven't been modified enough to show patterns
- All co-change rates are below 50% threshold

### LLM explanations not appearing

- Ensure `PHASE2_ENABLED=true`
- Set `GEMINI_API_KEY` environment variable
- Check API key validity and rate limits

## Performance

- **Query complexity**: O(C × B²) where C = commits, B = blocks per commit
- **Optimization**: Uses PostgreSQL CTEs and indexes for efficient computation
- **Scalability**: Tested with repositories containing 1000+ blocks
- **Incremental**: ON CONFLICT DO UPDATE ensures idempotent operation

## Related Components

- **AGENT-P2A**: LLM Code Block Atomizer (prerequisite)
- **AGENT-P2B**: Event Processor (prerequisite)
- **Pipeline 1**: File-level graph construction
- **Pipeline 3**: Block-level risk analysis

## References

- [AGENT-P3B Specification](/Users/rohankatakam/Documents/brain/docs/AGENT_P3B_COUPLING.md)
- [Completion Report](/Users/rohankatakam/Documents/brain/docs/AGENT_P3B_COMPLETION_REPORT.md)
- [Database Schema](../../migrations/001_code_block_schema.sql)
