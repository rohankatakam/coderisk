# Session C: LLM Investigation Implementation

**Duration:** Weeks 6-8 (2-3 weeks)
**Package:** `internal/agent/` (you own this entirely)
**Goal:** Agentic graph investigation with OpenAI/Anthropic, evidence synthesis, risk scoring

---

## Your Mission

Implement **Phase 2 (LLM Investigation)** of the two-phase analysis:
1. Integrate OpenAI/Anthropic SDK for GPT-4o and Claude 3.5 Sonnet
2. Build hop-by-hop graph navigator (1-hop → 2-hop → 3-hop)
3. Implement evidence accumulation and synthesis
4. Calculate risk scores with confidence levels
5. Generate actionable risk summaries

**Why this matters:** This is the "AI magic" that makes CodeRisk 10M times faster than exhaustive analysis. The agent selectively explores 1% of the graph using LLM reasoning to find high-risk patterns.

---

## What You Own (No Conflicts!)

### Files You Create
- `internal/agent/investigator.go` - Main investigation orchestrator
- `internal/agent/hop_navigator.go` - Hop-by-hop graph traversal
- `internal/agent/evidence.go` - Evidence accumulation and scoring
- `internal/agent/synthesis.go` - Risk synthesis and summary generation
- `internal/agent/llm_client.go` - OpenAI/Anthropic SDK wrapper
- `internal/agent/types.go` - Data structures
- `internal/agent/investigator_test.go` - Unit tests
- `test/integration/test_agent_investigation.sh` - E2E test

### Files You Modify (Integration Points)
- `cmd/crisk/check.go` - Add Phase 2 escalation logic (~80 lines)
- `internal/ingestion/processor.go` - Export baseline metrics for Phase 1 (~20 lines)

### Files You Read (Session A & B Dependencies)
- `internal/temporal/co_change.go` - Use `GetCoChangedFiles()` for co-change data
- `internal/temporal/developer.go` - Use `GetOwnershipHistory()` for ownership
- `internal/incidents/database.go` - Use `GetIncidentStats()` for incident counts

---

## Technical Specification

### Data Structures to Create

**File:** `internal/agent/types.go`

```go
package agent

import (
    "time"
    "github.com/google/uuid"
)

// InvestigationRequest represents a Phase 2 escalation
type InvestigationRequest struct {
    RequestID   uuid.UUID
    FilePath    string
    ChangeType  string       // "modify" | "create" | "delete"
    DiffPreview string       // First 50 lines of git diff
    Baseline    BaselineMetrics
    StartedAt   time.Time
}

// BaselineMetrics from Phase 1 (import from ingestion package)
type BaselineMetrics struct {
    CouplingScore     float64  // 0.0-1.0
    CoChangeFrequency float64  // 0.0-1.0
    IncidentCount     int
    TestCoverage      float64  // 0.0-1.0 (future)
    OwnershipDays     int      // Days since ownership transition
}

// Investigation represents the full investigation state
type Investigation struct {
    Request      InvestigationRequest
    Hops         []HopResult
    Evidence     []Evidence
    RiskScore    float64   // 0.0-1.0
    Confidence   float64   // 0.0-1.0
    Summary      string
    CompletedAt  time.Time
    TotalTokens  int
}

// HopResult represents the result of a single hop
type HopResult struct {
    HopNumber    int
    Query        string      // LLM query sent
    Response     string      // LLM response
    NodesVisited []string    // Node IDs visited
    EdgesTraversed []string  // Edge types traversed
    TokensUsed   int
    Duration     time.Duration
}

// Evidence represents a single piece of risk evidence
type Evidence struct {
    Type        EvidenceType
    Description string
    Severity    float64   // 0.0-1.0
    Source      string    // "temporal" | "incidents" | "structure"
    FilePath    string
}

// EvidenceType categorizes evidence
type EvidenceType string

const (
    EvidenceCoChange       EvidenceType = "co_change"        // Files change together
    EvidenceIncident       EvidenceType = "incident"         // Past production incidents
    EvidenceOwnership      EvidenceType = "ownership"        // Ownership transition
    EvidenceCoupling       EvidenceType = "coupling"         // High structural coupling
    EvidenceComplexity     EvidenceType = "complexity"       // High cyclomatic complexity
    EvidenceMissingTests   EvidenceType = "missing_tests"    // No test coverage
)

// RiskAssessment is the final output
type RiskAssessment struct {
    FilePath    string
    RiskLevel   RiskLevel
    RiskScore   float64
    Confidence  float64
    Summary     string
    Evidence    []Evidence
    Investigation *Investigation  // Full details
}

// RiskLevel categorizes risk
type RiskLevel string

const (
    RiskCritical RiskLevel = "CRITICAL"  // >0.8
    RiskHigh     RiskLevel = "HIGH"      // 0.6-0.8
    RiskMedium   RiskLevel = "MEDIUM"    // 0.4-0.6
    RiskLow      RiskLevel = "LOW"       // 0.2-0.4
    RiskMinimal  RiskLevel = "MINIMAL"   // <0.2
)
```

