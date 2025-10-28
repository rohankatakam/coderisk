package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/rohankatakam/coderisk/internal/git"
)

func TestKickoffPromptBuilder_BuildSystemPrompt(t *testing.T) {
	builder := NewKickoffPromptBuilder([]FileChangeContext{})
	prompt := builder.buildSystemPrompt()

	// Verify key phrases are present
	requiredPhrases := []string{
		"Incident prevention",
		"NOT responsible for:",
		"Code style violations",
		"Security vulnerabilities",
		"ARE responsible for:",
		"Incident history",
		"Ownership gaps",
		"Co-change patterns",
		"query_ownership",
		"query_incident_history",
		"stale ownership = incident risk",
		"incomplete changes = incident risk",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("System prompt missing required phrase: %s", phrase)
		}
	}

	// Verify it explicitly states NOT about code quality
	if !strings.Contains(prompt, "not code quality") {
		t.Error("System prompt should explicitly state 'not code quality'")
	}

	// Verify it doesn't contain positive code quality language
	forbiddenPhrases := []string{
		"improve code quality",
		"better code quality",
		"best code",
		"clean code",
	}

	for _, phrase := range forbiddenPhrases {
		if strings.Contains(strings.ToLower(prompt), strings.ToLower(phrase)) {
			t.Errorf("System prompt contains forbidden phrase: %s", phrase)
		}
	}
}

func TestKickoffPromptBuilder_BuildFileSection_NewFile(t *testing.T) {
	builder := NewKickoffPromptBuilder([]FileChangeContext{})

	change := FileChangeContext{
		FilePath:          "src/new_feature.py",
		ChangeStatus:      "ADDED",
		LinesAdded:        100,
		LinesDeleted:      0,
		ResolutionMatches: []git.FileMatch{}, // No historical data
		DiffSummary:       "def new_feature():\n    pass",
	}

	section := builder.buildFileSection(1, change)

	// Verify key elements
	if !strings.Contains(section, "File 1: src/new_feature.py") {
		t.Error("File section missing file path")
	}

	if !strings.Contains(section, "ADDED") {
		t.Error("File section missing change status")
	}

	if !strings.Contains(section, "No historical data (new file)") {
		t.Error("File section should indicate new file")
	}

	if !strings.Contains(section, "skip graph queries") {
		t.Error("File section should indicate to skip graph queries")
	}
}

func TestKickoffPromptBuilder_BuildFileSection_RenamedFile(t *testing.T) {
	builder := NewKickoffPromptBuilder([]FileChangeContext{})

	change := FileChangeContext{
		FilePath:     "src/auth/login.py",
		ChangeStatus: "MODIFIED",
		LinesAdded:   50,
		LinesDeleted: 20,
		ResolutionMatches: []git.FileMatch{
			{
				HistoricalPath: "auth/login.py",
				Confidence:     0.95,
				Method:         "git-follow",
			},
			{
				HistoricalPath: "backend/login.py",
				Confidence:     0.95,
				Method:         "git-follow",
			},
		},
		DiffSummary:       "def login():\n    ...",
		CouplingScore:     0.4,  // 8 dependencies
		CoChangeFrequency: 0.65,
		IncidentCount:     1,
		ChurnScore:        0.5,
		OwnerEmail:        "alice@example.com",
		LastModified:      time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
	}

	section := builder.buildFileSection(1, change)

	// Verify resolution information
	if !strings.Contains(section, "Historical paths found in graph:") {
		t.Error("File section missing historical paths")
	}

	if !strings.Contains(section, "auth/login.py") {
		t.Error("File section missing first historical path")
	}

	if !strings.Contains(section, "backend/login.py") {
		t.Error("File section missing second historical path")
	}

	if !strings.Contains(section, "confidence: 95%") {
		t.Error("File section missing confidence score")
	}

	if !strings.Contains(section, "git-follow") {
		t.Error("File section missing resolution method")
	}

	// Verify entry points for queries
	if !strings.Contains(section, "Entry points for queries:") {
		t.Error("File section missing entry points")
	}

	// Verify Phase 1 metrics
	if !strings.Contains(section, "Coupling: 8 dependencies") {
		t.Error("File section missing coupling info")
	}

	if !strings.Contains(section, "Co-change frequency: 0.65") {
		t.Error("File section missing co-change frequency")
	}

	if !strings.Contains(section, "frequently changes with other files") {
		t.Error("File section missing co-change interpretation")
	}

	if !strings.Contains(section, "Recent incidents: 1") {
		t.Error("File section missing incident count")
	}

	if !strings.Contains(section, "⚠️") {
		t.Error("File section missing incident warning symbol")
	}

	if !strings.Contains(section, "Churn score: 0.5") {
		t.Error("File section missing churn score")
	}

	if !strings.Contains(section, "moderate recent activity") {
		t.Error("File section missing churn interpretation")
	}

	if !strings.Contains(section, "Owner: alice@example.com") {
		t.Error("File section missing owner email")
	}

	if !strings.Contains(section, "60 days ago") {
		t.Error("File section missing last modified date")
	}
}

