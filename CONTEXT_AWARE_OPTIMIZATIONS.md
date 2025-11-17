# Context-Aware Optimizations for Maximum Signal Density

## Overview

The CodeRisk MCP server now includes **context-aware optimizations** that ensure the highest-relevance data is always returned, even when limiting results due to token constraints. These optimizations maximize signal density and ensure important results are never missed.

## New Parameters

### 1. **min_risk_score** (filter by calculated risk)

**Purpose**: Only return code blocks that meet a minimum risk threshold.

**Use Case**: Focus exclusively on high-risk areas, filtering out noise.

**Examples**:
```
min_risk_score: 5.0   → Only blocks with moderate+ risk
min_risk_score: 15.0  → Only high-risk blocks
min_risk_score: 30.0  → Only critical blocks
```

**Agentic Workflow**:
```
User: "Show me only the truly risky code"

Claude calls:
- min_risk_score: 15.0
- max_blocks: 10

Returns: Top 10 blocks that exceed risk score 15.0
```

**Why This Matters**: Without this filter, you might get 10 low-risk blocks and miss the critical ones. With it, you're guaranteed to see only blocks above the threshold.

---

### 2. **include_risk_score** (show calculated scores)

**Purpose**: Include the actual risk score in the output for verification and debugging.

**Use Case**: Verify Claude's analysis is based on actual data, not hallucinations.

**Example Output**:
```json
{
  "block_name": "authenticate",
  "block_type": "method",
  "incident_count": 5,
  "staleness_days": 45,
  "coupled_blocks": [...],
  "risk_score": 65.9
}
```

**Agentic Workflow**:
```
User: "What are the top risks? Show me the scores."

Claude calls:
- max_blocks: 10
- include_risk_score: true

Returns: 10 blocks with visible risk scores

Claude's response: "The highest risk is authenticate() with a score of 65.9,
primarily due to 5 historical incidents (50 points) and high coupling (14.4 points)."
```

**Why This Matters**: Enables transparent, verifiable risk analysis. You can see exactly why each block is ranked where it is.

---

### 3. **prioritize_recent** (boost fresh code)

**Purpose**: Boost risk scores for recently changed code (< 30 days old).

**Use Case**: Focus on active development areas where new bugs are most likely.

**Scoring Logic**:
```
If block changed within last 30 days:
  recency_boost = (30 - days_since_change) / 30 * 5.0

Examples:
- Changed today (0 days)    → +5.0 boost
- Changed 15 days ago       → +2.5 boost
- Changed 30 days ago       → +0.0 boost
- Changed 60+ days ago      → +0.0 boost (no penalty)
```

**Agentic Workflow**:
```
User: "What's risky in code we've been actively working on?"

Claude calls:
- prioritize_recent: true
- max_blocks: 10

Returns: Top 10 blocks with recency boost applied

Result: Fresh code with incidents surfaces first, even if absolute risk is lower
```

**Why This Matters**:
- Fresh code + incidents = likely to have more bugs being introduced
- Stale code + no incidents = likely stable
- This parameter shifts focus to "hot spots" in active development

**Example Comparison**:

**Without prioritize_recent**:
```
1. OldClass.method (60 days old, 3 incidents) → score: 32.0
2. NewService.handler (5 days old, 2 incidents) → score: 20.2
```

**With prioritize_recent**:
```
1. NewService.handler (5 days old, 2 incidents) → score: 24.4 (+4.2 boost)
2. OldClass.method (60 days old, 3 incidents) → score: 32.0 (no boost)
```

The fresh code with recent changes gets boosted, making it more likely to appear in limited results.

---

## Combined Optimization Strategies

### Strategy 1: Find Critical Bugs in Active Code
```
Parameters:
- prioritize_recent: true
- min_incidents: 1
- max_blocks: 5
- include_risk_score: true

Result: Top 5 recently-changed blocks with bug history
```

### Strategy 2: Focus on Severe Risk Only
```
Parameters:
- min_risk_score: 20.0
- max_blocks: 10
- include_risk_score: true

Result: 10 highest-risk blocks, all above threshold 20.0
```

### Strategy 3: Deep Investigation with Transparency
```
Parameters:
- max_blocks: 1
- max_coupled_blocks: 10
- max_incidents: 10
- include_risk_score: true

Result: Single highest-risk block with full coupling/incident data + score
```

### Strategy 4: Active Development Hot Spots
```
Parameters:
- prioritize_recent: true
- min_staleness: 0  # Fresh code only
- min_risk_score: 5.0
- max_blocks: 10

Result: Fresh code with moderate+ risk scores (boosted by recency)
```

---

## Risk Score Calculation (with new optimizations)

```
Base Score:
  incidents × 10.0
  + coupling × 2.0
  + (staleness_days / 30, capped at 3.0)
  + (2.0 if class)

If prioritize_recent=true AND staleness < 30 days:
  + ((30 - staleness) / 30 × 5.0)

Then filter:
  if score < min_risk_score: exclude
```

### Example Calculations

#### Case 1: Fresh Bug-Prone Code (with prioritize_recent)
```
Block: AuthService.login
- Incidents: 2 → 20 points
- Coupling: 5 blocks, avg 0.7 → 7 points
- Staleness: 3 days → 0.1 points
- Type: method → 0 points
- Recency boost: (30-3)/30 × 5 → 4.5 points
Total: 31.6 points (HIGH RISK due to recency boost)
```

