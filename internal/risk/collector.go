package risk

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Collector orchestrates Phase 1 data collection (7 queries)
type Collector struct {
	// Dependencies will be injected
}

// NewCollector creates a new Phase 1 data collector
func NewCollector() *Collector {
	return &Collector{}
}

// CollectPhase1Data executes all 7 Phase 1 queries and returns consolidated data
func (c *Collector) CollectPhase1Data(ctx context.Context, req *AnalysisRequest) (*Phase1Data, error) {
	if len(req.FilePaths) == 0 {
		return nil, fmt.Errorf("no file paths provided")
	}

	_ = req.FilePaths[0] // Will be used when implementing actual queries

	data := &Phase1Data{
		QueryDurations: make(map[string]time.Duration),
		CollectionTime: time.Now(),
	}

	// Query 6: Change Complexity (from git diff)
	if err := c.analyzeChangeComplexity(req.GitDiff, data); err != nil {
		return nil, fmt.Errorf("change complexity analysis failed: %w", err)
	}

	// Other queries placeholder - to be implemented with actual graph/DB clients
	data.BlastRadius = 0
	data.DependentFiles = []string{}
	data.CoChangePartners = []CoChangePartner{}
	data.FileOwner = "unknown"
	data.OwnerEmail = ""
	data.ChurnScore = 0.0
	data.CommitCount = 0
	data.TestRatio = 0.0
	data.TestFiles = []string{}
	data.HasTests = false
	data.IncidentCount = 0
	data.Incidents = []string{}
	data.IncidentDensity = 0.0
	data.DeveloperHistory = []DeveloperActivity{}
	data.TeamSize = 0

	return data, nil
}

func (c *Collector) analyzeChangeComplexity(gitDiff string, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["change_complexity"] = time.Since(start)
	}()

	lines := strings.Split(gitDiff, "\n")
	linesAdded := 0
	linesDeleted := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			linesAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			linesDeleted++
		}
	}

	totalChanges := linesAdded + linesDeleted
	complexityScore := float64(totalChanges) / 100.0
	if complexityScore > 1.0 {
		complexityScore = 1.0
	}

	data.LinesAdded = linesAdded
	data.LinesDeleted = linesDeleted
	data.LinesModified = 0
	data.ComplexityScore = complexityScore

	return nil
}
