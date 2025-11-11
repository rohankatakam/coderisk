package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rohankatakam/coderisk/internal/agent"
)

// DisplayRichNarrativeTrace shows the investigation as a transparent narrative journey
// This format prioritizes:
// 1. Transparency - show all data discovered
// 2. Digestibility - present in narrative, not terse bullets
// 3. Observability - user can see agent's thought process
// 4. Actionability - surface nitty-gritty details that inform decisions
func DisplayRichNarrativeTrace(assessment agent.RiskAssessment) {
	if assessment.Investigation == nil {
		DisplayPhase2Summary(assessment)
		return
	}

	inv := assessment.Investigation

	// Header
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           🔍 CodeRisk Deep Investigation Report                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📋 File Under Review: %s\n", assessment.FilePath)
	fmt.Printf("⏱️  Investigation Duration: %.1fs across %d investigative hops\n",
		inv.CompletedAt.Sub(inv.Request.StartedAt).Seconds(),
		len(inv.Hops))
	fmt.Printf("🧠 Total Analysis Tokens: %d\n", inv.TotalTokens)

	// Investigation Journey
	fmt.Println("\n" + strings.Repeat("─", 74))
	fmt.Println("📚 THE INVESTIGATION JOURNEY")
	fmt.Println(strings.Repeat("─", 74))
	fmt.Println("\nThe agent investigated this change by gathering evidence from your")
	fmt.Println("codebase history. Here's what it discovered, step by step:\n")

	// Show each hop as a narrative
	for _, hop := range inv.Hops {
		displayHopNarrative(hop)
	}

	// Final Assessment
	fmt.Println("\n" + strings.Repeat("═", 74))
	fmt.Println("🎯 FINAL RISK ASSESSMENT")
	fmt.Println(strings.Repeat("═", 74))

	// Risk level with visual emphasis
	riskEmoji := getRiskEmoji(assessment.RiskLevel)
	fmt.Printf("\n%s Risk Level: %s\n", riskEmoji, assessment.RiskLevel)
	fmt.Printf("📊 Confidence: %.0f%%\n", assessment.Confidence*100)

	if assessment.Summary != "" {
		fmt.Printf("\n💡 Summary:\n%s\n", wrapText(assessment.Summary, 70, "   "))
	}

	// Collect incident details for action items
	incidentNumbers := []string{}
	for _, hop := range inv.Hops {
		for _, toolResult := range hop.ToolResults {
			if toolResult.ToolName == "get_incidents_with_context" && toolResult.Result != nil {
				if incidents, ok := toolResult.Result.([]interface{}); ok {
					for _, inc := range incidents {
						if incMap, ok := inc.(map[string]interface{}); ok {
							if issueNum, ok := incMap["issue_number"].(float64); ok {
								incidentNumbers = append(incidentNumbers, fmt.Sprintf("#%.0f", issueNum))
							}
						}
					}
				}
			}
		}
	}

	// Developer-focused action items (what to do before committing)
	if len(incidentNumbers) > 0 || assessment.RiskLevel >= agent.RiskMedium {
		fmt.Println("\n" + strings.Repeat("─", 74))
		fmt.Println("🎯 WHAT SHOULD YOU DO?")
		fmt.Println(strings.Repeat("─", 74))
		fmt.Println("\nBefore committing this change, consider:")

		actionNum := 1
		if len(incidentNumbers) > 0 {
			incidentList := strings.Join(incidentNumbers, ", ")
			fmt.Printf("   %d. 📖 Review past incidents: %s\n", actionNum, incidentList)
			fmt.Printf("      Understand what went wrong before to avoid repeating it\n")
			actionNum++
		}

		fmt.Printf("   %d. ✅ Add test coverage for this file\n", actionNum)
		fmt.Printf("      This file has %s risk - tests will catch regressions\n", strings.ToLower(string(assessment.RiskLevel)))
		actionNum++

		fmt.Printf("   %d. 👥 Get a second pair of eyes\n", actionNum)
		fmt.Printf("      Consider asking someone familiar with this code to review\n")
		actionNum++

		// Check for co-change patterns in tool results
		hasCoChangeWarning := false
		for _, hop := range inv.Hops {
			for _, toolResult := range hop.ToolResults {
				if (toolResult.ToolName == "query_cochange_partners" || toolResult.ToolName == "get_cochange_with_explanations") &&
					toolResult.Result != nil {
					if partners, ok := toolResult.Result.([]interface{}); ok && len(partners) > 0 {
						hasCoChangeWarning = true
						break
					}
				}
			}
		}

		if hasCoChangeWarning {
			fmt.Printf("   %d. 🔗 Check co-change files\n", actionNum)
			fmt.Printf("      Files that usually change together were found - verify they're updated too\n")
			actionNum++
		}
	}

	// Recommendations
	if len(assessment.Recommendations) > 0 {
		fmt.Println("\n✅ Recommended Actions:")
		for i, rec := range assessment.Recommendations {
			fmt.Printf("   %d. %s\n", i+1, rec)
		}
	}

	// Evidence Summary
	if len(assessment.Evidence) > 0 {
		fmt.Println("\n📌 Key Evidence Points:")
		for _, ev := range assessment.Evidence {
			fmt.Printf("   • [%s] %s\n", ev.Type, ev.Description)
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 74))
	fmt.Printf("Investigation completed at %s\n", inv.CompletedAt.Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("═", 74) + "\n")
}

