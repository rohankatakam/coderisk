package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckpointStore manages investigation checkpoint persistence using PostgreSQL
// Enables pause/resume workflows for directive agent investigations
type CheckpointStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewCheckpointStore creates a new checkpoint store using an existing PostgreSQL connection pool
func NewCheckpointStore(pool *pgxpool.Pool) *CheckpointStore {
	logger := slog.Default().With("component", "checkpoint_store")
	return &CheckpointStore{
		pool:   pool,
		logger: logger,
	}
}

// NewCheckpointStoreFromParams creates a new checkpoint store with connection parameters
func NewCheckpointStoreFromParams(ctx context.Context, host string, port int, database, user, password string) (*CheckpointStore, error) {
	if host == "" || database == "" || user == "" {
		return nil, fmt.Errorf("postgres credentials missing: host=%s, db=%s, user=%s", host, database, user)
	}

	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		host, port, database, user, password,
	)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to postgres at %s:%d: %w", host, port, err)
	}

	logger := slog.Default().With("component", "checkpoint_store")
	logger.Info("checkpoint store connected", "host", host, "port", port, "database", database)

	return &CheckpointStore{
		pool:   pool,
		logger: logger,
	}, nil
}

// Save persists a directive investigation to PostgreSQL
func (s *CheckpointStore) Save(ctx context.Context, inv *DirectiveInvestigation) error {
	// Serialize state data
	stateData, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("failed to marshal investigation: %w", err)
	}

	// Serialize file paths
	filePaths, err := json.Marshal(inv.FilePaths)
	if err != nil {
		return fmt.Errorf("failed to marshal file paths: %w", err)
	}

	// Serialize Phase 1 results
	var phase1Results []byte
	if inv.Phase1Results != nil {
		phase1Results, err = json.Marshal(inv.Phase1Results)
		if err != nil {
			return fmt.Errorf("failed to marshal phase1 results: %w", err)
		}
	}

	// Serialize decisions
	decisions, err := json.Marshal(inv.Decisions)
	if err != nil {
		return fmt.Errorf("failed to marshal decisions: %w", err)
	}

	// Serialize resume data
	var resumeData []byte
	if inv.ResumeData != nil {
		resumeData, err = json.Marshal(inv.ResumeData)
		if err != nil {
			return fmt.Errorf("failed to marshal resume data: %w", err)
		}
	}

	// Prepare terminal state (nullable)
	var terminalStateStr *string
	if inv.TerminalState != nil {
		ts := string(*inv.TerminalState)
		terminalStateStr = &ts
	}

	// Insert or update
	query := `
		INSERT INTO investigations (
			id, phase, terminal_state, created_at, updated_at,
			file_paths, repository_id, phase1_results,
			state_data, decisions, can_resume, resume_data,
			final_risk_level, final_confidence, final_summary
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15
		)
		ON CONFLICT (id) DO UPDATE SET
			phase = EXCLUDED.phase,
			terminal_state = EXCLUDED.terminal_state,
			updated_at = EXCLUDED.updated_at,
			file_paths = EXCLUDED.file_paths,
			repository_id = EXCLUDED.repository_id,
			phase1_results = EXCLUDED.phase1_results,
			state_data = EXCLUDED.state_data,
			decisions = EXCLUDED.decisions,
			can_resume = EXCLUDED.can_resume,
			resume_data = EXCLUDED.resume_data,
			final_risk_level = EXCLUDED.final_risk_level,
			final_confidence = EXCLUDED.final_confidence,
			final_summary = EXCLUDED.final_summary
	`

	_, err = s.pool.Exec(ctx, query,
		inv.ID,
		inv.Phase,
		terminalStateStr,
		inv.CreatedAt,
		inv.UpdatedAt,
		filePaths,
		nullableInt64(inv.RepositoryID),
		phase1Results,
		stateData,
		decisions,
		inv.CanResume,
		resumeData,
		nullableString(inv.FinalRiskLevel),
		nullableFloat64(inv.FinalConfidence),
		nullableString(inv.FinalSummary),
	)

	if err != nil {
		return fmt.Errorf("failed to save investigation: %w", err)
	}

	s.logger.Debug("investigation checkpoint saved", "id", inv.ID, "phase", inv.Phase)
	return nil
}

