# Ground Truth Expansion Recommendations

## Current Status
- **Test cases:** 6
- **Precision:** 100% ✅
- **Recall:** 75% ✅
- **F1 Score:** 85.71% ✅

## Why Expand Ground Truth?

### Statistical Confidence
- 6 test cases is too small for production validation
- Recommended minimum: **15-20 test cases** for statistical significance
- Industry standard: 30+ cases for high-confidence metrics

### Pattern Coverage Gaps
| Pattern | Current Coverage | Need |
|---------|-----------------|------|
| Temporal (20%) | 3 cases ✅ | Add 2-3 more |
| Explicit (60%) | 0 cases ❌ | **Add 5-7 cases** |
| True Negatives | 2 cases ✅ | Add 2-3 more |
| Comment-based (15%) | 0 cases ❌ | Add 2-3 cases (future) |
| Semantic (10%) | 1 case ✅ | Add 1-2 more |
| Cross-ref (8%) | 0 cases ❌ | Add 1-2 cases (future) |

---

## Recommended Issues to Add (Priority Order)

### 🔴 High Priority: Explicit References (5 cases)

These test Pattern 1 (60% of real-world cases) which we haven't validated yet:

#### 1. Issue #122 → PR #123 ✅ HIGH VALUE
- **Title:** "[BUG] Dashboard does not show up claude code output"
- **Detection:** `pr_extraction` (confidence: 0.95)
- **Why:** High-confidence explicit reference (likely "Fixes #122" in PR)
- **Pattern:** Explicit reference in PR body
- **Difficulty:** Easy

#### 2. Issue #115 → PR #120 ✅ HIGH VALUE
- **Title:** "[BUG] Claude output didn't show up in dashboard"
- **Detection:** `pr_extraction` (confidence: 0.95)
- **Why:** High-confidence explicit reference, similar to #122
- **Pattern:** Explicit reference in PR body
- **Difficulty:** Easy

