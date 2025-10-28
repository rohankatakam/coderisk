# Agent Kickoff Prompt Design

**Date:** 2025-10-28
**Purpose:** Design the initial prompt for the agent-based query execution system
**Integration:** Bridges FileResolver output with Agent investigation

---

## Problem Statement

The agent needs:
1. **Entry points**: Which graph nodes to start from
2. **Context**: What changed, how much, and why it matters
3. **Guidance**: Which queries are likely most valuable

Without this, the agent would:
- ❌ Have to search for files in the graph (slow, error-prone)
- ❌ Not know which changes are risky vs trivial
- ❌ Waste tool calls on irrelevant queries

---

## Solution: Structured Kickoff Prompt

Populate the agent's initial context with:

### Part 1: Investigation Context (System Prompt)
```markdown
You are a risk assessment investigator analyzing code changes before commit.

Your goal: Determine if these changes are likely to cause production incidents.

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

Your tools:
- query_ownership: Find if code owner is still active (stale ownership = incident risk)
- query_cochange_partners: Find files that usually change together (incomplete changes = incident risk)
- query_incident_history: Find production incidents that originated from this code
- query_blast_radius: Find downstream files that depend on changed code
- get_commit_patch: Read code changes from past incidents to compare patterns
- query_recent_commits: See recent modification patterns and ownership trends
- finish_investigation: Return final risk assessment with incident-focused reasoning

Investigation principles:
1. Start with incident history (has this code caused production problems before?)
2. Check ownership (is the owner still active? do they know this code's history?)
3. Verify co-changes (if file X changes, does file Y usually change too?)
4. Assess blast radius (how many files depend on this change?)
5. Use commit patches to understand if current changes match past incident patterns

Risk escalation triggers:
- HIGH: Incident history + large change + stale ownership
- CRITICAL: Multiple incidents + incomplete co-change + high blast radius
```

### Part 2: File Resolution Results (User Prompt)
```markdown
# Changes to Investigate

## File 1: src/auth/login.py
**Status:** MODIFIED
**Changes:** +150 lines, -20 lines (LARGE CHANGE)
**Resolution:**
  - Current path: src/auth/login.py
  - Historical paths found in graph:
    * auth/login.py (confidence: 95%, method: git-follow)
    * backend/login.py (confidence: 95%, method: git-follow)
  - **Entry points for queries:** Use paths ["src/auth/login.py", "auth/login.py", "backend/login.py"]

**Git Diff Summary:**
```diff
@@ -45,6 +45,12 @@ def validate_password(user, password):
-    return user.password == password
+    return bcrypt.checkpw(password, user.password_hash)
+
+@@ -120,0 +126,15 @@ def login_user(email, password):
+def create_session(user_id):
+    session_token = secrets.token_urlsafe(32)
+    # ... new session logic
```

**Baseline Metrics (Phase 1):**
- Coupling: 8 dependencies
- Co-change frequency: 0.65 (frequently changes with session.py)
- Recent incidents: 1 (in last 90 days)
- Churn score: 0.4 (moderate recent activity)

---

## File 2: src/auth/session.py
**Status:** MODIFIED
**Changes:** +30 lines, -5 lines
**Resolution:**
  - Current path: src/auth/session.py
  - Historical paths found in graph:
    * auth/session.py (confidence: 95%, method: git-follow)
  - **Entry points for queries:** Use paths ["src/auth/session.py", "auth/session.py"]

**Git Diff Summary:**
```diff
@@ -78,3 +78,8 @@ def validate_session(token):
-    return session.user_id
+    if session.expired:
+        raise SessionExpiredError()
+    return session.user_id
```

**Baseline Metrics (Phase 1):**
- Coupling: 4 dependencies
- Co-change frequency: 0.65 (frequently changes with login.py)
- Recent incidents: 0
- Churn score: 0.3 (low recent activity)

---

## File 3: tests/test_auth.py
**Status:** NEW FILE
**Changes:** +200 lines, -0 lines
**Resolution:**
  - No historical data (new file)
  - **Entry points for queries:** None (skip graph queries)

**Notes:** New test file - low risk by default

---

# Your Task

Investigate the changes above and determine the overall risk level.

**Focus areas:**
1. login.py has a LARGE change (+150 lines) - what kind of changes are these?
2. login.py has incident history - was it similar to current changes?
3. login.py and session.py have high co-change frequency (0.65) - were both updated appropriately?
4. login.py was renamed twice - does the current owner know the full history?

Start your investigation. Call tools in a logical order based on risk signals.
```

