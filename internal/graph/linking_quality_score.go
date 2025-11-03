package graph

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
)

// LinkingQualityScore represents the Codebase Linking Quality Score (CLQS)
type LinkingQualityScore struct {
	stagingDB *database.StagingClient
	neo4jDB   *Client
}

// NewLinkingQualityScore creates a new CLQS calculator
func NewLinkingQualityScore(stagingDB *database.StagingClient, neo4jDB *Client) *LinkingQualityScore {
	return &LinkingQualityScore{
		stagingDB: stagingDB,
		neo4jDB:   neo4jDB,
	}
}

// CLQSReport represents the complete CLQS analysis
type CLQSReport struct {
	Repository    string                 `json:"repository"`
	OverallScore  float64                `json:"overall_score"`
	Grade         string                 `json:"grade"`
	Rank          string                 `json:"rank"`
	AnalyzedDate  string                 `json:"analyzed_date"`
	Components    CLQSComponents         `json:"components"`
	Confidence    ConfidenceDistribution `json:"confidence_distribution"`
	Comparisons   ComparisonData         `json:"comparisons"`
	Recommendations []string             `json:"recommendations"`
}

// CLQSComponents breaks down the score into sub-components
type CLQSComponents struct {
	ExplicitLinking      ComponentScore `json:"explicit_linking"`
	TemporalCorrelation  ComponentScore `json:"temporal_correlation"`
	CommentQuality       ComponentScore `json:"comment_quality"`
	SemanticConsistency  ComponentScore `json:"semantic_consistency"`
	BidirectionalRefs    ComponentScore `json:"bidirectional_references"`
}

// ComponentScore represents a single component's score
type ComponentScore struct {
	Score        float64                `json:"score"`
	Weight       float64                `json:"weight"`
	Contribution float64                `json:"contribution"`
	Details      map[string]interface{} `json:"details"`
}

// ConfidenceDistribution shows how confident the links are
type ConfidenceDistribution struct {
	HighConfidence   ConfidenceBucket `json:"high_confidence"`   // >= 0.85
	MediumConfidence ConfidenceBucket `json:"medium_confidence"` // 0.70-0.84
	LowConfidence    ConfidenceBucket `json:"low_confidence"`    // < 0.70
}

// ConfidenceBucket represents a range of confidence scores
type ConfidenceBucket struct {
	Count         int     `json:"count"`
	Percentage    float64 `json:"percentage"`
	AvgConfidence float64 `json:"avg_confidence"`
}

// ComparisonData compares this repo to others
type ComparisonData struct {
	IndustryAverage float64             `json:"industry_average"`
	Percentile      int                 `json:"percentile"`
	SimilarCodebases []ComparisonRepo   `json:"similar_codebases"`
}

// ComparisonRepo represents another repository for comparison
type ComparisonRepo struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// CalculateCLQS computes the Codebase Linking Quality Score
func (lqs *LinkingQualityScore) CalculateCLQS(ctx context.Context, repoID int64, repoFullName string) (*CLQSReport, error) {
	log.Printf("📊 Calculating Codebase Linking Quality Score for %s...", repoFullName)

	report := &CLQSReport{
		Repository: repoFullName,
		Components: CLQSComponents{},
	}

	// 1. Calculate Explicit Linking Score (40% weight)
	explicitScore, err := lqs.calculateExplicitScore(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate explicit score: %w", err)
	}
	report.Components.ExplicitLinking = explicitScore
	log.Printf("  ✓ Explicit Linking: %.1f%%", explicitScore.Score)

	// 2. Calculate Temporal Correlation Score (25% weight)
	temporalScore, err := lqs.calculateTemporalScore(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate temporal score: %w", err)
	}
	report.Components.TemporalCorrelation = temporalScore
	log.Printf("  ✓ Temporal Correlation: %.1f%%", temporalScore.Score)

	// 3. Calculate Comment Quality Score (20% weight)
	commentScore, err := lqs.calculateCommentScore(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate comment score: %w", err)
	}
	report.Components.CommentQuality = commentScore
	log.Printf("  ✓ Comment Quality: %.1f%%", commentScore.Score)

	// 4. Calculate Semantic Consistency Score (10% weight)
	semanticScore, err := lqs.calculateSemanticScore(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate semantic score: %w", err)
	}
	report.Components.SemanticConsistency = semanticScore
	log.Printf("  ✓ Semantic Consistency: %.1f%%", semanticScore.Score)

	// 5. Calculate Bidirectional Reference Score (5% weight)
	bidirectionalScore, err := lqs.calculateBidirectionalScore(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate bidirectional score: %w", err)
	}
	report.Components.BidirectionalRefs = bidirectionalScore
	log.Printf("  ✓ Bidirectional References: %.1f%%", bidirectionalScore.Score)

	// 6. Calculate overall score (weighted average)
	report.OverallScore = (explicitScore.Contribution +
		temporalScore.Contribution +
		commentScore.Contribution +
		semanticScore.Contribution +
		bidirectionalScore.Contribution)

	// 7. Assign grade and rank
	report.Grade = assignGrade(report.OverallScore)
	report.Rank = assignRank(report.OverallScore)

	// 8. Calculate confidence distribution
	confidenceDist, err := lqs.calculateConfidenceDistribution(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate confidence distribution: %w", err)
	}
	report.Confidence = confidenceDist

	// 9. Generate recommendations
	report.Recommendations = generateRecommendations(report)

	log.Printf("  🎯 Overall CLQS: %.1f (%s - %s)", report.OverallScore, report.Grade, report.Rank)

	return report, nil
}

