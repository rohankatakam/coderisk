# MCP Server Test Questions

## Test Setup
Before testing, ensure:
1. MCP server is rebuilt: `cd /Users/rohankatakam/Documents/brain/coderisk && go build -o bin/crisk-check-server ./cmd/crisk-check-server`
2. MCP server is configured in Claude Code
3. Database has Omnara data loaded

## Category 1: Token Limit Handling (Critical)

### Test 1.1: Large File Without Filters
**Question:** "What are the risk factors for libraries/python/mcp_use/client.py?"

**Expected Behavior:**
- Should automatically apply sensible defaults (max_blocks ~10)
- Should NOT exceed 25k token limit
- Should return highest-risk blocks first (ranked by risk score)
- Should include pagination indicators

**Success Criteria:**
- ✅ No token overflow error
- ✅ Returns top 10 blocks by risk score
- ✅ Includes incident counts, staleness, coupling

### Test 1.2: Large File With Manual Limits
**Question:** "Show me the top 5 riskiest blocks in libraries/python/mcp_use/client.py with their risk scores"

**Expected Behavior:**
- Should use max_blocks=5
- Should use include_risk_score=true
- Should show descending risk scores

**Success Criteria:**
- ✅ Exactly 5 blocks returned
- ✅ Risk scores visible and decreasing
- ✅ Most severe incidents shown first

## Category 2: Risk Ranking Verification

### Test 2.1: Basic Risk Ranking
**Question:** "What are the risk factors for apps/web/src/components/dashboard/chat/ChatMessage.tsx?"

**Expected Behavior:**
- Blocks should be ordered by risk score (descending)
- Blocks with more incidents should appear higher
- Highly coupled blocks should rank higher

**Success Criteria:**
- ✅ First block has highest incident count or coupling
- ✅ Risk decreases as you read down the list
- ✅ No low-risk utility functions at the top

### Test 2.2: Risk Score Transparency
**Question:** "Show me the risk scores for apps/studio/components/grid/components/menu/RowContextMenu.tsx"

**Expected Behavior:**
- Should include actual risk score numbers
- Should explain scoring components (incidents × 10, coupling × 2, etc.)

**Success Criteria:**
- ✅ Risk scores visible in output
- ✅ Can verify scoring formula manually
- ✅ Claude explains what contributes to each score

## Category 3: Filter Effectiveness

### Test 3.1: Minimum Risk Score Filter
**Question:** "Show me only code in apps/studio/pages/ with risk score above 15"

**Expected Behavior:**
- Should use min_risk_score=15.0
- Should only return blocks meeting threshold
- No low-risk noise

**Success Criteria:**
- ✅ All returned blocks have score ≥ 15
- ✅ Filters out stable utility code
- ✅ Response is concise and focused

### Test 3.2: Incident Filter
**Question:** "Show me code with historical bugs in packages/ui-patterns/src/form/"

**Expected Behavior:**
- Should use min_incidents=1
- Should only show blocks with bug history
- Should rank by incident count

**Success Criteria:**
- ✅ All blocks have incident_count > 0
- ✅ Blocks with most incidents appear first
- ✅ No incident-free code shown

### Test 3.3: Staleness Filter
**Question:** "Find abandoned code (30+ days old) in apps/studio/components/"

**Expected Behavior:**
- Should use min_staleness=30
- Should show staleness_days for each block
- Should exclude recently changed code

**Success Criteria:**
- ✅ All blocks have staleness ≥ 30 days
- ✅ Staleness values visible
- ✅ Fresh code excluded

## Category 4: Recency Prioritization

### Test 4.1: Active Development Focus
**Question:** "What's risky in code we've been actively working on in apps/web/src/?"

**Expected Behavior:**
- Should use prioritize_recent=true
- Should boost scores for fresh code (<30 days)
- Should surface recent bug introductions

**Success Criteria:**
- ✅ Recent code (0-30 days) appears first
- ✅ Risk scores boosted for fresh code
- ✅ Stale code deprioritized

### Test 4.2: Recent + Incidents Combo
**Question:** "Show me recently changed code with bug history in apps/studio/"

**Expected Behavior:**
- Should use prioritize_recent=true AND min_incidents=1
- Should find "hot spots" (active + buggy)

**Success Criteria:**
- ✅ All results are recent (<30 days) AND have incidents
- ✅ Highest boosted scores appear first
- ✅ Filters out stable old code and fresh bug-free code

## Category 5: Summary Mode Efficiency

### Test 5.1: Quick Overview
**Question:** "Give me a quick risk assessment of apps/web/src/utils/statusUtils.ts"

**Expected Behavior:**
- Should use summary_only=true
- Should return aggregated stats
- Should be <1k tokens

**Success Criteria:**
- ✅ Returns total_blocks, total_incidents, max_staleness
- ✅ No detailed block data (efficient)
- ✅ Response is very concise

### Test 5.2: Summary Then Drill-Down
**Question:** "Analyze apps/studio/components/layouts/AuthLayout/AuthLayout.tsx - start with overview then show risky parts"

**Expected Behavior:**
- Claude should make 2 calls:
  1. summary_only=true (overview)
  2. min_risk_score=10.0, max_blocks=10 (details)

**Success Criteria:**
- ✅ First response is summary stats
- ✅ Second response is filtered high-risk blocks
- ✅ Two-stage investigation pattern

