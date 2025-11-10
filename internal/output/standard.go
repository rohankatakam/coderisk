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
	// Use rich narrative format for transparency and observability
	DisplayRichNarrativeTrace(assessment)
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
// Enhanced per YC_DEMO_GAP_ANALYSIS.md - Business Impact Section
func DisplayManagerView(data *DemoOutputData) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  📊 BUSINESS IMPACT ASSESSMENT")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// === INCIDENT HISTORY WITH COST ===
	if len(data.Incidents) > 0 {
		fmt.Printf("  🔥 PRODUCTION INCIDENT HISTORY\n")
		fmt.Printf("  This file has caused %d production incident(s) in the last 180 days:\n\n", len(data.Incidents))

		// Show top 3-5 incidents with full details
		for i, inc := range data.Incidents {
			if i >= 5 {
				fmt.Printf("    ... and %d more incident(s)\n\n", len(data.Incidents)-5)
				break
			}

			// Incident severity indicator
			severityLabel := "SEV-?"
			if strings.Contains(strings.ToLower(inc.IssueTitle), "critical") ||
			   strings.Contains(strings.ToLower(inc.IssueTitle), "production") {
				severityLabel = "SEV-1"
			} else if strings.Contains(strings.ToLower(inc.IssueTitle), "urgent") {
				severityLabel = "SEV-2"
			}

			fmt.Printf("    %s #%d: %s\n", getSeverityEmoji(severityLabel), inc.IssueNumber, inc.IssueTitle)

			// Impact description
			if inc.ClosedAt != nil {
				mttr := inc.ClosedAt.Sub(inc.CreatedAt)
				fmt.Printf("      └─ MTTR: %.1f hours\n", mttr.Hours())

				// Cost estimate ($300K per hour is industry standard for SaaS downtime)
				estimatedCost := mttr.Hours() * 300000 / 24 // Daily cost
				if mttr.Hours() < 1 {
					fmt.Printf("      └─ Estimated Impact: ~$%.0fK (%.0f min downtime)\n",
						estimatedCost/1000, mttr.Minutes())
				} else {
					fmt.Printf("      └─ Estimated Impact: ~$%.0fK (%.1f hr downtime)\n",
						estimatedCost/1000, mttr.Hours())
				}
			}

			// Date and reporter
			daysAgo := int(time.Since(inc.CreatedAt).Hours() / 24)
			fmt.Printf("      └─ Occurred: %s (%d days ago)\n",
				inc.CreatedAt.Format("2006-01-02"), daysAgo)

			reporterType := "customer"
			if inc.AuthorRole != "NONE" {
				reporterType = "team member"
			}
			fmt.Printf("      └─ Reported by: %s\n", reporterType)
			fmt.Println()
		}

		// Total cost estimate
		totalMTTR := calculateAvgMTTR(data.Incidents)
		totalCost := totalMTTR * float64(len(data.Incidents)) * 300000 / 24

		fmt.Printf("  💰 TOTAL HISTORICAL COST: ~$%.0fK\n", totalCost/1000)
		fmt.Printf("  ⏱️  AVERAGE MTTR: %.1f hours\n\n", totalMTTR)
	} else {
		fmt.Println("  ✅ PRODUCTION INCIDENT HISTORY")
		fmt.Println("  No incidents found in the last 180 days\n")
	}

	// === CRITICAL USER FLOW DETECTION ===
	if strings.Contains(strings.ToLower(data.FilePath), "table") ||
		strings.Contains(strings.ToLower(data.FilePath), "editor") ||
		strings.Contains(strings.ToLower(data.FilePath), "auth") ||
		strings.Contains(strings.ToLower(data.FilePath), "payment") {
		fmt.Println("  ⚠️  CRITICAL USER FLOW IMPACT")

		flowType := "Unknown"
		if strings.Contains(strings.ToLower(data.FilePath), "table") ||
		   strings.Contains(strings.ToLower(data.FilePath), "editor") {
			flowType = "Database Table Editing (P0 functionality)"
		} else if strings.Contains(strings.ToLower(data.FilePath), "auth") {
			flowType = "Authentication (P0 functionality)"
		} else if strings.Contains(strings.ToLower(data.FilePath), "payment") {
			flowType = "Payment Processing (P0 revenue-critical)"
		}

		fmt.Printf("  Affected Flow: %s\n", flowType)
		fmt.Println("  User Impact: All customers affected if this breaks")
		fmt.Println()
	}

	// === CODE OWNERSHIP RISK ===
	if len(data.Ownership) > 0 {
		fmt.Println("  👤 CODE OWNERSHIP RISK")

		topOwner := data.Ownership[0]
		fmt.Printf("  Primary Author: %s (%s)\n", topOwner.Developer, topOwner.Email)
		fmt.Printf("  Last Commit: %d days ago\n", topOwner.DaysSinceCommit)

		if !topOwner.IsActive {
			fmt.Printf("  Status: ⚠️  STALE (no activity in last 30+ days)\n")
			fmt.Println("  Risk: Knowledge loss, lack of familiarity with recent changes")
		} else if topOwner.DaysSinceCommit > 30 {
			fmt.Printf("  Status: ⚠️  AGING (last modified %d days ago)\n", topOwner.DaysSinceCommit)
		} else {
			fmt.Println("  Status: ✅ ACTIVE")
		}

		// Bus factor calculation
		activeDevs := 0
		for _, owner := range data.Ownership {
			if owner.IsActive {
				activeDevs++
			}
		}

		fmt.Printf("\n  Bus Factor: %d developer%s familiar with this code\n",
			activeDevs, pluralize(activeDevs))

		if activeDevs <= 1 {
			fmt.Println("  ⚠️  CRITICAL: Single point of failure (only 1 active developer)")
		} else if activeDevs <= 2 {
			fmt.Println("  ⚠️  HIGH: Low redundancy (only 2 active developers)")
		}
		fmt.Println()
	}

	// === CO-CHANGE RISK ===
	if len(data.CoChange) > 0 {
		fmt.Println("  🔗 CO-CHANGE PATTERN RISK")
		fmt.Printf("  This file has %d co-change partner(s) - files that usually change together:\n\n", len(data.CoChange))

		for i, partner := range data.CoChange {
			if i >= 3 {
				fmt.Printf("    ... and %d more co-change partner(s)\n", len(data.CoChange)-3)
				break
			}

			fmt.Printf("    • %s\n", partner.PartnerFile)
			// Calculate total commits from frequency and co-change count
			// frequency = coChangeCount / totalCommits, so totalCommits = coChangeCount / frequency
			totalCommits := int(float64(partner.CoChangeCount) / partner.Frequency)
			if partner.Frequency > 0 {
				fmt.Printf("      └─ Co-change Rate: %.0f%% (%d/%d commits)\n",
					partner.Frequency*100,
					partner.CoChangeCount,
					totalCommits)
			} else {
				fmt.Printf("      └─ Co-change Rate: %.0f%% (%d commits)\n",
					partner.Frequency*100,
					partner.CoChangeCount)
			}

			if partner.Frequency > 0.7 {
				fmt.Println("      └─ ⚠️  VERY HIGH - These files are tightly coupled")
			} else if partner.Frequency > 0.5 {
				fmt.Println("      └─ ⚠️  HIGH - Coordination failure risk")
			}

			// TODO: Check if partner file is in current diff
			fmt.Println("      └─ ⚠️  NOT updated in current changes (potential incomplete change)")
			fmt.Println()
		}
	}

	// === FINAL RISK ASSESSMENT ===
	fmt.Println("  🎯 FINAL RISK ASSESSMENT")
	fmt.Printf("  Risk Level: %s\n", colorizeRiskLevel(string(data.Assessment.RiskLevel)))
	if data.CLQSScore != nil {
		fmt.Printf("  Confidence: %.0f%% (based on CLQS Score: %d - %s)\n",
			data.Assessment.Confidence*100,
			data.CLQSScore.Score,
			data.CLQSScore.Rank)
	} else {
		fmt.Printf("  Confidence: %.0f%%\n", data.Assessment.Confidence*100)
	}

	fmt.Println()

	// === ESTIMATED COST IF THIS BREAKS ===
	if len(data.Incidents) > 0 {
		avgMTTR := calculateAvgMTTR(data.Incidents)
		estimatedCost := avgMTTR * 300000 / 24 // Daily cost based on $300K/day industry standard

		fmt.Println("  💰 ESTIMATED COST IF THIS BREAKS:")
		if avgMTTR > 0 {
			fmt.Printf("  MTTR Estimate: %.1f hours (based on past incident average)\n", avgMTTR)
		}
		fmt.Printf("  Cost Estimate: ~$%.0fK per incident\n", estimatedCost/1000)
		fmt.Println("  Time Saved: ~%.0f hours if caught pre-commit vs. firefighting\n", avgMTTR)
	} else {
		fmt.Println("  💰 ESTIMATED COST IF THIS BREAKS:")
		fmt.Println("  No past incident data - unable to estimate MTTR")
		fmt.Println("  Default assumption: ~$300K/day downtime cost (industry standard)\n")
	}

	fmt.Println("  📋 RECOMMENDATION:")
	riskLevel := string(data.Assessment.RiskLevel)
	if riskLevel == "CRITICAL" || riskLevel == "HIGH" {
		fmt.Println("  ⚠️  REQUIRE senior engineer review before merge")
		if len(data.Ownership) > 0 && len(data.Ownership) > 1 {
			fmt.Printf("  ⚠️  REQUIRE review from original author: %s\n", data.Ownership[0].Developer)
		}
		if len(data.Incidents) > 0 {
			fmt.Println("  ⚠️  REQUIRE regression tests for past incident scenarios")
		}
	} else if riskLevel == "MEDIUM" {
		fmt.Println("  ✓ Standard peer review recommended")
		if len(data.Ownership) > 0 {
			fmt.Printf("  ✓ Consider pinging %s for context\n", data.Ownership[0].Developer)
		}
	} else {
		fmt.Println("  ✅ Standard review process is sufficient")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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

func getSeverityEmoji(severity string) string {
	switch severity {
	case "SEV-1", "CRITICAL":
		return "🔴"
	case "SEV-2", "HIGH":
		return "🟠"
	case "SEV-3", "MEDIUM":
		return "🟡"
	default:
		return "⚪"
	}
}
