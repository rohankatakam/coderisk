package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	ctx := context.Background()
	
	neoURI := os.Getenv("NEO4J_URI")
	if neoURI == "" {
		neoURI = "bolt://localhost:7688"
	}
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoPassword == "" {
		neoPassword = "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
	}
	
	driver, err := neo4j.NewDriverWithContext(neoURI, neo4j.BasicAuth("neo4j", neoPassword, ""))
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close(ctx)

	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	fmt.Println("=== OWNERSHIP VERIFICATION ===\n")

	// Verification 1: Check missing ownership
	result, _ := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (b:CodeBlock {repo_id: 4})
			WHERE b.original_author IS NULL
			   OR b.last_modifier IS NULL
			   OR b.staleness_days IS NULL
			RETURN count(b) AS missing_ownership
		`, nil)
		if err != nil {
			return nil, err
		}
		rec, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}
		count, _ := rec.Get("missing_ownership")
		return count, nil
	})
	
	missingCount := int(result.(int64))
	fmt.Printf("✓ Missing ownership properties: %d (expected: 0)\n", missingCount)
	if missingCount == 0 {
		fmt.Println("✅ All blocks have complete ownership properties!\n")
	} else {
		fmt.Printf("⚠️  WARNING: %d blocks missing ownership properties\n\n", missingCount)
	}

	// Verification 2: Sample ownership data
	fmt.Println("=== SAMPLE OWNERSHIP DATA ===")
	session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (b:CodeBlock {repo_id: 4})
			WHERE b.familiarity_map IS NOT NULL
			RETURN b.name AS name, 
			       b.original_author AS original_author,
			       b.last_modifier AS last_modifier,
			       b.staleness_days AS staleness_days,
			       b.familiarity_map AS familiarity_map
			LIMIT 5
		`, nil)
		if err != nil {
			return nil, err
		}
		records, _ := res.Collect(ctx)
		for i, rec := range records {
			name, _ := rec.Get("name")
			author, _ := rec.Get("original_author")
			modifier, _ := rec.Get("last_modifier")
			staleness, _ := rec.Get("staleness_days")
			famMap, _ := rec.Get("familiarity_map")
			fmt.Printf("\n[%d] Block: %s\n", i+1, name)
			fmt.Printf("    Original Author: %s\n", author)
			fmt.Printf("    Last Modifier: %s\n", modifier)
			fmt.Printf("    Staleness (days): %v\n", staleness)
			fmt.Printf("    Familiarity Map: %s\n", famMap)
		}
		return nil, nil
	})

	// Verification 3: Statistics
	fmt.Println("\n=== OWNERSHIP STATISTICS ===")
	session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (b:CodeBlock {repo_id: 4})
			RETURN 
				count(b) AS total_blocks,
				count(b.original_author) AS with_original_author,
				count(b.last_modifier) AS with_last_modifier,
				count(b.familiarity_map) AS with_familiarity_map,
				avg(b.staleness_days) AS avg_staleness_days
		`, nil)
		if err != nil {
			return nil, err
		}
		rec, _ := res.Single(ctx)
		total, _ := rec.Get("total_blocks")
		withAuthor, _ := rec.Get("with_original_author")
		withModifier, _ := rec.Get("with_last_modifier")
		withFamMap, _ := rec.Get("with_familiarity_map")
		avgStaleness, _ := rec.Get("avg_staleness_days")
		
		fmt.Printf("Total Blocks: %d\n", total)
		fmt.Printf("With Original Author: %d\n", withAuthor)
		fmt.Printf("With Last Modifier: %d\n", withModifier)
		fmt.Printf("With Familiarity Map: %d\n", withFamMap)
		fmt.Printf("Average Staleness (days): %.1f\n", avgStaleness)
		return nil, nil
	})
	
	fmt.Println("\n✅ VERIFICATION COMPLETE")
}
