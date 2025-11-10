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
	guidance.WriteString("Note: This assessment focuses on incident risk. Run additional style, security, and linting checks as part of your normal development workflow.\n\n")
	guidance.WriteString("**CRITICAL:** When you have gathered sufficient evidence (typically 2-5 tool calls), you MUST call `finish_investigation` with your final risk assessment. Do NOT respond with text - use the `finish_investigation` tool to complete your investigation.\n")

	return guidance.String()
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
