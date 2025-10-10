package fixtures

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/database"
	"github.com/coderisk/coderisk-go/internal/git"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/metrics"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/output"
)

// HighRiskFunction has high coupling, NO tests, complex logic
// Expected risk: HIGH
// - Coupling: 15+ imports (HIGH - exceeds threshold of 10)
// - Co-change: No pattern (but coupling alone triggers HIGH)
// - Test ratio: 0 (no test file exists) (HIGH - below threshold of 0.3)
//
// This function intentionally violates multiple risk thresholds:
// 1. Couples to 15+ packages (structural coupling > 10)
// 2. Has NO corresponding test file (test ratio = 0 < 0.3)
// 3. Contains complex logic with network calls, database operations, and file I/O
// 4. Mixes multiple concerns (HTTP, DB, file system)
//
// This should trigger Phase 2 escalation in production.
func HighRiskFunction(ctx context.Context, db *sql.DB, apiURL string, redisClient *cache.Client) error {
	// Network call (external dependency)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	// Database write (side effect)
	query := "INSERT INTO risk_data (json_data, created_at) VALUES ($1, $2)"
	_, err = db.ExecContext(ctx, query, string(body), time.Now())
	if err != nil {
		return fmt.Errorf("database insert failed: %w", err)
	}

	// File system operations
	tempFile := fmt.Sprintf("/tmp/risk_data_%d.json", time.Now().Unix())
	if err := os.WriteFile(tempFile, body, 0644); err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	// Cache operations
	cacheKey := fmt.Sprintf("risk:%s", strings.TrimSpace(apiURL))
	if err := redisClient.Set(ctx, cacheKey, data); err != nil {
		// Non-fatal cache error
		fmt.Printf("Warning: cache write failed: %v\n", err)
	}

	// Complex processing (uses multiple internal packages)
	_ = &graph.Client{}
	_ = &database.Client{}
	_ = &metrics.Registry{}
	_ = &models.RiskResult{}
	_ = &output.QuietFormatter{}

	return nil
}
