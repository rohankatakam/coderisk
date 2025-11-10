package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/rohankatakam/coderisk/internal/database"
)

// RiskInvestigator performs agent-based risk investigation using LLM tool calling
// Reference: dev_docs/mvp/AGENT_KICKOFF_PROMPT_DESIGN.md
type RiskInvestigator struct {
	llmClient       *LLMClient
	graphClient     GraphQueryExecutor
	postgresAdapter PostgresQueryExecutor
	hybridClient    *database.HybridClient
	maxHops         int
}

// GraphQueryExecutor interface for Neo4j queries
type GraphQueryExecutor interface {
	ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
}

// PostgresQueryExecutor interface for Postgres queries
type PostgresQueryExecutor interface {
	// For getting commit patches
	GetCommitPatch(ctx context.Context, commitSHA string) (string, error)
}

// NewRiskInvestigator creates a new risk investigator
func NewRiskInvestigator(
	llmClient *LLMClient,
	graphClient GraphQueryExecutor,
	postgresAdapter PostgresQueryExecutor,
	hybridClient *database.HybridClient,
) *RiskInvestigator {
	return &RiskInvestigator{
		llmClient:       llmClient,
		graphClient:     graphClient,
		postgresAdapter: postgresAdapter,
		hybridClient:    hybridClient,
		maxHops:         5, // Safety limit
	}
}

// Investigate performs agent-based risk investigation
func (inv *RiskInvestigator) Investigate(ctx context.Context, kickoffPrompt string) (*RiskAssessment, error) {
	// Initialize investigation state
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(kickoffPrompt),
	}

	var investigation Investigation
	investigation.Request = InvestigationRequest{
		StartedAt: time.Now(),
	}

	// Agent loop
	for hop := 1; hop <= inv.maxHops; hop++ {
		hopStart := time.Now()

		// Call LLM with available tools
		params := openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4o,
			Messages: messages,
			Tools:    inv.getToolDefinitions(),
		}

		completion, err := inv.llmClient.openaiClient.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("LLM request failed at hop %d: %w", hop, err)
		}

		if len(completion.Choices) == 0 {
			return nil, fmt.Errorf("no response from LLM at hop %d", hop)
		}

		choice := completion.Choices[0]
		investigation.TotalTokens += int(completion.Usage.TotalTokens)

		// Record hop
		hopResult := HopResult{
			HopNumber:  hop,
			Query:      "", // Query is in the messages
			Response:   choice.Message.Content,
			TokensUsed: int(completion.Usage.TotalTokens),
			Duration:   time.Since(hopStart),
		}

		// Check if agent wants to finish (no tool calls)
		toolCalls := choice.Message.ToolCalls
		if len(toolCalls) == 0 {
			// Agent finished without tool call - extract final assessment from content
			assessment, err := inv.extractFinalAssessment(choice.Message.Content, &investigation)
			if err != nil {
				// Agent stopped without valid assessment
				return inv.emergencyAssessment(&investigation, fmt.Sprintf("Agent stopped without valid assessment: %v", err)), nil
			}
			investigation.Hops = append(investigation.Hops, hopResult)
			investigation.CompletedAt = time.Now()
			investigation.StoppingReason = "Agent completed investigation"
			assessment.Investigation = &investigation
			return assessment, nil
		}

		// Execute tool calls
		// IMPORTANT: Add the assistant's message to the conversation FIRST
		// This is required by OpenAI API - the assistant message must come before tool responses
		messages = append(messages, choice.Message.ToParam())

		// Now execute each tool and collect responses
		for _, toolCall := range toolCalls {
			// Check if this is finish_investigation
			if toolCall.Function.Name == "finish_investigation" {
				assessment, err := inv.handleFinishInvestigation(toolCall.Function.Arguments, &investigation)
				if err != nil {
					return inv.emergencyAssessment(&investigation, fmt.Sprintf("Failed to parse finish_investigation: %v", err)), nil
				}
				investigation.Hops = append(investigation.Hops, hopResult)
				investigation.CompletedAt = time.Now()
				investigation.StoppingReason = "Agent called finish_investigation"
				assessment.Investigation = &investigation
				return assessment, nil
			}

			// Execute other tools
			result, err := inv.executeTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments)

			// Add tool result to messages using ToolMessage helper
			messages = append(messages, openai.ToolMessage(inv.formatToolResult(result, err), toolCall.ID))

			// Record tool call in hop
			hopResult.NodesVisited = append(hopResult.NodesVisited, toolCall.Function.Name)
		}

		investigation.Hops = append(investigation.Hops, hopResult)
	}

	// Hit max hops - return emergency assessment
	investigation.CompletedAt = time.Now()
	investigation.StoppingReason = fmt.Sprintf("Hit maximum hops (%d)", inv.maxHops)
	return inv.emergencyAssessment(&investigation, "Investigation exceeded maximum hops"), nil
}

