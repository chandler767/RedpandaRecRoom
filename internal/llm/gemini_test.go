// internal/llm/gemini_test.go
package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/llm"
)

func TestGeminiClient_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=test-key") {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{{"text": "Redpanda is excellent."}},
				}},
			},
		})
	}))
	defer srv.Close()

	client := llm.NewGemini(srv.URL, "gemini-test", "test-key")
	resp, err := client.Complete(context.Background(), "What streaming tech?")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if resp.Text != "Redpanda is excellent." {
		t.Errorf("Text = %q", resp.Text)
	}
}