// calculateExplicitScore computes the explicit linking score (40% weight)
func (lqs *LinkingQualityScore) calculateExplicitScore(ctx context.Context, repoID int64) (ComponentScore, error) {
	// Query Neo4j for issues with explicit references
	query := `
		MATCH (i:Issue {repo_id: $repo_id, state: 'closed'})
		OPTIONAL MATCH (i)-[r:FIXES_ISSUE|MENTIONS]->()
		WHERE 'explicit' IN r.evidence
		WITH COUNT(DISTINCT i) as total_issues,
		     COUNT(DISTINCT CASE WHEN r IS NOT NULL THEN i END) as linked_issues
		RETURN total_issues, linked_issues
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ComponentScore{}, err
	}

	totalIssues := int64(0)
	linkedIssues := int64(0)

	if len(result) > 0 {
		if ti, ok := result[0]["total_issues"].(int64); ok {
			totalIssues = ti
		}
		if li, ok := result[0]["linked_issues"].(int64); ok {
			linkedIssues = li
		}
	}

	score := 0.0
	if totalIssues > 0 {
		score = float64(linkedIssues) / float64(totalIssues) * 100
	}

	weight := 0.40
	contribution := (score / 100) * weight * 100

	return ComponentScore{
		Score:        score,
		Weight:       weight,
		Contribution: contribution,
		Details: map[string]interface{}{
			"total_closed_issues":        totalIssues,
			"issues_with_explicit_refs":  linkedIssues,
			"percentage":                 score,
		},
	}, nil
}

// calculateTemporalScore computes the temporal correlation score (25% weight)
func (lqs *LinkingQualityScore) calculateTemporalScore(ctx context.Context, repoID int64) (ComponentScore, error) {
	query := `
		MATCH (i:Issue {repo_id: $repo_id, state: 'closed'})
		OPTIONAL MATCH (i)-[r:FIXES_ISSUE|MENTIONS]->()
		WHERE 'temporal_match_5min' IN r.evidence
		   OR 'temporal_match_1hr' IN r.evidence
		   OR 'temporal_match_24hr' IN r.evidence
		WITH COUNT(DISTINCT i) as total_issues,
		     COUNT(DISTINCT CASE WHEN r IS NOT NULL THEN i END) as temporal_matches
		RETURN total_issues, temporal_matches
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ComponentScore{}, err
	}

	totalIssues := int64(0)
	temporalMatches := int64(0)

	if len(result) > 0 {
		if ti, ok := result[0]["total_issues"].(int64); ok {
			totalIssues = ti
		}
		if tm, ok := result[0]["temporal_matches"].(int64); ok {
			temporalMatches = tm
		}
	}

	score := 0.0
	if totalIssues > 0 {
		score = float64(temporalMatches) / float64(totalIssues) * 100
	}

	weight := 0.25
	contribution := (score / 100) * weight * 100

	return ComponentScore{
		Score:        score,
		Weight:       weight,
		Contribution: contribution,
		Details: map[string]interface{}{
			"total_closed_issues":         totalIssues,
			"issues_with_temporal_matches": temporalMatches,
			"percentage":                  score,
		},
	}, nil
}

