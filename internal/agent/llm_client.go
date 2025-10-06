package agent

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// LLMClient wraps OpenAI SDK for GPT-4o access
type LLMClient struct {
	openaiClient *openai.Client
	model        string
}

// NewLLMClient creates a new OpenAI LLM client
func NewLLMClient(apiKey string) (*LLMClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(apiKey)
	return &LLMClient{
		openaiClient: client,
		model:        "gpt-4o", // Default to GPT-4o
	}, nil
}

// SetModel allows overriding the default model
func (c *LLMClient) SetModel(model string) {
	c.model = model
}

// Query sends a prompt to OpenAI and returns the response
func (c *LLMClient) Query(ctx context.Context, prompt string) (string, int, error) {
	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	completion, err := c.openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from OpenAI")
	}

	response := completion.Choices[0].Message.Content
	tokens := completion.Usage.TotalTokens

	return response, tokens, nil
}
