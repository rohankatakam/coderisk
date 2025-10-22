package llm

// AgentRequest represents a request to an LLM agent
type AgentRequest struct {
	Prompt      string
	SystemPrompt string
	MaxTokens   int
	Temperature float64
	Model       string
}

// AgentResponse represents a response from an LLM agent
type AgentResponse struct {
	Content    string
	TokensUsed int
	Model      string
	Duration   int64
	Error      error
}