func TestKickoffPromptBuilder_ClassifyChangeSize(t *testing.T) {
	builder := NewKickoffPromptBuilder([]FileChangeContext{})

	tests := []struct {
		linesAdded   int
		linesDeleted int
		expected     string
	}{
		{5, 2, "SMALL"},
		{30, 0, "SMALL"},
		{50, 20, "MEDIUM"},
		{100, 0, "MEDIUM"},
		{150, 50, "LARGE"},
		{200, 100, "LARGE"},
	}

	for _, test := range tests {
		result := builder.classifyChangeSize(test.linesAdded, test.linesDeleted)
		if result != test.expected {
			t.Errorf("classifyChangeSize(%d, %d) = %s, expected %s",
				test.linesAdded, test.linesDeleted, result, test.expected)
		}
	}
}

func TestKickoffPromptBuilder_BuildInvestigationGuidance(t *testing.T) {
	// Test with various risk signals
	changes := []FileChangeContext{
		{
			FilePath:          "src/auth/login.py",
			LinesAdded:        150, // Large change
			LinesDeleted:      20,
			IncidentCount:     2,   // Incident history
			CoChangeFrequency: 0.7, // High co-change
			ResolutionMatches: []git.FileMatch{
				{HistoricalPath: "auth/login.py", Confidence: 0.95, Method: "git-follow"},
				{HistoricalPath: "backend/login.py", Confidence: 0.95, Method: "git-follow"},
			},
		},
	}

	builder := NewKickoffPromptBuilder(changes)
	guidance := builder.buildInvestigationGuidance()

	// Verify structure
	if !strings.Contains(guidance, "# Your Task") {
		t.Error("Guidance missing task header")
	}

	if !strings.Contains(guidance, "Focus areas:") {
		t.Error("Guidance missing focus areas")
	}

	// Verify dynamic focus areas based on risk signals
	if !strings.Contains(guidance, "LARGE change") {
		t.Error("Guidance should mention large change")
	}

	if !strings.Contains(guidance, "incident history") {
		t.Error("Guidance should mention incident history")
	}

	if !strings.Contains(guidance, "high co-change frequency") {
		t.Error("Guidance should mention high co-change frequency")
	}

	if !strings.Contains(guidance, "renamed 2 times") {
		t.Error("Guidance should mention file renames")
	}

	// Verify complementary positioning note
	if !strings.Contains(guidance, "This assessment focuses on incident risk") {
		t.Error("Guidance missing complementary positioning note")
	}

	if !strings.Contains(guidance, "additional style, security, and linting checks") {
		t.Error("Guidance should remind about additional checks")
	}
}

func TestKickoffPromptBuilder_BuildInvestigationGuidance_NoRiskSignals(t *testing.T) {
	// Test with low-risk change
	changes := []FileChangeContext{
		{
			FilePath:          "src/utils/helper.py",
			LinesAdded:        10,
			LinesDeleted:      5,
			IncidentCount:     0,
			CoChangeFrequency: 0.1,
			ResolutionMatches: []git.FileMatch{
				{HistoricalPath: "src/utils/helper.py", Confidence: 1.0, Method: "exact"},
			},
		},
	}

	builder := NewKickoffPromptBuilder(changes)
	guidance := builder.buildInvestigationGuidance()

	// Verify generic guidance is provided
	if !strings.Contains(guidance, "Review ownership") {
		t.Error("Guidance should provide generic ownership check")
	}

	if !strings.Contains(guidance, "Check for co-change partners") {
		t.Error("Guidance should provide generic co-change check")
	}

	if !strings.Contains(guidance, "Assess blast radius") {
		t.Error("Guidance should provide generic blast radius check")
	}
}