---

## Prompt Template Structure

```go
package agent

import (
	"fmt"
	"strings"
)

// KickoffPromptBuilder creates the initial prompt for the agent
type KickoffPromptBuilder struct {
	fileChanges      []FileChangeContext
	systemPrompt     string
	riskThresholds   RiskThresholds
}

// FileChangeContext contains all metadata for a single file
type FileChangeContext struct {
	// From git status
	FilePath      string
	ChangeStatus  string // "MODIFIED", "ADDED", "DELETED", "RENAMED"
	LinesAdded    int
	LinesDeleted  int

	// From FileResolver
	ResolutionMatches []FileMatch
	ResolutionMethod  string
	ResolutionConfidence float64

	// From git diff
	DiffSummary      string // Truncated diff for context

	// From Phase 1 metrics
	CouplingScore     float64
	CoChangeFrequency float64
	IncidentCount     int
	ChurnScore        float64
	OwnerEmail        string
	LastModified      time.Time
}

// BuildKickoffPrompt creates the complete prompt
func (b *KickoffPromptBuilder) BuildKickoffPrompt() string {
	var prompt strings.Builder

	// Part 1: System prompt (investigation principles)
	prompt.WriteString(b.buildSystemPrompt())
	prompt.WriteString("\n\n---\n\n")

	// Part 2: File changes with resolution context
	prompt.WriteString("# Changes to Investigate\n\n")
	for i, change := range b.fileChanges {
		prompt.WriteString(b.buildFileSection(i+1, change))
		prompt.WriteString("\n---\n\n")
	}

	// Part 3: Investigation guidance
	prompt.WriteString(b.buildInvestigationGuidance())

	return prompt.String()
}

func (b *KickoffPromptBuilder) buildFileSection(num int, change FileChangeContext) string {
	var section strings.Builder

	// Header
	section.WriteString(fmt.Sprintf("## File %d: %s\n", num, change.FilePath))
	section.WriteString(fmt.Sprintf("**Status:** %s\n", change.ChangeStatus))

	// Change size
	changeSize := "SMALL"
	totalLines := change.LinesAdded + change.LinesDeleted
	if totalLines > 100 {
		changeSize = "LARGE"
	} else if totalLines > 30 {
		changeSize = "MEDIUM"
	}
	section.WriteString(fmt.Sprintf("**Changes:** +%d lines, -%d lines (%s CHANGE)\n",
		change.LinesAdded, change.LinesDeleted, changeSize))

	// Resolution results
	section.WriteString("**Resolution:**\n")
	if len(change.ResolutionMatches) == 0 {
		section.WriteString("  - No historical data (new file)\n")
		section.WriteString("  - **Entry points for queries:** None (skip graph queries)\n")
	} else {
		section.WriteString(fmt.Sprintf("  - Current path: %s\n", change.FilePath))
		section.WriteString("  - Historical paths found in graph:\n")

		var entryPoints []string
		entryPoints = append(entryPoints, change.FilePath)

		for _, match := range change.ResolutionMatches {
			section.WriteString(fmt.Sprintf("    * %s (confidence: %.0f%%, method: %s)\n",
				match.HistoricalPath, match.Confidence*100, match.Method))
			entryPoints = append(entryPoints, match.HistoricalPath)
		}

		section.WriteString(fmt.Sprintf("  - **Entry points for queries:** Use paths %v\n",
			entryPoints))
	}

	// Diff summary (truncated)
	if change.DiffSummary != "" {
		section.WriteString("\n**Git Diff Summary:**\n")
		section.WriteString("```diff\n")
		section.WriteString(truncateDiff(change.DiffSummary, 300))
		section.WriteString("\n```\n")
	}

	// Phase 1 baseline metrics
	section.WriteString("\n**Baseline Metrics (Phase 1):**\n")
	section.WriteString(fmt.Sprintf("- Coupling: %d dependencies\n", int(change.CouplingScore*20)))
	section.WriteString(fmt.Sprintf("- Co-change frequency: %.2f", change.CoChangeFrequency))

	if change.CoChangeFrequency > 0.5 {
		section.WriteString(" (frequently changes with other files)")
	}
	section.WriteString("\n")

	section.WriteString(fmt.Sprintf("- Recent incidents: %d", change.IncidentCount))
	if change.IncidentCount > 0 {
		section.WriteString(" ⚠️  (in last 90 days)")
	}
	section.WriteString("\n")

	section.WriteString(fmt.Sprintf("- Churn score: %.1f ", change.ChurnScore))
	if change.ChurnScore > 0.7 {
		section.WriteString("(high recent activity)")
	} else if change.ChurnScore > 0.3 {
		section.WriteString("(moderate recent activity)")
	} else {
		section.WriteString("(low recent activity)")
	}
	section.WriteString("\n")

	// Ownership info
	if change.OwnerEmail != "" {
		daysSinceModified := int(time.Since(change.LastModified).Hours() / 24)
		section.WriteString(fmt.Sprintf("- Owner: %s (last modified %d days ago)\n",
			change.OwnerEmail, daysSinceModified))
	}

	return section.String()
}