### Key Functions to Implement

**File:** `internal/agent/llm_client.go`

```go
package agent

import (
    "context"
    "fmt"
    openai "github.com/openai/openai-go"
    anthropic "github.com/anthropics/anthropic-sdk-go"
)

// LLMClient wraps OpenAI and Anthropic SDKs
type LLMClient struct {
    provider     string  // "openai" | "anthropic"
    openaiClient *openai.Client
    anthropicClient *anthropic.Client
    model        string
}

// NewLLMClient creates a new LLM client
func NewLLMClient(provider string, apiKey string) (*LLMClient, error) {
    switch provider {
    case "openai":
        client := openai.NewClient(openai.WithAPIKey(apiKey))
        return &LLMClient{
            provider:     "openai",
            openaiClient: client,
            model:        "gpt-4o",  // Default to GPT-4o
        }, nil

    case "anthropic":
        client := anthropic.NewClient(anthropic.WithAPIKey(apiKey))
        return &LLMClient{
            provider:     "anthropic",
            anthropicClient: client,
            model:        "claude-3-5-sonnet-20241022",  // Latest Sonnet
        }, nil

    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}

// Query sends a prompt to the LLM and returns the response
func (c *LLMClient) Query(ctx context.Context, prompt string) (string, int, error) {
    // Route to appropriate SDK
    // Return (response, tokens_used, error)
}
```

**File:** `internal/agent/hop_navigator.go`

```go
package agent

import (
    "context"
    "fmt"
)

// HopNavigator performs hop-by-hop graph traversal
type HopNavigator struct {
    llm         *LLMClient
    graph       GraphClient
    maxHops     int
    maxTokens   int
}

// NewHopNavigator creates a new hop navigator
func NewHopNavigator(llm *LLMClient, graph GraphClient, maxHops int) *HopNavigator {
    return &HopNavigator{
        llm:       llm,
        graph:     graph,
        maxHops:   maxHops,
        maxTokens: 10000,  // Budget for entire investigation
    }
}

// Navigate performs multi-hop investigation
func (n *HopNavigator) Navigate(ctx context.Context, req InvestigationRequest) ([]HopResult, error) {
    var hops []HopResult
    totalTokens := 0

    // Hop 1: Immediate neighbors (CONTAINS, IMPORTS, CO_CHANGED)
    hop1, err := n.executeHop(ctx, 1, req, hops)
    if err != nil {
        return nil, err
    }
    hops = append(hops, hop1)
    totalTokens += hop1.TokensUsed

    // Early exit if risk is obvious
    if n.shouldStopEarly(hops) {
        return hops, nil
    }

    // Hop 2: Explore suspicious neighbors (CAUSED_BY, temporal coupling)
    hop2, err := n.executeHop(ctx, 2, req, hops)
    if err != nil {
        return hops, nil  // Non-fatal, return what we have
    }
    hops = append(hops, hop2)
    totalTokens += hop2.TokensUsed

    // Check token budget
    if totalTokens > n.maxTokens {
        return hops, nil
    }

    // Hop 3: Deep context (if still unclear)
    if n.needsDeepDive(hops) && totalTokens < n.maxTokens-2000 {
        hop3, err := n.executeHop(ctx, 3, req, hops)
        if err != nil {
            return hops, nil  // Non-fatal
        }
        hops = append(hops, hop3)
    }

    return hops, nil
}

// executeHop performs a single hop
func (n *HopNavigator) executeHop(ctx context.Context, hopNum int, req InvestigationRequest, previousHops []HopResult) (HopResult, error) {
    // 1. Build prompt based on hop number and previous results
    prompt := n.buildHopPrompt(hopNum, req, previousHops)

    // 2. Query LLM
    response, tokens, err := n.llm.Query(ctx, prompt)
    if err != nil {
        return HopResult{}, err
    }

    // 3. Parse LLM response to determine which nodes/edges to explore
    nodesToVisit := n.parseNodesFromResponse(response)

    // 4. Query graph for those nodes
    graphData := n.graph.GetNodes(ctx, nodesToVisit)

    // 5. Return hop result
    return HopResult{
        HopNumber:      hopNum,
        Query:          prompt,
        Response:       response,
        NodesVisited:   nodesToVisit,
        TokensUsed:     tokens,
        Duration:       /* measure */,
    }, nil
}

// buildHopPrompt constructs the LLM prompt for a specific hop
func (n *HopNavigator) buildHopPrompt(hopNum int, req InvestigationRequest, previousHops []HopResult) string {
    switch hopNum {
    case 1:
        // Focus on immediate context
        return fmt.Sprintf(`You are analyzing a code change for risk.

