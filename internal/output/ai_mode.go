package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/coderisk/coderisk-go/internal/models"
)

// AIFormatter outputs machine-readable JSON for AI assistants
// Schema v1.0: https://coderisk.com/schemas/ai-mode/v1.0.json
type AIFormatter struct {
	Version string
}

// NewAIFormatter creates a new AI mode formatter
func NewAIFormatter() *AIFormatter {
	return &AIFormatter{Version: "1.0"}
}

// Format outputs structured JSON following AI Mode schema v1.0
func (f *AIFormatter) Format(result *models.RiskResult, w io.Writer) error {
	output := f.buildAIModeOutput(result)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AIFormatter) buildAIModeOutput(result *models.RiskResult) map[string]interface{} {
	return map[string]interface{}{
		"meta":                                f.buildMeta(result),
		"risk":                                f.buildRisk(result),
		"files":                               f.buildFiles(result),
		"graph_analysis":                      f.buildGraphAnalysis(result),
		"investigation_trace":                 f.buildTrace(result),
		"recommendations":                     f.buildRecommendations(result),
		"ai_assistant_actions":                f.buildAIActions(result),
		"contextual_insights":                 f.buildInsights(result),
		"performance":                         f.buildPerformance(result),
		"should_block_commit":                 result.ShouldBlock,
		"block_reason":                        result.BlockReason,
		"override_allowed":                    result.OverrideAllowed,
		"override_requires_justification":     result.OverrideRequiresJustification,
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
