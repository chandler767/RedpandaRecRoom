// internal/config/config_test.go
package config_test

import (
	"os"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/config"
)

func TestLoad(t *testing.T) {
	base := `
mode: local
search_string: "test prompt"
calls_per_llm: 3
providers:
  - id: chatgpt
    name: ChatGPT
    type: openai_compat
    model: gpt-5
    endpoint: https://api.openai.com/v1/chat/completions
    enabled: true
git:
  auto_push: false
  remote: origin
  branch: main
`
	local := `
api_keys:
  chatgpt: sk-test123
`
	baseFile := writeTemp(t, base)
	localFile := writeTemp(t, local)

	cfg, err := config.Load(baseFile, localFile)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.SearchString != "test prompt" {
		t.Errorf("SearchString = %q, want %q", cfg.SearchString, "test prompt")
	}
	if cfg.CallsPerLLM != 3 {
		t.Errorf("CallsPerLLM = %d, want 3", cfg.CallsPerLLM)
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("Providers len = %d, want 1", len(cfg.Providers))
	}
	if cfg.APIKeys["chatgpt"] != "sk-test123" {
		t.Errorf("APIKeys[chatgpt] = %q, want sk-test123", cfg.APIKeys["chatgpt"])
	}
}

func TestLoad_MissingLocalConfig(t *testing.T) {
	base := `mode: local
search_string: "test"
calls_per_llm: 5`
	baseFile := writeTemp(t, base)

	cfg, err := config.Load(baseFile, "/nonexistent/config.local.yaml")
	if err != nil {
		t.Fatalf("Load() should succeed even when local config missing: %v", err)
	}
	if cfg.APIKeys == nil {
		t.Error("APIKeys should be empty map, not nil")
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.WriteString(content)
	f.Close()
	return f.Name()
}