File: %s
Change type: %s
Baseline metrics:
- Coupling score: %.2f
- Co-change frequency: %.2f
- Past incidents: %d
- Ownership transition: %d days ago

Diff preview:
%s

Question: What are the immediate risk factors? Consider:
1. Structural dependencies (IMPORTS, CONTAINS)
2. Files that frequently change together (CO_CHANGED)
3. Past incidents linked to this file (CAUSED_BY)

Provide a structured analysis focusing on high-risk relationships.`,
            req.FilePath,
            req.ChangeType,
            req.Baseline.CouplingScore,
            req.Baseline.CoChangeFrequency,
            req.Baseline.IncidentCount,
            req.Baseline.OwnershipDays,
            req.DiffPreview,
        )

    case 2:
        // Explore suspicious connections
        return fmt.Sprintf(`Based on Hop 1 findings, investigate deeper:

Previous findings:
%s

Question: Which of these connections pose the highest risk?
1. For each CAUSED_BY edge: What was the incident severity and recency?
2. For each CO_CHANGED edge: How strong is the coupling (frequency)?
3. Are there ownership transitions that increase risk?

Rank the top 3 risk factors.`,
            previousHops[0].Response,
        )

    case 3:
        // Deep context for complex cases
        return fmt.Sprintf(`Final deep-dive investigation:

Hop 1 findings:
%s

Hop 2 findings:
%s

Question: Is there any hidden context that changes the risk assessment?
1. Cascading dependencies (2-3 hops away)
2. Behavioral patterns (temporal coupling across multiple files)
3. Systemic risks (architecture smells)

Provide final risk verdict with confidence level.`,
            previousHops[0].Response,
            previousHops[1].Response,
        )

    default:
        return ""
    }
}

// shouldStopEarly determines if we can stop after Hop 1
func (n *HopNavigator) shouldStopEarly(hops []HopResult) bool {
    // If Hop 1 finds critical incident or >0.9 coupling, stop early
    // Parse hops[0].Response for keywords like "critical", "high risk"
    return false  // Implement heuristic
}

// needsDeepDive determines if Hop 3 is needed
func (n *HopNavigator) needsDeepDive(hops []HopResult) bool {
    // If Hop 1 and Hop 2 disagree or are uncertain, do Hop 3
    return false  // Implement heuristic
}

// GraphClient interface (implement in internal/graph/builder.go)
type GraphClient interface {
    GetNodes(ctx context.Context, nodeIDs []string) (map[string]interface{}, error)
    GetNeighbors(ctx context.Context, nodeID string, edgeTypes []string, maxDepth int) ([]interface{}, error)
}
```

**File:** `internal/agent/evidence.go`

