# CodeRisk MCP: Agentic Investigation Workflow

## Overview

The CodeRisk MCP server now supports **efficient agentic investigation** with smart filtering and pagination parameters. This enables Claude to explore code risk iteratively without overwhelming token limits.

## New Parameters for Agentic Workflows

### 1. **max_blocks** (default: 10)
Limits total number of code blocks returned. Use this to get a quick overview first, then drill down.

**Example workflow:**
```
1. Ask: "What are the risk factors for this file?"
   → Returns 10 blocks by default
2. Ask: "Show me ALL blocks" (max_blocks: 0)
   → Returns everything (use with caution!)
```

### 2. **summary_only** (default: false)
Returns aggregated statistics instead of detailed block data. Perfect for initial assessment.

**Example:**
```json
{
  "file_path": "client.py",
  "summary": {
    "total_blocks_analyzed": 52,
    "blocks_matching_filter": 52,
    "total_incidents": 0,
    "max_staleness_days": 29,
    "block_type_counts": {
      "class": 1,
      "method": 51
    }
  }
}
```

### 3. **block_types** (optional filter)
Filter by block types: `class`, `function`, `method`, etc.

**Example:**
```
Ask: "Show me risk for classes only"
→ block_types: ["class"]
```

### 4. **min_staleness** (default: 0)
Only return blocks that haven't been touched in N days. Great for finding abandoned code.

**Example:**
```
Ask: "Show me code that hasn't been touched in 30+ days"
→ min_staleness: 30
```

### 5. **min_incidents** (default: 0)
Only return blocks with at least N historical incidents. Perfect for finding truly risky code.

**Example:**
```
Ask: "Show me code with historical bugs"
→ min_incidents: 1
```

### 6. **max_coupled_blocks** (default: 1)
How many coupled blocks to show per code block. Keep low for efficiency.

### 7. **max_incidents** (default: 1)
How many incidents to show per code block. Keep low for efficiency.

### 8. **min_risk_score** (default: 0.0)
Only return blocks with risk score >= this threshold. Perfect for filtering noise and focusing on high-risk areas.

**Example:**
```
Ask: "Show me only the truly risky code"
→ min_risk_score: 15.0  # Only moderate+ risk
```

### 9. **include_risk_score** (default: false)
Include the calculated risk score in output for debugging and verification.

**Example:**
```
Ask: "Show me the risk scores"
→ include_risk_score: true
```

**Why This Matters**: Enables transparent, verifiable analysis. You can see exactly why each block is ranked where it is.

### 10. **prioritize_recent** (default: false)
Boost risk scores for recently changed code (< 30 days old). Focus on active development areas.

**Example:**
```
Ask: "What's risky in code we've been actively working on?"
→ prioritize_recent: true
```

**Scoring Boost**:
- Code changed today: +5.0 points
- Code changed 15 days ago: +2.5 points
- Code changed 30+ days ago: no boost

## Recommended Agentic Investigation Flow

### Step 1: Initial Assessment (Summary Mode)
```
User: "What's the overall risk profile of this file?"

Claude calls:
- summary_only: true
- max_blocks: 0  # Analyze all blocks for stats

Returns:
- Total blocks
- Block type distribution
- Total incident count
- Max staleness
```

### Step 2: Focus on High-Risk Areas
Based on summary, drill down into specific concerns:

#### Option A: Find Stale Code
```
User: "Show me code that might be abandoned"

Claude calls:
- min_staleness: 30
- max_blocks: 10
- max_coupled_blocks: 1
- max_incidents: 1
```

#### Option B: Find Buggy Code
```
User: "Show me code with historical bugs"

Claude calls:
- min_incidents: 1
- max_blocks: 10
- max_coupled_blocks: 2  # Show more coupling for risky code
- max_incidents: 3        # Show more incidents
```

#### Option C: Analyze Specific Block Types
```
User: "What's the risk for classes?"

Claude calls:
- block_types: ["class"]
- max_blocks: 5
```

