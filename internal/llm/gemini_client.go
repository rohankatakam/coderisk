package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

	// Generate content with retry logic for rate limits
	resp, err := c.generateContentWithRetry(ctx, c.model, genai.Text(userPrompt), genConfig)
	if err != nil {
		// Provide helpful error message for API key issues
		errMsg := err.Error()
		if contains(errMsg, "PERMISSION_DENIED") || contains(errMsg, "403") {
			if contains(errMsg, "leaked") {
				return nil, fmt.Errorf("gemini API key was reported as leaked - please use a different API key (set GEMINI_API_KEY environment variable)")
			}
			return nil, fmt.Errorf("gemini API key is invalid or lacks permissions - please check GEMINI_API_KEY environment variable")
		}
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

	// Generate content with full history and retry logic
	resp, err := c.generateContentWithRetry(ctx, c.model, history, genConfig)
	if err != nil {
		// Provide helpful error message for API key issues
		errMsg := err.Error()
		if contains(errMsg, "PERMISSION_DENIED") || contains(errMsg, "403") {
			if contains(errMsg, "leaked") {
				return nil, fmt.Errorf("gemini API key was reported as leaked - please use a different API key (set GEMINI_API_KEY environment variable)")
			}
			return nil, fmt.Errorf("gemini API key is invalid or lacks permissions - please check GEMINI_API_KEY environment variable")
		}
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

// generateContentWithRetry wraps GenerateContent with exponential backoff retry logic for rate limits
// Implements enhanced retry strategy per YC_DEMO_GAP_ANALYSIS.md Bug 3.1:
// - Increased retries: 3 → 5
// - Longer backoff: (2,4,8s) → (5,10,20,40,80s)
// - Better 429-specific handling
func (c *GeminiClient) generateContentWithRetry(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	maxRetries := 5  // Increased from 3 to 5 for better rate limit handling
	baseDelay := 5 * time.Second  // Increased from 2s to 5s for more aggressive backoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Make the API call
		resp, err := c.client.Models.GenerateContent(ctx, model, contents, config)

		// Check for rate limit errors (429)
		if err != nil {
			errMsg := err.Error()
			is429 := contains(errMsg, "429") || contains(errMsg, "Resource exhausted") || contains(errMsg, "RESOURCE_EXHAUSTED")

			if is429 && attempt < maxRetries {
				// Calculate exponential backoff: 5s, 10s, 20s, 40s, 80s
				delay := baseDelay * (1 << uint(attempt))
				c.logger.Warn("rate limit encountered, retrying with backoff",
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"delay_seconds", delay.Seconds(),
					"error", errMsg,
				)

				// Wait before retrying
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			// Non-429 error or exhausted retries
			if is429 {
				totalWaitTime := int((baseDelay * ((1 << uint(maxRetries)) - 1)).Seconds())
				return nil, fmt.Errorf("RATE_LIMIT_EXHAUSTED: Resource exhausted after %d retries (waited %ds total). "+
					"This indicates Gemini API quota limits. Consider: "+
					"1) Using a different API key, "+
					"2) Waiting a few minutes, or "+
					"3) Using --no-ai flag for Phase 1 only: %w",
					maxRetries,
					totalWaitTime,
					err)
			}
			return nil, err
		}

		// Success
		if attempt > 0 {
			c.logger.Info("request succeeded after retry", "attempt", attempt+1)
		}
		return resp, nil
	}

	return nil, fmt.Errorf("unexpected retry loop exit")
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
