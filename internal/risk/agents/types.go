package agents

import "context"

// Agent represents a specialized risk analysis agent in the sequential chain
type Agent interface {
	// Name returns the agent's name
	Name() string

	// Analyze performs the agent's specific risk analysis
	// AgentContext is passed as generic interface{} to avoid circular imports
	Analyze(ctx context.Context, agentCtx interface{}) error

	// Priority returns the agent's execution priority (1 = highest)
	Priority() int
}

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	name     string
	priority int
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(name string, priority int) *BaseAgent {
	return &BaseAgent{
		name:     name,
		priority: priority,
	}
}

// Name returns the agent's name
func (a *BaseAgent) Name() string {
	return a.name
}

// Priority returns the agent's execution priority
func (a *BaseAgent) Priority() int {
	return a.priority
}