func (b *KickoffPromptBuilder) buildInvestigationGuidance() string {
	var guidance strings.Builder

	guidance.WriteString("# Your Task\n\n")
	guidance.WriteString("Investigate the changes above and determine the overall risk level.\n\n")

	// Generate dynamic focus areas based on file contexts
	guidance.WriteString("**Focus areas:**\n")

	focusNum := 1
	for _, change := range b.fileChanges {
		totalLines := change.LinesAdded + change.LinesDeleted

		// Large changes
		if totalLines > 100 {
			guidance.WriteString(fmt.Sprintf("%d. %s has a LARGE change (+%d lines) - what kind of changes are these?\n",
				focusNum, change.FilePath, change.LinesAdded))
			focusNum++
		}

		// Incident history
		if change.IncidentCount > 0 {
			guidance.WriteString(fmt.Sprintf("%d. %s has incident history - was it similar to current changes?\n",
				focusNum, change.FilePath))
			focusNum++
		}

		// High co-change
		if change.CoChangeFrequency > 0.5 {
			guidance.WriteString(fmt.Sprintf("%d. %s has high co-change frequency (%.2f) - were related files updated?\n",
				focusNum, change.FilePath, change.CoChangeFrequency))
			focusNum++
		}

		// Renamed files
		if len(change.ResolutionMatches) > 1 {
			guidance.WriteString(fmt.Sprintf("%d. %s was renamed %d times - does the current owner know the full history?\n",
				focusNum, change.FilePath, len(change.ResolutionMatches)))
			focusNum++
		}
	}

	guidance.WriteString("\nStart your investigation. Call tools in a logical order based on risk signals.\n")

	return guidance.String()
}

func truncateDiff(diff string, maxChars int) string {
	if len(diff) <= maxChars {
		return diff
	}

	// Try to truncate at a newline
	truncated := diff[:maxChars]
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > maxChars/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n... (truncated)"
}
```

---

## Example: Complete Kickoff Prompt

### Scenario: Developer modifies auth system

**Input:**
- 2 files changed: `src/auth/login.py` (+150/-20), `src/auth/session.py` (+30/-5)
- FileResolver found historical paths for both
- Phase 1 shows high co-change frequency between them

**Generated Prompt:**

```markdown
You are a risk assessment investigator analyzing code changes before commit.

Your goal: Determine if these changes are likely to cause production incidents.

[... system prompt ...]

---

# Changes to Investigate