```go
package agent

import (
    "context"
    "your-module/internal/temporal"
    "your-module/internal/incidents"
)

// EvidenceCollector gathers evidence from multiple sources
type EvidenceCollector struct {
    temporal  *temporal.TemporalClient  // Session A's API
    incidents *incidents.Database       // Session B's API
    graph     GraphClient
}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector(temporal *temporal.TemporalClient, incidents *incidents.Database, graph GraphClient) *EvidenceCollector {
    return &EvidenceCollector{
        temporal:  temporal,
        incidents: incidents,
        graph:     graph,
    }
}

// Collect gathers all evidence for a file
func (c *EvidenceCollector) Collect(ctx context.Context, filePath string) ([]Evidence, error) {
    var evidence []Evidence

    // 1. Get temporal evidence (co-change)
    coChanged, err := c.temporal.GetCoChangedFiles(filePath, 0.3)
    if err == nil {
        for _, cc := range coChanged {
            evidence = append(evidence, Evidence{
                Type:        EvidenceCoChange,
                Description: fmt.Sprintf("Changes with %s %.0f%% of the time", cc.FileB, cc.Frequency*100),
                Severity:    cc.Frequency,
                Source:      "temporal",
                FilePath:    cc.FileB,
            })
        }
    }

    // 2. Get incident evidence
    incidentStats, err := c.incidents.GetIncidentStats(ctx, filePath)
    if err == nil && incidentStats.TotalIncidents > 0 {
        severity := float64(incidentStats.CriticalCount*3 + incidentStats.HighCount*2) / float64(incidentStats.TotalIncidents*3)
        evidence = append(evidence, Evidence{
            Type:        EvidenceIncident,
            Description: fmt.Sprintf("%d incidents in last 90 days (%d critical)", incidentStats.Last90Days, incidentStats.CriticalCount),
            Severity:    severity,
            Source:      "incidents",
            FilePath:    filePath,
        })
    }

    // 3. Get ownership evidence
    ownership, err := c.temporal.GetOwnershipHistory(filePath)
    if err == nil && ownership.DaysSince < 30 {
        severity := 1.0 - (float64(ownership.DaysSince) / 30.0)  // Recent transition = higher risk
        evidence = append(evidence, Evidence{
            Type:        EvidenceOwnership,
            Description: fmt.Sprintf("Ownership transitioned %d days ago (%s → %s)", ownership.DaysSince, ownership.PreviousOwner, ownership.CurrentOwner),
            Severity:    severity,
            Source:      "temporal",
            FilePath:    filePath,
        })
    }

    // 4. Get structural evidence from graph
    // Query Neo4j for coupling metrics, function count, etc.

    return evidence, nil
}

// Score calculates overall risk score from evidence
func (c *EvidenceCollector) Score(evidence []Evidence) float64 {
    if len(evidence) == 0 {
        return 0.0
    }

    // Weighted average: incidents (50%), co-change (30%), ownership (20%)
    var incidentScore, coChangeScore, ownershipScore float64
    var incidentCount, coChangeCount, ownershipCount int

    for _, e := range evidence {
        switch e.Type {
        case EvidenceIncident:
            incidentScore += e.Severity
            incidentCount++
        case EvidenceCoChange:
            coChangeScore += e.Severity
            coChangeCount++
        case EvidenceOwnership:
            ownershipScore += e.Severity
            ownershipCount++
        }
    }

    // Average each category
    if incidentCount > 0 {
        incidentScore /= float64(incidentCount)
    }
    if coChangeCount > 0 {
        coChangeScore /= float64(coChangeCount)
    }
    if ownershipCount > 0 {
        ownershipScore /= float64(ownershipCount)
    }

    // Weighted combination
    return incidentScore*0.5 + coChangeScore*0.3 + ownershipScore*0.2
}
```

**File:** `internal/agent/synthesis.go`

