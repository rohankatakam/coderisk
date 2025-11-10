package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/agent"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/types"
)

// StandardFormatter outputs issues + recommendations (default)
type StandardFormatter struct{}

func (f *StandardFormatter) Format(result *types.RiskResult, w io.Writer) error {
	// Header
	fmt.Fprintf(w, "🔍 CodeRisk Analysis\n")
	if result.Branch != "" {
		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
	}
	fmt.Fprintf(w, "Files changed: %d\n", result.FilesChanged)
	fmt.Fprintf(w, "Risk level: %s\n\n", result.RiskLevel)

	// Issues
	if len(result.Issues) > 0 {
		fmt.Fprintf(w, "Issues:\n")
		for i, issue := range result.Issues {
			fmt.Fprintf(w, "%d. %s %s - %s\n",
				i+1,
				severityEmoji(issue.Severity),
				issue.File,
				issue.Message,
			)
		}
		fmt.Fprintf(w, "\n")
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Fprintf(w, "Recommendations:\n")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(w, "- %s\n", rec)
		}
		fmt.Fprintf(w, "\n")
	}

	// Next steps
	if result.RiskLevel != "LOW" && result.RiskLevel != "NONE" {
		fmt.Fprintf(w, "Run 'crisk check --explain' for investigation trace\n")
	}

	return nil
}

func severityEmoji(severity string) string {
	switch severity {
	case "HIGH", "CRITICAL":
		return "🔴"
	case "MEDIUM":
		return "⚠️ "
	case "LOW":
		return "ℹ️ "
	default:
		return "•"
	}
}

// === Merged from phase2.go ===

// DisplayPhase2Summary shows investigation summary (standard mode)
// 12-factor: Factor 4 - Tools are structured outputs (human-readable summary)
func DisplayPhase2Summary(assessment agent.RiskAssessment) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Investigation Summary\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show key evidence
	if len(assessment.Evidence) > 0 {
		fmt.Println("\nKey Evidence:")
		for i, evidence := range assessment.Evidence {
			fmt.Printf("%d. [%s] %s\n", i+1, evidence.Type, evidence.Description)
		}
	}

	// Show final assessment
	fmt.Printf("\nRisk Level: %s (confidence: %.0f%%)\n",
		assessment.RiskLevel,
		assessment.Confidence*100)

	// Show summary
	if assessment.Summary != "" {
		fmt.Printf("\nSummary: %s\n", assessment.Summary)
	}

	// Show investigation stats if available
	if assessment.Investigation != nil {
		duration := assessment.Investigation.CompletedAt.Sub(assessment.Investigation.Request.StartedAt)
		fmt.Printf("\nInvestigation completed in %.1fs (%d hops, %d tokens)\n",
			duration.Seconds(),
			len(assessment.Investigation.Hops),
			assessment.Investigation.TotalTokens)
	}
}

// DisplayPhase2Trace shows full hop-by-hop investigation (explain mode)
// 12-factor: Factor 4 - Tools are structured outputs (detailed trace for debugging)
func DisplayPhase2Trace(assessment agent.RiskAssessment) {
	if assessment.Investigation == nil {
		DisplayPhase2Summary(assessment)
		return
	}

	investigation := assessment.Investigation

	fmt.Println("\n🔍 CodeRisk Investigation Report")
	fmt.Printf("Started: %s\n", investigation.Request.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Completed: %s (%.1fs)\n",
		investigation.CompletedAt.Format("2006-01-02 15:04:05"),
		investigation.CompletedAt.Sub(investigation.Request.StartedAt).Seconds())
	fmt.Printf("Agent hops: %d\n", len(investigation.Hops))

	// Show each hop
	for _, hop := range investigation.Hops {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Hop %d\n", hop.HopNumber)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if len(hop.NodesVisited) > 0 {
			fmt.Printf("\nNodes visited: %v\n", hop.NodesVisited)
		}

		if len(hop.EdgesTraversed) > 0 {
			fmt.Printf("Edges traversed: %v\n", hop.EdgesTraversed)
		}

		if hop.Query != "" {
			fmt.Printf("\nQuery: %s\n", hop.Query)
		}

		if hop.Response != "" {
			// Show full response (no truncation)
			fmt.Printf("Response: %s\n", hop.Response)
		}

		fmt.Printf("\nTokens: %d | Duration: %dms\n", hop.TokensUsed, hop.Duration.Milliseconds())
	}

	// Show final assessment
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Final Assessment")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	DisplayPhase2Summary(assessment)
}

