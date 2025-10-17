package risk

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rohankatakam/coderisk/internal/models"
	"github.com/sirupsen/logrus"
)

// Calculator performs risk calculations
type Calculator struct {
	logger *logrus.Logger
	config *Config
}

// Config holds risk calculation configuration
type Config struct {
	// Risk thresholds
	LowThreshold      float64
	MediumThreshold   float64
	HighThreshold     float64
	CriticalThreshold float64

	// Weights for different factors
	BlastRadiusWeight  float64
	TestCoverageWeight float64
	OwnershipWeight    float64
	TemporalWeight     float64
	CentralityWeight   float64
	IncidentWeight     float64
}

// DefaultConfig returns default risk configuration
func DefaultConfig() *Config {
	return &Config{
		LowThreshold:      0.25,
		MediumThreshold:   0.50,
		HighThreshold:     0.75,
		CriticalThreshold: 0.90,

		BlastRadiusWeight:  0.30,
		TestCoverageWeight: 0.20,
		OwnershipWeight:    0.15,
		TemporalWeight:     0.15,
		CentralityWeight:   0.10,
		IncidentWeight:     0.10,
	}
}

// NewCalculator creates a new risk calculator
func NewCalculator(logger *logrus.Logger, config *Config) *Calculator {
	if config == nil {
		config = DefaultConfig()
	}
	return &Calculator{
		logger: logger,
		config: config,
	}
}

// CalculateRisk performs complete risk assessment
func (c *Calculator) CalculateRisk(ctx context.Context, sketches []*models.RiskSketch, changes []string) (*models.RiskAssessment, error) {
	startTime := time.Now()
	c.logger.Info("Starting risk calculation")

	assessment := &models.RiskAssessment{
		ID:        generateID(),
		Timestamp: startTime,
		Factors:   []models.RiskFactor{},
	}

	// Calculate individual risk components
	blastRadius := c.calculateBlastRadius(sketches, changes)
	testCoverage := c.calculateTestCoverage(sketches, changes)
	ownershipRisk := c.calculateOwnershipRisk(sketches, changes)
	temporalRisk := c.calculateTemporalRisk(sketches, changes)
	centralityRisk := c.calculateCentralityRisk(sketches, changes)
	incidentRisk := c.calculateIncidentRisk(sketches, changes)

	// Weight and combine scores
	totalScore := 0.0
	totalScore += blastRadius * c.config.BlastRadiusWeight
	totalScore += (1 - testCoverage) * c.config.TestCoverageWeight // Inverse: lower coverage = higher risk
	totalScore += ownershipRisk * c.config.OwnershipWeight
	totalScore += temporalRisk * c.config.TemporalWeight
	totalScore += centralityRisk * c.config.CentralityWeight
	totalScore += incidentRisk * c.config.IncidentWeight

	assessment.Score = totalScore
	assessment.BlastRadius = blastRadius
	assessment.TestCoverage = testCoverage

	// Determine risk level
	assessment.Level = c.determineRiskLevel(totalScore)

	// Add top risk factors
	c.addRiskFactors(assessment, blastRadius, testCoverage, ownershipRisk,
		temporalRisk, centralityRisk, incidentRisk)

	// Generate suggestions
	assessment.Suggestions = c.generateSuggestions(assessment)

	duration := time.Since(startTime)
	c.logger.WithFields(logrus.Fields{
		"score":    totalScore,
		"level":    assessment.Level,
		"duration": duration.String(),
	}).Info("Risk calculation completed")

	return assessment, nil
}

// calculateBlastRadius determines impact scope
func (c *Calculator) calculateBlastRadius(sketches []*models.RiskSketch, changes []string) float64 {
	changeSet := make(map[string]bool)
	for _, change := range changes {
		changeSet[change] = true
	}

	affectedFiles := make(map[string]bool)
	for _, sketch := range sketches {
		if changeSet[sketch.FileID] {
			// This file was directly changed
			affectedFiles[sketch.FileID] = true

			// Check co-change frequency to find dependent files
			for depFile, freq := range sketch.CoChangeFrequency {
				if freq > 0.3 { // Files that change together >30% of the time
					affectedFiles[depFile] = true
				}
			}
		}
	}

	// Normalize blast radius (0-1)
	blastRadius := float64(len(affectedFiles)) / float64(max(len(sketches), 1))
	return min(blastRadius, 1.0)
}

// calculateTestCoverage computes test coverage for changed files
func (c *Calculator) calculateTestCoverage(sketches []*models.RiskSketch, changes []string) float64 {
	changeSet := make(map[string]bool)
	for _, change := range changes {
		changeSet[change] = true
	}

	totalCoverage := 0.0
	count := 0

	for _, sketch := range sketches {
		if changeSet[sketch.FileID] {
			totalCoverage += sketch.TestCoverage
			count++
		}
	}

	if count == 0 {
		return 0.5 // Default to medium coverage if no data
	}

	return totalCoverage / float64(count)
}

