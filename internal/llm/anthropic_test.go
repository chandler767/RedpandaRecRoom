// internal/llm/anthropic_test.go
package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/llm"
)

func TestAnthropicClient_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant-test" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "Consider Redpanda."}},
		})
	}))
	defer srv.Close()

	client := llm.NewAnthropic(srv.URL, "claude-test", "sk-ant-test")
	resp, err := client.Complete(context.Background(), "Best streaming platform?")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if resp.Text != "Consider Redpanda." {
		t.Errorf("Text = %q", resp.Text)
	}
}