// getToolDefinitions returns OpenAI function definitions
func (inv *RiskInvestigator) getToolDefinitions() []openai.ChatCompletionToolUnionParam {
	return []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "query_ownership",
			Description: openai.String("Find if code owner is still active (stale ownership = incident risk). Returns developers who have modified the file."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query (current + historical)",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "query_cochange_partners",
			Description: openai.String("Find files that usually change together (incomplete changes = incident risk). Useful for detecting forgotten updates."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query",
						"items": map[string]any{
							"type": "string",
						},
					},
					"frequency_threshold": map[string]any{
						"type":        "number",
						"description": "Minimum co-change frequency (0.0-1.0). Default: 0.5",
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "query_blast_radius",
			Description: openai.String("Find downstream files that depend on the target file. Shows potential impact of changes."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "File path to check dependencies for",
					},
				},
				"required": []string{"file_path"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "query_recent_commits",
			Description: openai.String("Get recent commits that modified the file. Shows recent activity and ownership trends."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query",
						"items": map[string]any{
							"type": "string",
						},
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Number of commits to return. Default: 5",
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "get_incidents_with_context",
			Description: openai.String("Get full incident history with issue details, link quality, and author roles. Returns comprehensive incident data including titles, bodies, confidence scores, and whether reporters were team members or external users."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query",
						"items": map[string]any{
							"type": "string",
						},
					},
					"days_back": map[string]any{
						"type":        "number",
						"description": "How many days back to search. Default: 180",
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "get_ownership_timeline",
			Description: openai.String("Get developer ownership history with activity status. Shows who owns the code, when they last contributed, and whether they're still active. Useful for detecting stale ownership and bus factor risks."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "get_cochange_with_explanations",
			Description: openai.String("Get files that frequently change together WITH sample commit messages explaining why. Useful for detecting incomplete changes and related file updates that were forgotten."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths to query",
						"items": map[string]any{
							"type": "string",
						},
					},
					"threshold": map[string]any{
						"type":        "number",
						"description": "Minimum co-change frequency (0.0-1.0). Default: 0.5",
					},
				},
				"required": []string{"file_paths"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "get_blast_radius_analysis",
			Description: openai.String("Get downstream files that depend on this file AND their incident history. Shows potential impact of changes and which dependent files are fragile."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "File path to analyze blast radius for",
					},
				},
				"required": []string{"file_path"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "get_commit_patch",
			Description: openai.String("Retrieve full code diff/patch for a specific commit. Use when you need to see actual code changes. Warning: can be large, only call when necessary."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"commit_sha": map[string]any{
						"type":        "string",
						"description": "Git commit SHA to retrieve patch for",
					},
				},
				"required": []string{"commit_sha"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "finish_investigation",
			Description: openai.String("Complete the investigation and return final risk assessment with incident-focused reasoning."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"risk_level": map[string]any{
						"type":        "string",
						"description": "Risk level: LOW, MEDIUM, HIGH, or CRITICAL",
						"enum":        []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"},
					},
					"confidence": map[string]any{
						"type":        "number",
						"description": "Confidence score 0.0-1.0",
					},
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Detailed explanation of the risk assessment focusing on incident risk",
					},
					"recommendations": map[string]any{
						"type":        "array",
						"description": "List of specific actions to mitigate risk",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"risk_level", "confidence", "reasoning", "recommendations"},
			},
		}),
	}
}

// executeTool executes a tool call
func (inv *RiskInvestigator) executeTool(ctx context.Context, toolName string, argsJSON string) (any, error) {
	// Parse arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	switch toolName {
	case "query_ownership":
		return inv.queryOwnership(ctx, args)
	case "query_cochange_partners":
		return inv.queryCoChangePartners(ctx, args)
	case "query_blast_radius":
		return inv.queryBlastRadius(ctx, args)
	case "query_recent_commits":
		return inv.queryRecentCommits(ctx, args)
	case "get_incidents_with_context":
		return inv.getIncidentsWithContext(ctx, args)
	case "get_ownership_timeline":
		return inv.getOwnershipTimeline(ctx, args)
	case "get_cochange_with_explanations":
		return inv.getCoChangeWithExplanations(ctx, args)
	case "get_blast_radius_analysis":
		return inv.getBlastRadiusAnalysis(ctx, args)
	case "get_commit_patch":
		return inv.getCommitPatch(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// Tool implementations

func (inv *RiskInvestigator) queryOwnership(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required")
	}

	query := `
		MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File)
		WHERE f.path IN $paths
		WITH d, COUNT(c) as commit_count
		ORDER BY commit_count DESC
		LIMIT 5
		RETURN d.email as developer, commit_count
	`

	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths": filePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

func (inv *RiskInvestigator) queryCoChangePartners(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	threshold := 0.5
	if t, ok := args["frequency_threshold"].(float64); ok {
		threshold = t
	}

	query := `
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)-[:MODIFIED]->(f2:File)
		WHERE f1.path IN $paths AND f1.path <> f2.path
		WITH f1.path as file1, f2.path as file2, COUNT(c) as cochange_count
		MATCH (f1:File {path: file1})
		MATCH (f2:File {path: file2})
		WITH file1, file2, cochange_count,
		     size((f1)<-[:MODIFIED]-()) as f1_total,
		     size((f2)<-[:MODIFIED]-()) as f2_total
		WITH file1, file2, cochange_count,
		     toFloat(cochange_count) / toFloat(f1_total) as frequency
		WHERE frequency >= $threshold
		ORDER BY frequency DESC
		LIMIT 10
		RETURN file2 as partner_file, frequency
	`

	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths":     filePaths,
		"threshold": threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

func (inv *RiskInvestigator) queryIncidentHistory(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	daysBack := 180
	if d, ok := args["days_back"].(float64); ok {
		daysBack = int(d)
	}

	query := `
		MATCH (issue:Issue)-[rel]->(pr:PR)<-[:IN_PR]-(c:Commit)-[:MODIFIED]->(f:File)
		WHERE f.path IN $paths
		  AND (type(rel) = 'FIXED_BY' OR type(rel) = 'ASSOCIATED_WITH')
		  AND issue.created_at > datetime() - duration({days: $days_back})
		RETURN issue.number as incident_number,
		       issue.title as title,
		       issue.body as body,
		       issue.labels as labels,
		       issue.created_at as created_at,
		       issue.closed_at as closed_at,
		       c.sha as fix_commit,
		       pr.number as pr_number,
		       type(rel) as link_type,
		       rel.confidence as confidence,
		       rel.detection_method as detection_method,
		       rel.evidence_sources as evidence
		ORDER BY rel.confidence DESC, issue.created_at DESC
		LIMIT 10
	`

	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths":     filePaths,
		"days_back": daysBack,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

func (inv *RiskInvestigator) queryBlastRadius(ctx context.Context, args map[string]any) (any, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path is required")
	}

	query := `
		MATCH (f1:File {path: $path})<-[:DEPENDS_ON]-(f2:File)
		RETURN f2.path as dependent_file
		LIMIT 20
	`

	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"path": filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

func (inv *RiskInvestigator) queryRecentCommits(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	query := `
		MATCH (c:Commit)-[:MODIFIED]->(f:File)
		WHERE f.path IN $paths
		MATCH (d:Developer)-[:AUTHORED]->(c)
		RETURN c.sha as commit_sha,
		       c.message as message,
		       c.timestamp as timestamp,
		       d.email as author
		ORDER BY c.timestamp DESC
		LIMIT $limit
	`

	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths": filePaths,
		"limit": limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return results, nil
}

func (inv *RiskInvestigator) getCommitPatch(ctx context.Context, args map[string]any) (any, error) {
	commitSHA, ok := args["commit_sha"].(string)
	if !ok || commitSHA == "" {
		return nil, fmt.Errorf("commit_sha is required")
	}

	patch, err := inv.postgresAdapter.GetCommitPatch(ctx, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve patch: %w", err)
	}

	return map[string]any{
		"commit_sha": commitSHA,
		"patch":      patch,
	}, nil
}

// Hybrid query tool handlers

func (inv *RiskInvestigator) getIncidentsWithContext(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required")
	}

	daysBack := 180
	if d, ok := args["days_back"].(float64); ok {
		daysBack = int(d)
	}

	incidents, err := inv.hybridClient.GetIncidentHistoryForFiles(ctx, filePaths, daysBack)
	if err != nil {
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	return incidents, nil
}

func (inv *RiskInvestigator) getOwnershipTimeline(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required")
	}

	ownership, err := inv.hybridClient.GetOwnershipHistoryForFiles(ctx, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to get ownership: %w", err)
	}

	return ownership, nil
}

func (inv *RiskInvestigator) getCoChangeWithExplanations(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required")
	}

	threshold := 0.5
	if t, ok := args["threshold"].(float64); ok {
		threshold = t
	}

	partners, err := inv.hybridClient.GetCoChangePartnersWithContext(ctx, filePaths, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get co-change partners: %w", err)
	}

	return partners, nil
}

func (inv *RiskInvestigator) getBlastRadiusAnalysis(ctx context.Context, args map[string]any) (any, error) {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	blastRadius, err := inv.hybridClient.GetBlastRadiusWithIncidents(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get blast radius: %w", err)
	}

	return blastRadius, nil
}

// handleFinishInvestigation processes the finish_investigation tool call
func (inv *RiskInvestigator) handleFinishInvestigation(argsJSON string, investigation *Investigation) (*RiskAssessment, error) {
	var args struct {
		RiskLevel       string   `json:"risk_level"`
		Confidence      float64  `json:"confidence"`
		Reasoning       string   `json:"reasoning"`
		Recommendations []string `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse finish_investigation args: %w", err)
	}

	// Parse risk level
	var riskLevel RiskLevel
	switch args.RiskLevel {
	case "CRITICAL":
		riskLevel = RiskCritical
	case "HIGH":
		riskLevel = RiskHigh
	case "MEDIUM":
		riskLevel = RiskMedium
	case "LOW":
		riskLevel = RiskLow
	default:
		riskLevel = RiskMedium
	}

	riskScore := riskLevelToScore(riskLevel)

	return &RiskAssessment{
		RiskLevel:       riskLevel,
		RiskScore:       riskScore,
		Confidence:      args.Confidence,
		Summary:         args.Reasoning,
		Recommendations: args.Recommendations,
	}, nil
}

// extractFinalAssessment tries to extract assessment from text response
func (inv *RiskInvestigator) extractFinalAssessment(content string, investigation *Investigation) (*RiskAssessment, error) {
	// Try to parse as JSON
	var args struct {
		RiskLevel       string   `json:"risk_level"`
		Confidence      float64  `json:"confidence"`
		Reasoning       string   `json:"reasoning"`
		Recommendations []string `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(content), &args); err != nil {
		return nil, fmt.Errorf("content is not valid JSON assessment")
	}

	return inv.handleFinishInvestigation(content, investigation)
}

// emergencyAssessment returns a conservative assessment when agent fails
func (inv *RiskInvestigator) emergencyAssessment(investigation *Investigation, reason string) *RiskAssessment {
	investigation.StoppingReason = reason

	return &RiskAssessment{
		RiskLevel:  RiskMedium,
		RiskScore:  0.5,
		Confidence: 0.3,
		Summary:    fmt.Sprintf("Investigation incomplete: %s. Using conservative MEDIUM risk assessment.", reason),
		Recommendations: []string{
			"Review changes manually due to incomplete investigation",
			"Contact file owner for additional review",
			"Verify test coverage before committing",
		},
		Investigation: investigation,
	}
}

// formatToolResult formats tool execution result for LLM
func (inv *RiskInvestigator) formatToolResult(result any, err error) string {
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("ERROR: Failed to format result: %v", err)
	}

	return string(jsonBytes)
}

// Helper functions

func extractStringArray(args map[string]any, key string) []string {
	val, ok := args[key]
	if !ok {
		return nil
	}

	arr, ok := val.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}

	return result
}

// riskLevelToScore is defined in simple_investigator.go - no need to redefine
