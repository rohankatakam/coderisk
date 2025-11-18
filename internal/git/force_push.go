package git

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// ForcePushDetector detects repository rewrites (force-pushes)
type ForcePushDetector struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewForcePushDetector creates a new force-push detector
func NewForcePushDetector(db *sql.DB) *ForcePushDetector {
	return &ForcePushDetector{
		db:     db,
		logger: slog.Default().With("component", "force_push_detector"),
	}
}

// DetectionResult contains force-push detection results
type DetectionResult struct {
	ForcePushDetected bool
	StoredHash        string
	CurrentHash       string
	Action            string // "none", "recompute", "re_atomize"
}

// CheckForForcePush detects if repository was force-pushed
// Compares stored parent_shas_hash with current repository state
func (fpd *ForcePushDetector) CheckForForcePush(ctx context.Context, repoID int64, repoPath string) (*DetectionResult, error) {
	// Step 1: Get stored parent hash from database
	var storedHash sql.NullString
	err := fpd.db.QueryRowContext(ctx,
		"SELECT parent_shas_hash FROM github_repositories WHERE id = $1",
		repoID,
	).Scan(&storedHash)

	if err != nil {
		return nil, fmt.Errorf("failed to query stored parent hash: %w", err)
	}

	// Step 2: Compute current parent hash from repository
	sorter := NewTopologicalSorter(repoPath)
	currentHash, err := sorter.ComputeParentSHAsHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compute current parent hash: %w", err)
	}

	// Step 3: Compare hashes
	if !storedHash.Valid || storedHash.String == "" {
		// First ingestion, no stored hash yet
		fpd.logger.Info("no stored parent hash, first ingestion",
			"repo_id", repoID,
			"current_hash", currentHash[:16],
		)

		// Store the hash for future comparisons
		_, err := fpd.db.ExecContext(ctx,
			"UPDATE github_repositories SET parent_shas_hash = $1 WHERE id = $2",
			currentHash, repoID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to store parent hash: %w", err)
		}

		return &DetectionResult{
			ForcePushDetected: false,
			StoredHash:        "",
			CurrentHash:       currentHash,
			Action:            "none",
		}, nil
	}

	// Step 4: Check for mismatch (force-push)
	if storedHash.String != currentHash {
		fpd.logger.Warn("FORCE-PUSH DETECTED",
			"repo_id", repoID,
			"stored_hash", storedHash.String[:16],
			"current_hash", currentHash[:16],
		)

		return &DetectionResult{
			ForcePushDetected: true,
			StoredHash:        storedHash.String,
			CurrentHash:       currentHash,
			Action:            "re_atomize", // Requires full re-atomization
		}, nil
	}

	// No force-push detected
	return &DetectionResult{
		ForcePushDetected: false,
		StoredHash:        storedHash.String,
		CurrentHash:       currentHash,
		Action:            "none",
	}, nil
}

// TriggerReAtomization clears semantic data and marks repo for re-atomization
// Called when force-push is detected
func (fpd *ForcePushDetector) TriggerReAtomization(ctx context.Context, repoID int64, newHash string) error {
	fpd.logger.Info("triggering full re-atomization",
		"repo_id", repoID,
		"new_hash", newHash[:16],
	)

	// Begin transaction
	tx, err := fpd.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Clear semantic data (CodeBlocks and derived data)
	tables := []string{
		"code_block_incidents",
		"code_block_coupling",
		"code_block_imports",
		"code_block_changes",
		"code_blocks",
	}

	for _, table := range tables {
		result, err := tx.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE repo_id = $1", table),
			repoID,
		)
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}

		rows, _ := result.RowsAffected()
		if rows > 0 {
			fpd.logger.Info("cleared semantic data",
				"table", table,
				"rows_deleted", rows,
			)
		}
	}

	// Step 2: Update parent hash
	_, err = tx.ExecContext(ctx,
		"UPDATE github_repositories SET parent_shas_hash = $1 WHERE id = $2",
		newHash, repoID,
	)
	if err != nil {
		return fmt.Errorf("failed to update parent hash: %w", err)
	}

	// Step 3: Mark repository for re-atomization
	_, err = tx.ExecContext(ctx,
		"UPDATE github_repositories SET ingestion_status = 'force_push_detected' WHERE id = $1",
		repoID,
	)
	if err != nil {
		return fmt.Errorf("failed to update ingestion status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fpd.logger.Info("re-atomization trigger completed",
		"repo_id", repoID,
		"status", "force_push_detected",
	)

	return nil
}

// Note: ComputeParentSHAsHash is already defined in topological.go
// This file only contains force-push detection logic that uses that method