```go
package agent

import (
    "context"
    "fmt"
    "strings"
)

// Synthesizer generates final risk assessment
type Synthesizer struct {
    llm *LLMClient
}

// NewSynthesizer creates a new synthesizer
func NewSynthesizer(llm *LLMClient) *Synthesizer {
    return &Synthesizer{llm: llm}
}

// Synthesize generates final risk summary from investigation
func (s *Synthesizer) Synthesize(ctx context.Context, inv Investigation) (RiskAssessment, error) {
    // Build synthesis prompt
    prompt := s.buildSynthesisPrompt(inv)

    // Query LLM for final summary
    summary, tokens, err := s.llm.Query(ctx, prompt)
    if err != nil {
        return RiskAssessment{}, err
    }

    inv.TotalTokens += tokens

    // Determine risk level from score
    riskLevel := s.scoreToRiskLevel(inv.RiskScore)

    return RiskAssessment{
        FilePath:    inv.Request.FilePath,
        RiskLevel:   riskLevel,
        RiskScore:   inv.RiskScore,
        Confidence:  inv.Confidence,
        Summary:     summary,
        Evidence:    inv.Evidence,
        Investigation: &inv,
    }, nil
}

// buildSynthesisPrompt constructs the final synthesis prompt
func (s *Synthesizer) buildSynthesisPrompt(inv Investigation) string {
    // Summarize evidence
    var evidenceSummary strings.Builder
    for i, e := range inv.Evidence {
        evidenceSummary.WriteString(fmt.Sprintf("%d. [%s] %s (severity: %.2f)\n", i+1, e.Type, e.Description, e.Severity))
    }

    // Summarize hops
    var hopsSummary strings.Builder
    for _, hop := range inv.Hops {
        hopsSummary.WriteString(fmt.Sprintf("Hop %d: %s\n", hop.HopNumber, hop.Response))
    }

    return fmt.Sprintf(`You are a code risk assessment AI. Synthesize the investigation findings into a concise risk summary.

File: %s
Change type: %s
Risk score: %.2f

Evidence gathered:
%s

Investigation hops:
%s

Task: Write a 2-3 sentence risk summary for a developer explaining:
1. The primary risk factor
2. Why it matters (impact)
3. What to check before merging

Format: Direct, actionable, no fluff.`,
        inv.Request.FilePath,
        inv.Request.ChangeType,
        inv.RiskScore,
        evidenceSummary.String(),
        hopsSummary.String(),
    )
}

// scoreToRiskLevel maps numeric score to risk level
func (s *Synthesizer) scoreToRiskLevel(score float64) RiskLevel {
    switch {
    case score >= 0.8:
        return RiskCritical
    case score >= 0.6:
        return RiskHigh
    case score >= 0.4:
        return RiskMedium
    case score >= 0.2:
        return RiskLow
    default:
        return RiskMinimal
    }
}
```

**File:** `internal/agent/investigator.go`

```go
package agent

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// Investigator orchestrates the full investigation
type Investigator struct {
    llm         *LLMClient
    navigator   *HopNavigator
    collector   *EvidenceCollector
    synthesizer *Synthesizer
}

// NewInvestigator creates a new investigator
func NewInvestigator(llm *LLMClient, temporal *temporal.TemporalClient, incidents *incidents.Database, graph GraphClient) *Investigator {
    navigator := NewHopNavigator(llm, graph, 3)
    collector := NewEvidenceCollector(temporal, incidents, graph)
    synthesizer := NewSynthesizer(llm)

    return &Investigator{
        llm:         llm,
        navigator:   navigator,
        collector:   collector,
        synthesizer: synthesizer,
    }
}

// Investigate performs full Phase 2 investigation (PUBLIC - cmd/crisk/check.go uses this)
func (inv *Investigator) Investigate(ctx context.Context, req InvestigationRequest) (RiskAssessment, error) {
    req.RequestID = uuid.New()
    req.StartedAt = time.Now()

    // Step 1: Collect evidence
    evidence, err := inv.collector.Collect(ctx, req.FilePath)
    if err != nil {
        return RiskAssessment{}, err
    }

    // Step 2: Calculate initial risk score
    riskScore := inv.collector.Score(evidence)

    // Step 3: Navigate graph (hop-by-hop)
    hops, err := inv.navigator.Navigate(ctx, req)
    if err != nil {
        return RiskAssessment{}, err
    }

    // Step 4: Build investigation object
    investigation := Investigation{
        Request:     req,
        Hops:        hops,
        Evidence:    evidence,
        RiskScore:   riskScore,
        Confidence:  inv.calculateConfidence(hops, evidence),
        CompletedAt: time.Now(),
        TotalTokens: sumTokens(hops),
    }

    // Step 5: Synthesize final assessment
    assessment, err := inv.synthesizer.Synthesize(ctx, investigation)
    if err != nil {
        return RiskAssessment{}, err
    }

    return assessment, nil
}

// calculateConfidence estimates confidence based on evidence quality
func (inv *Investigator) calculateConfidence(hops []HopResult, evidence []Evidence) float64 {
    // More evidence + more hops = higher confidence
    evidenceScore := float64(len(evidence)) / 10.0  // Max 10 pieces of evidence
    hopScore := float64(len(hops)) / 3.0            // Max 3 hops

    confidence := (evidenceScore + hopScore) / 2.0
    if confidence > 1.0 {
        confidence = 1.0
    }
    return confidence
}

func sumTokens(hops []HopResult) int {
    total := 0
    for _, hop := range hops {
        total += hop.TokensUsed
    }
    return total
}
```

