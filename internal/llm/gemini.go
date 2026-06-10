// internal/llm/gemini.go
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

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

type geminiClient struct {
	baseURL string
	model   string
	apiKey  string
	http    *http.Client
}

// NewGemini creates a Google Gemini client.
// Pass baseURL="" to use the production endpoint; override in tests.
func NewGemini(baseURL, model, apiKey string) Client {
	if baseURL == "" {
		baseURL = geminiBaseURL
	}
	return &geminiClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *geminiClient) Complete(ctx context.Context, prompt string) (Response, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return Response{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Candidates) == 0 {
		return Response{}, fmt.Errorf("unexpected response: %s", raw)
	}
	if len(result.Candidates[0].Content.Parts) == 0 {
		return Response{}, fmt.Errorf("empty parts in response: %s", raw)
	}

	return Response{
		Text:      result.Candidates[0].Content.Parts[0].Text,
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}
