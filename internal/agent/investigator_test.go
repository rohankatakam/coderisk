package agent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/coderisk/coderisk-go/internal/incidents"
	"github.com/coderisk/coderisk-go/internal/temporal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock LLM Client
type MockLLMClient struct {
	responses map[string]string
	tokens    int
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: map[string]string{
			"hop1": `High coupling detected with payment_gateway.py (co-changes 85% of the time).
This file has 3 critical incidents in the last 30 days, all related to payment processing failures.
The file recently transitioned ownership 15 days ago, which increases risk.`,
			"hop2": `The highest risk factors are:
1. Critical incident on 2025-09-20 (payment timeout) with severity 0.9
2. Strong coupling with payment_gateway.py (frequency 0.85)
3. Recent ownership transition from alice@example.com to bob@example.com`,
			"hop3": `Deep analysis reveals cascading dependencies through the payment module.
The coupling extends to database connection pooling code.
Risk verdict: HIGH (confidence: 0.85)`,
			"synthesis": `HIGH RISK: This file caused 3 production outages in the last month due to payment processing failures.
The recent ownership transition and high coupling with payment_gateway.py amplify the risk.
Before merging, verify payment flow integration tests and ensure connection pooling is configured correctly.`,
		},
		tokens: 150,
	}
}

func (m *MockLLMClient) Query(ctx context.Context, prompt string) (string, int, error) {
	// Determine which hop based on prompt content
	if contains(prompt, "immediate risk factors") {
		return m.responses["hop1"], m.tokens, nil
	} else if contains(prompt, "Based on Hop 1") {
		return m.responses["hop2"], m.tokens, nil
	} else if contains(prompt, "Final deep-dive") {
		return m.responses["hop3"], m.tokens, nil
	} else if contains(prompt, "Synthesize the investigation") {
		return m.responses["synthesis"], m.tokens, nil
	}
	return "Generic response", m.tokens, nil
}

func (m *MockLLMClient) SetModel(model string) {}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Mock Temporal Client
type MockTemporalClient struct {
	coChanged []temporal.CoChangeResult
	ownership *temporal.OwnershipHistory
}

func NewMockTemporalClient() *MockTemporalClient {
	return &MockTemporalClient{
		coChanged: []temporal.CoChangeResult{
			{FileA: "payment_processor.py", FileB: "payment_gateway.py", Frequency: 0.85, CoChanges: 12, WindowDays: 90},
			{FileA: "payment_processor.py", FileB: "database.py", Frequency: 0.65, CoChanges: 8, WindowDays: 90},
		},
		ownership: &temporal.OwnershipHistory{
			FilePath:       "payment_processor.py",
			CurrentOwner:   "bob@example.com",
			PreviousOwner:  "alice@example.com",
			TransitionDate: time.Now().AddDate(0, 0, -15),
			DaysSince:      15,
		},
	}
}

func (m *MockTemporalClient) GetCoChangedFiles(ctx context.Context, filePath string, minFrequency float64) ([]temporal.CoChangeResult, error) {
	var results []temporal.CoChangeResult
	for _, cc := range m.coChanged {
		if cc.Frequency >= minFrequency {
			results = append(results, cc)
		}
	}
	return results, nil
}

func (m *MockTemporalClient) GetOwnershipHistory(ctx context.Context, filePath string) (*temporal.OwnershipHistory, error) {
	return m.ownership, nil
}

// Mock Incidents Client
type MockIncidentsClient struct {
	stats *incidents.IncidentStats
}

func NewMockIncidentsClient() *MockIncidentsClient {
	lastIncident := time.Now().AddDate(0, 0, -5)
	return &MockIncidentsClient{
		stats: &incidents.IncidentStats{
			FilePath:       "payment_processor.py",
			TotalIncidents: 5,
			Last30Days:     3,
			Last90Days:     4,
			CriticalCount:  3,
			HighCount:      1,
			LastIncident:   &lastIncident,
			AvgResolution:  2 * time.Hour,
		},
	}
}

func (m *MockIncidentsClient) GetIncidentStats(ctx context.Context, filePath string) (*incidents.IncidentStats, error) {
	return m.stats, nil
}

func (m *MockIncidentsClient) SearchIncidents(ctx context.Context, query string, limit int) ([]incidents.SearchResult, error) {
	return []incidents.SearchResult{}, nil
}

// Mock Graph Client
type MockGraphClient struct{}

func (m *MockGraphClient) GetNodes(ctx context.Context, nodeIDs []string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockGraphClient) GetNeighbors(ctx context.Context, nodeID string, edgeTypes []string, maxDepth int) ([]interface{}, error) {
	return []interface{}{}, nil
}

