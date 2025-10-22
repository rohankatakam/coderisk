package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/rohankatakam/coderisk/internal/types"
	"github.com/rohankatakam/coderisk/internal/risk/agents"
)

// ChainOrchestrator coordinates the 5-phase sequential analysis chain
type ChainOrchestrator struct {
	heuristicFilter *HeuristicFilter
	collector       *Collector
	agentList       []agents.Agent
}

// NewChainOrchestrator creates a new chain orchestrator
func NewChainOrchestrator() *ChainOrchestrator {
	return &ChainOrchestrator{
		heuristicFilter: NewHeuristicFilter(),
		collector:       NewCollector(),
		agentList: []agents.Agent{
			agents.NewIncidentAgent(),
			agents.NewBlastRadiusAgent(),
			agents.NewCoChangeAgent(),
			agents.NewOwnershipAgent(),
			agents.NewQualityAgent(),
			agents.NewPatternsAgent(),
			agents.NewSynthesizerAgent(),
			agents.NewValidatorAgent(),
		},
	}
}

// Analyze performs the complete 5-phase risk analysis
func (o *ChainOrchestrator) Analyze(ctx context.Context, req *AnalysisRequest) (*ChainOrchestratorResult, error) {
	startTime := time.Now()
	
	result := &ChainOrchestratorResult{
		PhaseResults:  []PhaseResult{},
		Timestamp:     startTime,
		FilePath:      req.FilePaths[0], // MVP: single file
		CommitSHA:     req.CommitSHA,
		Branch:        req.Branch,
		CacheHit:      false,
		FastPathTaken: false,
	}

	// Phase 0: Heuristic Filter (Tier 0)
	heuristicResult, err := o.heuristicFilter.Filter(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("phase 0 failed: %w", err)
	}
	result.HeuristicResult = heuristicResult

	// Fast path: If trivial, skip full analysis
	if heuristicResult.IsTrivial {
		result.RiskLevel = types.RiskLevelLow
		result.RiskScore = 0.1
		result.Confidence = heuristicResult.Confidence
		result.Summary = heuristicResult.Reason
		result.FastPathTaken = true
		result.TotalDuration = time.Since(startTime)
		return result, nil
	}

	// Phase 1: Data Collection (7 queries)
	phase1Data, err := o.collector.CollectPhase1Data(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("phase 1 failed: %w", err)
	}
	result.Phase1Data = phase1Data

	// Phase 2-5: Agent Analysis Chain
	agentCtx := &AgentContext{
		FilePath:         req.FilePaths[0],
		RepoID:           req.RepoID,
		CommitSHA:        req.CommitSHA,
		GitDiff:          req.GitDiff,
		Branch:           req.Branch,
		HeuristicResult:  heuristicResult,
		Phase1Data:       phase1Data,
		AgentResults:     make(map[string]interface{}),
		RiskSignals:      []RiskSignal{},
		OverallRiskScore: 0.0,
		StartTime:        startTime,
		PhaseDurations:   make(map[string]time.Duration),
	}

	// Execute agents sequentially
	for _, agent := range o.agentList {
		phaseStart := time.Now()
		if err := agent.Analyze(ctx, agentCtx); err != nil {
			return nil, fmt.Errorf("agent %s failed: %w", agent.Name(), err)
		}
		agentCtx.PhaseDurations[agent.Name()] = time.Since(phaseStart)
	}

	// Compile final results
	result.RiskSignals = agentCtx.RiskSignals
	result.RiskScore = agentCtx.OverallRiskScore
	result.RiskLevel = calculateRiskLevel(agentCtx.OverallRiskScore)
	result.Confidence = 0.75 // Default confidence
	result.Summary = "Analysis completed"
	result.Recommendations = []string{"Review changes carefully"}
	result.CoordinationNeeded = []CoordinationInfo{}
	result.ForgottenUpdates = []ForgottenUpdate{}
	result.TotalDuration = time.Since(startTime)

	return result, nil
}

func calculateRiskLevel(score float64) types.RiskLevel {
	if score < 0.3 {
		return types.RiskLevelLow
	} else if score < 0.6 {
		return types.RiskLevelMedium
	} else if score < 0.8 {
		return types.RiskLevelHigh
	}
	return types.RiskLevelCritical
}
