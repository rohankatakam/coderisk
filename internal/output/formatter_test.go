package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/coderisk/coderisk-go/internal/models"
)

func TestQuietFormatter(t *testing.T) {
	tests := []struct {
		name     string
		result   *models.RiskResult
		expected string
	}{
		{
			name: "low risk",
			result: &models.RiskResult{
				RiskLevel: "LOW",
			},
			expected: "✅ LOW risk\n",
		},
		{
			name: "none risk",
			result: &models.RiskResult{
				RiskLevel: "NONE",
			},
			expected: "✅ NONE risk\n",
		},
		{
			name: "medium risk with 1 issue",
			result: &models.RiskResult{
				RiskLevel: "MEDIUM",
				Issues: []models.RiskIssue{
					{Message: "No tests"},
				},
			},
			expected: "⚠️  MEDIUM risk: 1 issues detected\nRun 'crisk check' for details\n",
		},
		{
			name: "high risk with 3 issues",
			result: &models.RiskResult{
				RiskLevel: "HIGH",
				Issues: []models.RiskIssue{
					{Message: "No tests"},
					{Message: "High coupling"},
					{Message: "No error handling"},
				},
			},
			expected: "⚠️  HIGH risk: 3 issues detected\nRun 'crisk check' for details\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := &QuietFormatter{}
			err := formatter.Format(tt.result, &buf)

			if err != nil {
				t.Errorf("Format() returned error: %v", err)
			}

			if buf.String() != tt.expected {
				t.Errorf("Format() output mismatch:\nGot:  %q\nWant: %q", buf.String(), tt.expected)
			}
		})
	}
}

func TestStandardFormatter(t *testing.T) {
	result := &models.RiskResult{
		Branch:       "feature/test",
		FilesChanged: 3,
		RiskLevel:    "MEDIUM",
		Issues: []models.RiskIssue{
			{
				Severity: "MEDIUM",
				File:     "auth.py",
				Message:  "No test coverage",
			},
			{
				Severity: "LOW",
				File:     "utils.py",
				Message:  "High complexity",
			},
		},
		Recommendations: []string{
			"Add tests for auth.py",
			"Reduce complexity in utils.py",
		},
	}

	var buf bytes.Buffer
	formatter := &StandardFormatter{}
	err := formatter.Format(result, &buf)

	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "🔍 CodeRisk Analysis") {
		t.Error("Output missing header")
	}
	if !strings.Contains(output, "Branch: feature/test") {
		t.Error("Output missing branch")
	}
	if !strings.Contains(output, "Files changed: 3") {
		t.Error("Output missing file count")
	}
	if !strings.Contains(output, "Risk level: MEDIUM") {
		t.Error("Output missing risk level")
	}

	// Check issues
	if !strings.Contains(output, "Issues:") {
		t.Error("Output missing issues section")
	}
	if !strings.Contains(output, "auth.py - No test coverage") {
		t.Error("Output missing first issue")
	}
	if !strings.Contains(output, "utils.py - High complexity") {
		t.Error("Output missing second issue")
	}

	// Check recommendations
	if !strings.Contains(output, "Recommendations:") {
		t.Error("Output missing recommendations section")
	}
	if !strings.Contains(output, "Add tests for auth.py") {
		t.Error("Output missing first recommendation")
	}

	// Check next steps
	if !strings.Contains(output, "Run 'crisk check --explain' for investigation trace") {
		t.Error("Output missing next steps")
	}
}

