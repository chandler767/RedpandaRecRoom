// internal/llm/openai_compat.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type openAICompatClient struct {
	endpoint string
	model    string
	apiKey   string
	http     *http.Client
}

// NewOpenAICompat creates a client for OpenAI-compatible APIs
// (OpenAI, Perplexity, DeepSeek, Mistral).
func NewOpenAICompat(endpoint, model, apiKey string) Client {
	return &openAICompatClient{
		endpoint: endpoint,
		model:    model,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *openAICompatClient) Complete(ctx context.Context, prompt string) (Response, error) {
	body, _ := json.Marshal(map[string]any{
		"model":    c.model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	})

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != 200 {
		return Response{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Choices) == 0 {
		return Response{}, fmt.Errorf("unexpected response: %s", raw)
	}

	return Response{
		Text:      result.Choices[0].Message.Content,
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}
