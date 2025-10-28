package metrics

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rohankatakam/coderisk/internal/graph"
)

// TestRatioResult represents the test coverage ratio metric result
// Reference: risk_assessment_methodology.md §2.3 - Test Coverage Ratio
type TestRatioResult struct {
	FilePath  string    `json:"file_path"`
	SourceLOC int       `json:"source_loc"` // Lines of code in source file
	TestLOC   int       `json:"test_loc"`   // Lines of code in test files
	Ratio     float64   `json:"ratio"`      // test_loc / source_loc
	RiskLevel RiskLevel `json:"risk_level"` // LOW, MEDIUM, HIGH
	TestFiles []string  `json:"test_files"` // Related test files
}

// CalculateTestRatio computes test coverage ratio for a file
// Reference: risk_assessment_methodology.md §2.3
// Formula: test_ratio = SUM(test_file.loc) / source_file.loc
// Test file discovery: naming conventions (*_test.py, *.test.js) or graph relationship
func CalculateTestRatio(ctx context.Context, neo4j *graph.Client, repoID, filePath string) (*TestRatioResult, error) {
	// Query Neo4j for source file LOC
	// TODO: Once graph construction is complete, use actual LOC from File nodes
	// For now, if no graph data exists, return 0 to indicate insufficient data
	sourceLOC := 0 // Default: no data
	testLOC := 0   // Default: no data

	// Try to query graph for actual file data
	// If file doesn't exist in graph, sourceLOC and testLOC remain 0
	// This signals lack of confidence and should trigger escalation

	// Calculate ratio with special handling for no data
	// If both are 0, we return 0 to indicate "unknown" (not "perfect coverage")
	var ratio float64
	if sourceLOC == 0 && testLOC == 0 {
		ratio = 0.0 // No data = unknown, not 1.0
	} else {
		// Normal smoothing formula when we have data
		ratio = calculateSmoothedRatio(testLOC, sourceLOC)
	}

	// Determine risk level
	riskLevel := classifyTestRatioRisk(ratio)

	result := &TestRatioResult{
		FilePath:  filePath,
		SourceLOC: sourceLOC,
		TestLOC:   testLOC,
		Ratio:     ratio,
		RiskLevel: riskLevel,
		TestFiles: []string{}, // Empty until we query graph
	}

	return result, nil
}

// discoverTestFiles finds test files using naming conventions
// Reference: risk_assessment_methodology.md §2.3 - Naming conventions
func discoverTestFiles(sourceFile string) []string {
	base := filepath.Base(sourceFile)
	dir := filepath.Dir(sourceFile)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	var testFiles []string

	// Python: test_*.py, *_test.py
	if ext == ".py" {
		testFiles = append(testFiles,
			filepath.Join(dir, "test_"+base),
			filepath.Join(dir, nameWithoutExt+"_test.py"),
			filepath.Join(dir, "tests", "test_"+base),
		)
	}

	// JavaScript/TypeScript: *.test.js, *.spec.js
	if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" {
		testFiles = append(testFiles,
			filepath.Join(dir, nameWithoutExt+".test"+ext),
			filepath.Join(dir, nameWithoutExt+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
		)
	}

	// Go: *_test.go
	if ext == ".go" {
		testFiles = append(testFiles,
			filepath.Join(dir, nameWithoutExt+"_test.go"),
		)
	}

	return testFiles
}

// estimateTestLOC estimates test LOC based on file count
// Temporary heuristic until graph has actual LOC data
func estimateTestLOC(testFileCount int) int {
	if testFileCount == 0 {
		return 0
	}
	// Heuristic: assume 50 LOC per test file
	return testFileCount * 50
}

// calculateSmoothedRatio applies smoothing formula from risk_assessment_methodology.md §2.3
func calculateSmoothedRatio(testLOC, sourceLOC int) float64 {
	// smoothed_ratio = (test_loc + 1) / (source_loc + 1)
	return float64(testLOC+1) / float64(sourceLOC+1)
}

// classifyTestRatioRisk applies threshold logic from risk_assessment_methodology.md §2.3
func classifyTestRatioRisk(ratio float64) RiskLevel {
	if ratio >= 0.8 {
		return RiskLevelLow // Excellent coverage
	} else if ratio >= 0.3 {
		return RiskLevelMedium // Adequate coverage
	}
	return RiskLevelHigh // Insufficient coverage
}

// FormatEvidence generates human-readable evidence string
// Reference: risk_assessment_methodology.md §2.3 - Evidence format
func (t *TestRatioResult) FormatEvidence() string {
	if len(t.TestFiles) == 0 {
		return fmt.Sprintf("No test files found (%.0f%% coverage - %s)", t.Ratio*100, t.RiskLevel)
	}
	return fmt.Sprintf("Test ratio: %.2f (%d test LOC / %d source LOC - %s coverage)",
		t.Ratio, t.TestLOC, t.SourceLOC, t.RiskLevel)
}

// ShouldEscalate returns true if this metric triggers Phase 2
// Reference: risk_assessment_methodology.md §2.4 - Escalation logic
func (t *TestRatioResult) ShouldEscalate() bool {
	return t.Ratio < 0.3 // test_ratio < 0.3 → escalate
}

// CalculateTestRatioMultiple computes test ratio across multiple file paths
func CalculateTestRatioMultiple(ctx context.Context, neo4j *graph.Client, repoID string, filePaths []string) (*TestRatioResult, error) {
	if len(filePaths) == 0 {
		return &TestRatioResult{}, nil
	}
	// TODO: Implement proper multi-path query
	// For now, use first path as proxy
	return CalculateTestRatio(ctx, neo4j, repoID, filePaths[0])
}

