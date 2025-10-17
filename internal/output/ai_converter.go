package output

import (
	"context"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/metrics"
	"github.com/rohankatakam/coderisk/internal/models"
)

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
	output.AIAssistantActions = GenerateAIActions(phase1, riskResult)

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
