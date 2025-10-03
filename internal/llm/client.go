package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/sashabaranov/go-openai"
)

// Provider represents the LLM provider (OpenAI or Anthropic)
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderNone      Provider = "none" // Phase 2 disabled
)

// Client provides unified interface for OpenAI and Anthropic
// Reference: agentic_design.md §2.2 - LLM investigation flow
// Reference: spec.md §1.3 - BYOK (Bring Your Own Key) model
type Client struct {
	provider       Provider
	openaiClient   *openai.Client
	anthropicClient *anthropic.Client
	logger         *slog.Logger
	enabled        bool
}

// NewClient creates an LLM client based on available API keys
// Security: NEVER hardcode API keys (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - LLM configuration
func NewClient(ctx context.Context) (*Client, error) {
	logger := slog.Default().With("component", "llm")

	// Check if Phase 2 is enabled
	phase2Enabled := os.Getenv("PHASE2_ENABLED") == "true"
	if !phase2Enabled {
		logger.Info("phase 2 disabled, LLM client not initialized")
		return &Client{
			provider: ProviderNone,
			logger:   logger,
			enabled:  false,
		}, nil
	}

	// Check for OpenAI API key first
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey != "" {
		client := openai.NewClient(openaiKey)
		logger.Info("openai client initialized")
		return &Client{
			provider:     ProviderOpenAI,
			openaiClient: client,
			logger:       logger,
			enabled:      true,
		}, nil
	}

	// Check for Anthropic API key
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey != "" {
		// NewClient reads ANTHROPIC_API_KEY from environment automatically
		client := anthropic.NewClient()
		logger.Info("anthropic client initialized")
		return &Client{
			provider:        ProviderAnthropic,
			anthropicClient: &client,
			logger:          logger,
			enabled:         true,
		}, nil
	}

	// No API keys configured but Phase 2 enabled - warning
	logger.Warn("phase 2 enabled but no LLM API key configured (set OPENAI_API_KEY or ANTHROPIC_API_KEY)")
	return &Client{
		provider: ProviderNone,
		logger:   logger,
		enabled:  false,
	}, nil
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
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("llm client not enabled (check PHASE2_ENABLED and API keys)")
	}

	switch c.provider {
	case ProviderOpenAI:
		return c.completeOpenAI(ctx, systemPrompt, userPrompt)
	case ProviderAnthropic:
		return c.completeAnthropic(ctx, systemPrompt, userPrompt)
	default:
		return "", fmt.Errorf("no llm provider configured")
	}
}

// completeOpenAI handles OpenAI chat completion
func (c *Client) completeOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := c.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini, // Cost-efficient model (spec.md §5.2)
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
		Temperature: 0.0, // Deterministic responses
		MaxTokens:   500, // Limit token usage
	})

	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	response := resp.Choices[0].Message.Content
	c.logger.Debug("openai completion",
		"prompt_length", len(userPrompt),
		"response_length", len(response),
		"tokens_used", resp.Usage.TotalTokens,
	)

	return response, nil
}

// completeAnthropic handles Anthropic message completion
// Note: Simplified implementation - Anthropic SDK not fully used yet
// TODO: Implement full Anthropic SDK integration in Phase 2
func (c *Client) completeAnthropic(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Placeholder - Anthropic SDK v1.13.0 has a different API than expected
	// Will implement properly when we build Phase 2 investigation
	c.logger.Warn("anthropic completion not yet implemented - falling back to error")
	return "", fmt.Errorf("anthropic completion not yet implemented")
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