// displayHopNarrative shows a single hop as a narrative with full data transparency
func displayHopNarrative(hop agent.HopResult) {
	fmt.Printf("\n┌─ Hop %d ", hop.HopNumber)
	fmt.Println(strings.Repeat("─", 67-len(fmt.Sprintf("Hop %d", hop.HopNumber))))

	// Show agent's reasoning
	if hop.Response != "" {
		fmt.Println("│")
		fmt.Printf("│ 🤔 Agent's Thought Process:\n")
		fmt.Printf("│    %s\n", wrapText(hop.Response, 67, "│    "))
	}

	// Show tools executed and their results
	if len(hop.ToolResults) > 0 {
		fmt.Println("│")
		for _, toolResult := range hop.ToolResults {
			displayToolResultNarrative(toolResult)
		}
	}

	// Show performance metrics
	fmt.Println("│")
	fmt.Printf("│ ⚡ Performance: %dms | 🔢 Tokens: %d\n",
		hop.Duration.Milliseconds(),
		hop.TokensUsed)
	fmt.Println("└" + strings.Repeat("─", 73))
}

// displayToolResultNarrative shows tool execution results with full transparency
func displayToolResultNarrative(toolResult agent.ToolResult) {
	toolName := toolResult.ToolName
	toolDesc := getToolDescription(toolName)

	fmt.Printf("│ 🔧 Tool Executed: %s\n", toolName)
	fmt.Printf("│    %s\n", wrapText(toolDesc, 67, "│    "))

	// Show arguments
	if toolResult.Args != nil {
		fmt.Println("│")
		fmt.Println("│ 📥 Query Parameters:")
		displayArgs(toolResult.Args)
	}

	// Show results
	if toolResult.Error != "" {
		fmt.Println("│")
		fmt.Printf("│ ❌ Error: %s\n", toolResult.Error)
	} else if toolResult.Result != nil {
		fmt.Println("│")
		fmt.Println("│ 📊 Data Discovered:")
		displayToolData(toolName, toolResult.Result)
	}
}

// displayArgs shows tool arguments in a readable format
func displayArgs(args interface{}) {
	argsMap, ok := args.(map[string]interface{})
	if !ok {
		fmt.Printf("│    %v\n", args)
		return
	}

	for key, value := range argsMap {
		switch v := value.(type) {
		case []interface{}:
			fmt.Printf("│    • %s: [%d items]\n", key, len(v))
			for i, item := range v {
				if i < 3 { // Show first 3 items
					fmt.Printf("│      - %v\n", item)
				} else if i == 3 {
					fmt.Printf("│      - ... and %d more\n", len(v)-3)
					break
				}
			}
		default:
			fmt.Printf("│    • %s: %v\n", key, value)
		}
	}
}

// displayToolData formats and displays tool results based on tool type
func displayToolData(toolName string, result interface{}) {
	switch toolName {
	case "get_incidents_with_context":
		displayIncidents(result)
	case "get_ownership_timeline":
		displayOwnership(result)
	case "query_recent_commits":
		displayCommits(result)
	case "get_cochange_with_explanations", "query_cochange_partners":
		displayCoChange(result)
	case "query_ownership":
		displayBasicOwnership(result)
	case "query_blast_radius", "get_blast_radius_analysis":
		displayBlastRadius(result)
	default:
		displayGenericData(result)
	}
}