// Load retrieves a directive investigation from PostgreSQL by ID
func (s *CheckpointStore) Load(ctx context.Context, id string) (*DirectiveInvestigation, error) {
	query := `
		SELECT
			id, phase, terminal_state, created_at, updated_at,
			file_paths, repository_id, phase1_results,
			state_data, decisions, can_resume, resume_data,
			final_risk_level, final_confidence, final_summary
		FROM investigations
		WHERE id = $1
	`

	var inv DirectiveInvestigation
	var stateDataJSON []byte
	var filePathsJSON []byte
	var phase1ResultsJSON []byte
	var decisionsJSON []byte
	var resumeDataJSON []byte
	var terminalStateStr *string
	var repoID *int64
	var finalRiskLevel *string
	var finalConfidence *float64
	var finalSummary *string

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID,
		&inv.Phase,
		&terminalStateStr,
		&inv.CreatedAt,
		&inv.UpdatedAt,
		&filePathsJSON,
		&repoID,
		&phase1ResultsJSON,
		&stateDataJSON,
		&decisionsJSON,
		&inv.CanResume,
		&resumeDataJSON,
		&finalRiskLevel,
		&finalConfidence,
		&finalSummary,
	)

	if err != nil {
		return nil, fmt.Errorf("investigation not found: %w", err)
	}

	// Deserialize file paths
	if err := json.Unmarshal(filePathsJSON, &inv.FilePaths); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file paths: %w", err)
	}

	// Deserialize Phase 1 results
	if phase1ResultsJSON != nil {
		var phase1 Phase1Result
		if err := json.Unmarshal(phase1ResultsJSON, &phase1); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phase1 results: %w", err)
		}
		inv.Phase1Results = &phase1
	}

	// Deserialize decisions
	if err := json.Unmarshal(decisionsJSON, &inv.Decisions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decisions: %w", err)
	}

	// Deserialize resume data
	if resumeDataJSON != nil {
		if err := json.Unmarshal(resumeDataJSON, &inv.ResumeData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resume data: %w", err)
		}
	} else {
		inv.ResumeData = make(map[string]interface{})
	}

	// Set optional fields
	if terminalStateStr != nil {
		ts := TerminalState(*terminalStateStr)
		inv.TerminalState = &ts
	}

	if repoID != nil {
		inv.RepositoryID = *repoID
	}

	if finalRiskLevel != nil {
		inv.FinalRiskLevel = *finalRiskLevel
	}

	if finalConfidence != nil {
		inv.FinalConfidence = *finalConfidence
	}

	if finalSummary != nil {
		inv.FinalSummary = *finalSummary
	}

	s.logger.Debug("investigation checkpoint loaded", "id", inv.ID, "phase", inv.Phase)
	return &inv, nil
}

// ListResumable lists all directive investigations that can be resumed
func (s *CheckpointStore) ListResumable(ctx context.Context, limit int) ([]DirectiveInvestigation, error) {
	query := `
		SELECT
			id, phase, created_at, updated_at, file_paths
		FROM investigations
		WHERE can_resume = true
		ORDER BY updated_at DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list resumable investigations: %w", err)
	}
	defer rows.Close()

	var investigations []DirectiveInvestigation
	for rows.Next() {
		var inv DirectiveInvestigation
		var filePathsJSON []byte

		err := rows.Scan(
			&inv.ID,
			&inv.Phase,
			&inv.CreatedAt,
			&inv.UpdatedAt,
			&filePathsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan investigation: %w", err)
		}

		// Deserialize file paths
		if err := json.Unmarshal(filePathsJSON, &inv.FilePaths); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file paths: %w", err)
		}

		investigations = append(investigations, inv)
	}

	return investigations, nil
}

// Delete removes an investigation from storage
func (s *CheckpointStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM investigations WHERE id = $1`

	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete investigation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("investigation not found: %s", id)
	}

	s.logger.Debug("investigation deleted", "id", id)
	return nil
}

// Close closes the checkpoint store (if it owns the connection pool)
func (s *CheckpointStore) Close() {
	// Only close if we created the pool (NewCheckpointStoreFromParams)
	// If using shared pool (NewCheckpointStore), don't close it
	// For now, we'll let the caller manage the pool lifecycle
	s.logger.Debug("checkpoint store closed")
}

// Helper functions for nullable types

func nullableInt64(val int64) *int64 {
	if val == 0 {
		return nil
	}
	return &val
}

func nullableString(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

func nullableFloat64(val float64) *float64 {
	if val == 0 {
		return nil
	}
	return &val
}
