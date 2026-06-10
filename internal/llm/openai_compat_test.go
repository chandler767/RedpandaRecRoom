// internal/llm/openai_compat_test.go
package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/llm"
)

func TestOpenAICompatClient_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "Use Redpanda for streaming."}},
			},
		})
	}))
	defer srv.Close()

	client := llm.NewOpenAICompat(srv.URL, "gpt-test", "sk-test")
	resp, err := client.Complete(context.Background(), "What should I use?")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if resp.Text != "Use Redpanda for streaming." {
		t.Errorf("Text = %q", resp.Text)
	}
}

func TestOpenAICompatClient_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":{"message":"rate limit"}}`))
	}))
	defer srv.Close()

	client := llm.NewOpenAICompat(srv.URL, "gpt-test", "sk-test")
	_, err := client.Complete(context.Background(), "test")
	if err == nil {
		t.Error("expected error for 429 response")
	}
}