// displayIncidents shows incident data with full context
func displayIncidents(result interface{}) {
	incidents, ok := result.([]interface{})
	if !ok || len(incidents) == 0 {
		fmt.Println("│    ✓ No incidents found (this is good news!)")
		return
	}

	fmt.Printf("│    🚨 FOUND %d PRODUCTION INCIDENT(S) linked to this file:\n", len(incidents))
	fmt.Println("│")
	fmt.Println("│    This file has a history of causing problems in production.")
	fmt.Println("│    Review these incidents before making changes:")
	fmt.Println("│")

	for i, inc := range incidents {
		incMap, ok := inc.(map[string]interface{})
		if !ok {
			continue
		}

		// More prominent incident display
		fmt.Printf("│    ┌─ Incident #%d ", i+1)
		fmt.Println(strings.Repeat("─", 52-len(fmt.Sprintf("Incident #%d", i+1))))

		if issueNum, ok := incMap["issue_number"].(float64); ok {
			fmt.Printf("│    │ 🔗 Issue #%.0f\n", issueNum)
		}
		if title, ok := incMap["issue_title"].(string); ok {
			fmt.Printf("│    │ 📝 Title: %s\n", wrapText(title, 57, "│    │         "))
		}
		if created, ok := incMap["created_at"].(string); ok {
			// Parse and format date nicely
			fmt.Printf("│    │ 📅 Date: %s\n", created[:10]) // Just show YYYY-MM-DD
		}
		if confidence, ok := incMap["link_confidence"].(float64); ok {
			confidencePct := confidence * 100
			confidenceIcon := "🎯"
			if confidencePct >= 90 {
				confidenceIcon = "✅"
			} else if confidencePct < 70 {
				confidenceIcon = "⚠️"
			}
			fmt.Printf("│    │ %s Link Confidence: %.0f%%\n", confidenceIcon, confidencePct)
		}
		fmt.Println("│    └" + strings.Repeat("─", 61))

		if i < len(incidents)-1 {
			fmt.Println("│")
		}
	}
}

// displayOwnership shows ownership timeline with activity status
func displayOwnership(result interface{}) {
	ownership, ok := result.([]interface{})
	if !ok || len(ownership) == 0 {
		fmt.Println("│    ℹ️  No ownership data available")
		return
	}

	fmt.Printf("│    👥 Found %d developer(s) who've touched this file:\n", len(ownership))
	fmt.Println("│")

	for i, own := range ownership {
		ownMap, ok := own.(map[string]interface{})
		if !ok {
			continue
		}

		fmt.Printf("│    Developer #%d:\n", i+1)
		if email, ok := ownMap["developer_email"].(string); ok {
			fmt.Printf("│      👤 %s\n", email)
		}
		if commits, ok := ownMap["commit_count"].(float64); ok {
			fmt.Printf("│      📝 Commits: %.0f\n", commits)
		}
		if lastActive, ok := ownMap["days_since_last_commit"].(float64); ok {
			status := "🟢 Active"
			if lastActive > 90 {
				status = "🔴 Inactive (>90 days)"
			} else if lastActive > 30 {
				status = "🟡 Less Active (>30 days)"
			}
			fmt.Printf("│      %s (%.0f days since last commit)\n", status, lastActive)
		}
		if i < len(ownership)-1 {
			fmt.Println("│")
		}
	}
}

// displayCommits shows recent commit history
func displayCommits(result interface{}) {
	commits, ok := result.([]interface{})
	if !ok || len(commits) == 0 {
		fmt.Println("│    ℹ️  No recent commits found")
		return
	}

	fmt.Printf("│    📜 Found %d recent commit(s):\n", len(commits))
	fmt.Println("│")

	for i, commit := range commits {
		if i >= 5 { // Show max 5 commits
			fmt.Printf("│    ... and %d more commits\n", len(commits)-5)
			break
		}

		commitMap, ok := commit.(map[string]interface{})
		if !ok {
			continue
		}

		if msg, ok := commitMap["message"].(string); ok {
			fmt.Printf("│    • %s\n", wrapText(msg, 65, "│      "))
		}
		if author, ok := commitMap["author"].(string); ok {
			fmt.Printf("│      by %s\n", author)
		}
		if timestamp, ok := commitMap["timestamp"].(string); ok {
			fmt.Printf("│      on %s\n", timestamp)
		}
		if i < len(commits)-1 && i < 4 {
			fmt.Println("│")
		}
	}
}