## File 1: src/auth/login.py
**Status:** MODIFIED
**Changes:** +150 lines, -20 lines (LARGE CHANGE)
**Resolution:**
  - Current path: src/auth/login.py
  - Historical paths found in graph:
    * auth/login.py (confidence: 95%, method: git-follow)
    * backend/login.py (confidence: 95%, method: git-follow)
  - **Entry points for queries:** Use paths ["src/auth/login.py", "auth/login.py", "backend/login.py"]

**Baseline Metrics:**
- Coupling: 8 dependencies
- Co-change frequency: 0.65 (frequently changes with session.py)
- Recent incidents: 1 ⚠️ (in last 90 days)

---

## File 2: src/auth/session.py
**Status:** MODIFIED
**Changes:** +30 lines, -5 lines (SMALL CHANGE)
**Resolution:**
  - Current path: src/auth/session.py
  - Historical paths: auth/session.py
  - **Entry points:** ["src/auth/session.py", "auth/session.py"]

**Baseline Metrics:**
- Co-change frequency: 0.65 (frequently changes with login.py)
- Recent incidents: 0

---

# Your Task

**Focus areas:**
1. login.py has a LARGE change (+150 lines) - what kind of changes are these?
2. login.py has incident history - was it similar to current changes?
3. login.py and session.py have high co-change (0.65) - were both updated appropriately?
4. login.py was renamed twice - does owner know full history?

Start your investigation.
```

**Agent's First Move:**
```json
{
  "tool": "query_incident_history",
  "parameters": {
    "file_paths": ["src/auth/login.py", "auth/login.py", "backend/login.py"],
    "days_back": 180
  }
}
```

Agent immediately:
1. ✅ Used the entry points we provided
2. ✅ Started with incident history (high-risk signal)
3. ✅ Queried all historical paths (won't miss data)

---

## Integration with crisk check

```go
// cmd/crisk/check.go

// After file resolution
for _, file := range files {
	matches := resolvedFilesMap[file]

	// Get git diff for this file
	diff, _ := git.GetFileDiff(file)
	linesAdded, linesDeleted := countDiffLines(diff)

	// Run Phase 1 to get baseline metrics
	phase1Result, _ := metrics.CalculatePhase1WithConfig(...)

	// Build file context for agent
	fileContext := agent.FileChangeContext{
		FilePath:             file,
		ChangeStatus:         "MODIFIED",
		LinesAdded:           linesAdded,
		LinesDeleted:         linesDeleted,
		ResolutionMatches:    matches,
		DiffSummary:          truncateDiff(diff, 500),
		CouplingScore:        phase1Result.Coupling.Score,
		CoChangeFrequency:    phase1Result.CoChange.MaxFrequency,
		IncidentCount:        phase1Result.Incidents.Count,
		ChurnScore:           phase1Result.Churn.Score,
		OwnerEmail:           phase1Result.Ownership.PrimaryOwner,
		LastModified:         phase1Result.Ownership.LastModified,
	}

	fileChanges = append(fileChanges, fileContext)
}

// Build kickoff prompt
promptBuilder := agent.NewKickoffPromptBuilder(fileChanges)
kickoffPrompt := promptBuilder.BuildKickoffPrompt()

// Create investigator with kickoff prompt
investigator := agent.NewRiskInvestigator(llmClient, graphClient, pgClient)
assessment, err := investigator.Investigate(ctx, kickoffPrompt)
```

---

## Benefits of This Approach

### 1. **No Wasted Tool Calls**
Agent doesn't need to:
- ❌ Search for files in the graph (we give entry points)
- ❌ Query git for diff stats (we include them)
- ❌ Run Phase 1 queries (we pre-compute them)

Instead, agent focuses on:
- ✅ Incident history (past problems)
- ✅ Commit patches (similar changes)
- ✅ Co-change verification (incomplete changes)

### 2. **Smarter Investigation Strategy**
Agent can reason:
```
"I see login.py has:
- Large change (+150 lines)
- Incident history (1 in 90 days)
- High co-change with session.py (0.65)
- Renamed twice (3 historical paths)

