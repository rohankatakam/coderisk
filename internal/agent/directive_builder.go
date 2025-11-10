package agent

import "fmt"

// BuildContactHumanDirective creates a directive for manually contacting a code owner
// This generates a message template for the user to send via Slack or other channels
func BuildContactHumanDirective(
	recipient string,
	recipientRole string,
	filePath string,
	lineNumbers string,
	incidentContext string,
	evidence []DirectiveEvidence,
) *DirectiveMessage {

	// Generate specific message template for user to send
	slackMessage := fmt.Sprintf(`Hi %s,

I'm analyzing a change to %s (lines %s) that you previously worked on.

%s

Questions:
1. Have you seen this type of issue before?
2. Is it safe to modify this logic without breaking existing behavior?
3. Should anyone else review this change?

Current change: [describe your changes here]

Could you respond when you get a chance? I'm running a risk assessment.

Thanks!`,
		recipient,
		filePath,
		lineNumbers,
		incidentContext,
	)

	return &DirectiveMessage{
		Action: DirectiveAction{
			Type: DirectiveTypeContactHuman,
			Description: fmt.Sprintf("Contact %s (%s) to verify safety", recipient, recipientRole),
			Parameters: map[string]interface{}{
				"recipient": recipient,
				"channel": "slack", // Manual - user sends via Slack
				"message_template": slackMessage,
				"expected_response_time": "30 minutes",
			},
		},
		Reasoning: fmt.Sprintf("%s is the %s with deep context on this code. Past incidents involved changes they made.", recipient, recipientRole),
		Evidence: evidence,
		Contingencies: []ContingencyPlan{
			{
				Trigger: "If they confirm it's safe (they say 'safe', 'yes', 'go ahead', or 'approved')",
				Condition: "response contains positive confirmation",
				NextAction: DirectiveAction{
					Type: DirectiveTypeDeepInvestigation,
					Description: "Continue with Phase 2 blast radius and co-change analysis",
					Parameters: map[string]interface{}{
						"estimated_time": "2 minutes",
						"estimated_cost": 0.03,
					},
				},
				Confidence: 0.9,
			},
			{
				Trigger: "If they warn of risks (they say 'risky', 'be careful', 'check with security', or express concerns)",
				Condition: "response contains warnings or concerns",
				NextAction: DirectiveAction{
					Type: DirectiveTypeEscalate,
					Description: "Investigation cannot proceed safely - high risk confirmed by code owner",
					Parameters: map[string]interface{}{
						"reason": "Code owner flagged risks",
						"recommendation": "Do not commit until risks are addressed",
					},
				},
				Confidence: 0.95,
			},
			{
				Trigger: "If they don't respond within 30 minutes",
				Condition: "timeout",
				NextAction: DirectiveAction{
					Type: DirectiveTypeProceedWithCaution,
					Description: "Continue investigation but flag for manual review",
					Parameters: map[string]interface{}{
						"warning": "Code owner not consulted - increased risk",
					},
				},
				Confidence: 0.6,
			},
		},
		UserOptions: []UserOption{
			{
				ID: "send_message",
				Label: "📤 I'll Send This Message",
				Description: "Copy message, send via Slack, and provide their response",
				Action: UserActionApprove,
				Shortcut: "a",
			},
			{
				ID: "edit_message",
				Label: "✏️ Edit Message First",
				Description: "Modify the message before sending",
				Action: UserActionModify,
				Shortcut: "e",
			},
			{
				ID: "skip",
				Label: "⏭️ Skip Contact",
				Description: "Continue investigation without owner input (higher risk)",
				Action: UserActionSkip,
				Shortcut: "s",
			},
			{
				ID: "abort",
				Label: "❌ Abort Check",
				Description: "Stop investigation",
				Action: UserActionAbort,
				Shortcut: "x",
			},
		},
	}
}

// BuildDeepInvestigationDirective creates a directive for proceeding with Phase 2 analysis
func BuildDeepInvestigationDirective(
	estimatedTime string,
	estimatedCost float64,
	reasoning string,
	evidence []DirectiveEvidence,
) *DirectiveMessage {
	return &DirectiveMessage{
		Action: DirectiveAction{
			Type: DirectiveTypeDeepInvestigation,
			Description: "Proceed with deep Phase 2 investigation",
			Parameters: map[string]interface{}{
				"estimated_time": estimatedTime,
				"estimated_cost": estimatedCost,
			},
		},
		Reasoning: reasoning,
		Evidence: evidence,
		Contingencies: []ContingencyPlan{
			{
				Trigger: "Investigation completes successfully",
				Condition: "success",
				NextAction: DirectiveAction{
					Type: "ASSESSMENT_COMPLETE",
					Description: "Provide final risk assessment",
				},
				Confidence: 0.95,
			},
		},
		UserOptions: []UserOption{
			{
				ID: "proceed",
				Label: "▶️ Proceed with Investigation",
				Description: "Continue with Phase 2 analysis",
				Action: UserActionApprove,
				Shortcut: "a",
			},
			{
				ID: "abort",
				Label: "❌ Abort",
				Description: "Stop investigation",
				Action: UserActionAbort,
				Shortcut: "x",
			},
		},
	}
}

// BuildEscalationDirective creates a directive for escalating a high-risk finding
func BuildEscalationDirective(
	reason string,
	recommendation string,
	evidence []DirectiveEvidence,
) *DirectiveMessage {
	return &DirectiveMessage{
		Action: DirectiveAction{
			Type: DirectiveTypeEscalate,
			Description: "Escalate to manual review - high risk detected",
			Parameters: map[string]interface{}{
				"reason": reason,
				"recommendation": recommendation,
			},
		},
		Reasoning: reason,
		Evidence: evidence,
		Contingencies: []ContingencyPlan{},
		UserOptions: []UserOption{
			{
				ID: "acknowledge",
				Label: "✓ Acknowledge",
				Description: "Acknowledge the escalation and stop",
				Action: UserActionApprove,
				Shortcut: "a",
			},
		},
	}
}