#### 3. Issue #53 → PR #54 ✅ IMPORTANT
- **Title:** "[BUG] Claude Code Not Found"
- **Detection:** `pr_extraction` (confidence: 0.90)
- **Why:** Multiple PRs linked (PR #38 and #54), tests multi-PR handling
- **Pattern:** Multiple explicit references
- **Difficulty:** Medium

#### 4. Issue #160 → PR #162 ✅ IMPORTANT
- **Title:** "[FEATURE] Name Agents"
- **Detection:** `pr_extraction` (confidence: 0.75)
- **Why:** Feature request with explicit link
- **Pattern:** Explicit reference in feature PR
- **Difficulty:** Easy

#### 5. Issue #165 → PR #166 ✅ IMPORTANT
- **Title:** "[FEATURE] [DOCS] Architecture visibility in dark mode"
- **Detection:** `pr_extraction` (confidence: 0.75)
- **Why:** Documentation change, tests non-code PRs
- **Pattern:** Explicit reference in docs PR
- **Difficulty:** Easy

### 🟡 Medium Priority: True Negatives (3 cases)

Test precision by ensuring we don't create false positives:

#### 6. Issue #206 ⚠️ VALIDATE FIRST
- **Title:** "[Proposal] Rename Omnara"
- **State:** Closed (completed)
- **Why:** Proposal that was rejected/not implemented
- **Expected:** No links (true negative)
- **Difficulty:** Easy
- **Action:** Manually verify on GitHub that no PR exists

#### 7. Issue #178 ⚠️ VALIDATE FIRST
- **Title:** "[BUG] incorrect monthly agent limit exceeded"
- **State:** Closed (completed)
- **Why:** May have been config change, not code change
- **Expected:** No links (true negative)
- **Difficulty:** Medium
- **Action:** Check if closed without PR

#### 8. Issue #76 ⚠️ VALIDATE FIRST
- **Title:** "[BUG] Weird behavior with subagents/tasks"
- **State:** Closed (completed)
- **Why:** Vague bug report, likely closed without fix
- **Expected:** No links (true negative)
- **Difficulty:** Hard

### 🟢 Low Priority: Temporal Edge Cases (3 cases)

Test boundary conditions and complex scenarios:

#### 9. Issue #62 → PR (multiple) 📊 COMPLEX
- **Title:** "[FEATURE] Other modes of Claude Code"
- **Detection:** `temporal` (confidence: 0.85)
- **Why:** Highest temporal confidence (0.85), 3 PRs linked
- **Pattern:** Multiple PRs for one feature
- **Difficulty:** Medium

#### 10. Issue #164 → PR (multiple) 📊 COMPLEX
- **Title:** "[FEATURE] codex cli integration"
- **Detection:** `temporal` (confidence: 0.65)
- **Why:** 6 PRs linked, tests incremental feature development
- **Pattern:** Feature built across multiple PRs
- **Difficulty:** Hard

#### 11. Issue #147 → PR ⏱️ TIME BOUNDARY
- **Title:** "[FEATURE] Fast Apply model for better/faster/cheaper edits"
- **Detection:** `temporal` (confidence: 0.75)
- **Why:** Mix of temporal + explicit (0.75 confidence suggests boost)
- **Pattern:** Temporal + possible explicit reference
- **Difficulty:** Medium

### 🔵 Future: Comment-Based (2-3 cases)

Save for after implementing Pattern 3 (Comment-Based Linking):

#### 12-14. TBD - Need to query issue comments
- Search for "Fixed in PR #", "Addressed in", "See PR #" in comments
- Requires GitHub API or comment scraping

---

## Validation Process for New Ground Truth

Before adding each issue to `omnara_ground_truth.json`, **manually verify on GitHub**:

### For Expected Positives:
1. ✅ **Find the PR/commit:** Navigate to issue on GitHub, check linked PRs
2. ✅ **Verify the fix:** Confirm PR actually addresses the issue
3. ✅ **Check references:** Look for "Fixes #", "Closes #" in PR body/commit
4. ✅ **Note timestamps:** Record issue closed_at and PR merged_at
5. ✅ **Calculate similarity:** Estimate keyword overlap between issue title and PR title

### For True Negatives:
1. ✅ **Check issue comments:** Look for any PR/commit mentions
2. ✅ **Search PR titles:** Search repo PRs for issue number
3. ✅ **Verify close reason:** Confirm issue closed without code changes
4. ✅ **Check commits:** Search commits for issue number

---

## Proposed Ground Truth Structure for New Cases

### Example: Issue #122 (Explicit Reference)

```json
{
  "issue_number": 122,
  "title": "[BUG] Dashboard does not show up claude code output",
  "issue_url": "https://github.com/omnara-ai/omnara/issues/122",
  "expected_links": {
    "fixed_by_commits": [],
    "associated_prs": [123],
    "associated_issues": []
  },
  "linking_patterns": ["explicit"],
  "primary_evidence": {
    "pr_body_contains": "Fixes #122",
    "reference_type": "fixes",
    "explicit": true
  },
  "link_quality": "high",
  "difficulty": "easy",
  "notes": "Clear explicit reference in PR body - should be detected with high confidence",
  "expected_confidence": 0.95,
  "should_detect": true,
  "github_verification": {
    "issue_state": "closed",
    "pr_state": "merged",
    "verified_manually": false
  }
}
```

### Example: Issue #206 (True Negative)

```json
{
  "issue_number": 206,
  "title": "[Proposal] Rename Omnara",
  "issue_url": "https://github.com/omnara-ai/omnara/issues/206",
  "expected_links": {
    "fixed_by_commits": [],
    "associated_prs": [],
    "associated_issues": []
  },
  "linking_patterns": ["none"],
  "primary_evidence": {
    "close_reason": "completed",
    "issue_state": "closed",
    "note": "Proposal discussed but not implemented"
  },
  "link_quality": "n/a",
  "difficulty": "easy",
  "notes": "True negative - proposal closed without implementation",
  "expected_confidence": 0,
  "should_detect": false,
  "github_verification": {
    "issue_state": "closed",
    "pr_state": null,
    "verified_manually": false
  }
}
```

---

## Action Plan

### Phase 1: High Priority (This Week)
1. ✅ Manually verify Issues #122, #115, #53, #160, #165 on GitHub
2. ✅ Add to `omnara_ground_truth.json`
3. ✅ Run backtesting: `./bin/backtest --ground-truth test_data/omnara_ground_truth.json`
4. ✅ Verify metrics: Precision ≥ 98%, Recall ≥ 75%

**Expected Impact:** Validate explicit reference handling (Pattern 1)

### Phase 2: Medium Priority (Next Week)
1. ✅ Manually verify Issues #206, #178, #76 on GitHub
2. ✅ Add 2-3 confirmed true negatives
3. ✅ Run backtesting
4. ✅ Verify precision remains 100%

**Expected Impact:** Strengthen precision validation

### Phase 3: Low Priority (As Needed)
1. ✅ Add temporal edge cases (#62, #164, #147)
2. ✅ Test complex scenarios (multiple PRs, incremental features)

**Expected Impact:** Validate edge case handling

### Phase 4: Future (After Pattern 3 Implementation)
1. ⏳ Implement Pattern 3 (Comment-Based Linking)
2. ⏳ Find comment-based test cases
3. ⏳ Expand ground truth with comment patterns

---

## Summary

### Recommended Expansion
| Phase | Issues | Focus | Expected Metrics |
|-------|--------|-------|------------------|
| Current | 6 | Temporal + true negatives | P: 100%, R: 75% ✅ |
| Phase 1 | +5 (→11) | Explicit references | P: 100%, R: 75-80% |
| Phase 2 | +3 (→14) | More true negatives | P: 100%, R: 75-80% |
| Phase 3 | +3 (→17) | Edge cases | P: 98-100%, R: 80% |
| **Total** | **17** | **Comprehensive** | **P: 98-100%, R: 80%** |

### Next Steps
1. **Review this document** and confirm recommended issues
2. **I can manually verify** the top 5 issues on GitHub for you
3. **Create expanded ground truth JSON** with new cases
4. **Run backtesting** to validate performance at scale

Would you like me to:
- **Option A:** Manually verify the top 5 explicit reference cases on GitHub and create the expanded ground truth JSON?
- **Option B:** Provide you with a script to auto-fetch issue/PR details from GitHub API?
- **Option C:** Both - I verify, you review, then we commit the expanded ground truth?