#### Case 2: Same Block Without Recency Boost
```
Block: AuthService.login
- Incidents: 2 → 20 points
- Coupling: 5 blocks, avg 0.7 → 7 points
- Staleness: 3 days → 0.1 points
- Type: method → 0 points
Total: 27.1 points (MODERATE RISK without boost)
```

**Impact**: The 4.5 point boost can push a block from "might not appear in top 10" to "guaranteed in top 5".

---

## Signal Density Maximization

### Problem: Limited Context Window
Claude Code limits MCP responses to ~25,000 tokens. For large files (50+ blocks), we must choose which blocks to return.

### Solution: Multi-Layered Optimization

1. **Calculate risk scores for ALL blocks** (before filtering)
2. **Apply filters** (min_incidents, min_staleness, min_risk_score)
3. **Apply ranking** (sort by risk score descending)
4. **Apply limits** (max_blocks)

**Result**: The blocks returned are guaranteed to be the most relevant, not arbitrary.

### Verification Example

**File with 92 blocks, max_blocks=10**:

**Without Optimizations** (before):
- Returns blocks 1-10 in file order
- Might miss block 50 (highest risk)
- Signal-to-noise: LOW

**With Risk Ranking Only**:
- Returns top 10 by risk score
- But includes low-risk blocks if no filter
- Signal-to-noise: MEDIUM

**With Full Optimizations**:
```
Parameters:
- min_risk_score: 10.0
- prioritize_recent: true
- max_blocks: 10
- include_risk_score: true

Returns:
1. Block A (score: 45.2) - fresh, high incidents
2. Block B (score: 38.7) - fresh, high coupling
3. Block C (score: 32.1) - fresh, moderate incidents
...
10. Block J (score: 10.3) - barely above threshold

Signal-to-noise: VERY HIGH
```

All returned blocks are above threshold AND ranked by relevance.

---

## Agentic Investigation Flow (Updated)

### Step 1: Initial Assessment
```
Claude calls:
- summary_only: true

Returns: Aggregated stats (total blocks, incidents, staleness)
```

### Step 2: Identify Risk Areas
```
Based on stats, decide which filter to use:

If high incident count:
  → min_incidents: 1, prioritize_recent: true

If high staleness:
  → min_staleness: 90, min_risk_score: 5.0

If large file:
  → min_risk_score: 15.0 (only high risk)
```

### Step 3: Focused Investigation
```
Claude calls:
- min_risk_score: 15.0
- prioritize_recent: true
- max_blocks: 10
- include_risk_score: true

Returns: Top 10 high-risk blocks with scores
```

### Step 4: Verification
```
Claude analyzes response, checking:
- Are risk_scores decreasing? (verify ranking)
- Do incident_counts match risk_scores? (verify calculation)
- Is analysis grounded in actual data? (no hallucination)
```

---

## Best Practices for Maximum Signal Density

### DO:
✅ Use `min_risk_score` to filter noise
✅ Use `prioritize_recent` for active development investigations
✅ Use `include_risk_score` to verify Claude's analysis
✅ Combine filters for compound queries (e.g., fresh + high incidents)
✅ Start with summary mode, then drill down with filters

### DON'T:
❌ Use low limits without filters (might miss high-risk code)
❌ Ignore `include_risk_score` when debugging incorrect analysis
❌ Use `prioritize_recent` for finding all bugs (biases toward fresh code)
❌ Forget to verify that risk scores match expectations

---

## Performance Characteristics

### Token Usage by Configuration

| Configuration | Est. Tokens | Signal Density |
|--------------|-------------|----------------|
| max_blocks: 10, no filters | ~15k | Low (arbitrary) |
| max_blocks: 10, min_incidents: 1 | ~12k | Medium (filtered) |
| max_blocks: 10, min_risk_score: 15.0 | ~8k | High (threshold) |
| max_blocks: 5, min_risk_score: 20.0, include_risk_score: true | ~6k | Very High |

**Observation**: Stricter filters reduce token usage AND increase relevance.

---

## Examples of Context-Aware Behavior

### Example 1: Large File with Mixed Risk

**File**: `client.py` (92 blocks)

**Without Optimization**:
```
max_blocks: 10
→ Returns blocks 1-10 (might all be low-risk utilities)
```

**With Optimization**:
```
min_risk_score: 10.0
max_blocks: 10
include_risk_score: true

→ Returns:
1. authenticate (score: 45.2)
2. sendRequest (score: 32.1)
3. handleError (score: 28.3)
...
10. parseResponse (score: 10.5)
```

All blocks are above threshold. High signal.

---

### Example 2: Active Development Focus

**Question**: "What's risky in code changed this week?"

**Claude calls**:
```
prioritize_recent: true
min_staleness: 0
max_blocks: 10
include_risk_score: true
```

**Returns**: Fresh code with recency boost applied, ranked by boosted scores.

---

### Example 3: Finding Hidden Risks

**Question**: "Show me risky code that might not be obvious"

**Claude calls**:
```
min_risk_score: 15.0  # Moderate+ risk
min_incidents: 0      # Include code without historical bugs
max_blocks: 10
include_risk_score: true
```

**Returns**: Code with high risk due to coupling/staleness, not just incidents.

---

## Summary

The new context-aware optimizations ensure:

1. **Highest relevance**: Filtering by risk score ensures you only see what matters
2. **Focus flexibility**: `prioritize_recent` shifts focus to active development
3. **Transparency**: `include_risk_score` enables verification of analysis
4. **Maximum density**: Combined filters + ranking = best signal-to-noise ratio

**Result**: Even with strict token limits, you always see the most important code first.