### Step 3: Deep Dive on Specific Blocks
```
User: "Tell me more about the MCPClient class"

Claude calls:
- block_types: ["class"]
- max_blocks: 1
- max_coupled_blocks: 10  # Show all coupling
- max_incidents: 10       # Show all incidents
```

## Token Optimization Strategy

| Scenario | max_blocks | max_coupled | max_incidents | Estimated Tokens |
|----------|-----------|-------------|---------------|------------------|
| Quick overview | 10 | 1 | 1 | ~3-5k |
| Detailed analysis | 20 | 2 | 2 | ~8-12k |
| Deep dive single block | 1 | 10 | 10 | ~2-3k |
| Summary only | 0 | 0 | 0 | <1k |
| Find risky code | 10 | 1 | 1 (min_incidents: 1) | ~3-5k |

## Example Agentic Conversations

### Conversation 1: Exploring Unknown Code

```
User: What's risky about src/auth.py?

Claude → Tool call 1:
  - summary_only: true

Response:
  "45 blocks, 3 with incidents, max staleness 120 days"

Claude → Tool call 2:
  - min_incidents: 1
  - max_blocks: 5

Response:
  Shows 3 blocks with bugs

Claude → Analysis:
  "The authenticate() method has 2 historical incidents..."
```

### Conversation 2: Finding Technical Debt

```
User: Find abandoned code in src/

Claude → Tool call 1:
  - summary_only: true

Response:
  "12 files analyzed, max staleness 365 days"

Claude → Tool call 2:
  - min_staleness: 180
  - max_blocks: 10

Response:
  Shows old code

Claude → Analysis:
  "Found 4 blocks not touched in 6+ months..."
```

### Conversation 3: Understanding Coupling

```
User: What depends on UserService?

Claude → Tool call 1:
  - block_types: ["class"]
  - max_blocks: 1
  - max_coupled_blocks: 20  # Show all dependencies

Response:
  UserService class with all coupled blocks

Claude → Analysis:
  "UserService is coupled with 15 other blocks..."
```

## Best Practices for Claude

### DO:
✅ Start with `summary_only: true` for unknown files
✅ Use filters (`min_staleness`, `min_incidents`) to reduce noise
✅ Keep `max_coupled_blocks` and `max_incidents` low (1-2) unless specifically investigating coupling/incidents
✅ Use `max_blocks: 10` as default for detailed analysis
✅ Increase limits only when user asks for more details

### DON'T:
❌ Call with `max_blocks: 0` (all blocks) unless explicitly asked
❌ Use high `max_coupled_blocks` or `max_incidents` by default
❌ Ignore filters when investigating specific concerns
❌ Return massive datasets when summary would suffice

## Handling Edge Cases

### File with Many Blocks (50+)
1. Start with `summary_only: true`
2. Filter by `block_types: ["class"]` to reduce noise
3. Use `min_incidents` or `min_staleness` to focus on risk
4. Drill down on specific blocks

### File with High Coupling
1. Use `max_coupled_blocks: 1` initially
2. Only increase when investigating specific coupling relationships
3. Consider using `block_types` to filter to high-level structures

### File with Many Incidents
1. Start with `summary_only: true` to see total count
2. Use `min_incidents: 5` to find "hot spots"
3. Show top N incidents per block (`max_incidents: 3`)

## Verification of Analysis

**To verify Claude's analysis is accurate:**

1. Check the `total_blocks` count in response
2. Verify `incident_count` field per block
3. Check `staleness_days` field per block
4. Look at actual `incidents` array for evidence

**Red flags for hallucination:**
- Claude mentions data not in the response
- Block counts don't match the response
- Incident details differ from response data

## Summary

The new parameters enable Claude to:
- **Explore efficiently**: Start broad with summaries
- **Focus iteratively**: Drill down on high-risk areas
- **Manage tokens**: Use filters to reduce response size
- **Investigate thoroughly**: Increase detail only when needed

This creates a natural **agentic investigation workflow** where Claude can autonomously explore code risk without hitting token limits.
