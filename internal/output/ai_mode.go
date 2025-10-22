package output

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/metrics"
	"github.com/rohankatakam/coderisk/internal/models"
)

// Helper functions (moved from ai_converter.go)
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// AIFormatter outputs machine-readable JSON for AI assistants
// Schema v1.0: https://coderisk.com/schemas/ai-mode/v1.0.json
// 12-factor: Factor 4 - Tools are structured outputs
type AIFormatter struct {
	Version     string
	Phase1      *metrics.Phase1Result
	GraphClient *graph.Client
}

// NewAIFormatter creates a new AI mode formatter
func NewAIFormatter() *AIFormatter {
	return &AIFormatter{Version: "1.0"}
}

// SetPhase1Result sets the Phase 1 result for enhanced analysis
func (f *AIFormatter) SetPhase1Result(phase1 *metrics.Phase1Result) {
	f.Phase1 = phase1
}

// SetGraphClient sets the graph client for graph analysis
func (f *AIFormatter) SetGraphClient(client *graph.Client) {
	f.GraphClient = client
}

// Format outputs structured JSON following AI Mode schema v1.0
// 12-factor: Factor 4 - Tools are structured outputs
func (f *AIFormatter) Format(result *models.RiskResult, w io.Writer) error {
	var output interface{}

	// Use new ToAIMode converter if Phase1 result is available
	if f.Phase1 != nil {
		ctx := context.Background()
		aiOutput := ToAIMode(ctx, f.Phase1, result, f.GraphClient)
		output = aiOutput
	} else {
		// Fallback to legacy output format
		output = f.buildAIModeOutput(result)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AIFormatter) buildAIModeOutput(result *models.RiskResult) map[string]interface{} {
	return map[string]interface{}{
		"meta":                            f.buildMeta(result),
		"risk":                            f.buildRisk(result),
		"files":                           f.buildFiles(result),
		"graph_analysis":                  f.buildGraphAnalysis(result),
		"investigation_trace":             f.buildTrace(result),
		"recommendations":                 f.buildRecommendations(result),
		"ai_assistant_actions":            f.buildAIActions(result),
		"contextual_insights":             f.buildInsights(result),
		"performance":                     f.buildPerformance(result),
		"should_block_commit":             result.ShouldBlock,
		"block_reason":                    result.BlockReason,
		"override_allowed":                result.OverrideAllowed,
		"override_requires_justification": result.OverrideRequiresJustification,
	}
}

// buildMeta creates the meta section (request context)
func (f *AIFormatter) buildMeta(result *models.RiskResult) map[string]interface{} {
	meta := map[string]interface{}{
		"version":        f.Version,
		"timestamp":      time.Now().Format(time.RFC3339),
		"duration_ms":    result.Duration.Milliseconds(),
		"branch":         result.Branch,
		"files_analyzed": result.FilesChanged,
		"agent_hops":     len(result.InvestigationTrace),
		"cache_hit":      result.CacheHit,
	}

	return meta
}

// buildRisk creates the risk section (summary assessment)
func (f *AIFormatter) buildRisk(result *models.RiskResult) map[string]interface{} {
	return map[string]interface{}{
		"level":      result.RiskLevel,
		"score":      result.RiskScore,
		"confidence": result.Confidence,
	}
}

// buildFiles creates the files section (per-file details)
func (f *AIFormatter) buildFiles(result *models.RiskResult) []map[string]interface{} {
	files := []map[string]interface{}{}

	for _, file := range result.Files {
		// Convert metrics map to proper format
		metrics := map[string]interface{}{}
		for key, metric := range file.Metrics {
			metrics[key] = metric.Value
		}

		// Filter issues for this file
		fileIssues := []models.RiskIssue{}
		for _, issue := range result.Issues {
			if issue.File == file.Path {
				fileIssues = append(fileIssues, issue)
			}
		}

		fileData := map[string]interface{}{
			"path":          file.Path,
			"language":      file.Language,
			"lines_changed": file.LinesChanged,
			"risk_score":    file.RiskScore,
			"metrics":       metrics,
			"issues":        f.transformIssues(fileIssues),
		}
		files = append(files, fileData)
	}

	return files
}

// transformIssues converts issues to AI mode format
func (f *AIFormatter) transformIssues(issues []models.RiskIssue) []map[string]interface{} {
	transformed := []map[string]interface{}{}

	for _, issue := range issues {
		issueData := map[string]interface{}{
			"id":                     issue.ID,
			"severity":               issue.Severity,
			"category":               issue.Category,
			"line_start":             issue.LineStart,
			"line_end":               issue.LineEnd,
			"function":               issue.Function,
			"message":                issue.Message,
			"impact_score":           issue.ImpactScore,
			"fix_priority":           issue.FixPriority,
			"estimated_fix_time_min": issue.EstimatedFixTimeMin,
			"auto_fixable":           issue.AutoFixable,
			"fix_command":            issue.FixCommand,
		}
		transformed = append(transformed, issueData)
	}

	return transformed
}

// buildGraphAnalysis creates the graph_analysis section
func (f *AIFormatter) buildGraphAnalysis(result *models.RiskResult) map[string]interface{} {
	// Build blast radius info
	blastRadius := map[string]interface{}{
		"total_affected_files": result.BlastRadius,
	}

	// Build temporal coupling array
	temporalCoupling := []map[string]interface{}{}
	for _, coupling := range result.TemporalCoupling {
		temporalCoupling = append(temporalCoupling, map[string]interface{}{
			"file_a":         coupling.FileA,
			"file_b":         coupling.FileB,
			"strength":       coupling.Strength,
			"commits":        coupling.Commits,
			"total_commits":  coupling.TotalCommits,
			"window_days":    coupling.WindowDays,
			"last_co_change": coupling.LastCoChange.Format(time.RFC3339),
		})
	}

	// Build hotspots array
	hotspots := []map[string]interface{}{}
	for _, hotspot := range result.Hotspots {
		hotspots = append(hotspots, map[string]interface{}{
			"file":      hotspot.File,
			"score":     hotspot.Score,
			"reason":    hotspot.Reason,
			"churn":     hotspot.Churn,
			"coverage":  hotspot.Coverage,
			"incidents": hotspot.Incidents,
		})
	}

	return map[string]interface{}{
		"blast_radius":      blastRadius,
		"temporal_coupling": temporalCoupling,
		"hotspots":          hotspots,
	}
}

// buildTrace creates the investigation_trace section
func (f *AIFormatter) buildTrace(result *models.RiskResult) []map[string]interface{} {
	trace := []map[string]interface{}{}

	for i, hop := range result.InvestigationTrace {
		// Convert metrics to simple map
		metrics := []string{}
		for _, metric := range hop.Metrics {
			metrics = append(metrics, metric.Name)
		}

		// Convert changed entities
		changedEntities := []string{}
		for _, entity := range hop.ChangedEntities {
			changedEntities = append(changedEntities, entity.Name)
		}

		hopData := map[string]interface{}{
			"hop":                i + 1,
			"node_type":          hop.NodeType,
			"node_id":            hop.NodeID,
			"action":             "analyze_" + hop.NodeType,
			"metrics_calculated": metrics,
			"decision":           hop.Decision,
			"reasoning":          hop.Reasoning,
			"confidence":         hop.Confidence,
			"duration_ms":        hop.DurationMS,
		}
		trace = append(trace, hopData)
	}

	return trace
}

// buildRecommendations creates the recommendations section
func (f *AIFormatter) buildRecommendations(result *models.RiskResult) map[string]interface{} {
	// For now, group recommendations by priority based on severity
	critical := []map[string]interface{}{}
	high := []map[string]interface{}{}
	medium := []map[string]interface{}{}

	for _, rec := range result.Recommendations {
		// Parse recommendation (simplified - would need smarter parsing in production)
		recItem := map[string]interface{}{
			"action": rec,
			"target": "", // Would extract from rec text
			"reason": "", // Would extract from rec text
		}

		// Categorize based on keywords (simplified)
		if contains(rec, "critical") || contains(rec, "security") {
			critical = append(critical, recItem)
		} else if contains(rec, "test") || contains(rec, "coverage") {
			high = append(high, recItem)
		} else {
			medium = append(medium, recItem)
		}
	}

	return map[string]interface{}{
		"critical": critical,
		"high":     high,
		"medium":   medium,
	}
}

// buildAIActions creates the ai_assistant_actions section
func (f *AIFormatter) buildAIActions(result *models.RiskResult) []map[string]interface{} {
	actions := []map[string]interface{}{}

	for _, issue := range result.Issues {
		if issue.AutoFixable && issue.AIPromptTemplate != "" {
			action := map[string]interface{}{
				"action_type":      issue.FixType,
				"confidence":       issue.FixConfidence,
				"ready_to_execute": issue.FixConfidence > 0.85,
				"prompt":           issue.AIPromptTemplate,
				"expected_files":   issue.ExpectedFiles,
				"estimated_lines":  issue.EstimatedLines,
			}

			if issue.FixConfidence <= 0.85 {
				action["reason"] = "confidence_below_threshold"
			}

			actions = append(actions, action)
		}
	}

	return actions
}

// buildInsights creates the contextual_insights section
func (f *AIFormatter) buildInsights(result *models.RiskResult) map[string]interface{} {
	// Build similar past changes
	similarChanges := []map[string]interface{}{}
	for _, change := range result.SimilarPastChanges {
		similarChanges = append(similarChanges, map[string]interface{}{
			"commit_sha":    change.CommitSHA,
			"date":          change.Date,
			"author":        change.Author,
			"files_changed": change.FilesChanged,
			"outcome":       change.Outcome,
			"lesson":        change.Lesson,
		})
	}

	// Build team patterns
	var teamPatterns map[string]interface{}
	if result.TeamPatterns != nil {
		teamPatterns = map[string]interface{}{
			"avg_test_coverage": result.TeamPatterns.AvgTestCoverage,
			"your_coverage":     result.TeamPatterns.YourCoverage,
			"percentile":        result.TeamPatterns.Percentile,
			"team_avg_coupling": result.TeamPatterns.TeamAvgCoupling,
			"your_coupling":     result.TeamPatterns.YourCoupling,
			"recommendation":    result.TeamPatterns.Recommendation,
		}
	}

	// Build file reputation
	fileReputation := map[string]interface{}{}
	for file, rep := range result.FileReputation {
		fileReputation[file] = map[string]interface{}{
			"incident_density":         rep.IncidentDensity,
			"team_avg":                 rep.TeamAvg,
			"classification":           rep.Classification,
			"extra_review_recommended": rep.ExtraReviewRecommended,
		}
	}

	return map[string]interface{}{
		"similar_past_changes": similarChanges,
		"team_patterns":        teamPatterns,
		"file_reputation":      fileReputation,
	}
}

// buildPerformance creates the performance section
func (f *AIFormatter) buildPerformance(result *models.RiskResult) map[string]interface{} {
	return map[string]interface{}{
		"total_duration_ms": result.Performance.TotalDurationMS,
		"breakdown":         result.Performance.Breakdown,
		"cache_efficiency":  result.Performance.CacheEfficiency,
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// === Functions merged from ai_converter.go ===

// ToAIMode converts risk assessment to AI-consumable JSON
// 12-factor: Factor 4 - Tools are structured outputs
func ToAIMode(ctx context.Context, phase1 *metrics.Phase1Result, riskResult *models.RiskResult, graphClient *graph.Client) *AIJSONOutput {
	// Get repository ID dynamically
	repoID, err := git.GetRepoID()
	if err != nil {
		repoID = "local" // Fallback if not in git repo
	}

	output := &AIJSONOutput{
		Branch:      riskResult.Branch,
		Repository:  repoID,
		Timestamp:   time.Now(),
		OverallRisk: string(phase1.OverallRisk),
	}

	// Generate AI assistant actions (auto-fix prompts)
	// NOTE: Auto-fix deferred to v2 (not in MVP scope)
	output.AIAssistantActions = []AIAssistantAction{}

	// Calculate graph analysis if graph client is available
	if graphClient != nil {
		output.GraphAnalysis = GraphAnalysis{
			BlastRadius:      CalculateBlastRadius(ctx, phase1.FilePath, graphClient),
			TemporalCoupling: GetTemporalCoupling(ctx, phase1.FilePath, graphClient, 0.5),
			Hotspots:         IdentifyHotspots(ctx, phase1, riskResult, graphClient),
		}
	} else {
		// Empty graph analysis if no client
		output.GraphAnalysis = GraphAnalysis{
			BlastRadius:      BlastRadius{},
			TemporalCoupling: []TemporalCouplingPair{},
			Hotspots:         []Hotspot{},
		}
	}

	// Contextual insights
	output.ContextualInsights = ContextualInsights{
		FileReputation: map[string]float64{
			phase1.FilePath: calculateFileReputation(phase1, riskResult),
		},
		SimilarPastChanges: []SimilarChange{}, // TODO: Query from git history
	}

	// Convert file analysis
	output.Files = []FileAnalysis{
		convertToFileAnalysis(phase1, riskResult),
	}

	// Investigation trace (if Phase 2 ran)
	output.InvestigationTrace = convertInvestigationTrace(riskResult)

	// Recommendations
	output.Recommendations = generateRecommendationsFromActions(riskResult, output.AIAssistantActions)

	// Block reason (if HIGH/CRITICAL risk)
	if phase1.OverallRisk == metrics.RiskLevelHigh {
		output.BlockReason = determineBlockReason(phase1, riskResult)
	}

	return output
}

func convertToFileAnalysis(phase1 *metrics.Phase1Result, result *models.RiskResult) FileAnalysis {
	// Find the file in the result
	var fileRisk models.FileRisk
	for _, f := range result.Files {
		if f.Path == phase1.FilePath {
			fileRisk = f
			break
		}
	}

	// Convert metrics to interface{} map
	metricsMap := make(map[string]interface{})
	for key, metric := range fileRisk.Metrics {
		metricsMap[key] = metric.Value
	}

	// Convert issues
	var issues []Issue
	for _, issue := range result.Issues {
		if issue.File == phase1.FilePath {
			issues = append(issues, Issue{
				ID:       issue.ID,
				Severity: issue.Severity,
				Category: issue.Category,
				Message:  issue.Message,
				Line:     issue.LineStart,
				Function: issue.Function,
			})
		}
	}

	return FileAnalysis{
		Path:         phase1.FilePath,
		Language:     fileRisk.Language,
		LinesChanged: fileRisk.LinesChanged,
		RiskScore:    fileRisk.RiskScore,
		Metrics:      metricsMap,
		Issues:       issues,
	}
}

func calculateFileReputation(phase1 *metrics.Phase1Result, result *models.RiskResult) float64 {
	// Reputation = 1.0 - (normalized risk factors)
	reputation := 1.0

	// Penalize for low test coverage
	if phase1.TestRatio != nil {
		reputation -= (1.0 - phase1.TestRatio.Ratio) * 0.3
	}

	// Penalize for high coupling
	if phase1.Coupling != nil {
		couplingPenalty := minFloat64(float64(phase1.Coupling.Count)/20.0, 1.0) * 0.3
		reputation -= couplingPenalty
	}

	// Penalize for high co-change
	if phase1.CoChange != nil {
		reputation -= phase1.CoChange.MaxFrequency * 0.2
	}

	return maxFloat64(reputation, 0.0)
}

func convertInvestigationTrace(result *models.RiskResult) []InvestigationHop {
	trace := make([]InvestigationHop, len(result.InvestigationTrace))
	for i, hop := range result.InvestigationTrace {
		// Convert metrics to MetricResult
		metricResults := make([]MetricResult, len(hop.Metrics))
		for j, metric := range hop.Metrics {
			metricResults[j] = MetricResult{
				Name:  metric.Name,
				Value: metric.Value,
			}
		}

		trace[i] = InvestigationHop{
			Hop:               i + 1,
			NodeType:          hop.NodeType,
			NodeID:            hop.NodeID,
			Action:            "analyze_" + hop.NodeType,
			MetricsCalculated: metricResults,
			Decision:          hop.Decision,
			Reasoning:         hop.Reasoning,
			Confidence:        hop.Confidence,
			DurationMS:        hop.DurationMS,
		}
	}
	return trace
}

func generateRecommendationsFromActions(result *models.RiskResult, actions []AIAssistantAction) Recommendations {
	recs := Recommendations{
		Critical: []Recommendation{},
		High:     []Recommendation{},
		Medium:   []Recommendation{},
		Future:   []Recommendation{},
	}

	// Convert AI actions to recommendations
	for _, action := range actions {
		// Truncate prompt for reason field
		reason := action.Prompt
		if len(reason) > 100 {
			reason = reason[:97] + "..."
		}

		rec := Recommendation{
			Action:        action.Description,
			Reason:        reason,
			EstimatedTime: action.EstimatedLines * 2, // 2 min per line rough estimate
			AutoFixable:   action.ReadyToExecute,
			Priority:      determinePriority(action.Confidence, action.RiskReduction),
		}

		switch rec.Priority {
		case "critical":
			recs.Critical = append(recs.Critical, rec)
		case "high":
			recs.High = append(recs.High, rec)
		case "medium":
			recs.Medium = append(recs.Medium, rec)
		default:
			recs.Future = append(recs.Future, rec)
		}
	}

	// Add recommendations from existing result
	for _, recStr := range result.Recommendations {
		rec := Recommendation{
			Action:        recStr,
			Reason:        "Phase 1 heuristic detected risk",
			EstimatedTime: 30,
			AutoFixable:   false,
			Priority:      "medium",
		}
		recs.Medium = append(recs.Medium, rec)
	}

	return recs
}

func determinePriority(confidence, riskReduction float64) string {
	score := confidence * riskReduction
	if score > 0.5 {
		return "critical"
	} else if score > 0.3 {
		return "high"
	} else if score > 0.15 {
		return "medium"
	}
	return "low"
}

func determineBlockReason(phase1 *metrics.Phase1Result, result *models.RiskResult) string {
	reasons := []string{}

	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		reasons = append(reasons, "High coupling (blast radius)")
	}
	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		reasons = append(reasons, "Critically low test coverage")
	}
	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		reasons = append(reasons, "High temporal coupling detected")
	}

	if len(reasons) == 0 {
		return "High risk detected"
	}

	return strings.Join(reasons, "; ")
}
