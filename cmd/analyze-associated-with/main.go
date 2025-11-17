package main

import (
	"context"
	"fmt"
	"log"

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

	fmt.Println("🔍 Analyzing ASSOCIATED_WITH edges...")
	fmt.Println("================================================================================")

	// Query 1: Count by source type (Issue vs PR)
	fmt.Println("\n📊 ASSOCIATED_WITH edges by source type:")
	results, err := backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN labels(source)[0] as source_type, count(r) as count
		ORDER BY count DESC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, row := range results {
		fmt.Printf("  %s → Commit: %v edges\n", row["source_type"], row["count"])
	}

	// Query 2: Confidence score distribution
	fmt.Println("\n📈 Confidence score distribution:")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN r.confidence as confidence, count(*) as count
		ORDER BY confidence DESC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, row := range results {
		fmt.Printf("  Confidence %.1f: %v edges\n", row["confidence"], row["count"])
	}

	// Query 3: Relationship types
	fmt.Println("\n🔗 Relationship types:")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN r.relationship_type as rel_type, count(*) as count
		ORDER BY count DESC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, row := range results {
		fmt.Printf("  %s: %v edges\n", row["rel_type"], row["count"])
	}

	// Query 4: Detection methods
	fmt.Println("\n🎯 Detection methods:")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN r.detected_via as method, count(*) as count
		ORDER BY count DESC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, row := range results {
		fmt.Printf("  %s: %v edges\n", row["method"], row["count"])
	}

	// Query 5: Rationale patterns
	fmt.Println("\n📝 Rationale patterns:")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN r.rationale as rationale, count(*) as count
		ORDER BY count DESC
		LIMIT 10
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, row := range results {
		fmt.Printf("  %s: %v edges\n", row["rationale"], row["count"])
	}

	// Query 6: Sample edges with high confidence
	fmt.Println("\n✅ Sample HIGH confidence edges (0.9+):")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1 AND r.confidence >= 0.9
		RETURN labels(source)[0] as source_type,
		       source.number as source_number,
		       source.title as source_title,
		       c.sha as commit_sha,
		       c.message as commit_message,
		       r.confidence as confidence,
		       r.relationship_type as rel_type
		ORDER BY source.number ASC
		LIMIT 5
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for i, row := range results {
		fmt.Printf("\n  Example %d:\n", i+1)
		fmt.Printf("    %s #%v: %s\n", row["source_type"], row["source_number"],
			truncate(row["source_title"], 60))
		fmt.Printf("    → Commit %.7s: %s\n", row["commit_sha"],
			truncate(row["commit_message"], 60))
		fmt.Printf("    Confidence: %.1f | Type: %s\n",
			row["confidence"], row["rel_type"])
	}

	// Query 7: Sample edges with medium confidence
	fmt.Println("\n⚠️  Sample MEDIUM confidence edges (0.7):")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1 AND r.confidence = 0.7
		RETURN labels(source)[0] as source_type,
		       source.number as source_number,
		       source.title as source_title,
		       c.sha as commit_sha,
		       c.message as commit_message,
		       r.confidence as confidence,
		       r.relationship_type as rel_type
		ORDER BY source.number ASC
		LIMIT 5
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for i, row := range results {
		fmt.Printf("\n  Example %d:\n", i+1)
		fmt.Printf("    %s #%v: %s\n", row["source_type"], row["source_number"],
			truncate(row["source_title"], 60))
		fmt.Printf("    → Commit %.7s: %s\n", row["commit_sha"],
			truncate(row["commit_message"], 60))
		fmt.Printf("    Confidence: %.1f | Type: %s\n",
			row["confidence"], row["rel_type"])
	}

	// Query 8: Issue vs PR breakdown by confidence
	fmt.Println("\n📊 Confidence distribution by source type:")
	results, err = backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN labels(source)[0] as source_type,
		       r.confidence as confidence,
		       count(*) as count
		ORDER BY source_type, confidence DESC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Println("\n  Source Type | Confidence | Count")
	fmt.Println("  ------------|------------|------")
	for _, row := range results {
		fmt.Printf("  %-11s | %.1f        | %v\n",
			row["source_type"], row["confidence"], row["count"])
	}

	fmt.Println("\n================================================================================")
}

func truncate(val interface{}, maxLen int) string {
	if val == nil {
		return "(null)"
	}
	str := fmt.Sprintf("%v", val)
	if len(str) > maxLen {
		return str[:maxLen-3] + "..."
	}
	return str
}
