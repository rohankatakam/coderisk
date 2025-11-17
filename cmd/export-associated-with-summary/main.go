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
	outputFile := "/Users/rohankatakam/Documents/brain/coderisk/ASSOCIATED_WITH_EDGES_SUMMARY.txt"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// Write header
	fmt.Fprintf(f, "ASSOCIATED_WITH EDGES - COMPACT SUMMARY VIEW\n")
	fmt.Fprintf(f, "Repository: omnara-ai/omnara (repo_id=1)\n")
	fmt.Fprintf(f, "%s\n\n", strings.Repeat("=", 100))

	// Query all ASSOCIATED_WITH edges
	results, err := backend.QueryWithParams(ctx, `
		MATCH (source)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.repo_id = 1
		RETURN
			labels(source)[0] as source_type,
			source.number as source_number,
			source.title as source_title,
			c.sha as commit_sha,
			c.message as commit_message,
			r.relationship_type as rel_type,
			r.confidence as confidence,
			r.detected_via as detected_via,
			r.rationale as rationale
		ORDER BY source.number ASC, c.sha ASC
	`, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Found %d ASSOCIATED_WITH edges\n", len(results))

	// Group by detection method
	byMethod := make(map[string][]map[string]interface{})
	for _, row := range results {
		method := fmt.Sprintf("%v", row["detected_via"])
		byMethod[method] = append(byMethod[method], row)
	}

	// Write by detection method
	for _, method := range []string{"commit_extraction", "pr_extraction", "temporal"} {
		edges := byMethod[method]
		if len(edges) == 0 {
			continue
		}

		fmt.Fprintf(f, "\n"+strings.Repeat("=", 100)+"\n")
		fmt.Fprintf(f, "DETECTION METHOD: %s (%d edges)\n", strings.ToUpper(method), len(edges))
		fmt.Fprintf(f, strings.Repeat("=", 100)+"\n\n")

		for i, row := range edges {
			sourceType := row["source_type"]
			sourceNum := row["source_number"]
			sourceTitle := truncate(fmt.Sprintf("%v", row["source_title"]), 50)
			commitSha := fmt.Sprintf("%v", row["commit_sha"])[:7]
			commitMsg := truncate(fmt.Sprintf("%v", row["commit_message"]), 60)
			relType := row["rel_type"]
			conf := row["confidence"]
			rationale := row["rationale"]

			fmt.Fprintf(f, "[%d] (%s #%v)-[%s, %.1f]->(%s)\n",
				i+1, sourceType, sourceNum, relType, conf, commitSha)
			fmt.Fprintf(f, "    Source: %s\n", sourceTitle)
			fmt.Fprintf(f, "    Target: %s\n", commitMsg)
			fmt.Fprintf(f, "    Why:    %v\n", rationale)
			fmt.Fprintf(f, "\n")
		}
	}

	// Write grouped analysis
	fmt.Fprintf(f, "\n"+strings.Repeat("=", 100)+"\n")
	fmt.Fprintf(f, "ANALYSIS BY RELATIONSHIP TYPE\n")
	fmt.Fprintf(f, strings.Repeat("=", 100)+"\n\n")

	byRelType := make(map[string][]map[string]interface{})
	for _, row := range results {
		relType := fmt.Sprintf("%v", row["rel_type"])
		byRelType[relType] = append(byRelType[relType], row)
	}

	for relType, edges := range byRelType {
		fmt.Fprintf(f, "\n%s (%d edges):\n", strings.ToUpper(relType), len(edges))
		fmt.Fprintf(f, strings.Repeat("-", 100)+"\n")

		// Show first 10 examples
		limit := 10
		if len(edges) < limit {
			limit = len(edges)
		}

		for i := 0; i < limit; i++ {
			row := edges[i]
			sourceType := row["source_type"]
			sourceNum := row["source_number"]
			commitSha := fmt.Sprintf("%v", row["commit_sha"])[:7]
			conf := row["confidence"]
			method := row["detected_via"]

			fmt.Fprintf(f, "  [%.1f] %s #%v -> %s (via %s)\n",
				conf, sourceType, sourceNum, commitSha, method)
		}

		if len(edges) > limit {
			fmt.Fprintf(f, "  ... and %d more\n", len(edges)-limit)
		}
	}

	// Write confidence distribution
	fmt.Fprintf(f, "\n"+strings.Repeat("=", 100)+"\n")
	fmt.Fprintf(f, "CONFIDENCE DISTRIBUTION\n")
	fmt.Fprintf(f, strings.Repeat("=", 100)+"\n\n")

	byConf := make(map[float64][]map[string]interface{})
	for _, row := range results {
		conf := row["confidence"].(float64)
		byConf[conf] = append(byConf[conf], row)
	}

	for _, conf := range []float64{0.9, 0.8, 0.7, 0.6, 0.5} {
		edges := byConf[conf]
		if len(edges) == 0 {
			continue
		}

		fmt.Fprintf(f, "\nConfidence %.1f (%d edges):\n", conf, len(edges))

		// Count by method
		methodCounts := make(map[string]int)
		typeCounts := make(map[string]int)
		for _, row := range edges {
			method := fmt.Sprintf("%v", row["detected_via"])
			relType := fmt.Sprintf("%v", row["rel_type"])
			methodCounts[method]++
			typeCounts[relType]++
		}

		fmt.Fprintf(f, "  By method: ")
		for method, count := range methodCounts {
			fmt.Fprintf(f, "%s=%d ", method, count)
		}
		fmt.Fprintf(f, "\n  By type: ")
		for relType, count := range typeCounts {
			fmt.Fprintf(f, "%s=%d ", relType, count)
		}
		fmt.Fprintf(f, "\n")
	}

	fmt.Printf("✅ Summary written to: %s\n", outputFile)
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
