# Agent Kickoff Prompt - Competitive Positioning Alignment

**Date:** 2025-10-28
**Status:** ✅ UPDATED
**Purpose:** Align agent prompt design with competitive positioning strategy

---

## Changes Made

Updated [AGENT_KICKOFF_PROMPT_DESIGN.md](./AGENT_KICKOFF_PROMPT_DESIGN.md) to reinforce our **complementary positioning** strategy from [competitive_positioning_and_niche.md](./competitive_positioning_and_niche.md).

---

## Key Updates

### 1. **System Prompt: Added Explicit Scope Definition**

**Added:**
```markdown
Your focus: **Incident prevention**, not code quality

You are NOT responsible for:
- Code style violations
- Security vulnerabilities
- General bug detection
- Linting issues
- Best practices enforcement

You ARE responsible for:
- Incident history: Has this code caused production problems before?
- Ownership gaps: Is the code owner still active and knowledgeable?
- Co-change patterns: Were related files that usually change together updated?
- Blast radius: How many downstream files depend on this change?
- Pattern matching: Is this similar to changes that caused past incidents?
```

**Why:** Makes it crystal clear that we're **not competing** with code quality tools. Agent stays focused on incident prevention.

---

### 2. **Tool Descriptions: Incident-Focused Language**

**Updated tool descriptions to emphasize incident risk:**

**Before:**
```
query_ownership: Find who owns a file based on commit history
```

**After:**
```
query_ownership: Find if code owner is still active and knowledgeable.
Use this to detect stale ownership (incident risk factor).
```

**Why:** Every tool description now explicitly connects to **incident prevention**, not general code analysis.

---

### 3. **Recommendations: Added Workflow Context**

**Updated final assessment format:**

```markdown
Recommendations:
1. Request security review from alice@example.com
2. Add integration tests for authentication flow
3. Consider manual code review before committing

Note: This assessment focuses on incident risk.
Run additional code quality and security checks as part of your normal development workflow.
```

**Why:** Reminds users we're **one part** of their workflow, not a replacement for all checks.

---

### 4. **Conclusion: Added Strategic Positioning Note**

**Added to conclusion section:**

```markdown
### Strategic Positioning Note

This agent focuses **exclusively** on incident prevention, not code quality:
- ✅ **What we assess:** Will this cause a production incident?
- ❌ **What we don't assess:** Is this code well-written?

This makes us **complementary** to existing tools in the developer workflow:
1. AI code generation (Cursor, Copilot) - writes code fast
2. **CodeRisk** (us) - checks incident risk before commit
3. Version control (Git) - commits changes
4. Code review tools - check code quality before merge
```

**Why:** Reinforces our **strategic positioning** as complementary, not competitive.

---

## Alignment Verification

### ✅ Competitive Positioning Requirements

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| **Focus on incident prevention** | ✅ | System prompt explicitly states "incident prevention, not code quality" |
| **Temporal intelligence** | ✅ | All tools query historical data (incidents, ownership, co-changes) |
| **Pre-commit stage** | ✅ | Prompt uses git diff/status, not PR diffs |
| **Complementary positioning** | ✅ | Added note about workflow integration |
| **NOT code quality checks** | ✅ | Explicit "NOT responsible for" list in system prompt |
| **NOT security scanning** | ✅ | Removed from agent scope |
| **NOT PR review** | ✅ | Focus on pre-commit, not post-commit |

---

## What Stayed the Same

### ✅ Core Architecture (No Changes Needed)

- File resolution strategy (already incident-focused)
- Phase 1 baseline metrics (already temporal intelligence)
- Agent tool set (already graph-based historical queries)
- Prompt template structure (already well-designed)
- Token usage optimization (already efficient)

**Why no changes:** The technical design was already aligned. We only needed to make the **positioning explicit** in the language.

---

## Impact on Implementation

### No Code Changes Required

The alignment updates are **purely prompt engineering**:
- ✅ No changes to FileResolver
- ✅ No changes to KickoffPromptBuilder structure
- ✅ No changes to tool implementations
- ✅ No changes to agent loop

### Only Prompt Text Changes

When implementing `KickoffPromptBuilder`, use the updated system prompt from the document. That's it!

---

## Developer Communication

### Message to Users

When CodeRisk displays results, it should communicate:

```
🔍 Incident Risk Assessment

Risk Level: HIGH (confidence: 90%)

Why: Similar change caused incident #122 (login bypass)

Recommendations:
1. Request security review
2. Add integration tests
3. Run additional code quality checks

Note: This assessment focuses on incident risk only.
Continue with your normal code review process.
```

**Key phrase:** "This assessment focuses on incident risk only"

This sets expectations that we're **one check** in their workflow, not the only check.

---

## Next Steps

### For Implementation

1. ✅ Use updated system prompt from AGENT_KICKOFF_PROMPT_DESIGN.md
2. ✅ Include "NOT responsible for" section in all agent contexts
3. ✅ Add workflow context note to final recommendations
4. ✅ Test that agent doesn't suggest code quality improvements

### For Messaging

When launching:
- ✅ Position as "pre-commit incident checker"
- ✅ NOT "code review tool"
- ✅ Emphasize complementary nature
- ✅ Show it in workflow: Cursor → CodeRisk → Git → Code Review

---

## Conclusion

The agent kickoff prompt design is now **fully aligned** with competitive positioning:

- **Focuses on:** Incident prevention through temporal intelligence
- **Avoids:** Code quality, security scanning, PR review automation
- **Positions as:** Complementary tool in developer workflow
- **Targets:** Pre-commit stage (before Git commit)

All updates are **non-breaking** and require no code changes - just using the updated prompt language when building `KickoffPromptBuilder`.

**Ready for implementation!** ✅
