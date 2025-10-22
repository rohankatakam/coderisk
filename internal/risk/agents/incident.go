package agents

import "context"

type IncidentAgent struct {
	*BaseAgent
}

func NewIncidentAgent() *IncidentAgent {
	return &IncidentAgent{BaseAgent: NewBaseAgent("IncidentAgent", 1)}
}

func (a *IncidentAgent) Analyze(ctx context.Context, agentCtx interface{}) error {
	// TODO: Implement incident analysis logic
	return nil
}
