package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/sashabaranov/go-openai"
)

// Provider represents the LLM provider
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderGemini Provider = "gemini"
	ProviderNone   Provider = "none" // Phase 2 disabled
)

// Client provides multi-provider LLM interface
// Reference: agentic_design.md §2.2 - LLM investigation flow
// Reference: spec.md §1.3 - BYOK (Bring Your Own Key) model
// Supports OpenAI and Gemini providers
type Client struct {
	provider     Provider
	openaiClient *openai.Client
	geminiClient *GeminiClient
	logger       *slog.Logger
	enabled      bool
	fastModel    string // GPT-4o-mini for Agents 1-6 | gemini-2.0-flash
	deepModel    string // GPT-4o for Agents 7-8 | gemini-1.5-pro
}

// NewClient creates a multi-provider LLM client (OpenAI or Gemini)
// Security: NEVER hardcode API keys (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - LLM configuration
// Now uses config system with OS keychain support
// Supports OpenAI and Gemini providers
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	logger := slog.Default().With("component", "llm")

	// Check if Phase 2 is enabled
	phase2Enabled := os.Getenv("PHASE2_ENABLED") == "true"
	if !phase2Enabled {
		logger.Info("phase 2 disabled, LLM client not initialized")
		return &Client{
			provider:  ProviderNone,
			logger:    logger,
			enabled:   false,
			fastModel: "gpt-4o-mini",
			deepModel: "gpt-4o",
		}, nil
	}

	// Determine provider (priority: env var > config > default to gemini)
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = cfg.API.Provider
	}
	if provider == "" {
		provider = "gemini" // Default to Gemini
	}

	switch Provider(provider) {
	case ProviderGemini:
		return newGeminiClient(ctx, cfg, logger)
	case ProviderOpenAI:
		return newOpenAIClient(ctx, cfg, logger)
	default:
		logger.Warn("unknown provider, falling back to gemini", "provider", provider)
		return newGeminiClient(ctx, cfg, logger)
	}
}

// newGeminiClient initializes a Gemini provider client
func newGeminiClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Client, error) {
	// Get Gemini API key from config or environment
	geminiKey := cfg.API.GeminiKey
	if geminiKey == "" {
		geminiKey = os.Getenv("GEMINI_API_KEY")
	}

	if geminiKey == "" {
		logger.Warn("phase 2 enabled but no Gemini API key configured")
		logger.Info("set GEMINI_API_KEY environment variable or run 'crisk configure'")
		return &Client{
			provider:  ProviderNone,
			logger:    logger,
			enabled:   false,
			fastModel: "gemini-2.0-flash",
			deepModel: "gemini-1.5-pro",
		}, nil
	}

	// Determine model (from config or use defaults)
	model := cfg.API.GeminiModel
	if model == "" {
		model = "gemini-2.0-flash" // Default to production flash for speed and higher rate limits
	}

	geminiClient, err := NewGeminiClient(ctx, geminiKey, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	keySource := getGeminiKeySource(cfg)
	logger.Info("gemini client initialized", "key_source", keySource, "model", model)

	return &Client{
		provider:     ProviderGemini,
		geminiClient: geminiClient,
		logger:       logger,
		enabled:      true,
		fastModel:    model,
		deepModel:    "gemini-1.5-pro", // Use Pro for deep analysis
	}, nil
}

// newOpenAIClient initializes an OpenAI provider client
func newOpenAIClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Client, error) {
	// Get OpenAI API key from config (which already checked env var and keychain)
	openaiKey := cfg.API.OpenAIKey
	if openaiKey == "" {
		// No API key configured but Phase 2 enabled - helpful message
		logger.Warn("phase 2 enabled but no OpenAI API key configured")
		logger.Info("run 'crisk configure' to set up your OpenAI API key securely")
		logger.Info("or configure via coderisk-frontend web interface")
		return &Client{
			provider:  ProviderNone,
			logger:    logger,
			enabled:   false,
			fastModel: "gpt-4o-mini",
			deepModel: "gpt-4o",
		}, nil
	}

	client := openai.NewClient(openaiKey)
	keySource := getKeySource(cfg)
	logger.Info("openai client initialized", "key_source", keySource, "fast_model", "gpt-4o-mini", "deep_model", "gpt-4o")
	return &Client{
		provider:     ProviderOpenAI,
		openaiClient: client,
		logger:       logger,
		enabled:      true,
		fastModel:    "gpt-4o-mini", // Agents 1-6: fast analysis
		deepModel:    "gpt-4o",      // Agents 7-8: synthesis, validation
	}, nil
}

// getKeySource returns a string indicating where the API key came from
func getKeySource(cfg *config.Config) string {
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "environment"
	}
	if cfg.API.UseKeychain {
		return "keychain"
	}
	return "config_file"
}

