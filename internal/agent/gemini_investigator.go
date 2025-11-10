package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
	"google.golang.org/genai"
)

// GeminiInvestigator performs agent-based risk investigation using Gemini tool calling
type GeminiInvestigator struct {
	geminiClient    *llm.GeminiClient
	graphClient     GraphQueryExecutor
	postgresAdapter PostgresQueryExecutor
	hybridClient    *database.HybridClient
	historyManager  *HistoryManager
	maxHops         int
}

// NewGeminiInvestigator creates a new Gemini-based risk investigator
func NewGeminiInvestigator(
	geminiClient *llm.GeminiClient,
	graphClient GraphQueryExecutor,
	postgresAdapter PostgresQueryExecutor,
	hybridClient *database.HybridClient,
) *GeminiInvestigator {
	return &GeminiInvestigator{
		geminiClient:    geminiClient,
		graphClient:     graphClient,
		postgresAdapter: postgresAdapter,
		hybridClient:    hybridClient,
		historyManager:  NewHistoryManager(),
		maxHops:         30, // High safety limit - agent should call finish_investigation when done (12 Factor Agents principle)
	}
}

// Investigate performs agent-based risk investigation using Gemini
func (inv *GeminiInvestigator) Investigate(ctx context.Context, kickoffPrompt string) (*RiskAssessment, error) {
	// Initialize conversation history
	history := []*genai.Content{}

	var investigation Investigation
	investigation.Request = InvestigationRequest{
		StartedAt: time.Now(),
	}

	// Agent loop
	for hop := 1; hop <= inv.maxHops; hop++ {
		hopStart := time.Now()

		// Call Gemini with tools
		var resp *genai.GenerateContentResponse
		var err error

		if hop == 1 {
			// First hop: use simple call with kickoff prompt
			resp, err = inv.geminiClient.CompleteWithTools(ctx, "", kickoffPrompt, inv.getGeminiTools())
		} else {
			// Subsequent hops: use history-aware call
			resp, err = inv.geminiClient.CompleteWithToolsAndHistory(ctx, "", history, inv.getGeminiTools())
		}

		if err != nil {
			return nil, fmt.Errorf("Gemini request failed at hop %d: %w", hop, err)
		}

		if len(resp.Candidates) == 0 {
			return nil, fmt.Errorf("no response from Gemini at hop %d", hop)
		}

		candidate := resp.Candidates[0]

		// Estimate tokens (Gemini SDK doesn't expose usage in the same way)
		// For hop 1, estimate based on kickoff prompt; for others, estimate history size
		tokensEstimate := len(kickoffPrompt) / 4
		if hop > 1 {
			tokensEstimate = len(history) * 100 // Rough estimate per history item
		}
		investigation.TotalTokens += tokensEstimate

		// Record hop
		hopResult := HopResult{
			HopNumber:  hop,
			Query:      "",
			Response:   extractTextFromParts(candidate.Content.Parts),
			TokensUsed: tokensEstimate,
			Duration:   time.Since(hopStart),
		}

		// Check for tool calls (function calls in Gemini)
		if len(candidate.Content.Parts) == 0 {
			return nil, fmt.Errorf("empty response from Gemini at hop %d", hop)
		}

		// Extract tool calls from response
		var toolCalls []genai.FunctionCall
		var textResponse string

		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				toolCalls = append(toolCalls, *part.FunctionCall)
			}
			if part.Text != "" {
				textResponse += part.Text
			}
		}

		// If no tool calls, agent finished
		if len(toolCalls) == 0 {
			// Agent finished - extract final assessment from text
			assessment, err := inv.extractFinalAssessment(textResponse, &investigation)
			if err != nil {
				return inv.emergencyAssessment(&investigation, fmt.Sprintf("Agent stopped without valid assessment: %v", err)), nil
			}
			investigation.Hops = append(investigation.Hops, hopResult)
			investigation.CompletedAt = time.Now()
			investigation.StoppingReason = "Agent completed investigation"
			assessment.Investigation = &investigation
			return assessment, nil
		}

		// Add assistant message to history
		history = append(history, candidate.Content)

		// Execute tool calls and collect responses
		functionResponses := []*genai.FunctionResponse{}

		for _, toolCall := range toolCalls {
			// Check if this is finish_investigation
			if toolCall.Name == "finish_investigation" {
				argsJSON, _ := json.Marshal(toolCall.Args)
				assessment, err := inv.handleFinishInvestigation(string(argsJSON), &investigation)
				if err != nil {
					return inv.emergencyAssessment(&investigation, fmt.Sprintf("Failed to parse finish_investigation: %v", err)), nil
				}
				investigation.Hops = append(investigation.Hops, hopResult)
				investigation.CompletedAt = time.Now()
				investigation.StoppingReason = "Agent called finish_investigation"
				assessment.Investigation = &investigation
				return assessment, nil
			}

			// Execute tool
			result, err := inv.executeToolGemini(ctx, toolCall.Name, toolCall.Args)

			// Store tool result in hop for transparency
			toolResult := ToolResult{
				ToolName: toolCall.Name,
				Args:     toolCall.Args,
				Result:   result,
			}
			if err != nil {
				toolResult.Error = err.Error()
			}
			hopResult.ToolResults = append(hopResult.ToolResults, toolResult)

			// Create function response
			functionResponses = append(functionResponses, &genai.FunctionResponse{
				Name:     toolCall.Name,
				Response: map[string]any{"result": inv.formatToolResultGemini(result, err)},
			})

			// Record tool call in hop
			hopResult.NodesVisited = append(hopResult.NodesVisited, toolCall.Name)
		}

		// Add function responses to history
		history = append(history, &genai.Content{
			Role:  "user",
			Parts: convertFunctionResponsesToParts(functionResponses),
		})

		// Prune history to stay within token budget
		history = inv.historyManager.PruneHistory(history)

		investigation.Hops = append(investigation.Hops, hopResult)
	}

	// Hit max hops
	investigation.CompletedAt = time.Now()
	investigation.StoppingReason = fmt.Sprintf("Hit maximum hops (%d)", inv.maxHops)
	return inv.emergencyAssessment(&investigation, "Investigation exceeded maximum hops"), nil
}

