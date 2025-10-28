package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
)

// LLMClientInterface defines the interface for LLM clients
type LLMClientInterface interface {
	Query(ctx context.Context, prompt string) (string, int, error)
	SetModel(model string)
}

// LLMClient wraps OpenAI SDK for GPT-4o access
type LLMClient struct {
	openaiClient openai.Client
	model        openai.ChatModel
}

// NewLLMClient creates a new OpenAI LLM client
func NewLLMClient(apiKey string) (*LLMClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Set API key in environment for the official SDK
	os.Setenv("OPENAI_API_KEY", apiKey)

	return &LLMClient{
		openaiClient: openai.NewClient(),
		model:        openai.ChatModelGPT4o, // Default to GPT-4o
	}, nil
}

// SetModel allows overriding the default model
func (c *LLMClient) SetModel(model string) {
	c.model = openai.ChatModel(model)
}

// Query sends a prompt to OpenAI and returns the response
func (c *LLMClient) Query(ctx context.Context, prompt string) (string, int, error) {
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: c.model,
	}

	completion, err := c.openaiClient.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", 0, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from OpenAI")
	}

	response := completion.Choices[0].Message.Content
	tokens := int(completion.Usage.TotalTokens)

	return response, tokens, nil
}
