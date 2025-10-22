package llm

import (
	"context"
)

// AgentExecutor handles parallel and sequential agent execution
type AgentExecutor struct {
	client *Client
}

// NewAgentExecutor creates a new agent executor
func NewAgentExecutor(client *Client) *AgentExecutor {
	return &AgentExecutor{client: client}
}

// ExecuteParallel runs multiple agent requests in parallel
func (e *AgentExecutor) ExecuteParallel(ctx context.Context, requests []AgentRequest) ([]AgentResponse, error) {
	// TODO: Implement parallel execution
	return []AgentResponse{}, nil
}

// ExecuteSequential runs multiple agent requests sequentially
func (e *AgentExecutor) ExecuteSequential(ctx context.Context, requests []AgentRequest) ([]AgentResponse, error) {
	// TODO: Implement sequential execution
	return []AgentResponse{}, nil
}
