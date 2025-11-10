package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/metrics"
)

// CLQSInfo holds CLQS confidence information
type CLQSInfo struct {
	Score              int
	Grade              string
	Rank               string
	LinkCoverage       int
	ConfidenceQuality  int
	EvidenceDiversity  int
	TemporalPrecision  int
	SemanticStrength   int
}

// KickoffPromptBuilder creates the initial prompt for agent-based risk investigation
// Bridges FileResolver output with Agent exploration
// Reference: dev_docs/mvp/AGENT_KICKOFF_PROMPT_DESIGN.md
type KickoffPromptBuilder struct {
	fileChanges []FileChangeContext
	clqsScore   *CLQSInfo // Optional CLQS confidence information
}

// FileChangeContext contains all metadata for a single file change
type FileChangeContext struct {
	// From git status
	FilePath     string
	ChangeStatus string // "MODIFIED", "ADDED", "DELETED", "RENAMED"
	LinesAdded   int
	LinesDeleted int

	// From FileResolver
	ResolutionMatches    []git.FileMatch
	ResolutionMethod     string
	ResolutionConfidence float64

	// From git diff
	DiffSummary string // Truncated diff for context

	// From Phase 1 metrics
	CouplingScore     float64
	CoChangeFrequency float64
	IncidentCount     int
	ChurnScore        float64
	OwnerEmail        string
	LastModified      time.Time
}

// NewKickoffPromptBuilder creates a new prompt builder
func NewKickoffPromptBuilder(fileChanges []FileChangeContext) *KickoffPromptBuilder {
	return &KickoffPromptBuilder{
		fileChanges: fileChanges,
		clqsScore:   nil,
	}
}

// WithCLQS adds CLQS confidence information to the prompt
func (b *KickoffPromptBuilder) WithCLQS(clqs *CLQSInfo) *KickoffPromptBuilder {
	b.clqsScore = clqs
	return b
}

