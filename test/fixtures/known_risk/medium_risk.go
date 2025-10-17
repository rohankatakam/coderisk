package fixtures

import (
	"context"

	"github.com/rohankatakam/coderisk/internal/models"
)

// MediumRiskFunction has moderate coupling, some tests
// Expected risk: MEDIUM
// - Coupling: 2 imports (LOW-MEDIUM boundary)
// - Co-change: No pattern detected (LOW)
// - Test ratio: ~0.4 (1 test function vs 1 source function, similar LOC) (MEDIUM)
func MediumRiskFunction(ctx context.Context) (*models.FileRisk, error) {
	// Couples to 2 packages (context, models)
	// Has tests but not comprehensive
	return &models.FileRisk{
		Path:      "medium_risk.go",
		Language:  "go",
		RiskScore: 5.0,
	}, nil
}