// calculateCommentScore computes the comment quality score (20% weight)
func (lqs *LinkingQualityScore) calculateCommentScore(ctx context.Context, repoID int64) (ComponentScore, error) {
	query := `
		MATCH (i:Issue {repo_id: $repo_id, state: 'closed'})
		OPTIONAL MATCH (i)-[r:FIXES_ISSUE|MENTIONS]->()
		WHERE 'owner_comment' IN r.evidence
		   OR 'collaborator_comment' IN r.evidence
		   OR 'bot_comment' IN r.evidence
		WITH COUNT(DISTINCT i) as total_issues,
		     COUNT(DISTINCT CASE WHEN r IS NOT NULL THEN i END) as issues_with_comments
		RETURN total_issues, issues_with_comments
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ComponentScore{}, err
	}

	totalIssues := int64(0)
	issuesWithComments := int64(0)

	if len(result) > 0 {
		if ti, ok := result[0]["total_issues"].(int64); ok {
			totalIssues = ti
		}
		if iwc, ok := result[0]["issues_with_comments"].(int64); ok {
			issuesWithComments = iwc
		}
	}

	score := 0.0
	if totalIssues > 0 {
		score = float64(issuesWithComments) / float64(totalIssues) * 100
	}

	weight := 0.20
	contribution := (score / 100) * weight * 100

	return ComponentScore{
		Score:        score,
		Weight:       weight,
		Contribution: contribution,
		Details: map[string]interface{}{
			"total_closed_issues":         totalIssues,
			"issues_with_maintainer_comments": issuesWithComments,
			"percentage":                  score,
		},
	}, nil
}

// calculateSemanticScore computes the semantic consistency score (10% weight)
func (lqs *LinkingQualityScore) calculateSemanticScore(ctx context.Context, repoID int64) (ComponentScore, error) {
	// Query for issues with semantic matches
	query := `
		MATCH (i:Issue {repo_id: $repo_id, state: 'closed'})-[r:FIXES_ISSUE|MENTIONS]->(pr:PullRequest)
		WHERE r.semantic_score IS NOT NULL
		RETURN AVG(r.semantic_score) as avg_semantic_score,
		       COUNT(*) as total_links
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ComponentScore{}, err
	}

	avgSemanticScore := 0.0
	totalLinks := int64(0)

	if len(result) > 0 {
		if ass, ok := result[0]["avg_semantic_score"].(float64); ok {
			avgSemanticScore = ass
		}
		if tl, ok := result[0]["total_links"].(int64); ok {
			totalLinks = tl
		}
	}

	score := avgSemanticScore * 100

	weight := 0.10
	contribution := (score / 100) * weight * 100

	return ComponentScore{
		Score:        score,
		Weight:       weight,
		Contribution: contribution,
		Details: map[string]interface{}{
			"avg_keyword_overlap": avgSemanticScore,
			"total_links_analyzed": totalLinks,
			"percentage":          score,
		},
	}, nil
}

