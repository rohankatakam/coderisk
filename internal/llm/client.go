package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/sashabaranov/go-openai"
)

// Provider represents the LLM provider (OpenAI only for MVP)
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderNone   Provider = "none" // Phase 2 disabled
)

// Client provides interface for OpenAI LLM
// Reference: agentic_design.md §2.2 - LLM investigation flow
// Reference: spec.md §1.3 - BYOK (Bring Your Own Key) model
// MVP: OpenAI only (matches coderisk-frontend implementation)
type Client struct {
	provider     Provider
	openaiClient *openai.Client
	logger       *slog.Logger
	enabled      bool
	fastModel    string // GPT-4o-mini for Agents 1-6
	deepModel    string // GPT-4o for Agents 7-8
}

// NewClient creates an OpenAI LLM client
// Security: NEVER hardcode API keys (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - LLM configuration
// Now uses config system with OS keychain support
// MVP: OpenAI only (matches coderisk-frontend implementation)
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
// Uses fastModel (GPT-4o-mini) by default
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and OPENAI_API_KEY)")
	}

	if c.provider != ProviderOpenAI {
		return "", fmt.Errorf("no openai provider configured")
	}

	return c.completeOpenAI(ctx, systemPrompt, userPrompt, c.fastModel)
}

// CompleteWithDeepModel sends a prompt using the deep model (GPT-4o)
// Use for complex synthesis and validation tasks (Agents 7-8)
func (c *Client) CompleteWithDeepModel(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and OPENAI_API_KEY)")
	}

	if c.provider != ProviderOpenAI {
		return "", fmt.Errorf("no openai provider configured")
	}

	return c.completeOpenAI(ctx, systemPrompt, userPrompt, c.deepModel)
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