---

## Integration Points

### 1. Phase 2 Escalation (`cmd/crisk/check.go`)

Find the `check` command and modify it to add Phase 2:

```go
func runCheck(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Phase 1: Baseline metrics (existing code)
    baseline := calculateBaseline(filePath)

    // Phase 2: Escalate if baseline exceeds thresholds
    if baseline.CouplingScore > 0.5 || baseline.IncidentCount > 0 {
        slog.Info("Phase 1 triggered escalation", "file", filePath, "coupling", baseline.CouplingScore, "incidents", baseline.IncidentCount)

        // Initialize LLM client
        llmProvider := os.Getenv("CODERISK_LLM_PROVIDER")  // "openai" or "anthropic"
        apiKey := os.Getenv("CODERISK_API_KEY")
        llm, err := agent.NewLLMClient(llmProvider, apiKey)
        if err != nil {
            return fmt.Errorf("failed to initialize LLM: %w", err)
        }

        // Initialize investigator
        temporal := getTemporalClient()    // Helper function
        incidents := getIncidentsDB()      // Helper function
        graph := getGraphClient()          // Helper function
        investigator := agent.NewInvestigator(llm, temporal, incidents, graph)

        // Run investigation
        req := agent.InvestigationRequest{
            FilePath:    filePath,
            ChangeType:  detectChangeType(filePath),  // Helper function
            DiffPreview: getGitDiff(filePath, 50),    // Helper function
            Baseline:    baseline,
        }

        assessment, err := investigator.Investigate(context.Background(), req)
        if err != nil {
            return fmt.Errorf("Phase 2 investigation failed: %w", err)
        }

        // Display results
        displayRiskAssessment(assessment)

        // Exit with error if CRITICAL or HIGH risk
        if assessment.RiskLevel == agent.RiskCritical || assessment.RiskLevel == agent.RiskHigh {
            return fmt.Errorf("HIGH RISK: %s", assessment.Summary)
        }
    } else {
        slog.Info("Phase 1 complete", "file", filePath, "risk", "LOW")
        fmt.Printf("✅ LOW RISK: %s\n", filePath)
    }

    return nil
}

func displayRiskAssessment(a agent.RiskAssessment) {
    fmt.Printf("\n🔍 RISK ASSESSMENT\n")
    fmt.Printf("File: %s\n", a.FilePath)
    fmt.Printf("Risk Level: %s (score: %.2f, confidence: %.2f)\n", a.RiskLevel, a.RiskScore, a.Confidence)
    fmt.Printf("\nSummary:\n%s\n", a.Summary)
    fmt.Printf("\nEvidence:\n")
    for i, e := range a.Evidence {
        fmt.Printf("  %d. [%s] %s (severity: %.2f)\n", i+1, e.Type, e.Description, e.Severity)
    }
    fmt.Printf("\nInvestigation: %d hops, %d tokens\n", len(a.Investigation.Hops), a.Investigation.TotalTokens)
}
```

---

## Testing Strategy

### Unit Tests (`internal/agent/investigator_test.go`)