This is HIGH RISK. I should:
1. Check incident_history to see what broke before
2. Get commit_patch of that incident to compare
3. Verify session.py was updated (co-change partner)
4. Check ownership (do they know old paths?)"
```

### 3. **Rich Context = Better Confidence**
Agent can justify its assessment:
```
Risk: HIGH (confidence: 90%)

Reasoning:
- Similar change in commit abc123 caused incident #122 (login bypass)
- That commit modified bcrypt logic (current change also touches bcrypt)
- session.py WAS updated (good - co-change partner addressed)
- Owner alice@example.com knows this code (60% of commits)
- BUT: Last modified 2 months ago (stale knowledge risk)

Recommendations:
1. Request security review from alice@example.com
2. Add integration tests for authentication flow
3. Consider manual code review before committing

Note: This assessment focuses on incident risk. Run additional code quality and security checks as part of your normal development workflow.
```

### 4. **Prompt = Easy Tuning**
Instead of changing code, tune the prompt:
```diff
- "Start with incident history"
+ "Start with blast radius if >50 dependencies"

- "High risk = incident history + large change"
+ "High risk = incident history OR (large change AND stale ownership)"
```

---

## Performance Considerations

### Token Usage Optimization

**Full prompt example:**
- System prompt: ~500 tokens
- File context (each): ~300 tokens
- 3 files = ~1,400 tokens
- **Total: ~1,900 tokens** (well within 8k context limit)

**If checking 10 files:**
- System prompt: ~500 tokens
- File contexts: 10 × 300 = 3,000 tokens
- **Total: ~3,500 tokens** (still comfortable)

### Cost Optimization

Instead of including full diffs:
```go
// Truncate diff to most relevant parts
diff = extractFunctionChanges(diff) // Only show changed functions
diff = truncateDiff(diff, 300)      // Max 300 chars per file
```

Instead of including all metrics:
```go
// Only include Phase 1 metrics that exceed thresholds
if phase1Result.IncidentCount > 0 {
    context.IncidentCount = phase1Result.IncidentCount
}
```

---

## Next Steps: Implementation Order

### Phase 1: Basic Kickoff Prompt (This Week)
1. Create `KickoffPromptBuilder` in `internal/agent/`
2. Integrate with `check.go` after file resolution
3. Pass to existing `SimpleInvestigator`
4. Test with omnara repository

### Phase 2: Enhanced Context (Next Week)
1. Add git diff extraction and truncation
2. Include Phase 1 metrics in file context
3. Generate dynamic focus areas
4. Add risk signal detection

### Phase 3: Agent Loop (Week 3)
1. Implement full agent loop with tools
2. Add tool execution (query_ownership, etc.)
3. Test investigation quality
4. Tune prompts based on results

---

## Conclusion

**Yes, this is the right approach!**

By combining:
1. ✅ File resolution (entry points to graph)
2. ✅ Git metadata (change size, status)
3. ✅ Phase 1 metrics (baseline risk signals)
4. ✅ Prompt engineering (agent guidance)

You create a **powerful, flexible, and intelligent** investigation system that:
- Knows where to look (file resolution)
- Knows what to look for (Phase 1 metrics)
- Knows how to investigate (prompt guidance)
- Can adapt and learn (prompt tuning)

This bridges the gap perfectly between:
- **Static data** (file resolution, Phase 1)
- **Dynamic intelligence** (agent exploration)

### Strategic Positioning Note

This agent focuses **exclusively** on incident prevention, not code quality:
- ✅ **What we assess:** Will this cause a production incident?
- ❌ **What we don't assess:** Is this code well-written?

This makes us **complementary** to existing tools in the developer workflow:
1. AI code generation (Cursor, Copilot) - writes code fast
2. **CodeRisk** (us) - checks incident risk before commit
3. Version control (Git) - commits changes
4. Code review tools - check code quality before merge

By staying focused on our niche (temporal intelligence for incident prevention), we avoid competing with established tools and provide unique value.

Ready to implement? Let me know and I'll start building the `KickoffPromptBuilder`!