## Category 6: Coupling Analysis

### Test 6.1: Basic Coupling
**Question:** "What files are coupled with libraries/python/mcp_use/session.py?"

**Expected Behavior:**
- Should show coupled_blocks with confidence scores
- Should limit to max_coupled_blocks (1-2 by default)

**Success Criteria:**
- ✅ Shows coupling relationships
- ✅ Includes coupling confidence/rate
- ✅ Not overwhelming (limited results)

### Test 6.2: Deep Coupling Investigation
**Question:** "Show me ALL coupling relationships for libraries/python/mcp_use/client.py class definition"

**Expected Behavior:**
- Should use max_coupled_blocks=10+ for deep dive
- Should focus on class-level coupling

**Success Criteria:**
- ✅ Shows extensive coupling data
- ✅ Focused on specific block type
- ✅ Explains ripple effect of changes

## Category 7: Multi-file Comparison

### Test 7.1: Compare Risk Levels
**Question:** "Compare the risk between apps/web/src/utils/statusUtils.ts and apps/studio/pages/new/[slug].tsx"

**Expected Behavior:**
- Claude should call tool twice (once per file)
- Should use efficient parameters (summary or limited blocks)
- Should synthesize comparison

**Success Criteria:**
- ✅ Analyzes both files
- ✅ Provides comparative analysis
- ✅ Highlights which is riskier and why

## Category 8: Edge Cases & Error Handling

### Test 8.1: Non-existent File
**Question:** "What's the risk for fake/nonexistent/file.ts?"

**Expected Behavior:**
- Should return error gracefully
- Should not crash or timeout

**Success Criteria:**
- ✅ Error message is clear
- ✅ Doesn't cause MCP server crash
- ✅ Claude handles gracefully

### Test 8.2: File with No Risk Data
**Question:** "Analyze pyproject.toml for risk"

**Expected Behavior:**
- Should handle config files gracefully
- May have limited/no risk data

**Success Criteria:**
- ✅ Returns valid response (even if empty)
- ✅ Explains lack of risk data
- ✅ Doesn't error out

## Category 9: Performance & Speed

### Test 9.1: Response Time
**Question:** "Quick risk check for apps/web/src/components/dashboard/SidebarDashboardLayout.tsx"

**Expected Behavior:**
- Should respond in <5 seconds
- Should use efficient parameters

**Success Criteria:**
- ✅ Response time <5 seconds
- ✅ Doesn't timeout
- ✅ Returns useful data quickly

## Category 10: AI Explanation Quality (Phase 2)

### Test 10.1: Explain Risk
**Question:** "Explain why apps/studio/components/grid/components/menu/RowContextMenu.tsx has the risk level it does"

**Expected Behavior:**
- Should provide AI-generated explanation
- Should reference specific risk factors
- Should be actionable

**Success Criteria:**
- ✅ Explanation references actual data (incidents, coupling, staleness)
- ✅ Provides context and recommendations
- ✅ Not hallucinated (grounded in actual metrics)

## Testing Checklist

After running all tests, verify:

- [ ] No token overflow errors (25k limit)
- [ ] Risk scores are visible when requested
- [ ] Filtering works (min_risk_score, min_incidents, min_staleness)
- [ ] Ranking is correct (highest risk first)
- [ ] Recency boost works (prioritize_recent)
- [ ] Summary mode is efficient
- [ ] Coupling data is limited but useful
- [ ] Multi-file queries work
- [ ] Error handling is graceful
- [ ] Response times are reasonable (<5s)
- [ ] AI explanations are grounded in data

## Quick Test Script

```bash
# Rebuild MCP server
cd /Users/rohankatakam/Documents/brain/coderisk && \
go build -o bin/crisk-check-server ./cmd/crisk-check-server && \
echo "✅ MCP server rebuilt"

# Test 1: Basic query (should not overflow)
# Ask in Claude Code: "What are the risk factors for libraries/python/mcp_use/client.py?"

# Test 2: Risk ranking
# Ask: "Show me risk scores for apps/web/src/components/dashboard/chat/ChatMessage.tsx"

# Test 3: Filtering
# Ask: "Show me only code with risk score above 15 in apps/studio/"

# Test 4: Recency
# Ask: "What's risky in recently changed code?"

# Test 5: Summary
# Ask: "Quick risk assessment of apps/web/src/utils/statusUtils.ts"
```

## Expected Improvements from Optimizations

### Before Optimizations:
- ❌ Token overflows on large files
- ❌ Arbitrary block order (not risk-ranked)
- ❌ Can't filter noise
- ❌ No recency awareness
- ❌ All-or-nothing detail level

### After Optimizations:
- ✅ Never exceeds 25k tokens
- ✅ Highest-risk blocks always shown first
- ✅ Smart filtering (min_risk_score, min_incidents, min_staleness)
- ✅ Recency boost for active development
- ✅ Summary mode for efficient overviews
- ✅ Transparent risk scores (include_risk_score)

## Notes for Testing

1. **Test incrementally**: Don't change multiple things at once
2. **Verify actual data**: Check if risk scores match incident/coupling counts
3. **Compare before/after**: If you have old outputs, compare
4. **Check token counts**: Claude Code shows token usage - watch for reduction
5. **Iterate**: If a test fails, rebuild and retest

Good luck! 🚀