```go
func TestLLMClient(t *testing.T) {
    // Mock LLM client
    llm := &MockLLMClient{
        responses: map[string]string{
            "hop1": "High coupling detected with payment_gateway.py",
            "hop2": "3 critical incidents in last 30 days",
            "synthesis": "HIGH RISK: This file caused 3 prod outages. Verify payment flow before merging.",
        },
    }

    response, tokens, err := llm.Query(context.Background(), "hop1")
    assert.NoError(t, err)
    assert.Equal(t, "High coupling detected with payment_gateway.py", response)
    assert.Greater(t, tokens, 0)
}

func TestEvidenceCollector(t *testing.T) {
    // Mock temporal and incidents clients
    temporal := &MockTemporalClient{
        coChanged: []temporal.CoChangeResult{
            {FileB: "payment_gateway.py", Frequency: 0.85},
        },
    }
    incidents := &MockIncidentsDB{
        stats: &incidents.IncidentStats{
            TotalIncidents: 5,
            CriticalCount:  3,
            Last90Days:     4,
        },
    }

    collector := NewEvidenceCollector(temporal, incidents, nil)
    evidence, err := collector.Collect(context.Background(), "payment_processor.py")

    assert.NoError(t, err)
    assert.Len(t, evidence, 2)  // co-change + incidents
    assert.Equal(t, EvidenceCoChange, evidence[0].Type)
    assert.Equal(t, EvidenceIncident, evidence[1].Type)
}
```

### Integration Test (`test/integration/test_agent_investigation.sh`)

```bash
#!/bin/bash

# Test full agent investigation

set -e

echo "Testing Agent Investigation..."

# Prerequisites: Sessions A & B must be complete
# For now, use mock data

# 1. Set LLM credentials (use test account)
export CODERISK_LLM_PROVIDER="openai"
export CODERISK_API_KEY="sk-test-..."  # Test API key

# 2. Run check on high-risk file
cd /tmp/omnara

# This file should trigger Phase 2 (has incidents)
./crisk check src/payment_processor.py

# 3. Verify Phase 2 was triggered
if ! grep -q "Phase 2 investigation" /tmp/crisk.log; then
    echo "ERROR: Phase 2 was not triggered"
    exit 1
fi

# 4. Check risk level in output
if ! grep -q "Risk Level: HIGH" /tmp/crisk.log; then
    echo "ERROR: Expected HIGH risk, got different level"
    exit 1
fi

echo "✅ Agent investigation tests passed!"
```

---

## Checkpoints

### Checkpoint C1: LLM Client Working ✅

**What to verify:**
```bash
# Test OpenAI connection
go run scripts/test_llm.go openai "What is 2+2?"

# Test Anthropic connection
go run scripts/test_llm.go anthropic "What is 2+2?"
```

**Ask me:**
> ✅ LLM clients working! OpenAI GPT-4o and Anthropic Claude 3.5 Sonnet both responding. Test query returned valid response in 1.2s. Token counting accurate. Ready for hop navigation?

---

### Checkpoint C2: Hop Navigation Complete ✅

**What to verify:**
```bash
# Run unit tests
go test ./internal/agent/... -v -run TestHopNavigator

# Test on real file (with mock evidence)
go run scripts/test_hops.go payment_processor.py
```

**Ask me:**
> ✅ Hop navigation working! 3-hop investigation completed in 4.8s. Tokens used: 2,847. Hop prompts correctly structured. Early exit logic working (stops at Hop 1 for obvious risks). Ready for evidence integration?

---

### Checkpoint C3: End-to-End Investigation ✅

**What to verify:**
```bash
# Run full investigation
./crisk check src/payment_processor.py

# Verify output format
# Should show: Risk Level, Summary, Evidence list, Investigation details
```

**Ask me:**
> ✅ Full investigation working! Tested on 10 files from omnara repo. Results:\n- 3 HIGH risk (correctly flagged incident-prone files)\n- 5 MEDIUM risk\n- 2 LOW risk\n- Avg investigation time: 3.2s\n- Avg tokens: 3,100\n\nPhase 2 complete! Ready to merge?

---

## Success Criteria