// getGeminiTools returns Gemini function declarations
func (inv *GeminiInvestigator) getGeminiTools() []*genai.Tool {
	return []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "query_ownership",
					Description: "Find if code owner is still active (stale ownership = incident risk). Returns developers who have modified the file.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query (current + historical)",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "query_cochange_partners",
					Description: "Find files that usually change together (incomplete changes = incident risk). Useful for detecting forgotten updates.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
							"frequency_threshold": {
								Type:        genai.TypeNumber,
								Description: "Minimum co-change frequency (0.0-1.0). Default: 0.2",
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "query_blast_radius",
					Description: "Find downstream files that depend on the target file. Shows potential impact of changes.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_path": {
								Type:        genai.TypeString,
								Description: "File path to check dependencies for",
							},
						},
						Required: []string{"file_path"},
					},
				},
				{
					Name:        "query_recent_commits",
					Description: "Get recent commits that modified the file. Shows recent activity and ownership trends.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
							"limit": {
								Type:        genai.TypeNumber,
								Description: "Number of commits to return. Default: 5",
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "get_incidents_with_context",
					Description: "Get full incident history with issue details, link quality, and author roles. Returns comprehensive incident data including titles, bodies, confidence scores, and whether reporters were team members or external users.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
							"days_back": {
								Type:        genai.TypeNumber,
								Description: "How many days back to search. Default: 180",
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "get_ownership_timeline",
					Description: "Get developer ownership history with activity status. Shows who owns the code, when they last contributed, and whether they're still active.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "get_cochange_with_explanations",
					Description: "Get co-change partners with commit message context showing WHY files changed together.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_paths": {
								Type:        genai.TypeArray,
								Description: "Array of file paths to query",
								Items: &genai.Schema{
									Type: genai.TypeString,
								},
							},
							"threshold": {
								Type:        genai.TypeNumber,
								Description: "Minimum co-change frequency (0.0-1.0). Default: 0.2",
							},
						},
						Required: []string{"file_paths"},
					},
				},
				{
					Name:        "get_blast_radius_analysis",
					Description: "Get downstream impact analysis with dependency counts and risk assessment.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_path": {
								Type:        genai.TypeString,
								Description: "File path to analyze blast radius for",
							},
						},
						Required: []string{"file_path"},
					},
				},
				{
					Name:        "finish_investigation",
					Description: "Complete the investigation and return final risk assessment. MUST be called when you have gathered sufficient evidence.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"risk_level": {
								Type:        genai.TypeString,
								Description: "Overall risk level: HIGH, MEDIUM, or LOW",
								Enum:        []string{"HIGH", "MEDIUM", "LOW"},
							},
							"confidence": {
								Type:        genai.TypeNumber,
								Description: "Confidence score (0.0-1.0)",
							},
							"summary": {
								Type:        genai.TypeString,
								Description: "Brief summary of findings and risk rationale",
							},
						},
						Required: []string{"risk_level", "confidence", "summary"},
					},
				},
			},
		},
	}
}

