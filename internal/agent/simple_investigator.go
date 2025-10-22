package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DEPRECATED: SimpleInvestigator will be replaced by the Sequential Analysis Chain (8 agents)
//
// Current status: IN USE by cmd/crisk/check.go (line 297)
//
// Replacement plan:
//   - Target: Week 3 of MVP development
//   - New implementation: internal/risk/chain_orchestrator.go + internal/risk/agents/*
//   - Migration path: See dev_docs/00-product/mvp_development_plan.md FR-3 to FR-5
//
// What's changing:
//   - FROM: Single LLM call with all evidence
//   - TO: Sequential 8-agent chain (Ownership → Churn → Co-change → ... → Synthesis)
//   - Benefits: Better accuracy, explainability, and control over reasoning process
//
// DO NOT ENHANCE this file - all new features go into the agent chain implementation
//
// SimpleInvestigator performs Phase 2 risk assessment with a single LLM call
// This is the MVP-aligned implementation that replaces the complex multi-hop navigator
type SimpleInvestigator struct {
	llm       LLMClientInterface
	temporal  TemporalClient
	incidents IncidentsClient
	graph     GraphClient
}

// DueDiligenceContext holds all evidence collected for the LLM prompt
type DueDiligenceContext struct {
	// Ownership
	OwnerName         string
	OwnerEmail        string
	LastModifiedDate  string
	LastModifier      string
	CommitCount       int

	// Blast Radius
	CouplingCount   int
	DependencyList  []string

	// Co-Change Patterns
	CoChangePartners  []CoChangePartner
	PotentialBreakage string

	// Incident History
	IncidentCount     int
	IncidentSummaries []string
	IncidentPattern   string

	// Test Coverage
	TestRatio  float64
	TestFiles  []string

	// Git diff
	GitDiff string

	// Phase 1 results
	Phase1RiskLevel string
	Phase1Metrics   string
}

// CoChangePartner represents a file that frequently changes with the target file
type CoChangePartner struct {
	FilePath  string
	Frequency float64
}

// SimpleDueDiligenceAssessment is the structured JSON response from the LLM
type SimpleDueDiligenceAssessment struct {
	RiskLevel            string              `json:"risk_level"`
	DueDiligenceSummary  string              `json:"due_diligence_summary"`
	CoordinationNeeded   CoordinationInfo    `json:"coordination_needed"`
	ForgottenUpdates     ForgottenUpdateInfo `json:"forgotten_updates"`
	IncidentRisk         IncidentRiskInfo    `json:"incident_risk"`
	Recommendations      []string            `json:"recommendations"`
}

// NewSimpleInvestigator creates a new simple investigator
func NewSimpleInvestigator(llm LLMClientInterface, temporal TemporalClient, incidents IncidentsClient, graph GraphClient) *SimpleInvestigator {
	return &SimpleInvestigator{
		llm:       llm,
		temporal:  temporal,
		incidents: incidents,
		graph:     graph,
	}
}

// Investigate performs Phase 2 investigation with a single LLM call
func (inv *SimpleInvestigator) Investigate(ctx context.Context, req InvestigationRequest) (RiskAssessment, error) {
	// Set request metadata
	req.RequestID = uuid.New()
	req.StartedAt = time.Now()

	// Step 1: Collect evidence in parallel from all sources
	ddContext, err := inv.collectDueDiligenceContext(ctx, req)
	if err != nil {
		// Non-fatal: proceed with whatever context we have
		// Error just means some data sources were unavailable
	}

	// Step 2: Build the prompt using MVP template
	prompt := inv.buildDueDiligencePrompt(req, ddContext)

	// Step 3: Query LLM with timeout
	llmCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	response, tokens, err := inv.llm.Query(llmCtx, prompt)
	if err != nil {
		// Fallback: return Phase 1 + basic info
		return inv.fallbackAssessment(req, ddContext, fmt.Sprintf("LLM query failed: %v", err)), nil
	}

	// Step 4: Parse JSON response
	var assessment SimpleDueDiligenceAssessment
	if err := json.Unmarshal([]byte(response), &assessment); err != nil {
		// Try to extract JSON from markdown code blocks
		cleaned := extractJSON(response)
		if err := json.Unmarshal([]byte(cleaned), &assessment); err != nil {
			// Fallback: return Phase 1 + basic info
			return inv.fallbackAssessment(req, ddContext, fmt.Sprintf("Failed to parse LLM response: %v", err)), nil
		}
	}

	// Step 5: Build final RiskAssessment
	return inv.buildRiskAssessment(req, ddContext, assessment, tokens), nil
}

