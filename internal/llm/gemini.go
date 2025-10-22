package llm

// GeminiClient implements Gemini Flash 2.0 integration
type GeminiClient struct {
	apiKey string
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{apiKey: apiKey}
}

// TODO: Implement Gemini Flash 2.0 API integration