// Tests

func TestLLMClient(t *testing.T) {
	llm := NewMockLLMClient()

	t.Run("Hop1Query", func(t *testing.T) {
		response, tokens, err := llm.Query(context.Background(), "What are the immediate risk factors?")
		require.NoError(t, err)
		assert.Contains(t, response, "High coupling")
		assert.Greater(t, tokens, 0)
	})

	t.Run("SynthesisQuery", func(t *testing.T) {
		response, tokens, err := llm.Query(context.Background(), "Synthesize the investigation findings")
		require.NoError(t, err)
		assert.Contains(t, response, "HIGH RISK")
		assert.Greater(t, tokens, 0)
	})
}

func TestEvidenceCollector(t *testing.T) {
	temporal := NewMockTemporalClient()
	incidents := NewMockIncidentsClient()
	graph := &MockGraphClient{}

	collector := NewEvidenceCollector(temporal, incidents, graph)

	t.Run("CollectEvidence", func(t *testing.T) {
		evidence, err := collector.Collect(context.Background(), "payment_processor.py")
		require.NoError(t, err)

		// Should have at least 3 types of evidence
		assert.GreaterOrEqual(t, len(evidence), 3)

		// Check for co-change evidence
		hasCoChange := false
		hasIncident := false
		hasOwnership := false

		for _, e := range evidence {
			switch e.Type {
			case EvidenceCoChange:
				hasCoChange = true
				assert.Greater(t, e.Severity, 0.0)
			case EvidenceIncident:
				hasIncident = true
				assert.Greater(t, e.Severity, 0.0)
			case EvidenceOwnership:
				hasOwnership = true
				assert.Greater(t, e.Severity, 0.0)
			}
		}

		assert.True(t, hasCoChange, "Should have co-change evidence")
		assert.True(t, hasIncident, "Should have incident evidence")
		assert.True(t, hasOwnership, "Should have ownership evidence")
	})

	t.Run("ScoreEvidence", func(t *testing.T) {
		evidence := []Evidence{
			{Type: EvidenceIncident, Severity: 0.9},
			{Type: EvidenceCoChange, Severity: 0.85},
			{Type: EvidenceOwnership, Severity: 0.5},
		}

		score := collector.Score(evidence)

		// Score should be weighted: incidents (50%), co-change (30%), ownership (20%)
		// Expected: 0.9*0.5 + 0.85*0.3 + 0.5*0.2 = 0.45 + 0.255 + 0.1 = 0.805
		assert.InDelta(t, 0.805, score, 0.01)
	})
}

func TestHopNavigator(t *testing.T) {
	llm := NewMockLLMClient()
	graph := &MockGraphClient{}

	navigator := NewHopNavigator(llm, graph, 3)

	t.Run("Navigate3Hops", func(t *testing.T) {
		req := InvestigationRequest{
			FilePath:   "payment_processor.py",
			ChangeType: "modify",
			DiffPreview: `
@@ -10,7 +10,7 @@ def process_payment(amount, customer_id):
-    connection = get_connection()
+    connection = get_connection_pool().acquire()
`,
			Baseline: BaselineMetrics{
				CouplingScore:     0.8,
				CoChangeFrequency: 0.85,
				IncidentCount:     3,
				OwnershipDays:     15,
			},
		}

		hops, err := navigator.Navigate(context.Background(), req)
		require.NoError(t, err)

		// Should execute at least 1 hop
		assert.GreaterOrEqual(t, len(hops), 1)
		assert.LessOrEqual(t, len(hops), 3)

		// Check hop 1
		assert.Equal(t, 1, hops[0].HopNumber)
		assert.NotEmpty(t, hops[0].Query)
		assert.NotEmpty(t, hops[0].Response)
		assert.Greater(t, hops[0].TokensUsed, 0)
	})

	t.Run("EarlyExit", func(t *testing.T) {
		// Create a mock that returns "critical incident" in hop 1
		criticalLLM := &MockLLMClient{
			responses: map[string]string{
				"hop1": "This file has a critical incident with production outage risk. Severe coupling detected.",
			},
			tokens: 100,
		}

		nav := NewHopNavigator(criticalLLM, graph, 3)

		req := InvestigationRequest{
			FilePath:   "critical_file.py",
			ChangeType: "modify",
			Baseline: BaselineMetrics{
				IncidentCount: 5,
			},
		}

		hops, err := nav.Navigate(context.Background(), req)
		require.NoError(t, err)

		// Should stop after hop 1 due to critical risk
		assert.Equal(t, 1, len(hops))
	})
}

