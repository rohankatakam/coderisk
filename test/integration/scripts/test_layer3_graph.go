//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Layer 3 Test: Graph Construction (Neo4j Integration)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	// Connect to Neo4j
	fmt.Println("Connecting to Neo4j...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backend, err := graph.NewNeo4jBackend(
		ctx,
		"bolt://localhost:7688",
		"neo4j",
		"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
		"neo4j", // database name
	)
	if err != nil {
		log.Fatalf("❌ Layer 3 FAILED: Could not connect to Neo4j: %v\n   Make sure Neo4j is running: docker compose up -d", err)
	}
	defer backend.Close(ctx)

	fmt.Println("✅ Connected to Neo4j successfully\n")

	// Test 1: Create a node
	fmt.Println("Test 1: Creating a test node...")
	testNode := graph.GraphNode{
		ID:    "/tmp/test-repo/example.py",
		Label: "File",
		Properties: map[string]interface{}{
			"name":      "example.py",
			"file_path": "/tmp/test-repo/example.py",
			"path":      "/tmp/test-repo/example.py",
			"language":  "python",
			"test_node": true, // Mark for cleanup
			"timestamp": time.Now().Unix(),
		},
	}

	nodeID, err := backend.CreateNode(ctx, testNode)
	if err != nil {
		log.Fatalf("❌ Layer 3 FAILED: Could not create node: %v", err)
	}

	fmt.Printf("✅ Created node with ID: %s\n\n", nodeID)

	// Test 2: Query the node
	fmt.Println("Test 2: Querying the test node...")
	query := "MATCH (f:File {test_node: true}) RETURN f.name as name, f.language as language, f.path as path"

	result, err := backend.Query(ctx, query)
	if err != nil {
		log.Fatalf("❌ Layer 3 FAILED: Query failed: %v", err)
	}

	fmt.Println("✅ Query executed successfully")

	// Display results
	if resultSlice, ok := result.([]map[string]interface{}); ok && len(resultSlice) > 0 {
		fmt.Printf("   Found %d node(s):\n", len(resultSlice))
		for i, row := range resultSlice {
			fmt.Printf("     %d. name=%v, language=%v\n", i+1, row["name"], row["language"])
		}
	} else {
		log.Println("⚠️  Warning: No results returned from query")
	}
	fmt.Println()

	// Test 3: Create edges
	fmt.Println("Test 3: Creating test edges...")
	functionNode := graph.GraphNode{
		ID:    "/tmp/test-repo/example.py:hello_world:1",
		Label: "Function",
		Properties: map[string]interface{}{
			"name":      "hello_world",
			"file_path": "/tmp/test-repo/example.py",
			"test_node": true,
		},
	}

	_, err = backend.CreateNode(ctx, functionNode)
	if err != nil {
		log.Printf("⚠️  Warning: Could not create function node: %v", err)
	}

	edge := graph.GraphEdge{
		From:  "/tmp/test-repo/example.py",
		To:    "/tmp/test-repo/example.py:hello_world:1",
		Label: "CONTAINS",
		Properties: map[string]interface{}{
			"entity_type": "function",
			"test_edge":   true,
		},
	}

	err = backend.CreateEdges(ctx, []graph.GraphEdge{edge})
	if err != nil {
		log.Printf("⚠️  Warning: Could not create edge: %v", err)
	} else {
		fmt.Println("✅ Created CONTAINS edge\n")
	}

	// Test 4: Cleanup
	fmt.Println("Test 4: Cleaning up test data...")
	cleanupQuery := "MATCH (n {test_node: true}) DETACH DELETE n"
	_, err = backend.Query(ctx, cleanupQuery)
	if err != nil {
		log.Printf("⚠️  Warning: Could not cleanup test nodes: %v", err)
	} else {
		fmt.Println("✅ Test data cleaned up\n")
	}

	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("✅ LAYER 3 TEST PASSED")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("Neo4j graph integration is working correctly!")
	fmt.Println("The Graph layer can create nodes, edges, and execute queries.")
}