// collectDueDiligenceContext gathers all evidence from available sources
func (inv *SimpleInvestigator) collectDueDiligenceContext(ctx context.Context, req InvestigationRequest) (DueDiligenceContext, error) {
	var ddCtx DueDiligenceContext

	// Ownership data
	if inv.temporal != nil {
		ownership, err := inv.temporal.GetOwnershipHistory(ctx, req.FilePath)
		if err == nil && ownership != nil && ownership.CurrentOwner != "" {
			ddCtx.OwnerEmail = ownership.CurrentOwner
			ddCtx.OwnerName = ownership.CurrentOwner // Extract name from email if needed
			ddCtx.LastModifiedDate = ownership.TransitionDate.Format("2006-01-02")
			ddCtx.LastModifier = ownership.CurrentOwner
			ddCtx.CommitCount = 0 // Not available in current OwnershipHistory type
		}
	}

	// Co-change patterns
	if inv.temporal != nil {
		coChanges, err := inv.temporal.GetCoChangedFiles(ctx, req.FilePath, 0.3)
		if err == nil && len(coChanges) > 0 {
			// Take top 5 co-change partners (MVP requirement)
			limit := 5
			if len(coChanges) < limit {
				limit = len(coChanges)
			}
			for i := 0; i < limit; i++ {
				ddCtx.CoChangePartners = append(ddCtx.CoChangePartners, CoChangePartner{
					FilePath:  coChanges[i].FileB,
					Frequency: coChanges[i].Frequency,
				})
			}
		}
	}

	// Incident history
	if inv.incidents != nil {
		incidentStats, err := inv.incidents.GetIncidentStats(ctx, req.FilePath)
		if err == nil && incidentStats != nil {
			ddCtx.IncidentCount = incidentStats.TotalIncidents

			// Search for recent incidents to get summaries
			searchResults, err := inv.incidents.SearchIncidents(ctx, req.FilePath, 3)
			if err == nil {
				for _, result := range searchResults {
					summary := fmt.Sprintf("%s: %s", result.Incident.ID, result.Incident.Title)
					ddCtx.IncidentSummaries = append(ddCtx.IncidentSummaries, summary)
				}
			}

			// Infer pattern if multiple incidents
			if incidentStats.TotalIncidents > 1 {
				ddCtx.IncidentPattern = fmt.Sprintf("Recurring issues (%d incidents in 90 days)", incidentStats.Last90Days)
			}
		}
	}

	// Blast radius (dependencies) - if graph available
	// For MVP, we use coupling count from Phase 1
	ddCtx.CouplingCount = int(req.Baseline.CouplingScore * 100) // Convert score to count approximation

	// Git diff
	ddCtx.GitDiff = req.DiffPreview

	// Phase 1 metrics
	ddCtx.Phase1RiskLevel = inferRiskLevelFromBaseline(req.Baseline)
	ddCtx.Phase1Metrics = formatPhase1Metrics(req.Baseline)

	// Test coverage
	ddCtx.TestRatio = req.Baseline.TestCoverage

	return ddCtx, nil
}