// displayCoChange shows co-change patterns with explanations
func displayCoChange(result interface{}) {
	partners, ok := result.([]interface{})
	if !ok || len(partners) == 0 {
		fmt.Println("│    ✓ No strong co-change patterns detected")
		return
	}

	fmt.Printf("│    🔗 Found %d file(s) that frequently change together:\n", len(partners))
	fmt.Println("│")
	fmt.Println("│    These files historically change in the same commits.")
	fmt.Println("│    If you modified this file, consider whether these files")
	fmt.Println("│    also need updates:")
	fmt.Println("│")

	for i, partner := range partners {
		if i >= 5 { // Show max 5 partners
			fmt.Printf("│    ... and %d more co-change partners\n", len(partners)-5)
			break
		}

		partnerMap, ok := partner.(map[string]interface{})
		if !ok {
			continue
		}

		if file, ok := partnerMap["partner_file"].(string); ok {
			fmt.Printf("│    • %s\n", file)
		}
		if freq, ok := partnerMap["frequency"].(float64); ok {
			fmt.Printf("│      📊 Co-change Rate: %.0f%%\n", freq*100)
		}
		if explanation, ok := partnerMap["explanation"].(string); ok {
			fmt.Printf("│      💬 Why: %s\n", wrapText(explanation, 61, "│           "))
		}
		if i < len(partners)-1 && i < 4 {
			fmt.Println("│")
		}
	}
}

// displayBasicOwnership shows simple ownership list
func displayBasicOwnership(result interface{}) {
	owners, ok := result.([]interface{})
	if !ok || len(owners) == 0 {
		fmt.Println("│    ℹ️  No ownership data available")
		return
	}

	fmt.Printf("│    👥 %d developer(s) have modified this file:\n", len(owners))
	for _, owner := range owners {
		ownerMap, ok := owner.(map[string]interface{})
		if !ok {
			continue
		}

		if email, ok := ownerMap["developer"].(string); ok {
			commitCount := ""
			if count, ok := ownerMap["commit_count"].(float64); ok {
				commitCount = fmt.Sprintf(" (%.0f commits)", count)
			}
			fmt.Printf("│      • %s%s\n", email, commitCount)
		}
	}
}

// displayBlastRadius shows downstream dependencies
func displayBlastRadius(result interface{}) {
	deps, ok := result.([]interface{})
	if !ok || len(deps) == 0 {
		fmt.Println("│    ✓ No downstream dependencies detected")
		fmt.Println("│      (Changes to this file have limited blast radius)")
		return
	}

	fmt.Printf("│    🎯 Found %d downstream file(s) that depend on this:\n", len(deps))
	fmt.Println("│")
	fmt.Println("│    Changes here may impact:")
	for i, dep := range deps {
		if i >= 10 { // Show max 10 dependencies
			fmt.Printf("│      ... and %d more dependent files\n", len(deps)-10)
			break
		}

		depMap, ok := dep.(map[string]interface{})
		if !ok {
			fmt.Printf("│      • %v\n", dep)
			continue
		}

		if file, ok := depMap["dependent_file"].(string); ok {
			fmt.Printf("│      • %s\n", file)
		}
	}
}

// displayGenericData shows any other data in a readable format
func displayGenericData(result interface{}) {
	// Try to marshal and pretty-print
	jsonData, err := json.MarshalIndent(result, "│    ", "  ")
	if err != nil {
		fmt.Printf("│    %v\n", result)
		return
	}

	lines := strings.Split(string(jsonData), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Truncate very long lines
		if len(line) > 70 {
			fmt.Printf("│    %s...\n", line[:67])
		} else {
			fmt.Println(line)
		}
	}
}

// getToolDescription returns a human-readable description of what each tool does
func getToolDescription(toolName string) string {
	descriptions := map[string]string{
		"get_incidents_with_context":    "Searches for past production incidents linked to this file",
		"get_ownership_timeline":        "Analyzes who owns this code and when they last contributed",
		"query_recent_commits":          "Retrieves recent commit history to understand recent changes",
		"get_cochange_with_explanations": "Identifies files that usually change together (forgotten updates risk)",
		"query_cochange_partners":       "Finds files with co-change patterns",
		"query_ownership":               "Lists developers who have modified this file",
		"query_blast_radius":            "Finds downstream files that depend on this one",
		"get_blast_radius_analysis":     "Analyzes the potential impact of changes to this file",
		"finish_investigation":          "Agent completed its investigation and is ready to provide final assessment",
	}

	if desc, ok := descriptions[toolName]; ok {
		return desc
	}
	return "Gathering additional context"
}

// getRiskEmoji returns an appropriate emoji for the risk level
func getRiskEmoji(riskLevel agent.RiskLevel) string {
	switch riskLevel {
	case agent.RiskCritical:
		return "🚨"
	case agent.RiskHigh:
		return "🔴"
	case agent.RiskMedium:
		return "🟡"
	case agent.RiskLow:
		return "🟢"
	case agent.RiskMinimal:
		return "✅"
	default:
		return "⚪"
	}
}

// wrapText wraps text to a specified width with an indent
func wrapText(text string, width int, indent string) string {
	if len(text) <= width {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n"+indent)
}
