package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/mcp"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
	bolt "go.etcd.io/bbolt"
)

func main() {
	ctx := context.Background()

	// 1. Get environment variables (CRITICAL: Use actual values from implementation)
	neo4jURI := getEnvOrDefault("NEO4J_URI", "bolt://localhost:7688")
	neo4jPassword := getEnvOrDefault("NEO4J_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")
	postgresDSN := getEnvOrDefault("POSTGRES_DSN", "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable")

	// 2. Connect to Neo4j (port 7688, not 7687!)
	driver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j at %s: %v", neo4jURI, err)
	}
	defer driver.Close(ctx)

	// Verify Neo4j connection
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Neo4j connectivity check failed: %v", err)
	}
	log.Println("✅ Connected to Neo4j")

	// 3. Connect to PostgreSQL (port 5433, user coderisk, not coderisk_user!)
	pgPool, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Verify PostgreSQL connection
	if err := pgPool.Ping(ctx); err != nil {
		log.Fatalf("PostgreSQL ping failed: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// 4. Open bbolt cache
	cacheDB, err := bolt.Open("/tmp/crisk-mcp-cache.db", 0600, nil)
	if err != nil {
		log.Fatalf("Failed to open cache: %v", err)
	}
	defer cacheDB.Close()
	log.Println("✅ Cache initialized")

	// 5. Create graph client
	graphClient := mcp.NewLocalGraphClient(driver, pgPool)

	// 6. Create identity resolver
	resolver := mcp.NewIdentityResolver(cacheDB)

	// 7. Register tools
	handler := mcp.NewHandler()
	handler.RegisterTool("crisk.get_risk_summary", tools.NewGetRiskSummaryTool(graphClient, resolver))
	log.Println("✅ Registered tool: crisk.get_risk_summary")

	// 8. Start stdio transport
	transport := mcp.NewStdioTransport(handler)

	// 9. Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	// 10. Start server
	log.Println("🚀 MCP server started on stdio")
	if err := transport.Start(); err != nil {
		log.Fatalf("Transport error: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
