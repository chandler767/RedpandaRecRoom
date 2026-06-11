// internal/llm/anthropic.go
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

const anthropicBaseURL = "https://api.anthropic.com/v1/messages"

type anthropicClient struct {
	baseURL string
	model   string
	apiKey  string
	http    *http.Client
}

// NewAnthropic creates an Anthropic Claude client.
// Pass baseURL="" to use the production endpoint; override in tests.
func NewAnthropic(baseURL, model, apiKey string) Client {
	if baseURL == "" {
		baseURL = anthropicBaseURL
	}
	return &anthropicClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *anthropicClient) Complete(ctx context.Context, prompt string) (Response, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      c.model,
		"max_tokens": 1024,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
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
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Content) == 0 {
		return Response{}, fmt.Errorf("unexpected response: %s", raw)
	}

	return Response{
		Text:      result.Content[0].Text,
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}