// calculateBidirectionalScore computes the bidirectional reference score (5% weight)
func (lqs *LinkingQualityScore) calculateBidirectionalScore(ctx context.Context, repoID int64) (ComponentScore, error) {
	query := `
		MATCH (i:Issue {repo_id: $repo_id})-[r:FIXES_ISSUE|MENTIONS]->()
		WITH COUNT(*) as total_links,
		     COUNT(CASE WHEN 'bidirectional_mention' IN r.evidence THEN 1 END) as bidirectional_links
		RETURN total_links, bidirectional_links
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ComponentScore{}, err
	}

	totalLinks := int64(0)
	bidirectionalLinks := int64(0)

	if len(result) > 0 {
		if tl, ok := result[0]["total_links"].(int64); ok {
			totalLinks = tl
		}
		if bl, ok := result[0]["bidirectional_links"].(int64); ok {
			bidirectionalLinks = bl
		}
	}

	score := 0.0
	if totalLinks > 0 {
		score = float64(bidirectionalLinks) / float64(totalLinks) * 100
	}

	weight := 0.05
	contribution := (score / 100) * weight * 100

	return ComponentScore{
		Score:        score,
		Weight:       weight,
		Contribution: contribution,
		Details: map[string]interface{}{
			"total_links":         totalLinks,
			"bidirectional_links": bidirectionalLinks,
			"percentage":          score,
		},
	}, nil
}

// calculateConfidenceDistribution breaks down links by confidence level
func (lqs *LinkingQualityScore) calculateConfidenceDistribution(ctx context.Context, repoID int64) (ConfidenceDistribution, error) {
	query := `
		MATCH (i:Issue {repo_id: $repo_id})-[r:FIXES_ISSUE|MENTIONS]->()
		RETURN
		  COUNT(CASE WHEN r.confidence >= 0.85 THEN 1 END) as high_count,
		  AVG(CASE WHEN r.confidence >= 0.85 THEN r.confidence END) as high_avg,
		  COUNT(CASE WHEN r.confidence >= 0.70 AND r.confidence < 0.85 THEN 1 END) as medium_count,
		  AVG(CASE WHEN r.confidence >= 0.70 AND r.confidence < 0.85 THEN r.confidence END) as medium_avg,
		  COUNT(CASE WHEN r.confidence < 0.70 THEN 1 END) as low_count,
		  AVG(CASE WHEN r.confidence < 0.70 THEN r.confidence END) as low_avg,
		  COUNT(*) as total_count
	`

	result, err := lqs.neo4jDB.ExecuteQuery(ctx, query, map[string]any{"repo_id": repoID})
	if err != nil {
		return ConfidenceDistribution{}, err
	}

	dist := ConfidenceDistribution{}

	if len(result) > 0 {
		r := result[0]

		totalCount := int64(1)
		if tc, ok := r["total_count"].(int64); ok {
			totalCount = tc
		}

		// High confidence
		if hc, ok := r["high_count"].(int64); ok {
			dist.HighConfidence.Count = int(hc)
			dist.HighConfidence.Percentage = float64(hc) / float64(totalCount) * 100
		}
		if ha, ok := r["high_avg"].(float64); ok {
			dist.HighConfidence.AvgConfidence = ha
		}

		// Medium confidence
		if mc, ok := r["medium_count"].(int64); ok {
			dist.MediumConfidence.Count = int(mc)
			dist.MediumConfidence.Percentage = float64(mc) / float64(totalCount) * 100
		}
		if ma, ok := r["medium_avg"].(float64); ok {
			dist.MediumConfidence.AvgConfidence = ma
		}

		// Low confidence
		if lc, ok := r["low_count"].(int64); ok {
			dist.LowConfidence.Count = int(lc)
			dist.LowConfidence.Percentage = float64(lc) / float64(totalCount) * 100
		}
		if la, ok := r["low_avg"].(float64); ok {
			dist.LowConfidence.AvgConfidence = la
		}
	}

	return dist, nil
}

// assignGrade converts score to letter grade
func assignGrade(score float64) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "A-"
	case score >= 80:
		return "B+"
	case score >= 75:
		return "B"
	case score >= 70:
		return "B-"
	case score >= 65:
		return "C+"
	case score >= 60:
		return "C"
	case score >= 55:
		return "C-"
	case score >= 50:
		return "D+"
	case score >= 45:
		return "D"
	default:
		return "F"
	}
}

// assignRank converts score to quality rank
func assignRank(score float64) string {
	switch {
	case score >= 90:
		return "World-Class"
	case score >= 75:
		return "High Quality"
	case score >= 60:
		return "Moderate Quality"
	case score >= 45:
		return "Below Average"
	default:
		return "Poor Quality"
	}
}

// generateRecommendations provides actionable advice based on score
func generateRecommendations(report *CLQSReport) []string {
	recommendations := []string{}

	score := report.OverallScore

	if score >= 90 {
		recommendations = append(recommendations,
			fmt.Sprintf("Excellent! You're in the top 5%% globally (%.1f/100)", score))
		recommendations = append(recommendations,
			"Consider documenting your practices as a case study")
	} else if score >= 75 {
		recommendations = append(recommendations,
			fmt.Sprintf("Strong performance (%.1f/100) - minor improvements possible", score))
	} else {
		recommendations = append(recommendations,
			fmt.Sprintf("Score: %.1f/100 - significant room for improvement", score))
	}

	// Component-specific recommendations
	if report.Components.ExplicitLinking.Score < 70 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  Explicit Linking is low (%.1f%%) - implement PR templates with required issue references",
				report.Components.ExplicitLinking.Score))
	}

	if report.Components.TemporalCorrelation.Score < 60 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  Temporal Correlation is low (%.1f%%) - automate issue closure on PR merge",
				report.Components.TemporalCorrelation.Score))
	}

	if report.Components.CommentQuality.Score < 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  Comment Quality is low (%.1f%%) - encourage maintainer engagement",
				report.Components.CommentQuality.Score))
	}

	if report.Components.SemanticConsistency.Score < 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  Semantic Consistency is low (%.1f%%) - use clearer PR titles",
				report.Components.SemanticConsistency.Score))
	}

	// Confidence distribution insights
	if report.Confidence.LowConfidence.Percentage > 20 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  %.1f%% of links have low confidence - improve explicit references",
				report.Confidence.LowConfidence.Percentage))
	}

	return recommendations
}

// ComputeSemanticSimilarity calculates Jaccard similarity between two texts
func ComputeSemanticSimilarity(text1, text2 string) float64 {
	keywords1 := extractKeywords(text1)
	keywords2 := extractKeywords(text2)

	intersection := 0
	for kw := range keywords1 {
		if keywords2[kw] {
			intersection++
		}
	}

	union := len(keywords1)
	for kw := range keywords2 {
		if !keywords1[kw] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// extractKeywords extracts meaningful keywords from text
func extractKeywords(text string) map[string]bool {
	// Convert to lowercase and split
	words := strings.Fields(strings.ToLower(text))

	keywords := make(map[string]bool)

	// Filter out common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
	}

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}#")

		// Keep words that are 3+ chars and not stop words
		if len(word) >= 3 && !stopWords[word] {
			keywords[word] = true
		}
	}

	return keywords
}
