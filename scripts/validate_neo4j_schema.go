package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7688"
	}
	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		log.Fatal("NEO4J_PASSWORD environment variable must be set")
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth("neo4j", password, ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close(context.Background())

	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Query 1: Node counts by type
	fmt.Println("=== Node Counts for repo_id=18 ===")
	result, err := session.Run(ctx,
		`MATCH (n) WHERE n.repo_id = 18 
		 RETURN labels(n)[0] as node_type, count(*) as count 
		 ORDER BY count DESC`,
		nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for result.Next(ctx) {
		record := result.Record()
		nodeType, _ := record.Get("node_type")
		count, _ := record.Get("count")
		fmt.Printf("  %s: %v\n", nodeType, count)
	}

	// Query 2: CodeBlock properties
	fmt.Println("\n=== CodeBlock Property Coverage ===")
	result2, err := session.Run(ctx,
		`MATCH (cb:CodeBlock) WHERE cb.repo_id = 18 
		 RETURN count(cb) as total_blocks, 
		        count(cb.signature) as with_signature,
		        count(cb.familiarity_map_json) as with_familiarity_json,
		        count(cb.historical_block_names) as with_historical_names,
		        count(cb.topological_index) as with_topo_index
		 LIMIT 1`,
		nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	if result2.Next(ctx) {
		record := result2.Record()
		total, _ := record.Get("total_blocks")
		sig, _ := record.Get("with_signature")
		fam, _ := record.Get("with_familiarity_json")
		hist, _ := record.Get("with_historical_names")
		topo, _ := record.Get("with_topo_index")
		fmt.Printf("  Total blocks: %v\n", total)
		fmt.Printf("  With signature: %v\n", sig)
		fmt.Printf("  With familiarity_map_json: %v\n", fam)
		fmt.Printf("  With historical_block_names: %v\n", hist)
		fmt.Printf("  With topological_index: %v\n", topo)
	}

	// Query 3: Sample familiarity_map_json values
	fmt.Println("\n=== Sample familiarity_map_json Values ===")
	result3, err := session.Run(ctx,
		`MATCH (cb:CodeBlock {repo_id: 18}) 
		 WHERE cb.familiarity_map_json IS NOT NULL 
		 RETURN cb.block_name, cb.familiarity_map_json 
		 LIMIT 3`,
		nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for result3.Next(ctx) {
		record := result3.Record()
		name, _ := record.Get("cb.block_name")
		fam, _ := record.Get("cb.familiarity_map_json")
		fmt.Printf("  %s: %v\n", name, fam)
	}

	// Query 4: Edge counts
	fmt.Println("\n=== Edge Counts for repo_id=18 ===")
	result4, err := session.Run(ctx,
		`MATCH (n)-[r]->() 
		 WHERE n.repo_id = 18 
		 RETURN type(r) as edge_type, count(*) as count 
		 ORDER BY count DESC`,
		nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for result4.Next(ctx) {
		record := result4.Record()
		edgeType, _ := record.Get("edge_type")
		count, _ := record.Get("count")
		fmt.Printf("  %s: %v\n", edgeType, count)
	}

	// Query 5: Commit topological_index coverage
	fmt.Println("\n=== Commit Topological Index Coverage ===")
	result5, err := session.Run(ctx,
		`MATCH (c:Commit) WHERE c.repo_id = 18 
		 RETURN count(c) as total_commits, 
		        count(c.topological_index) as with_topo_index
		 LIMIT 1`,
		nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	if result5.Next(ctx) {
		record := result5.Record()
		total, _ := record.Get("total_commits")
		topo, _ := record.Get("with_topo_index")
		fmt.Printf("  Total commits: %v\n", total)
		fmt.Printf("  With topological_index: %v\n", topo)
	}

	fmt.Println("\n=== Validation Complete ===")
}
