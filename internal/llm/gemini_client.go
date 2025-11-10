package llm

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/genai"
)

// GeminiClient wraps Google's Generative AI SDK
// Supports Gemini models for directive agent workflows
type GeminiClient struct {
	client *genai.Client
	model  string
	logger *slog.Logger
}

// NewGeminiClient creates a new Gemini API client
// apiKey: Google AI API key (from environment or config)
// model: Model name (e.g., "gemini-2.0-flash-exp", "gemini-1.5-pro")
func NewGeminiClient(ctx context.Context, apiKey, model string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}

	if model == "" {
		model = "gemini-2.0-flash" // Default to production flash model with higher rate limits
	}

	// Create client config with API key
	clientConfig := &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	logger := slog.Default().With("component", "gemini", "model", model)
	logger.Info("gemini client initialized")

	return &GeminiClient{
		client: client,
		model:  model,
		logger: logger,
	}, nil
}

// Complete sends a prompt to Gemini and returns the text response
// Uses standard text generation with system instruction support
func (c *GeminiClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Build system instruction if provided
	var systemInstruction *genai.Content
	if systemPrompt != "" {
		systemInstruction = genai.Text(systemPrompt)[0]
	}

	// Create generation config
	genConfig := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       ptrFloat32(0.1), // Low temperature for consistency
		MaxOutputTokens:   2000,
	}

	// Generate content
	resp, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(userPrompt), genConfig)
	if err != nil {
		return "", fmt.Errorf("gemini completion failed: %w", err)
	}

	// Validate response
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content parts")
	}

	// Extract text from first part
	part := candidate.Content.Parts[0]
	text := part.Text

	c.logger.Debug("gemini completion",
		"prompt_length", len(userPrompt),
		"response_length", len(text),
	)

	return text, nil
}

// CompleteJSON sends a prompt to Gemini and requests JSON response
// Uses Gemini's native JSON mode with MIME type specification
func (c *GeminiClient) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Build system instruction if provided
	var systemInstruction *genai.Content
	if systemPrompt != "" {
		systemInstruction = genai.Text(systemPrompt)[0]
	}

	// Create generation config with JSON output
	genConfig := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       ptrFloat32(0.1),
		ResponseMIMEType:  "application/json",
	}

	// Generate content
	resp, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(userPrompt), genConfig)
	if err != nil {
		return "", fmt.Errorf("gemini json completion failed: %w", err)
	}

	// Validate response
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content parts")
	}

	// Extract JSON text
	part := candidate.Content.Parts[0]
	jsonText := part.Text

	c.logger.Debug("gemini json completion",
		"prompt_length", len(userPrompt),
		"response_length", len(jsonText),
	)

	return jsonText, nil
}

// CompleteWithTools sends a prompt with function calling tools enabled
// Returns the full response which may include tool calls
// Tools should be defined using Gemini's tool/function declaration format
func (c *GeminiClient) CompleteWithTools(ctx context.Context, systemPrompt, userPrompt string, tools []*genai.Tool) (*genai.GenerateContentResponse, error) {
	// Build system instruction if provided
	var systemInstruction *genai.Content
	if systemPrompt != "" {
		systemInstruction = genai.Text(systemPrompt)[0]
	}

	// Create generation config with tools
	genConfig := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       ptrFloat32(0.1),
		Tools:             tools,
	}

	// Generate content
	resp, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(userPrompt), genConfig)
	if err != nil {
		return nil, fmt.Errorf("gemini tool completion failed: %w", err)
	}

	c.logger.Debug("gemini tool completion",
		"prompt_length", len(userPrompt),
		"num_candidates", len(resp.Candidates),
	)

	return resp, nil
}

// CompleteWithToolsAndHistory sends a multi-turn conversation with function calling
// Supports full conversation history for multi-hop agent interactions
func (c *GeminiClient) CompleteWithToolsAndHistory(ctx context.Context, systemPrompt string, history []*genai.Content, tools []*genai.Tool) (*genai.GenerateContentResponse, error) {
	// Build system instruction if provided
	var systemInstruction *genai.Content
	if systemPrompt != "" {
		systemInstruction = genai.Text(systemPrompt)[0]
	}

	// Create generation config with tools
	genConfig := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       ptrFloat32(0.1),
		Tools:             tools,
	}

	// Generate content with full history
	resp, err := c.client.Models.GenerateContent(ctx, c.model, history, genConfig)
	if err != nil {
		return nil, fmt.Errorf("gemini tool completion with history failed: %w", err)
	}

	c.logger.Debug("gemini tool completion with history",
		"history_length", len(history),
		"num_candidates", len(resp.Candidates),
	)

	return resp, nil
}

// Close releases resources held by the Gemini client
func (c *GeminiClient) Close() error {
	// Gemini client doesn't require explicit cleanup in current SDK version
	return nil
}

// Helper functions for pointer types

func ptrFloat32(f float64) *float32 {
	f32 := float32(f)
	return &f32
}

func ptrInt32(i int) *int32 {
	i32 := int32(i)
	return &i32
}