// buildDueDiligencePrompt constructs the LLM prompt using the MVP template
func (inv *SimpleInvestigator) buildDueDiligencePrompt(req InvestigationRequest, ctx DueDiligenceContext) string {
	// Format co-change partners
	coChangeText := "None detected"
	if len(ctx.CoChangePartners) > 0 {
		var parts []string
		for _, partner := range ctx.CoChangePartners {
			parts = append(parts, fmt.Sprintf("     - %s (%.0f%% co-change frequency)", partner.FilePath, partner.Frequency*100))
		}
		coChangeText = strings.Join(parts, "\n")
	}

	// Format dependency list
	depListText := "Not available"
	if len(ctx.DependencyList) > 0 {
		var parts []string
		for i, dep := range ctx.DependencyList {
			if i >= 10 { // Limit to top 10 (MVP requirement)
				break
			}
			parts = append(parts, fmt.Sprintf("     - %s", dep))
		}
		depListText = strings.Join(parts, "\n")
	}

	// Format incident summaries
	incidentText := "None"
	if len(ctx.IncidentSummaries) > 0 {
		incidentText = strings.Join(ctx.IncidentSummaries, "\n     ")
	}

	// Build prompt following MVP template exactly
	prompt := fmt.Sprintf(`You are a code risk assessment expert helping developers perform pre-commit due diligence.

A developer is about to commit changes to: %s

DUE DILIGENCE CONTEXT:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. OWNERSHIP
   - Primary owner: %s (%s)
   - Last modified: %s by %s
   - Commit frequency: %d commits in last 30 days

2. BLAST RADIUS (What depends on this?)
   - %d files depend on this file:
     %s

3. CO-CHANGE PATTERNS (What should change together?)
   - Files that frequently change with this file:
%s
   - Forgotten updates may cause: %s

4. INCIDENT HISTORY (Has this failed before?)
   - Past incidents: %d
     %s
   - Pattern: %s

5. TEST COVERAGE
   - Current coverage: %.1f%%
   - Test files: %s

GIT DIFF:
%s

PHASE 1 RISK ASSESSMENT: %s
EVIDENCE: %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
YOUR TASK: Provide a pre-commit risk assessment and action plan

Consider these due diligence questions:
1. Should the developer coordinate with the file owner before making this change?
2. What dependent files might break? Should they be updated together?
3. Is this similar to a past incident pattern? What should be done differently?
4. Is test coverage adequate for this change?
5. Who should review this change (beyond the owner)?

Respond in JSON format:
{
  "risk_level": "LOW|MEDIUM|HIGH|CRITICAL",
  "due_diligence_summary": "1-2 sentence summary of key risks",
  "coordination_needed": {
    "should_contact_owner": true/false,
    "should_contact_others": ["@alice (fraud expert)", "@bob (knows session logic)"],
    "reason": "why coordination is needed"
  },
  "forgotten_updates": {
    "likely_forgotten_files": ["file1.py", "file2.py"],
    "reason": "these files changed together in 15/20 commits"
  },
  "incident_risk": {
    "similar_incident": "INC-453",
    "pattern": "timeout cascade in auth flow",
    "prevention": "Add timeout handling, test with session_manager.py"
  },
  "recommendations": [
    "Action 1 (most important)",
    "Action 2",
    "Action 3"
  ]
}`,
		req.FilePath,
		ctx.OwnerName,
		ctx.OwnerEmail,
		ctx.LastModifiedDate,
		ctx.LastModifier,
		ctx.CommitCount,
		ctx.CouplingCount,
		depListText,
		coChangeText,
		ctx.PotentialBreakage,
		ctx.IncidentCount,
		incidentText,
		ctx.IncidentPattern,
		ctx.TestRatio*100,
		formatTestFiles(ctx.TestFiles),
		truncateGitDiff(ctx.GitDiff, 500),
		ctx.Phase1RiskLevel,
		ctx.Phase1Metrics,
	)

	return prompt
}

// fallbackAssessment returns a conservative assessment when LLM fails
func (inv *SimpleInvestigator) fallbackAssessment(req InvestigationRequest, ctx DueDiligenceContext, reason string) RiskAssessment {
	recommendations := []string{
		fmt.Sprintf("Contact file owner: %s (%s)", ctx.OwnerName, ctx.OwnerEmail),
		"Review dependencies and co-change patterns",
		"Verify test coverage is adequate",
	}

	if ctx.IncidentCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Review past incidents (%d total)", ctx.IncidentCount))
	}

	return RiskAssessment{
		FilePath:   req.FilePath,
		RiskLevel:  RiskMedium, // Conservative: assume MEDIUM risk
		RiskScore:  0.5,
		Confidence: 0.3, // Low confidence since we couldn't get LLM assessment
		Summary:    fmt.Sprintf("LLM assessment unavailable (%s). Using conservative risk level based on Phase 1 metrics.", reason),
		Evidence: []Evidence{
			{
				Type:        EvidenceOwnership,
				Description: fmt.Sprintf("Owner: %s, %d commits in 30 days", ctx.OwnerName, ctx.CommitCount),
				Severity:    0.3,
				Source:      "temporal",
				FilePath:    req.FilePath,
			},
		},
		Investigation: &Investigation{
			Request:        req,
			Summary:        "Fallback assessment due to LLM unavailability",
			StoppingReason: reason,
			CompletedAt:    time.Now(),
		},
		// MVP due diligence fields
		CoordinationNeeded: CoordinationInfo{
			ShouldContactOwner:  ctx.OwnerName != "",
			ShouldContactOthers: []string{},
			Reason:              "Unable to perform detailed assessment, recommend contacting owner for review",
		},
		ForgottenUpdates: ForgottenUpdateInfo{
			LikelyForgottenFiles: extractFileNames(ctx.CoChangePartners, 3),
			Reason:               fmt.Sprintf("These files frequently change together (based on %d co-change partners)", len(ctx.CoChangePartners)),
		},
		IncidentRisk: IncidentRiskInfo{
			SimilarIncident: "",
			Pattern:         ctx.IncidentPattern,
			Prevention:      "Review incident history before committing",
		},
		Recommendations: recommendations,
	}
}

