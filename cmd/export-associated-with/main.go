package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	// Connect to Neo4j
	backend, err := graph.NewNeo4jBackend(
		ctx,
		"bolt://localhost:7688",
		"neo4j",
		"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
		"neo4j",
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer backend.Close(ctx)

	// Open output file
	outputFile := "/Users/rohankatakam/Documents/brain/coderisk/ASSOCIATED_WITH_EDGES_ANALYSIS.md"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// Write header
	fmt.Fprintf(f, "# ASSOCIATED_WITH Edges Analysis - Complete Export\n\n")
	fmt.Fprintf(f, "**Repository:** omnara-ai/omnara (repo_id=1)\n")
	fmt.Fprintf(f, "**Generated:** %s\n\n", "2025-11-13")
	fmt.Fprintf(f, "This document contains all 245 ASSOCIATED_WITH edges with full metadata and node details.\n\n")
	fmt.Fprintf(f, "---\n\n")

	// Query all ASSOCIATED_WITH edges with full node and edge details
	results, err := backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN
			labels(source)[0] as source_type,
			source.repo_id as source_repo_id,
			source.number as source_number,
			source.title as source_title,
			source.state as source_state,
			source.created_at as source_created_at,
			source.closed_at as source_closed_at,
			c.repo_id as commit_repo_id,
			c.sha as commit_sha,
			c.message as commit_message,
			c.author_name as commit_author_name,
			c.author_email as commit_author_email,
			c.committed_at as commit_timestamp,
			r.relationship_type as rel_type,
			r.confidence as confidence,
			r.detected_via as detected_via,
			r.rationale as rationale,
			r.evidence as evidence
		ORDER BY source.number ASC, c.sha ASC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Found %d ASSOCIATED_WITH edges\n", len(results))

	// Write summary statistics
	fmt.Fprintf(f, "## Summary Statistics\n\n")
	fmt.Fprintf(f, "**Total Edges:** %d\n\n", len(results))

	// Count by source type
	prCount := 0
	issueCount := 0
	for _, row := range results {
		if row["source_type"] == "PR" {
			prCount++
		} else {
			issueCount++
		}
	}
	fmt.Fprintf(f, "- **PR → Commit:** %d edges\n", prCount)
	fmt.Fprintf(f, "- **Issue → Commit:** %d edges\n\n", issueCount)

	// Count by detection method
	detectionCounts := make(map[string]int)
	for _, row := range results {
		method := fmt.Sprintf("%v", row["detected_via"])
		detectionCounts[method]++
	}
	fmt.Fprintf(f, "### By Detection Method:\n")
	for method, count := range detectionCounts {
		fmt.Fprintf(f, "- **%s:** %d edges\n", method, count)
	}
	fmt.Fprintf(f, "\n")

	// Count by relationship type
	relCounts := make(map[string]int)
	for _, row := range results {
		relType := fmt.Sprintf("%v", row["rel_type"])
		relCounts[relType]++
	}
	fmt.Fprintf(f, "### By Relationship Type:\n")
	for relType, count := range relCounts {
		fmt.Fprintf(f, "- **%s:** %d edges\n", relType, count)
	}
	fmt.Fprintf(f, "\n")

	// Count by confidence
	confidenceCounts := make(map[float64]int)
	for _, row := range results {
		conf := row["confidence"].(float64)
		confidenceCounts[conf]++
	}
	fmt.Fprintf(f, "### By Confidence Score:\n")
	confidenceScores := []float64{0.9, 0.8, 0.7, 0.6, 0.5}
	for _, score := range confidenceScores {
		if count, exists := confidenceCounts[score]; exists {
			fmt.Fprintf(f, "- **%.1f:** %d edges\n", score, count)
		}
	}
	fmt.Fprintf(f, "\n---\n\n")

	// Write all edges with full details
	fmt.Fprintf(f, "## Complete Edge List\n\n")

	for i, row := range results {
		fmt.Fprintf(f, "### Edge %d\n\n", i+1)

		// Source node details
		sourceType := row["source_type"]
		sourceNum := row["source_number"]
		sourceTitle := truncateString(fmt.Sprintf("%v", row["source_title"]), 100)
		sourceState := row["source_state"]

		fmt.Fprintf(f, "**Source Node:** %s #%v\n", sourceType, sourceNum)
		fmt.Fprintf(f, "- **Title:** %s\n", sourceTitle)
		fmt.Fprintf(f, "- **State:** %s\n", sourceState)
		fmt.Fprintf(f, "- **repo_id:** %v\n", row["source_repo_id"])

		if row["source_created_at"] != nil {
			fmt.Fprintf(f, "- **Created:** %v\n", formatTimestamp(row["source_created_at"]))
		}
		if row["source_closed_at"] != nil {
			fmt.Fprintf(f, "- **Closed:** %v\n", formatTimestamp(row["source_closed_at"]))
		}
		fmt.Fprintf(f, "\n")

		// Edge details
		relType := row["rel_type"]
		confidence := row["confidence"]
		detectedVia := row["detected_via"]
		rationale := row["rationale"]

		fmt.Fprintf(f, "**Edge Metadata:**\n")
		fmt.Fprintf(f, "```\n")
		fmt.Fprintf(f, "relationship_type: %v\n", relType)
		fmt.Fprintf(f, "confidence: %v\n", confidence)
		fmt.Fprintf(f, "detected_via: %v\n", detectedVia)
		fmt.Fprintf(f, "rationale: %v\n", rationale)
		fmt.Fprintf(f, "evidence: %v\n", row["evidence"])
		fmt.Fprintf(f, "```\n\n")

		// Target node details
		commitSha := fmt.Sprintf("%v", row["commit_sha"])
		commitMsg := truncateString(fmt.Sprintf("%v", row["commit_message"]), 150)
		authorName := row["commit_author_name"]
		authorEmail := row["commit_author_email"]

		fmt.Fprintf(f, "**Target Node:** Commit %s\n", commitSha[:7])
		fmt.Fprintf(f, "- **Full SHA:** `%s`\n", commitSha)
		fmt.Fprintf(f, "- **Message:** %s\n", commitMsg)
		fmt.Fprintf(f, "- **Author:** %v <%v>\n", authorName, authorEmail)
		fmt.Fprintf(f, "- **repo_id:** %v\n", row["commit_repo_id"])
		if row["commit_timestamp"] != nil {
			fmt.Fprintf(f, "- **Timestamp:** %v\n", formatTimestamp(row["commit_timestamp"]))
		}
		fmt.Fprintf(f, "\n")

		// Visual representation
		fmt.Fprintf(f, "**Relationship:**\n")
		fmt.Fprintf(f, "```\n")
		fmt.Fprintf(f, "(%s #%v)-[:ASSOCIATED_WITH {%s, conf:%.1f}]->(%s)\n",
			sourceType, sourceNum, relType, confidence, commitSha[:7])
		fmt.Fprintf(f, "```\n\n")

		fmt.Fprintf(f, "---\n\n")
	}

	fmt.Printf("✅ Analysis written to: %s\n", outputFile)
}

func truncateString(s string, maxLen int) string {
	// Replace newlines with spaces for readability
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	s = strings.TrimSpace(s)

	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func formatTimestamp(val interface{}) string {
	if val == nil {
		return "null"
	}

	// Neo4j returns timestamps as int64 (Unix epoch)
	if timestamp, ok := val.(int64); ok {
		// Convert to readable format
		return fmt.Sprintf("%d", timestamp)
	}

	return fmt.Sprintf("%v", val)
}