// getGeminiKeySource returns a string indicating where the Gemini API key came from
func getGeminiKeySource(cfg *config.Config) string {
	if os.Getenv("GEMINI_API_KEY") != "" {
		return "environment"
	}
	if cfg.API.GeminiKey != "" {
		return "config_file"
	}
	return "unknown"
}

// IsEnabled returns true if an LLM client is configured and ready
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// GetProvider returns the active LLM provider
func (c *Client) GetProvider() Provider {
	return c.provider
}

// Complete sends a prompt to the LLM and returns the response
// Reference: agentic_design.md §2.2 - Investigation decisions
// Uses fastModel (GPT-4o-mini or gemini-2.0-flash) by default
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and API key)")
	}

	switch c.provider {
	case ProviderGemini:
		return c.geminiClient.Complete(ctx, systemPrompt, userPrompt)
	case ProviderOpenAI:
		return c.completeOpenAI(ctx, systemPrompt, userPrompt, c.fastModel)
	default:
		return "", fmt.Errorf("no provider configured")
	}
}

// CompleteWithDeepModel sends a prompt using the deep model (GPT-4o or gemini-1.5-pro)
// Use for complex synthesis and validation tasks (Agents 7-8)
func (c *Client) CompleteWithDeepModel(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and API key)")
	}

	switch c.provider {
	case ProviderGemini:
		// For Gemini, create a new client with the deep model
		geminiDeep, err := NewGeminiClient(context.Background(), os.Getenv("GEMINI_API_KEY"), c.deepModel)
		if err != nil {
			return "", fmt.Errorf("failed to create deep model client: %w", err)
		}
		defer geminiDeep.Close()
		return geminiDeep.Complete(ctx, systemPrompt, userPrompt)
	case ProviderOpenAI:
		return c.completeOpenAI(ctx, systemPrompt, userPrompt, c.deepModel)
	default:
		return "", fmt.Errorf("no provider configured")
	}
}

// completeOpenAI handles OpenAI chat completion with configurable model
func (c *Client) completeOpenAI(ctx context.Context, systemPrompt, userPrompt, model string) (string, error) {
	resp, err := c.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model, // Either gpt-4o-mini (fast) or gpt-4o (deep)
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.1, // Low temperature for consistent, focused responses (MVP spec)
		MaxTokens:   2000, // MVP spec: Max tokens per agent
	})

	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	response := resp.Choices[0].Message.Content
	c.logger.Debug("openai completion",
		"model", model,
		"prompt_length", len(userPrompt),
		"response_length", len(response),
		"tokens_used", resp.Usage.TotalTokens,
	)

	return response, nil
}

// ShouldEscalateToPhase2 determines if a file should trigger Phase 2 investigation
// Reference: agentic_design.md §2.1 - Phase 1 to Phase 2 transition
// This is a placeholder - actual logic will use baseline metrics
func (c *Client) ShouldEscalateToPhase2(couplingScore, coChangeScore, testRatio float64) bool {
	// Escalate if:
	// 1. Coupling is high (>10 dependencies)
	// 2. Co-change is high (>5 related files)
	// 3. Test ratio is low (<0.5)
	// Reference: risk_assessment_methodology.md §2 - Tier 1 thresholds
	if couplingScore > 10 {
		c.logger.Debug("escalating to phase 2", "reason", "high_coupling", "score", couplingScore)
		return true
	}
	if coChangeScore > 5 {
		c.logger.Debug("escalating to phase 2", "reason", "high_co_change", "score", coChangeScore)
		return true
	}
	if testRatio < 0.5 {
		c.logger.Debug("escalating to phase 2", "reason", "low_test_coverage", "ratio", testRatio)
		return true
	}

	c.logger.Debug("staying in phase 1", "coupling", couplingScore, "co_change", coChangeScore, "test_ratio", testRatio)
	return false
}

// CompleteJSON sends a prompt to the LLM and returns JSON response
// Uses response_format: json_object for structured outputs (OpenAI)
// Uses ResponseMIMEType: application/json for Gemini
// Reference: REVISED_MVP_STRATEGY.md - Two-way extraction with structured outputs
func (c *Client) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and API key)")
	}

	switch c.provider {
	case ProviderGemini:
		return c.geminiClient.CompleteJSON(ctx, systemPrompt, userPrompt)
	case ProviderOpenAI:
		return c.completeOpenAIJSON(ctx, systemPrompt, userPrompt)
	default:
		return "", fmt.Errorf("no provider configured")
	}
}

// completeOpenAIJSON handles OpenAI JSON completion
func (c *Client) completeOpenAIJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := c.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.fastModel, // Use fast model (gpt-4o-mini) for extraction
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.1,  // Low temperature for consistency
		MaxTokens:   2000, // Sufficient for batch processing
	})

	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	response := resp.Choices[0].Message.Content
	c.logger.Debug("openai json completion",
		"model", c.fastModel,
		"prompt_length", len(userPrompt),
		"response_length", len(response),
		"tokens_used", resp.Usage.TotalTokens,
	)

	return response, nil
}