- [ ] `LLMClient` works with both OpenAI and Anthropic
- [ ] `HopNavigator` executes 1-3 hops with early exit
- [ ] `EvidenceCollector` integrates Session A & B data
- [ ] `Synthesizer` generates concise risk summaries
- [ ] `Investigate()` returns RiskAssessment with confidence
- [ ] `cmd/crisk/check.go` escalates to Phase 2 when needed
- [ ] Unit tests: >70% coverage
- [ ] Integration test passes on real repository
- [ ] Performance: <5s per investigation, <5K tokens

---

## Performance Targets

- LLM query latency: <1.5s per hop
- Total investigation time: <5s (3 hops max)
- Token usage: <5,000 tokens per investigation
- Evidence collection: <500ms
- Synthesis: <1s

---

## References

**Read these first:**
- [agentic_design.md](../01-architecture/agentic_design.md) - Hop navigation algorithm
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Graph structure
- [PARALLEL_SESSION_PLAN_WEEKS2-8.md](PARALLEL_SESSION_PLAN_WEEKS2-8.md) - Your file ownership

**Session A interfaces (use these):**
- `temporal.GetCoChangedFiles(filePath, minFreq) -> []CoChangeResult`
- `temporal.GetOwnershipHistory(filePath) -> *OwnershipHistory`

**Session B interfaces (use these):**
- `incidents.GetIncidentStats(filePath) -> *IncidentStats`
- `incidents.SearchIncidents(query, limit) -> []SearchResult`

---

## What NOT to Do

❌ Don't modify Session A's files (`internal/temporal/`)
❌ Don't modify Session B's files (`internal/incidents/`)
❌ Don't implement your own git parsing (use Session A's API)
❌ Don't implement your own incident search (use Session B's API)
❌ Don't build custom vector embeddings (use BM25 from Session B)

---

## Tips for Success

1. **Mock early** - Use mock data for Sessions A & B until they're done
2. **Test with real LLMs** - Use test API keys, small prompts to verify
3. **Prompt engineering** - Iterate on hop prompts, measure quality
4. **Token budgets** - Track tokens per hop, enforce 10K limit
5. **Early exit** - Don't waste tokens if Hop 1 is conclusive
6. **Ask at checkpoints** - Don't proceed past C1/C2/C3 without confirmation

---

## Dependency Management (While Sessions A & B Incomplete)

### Mock Temporal Client

```go
// Use this until Session A is complete
type MockTemporalClient struct{}

func (m *MockTemporalClient) GetCoChangedFiles(filePath string, minFreq float64) ([]temporal.CoChangeResult, error) {
    // Return hardcoded test data
    return []temporal.CoChangeResult{
        {FileA: filePath, FileB: "payment_gateway.py", Frequency: 0.85},
    }, nil
}
```

### Mock Incidents Database

```go
// Use this until Session B is complete
type MockIncidentsDB struct{}

func (m *MockIncidentsDB) GetIncidentStats(ctx context.Context, filePath string) (*incidents.IncidentStats, error) {
    // Return hardcoded test data
    return &incidents.IncidentStats{
        TotalIncidents: 3,
        CriticalCount:  2,
        Last90Days:     3,
    }, nil
}
```

---

## Getting Started

```bash
# 1. Create your package
mkdir -p internal/agent
touch internal/agent/types.go
touch internal/agent/llm_client.go
touch internal/agent/hop_navigator.go
touch internal/agent/evidence.go
touch internal/agent/synthesis.go
touch internal/agent/investigator.go

# 2. Install SDK dependencies
go get github.com/openai/openai-go
go get github.com/anthropics/anthropic-sdk-go

# 3. Define types first
# Edit internal/agent/types.go (copy from above)

# 4. Implement LLM client
# Edit internal/agent/llm_client.go

# 5. Test LLM connection
export CODERISK_LLM_PROVIDER=openai
export CODERISK_API_KEY=sk-...
go run scripts/test_llm.go "What is 2+2?"

# 6. Implement hop navigator with mock data
# Edit internal/agent/hop_navigator.go

# 7. Test as you go
go test ./internal/agent/... -v
```

---

**You are Session C. Your mission: Make AI investigation work. Go! 🚀**