func TestExplainFormatter(t *testing.T) {
	threshold1 := 10.0
	threshold2 := 0.7

	result := &models.RiskResult{
		StartTime: time.Date(2025, 10, 4, 14, 23, 15, 0, time.UTC),
		EndTime:   time.Date(2025, 10, 4, 14, 23, 17, 0, time.UTC),
		Duration:  2 * time.Second,
		RiskLevel: "MEDIUM",
		InvestigationTrace: []models.InvestigationHop{
			{
				NodeID: "auth.py",
				ChangedEntities: []models.ChangedEntity{
					{
						Name:      "authenticate_user()",
						StartLine: 45,
						EndLine:   67,
					},
				},
				Metrics: []models.Metric{
					{
						Name:      "Complexity",
						Value:     6,
						Threshold: &threshold1,
					},
					{
						Name:      "Test coverage",
						Value:     0,
						Threshold: &threshold2,
					},
				},
				Decision:  "Investigate callers",
				Reasoning: "High coupling signal",
			},
		},
		Evidence: []string{
			"Zero test coverage on changed functions",
			"Strong temporal coupling (85%)",
		},
		Recommendations: []string{
			"Add integration tests",
			"Add unit tests",
		},
		NextSteps: []string{
			"crisk suggest-tests auth.py",
		},
	}

	var buf bytes.Buffer
	formatter := &ExplainFormatter{}
	err := formatter.Format(result, &buf)

	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "🔍 CodeRisk Investigation Report") {
		t.Error("Output missing header")
	}
	if !strings.Contains(output, "Agent hops: 1") {
		t.Error("Output missing hop count")
	}

	// Check hop details
	if !strings.Contains(output, "Hop 1: auth.py") {
		t.Error("Output missing hop 1")
	}
	if !strings.Contains(output, "authenticate_user() (lines 45-67)") {
		t.Error("Output missing changed entity")
	}
	if !strings.Contains(output, "Metrics calculated:") {
		t.Error("Output missing metrics section")
	}
	if !strings.Contains(output, "Agent decision: Investigate callers") {
		t.Error("Output missing agent decision")
	}
	if !strings.Contains(output, "Reasoning: High coupling signal") {
		t.Error("Output missing reasoning")
	}

	// Check final assessment
	if !strings.Contains(output, "Final Assessment") {
		t.Error("Output missing final assessment")
	}
	if !strings.Contains(output, "Risk Level: MEDIUM") {
		t.Error("Output missing risk level")
	}
	if !strings.Contains(output, "Evidence:") {
		t.Error("Output missing evidence")
	}
	if !strings.Contains(output, "Recommendations (priority order):") {
		t.Error("Output missing recommendations")
	}
	if !strings.Contains(output, "Suggested next steps:") {
		t.Error("Output missing next steps")
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		level    VerbosityLevel
		expected string
	}{
		{VerbosityQuiet, "*output.QuietFormatter"},
		{VerbosityStandard, "*output.StandardFormatter"},
		{VerbosityExplain, "*output.ExplainFormatter"},
		{VerbosityAIMode, "*output.AIFormatter"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Level_%d", tt.level), func(t *testing.T) {
			formatter := NewFormatter(tt.level)
			typeName := fmt.Sprintf("%T", formatter)
			if typeName != tt.expected {
				t.Errorf("NewFormatter(%v) returned %s, want %s", tt.level, typeName, tt.expected)
			}
		})
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"HIGH", "🔴"},
		{"CRITICAL", "🔴"},
		{"MEDIUM", "⚠️ "},
		{"LOW", "ℹ️ "},
		{"UNKNOWN", "•"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			emoji := severityEmoji(tt.severity)
			if emoji != tt.expected {
				t.Errorf("severityEmoji(%q) = %q, want %q", tt.severity, emoji, tt.expected)
			}
		})
	}
}

func TestMetricStatus(t *testing.T) {
	formatter := &ExplainFormatter{}

	tests := []struct {
		name     string
		metric   models.Metric
		expected string
	}{
		{
			name: "below threshold",
			metric: models.Metric{
				Value:     5.0,
				Threshold: ptrFloat64(10.0),
			},
			expected: "✅",
		},
		{
			name: "above threshold",
			metric: models.Metric{
				Value:     15.0,
				Threshold: ptrFloat64(10.0),
			},
			expected: "❌",
		},
		{
			name: "above warning but below threshold",
			metric: models.Metric{
				Value:     8.0,
				Warning:   ptrFloat64(7.0),
				Threshold: ptrFloat64(10.0),
			},
			expected: "⚠️ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := formatter.metricStatus(tt.metric)
			if status != tt.expected {
				t.Errorf("metricStatus() = %q, want %q", status, tt.expected)
			}
		})
	}
}

func ptrFloat64(f float64) *float64 {
	return &f
}