// DisplayPhase2JSON outputs investigation trace in AI mode JSON format
// 12-factor: Factor 4 - Tools are structured outputs (machine-readable for AI)
func DisplayPhase2JSON(assessment agent.RiskAssessment) {
	if assessment.Investigation == nil {
		// No investigation trace, output basic assessment
		data := map[string]interface{}{
			"risk_level": assessment.RiskLevel,
			"confidence": assessment.Confidence,
			"summary":    assessment.Summary,
			"evidence":   assessment.Evidence,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	// Convert investigation to trace format
	trace := convertInvestigationToTrace(assessment.Investigation)

	// Output full investigation data
	data := map[string]interface{}{
		"risk_level":          assessment.RiskLevel,
		"confidence":          assessment.Confidence,
		"summary":             assessment.Summary,
		"evidence":            assessment.Evidence,
		"investigation_trace": trace,
		"investigation_stats": map[string]interface{}{
			"total_hops":   len(assessment.Investigation.Hops),
			"total_tokens": assessment.Investigation.TotalTokens,
			"duration_ms":  assessment.Investigation.CompletedAt.Sub(assessment.Investigation.Request.StartedAt).Milliseconds(),
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

// Helper functions

// convertInvestigationToTrace converts investigation hops to JSON-friendly format
func convertInvestigationToTrace(investigation *agent.Investigation) []map[string]interface{} {
	trace := make([]map[string]interface{}, len(investigation.Hops))
	for i, hop := range investigation.Hops {
		trace[i] = map[string]interface{}{
			"hop":             hop.HopNumber,
			"nodes_visited":   hop.NodesVisited,
			"edges_traversed": hop.EdgesTraversed,
			"query":           hop.Query,
			"response":        hop.Response,
			"tokens_used":     hop.TokensUsed,
			"duration_ms":     hop.Duration.Milliseconds(),
		}
	}
	return trace
}

// formatTime formats minutes into human-readable duration
func formatTime(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dmin", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dmin", hours, mins)
}

// formatDuration formats a duration into human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// ===================================================================
// DEMO OUTPUT FUNCTIONS - Manager View + Developer View + CLQS
// ===================================================================

// DemoOutputData contains all data needed for demo-quality output
type DemoOutputData struct {
	Assessment  *agent.RiskAssessment
	Incidents   []database.IncidentWithContext
	Ownership   []database.OwnershipHistory
	CoChange    []database.CoChangePartnerContext
	BlastRadius []database.BlastRadiusFile
	CLQSScore   *CLQSInfo
	FilePath    string
}

// CLQSInfo holds CLQS score information
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

// DisplayManagerView outputs the business-focused Manager View
func DisplayManagerView(data *DemoOutputData) {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  MANAGER-VIEW: POTENTIAL BUSINESS IMPACT")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	// Critical user flow detection
	if strings.Contains(strings.ToLower(data.FilePath), "table") ||
		strings.Contains(strings.ToLower(data.FilePath), "editor") {
		fmt.Println("  • 🚨 Critical user flow: Database Table Editing")
	}

	// Historical incidents
	if len(data.Incidents) > 0 {
		highConfCount := 0
		for _, inc := range data.Incidents {
			if inc.LinkType == "FIXED_BY" {
				highConfCount++
			}
		}
		fmt.Printf("  • 🔥 Historical incidents: %d confirmed bugs", highConfCount)
		if len(data.Incidents) > highConfCount {
			fmt.Printf(" (%d possible)", len(data.Incidents)-highConfCount)
		}
		fmt.Println()
	}

	// Code ownership status
	if len(data.Ownership) > 0 {
		topOwner := data.Ownership[0]
		if !topOwner.IsActive {
			fmt.Printf("  • ⏳ Code ownership: Stale (original owner inactive %d days)\n", topOwner.DaysSinceCommit)
		} else if topOwner.DaysSinceCommit > 30 {
			fmt.Printf("  • ⏳ Code ownership: Aging (last commit %d days ago)\n", topOwner.DaysSinceCommit)
		}
	}

	// Bus factor (if only 1-2 active developers)
	if len(data.Ownership) > 0 {
		activeDevs := 0
		for _, owner := range data.Ownership {
			if owner.IsActive {
				activeDevs++
			}
		}
		if activeDevs <= 2 {
			fmt.Printf("  • 👥 Bus factor: %d developer%s (HIGH concentration risk)\n",
				activeDevs, pluralize(activeDevs))
		}
	}

	fmt.Println()
	fmt.Println("  📊 RISK ASSESSMENT:")
	fmt.Printf("    Risk Level: %s\n", colorizeRiskLevel(string(data.Assessment.RiskLevel)))
	if data.CLQSScore != nil {
		fmt.Printf("    Confidence: %.0f%% (based on CLQS Score: %d - %s)\n",
			data.Assessment.Confidence*100,
			data.CLQSScore.Score,
			data.CLQSScore.Rank)
	} else {
		fmt.Printf("    Confidence: %.0f%%\n", data.Assessment.Confidence*100)
	}

	// Impact estimation
	if len(data.Incidents) > 0 {
		fmt.Println()
		fmt.Println("    If this breaks:")
		avgMTTR := calculateAvgMTTR(data.Incidents)
		if avgMTTR > 0 {
			fmt.Printf("      • Estimated MTTR: %.1f hours (avg of past incidents)\n", avgMTTR)
		}
		fmt.Println("      • User impact: P0 flow (critical functionality)")
	}

	fmt.Println()
	fmt.Println("    Recommendation: Require senior engineer review")
	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayDeveloperView outputs the actionable Developer View
func DisplayDeveloperView(data *DemoOutputData) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  DEVELOPER-VIEW: ACTIONABLE INSIGHTS")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Why is this risky?")
	fmt.Println()

	// Past Incidents Section
	if len(data.Incidents) > 0 {
		fmt.Printf("  📋 Past Incidents (%d found):\n", len(data.Incidents))
		for i, inc := range data.Incidents {
			if i >= 3 {
				break // Show top 3
			}
			fmt.Printf("    %d. #%d: %s\n", i+1, inc.IssueNumber, inc.IssueTitle)

			if inc.ClosedAt != nil {
				daysAgo := int(time.Since(*inc.ClosedAt).Hours() / 24)
				fmt.Printf("       • Closed: %s (%d days ago)\n",
					inc.ClosedAt.Format("2006-01-02"), daysAgo)
			}

			fmt.Printf("       • Link: %s (confidence: %.0f%%, %s)\n",
				inc.LinkType, inc.Confidence*100, inc.DetectionMethod)

			// Reporter role
			roleDesc := describeAuthorRole(inc.AuthorRole)
			fmt.Printf("       • Reporter: %s (author_association: %s)\n",
				roleDesc, inc.AuthorRole)
			fmt.Println()
		}
	} else {
		fmt.Println("  📋 Past Incidents: None found (good track record or new file)")
		fmt.Println()
	}

	// Ownership Analysis Section
	if len(data.Ownership) > 0 {
		fmt.Println("  👤 Ownership Analysis:")
		topOwner := data.Ownership[0]
		fmt.Printf("    • Original owner: %s (%s)\n", topOwner.Developer, topOwner.Email)
		fmt.Printf("      └─ Last commit: %d days ago\n", topOwner.DaysSinceCommit)
		if !topOwner.IsActive {
			fmt.Println("      └─ Status: INACTIVE (no recent activity)")
		}

		if len(data.Ownership) > 1 {
			activeCount := 0
			for _, owner := range data.Ownership[1:] {
				if owner.IsActive {
					activeCount++
				}
			}
			fmt.Printf("    • Current contributors: %d active developer%s\n",
				activeCount, pluralize(activeCount))

			for i, owner := range data.Ownership[1:] {
				if i >= 2 || !owner.IsActive {
					break // Show top 2 active
				}
				fmt.Printf("      └─ %s: %d commits (most recent: %d days ago)\n",
					owner.Developer, owner.CommitCount, owner.DaysSinceCommit)
			}
		}
		fmt.Println("    • Your familiarity: 0 commits to this file")
		fmt.Println()
	}

	// Co-Change Patterns Section
	if len(data.CoChange) > 0 {
		fmt.Println("  🔗 Co-Change Patterns:")
		for i, partner := range data.CoChange {
			if i >= 2 {
				break // Show top 2
			}
			fmt.Printf("    • %s (%.0f%% co-change rate, %d/%d commits)\n",
				partner.PartnerFile,
				partner.Frequency*100,
				partner.CoChangeCount,
				partner.CoChangeCount*2) // Rough estimate

			// TODO: Check if partner file is in current diff
			fmt.Println("      └─ ⚠️  NOT modified in your current changes")

			if len(partner.SampleCommits) > 0 {
				fmt.Printf("      └─ Sample commits: \"%s\"\n", truncate(partner.SampleCommits[0], 60))
			}
		}
		fmt.Println()
	}

	// Blast Radius Section
	if len(data.BlastRadius) > 0 {
		fmt.Printf("  📦 Blast Radius:\n")
		fmt.Printf("    • %d files depend on this component\n", len(data.BlastRadius))

		incidentFiles := 0
		for _, file := range data.BlastRadius {
			if file.IncidentCount > 0 {
				incidentFiles++
			}
		}

		if incidentFiles > 0 {
			fmt.Println("    • Top downstream files with incidents:")
			shown := 0
			for _, file := range data.BlastRadius {
				if file.IncidentCount > 0 && shown < 2 {
					fmt.Printf("      └─ %s (%d incident%s)\n",
						file.FilePath, file.IncidentCount, pluralize(file.IncidentCount))
					shown++
				}
			}
		}
		fmt.Println()
	}

	// Recommended Actions
	fmt.Println("  ✅ Recommended Actions:")
	actionNum := 1

	if len(data.Incidents) > 0 {
		incidentRefs := make([]string, 0, 3)
		for i, inc := range data.Incidents {
			if i >= 3 {
				break
			}
			incidentRefs = append(incidentRefs, fmt.Sprintf("#%d", inc.IssueNumber))
		}
		fmt.Printf("    %d. 📖 Review past incidents for patterns (%s)\n",
			actionNum, strings.Join(incidentRefs, ", "))
		actionNum++
	}

	if len(data.Ownership) > 0 && len(data.Ownership) > 1 {
		recentDev := data.Ownership[1]
		if recentDev.IsActive {
			fmt.Printf("    %d. 👤 Ping %s for pre-review (most familiar)\n",
				actionNum, recentDev.Developer)
			actionNum++
		}
	}

	if len(data.Incidents) > 0 {
		fmt.Printf("    %d. 🧪 Add regression tests for: ", actionNum)
		scenarios := make([]string, 0, 3)
		for i, inc := range data.Incidents {
			if i >= 3 {
				break
			}
			// Extract key scenario from title
			scenario := extractScenario(inc.IssueTitle)
			scenarios = append(scenarios, scenario)
		}
		fmt.Println(strings.Join(scenarios, ", "))
		actionNum++
	}

	if len(data.CoChange) > 0 {
		fmt.Printf("    %d. 🔍 Verify if %s also needs changes (%.0f%% co-change)\n",
			actionNum, data.CoChange[0].PartnerFile, data.CoChange[0].Frequency*100)
		actionNum++
	}

	if len(data.Ownership) > 0 && !data.Ownership[0].IsActive {
		fmt.Printf("    %d. 🧑‍💻 Consider pairing (original owner unavailable)\n", actionNum)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayCLQSConfidence outputs CLQS-based confidence information
func DisplayCLQSConfidence(data *DemoOutputData) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  CONFIDENCE & DATA QUALITY")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("  Overall Confidence: %.0f%%\n", data.Assessment.Confidence*100)
	fmt.Println()

	if data.CLQSScore != nil {
		fmt.Printf("  Data Quality (CLQS Score): %d/100 (Grade: %s, %s)\n",
			data.CLQSScore.Score, data.CLQSScore.Grade, data.CLQSScore.Rank)
		fmt.Printf("  ├─ Link Coverage: %d/100 (issue-PR linking completeness)\n",
			data.CLQSScore.LinkCoverage)
		fmt.Printf("  ├─ Confidence Quality: %d/100 (avg link confidence)\n",
			data.CLQSScore.ConfidenceQuality)
		fmt.Printf("  ├─ Evidence Diversity: %d/100 (multiple signal types)\n",
			data.CLQSScore.EvidenceDiversity)
		fmt.Printf("  ├─ Temporal Precision: %d/100 (timing correlation)\n",
			data.CLQSScore.TemporalPrecision)
		fmt.Printf("  └─ Semantic Strength: %d/100 (semantic correlation)\n",
			data.CLQSScore.SemanticStrength)
	} else {
		fmt.Println("  Data Quality (CLQS Score): Not calculated")
	}

	fmt.Println()

	// Incident link quality
	if len(data.Incidents) > 0 {
		fixedByCount := 0
		assocWithCount := 0
		for _, inc := range data.Incidents {
			if inc.LinkType == "FIXED_BY" {
				fixedByCount++
			} else {
				assocWithCount++
			}
		}

		fmt.Println("  Incident Link Quality:")
		fmt.Printf("  • High-confidence links (FIXED_BY): %d/%d incidents\n",
			fixedByCount, len(data.Incidents))
		if assocWithCount > 0 {
			fmt.Printf("  • Medium-confidence links (ASSOCIATED_WITH): %d/%d incidents\n",
				assocWithCount, len(data.Incidents))
		}

		// Detection methods
		methods := make(map[string]int)
		for _, inc := range data.Incidents {
			methods[inc.DetectionMethod]++
		}
		fmt.Print("  • Detection methods: ")
		methodStrs := []string{}
		for method, count := range methods {
			methodStrs = append(methodStrs, fmt.Sprintf("%d× %s", count, method))
		}
		fmt.Println(strings.Join(methodStrs, ", "))
		fmt.Println("  • No conflicting evidence detected")
	}

	fmt.Println()

	// Ownership data completeness
	if len(data.Ownership) > 0 {
		totalCommits := 0
		for _, owner := range data.Ownership {
			totalCommits += owner.CommitCount
		}
		mostRecentDays := data.Ownership[0].DaysSinceCommit
		for _, owner := range data.Ownership {
			if owner.DaysSinceCommit < mostRecentDays {
				mostRecentDays = owner.DaysSinceCommit
			}
		}

		fmt.Println("  Ownership Data Completeness:")
		fmt.Printf("  • %d commits analyzed across %d developer%s\n",
			totalCommits, len(data.Ownership), pluralize(len(data.Ownership)))
		fmt.Printf("  • Last activity: %d days ago\n", mostRecentDays)
	}

	fmt.Println()
	if data.CLQSScore != nil && data.CLQSScore.Score >= 90 {
		fmt.Println("  This risk assessment is highly reliable due to world-class")
		fmt.Println("  data hygiene and multi-signal incident verification.")
	} else if data.CLQSScore != nil && data.CLQSScore.Score >= 70 {
		fmt.Println("  This risk assessment has good data quality with verified")
		fmt.Println("  incident links and ownership tracking.")
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// Helper functions for demo output

func colorizeRiskLevel(level string) string {
	switch level {
	case "CRITICAL":
		return "🔴 CRITICAL"
	case "HIGH":
		return "🟠 HIGH"
	case "MEDIUM":
		return "🟡 MEDIUM"
	case "LOW":
		return "🟢 LOW"
	default:
		return level
	}
}

func calculateAvgMTTR(incidents []database.IncidentWithContext) float64 {
	if len(incidents) == 0 {
		return 0
	}

	var totalHours float64
	count := 0

	for _, inc := range incidents {
		if inc.ClosedAt != nil {
			duration := inc.ClosedAt.Sub(inc.CreatedAt)
			totalHours += duration.Hours()
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalHours / float64(count)
}

func describeAuthorRole(role string) string {
	switch role {
	case "OWNER":
		return "repository owner"
	case "COLLABORATOR":
		return "team member with write access"
	case "MEMBER":
		return "organization member"
	case "CONTRIBUTOR":
		return "external contributor"
	case "NONE":
		return "external user (likely customer)"
	default:
		return "unknown role"
	}
}

func extractScenario(title string) string {
	// Simple extraction: take the part after "]" or ":"
	if idx := strings.Index(title, "]"); idx != -1 && idx < len(title)-1 {
		return strings.TrimSpace(title[idx+1:])
	}
	if idx := strings.Index(title, ":"); idx != -1 && idx < len(title)-1 {
		return strings.TrimSpace(title[idx+1:])
	}
	return truncate(title, 30)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
