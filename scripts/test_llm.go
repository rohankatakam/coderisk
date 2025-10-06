package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/coderisk/coderisk-go/internal/agent"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run scripts/test_llm.go <provider> <prompt>")
		fmt.Println("Example: go run scripts/test_llm.go openai \"What is 2+2?\"")
		os.Exit(1)
	}

	provider := os.Args[1]
	prompt := os.Args[2]

	// Get API key from environment
	apiKey := os.Getenv("CODERISK_API_KEY")
	if apiKey == "" {
		log.Fatal("CODERISK_API_KEY environment variable not set")
	}

	// Create LLM client
	client, err := agent.NewLLMClient(provider, apiKey)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	fmt.Printf("Testing %s...\n", provider)
	fmt.Printf("Prompt: %s\n\n", prompt)

	// Query LLM
	response, tokens, err := client.Query(context.Background(), prompt)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Response:\n%s\n\n", response)
	fmt.Printf("Tokens used: %d\n", tokens)
	fmt.Println("✅ LLM client working!")
}
