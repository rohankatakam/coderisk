# CodeRisk Scoring Algorithm

## Overview

When returning code blocks, CodeRisk now **ranks them by risk score** before applying limits. This ensures you always see the **highest-risk code first**, not just arbitrary file order.

## Risk Score Formula

```
risk_score = (incidents × 10) + (coupling × 2) + staleness_capped + block_type_bonus
```

### Components

#### 1. **Temporal Risk** (Weight: 10x) - Strongest Signal
```
temporal_risk = incident_count × 10.0
```

**Rationale**: Historical incidents (bugs, issues, PRs) are the **strongest predictor** of future risk. Code that has broken before is likely to break again.

**Examples**:
- 0 incidents → 0 points
- 1 incident → 10 points
- 3 incidents → 30 points
- 10 incidents → 100 points

#### 2. **Coupling Risk** (Weight: 2x)
```
coupling_score = (sum of coupling_rates) / num_coupled_blocks × num_coupled_blocks
coupling_risk = coupling_score × 2.0
```

**Rationale**: Highly coupled code is risky because:
- Changes ripple to many places
- Harder to understand in isolation
- More likely to break when dependencies change

**Examples**:
- No coupling → 0 points
- Coupled with 1 block (rate 0.5) → 1 point
- Coupled with 5 blocks (avg rate 0.8) → 8 points
- Coupled with 10 blocks (avg rate 1.0) → 20 points

#### 3. **Staleness Risk** (Weight: 1x, capped)
```
staleness_score = min(staleness_days / 30, 3.0)
```

**Rationale**: Old code may have **knowledge risk** (nobody understands it), but fresh code isn't necessarily safe. We cap at 90 days.

**Examples**:
- 0 days → 0 points (freshly changed)
- 30 days → 1 point
- 60 days → 2 points
- 90+ days → 3 points (capped)

#### 4. **Block Type Bonus**
```
if block_type == "class":
    risk += 2.0
```

**Rationale**: Classes are structural components with broader impact than individual methods.

#### 5. **Recency Boost** (Optional, when `prioritize_recent=true`)
```
if prioritize_recent AND staleness_days < 30:
    recency_boost = (30 - staleness_days) / 30 × 5.0
    risk += recency_boost
```

**Rationale**: Recently changed code in active development is more likely to have newly introduced bugs. This boost helps surface "hot spots" where bugs are actively being introduced.

**Examples**:
- 0 days (changed today) → +5.0 points
- 7 days → +3.8 points
- 15 days → +2.5 points
- 30+ days → +0.0 points (no boost)

**When to use**: Enable when investigating active development areas or recent changes.

## Examples

### Example 1: Critical Bug-Prone Code
```
Block: PaymentProcessor.processPayment
- Incidents: 5 → 50 points
- Coupling: 8 blocks, avg rate 0.9 → 14.4 points
- Staleness: 45 days → 1.5 points
- Type: method → 0 points
Total: 65.9 points ⚠️ HIGH RISK
```

### Example 2: Stale But Stable Code
```
Block: LegacyUtils.helper
- Incidents: 0 → 0 points
- Coupling: 2 blocks, avg rate 0.3 → 1.2 points
- Staleness: 365 days → 3 points (capped)
- Type: method → 0 points
Total: 4.2 points ✅ LOW RISK
```

### Example 3: Fresh But Coupled Code
```
Block: UserService (class)
- Incidents: 1 → 10 points
- Coupling: 15 blocks, avg rate 0.7 → 21 points
- Staleness: 5 days → 0.17 points
- Type: class → 2 points
Total: 33.17 points ⚠️ MODERATE RISK
```

### Example 4: Perfect Code
```
Block: StringUtils.capitalize
- Incidents: 0 → 0 points
- Coupling: 0 blocks → 0 points
- Staleness: 10 days → 0.33 points
- Type: method → 0 points
Total: 0.33 points ✅ VERY LOW RISK
```

## Risk Levels (Approximate)

| Score Range | Risk Level | Interpretation |
|-------------|-----------|----------------|
| 0-5 | ✅ Very Low | Stable, isolated, no incident history |
| 5-15 | 🟢 Low | Minor concerns, generally safe |
| 15-30 | 🟡 Moderate | Some risk factors present |
| 30-50 | 🟠 High | Multiple risk factors, needs attention |
| 50+ | 🔴 Critical | Bug-prone, highly coupled, requires investigation |

## Sorting Behavior

When `max_blocks` is applied, blocks are returned in **descending risk score order**:

```
Block 1: risk_score = 65.9  (returned first)
Block 2: risk_score = 33.2
Block 3: risk_score = 12.5
Block 4: risk_score = 4.1
Block 5: risk_score = 0.8
...
Block 50: risk_score = 0.1  (would be cut off at max_blocks=10)
```

## Why This Matters

### Without Ranking (Before)
```
User: "Show me risk for client.py (max 10 blocks)"

Returns: First 10 blocks in file order
- Block 1: utility method (score 0.5)
- Block 2: helper function (score 1.2)
- Block 3: simple getter (score 0.3)
...
- Block 10: another helper (score 0.8)

⚠️ PROBLEM: The bug-prone authenticate() method (score 45.0) is block 23, never shown!
```

### With Ranking (After)
```
User: "Show me risk for client.py (max 10 blocks)"

Returns: Top 10 highest-risk blocks
- Block 1: authenticate() (score 45.0) ← Found the problem!
- Block 2: processPayment() (score 38.5)
- Block 3: handleError() (score 22.3)
...
- Block 10: validateInput() (score 8.1)

✅ SUCCESS: Critical issues surface first
```

## Impact on Agentic Workflows

### Scenario 1: Quick Assessment
```
Claude calls: max_blocks=10

Before: Random 10 blocks, might miss all risky code
After: Top 10 riskiest blocks guaranteed
```

### Scenario 2: Finding Hot Spots
```
Claude calls: min_incidents=1, max_blocks=5

Before: First 5 blocks with incidents (arbitrary order)
After: 5 worst bug-prone blocks
```

### Scenario 3: Understanding Coupling
```
Claude calls: max_blocks=10

Before: Might show 10 isolated utilities
After: Shows highly coupled blocks first
```

## Tuning Considerations

The weights can be adjusted based on your needs:

**Conservative** (prioritize proven issues):
- Incidents: 15x (emphasize historical bugs)
- Coupling: 1x (de-emphasize coupling)
- Staleness: 0.5x (less concern about old code)

**Aggressive** (catch potential issues):
- Incidents: 5x (don't over-index on past)
- Coupling: 5x (flag architectural risks)
- Staleness: 2x (highlight knowledge gaps)

**Current** (balanced):
- Incidents: 10x
- Coupling: 2x
- Staleness: 1x (capped at 3)

## Verification

To verify ranking is working:

1. Call with `max_blocks=10`
2. Note the incident counts in results
3. They should generally decrease (highest first)
4. Coupling should be relatively high in top results

## Future Enhancements

Potential improvements:

1. **Change frequency**: Factor in recent change velocity
2. **Author diversity**: Risk when many authors touch code
3. **Complexity**: Use cyclomatic complexity if available
4. **Dependency depth**: Code deep in call chains is risky
5. **Test coverage**: Lower coverage = higher risk

## Summary

**Risk ranking ensures you always see the most important code first**, making the `max_blocks` limit a feature rather than a limitation. You get the highest signal-to-noise ratio possible within token constraints.