func TestKickoffPromptBuilder_BuildKickoffPrompt_Complete(t *testing.T) {
	// Test complete prompt generation
	changes := []FileChangeContext{
		{
			FilePath:          "src/auth/login.py",
			ChangeStatus:      "MODIFIED",
			LinesAdded:        150,
			LinesDeleted:      20,
			ResolutionMatches: []git.FileMatch{
				{HistoricalPath: "auth/login.py", Confidence: 0.95, Method: "git-follow"},
			},
			DiffSummary:       "def login():\n    # new implementation",
			CouplingScore:     0.4,
			CoChangeFrequency: 0.65,
			IncidentCount:     1,
			ChurnScore:        0.5,
			OwnerEmail:        "alice@example.com",
			LastModified:      time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			FilePath:          "tests/test_auth.py",
			ChangeStatus:      "ADDED",
			LinesAdded:        200,
			LinesDeleted:      0,
			ResolutionMatches: []git.FileMatch{}, // New file
			DiffSummary:       "def test_login():\n    assert True",
		},
	}

	builder := NewKickoffPromptBuilder(changes)
	prompt := builder.BuildKickoffPrompt()

	// Verify all major sections are present
	sections := []string{
		"You are a risk assessment investigator",
		"Incident prevention",
		"# Changes to Investigate",
		"## File 1: src/auth/login.py",
		"## File 2: tests/test_auth.py",
		"# Your Task",
		"Focus areas:",
	}

	for _, section := range sections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Complete prompt missing section: %s", section)
		}
	}

	// Verify separator lines
	if strings.Count(prompt, "---") < 3 {
		t.Error("Complete prompt should have at least 3 separator lines")
	}
}

func TestTruncateDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		maxChars int
		expected string
	}{
		{
			name:     "short diff unchanged",
			diff:     "short diff",
			maxChars: 100,
			expected: "short diff",
		},
		{
			name:     "long diff truncated",
			diff:     strings.Repeat("a", 500),
			maxChars: 100,
			expected: strings.Repeat("a", 100) + "\n... (truncated)",
		},
		{
			name:     "truncate at newline",
			diff:     strings.Repeat("line\n", 100),
			maxChars: 250,
			expected: strings.Repeat("line\n", 49) + "... (truncated)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := truncateDiff(test.diff, test.maxChars)
			if result != test.expected {
				t.Errorf("truncateDiff() = %q, expected %q", result, test.expected)
			}
		})
	}
}

func TestFromPhase1Result(t *testing.T) {
	// This is an integration helper - just verify it constructs properly
	matches := []git.FileMatch{
		{HistoricalPath: "old/path.py", Confidence: 0.95, Method: "git-follow"},
	}

	ctx := FromPhase1Result(
		"src/path.py",
		"MODIFIED",
		50,
		10,
		matches,
		"diff content",
		nil, // No Phase 1 result
	)

	if ctx.FilePath != "src/path.py" {
		t.Errorf("FilePath = %s, expected src/path.py", ctx.FilePath)
	}

	if ctx.ChangeStatus != "MODIFIED" {
		t.Errorf("ChangeStatus = %s, expected MODIFIED", ctx.ChangeStatus)
	}

	if ctx.LinesAdded != 50 {
		t.Errorf("LinesAdded = %d, expected 50", ctx.LinesAdded)
	}

	if ctx.ResolutionMethod != "git-follow" {
		t.Errorf("ResolutionMethod = %s, expected git-follow", ctx.ResolutionMethod)
	}

	if ctx.ResolutionConfidence != 0.95 {
		t.Errorf("ResolutionConfidence = %.2f, expected 0.95", ctx.ResolutionConfidence)
	}
}
