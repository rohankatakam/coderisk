package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// BlockWithOwnership represents a code block with ownership and risk data
type BlockWithOwnership struct {
	ID                   int64
	BlockName            string
	BlockType            string
	Signature            string
	CanonicalFilePath    string
	StartLine            int
	EndLine              int
	OriginalAuthorEmail  string
	LastModifierEmail    string
	LastModifiedDate     time.Time
	FamiliarityMap       map[string]int
	StalenessDays        int
	IncidentCount        int
	RiskScore            float64
	CoChangeCount        int
	AvgCouplingRate      float64
}

// GetFileBlocks retrieves all code blocks in a file with ownership data
func GetFileBlocks(ctx context.Context, db *sqlx.DB, repoID int64, filePath string) ([]BlockWithOwnership, error) {
	query := `
		SELECT
			cb.id,
			cb.block_name,
			cb.block_type,
			cb.signature,
			cb.canonical_file_path,
			cb.start_line,
			cb.end_line,
			COALESCE(cb.original_author_email, ''),
			COALESCE(cb.last_modifier_email, ''),
			COALESCE(cb.last_modified_date, NOW()),
			COALESCE(cb.familiarity_map, '{}'::jsonb)::text,
			EXTRACT(DAY FROM (NOW() - cb.last_modified_date))::INTEGER as staleness_days,
			COALESCE(cb.incident_count, 0),
			COALESCE(cb.risk_score, 0.0),
			COALESCE(cb.co_change_count, 0),
			COALESCE(cb.avg_coupling_rate, 0.0)
		FROM code_blocks cb
		WHERE cb.repo_id = $1
			AND cb.canonical_file_path = $2
		ORDER BY cb.start_line ASC
	`

	rows, err := db.QueryContext(ctx, query, repoID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query file blocks: %w", err)
	}
	defer rows.Close()

	var blocks []BlockWithOwnership
	for rows.Next() {
		var block BlockWithOwnership
		var familiarityMapJSON string
		var stalenessDays sql.NullInt64

		err := rows.Scan(
			&block.ID,
			&block.BlockName,
			&block.BlockType,
			&block.Signature,
			&block.CanonicalFilePath,
			&block.StartLine,
			&block.EndLine,
			&block.OriginalAuthorEmail,
			&block.LastModifierEmail,
			&block.LastModifiedDate,
			&familiarityMapJSON,
			&stalenessDays,
			&block.IncidentCount,
			&block.RiskScore,
			&block.CoChangeCount,
			&block.AvgCouplingRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		// Parse familiarity map from array format to map format
		block.FamiliarityMap = parseFamiliarityMap(familiarityMapJSON)

		// Handle null staleness
		if stalenessDays.Valid {
			block.StalenessDays = int(stalenessDays.Int64)
		}

		blocks = append(blocks, block)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file blocks: %w", err)
	}

	return blocks, nil
}

// GetBlocksInDirectory retrieves all code blocks in a directory (recursive)
func GetBlocksInDirectory(ctx context.Context, db *sqlx.DB, repoID int64, dirPath string) ([]BlockWithOwnership, error) {
	// Use LIKE pattern to match directory prefix
	pattern := dirPath + "%"

	query := `
		SELECT
			cb.id,
			cb.block_name,
			cb.block_type,
			cb.signature,
			cb.canonical_file_path,
			cb.start_line,
			cb.end_line,
			COALESCE(cb.original_author_email, ''),
			COALESCE(cb.last_modifier_email, ''),
			COALESCE(cb.last_modified_date, NOW()),
			COALESCE(cb.familiarity_map, '{}'::jsonb)::text,
			EXTRACT(DAY FROM (NOW() - cb.last_modified_date))::INTEGER as staleness_days,
			COALESCE(cb.incident_count, 0),
			COALESCE(cb.risk_score, 0.0),
			COALESCE(cb.co_change_count, 0),
			COALESCE(cb.avg_coupling_rate, 0.0)
		FROM code_blocks cb
		WHERE cb.repo_id = $1
			AND cb.canonical_file_path LIKE $2
		ORDER BY cb.canonical_file_path, cb.start_line ASC
	`

	rows, err := db.QueryContext(ctx, query, repoID, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query directory blocks: %w", err)
	}
	defer rows.Close()

	var blocks []BlockWithOwnership
	for rows.Next() {
		var block BlockWithOwnership
		var familiarityMapJSON string
		var stalenessDays sql.NullInt64

		err := rows.Scan(
			&block.ID,
			&block.BlockName,
			&block.BlockType,
			&block.Signature,
			&block.CanonicalFilePath,
			&block.StartLine,
			&block.EndLine,
			&block.OriginalAuthorEmail,
			&block.LastModifierEmail,
			&block.LastModifiedDate,
			&familiarityMapJSON,
			&stalenessDays,
			&block.IncidentCount,
			&block.RiskScore,
			&block.CoChangeCount,
			&block.AvgCouplingRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		// Parse familiarity map from array format to map format
		block.FamiliarityMap = parseFamiliarityMap(familiarityMapJSON)

		// Handle null staleness
		if stalenessDays.Valid {
			block.StalenessDays = int(stalenessDays.Int64)
		}

		blocks = append(blocks, block)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating directory blocks: %w", err)
	}

	return blocks, nil
}

// CalculateSME determines the Subject Matter Expert for a block
func CalculateSME(familiarityMap map[string]int) (sme string, busFactor string, topFamiliarity int) {
	if len(familiarityMap) == 0 {
		return "", "UNKNOWN", 0
	}

	// Find developer with highest familiarity
	maxFamiliarity := 0
	smeEmail := ""
	totalFamiliarity := 0

	for email, familiarity := range familiarityMap {
		totalFamiliarity += familiarity
		if familiarity > maxFamiliarity {
			maxFamiliarity = familiarity
			smeEmail = email
		}
	}

	// Calculate bus factor based on concentration
	var busFact string
	if totalFamiliarity > 0 {
		concentration := float64(maxFamiliarity) / float64(totalFamiliarity) * 100
		switch {
		case concentration >= 90:
			busFact = "CRITICAL"
		case concentration >= 70:
			busFact = "HIGH"
		case concentration >= 50:
			busFact = "MEDIUM"
		default:
			busFact = "LOW"
		}
	} else {
		busFact = "UNKNOWN"
	}

	return smeEmail, busFact, maxFamiliarity
}

// OwnershipStats provides aggregate statistics for directory scanning
type OwnershipStats struct {
	TotalBlocks      int
	CriticalRisk     int // risk_score > 70
	MediumRisk       int // risk_score 40-70
	LowRisk          int // risk_score < 40
	StaleBlocks      int // staleness > 90 days
	BusFactorWarnings int
	TopRiskyBlocks   []BlockWithOwnership
}

// CalculateOwnershipStats aggregates ownership statistics
func CalculateOwnershipStats(blocks []BlockWithOwnership) OwnershipStats {
	stats := OwnershipStats{
		TotalBlocks:    len(blocks),
		TopRiskyBlocks: make([]BlockWithOwnership, 0),
	}

	for _, block := range blocks {
		// Risk level categorization
		if block.RiskScore > 70 {
			stats.CriticalRisk++
		} else if block.RiskScore >= 40 {
			stats.MediumRisk++
		} else {
			stats.LowRisk++
		}

		// Staleness
		if block.StalenessDays > 90 {
			stats.StaleBlocks++
		}

		// Bus factor
		_, busFactor, _ := CalculateSME(block.FamiliarityMap)
		if busFactor == "CRITICAL" || busFactor == "HIGH" {
			stats.BusFactorWarnings++
		}
	}

	// Sort blocks by risk score and take top 10
	sortedBlocks := make([]BlockWithOwnership, len(blocks))
	copy(sortedBlocks, blocks)

	// Simple bubble sort for top 10 (good enough for small datasets)
	for i := 0; i < len(sortedBlocks) && i < 10; i++ {
		for j := i + 1; j < len(sortedBlocks); j++ {
			if sortedBlocks[j].RiskScore > sortedBlocks[i].RiskScore {
				sortedBlocks[i], sortedBlocks[j] = sortedBlocks[j], sortedBlocks[i]
			}
		}
	}

	// Take top 10
	limit := 10
	if len(sortedBlocks) < limit {
		limit = len(sortedBlocks)
	}
	stats.TopRiskyBlocks = sortedBlocks[:limit]

	return stats
}

// parseFamiliarityMap converts familiarity_map from array format to map format
// Database stores: [{"dev": "email@example.com", "edits": 5}]
// Returns: {"email@example.com": 5}
func parseFamiliarityMap(familiarityMapJSON string) map[string]int {
	result := make(map[string]int)

	// Handle empty or null cases
	if familiarityMapJSON == "" || familiarityMapJSON == "{}" || familiarityMapJSON == "[]" {
		return result
	}

	// Try to parse as array format first (expected format)
	type familiarityEntry struct {
		Dev   string `json:"dev"`
		Edits int    `json:"edits"`
	}
	var entries []familiarityEntry
	if err := json.Unmarshal([]byte(familiarityMapJSON), &entries); err == nil {
		for _, entry := range entries {
			result[entry.Dev] = entry.Edits
		}
		return result
	}

	// Fallback: try to parse as simple map format (backward compatibility)
	if err := json.Unmarshal([]byte(familiarityMapJSON), &result); err == nil {
		return result
	}

	// If all parsing fails, return empty map
	return result
}