// BuildKickoffPrompt creates the complete prompt for agent investigation
func (b *KickoffPromptBuilder) BuildKickoffPrompt() string {
	var prompt strings.Builder

	// Part 1: System prompt (investigation principles)
	prompt.WriteString(b.buildSystemPrompt())
	prompt.WriteString("\n\n---\n\n")

	// Part 1.5: CLQS context (if available)
	if b.clqsScore != nil {
		prompt.WriteString(b.buildCLQSContext())
		prompt.WriteString("\n\n---\n\n")
	}

	// Part 1.6: Phase 1 Quantitative Assessment Summary
	prompt.WriteString(b.buildPhase1Summary())
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

// buildSystemPrompt creates the system prompt with incident-focused language
func (b *KickoffPromptBuilder) buildSystemPrompt() string {
	return `You are a risk assessment investigator analyzing code changes before commit.

Your goal: Determine if these changes are likely to cause production incidents.

Your focus: **Incident prevention**, not code quality

**CRITICAL UNDERSTANDING:**
You are analyzing a PROPOSED CHANGE that has NOT been committed yet.
- The user is making these changes RIGHT NOW in their working directory
- If you find past incidents, check if the PROPOSED change matches the BUGGY pattern
- If the proposed change matches a past bug pattern, risk is HIGH (potential regression)
- A revert PR in history means the bug was fixed, but if the user is RE-INTRODUCING the same pattern, that's HIGH risk!

**BACKWARDS LOGIC TRAP - AVOID THIS:**
❌ WRONG: "The bug was reverted, so current code is safe → risk is LOW"
✅ CORRECT: "The bug was reverted, but the user's proposed change matches that buggy pattern → risk is HIGH"

You are NOT responsible for:
- Code style violations
- Security vulnerabilities
- General bug detection
- Linting issues
- Best practices enforcement

You ARE responsible for:
- Incident history: Has this code caused production problems before?
- Pattern matching: Does the PROPOSED change match patterns from past incidents?
- Ownership gaps: Is the code owner still active and knowledgeable?
- Co-change patterns: Were related files that usually change together updated?
- Blast radius: How many downstream files depend on this change?

Your tools:
- get_incidents_with_context: Get full incident history with issue titles, bodies, confidence scores, and author roles
- get_ownership_timeline: Get developer ownership history with activity status
- get_cochange_with_explanations: Get files that frequently change together WITH sample commit messages explaining why
- get_blast_radius_analysis: Get downstream files that depend on changed code with impact analysis
- get_commit_patch: Read code changes from past incidents to compare patterns
- query_recent_commits: See recent modification patterns and ownership trends
- finish_investigation: Return final risk assessment with incident-focused reasoning

Investigation principles:
1. Start with incident history (has this code caused production problems before?)
2. **Pattern comparison:** If incidents found, compare the PROPOSED change with past buggy commits
   - Look for similar code patterns, similar change locations, similar modification types
   - If the proposed change matches a past bug → HIGH risk (regression alert!)
3. Check ownership (is the owner still active? do they know this code's history?)
4. Verify co-changes (if file X changes, does file Y usually change too?)
5. Assess blast radius (how many files depend on this change?)

**Risk Logic Rules:**
- If proposed change matches a pattern from a past incident → **HIGH risk** (even if that bug was fixed!)
- If file has incident history but proposed change is different → MEDIUM risk (risky file, but new pattern)
- If file has no incident history → LOW risk (unless other factors like stale ownership apply)

**CRITICAL OUTPUT FORMATTING RULES:**

When reporting incident history, you MUST list specific incidents with full details:
- Show top 3-5 most relevant incidents (highest confidence scores first)
- For each incident include: issue number, title, and date
- Format: "• #123: [BUG] Description here (2025-01-15, confidence: 85%)"
- If NO incidents found, explicitly state: "No production incidents found in the last 180 days"
- NEVER use vague summaries like "multiple incidents" - always enumerate them

Example incident reporting:
"Incident History:
• #87: [BUG] Signing up with apple gives localhost link (2025-08-16, confidence: 83%)
• #94: [BUG] Not showing all options for edit file (2025-08-17, confidence: 83%)
• #122: [BUG] Dashboard does not show claude code output (2025-08-15, confidence: 80%)"

Risk escalation triggers:
- HIGH: Incident history + large change + stale ownership
- CRITICAL: Multiple incidents + incomplete co-change + high blast radius`
}

// buildFileSection creates the section for a single file change
func (b *KickoffPromptBuilder) buildFileSection(num int, change FileChangeContext) string {
	var section strings.Builder

	// Header
	section.WriteString(fmt.Sprintf("## File %d: %s\n", num, change.FilePath))
	section.WriteString(fmt.Sprintf("**Status:** %s\n", change.ChangeStatus))

	// Change size
	changeSize := b.classifyChangeSize(change.LinesAdded, change.LinesDeleted)
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
	section.WriteString(fmt.Sprintf("- Coupling: %d dependencies\n", b.couplingToDependencies(change.CouplingScore)))
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

// buildInvestigationGuidance creates dynamic guidance based on file contexts
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

	if focusNum == 1 {
		// No specific focus areas - provide general guidance
		guidance.WriteString("1. Review ownership and activity patterns\n")
		guidance.WriteString("2. Check for co-change partners that may need updates\n")
		guidance.WriteString("3. Assess blast radius and downstream impact\n")
	}

	guidance.WriteString("\nStart your investigation. Call tools in a logical order based on risk signals.\n\n")

	guidance.WriteString("**OUTPUT FORMATTING REQUIREMENTS:**\n\n")
	guidance.WriteString("Structure your final assessment with these sections:\n\n")
	guidance.WriteString("1. **Incident History** - List top 3-5 specific incidents with details:\n")
	guidance.WriteString("   • #123: [BUG] Title here (2025-01-15, confidence: 85%)\n")
	guidance.WriteString("   • State \"No incidents found\" if none exist\n\n")
	guidance.WriteString("2. **Ownership Status** - List current owners with activity:\n")
	guidance.WriteString("   • user@email.com: 15 commits (last activity: 2 days ago) ✓ Active\n")
	guidance.WriteString("   • old@email.com: 3 commits (last activity: 180 days ago) ⚠️ Stale\n\n")
	guidance.WriteString("3. **Co-change Partners** - ENUMERATE specific files that change together:\n")
	guidance.WriteString("   • file/path.py (75% co-change frequency, 12 co-changes)\n")
	guidance.WriteString("   • another/file.tsx (60% co-change frequency, 8 co-changes)\n")
	guidance.WriteString("   • If get_cochange_with_explanations returns data, you MUST list specific files\n")
	guidance.WriteString("   • State \"No co-change partners found\" ONLY if the tool returns empty results\n")
	guidance.WriteString("   • NEVER say \"error retrieving\" unless the tool actually failed with an error\n\n")
	guidance.WriteString("4. **Blast Radius** - List downstream dependencies:\n")
	guidance.WriteString("   • 5 files depend on this change\n")
	guidance.WriteString("   • State \"No downstream dependencies\" if none\n\n")
	guidance.WriteString("5. **Risk Summary** - Synthesize all findings into clear risk assessment\n\n")

	guidance.WriteString("Note: This assessment focuses on incident risk. Run additional style, security, and linting checks as part of your normal development workflow.\n\n")
	guidance.WriteString("**CRITICAL:** When you have gathered sufficient evidence (typically 2-5 tool calls), you MUST call `finish_investigation` with your final risk assessment. Do NOT respond with text - use the `finish_investigation` tool to complete your investigation.\n")

	return guidance.String()
}

// buildPhase1Summary creates a summary of Phase 1 quantitative findings
// Reference: YC_DEMO_GAP_ANALYSIS.md Bug 3.4 - Phase 1/Phase 2 alignment
func (b *KickoffPromptBuilder) buildPhase1Summary() string {
	var summary strings.Builder

	summary.WriteString("# Phase 1 Quantitative Assessment\n\n")
	summary.WriteString("**IMPORTANT:** Phase 1 has already analyzed these changes using quantitative metrics.\n")
	summary.WriteString("Your job is to investigate WHY Phase 1 flagged these changes, not to contradict it.\n\n")

	// Aggregate Phase 1 findings across all files
	hasHighCoupling := false
	hasHighCoChange := false
	hasIncidents := false
	highestRisk := ""

	for _, change := range b.fileChanges {
		if change.CouplingScore > 0.5 {
			hasHighCoupling = true
		}
		if change.CoChangeFrequency > 0.5 {
			hasHighCoChange = true
		}
		if change.IncidentCount > 0 {
			hasIncidents = true
		}
	}

	summary.WriteString("**Phase 1 Findings:**\n")

	if hasIncidents {
		summary.WriteString("- ⚠️  **Incident history detected** - File(s) have caused production problems before\n")
		highestRisk = "HIGH (incident history)"
	}

	if hasHighCoChange {
		summary.WriteString("- ⚠️  **High co-change frequency** - File(s) frequently change with other files\n")
		if highestRisk == "" {
			highestRisk = "MEDIUM (co-change risk)"
		}
	}

	if hasHighCoupling {
		summary.WriteString("- ⚠️  **High structural coupling** - File(s) have many dependencies\n")
		if highestRisk == "" {
			highestRisk = "MEDIUM (coupling risk)"
		}
	}

	if highestRisk == "" {
		summary.WriteString("- ✓ No major quantitative risk signals detected\n")
		highestRisk = "LOW (metrics look clean)"
	}

	summary.WriteString(fmt.Sprintf("\n**Phase 1 Risk Assessment:** %s\n\n", highestRisk))
	summary.WriteString("**Your Task:** Investigate WHY Phase 1 flagged these risks. ")
	summary.WriteString("Look for:\n")
	summary.WriteString("- What incidents occurred in the past?\n")
	summary.WriteString("- Why do these files change together?\n")
	summary.WriteString("- Who owns this code and are they still active?\n")
	summary.WriteString("- Does the proposed change match past incident patterns?\n")

	return summary.String()
}

// buildCLQSContext adds data quality context from CLQS scores
func (b *KickoffPromptBuilder) buildCLQSContext() string {
	var clqs strings.Builder

	clqs.WriteString("# Data Quality Context (CLQS)\n\n")
	clqs.WriteString(fmt.Sprintf("This repository has a **CLQS Score of %d (%s - %s)**\n\n",
		b.clqsScore.Score, b.clqsScore.Grade, b.clqsScore.Rank))

	clqs.WriteString("**What this means:**\n")
	clqs.WriteString(fmt.Sprintf("- Link Coverage: %d/100 (How many issues are linked to PRs)\n", b.clqsScore.LinkCoverage))
	clqs.WriteString(fmt.Sprintf("- Confidence Quality: %d/100 (Strength of issue-PR links)\n", b.clqsScore.ConfidenceQuality))
	clqs.WriteString(fmt.Sprintf("- Evidence Diversity: %d/100 (Variety of linking signals)\n", b.clqsScore.EvidenceDiversity))
	clqs.WriteString(fmt.Sprintf("- Temporal Precision: %d/100 (Timing alignment of fixes)\n", b.clqsScore.TemporalPrecision))
	clqs.WriteString(fmt.Sprintf("- Semantic Strength: %d/100 (Quality of commit messages)\n", b.clqsScore.SemanticStrength))

	clqs.WriteString("\n**Impact on your investigation:**\n")

	// Provide guidance based on CLQS grade
	switch b.clqsScore.Grade {
	case "A", "B":
		clqs.WriteString("- ✅ High-quality incident data - trust the historical patterns you find\n")
		clqs.WriteString("- ✅ Issue links are well-documented - follow the evidence trail\n")
		clqs.WriteString("- ✅ Commit messages are informative - use them for context\n")
	case "C":
		clqs.WriteString("- ⚠️  Moderate data quality - verify critical findings with multiple signals\n")
		clqs.WriteString("- ⚠️  Some incident links may be missing - don't rely on absence of evidence\n")
		clqs.WriteString("- ⚠️  Cross-reference commit messages with other evidence\n")
	case "D", "F":
		clqs.WriteString("- ⚠️  Limited incident data - focus more on structural risks (coupling, complexity)\n")
		clqs.WriteString("- ⚠️  Incident history may be incomplete - absence of incidents != low risk\n")
		clqs.WriteString("- ⚠️  Rely more on code patterns and dependencies than historical data\n")
	}

	return clqs.String()
}

// Helper functions

// classifyChangeSize determines the size category of a change
func (b *KickoffPromptBuilder) classifyChangeSize(linesAdded, linesDeleted int) string {
	totalLines := linesAdded + linesDeleted
	if totalLines > 100 {
		return "LARGE"
	} else if totalLines > 30 {
		return "MEDIUM"
	}
	return "SMALL"
}

// couplingToDependencies converts coupling score to approximate dependency count
func (b *KickoffPromptBuilder) couplingToDependencies(couplingScore float64) int {
	// Approximate mapping: score 0.0-1.0 -> 0-20 dependencies
	return int(couplingScore * 20)
}

// truncateDiff truncates diff to specified character count
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

// FromPhase1Result creates FileChangeContext from Phase 1 result
// Convenience method for integration with existing code
func FromPhase1Result(
	filePath string,
	changeStatus string,
	linesAdded, linesDeleted int,
	matches []git.FileMatch,
	diffSummary string,
	phase1Result *metrics.Phase1Result,
) FileChangeContext {
	ctx := FileChangeContext{
		FilePath:         filePath,
		ChangeStatus:     changeStatus,
		LinesAdded:       linesAdded,
		LinesDeleted:     linesDeleted,
		ResolutionMatches: matches,
		DiffSummary:      diffSummary,
	}

	// Extract resolution metadata from first match
	if len(matches) > 0 {
		ctx.ResolutionMethod = matches[0].Method
		ctx.ResolutionConfidence = matches[0].Confidence
	}

	// Extract Phase 1 metrics if available
	if phase1Result != nil {
		if phase1Result.Coupling != nil {
			// Normalize count to 0-1 score (max 20 dependencies = 1.0)
			ctx.CouplingScore = float64(phase1Result.Coupling.Count) / 20.0
			if ctx.CouplingScore > 1.0 {
				ctx.CouplingScore = 1.0
			}
		}

		if phase1Result.CoChange != nil {
			ctx.CoChangeFrequency = phase1Result.CoChange.MaxFrequency
		}

		// Note: Phase1Result doesn't currently have Incidents, Churn, or Ownership fields
		// These would need to be passed separately or added to Phase1Result in the future
		// For now, these fields remain at their zero values
	}

	return ctx
}