// calculateOwnershipRisk assesses ownership-related risk
func (c *Calculator) calculateOwnershipRisk(sketches []*models.RiskSketch, changes []string) float64 {
	changeSet := make(map[string]bool)
	for _, change := range changes {
		changeSet[change] = true
	}

	totalOwnershipScore := 0.0
	count := 0

	for _, sketch := range sketches {
		if changeSet[sketch.FileID] {
			// Lower ownership score = higher risk (unfamiliar code)
			totalOwnershipScore += (1 - sketch.OwnershipScore)
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	return totalOwnershipScore / float64(count)
}

// calculateTemporalRisk analyzes time-based patterns
func (c *Calculator) calculateTemporalRisk(sketches []*models.RiskSketch, changes []string) float64 {
	// Simple implementation: risk increases with recency of changes
	now := time.Now()
	totalRisk := 0.0
	count := 0

	for _, sketch := range sketches {
		daysSinceUpdate := now.Sub(sketch.LastUpdated).Hours() / 24
		if daysSinceUpdate < 7 {
			// Recently changed files are higher risk
			totalRisk += 1.0 - (daysSinceUpdate / 7)
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalRisk / float64(count)
}

// calculateCentralityRisk measures graph centrality impact
func (c *Calculator) calculateCentralityRisk(sketches []*models.RiskSketch, changes []string) float64 {
	changeSet := make(map[string]bool)
	for _, change := range changes {
		changeSet[change] = true
	}

	maxCentrality := 0.0
	for _, sketch := range sketches {
		if changeSet[sketch.FileID] {
			if sketch.CentralityScore > maxCentrality {
				maxCentrality = sketch.CentralityScore
			}
		}
	}

	return maxCentrality
}

// calculateIncidentRisk checks historical incident correlation
func (c *Calculator) calculateIncidentRisk(sketches []*models.RiskSketch, changes []string) float64 {
	changeSet := make(map[string]bool)
	for _, change := range changes {
		changeSet[change] = true
	}

	totalIncidents := 0
	for _, sketch := range sketches {
		if changeSet[sketch.FileID] {
			totalIncidents += len(sketch.IncidentHistory)
		}
	}

	// Normalize: cap at 5 incidents for max risk
	return min(float64(totalIncidents)/5.0, 1.0)
}

// determineRiskLevel converts score to risk level
func (c *Calculator) determineRiskLevel(score float64) models.RiskLevel {
	switch {
	case score >= c.config.CriticalThreshold:
		return models.RiskLevelCritical
	case score >= c.config.HighThreshold:
		return models.RiskLevelHigh
	case score >= c.config.MediumThreshold:
		return models.RiskLevelMedium
	default:
		return models.RiskLevelLow
	}
}

// addRiskFactors adds detailed risk factors to assessment
func (c *Calculator) addRiskFactors(assessment *models.RiskAssessment,
	blastRadius, testCoverage, ownershipRisk, temporalRisk, centralityRisk, incidentRisk float64) {

	factors := []models.RiskFactor{}

	if blastRadius > 0.5 {
		factors = append(factors, models.RiskFactor{
			Signal: "High Blast Radius",
			Impact: c.getImpactLevel(blastRadius),
			Score:  blastRadius,
			Detail: "Changes affect many dependent files",
		})
	}

	if testCoverage < 0.3 {
		factors = append(factors, models.RiskFactor{
			Signal: "Low Test Coverage",
			Impact: c.getImpactLevel(1 - testCoverage),
			Score:  1 - testCoverage,
			Detail: "Insufficient test coverage for changed files",
		})
	}

	if ownershipRisk > 0.6 {
		factors = append(factors, models.RiskFactor{
			Signal: "Ownership Risk",
			Impact: c.getImpactLevel(ownershipRisk),
			Score:  ownershipRisk,
			Detail: "Modifying code with low ownership familiarity",
		})
	}

	if centralityRisk > 0.7 {
		factors = append(factors, models.RiskFactor{
			Signal: "High Centrality",
			Impact: c.getImpactLevel(centralityRisk),
			Score:  centralityRisk,
			Detail: "Modifying highly connected components",
		})
	}

	if incidentRisk > 0.5 {
		factors = append(factors, models.RiskFactor{
			Signal: "Incident History",
			Impact: c.getImpactLevel(incidentRisk),
			Score:  incidentRisk,
			Detail: "Files have history of production incidents",
		})
	}

	// Sort by score and take top 3
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Score > factors[j].Score
	})

	if len(factors) > 3 {
		factors = factors[:3]
	}

	assessment.Factors = factors
}

// getImpactLevel converts score to impact level string
func (c *Calculator) getImpactLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "CRITICAL"
	case score >= 0.6:
		return "HIGH"
	case score >= 0.4:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// generateSuggestions creates actionable suggestions
func (c *Calculator) generateSuggestions(assessment *models.RiskAssessment) []string {
	suggestions := []string{}

	for _, factor := range assessment.Factors {
		switch factor.Signal {
		case "High Blast Radius":
			suggestions = append(suggestions, "Consider breaking changes into smaller, isolated commits")
			suggestions = append(suggestions, "Add integration tests for affected components")
		case "Low Test Coverage":
			suggestions = append(suggestions, "Add unit tests before committing")
			suggestions = append(suggestions, "Ensure critical paths have test coverage")
		case "Ownership Risk":
			suggestions = append(suggestions, "Request review from code owners")
			suggestions = append(suggestions, "Pair with team member familiar with this code")
		case "High Centrality":
			suggestions = append(suggestions, "Extra careful review needed for core components")
			suggestions = append(suggestions, "Consider gradual rollout strategy")
		case "Incident History":
			suggestions = append(suggestions, "Review previous incident reports")
			suggestions = append(suggestions, "Add monitoring before deployment")
		}
	}

	return suggestions
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("risk-%d", time.Now().UnixNano())
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
