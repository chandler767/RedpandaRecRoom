// internal/llm/client.go
package llm

import (
	"context"
	"fmt"

	"github.com/chandler767/redpanda-rec-room/internal/config"
)

// Response is the normalized result from any LLM provider.
type Response struct {
	Text      string
	LatencyMS int64
}

// Client makes completions against an LLM API.
type Client interface {
	Complete(ctx context.Context, prompt string) (Response, error)
}

// New creates the appropriate LLM client for a provider config and API key.
func New(p config.Provider, apiKey string) (Client, error) {
	switch p.Type {
	case "openai_compat":
		return NewOpenAICompat(p.Endpoint, p.Model, apiKey), nil
	case "gemini":
		return NewGemini("", p.Model, apiKey), nil
	case "anthropic":
		return NewAnthropic("", p.Model, apiKey), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", p.Type)
	}
}
