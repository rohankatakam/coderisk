package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// DisplayDirective renders a directive message to the terminal with rich formatting
func DisplayDirective(dm *DirectiveMessage, decisionNum int) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("🤖 DECISION POINT #%d\n", decisionNum)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Proposed action
	fmt.Printf("📋 PROPOSED ACTION:\n")
	fmt.Printf("%s\n\n", dm.Action.Description)

	// Show message template for manual contact if applicable
	if dm.Action.Type == DirectiveTypeContactHuman {
		if recipient, ok := dm.Action.Parameters["recipient"].(string); ok {
			if message, ok := dm.Action.Parameters["message_template"].(string); ok {
				fmt.Println("💬 MESSAGE TO SEND VIA SLACK:")
				fmt.Println("───────────────────────────────────────────────────────────")
				fmt.Println(message)
				fmt.Println("───────────────────────────────────────────────────────────")
				fmt.Println()
				fmt.Printf("⚠️  NOTE: You'll need to manually send this via Slack to %s\n", recipient)
				fmt.Println()
			}
		}
	}

	// Contingency plan
	if len(dm.Contingencies) > 0 {
		fmt.Println("🧭 CONTINGENCY PLAN:")
		fmt.Println()
		for i, cont := range dm.Contingencies {
			fmt.Printf("→ %s:\n", cont.Trigger)
			fmt.Printf("  %s\n", cont.NextAction.Description)
			if params := cont.NextAction.Parameters; params != nil {
				if time, ok := params["estimated_time"].(string); ok {
					fmt.Printf("  ⏱️  Time: %s\n", time)
				}
				if cost, ok := params["estimated_cost"].(float64); ok {
					fmt.Printf("  💰 Cost: $%.2f\n", cost)
				}
			}
			if i < len(dm.Contingencies)-1 {
				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Reasoning
	if dm.Reasoning != "" {
		fmt.Println("❓ WHY AM I ASKING THIS?")
		fmt.Println(dm.Reasoning)
		fmt.Println()
	}

	// Evidence
	if len(dm.Evidence) > 0 {
		fmt.Println("📊 SUPPORTING EVIDENCE:")
		for _, ev := range dm.Evidence {
			fmt.Printf("  • %s (%s)\n", ev.Data, ev.Source)
		}
		fmt.Println()
	}

	// Options
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("YOUR OPTIONS:")
	for _, opt := range dm.UserOptions {
		fmt.Printf("  [%s] %s\n      %s\n", opt.Shortcut, opt.Label, opt.Description)
	}
	fmt.Println()
}

// PromptForChoice displays a prompt and waits for user input
func PromptForChoice(defaultChoice string, shortcuts []string) string {
	fmt.Printf("Type your choice [%s] or Enter for [%s]: ", strings.Join(shortcuts, "/"), defaultChoice)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultChoice
	}

	return strings.ToLower(strings.TrimSpace(input))
}

// PromptForResponse prompts the user for a text response
func PromptForResponse(prompt string) string {
	fmt.Print(prompt)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}

	return ""
}

// PromptForMultilineResponse prompts for a multi-line text response
// User can end input with Ctrl+D (EOF)
func PromptForMultilineResponse(prompt string) string {
	fmt.Println(prompt)
	fmt.Println("(Enter your response. Press Ctrl+D when done, or Ctrl+C to cancel)")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return strings.Join(lines, "\n")
}

// DisplayInvestigationHeader shows the investigation start banner
func DisplayInvestigationHeader(investigationID string, files []string) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         CodeRisk Directive Investigation Started          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Investigation ID: %s\n", investigationID)
	fmt.Printf("Files: %s\n", strings.Join(files, ", "))
	fmt.Println()
}

// DisplayPhaseHeader shows a phase transition banner
func DisplayPhaseHeader(phaseName string, description string) {
	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("Phase: %s\n", phaseName)
	if description != "" {
		fmt.Printf("%s\n", description)
	}
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Println()
}

// DisplayProgressMessage shows a progress update
func DisplayProgressMessage(message string) {
	fmt.Printf("⏳ %s\n", message)
}

// DisplaySuccessMessage shows a success message
func DisplaySuccessMessage(message string) {
	fmt.Printf("✅ %s\n", message)
}

// DisplayWarningMessage shows a warning message
func DisplayWarningMessage(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// DisplayErrorMessage shows an error message
func DisplayErrorMessage(message string) {
	fmt.Printf("❌ %s\n", message)
}

// DisplayCheckpointSaved shows checkpoint save confirmation
func DisplayCheckpointSaved(investigationID string) {
	fmt.Println()
	fmt.Printf("💾 Investigation saved. Resume with: crisk check --resume %s\n", investigationID)
	fmt.Println()
}

// DisplayFinalAssessment shows the final risk assessment
func DisplayFinalAssessment(riskLevel string, confidence float64, summary string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("           FINAL RISK ASSESSMENT")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Color-code risk level
	var riskIndicator string
	switch strings.ToUpper(riskLevel) {
	case "LOW":
		riskIndicator = "✅ LOW RISK"
	case "MEDIUM":
		riskIndicator = "⚠️  MEDIUM RISK"
	case "HIGH":
		riskIndicator = "🔴 HIGH RISK"
	case "CRITICAL":
		riskIndicator = "🚨 CRITICAL RISK"
	default:
		riskIndicator = "❓ UNKNOWN RISK"
	}

	fmt.Printf("Risk Level: %s\n", riskIndicator)
	fmt.Printf("Confidence: %.1f%%\n", confidence*100)
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println(summary)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
}