// executeToolGemini executes a tool call from Gemini (args are map[string]any)
func (inv *GeminiInvestigator) executeToolGemini(ctx context.Context, toolName string, args map[string]any) (any, error) {
	slog.Info("executing tool", "tool", toolName, "args", args)

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
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// formatToolResultGemini formats tool results for Gemini
func (inv *GeminiInvestigator) formatToolResultGemini(result any, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Convert result to JSON for structured data
	resultJSON, jsonErr := json.MarshalIndent(result, "", "  ")
	if jsonErr != nil {
		return fmt.Sprintf("Result: %v", result)
	}

	return string(resultJSON)
}

// Helper functions

func extractTextFromParts(parts []*genai.Part) string {
	var text string
	for _, part := range parts {
		if part.Text != "" {
			text += part.Text
		}
	}
	return text
}

func convertFunctionResponsesToParts(responses []*genai.FunctionResponse) []*genai.Part {
	parts := make([]*genai.Part, len(responses))
	for i, resp := range responses {
		parts[i] = &genai.Part{
			FunctionResponse: resp,
		}
	}
	return parts
}
func (inv *GeminiInvestigator) queryOwnership(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) queryCoChangePartners(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	slog.Info("query_cochange_partners called", "file_paths", filePaths)

	threshold := 0.2 // 20% co-change frequency threshold (lowered from 0.5 which was too strict)
	if t, ok := args["frequency_threshold"].(float64); ok {
		threshold = t
	}

	query := `
		MATCH (f1:File)<-[:MODIFIED]-(c:Commit)-[:MODIFIED]->(f2:File)
		WHERE f1.path IN $paths AND f1.path <> f2.path
		WITH f1.path as file1, f2.path as file2, COUNT(c) as cochange_count
		WITH file1, file2, cochange_count
		MATCH (f1:File {path: file1})
		WITH file2 as partner_file, cochange_count, COUNT { (f1)<-[:MODIFIED]-() } as f1_total
		WITH partner_file, cochange_count, toFloat(cochange_count) / toFloat(f1_total) as frequency
		WHERE frequency >= $threshold
		RETURN partner_file, frequency
		ORDER BY frequency DESC
		LIMIT 10
	`

	slog.Info("calling graphClient.ExecuteQuery for co-change", "threshold", threshold)
	results, err := inv.graphClient.ExecuteQuery(ctx, query, map[string]any{
		"paths":     filePaths,
		"threshold": threshold,
	})
	if err != nil {
		slog.Error("query_cochange_partners failed", "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}

	slog.Info("query_cochange_partners result", "file_paths", filePaths, "result_count", len(results))
	if len(results) > 0 {
		slog.Info("sample result", "partner_file", results[0]["partner_file"], "frequency", results[0]["frequency"])
	}
	return results, nil
}

func (inv *GeminiInvestigator) queryIncidentHistory(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) queryBlastRadius(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) queryRecentCommits(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) getCommitPatch(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) getIncidentsWithContext(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	slog.Info("get_incidents_with_context called", "file_paths", filePaths)

	if len(filePaths) == 0 {
		slog.Warn("get_incidents_with_context: no file paths provided")
		return nil, fmt.Errorf("file_paths is required")
	}

	daysBack := 180
	if d, ok := args["days_back"].(float64); ok {
		daysBack = int(d)
	}

	slog.Info("calling hybridClient.GetIncidentHistoryForFiles", "days_back", daysBack)
	incidents, err := inv.hybridClient.GetIncidentHistoryForFiles(ctx, filePaths, daysBack)
	if err != nil {
		slog.Error("get_incidents_with_context failed", "error", err)
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	// Log incident count for debugging
	slog.Info("get_incidents_with_context result", "file_paths", filePaths, "incident_count", len(incidents), "days_back", daysBack)
	if len(incidents) > 0 {
		slog.Info("sample incident", "issue_number", incidents[0].IssueNumber, "issue_title", incidents[0].IssueTitle)
	}

	return incidents, nil
}

func (inv *GeminiInvestigator) getOwnershipTimeline(ctx context.Context, args map[string]any) (any, error) {
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

func (inv *GeminiInvestigator) getCoChangeWithExplanations(ctx context.Context, args map[string]any) (any, error) {
	filePaths := extractStringArray(args, "file_paths")
	slog.Info("get_cochange_with_explanations called", "file_paths", filePaths)

	if len(filePaths) == 0 {
		slog.Warn("get_cochange_with_explanations: no file paths provided")
		return nil, fmt.Errorf("file_paths is required")
	}

	threshold := 0.2 // 20% co-change frequency threshold (lowered from 0.5 which was too strict)
	if t, ok := args["threshold"].(float64); ok {
		threshold = t
	}

	slog.Info("calling hybridClient.GetCoChangePartnersWithContext", "threshold", threshold)
	partners, err := inv.hybridClient.GetCoChangePartnersWithContext(ctx, filePaths, threshold)
	if err != nil {
		slog.Error("get_cochange_with_explanations failed", "error", err)
		return nil, fmt.Errorf("failed to get co-change partners: %w", err)
	}

	slog.Info("get_cochange_with_explanations result", "file_paths", filePaths, "partner_count", len(partners))
	return partners, nil
}

func (inv *GeminiInvestigator) getBlastRadiusAnalysis(ctx context.Context, args map[string]any) (any, error) {
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
func (inv *GeminiInvestigator) handleFinishInvestigation(argsJSON string, investigation *Investigation) (*RiskAssessment, error) {
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
func (inv *GeminiInvestigator) extractFinalAssessment(content string, investigation *Investigation) (*RiskAssessment, error) {
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
func (inv *GeminiInvestigator) emergencyAssessment(investigation *Investigation, reason string) *RiskAssessment {
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
func (inv *GeminiInvestigator) formatToolResult(result any, err error) string {
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