// buildRiskAssessment converts SimpleDueDiligenceAssessment to RiskAssessment
func (inv *SimpleInvestigator) buildRiskAssessment(req InvestigationRequest, ctx DueDiligenceContext, assessment SimpleDueDiligenceAssessment, tokens int) RiskAssessment {
	riskLevel := parseRiskLevel(assessment.RiskLevel)
	riskScore := riskLevelToScore(riskLevel)

	return RiskAssessment{
		FilePath:   req.FilePath,
		RiskLevel:  riskLevel,
		RiskScore:  riskScore,
		Confidence: 0.85, // High confidence from LLM assessment
		Summary:    assessment.DueDiligenceSummary,
		Evidence:   buildEvidenceFromContext(ctx),
		Investigation: &Investigation{
			Request:        req,
			Summary:        assessment.DueDiligenceSummary,
			StoppingReason: "Single LLM call completed",
			CompletedAt:    time.Now(),
			TotalTokens:    tokens,
		},
		CoordinationNeeded: assessment.CoordinationNeeded,
		ForgottenUpdates:   assessment.ForgottenUpdates,
		IncidentRisk:       assessment.IncidentRisk,
		Recommendations:    assessment.Recommendations,
	}
}

// Helper functions

func extractJSON(response string) string {
	// Try to extract JSON from markdown code blocks
	if idx := strings.Index(response, "```json"); idx != -1 {
		response = response[idx+7:]
		if idx := strings.Index(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if idx := strings.Index(response, "```"); idx != -1 {
		response = response[idx+3:]
		if idx := strings.Index(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}
	return strings.TrimSpace(response)
}

func inferRiskLevelFromBaseline(baseline BaselineMetrics) string {
	// Simple heuristic based on Phase 1 metrics
	if baseline.IncidentCount >= 3 || baseline.CouplingScore > 0.8 {
		return "HIGH"
	} else if baseline.IncidentCount >= 1 || baseline.CouplingScore > 0.5 {
		return "MEDIUM"
	}
	return "LOW"
}

func formatPhase1Metrics(baseline BaselineMetrics) string {
	return fmt.Sprintf("Coupling: %.2f, Co-change: %.2f, Incidents: %d, Test Coverage: %.1f%%",
		baseline.CouplingScore,
		baseline.CoChangeFrequency,
		baseline.IncidentCount,
		baseline.TestCoverage*100,
	)
}

func formatTestFiles(files []string) string {
	if len(files) == 0 {
		return "None detected"
	}
	return strings.Join(files, ", ")
}

func truncateGitDiff(diff string, maxLines int) string {
	if diff == "" {
		return "Not available"
	}
	lines := strings.Split(diff, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... (truncated)")
	}
	return strings.Join(lines, "\n")
}

func parseRiskLevel(level string) RiskLevel {
	switch strings.ToUpper(level) {
	case "CRITICAL":
		return RiskCritical
	case "HIGH":
		return RiskHigh
	case "MEDIUM":
		return RiskMedium
	case "LOW":
		return RiskLow
	default:
		return RiskMedium // Conservative default
	}
}

func riskLevelToScore(level RiskLevel) float64 {
	switch level {
	case RiskCritical:
		return 0.9
	case RiskHigh:
		return 0.7
	case RiskMedium:
		return 0.5
	case RiskLow:
		return 0.3
	default:
		return 0.5
	}
}

func extractFileNames(partners []CoChangePartner, limit int) []string {
	var files []string
	for i, partner := range partners {
		if i >= limit {
			break
		}
		files = append(files, partner.FilePath)
	}
	return files
}

func buildEvidenceFromContext(ctx DueDiligenceContext) []Evidence {
	var evidence []Evidence

	// Ownership evidence
	if ctx.OwnerName != "" {
		evidence = append(evidence, Evidence{
			Type:        EvidenceOwnership,
			Description: fmt.Sprintf("Owner: %s (%s), %d commits in 30 days", ctx.OwnerName, ctx.OwnerEmail, ctx.CommitCount),
			Severity:    0.3,
			Source:      "temporal",
		})
	}

	// Co-change evidence
	for _, partner := range ctx.CoChangePartners {
		evidence = append(evidence, Evidence{
			Type:        EvidenceCoChange,
			Description: fmt.Sprintf("Co-changes with %s (%.0f%%)", partner.FilePath, partner.Frequency*100),
			Severity:    partner.Frequency,
			Source:      "temporal",
			FilePath:    partner.FilePath,
		})
	}

	// Incident evidence
	if ctx.IncidentCount > 0 {
		severity := float64(ctx.IncidentCount) / 10.0 // Normalize to 0-1
		if severity > 1.0 {
			severity = 1.0
		}
		evidence = append(evidence, Evidence{
			Type:        EvidenceIncident,
			Description: fmt.Sprintf("%d incidents, pattern: %s", ctx.IncidentCount, ctx.IncidentPattern),
			Severity:    severity,
			Source:      "incidents",
		})
	}

	return evidence
}
