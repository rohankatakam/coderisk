package diffanalyzer

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GraphMatcher matches diff blocks to Neo4j CodeBlock nodes with 3-tier fallback
type GraphMatcher struct {
	driver neo4j.DriverWithContext
	logger *log.Logger
}

// NewGraphMatcher creates a new graph matcher
func NewGraphMatcher(driver neo4j.DriverWithContext, logger *log.Logger) *GraphMatcher {
	return &GraphMatcher{
		driver: driver,
		logger: logger,
	}
}

// MatchBlock matches a diff block to a Neo4j CodeBlock node using 3-tier strategy:
// Tier 1: Exact match (block_name + canonical_file_path + signature)
// Tier 2: Fuzzy match by signature (handles renames)
// Tier 3: Historical name search (historical_block_names)
func (m *GraphMatcher) MatchBlock(ctx context.Context, repoID int64, canonicalPath, blockName, signature string) (*CodeBlockMatch, error) {
	m.logger.Printf("[GraphMatcher] Matching block: repo_id=%d, path=%s, name=%s, sig=%s",
		repoID, canonicalPath, blockName, signature)

	// Tier 1: Exact match
	match, err := m.queryExactMatch(ctx, repoID, canonicalPath, blockName, signature)
	if err != nil {
		m.logger.Printf("[GraphMatcher] WARN: Exact match query failed: %v", err)
	}
	if match != nil {
		m.logger.Printf("[GraphMatcher] SUCCESS: Exact match found - block_id=%d", match.ID)
		return match, nil
	}

	// Tier 2: Fuzzy match by signature (renamed function)
	if signature != "" {
		match, err = m.queryBySignature(ctx, repoID, canonicalPath, signature)
		if err != nil {
			m.logger.Printf("[GraphMatcher] WARN: Signature match query failed: %v", err)
		}
		if match != nil {
			match.MatchType = "fuzzy_signature"
			match.Confidence = "medium"
			m.logger.Printf("[GraphMatcher] SUCCESS: Fuzzy signature match - block_id=%d, actual_name=%s",
				match.ID, match.BlockName)
			return match, nil
		}
	}

	// Tier 3: Historical name search
	match, err = m.queryByHistoricalName(ctx, repoID, canonicalPath, blockName)
	if err != nil {
		m.logger.Printf("[GraphMatcher] WARN: Historical name query failed: %v", err)
	}
	if match != nil {
		match.MatchType = "historical_name"
		match.Confidence = "high"
		m.logger.Printf("[GraphMatcher] SUCCESS: Historical name match - block_id=%d, current_name=%s",
			match.ID, match.BlockName)
		return match, nil
	}

	// Not found - new function or not indexed yet
	m.logger.Printf("[GraphMatcher] NO MATCH: Function not found in graph (new or not indexed)")
	return &CodeBlockMatch{
		BlockName:  blockName,
		Signature:  signature,
		MatchType:  "new_function",
		Confidence: "low",
	}, nil
}

// queryExactMatch performs Tier 1 exact match query
func (m *GraphMatcher) queryExactMatch(ctx context.Context, repoID int64, canonicalPath, blockName, signature string) (*CodeBlockMatch, error) {
	session := m.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock {
			block_name: $name,
			canonical_file_path: $path,
			signature: $sig,
			repo_id: $repoId
		})
		RETURN cb.id AS id,
		       cb.block_name AS block_name,
		       cb.historical_block_names AS historical_names,
		       cb.signature AS signature
		LIMIT 1
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"name":   blockName,
		"path":   canonicalPath,
		"sig":    signature,
		"repoId": repoID,
	})
	if err != nil {
		return nil, fmt.Errorf("exact match query failed: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		match := &CodeBlockMatch{
			ID:        record.Values[0].(int64),
			BlockName: record.Values[1].(string),
			Signature: record.Values[3].(string),
			MatchType: "exact",
			Confidence: "high",
		}

		// Extract historical_block_names if present
		if histNames, ok := record.Values[2].([]interface{}); ok {
			for _, name := range histNames {
				if nameStr, ok := name.(string); ok {
					match.HistoricalBlockNames = append(match.HistoricalBlockNames, nameStr)
				}
			}
		}

		return match, nil
	}

	return nil, nil
}

// queryBySignature performs Tier 2 fuzzy match by signature
func (m *GraphMatcher) queryBySignature(ctx context.Context, repoID int64, canonicalPath, signature string) (*CodeBlockMatch, error) {
	session := m.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock {
			canonical_file_path: $path,
			signature: $sig,
			repo_id: $repoId
		})
		RETURN cb.id AS id,
		       cb.block_name AS block_name,
		       cb.historical_block_names AS historical_names,
		       cb.signature AS signature
		LIMIT 1
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"path":   canonicalPath,
		"sig":    signature,
		"repoId": repoID,
	})
	if err != nil {
		return nil, fmt.Errorf("signature match query failed: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		match := &CodeBlockMatch{
			ID:        record.Values[0].(int64),
			BlockName: record.Values[1].(string),
			Signature: record.Values[3].(string),
		}

		// Extract historical_block_names
		if histNames, ok := record.Values[2].([]interface{}); ok {
			for _, name := range histNames {
				if nameStr, ok := name.(string); ok {
					match.HistoricalBlockNames = append(match.HistoricalBlockNames, nameStr)
				}
			}
		}

		return match, nil
	}

	return nil, nil
}

// queryByHistoricalName performs Tier 3 historical name search
func (m *GraphMatcher) queryByHistoricalName(ctx context.Context, repoID int64, canonicalPath, blockName string) (*CodeBlockMatch, error) {
	session := m.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (cb:CodeBlock {
			canonical_file_path: $path,
			repo_id: $repoId
		})
		WHERE $name IN cb.historical_block_names
		RETURN cb.id AS id,
		       cb.block_name AS block_name,
		       cb.historical_block_names AS historical_names,
		       cb.signature AS signature
		LIMIT 1
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"path":   canonicalPath,
		"name":   blockName,
		"repoId": repoID,
	})
	if err != nil {
		return nil, fmt.Errorf("historical name query failed: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		match := &CodeBlockMatch{
			ID:        record.Values[0].(int64),
			BlockName: record.Values[1].(string),
			Signature: record.Values[3].(string),
		}

		// Extract historical_block_names
		if histNames, ok := record.Values[2].([]interface{}); ok {
			for _, name := range histNames {
				if nameStr, ok := name.(string); ok {
					match.HistoricalBlockNames = append(match.HistoricalBlockNames, nameStr)
				}
			}
		}

		return match, nil
	}

	return nil, nil
}
