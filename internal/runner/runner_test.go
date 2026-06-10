// internal/runner/runner_test.go
package runner_test

import (
	"context"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
)

type mockLLM struct{ text string }

func (m *mockLLM) Complete(_ context.Context, _ string) (llm.Response, error) {
	return llm.Response{Text: m.text, LatencyMS: 100}, nil
}

func TestRun(t *testing.T) {
	cfg := &config.Config{
		Mode:         "local",
		SearchString: "What streaming tech should I use?",
		CallsPerLLM:  2,
		Providers: []config.Provider{
			{ID: "chatgpt", Name: "ChatGPT", Model: "gpt-5", Enabled: true},
		},
	}
	clients := map[string]llm.Client{
		"chatgpt": &mockLLM{text: "I recommend Redpanda for its Kafka-compatible API."},
	}
	geoInfo := geo.Info{IP: "1.2.3.4", Location: "SF, US"}

	report, err := runner.Run(context.Background(), cfg, clients, geoInfo, nil)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(report.ByLLM) != 1 {
		t.Fatalf("ByLLM len = %d, want 1", len(report.ByLLM))
	}
	result := report.ByLLM[0]
	if result.Calls != 2 {
		t.Errorf("Calls = %d, want 2", result.Calls)
	}
	if result.RedpandaMentions != 2 {
		t.Errorf("RedpandaMentions = %d, want 2", result.RedpandaMentions)
	}
	if report.Summary.MentionRate != 1.0 {
		t.Errorf("MentionRate = %f, want 1.0", report.Summary.MentionRate)
	}
	if report.StackDist["Redpanda"].Count == 0 {
		t.Error("expected Redpanda in stack distribution")
	}
}