func TestSynthesizer(t *testing.T) {
	llm := NewMockLLMClient()
	synthesizer := NewSynthesizer(llm)

	t.Run("Synthesize", func(t *testing.T) {
		investigation := Investigation{
			Request: InvestigationRequest{
				FilePath:   "payment_processor.py",
				ChangeType: "modify",
			},
			Hops: []HopResult{
				{HopNumber: 1, Response: "High coupling detected", TokensUsed: 150},
				{HopNumber: 2, Response: "Critical incidents found", TokensUsed: 150},
			},
			Evidence: []Evidence{
				{Type: EvidenceIncident, Description: "3 critical incidents", Severity: 0.9},
			},
			RiskScore:  0.8,
			Confidence: 0.85,
		}

		assessment, err := synthesizer.Synthesize(context.Background(), investigation)
		require.NoError(t, err)

		assert.Equal(t, "payment_processor.py", assessment.FilePath)
		assert.Equal(t, RiskCritical, assessment.RiskLevel) // 0.8 >= 0.8 is CRITICAL
		assert.Equal(t, 0.8, assessment.RiskScore)
		assert.Equal(t, 0.85, assessment.Confidence)
		assert.NotEmpty(t, assessment.Summary)
		assert.Contains(t, assessment.Summary, "HIGH RISK")
	})

	t.Run("RiskLevelMapping", func(t *testing.T) {
		tests := []struct {
			score    float64
			expected RiskLevel
		}{
			{0.9, RiskCritical},
			{0.7, RiskHigh},
			{0.5, RiskMedium},
			{0.3, RiskLow},
			{0.1, RiskMinimal},
		}

		for _, tt := range tests {
			level := synthesizer.scoreToRiskLevel(tt.score)
			assert.Equal(t, tt.expected, level)
		}
	})
}

func TestInvestigator(t *testing.T) {
	llm := NewMockLLMClient()
	temporal := NewMockTemporalClient()
	incidents := NewMockIncidentsClient()
	graph := &MockGraphClient{}

	investigator := NewInvestigator(llm, temporal, incidents, graph)

	t.Run("FullInvestigation", func(t *testing.T) {
		req := InvestigationRequest{
			FilePath:   "payment_processor.py",
			ChangeType: "modify",
			DiffPreview: `
@@ -10,7 +10,7 @@ def process_payment(amount, customer_id):
-    connection = get_connection()
+    connection = get_connection_pool().acquire()
`,
			Baseline: BaselineMetrics{
				CouplingScore:     0.8,
				CoChangeFrequency: 0.85,
				IncidentCount:     3,
				OwnershipDays:     15,
			},
		}

		assessment, err := investigator.Investigate(context.Background(), req)
		require.NoError(t, err)

		// Check assessment structure
		assert.NotEqual(t, uuid.Nil, assessment.Investigation.Request.RequestID)
		assert.Equal(t, "payment_processor.py", assessment.FilePath)
		assert.Greater(t, assessment.RiskScore, 0.0)
		assert.Greater(t, assessment.Confidence, 0.0)
		assert.NotEmpty(t, assessment.Summary)
		assert.NotNil(t, assessment.Investigation)

		// Check evidence was collected
		assert.GreaterOrEqual(t, len(assessment.Evidence), 1)

		// Check hops were executed
		assert.GreaterOrEqual(t, len(assessment.Investigation.Hops), 1)
		assert.Greater(t, assessment.Investigation.TotalTokens, 0)

		// Check timestamps
		assert.False(t, assessment.Investigation.Request.StartedAt.IsZero())
		assert.False(t, assessment.Investigation.CompletedAt.IsZero())
		assert.True(t, assessment.Investigation.CompletedAt.After(assessment.Investigation.Request.StartedAt))
	})

	t.Run("ConfidenceCalculation", func(t *testing.T) {
		// Test with varying amounts of evidence and hops
		tests := []struct {
			name          string
			evidenceCount int
			hopCount      int
			minConfidence float64
		}{
			{"FullEvidence", 10, 3, 0.8},
			{"PartialEvidence", 5, 2, 0.5},
			{"MinimalEvidence", 2, 1, 0.2},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				evidence := make([]Evidence, tt.evidenceCount)
				hops := make([]HopResult, tt.hopCount)

				confidence := investigator.calculateConfidence(hops, evidence)
				assert.GreaterOrEqual(t, confidence, tt.minConfidence)
				assert.LessOrEqual(t, confidence, 1.0)
			})
		}
	})
}
