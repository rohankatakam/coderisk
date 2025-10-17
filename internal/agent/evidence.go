package agent

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/incidents"
	"github.com/rohankatakam/coderisk/internal/temporal"
)

// TemporalClient interface for Session A's temporal analysis
type TemporalClient interface {
	GetCoChangedFiles(ctx context.Context, filePath string, minFrequency float64) ([]temporal.CoChangeResult, error)
	GetOwnershipHistory(ctx context.Context, filePath string) (*temporal.OwnershipHistory, error)
}

// IncidentsClient interface for Session B's incident database
type IncidentsClient interface {
	GetIncidentStats(ctx context.Context, filePath string) (*incidents.IncidentStats, error)
	SearchIncidents(ctx context.Context, query string, limit int) ([]incidents.SearchResult, error)
}

// GraphClient interface for graph queries
type GraphClient interface {
	GetNodes(ctx context.Context, nodeIDs []string) (map[string]interface{}, error)
	GetNeighbors(ctx context.Context, nodeID string, edgeTypes []string, maxDepth int) ([]interface{}, error)
}

// EvidenceCollector gathers evidence from multiple sources
type EvidenceCollector struct {
	temporal  TemporalClient
	incidents IncidentsClient
	graph     GraphClient
}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector(temporal TemporalClient, incidents IncidentsClient, graph GraphClient) *EvidenceCollector {
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
	if c.temporal != nil {
		coChanged, err := c.temporal.GetCoChangedFiles(ctx, filePath, 0.3)
		if err == nil && len(coChanged) > 0 {
			// Take top 3 co-changed files
			limit := 3
			if len(coChanged) < limit {
				limit = len(coChanged)
			}
			for i := 0; i < limit; i++ {
				cc := coChanged[i]
				evidence = append(evidence, Evidence{
					Type:        EvidenceCoChange,
					Description: fmt.Sprintf("Changes with %s %.0f%% of the time", cc.FileB, cc.Frequency*100),
					Severity:    cc.Frequency,
					Source:      "temporal",
					FilePath:    cc.FileB,
				})
			}
		}
	}

	// 2. Get incident evidence
	if c.incidents != nil {
		incidentStats, err := c.incidents.GetIncidentStats(ctx, filePath)
		if err == nil && incidentStats.TotalIncidents > 0 {
			// Calculate severity based on incident counts and types
			// Formula: (critical_count * 3 + high_count * 2) / (total_incidents * 3)
			// This gives higher weight to critical incidents
			severity := float64(incidentStats.CriticalCount*3+incidentStats.HighCount*2) / float64(incidentStats.TotalIncidents*3)
			if severity > 1.0 {
				severity = 1.0
			}

			desc := fmt.Sprintf("%d incidents in last 90 days", incidentStats.Last90Days)
			if incidentStats.CriticalCount > 0 {
				desc = fmt.Sprintf("%d critical incidents, %d total in last 90 days", incidentStats.CriticalCount, incidentStats.Last90Days)
			}

			evidence = append(evidence, Evidence{
				Type:        EvidenceIncident,
				Description: desc,
				Severity:    severity,
				Source:      "incidents",
				FilePath:    filePath,
			})
		}
	}

	// 3. Get ownership evidence
	if c.temporal != nil {
		ownership, err := c.temporal.GetOwnershipHistory(ctx, filePath)
		if err == nil && ownership.DaysSince > 0 && ownership.DaysSince < 90 {
			// Recent ownership transition = higher risk
			// Severity decreases linearly: 1.0 at day 0, 0.0 at day 90
			severity := 1.0 - (float64(ownership.DaysSince) / 90.0)
			if severity < 0.0 {
				severity = 0.0
			}

			evidence = append(evidence, Evidence{
				Type:        EvidenceOwnership,
				Description: fmt.Sprintf("Ownership transitioned %d days ago (%s → %s)", ownership.DaysSince, ownership.PreviousOwner, ownership.CurrentOwner),
				Severity:    severity,
				Source:      "temporal",
				FilePath:    filePath,
			})
		}
	}

	// 4. Get structural evidence from graph (future implementation)
	// TODO: Query Neo4j for coupling metrics, function count, etc.
	// if c.graph != nil {
	//     // Example: High coupling score
	//     // evidence = append(evidence, Evidence{
	//     //     Type:        EvidenceCoupling,
	//     //     Description: fmt.Sprintf("High coupling: %d dependencies", depCount),
	//     //     Severity:    couplingScore,
	//     //     Source:      "structure",
	//     //     FilePath:    filePath,
	//     // })
	// }

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
	finalScore := incidentScore*0.5 + coChangeScore*0.3 + ownershipScore*0.2

	// Cap at 1.0
	if finalScore > 1.0 {
		finalScore = 1.0
	}

	return finalScore
}
