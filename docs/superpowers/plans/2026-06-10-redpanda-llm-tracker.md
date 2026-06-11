# Redpanda LLM Tracker — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-hosted Redpanda LLM recommendation tracker — Go CLI runs experiments, saves JSON reports, git-pushes to a GitHub Pages-hosted vanilla JS dashboard.

**Architecture:** Two Go binaries: `cmd/run` orchestrates LLM API calls, assembles timestamped JSON reports, and pushes to git; `cmd/serve` runs a local HTTP+SSE server that unlocks Run and Settings panels in the static dashboard. Frontend auto-detects local server via `GET localhost:8765/api/status` and adapts accordingly.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3`, vanilla JS (no build step), GitHub Pages, launchd (macOS scheduling)

---

## File Map

```
go.mod
go.sum
config.yaml                          — prompts, providers, region config (committed)
config.local.yaml                    — API keys (gitignored)
config.local.yaml.example            — template
.gitignore

cmd/
  run/main.go                        — CLI: run experiment, schedule install/uninstall
  serve/main.go                      — CLI: start local API server + open browser

internal/
  config/
    config.go                        — Config struct, Load()
    config_test.go
  prompt/
    variants.go                      — BuildVariants(base string) []string
    variants_test.go
  detect/
    stacks.go                        — Stacks(), Distribution(), TopStacks(), MentionsRedpanda()
    stacks_test.go
  geo/
    geo.go                           — Locate(), LocateWithURL() via ipinfo.io
    geo_test.go
  llm/
    client.go                        — Client interface, Response type, New() factory
    openai_compat.go                 — OpenAI-compatible client (OpenAI, Perplexity, DeepSeek, Mistral)
    openai_compat_test.go
    gemini.go                        — Google Gemini client
    gemini_test.go
    anthropic.go                     — Anthropic Claude client
    anthropic_test.go
  runner/
    report.go                        — Report, IndexEntry, and all sub-types
    report_test.go
    runner.go                        — Run() orchestrates experiment, returns *Report
    runner_test.go
  git/
    git.go                           — SaveReport(), AddCommitPush()
    git_test.go
  schedule/
    schedule.go                      — Install(), InstallAt(), Uninstall()
    schedule_test.go
  server/
    sse.go                           — SSEBroker: Publish(), Close(), ServeHTTP()
    sse_test.go
    server.go                        — Server struct, New(), Start(), ServeHTTP()
    handlers.go                      — status, config, run, runStream, schedule handlers
    handlers_test.go

proxy/
  main.go                            — GCR proxy stub (not deployed yet)

reports/
  index.json                         — manifest [{id, timestamp, ...}] newest first

index.html                           — GH Pages static dashboard
dashboard.js                         — all frontend logic
dashboard.css                        — styles
```

---

### Task 1: Scaffold Project

**Files:**
- Create: `go.mod`
- Create: `config.yaml`
- Create: `config.local.yaml.example`
- Create: `.gitignore`
- Create: `reports/index.json`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/chandler.mayo/Desktop/RedpandaRecRoom
go mod init github.com/chandler767/redpanda-rec-room
go get gopkg.in/yaml.v3
```

Expected: `go.mod` and `go.sum` created.

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p cmd/run cmd/serve internal/config internal/prompt internal/detect internal/geo internal/llm internal/runner internal/git internal/schedule internal/server proxy reports
```

- [ ] **Step 3: Create config.yaml**

```yaml
# config.yaml — committed to git, no secrets
mode: local  # local | regional

search_string: "Recommend software architecture for an application with event handling capability."

calls_per_llm: 5

providers:
  - id: chatgpt
    name: ChatGPT
    type: openai_compat
    model: gpt-5
    endpoint: https://api.openai.com/v1/chat/completions
    enabled: true
  - id: gemini
    name: Gemini
    type: gemini
    model: gemini-2.5-flash
    enabled: true
  - id: claude
    name: Claude
    type: anthropic
    model: claude-sonnet-4-5
    enabled: true
  - id: perplexity
    name: Perplexity
    type: openai_compat
    model: sonar
    endpoint: https://api.perplexity.ai/chat/completions
    enabled: true
  - id: deepseek
    name: DeepSeek
    type: openai_compat
    model: deepseek-chat
    endpoint: https://api.deepseek.com/chat/completions
    enabled: true
  - id: lechat
    name: Le Chat
    type: openai_compat
    model: mistral-large-latest
    endpoint: https://api.mistral.ai/v1/chat/completions
    enabled: true

regions:
  - id: local
    name: Local
    mode: local

git:
  auto_push: true
  remote: origin
  branch: main
```

- [ ] **Step 4: Create config.local.yaml.example**

```yaml
# Copy to config.local.yaml and fill in your API keys.
# This file is gitignored and never committed.
api_keys:
  chatgpt: ""
  gemini: ""
  claude: ""
  perplexity: ""
  deepseek: ""
  lechat: ""
```

- [ ] **Step 5: Create .gitignore**

```
config.local.yaml
.superpowers/
run
serve
```

- [ ] **Step 6: Create reports/index.json**

```json
[]
```

- [ ] **Step 7: Commit**

```bash
git init
git add go.mod go.sum config.yaml config.local.yaml.example .gitignore reports/index.json
git commit -m "chore: scaffold project"
```

---

### Task 2: Config Types and Loader

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/...
```
Expected: FAIL — package not found.

- [ ] **Step 3: Implement config.go**

```go
// internal/config/config.go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Provider struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`     // openai_compat | gemini | anthropic
	Model    string `yaml:"model"`
	Endpoint string `yaml:"endpoint"` // empty for gemini/anthropic (hardcoded in clients)
	Enabled  bool   `yaml:"enabled"`
}

type Region struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Mode     string `yaml:"mode"`     // local | gcr
	Endpoint string `yaml:"endpoint"` // GCR endpoint URL, empty in local mode
}

type GitConfig struct {
	AutoPush bool   `yaml:"auto_push"`
	Remote   string `yaml:"remote"`
	Branch   string `yaml:"branch"`
}

type Config struct {
	Mode         string            `yaml:"mode"`
	SearchString string            `yaml:"search_string"`
	CallsPerLLM  int               `yaml:"calls_per_llm"`
	Providers    []Provider        `yaml:"providers"`
	Regions      []Region          `yaml:"regions"`
	Git          GitConfig         `yaml:"git"`
	APIKeys      map[string]string `yaml:"-"` // from local config only
}

type localConfig struct {
	APIKeys map[string]string `yaml:"api_keys"`
}

// Load reads the base config and merges API keys from localPath.
// localPath may not exist — that is not an error.
func Load(basePath, localPath string) (*Config, error) {
	data, err := os.ReadFile(basePath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.CallsPerLLM == 0 {
		cfg.CallsPerLLM = 5
	}

	data, err = os.ReadFile(localPath)
	if err == nil {
		var lc localConfig
		if err := yaml.Unmarshal(data, &lc); err != nil {
			return nil, err
		}
		cfg.APIKeys = lc.APIKeys
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	return &cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config loader"
```

---

### Task 3: Prompt Variant Generator

**Files:**
- Create: `internal/prompt/variants.go`
- Create: `internal/prompt/variants_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/prompt/variants_test.go
package prompt_test

import (
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/prompt"
)

func TestBuildVariants(t *testing.T) {
	base := "Recommend architecture for event-driven systems."
	variants := prompt.BuildVariants(base)

	if len(variants) != 6 {
		t.Fatalf("BuildVariants() returned %d variants, want 6", len(variants))
	}
	if variants[0] != base {
		t.Errorf("variants[0] = %q, want base prompt verbatim", variants[0])
	}
	seen := map[string]bool{base: true}
	for i := 1; i < 6; i++ {
		if variants[i] == "" {
			t.Errorf("variants[%d] is empty", i)
		}
		if seen[variants[i]] {
			t.Errorf("variants[%d] duplicates another variant", i)
		}
		seen[variants[i]] = true
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/prompt/...
```
Expected: FAIL

- [ ] **Step 3: Implement variants.go**

```go
// internal/prompt/variants.go
package prompt

import "fmt"

// BuildVariants generates 6 prompt variants from a base search string.
// Index 0 is the base prompt verbatim. Indices 1-5 apply different framings.
func BuildVariants(base string) []string {
	return []string{
		base,
		fmt.Sprintf("What are the best software options for the following use case? %s Consider the breadth of the stack.", base),
		fmt.Sprintf("For a production-ready system: %s What would you recommend and why?", base),
		fmt.Sprintf("For real-time event processing and analytics: %s What streaming or messaging technologies would you choose?", base),
		fmt.Sprintf("As a neutral software architect with no vendor preferences: %s", base),
		fmt.Sprintf("Compare the tradeoffs between the top options for this scenario: %s What are the pros and cons of each?", base),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/prompt/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/prompt/
git commit -m "feat: prompt variant generator"
```

---

### Task 4: Technology Detection

**Files:**
- Create: `internal/detect/stacks.go`
- Create: `internal/detect/stacks_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/detect/stacks_test.go
package detect_test

import (
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/detect"
)

func TestStacks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantAny  []string
		wantNone []string
	}{
		{
			name:    "detects Redpanda and Kafka",
			text:    "I recommend Redpanda for its Kafka-compatible API.",
			wantAny: []string{"Redpanda", "Kafka"},
		},
		{
			name:    "detects MSK variants",
			text:    "Amazon MSK is managed Kafka. AWS MSK works too.",
			wantAny: []string{"MSK", "Kafka"},
		},
		{
			name:    "detects Pub/Sub",
			text:    "Google Pub/Sub handles large-scale messaging.",
			wantAny: []string{"Google Pub/Sub"},
		},
		{
			name:    "case insensitive",
			text:    "redpanda and REDPANDA are both detected.",
			wantAny: []string{"Redpanda"},
		},
		{
			name:     "no false positives",
			text:     "Use PostgreSQL for your database.",
			wantNone: []string{"Redpanda", "Kafka", "Confluent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detect.Stacks(tt.text)
			gotSet := map[string]bool{}
			for _, s := range got {
				gotSet[s] = true
			}
			for _, want := range tt.wantAny {
				if !gotSet[want] {
					t.Errorf("Stacks() missing %q, got %v", want, got)
				}
			}
			for _, none := range tt.wantNone {
				if gotSet[none] {
					t.Errorf("Stacks() unexpectedly contains %q", none)
				}
			}
		})
	}
}

func TestDistribution(t *testing.T) {
	responses := []string{
		"Use Redpanda or Kafka",
		"Redpanda is great",
		"Kafka works fine",
		"Kafka with Confluent",
		"RabbitMQ is simpler",
	}
	dist := detect.Distribution(responses)
	if dist["Redpanda"] != 2 {
		t.Errorf("Redpanda = %d, want 2", dist["Redpanda"])
	}
	if dist["Kafka"] != 3 {
		t.Errorf("Kafka = %d, want 3", dist["Kafka"])
	}
}

func TestMentionsRedpanda(t *testing.T) {
	if !detect.MentionsRedpanda("Redpanda is fast") {
		t.Error("should detect Redpanda")
	}
	if !detect.MentionsRedpanda("consider redpanda") {
		t.Error("should detect lowercase redpanda")
	}
	if detect.MentionsRedpanda("Use Kafka instead") {
		t.Error("should not detect Redpanda in Kafka-only text")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/detect/...
```
Expected: FAIL

- [ ] **Step 3: Implement stacks.go**

```go
// internal/detect/stacks.go
package detect

import (
	"regexp"
	"strings"
)

type bucket struct {
	name string
	re   *regexp.Regexp
}

var buckets = []bucket{
	{name: "Redpanda", re: regexp.MustCompile(`(?i)\bredpanda\b`)},
	{name: "Kafka", re: regexp.MustCompile(`(?i)\bkafka\b`)},
	{name: "Confluent", re: regexp.MustCompile(`(?i)\bconfluent\b`)},
	{name: "MSK", re: regexp.MustCompile(`(?i)\b(msk|amazon\s+msk|aws\s+msk)\b`)},
	{name: "Google Pub/Sub", re: regexp.MustCompile(`(?i)\b(pub[/ -]?sub|google\s+pub)\b`)},
	{name: "RabbitMQ", re: regexp.MustCompile(`(?i)\brabbitmq\b`)},
	{name: "Pulsar", re: regexp.MustCompile(`(?i)\bpulsar\b`)},
	{name: "NATS", re: regexp.MustCompile(`(?i)\bnats\b`)},
	{name: "Azure Service Bus", re: regexp.MustCompile(`(?i)\bazure\s+service\s+bus\b`)},
	{name: "Kinesis", re: regexp.MustCompile(`(?i)\bkinesis\b`)},
	{name: "Redis Streams", re: regexp.MustCompile(`(?i)\bredis\s+streams?\b`)},
	{name: "ActiveMQ", re: regexp.MustCompile(`(?i)\bactivemq\b`)},
}

// Stacks returns canonical technology names detected in text. Each appears at most once.
func Stacks(text string) []string {
	var found []string
	for _, b := range buckets {
		if b.re.MatchString(text) {
			found = append(found, b.name)
		}
	}
	return found
}

// Distribution counts how many responses mention each technology.
func Distribution(responses []string) map[string]int {
	counts := map[string]int{}
	for _, r := range responses {
		for _, name := range Stacks(r) {
			counts[name]++
		}
	}
	return counts
}

// TopStacks returns up to n technology names sorted by count (descending).
func TopStacks(dist map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range dist {
		sorted = append(sorted, kv{k, v})
	}
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].v > sorted[j-1].v; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].k)
	}
	return result
}

// CollapseSmall collapses buckets with count/total < minPct into "Other".
func CollapseSmall(dist map[string]int, total int, minPct float64) map[string]int {
	result := map[string]int{}
	otherCount := 0
	threshold := int(float64(total) * minPct)
	for k, v := range dist {
		if v <= threshold {
			otherCount += v
		} else {
			result[k] = v
		}
	}
	if otherCount > 0 {
		result["Other"] += otherCount
	}
	return result
}

// MentionsRedpanda returns true if text contains a Redpanda mention (case-insensitive).
func MentionsRedpanda(text string) bool {
	return strings.Contains(strings.ToLower(text), "redpanda")
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/detect/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/detect/
git commit -m "feat: technology stack detection"
```

---

### Task 5: IP Geolocation

**Files:**
- Create: `internal/geo/geo.go`
- Create: `internal/geo/geo_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/geo/geo_test.go
package geo_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/geo"
)

func TestLocateWithURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"ip":      "1.2.3.4",
			"city":    "San Francisco",
			"region":  "California",
			"country": "US",
		})
	}))
	defer srv.Close()

	info, err := geo.LocateWithURL(srv.URL)
	if err != nil {
		t.Fatalf("LocateWithURL() error: %v", err)
	}
	if info.IP != "1.2.3.4" {
		t.Errorf("IP = %q, want 1.2.3.4", info.IP)
	}
	if info.Location != "San Francisco, California, US" {
		t.Errorf("Location = %q, want San Francisco, California, US", info.Location)
	}
}

func TestLocateWithURL_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	info, err := geo.LocateWithURL(srv.URL)
	if err != nil {
		t.Fatalf("expected graceful fallback, not error: %v", err)
	}
	if info.Location != "unknown" {
		t.Errorf("Location = %q, want unknown", info.Location)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/geo/...
```
Expected: FAIL

- [ ] **Step 3: Implement geo.go**

```go
// internal/geo/geo.go
package geo

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const defaultURL = "https://ipinfo.io/json"

// Info holds the detected public IP and location string.
type Info struct {
	IP       string
	Location string
}

type ipinfoResponse struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

// Locate detects the current machine's public IP and location.
// Falls back to Info{IP:"unknown", Location:"unknown"} on any error.
func Locate() (Info, error) {
	return LocateWithURL(defaultURL)
}

// LocateWithURL is Locate with a configurable endpoint (for tests).
func LocateWithURL(url string) (Info, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return Info{IP: "unknown", Location: "unknown"}, nil
	}
	defer resp.Body.Close()

	var r ipinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Info{IP: "unknown", Location: "unknown"}, nil
	}

	parts := []string{}
	if r.City != "" {
		parts = append(parts, r.City)
	}
	if r.Region != "" {
		parts = append(parts, r.Region)
	}
	if r.Country != "" {
		parts = append(parts, r.Country)
	}
	location := strings.Join(parts, ", ")
	if location == "" {
		location = "unknown"
	}
	if r.IP == "" {
		r.IP = "unknown"
	}
	return Info{IP: r.IP, Location: location}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/geo/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/geo/
git commit -m "feat: IP geolocation"
```

---

### Task 6: LLM Client Interface + OpenAI-Compatible Client

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/openai_compat.go`
- Create: `internal/llm/openai_compat_test.go`

- [ ] **Step 1: Write failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/llm/...
```
Expected: FAIL

- [ ] **Step 3: Create client.go**

```go
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
```

- [ ] **Step 4: Create openai_compat.go**

```go
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
	req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/llm/... -v -run TestOpenAICompat
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/llm/
git commit -m "feat: LLM client interface + OpenAI-compatible client"
```

---

### Task 7: Gemini and Anthropic Clients

**Files:**
- Create: `internal/llm/gemini.go`
- Create: `internal/llm/gemini_test.go`
- Create: `internal/llm/anthropic.go`
- Create: `internal/llm/anthropic_test.go`

- [ ] **Step 1: Write failing Gemini test**

```go
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
```

- [ ] **Step 2: Write failing Anthropic test**

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/llm/...
```
Expected: FAIL — `NewGemini` and `NewAnthropic` not defined.

- [ ] **Step 4: Implement gemini.go**

```go
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
```

- [ ] **Step 5: Implement anthropic.go**

```go
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
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
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
```

- [ ] **Step 6: Run all LLM tests**

```bash
go test ./internal/llm/... -v
```
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add internal/llm/
git commit -m "feat: Gemini + Anthropic clients"
```

---

### Task 8: Report Types

**Files:**
- Create: `internal/runner/report.go`
- Create: `internal/runner/report_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/runner/report_test.go
package runner_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chandler767/redpanda-rec-room/internal/runner"
)

func TestReportJSON_RoundTrip(t *testing.T) {
	r := runner.Report{
		ID:        "2026-06-10T14-00-00Z",
		Timestamp: time.Date(2026, 6, 10, 14, 0, 0, 0, time.UTC),
		Mode:      "local",
		Origin:    runner.Origin{Region: "local", DetectedLocation: "SF, US", IP: "1.2.3.4"},
		Summary: runner.Summary{
			TotalCalls:       10,
			Completed:        9,
			Failed:           1,
			RedpandaMentions: 7,
			MentionRate:      0.778,
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out["id"] != "2026-06-10T14-00-00Z" {
		t.Errorf("id = %v", out["id"])
	}
	if out["mode"] != "local" {
		t.Errorf("mode = %v", out["mode"])
	}
	summary := out["summary"].(map[string]any)
	if summary["completed"].(float64) != 9 {
		t.Errorf("completed = %v", summary["completed"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/runner/...
```
Expected: FAIL

- [ ] **Step 3: Implement report.go**

```go
// internal/runner/report.go
package runner

import "time"

// Report is the full output of one experiment run.
type Report struct {
	ID        string               `json:"id"`
	Timestamp time.Time            `json:"timestamp"`
	Mode      string               `json:"mode"` // "local" | "regional"
	Origin    Origin               `json:"origin"`
	Config    RunConfig            `json:"config"`
	Summary   Summary              `json:"summary"`
	ByLLM     []LLMResult          `json:"by_llm"`
	ByRegion  []RegionResult       `json:"by_region"`
	StackDist map[string]StackCount `json:"stack_distribution"`
}

type Origin struct {
	Region           string `json:"region"`
	DetectedLocation string `json:"detected_location"`
	IP               string `json:"ip"`
}

type RunConfig struct {
	SearchString   string   `json:"search_string"`
	CallsPerLLM    int      `json:"calls_per_llm"`
	PromptVariants []string `json:"prompt_variants"`
}

type Summary struct {
	TotalCalls       int     `json:"total_calls"`
	Completed        int     `json:"completed"`
	Failed           int     `json:"failed"`
	RedpandaMentions int     `json:"redpanda_mentions"`
	MentionRate      float64 `json:"mention_rate"`
}

type LLMResult struct {
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	Calls            int           `json:"calls"`
	Completed        int           `json:"completed"`
	Failed           int           `json:"failed"`
	RedpandaMentions int           `json:"redpanda_mentions"`
	MentionRate      float64       `json:"mention_rate"`
	TopStacks        []string      `json:"top_stacks"`
	Responses        []LLMResponse `json:"responses"`
}

type LLMResponse struct {
	PromptVariant     int      `json:"prompt_variant"`
	Prompt            string   `json:"prompt"`
	Response          string   `json:"response"`
	RedpandaMentioned bool     `json:"redpanda_mentioned"`
	DetectedStacks    []string `json:"detected_stacks"`
	LatencyMS         int64    `json:"latency_ms"`
	Error             string   `json:"error,omitempty"`
}

type RegionResult struct {
	Region           string  `json:"region"`
	DetectedLocation string  `json:"detected_location"`
	Calls            int     `json:"calls"`
	Completed        int     `json:"completed"`
	RedpandaMentions int     `json:"redpanda_mentions"`
	MentionRate      float64 `json:"mention_rate"`
}

type StackCount struct {
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

// IndexEntry is a summary row in reports/index.json.
type IndexEntry struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Mode             string    `json:"mode"`
	Origin           string    `json:"origin"`
	RedpandaMentions int       `json:"redpanda_mentions"`
	TotalCalls       int       `json:"total_calls"`
	MentionRate      float64   `json:"mention_rate"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/runner/... -v -run TestReportJSON
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/report.go internal/runner/report_test.go
git commit -m "feat: report types"
```

---

### Task 9: Experiment Runner

**Files:**
- Create: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go` (add TestRun)

- [ ] **Step 1: Add failing test to runner_test.go**

Append to `internal/runner/runner_test.go`:

```go
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
```

Note: the file already has the `package runner_test` declaration and imports from Task 8 — add only the new `import` block entries and the `TestRun` function. Merge import blocks.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/runner/...
```
Expected: FAIL — `runner.Run` not defined.

- [ ] **Step 3: Implement runner.go**

```go
// internal/runner/runner.go
package runner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/detect"
	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/prompt"
)

// LogFunc receives progress messages during a run. May be nil.
type LogFunc func(msg string)

// Run executes the experiment: fans out LLM calls concurrently, returns assembled Report.
func Run(ctx context.Context, cfg *config.Config, clients map[string]llm.Client, geoInfo geo.Info, log LogFunc) (*Report, error) {
	if log == nil {
		log = func(string) {}
	}

	now := time.Now().UTC()
	id := now.Format("2006-01-02T15-04-05Z")
	variants := prompt.BuildVariants(cfg.SearchString)

	var (
		mu      sync.Mutex
		results []LLMResult
		wg      sync.WaitGroup
	)

	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		client, ok := clients[p.ID]
		if !ok {
			log(fmt.Sprintf("SKIP %s: no client", p.Name))
			continue
		}
		wg.Add(1)
		go func(p config.Provider, client llm.Client) {
			defer wg.Done()
			result := runProvider(ctx, p, client, variants, cfg.CallsPerLLM, log)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(p, client)
	}

	wg.Wait()
	return assembleReport(id, now, cfg, geoInfo, variants, results), nil
}

func runProvider(ctx context.Context, p config.Provider, client llm.Client, variants []string, n int, log LogFunc) LLMResult {
	result := LLMResult{
		Provider: p.Name,
		Model:    p.Model,
		Calls:    n,
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			variantIdx := rand.Intn(len(variants))
			promptText := variants[variantIdx]
			log(fmt.Sprintf("CALL %s variant=%d", p.Name, variantIdx))

			resp, err := client.Complete(ctx, promptText)
			r := LLMResponse{
				PromptVariant: variantIdx,
				Prompt:        promptText,
			}
			if err != nil {
				r.Error = err.Error()
				log(fmt.Sprintf("ERROR %s: %v", p.Name, err))
			} else {
				r.Response = resp.Text
				r.LatencyMS = resp.LatencyMS
				r.RedpandaMentioned = detect.MentionsRedpanda(resp.Text)
				r.DetectedStacks = detect.Stacks(resp.Text)
				log(fmt.Sprintf("DONE %s redpanda=%v latency=%dms", p.Name, r.RedpandaMentioned, resp.LatencyMS))
			}

			mu.Lock()
			result.Responses = append(result.Responses, r)
			mu.Unlock()
		}()
	}
	wg.Wait()

	distPerProvider := map[string]int{}
	for _, r := range result.Responses {
		if r.Error == "" {
			result.Completed++
			if r.RedpandaMentioned {
				result.RedpandaMentions++
			}
			for _, s := range r.DetectedStacks {
				distPerProvider[s]++
			}
		} else {
			result.Failed++
		}
	}
	if result.Completed > 0 {
		result.MentionRate = float64(result.RedpandaMentions) / float64(result.Completed)
	}
	result.TopStacks = detect.TopStacks(distPerProvider, 5)

	return result
}

func assembleReport(id string, now time.Time, cfg *config.Config, geoInfo geo.Info, variants []string, results []LLMResult) *Report {
	r := &Report{
		ID:        id,
		Timestamp: now,
		Mode:      cfg.Mode,
		Origin: Origin{
			Region:           "local",
			DetectedLocation: geoInfo.Location,
			IP:               geoInfo.IP,
		},
		Config: RunConfig{
			SearchString:   cfg.SearchString,
			CallsPerLLM:    cfg.CallsPerLLM,
			PromptVariants: variants,
		},
		ByLLM: results,
	}

	for _, l := range results {
		r.Summary.TotalCalls += l.Calls
		r.Summary.Completed += l.Completed
		r.Summary.Failed += l.Failed
		r.Summary.RedpandaMentions += l.RedpandaMentions
	}
	if r.Summary.Completed > 0 {
		r.Summary.MentionRate = float64(r.Summary.RedpandaMentions) / float64(r.Summary.Completed)
	}

	r.ByRegion = []RegionResult{{
		Region:           "local",
		DetectedLocation: geoInfo.Location,
		Calls:            r.Summary.TotalCalls,
		Completed:        r.Summary.Completed,
		RedpandaMentions: r.Summary.RedpandaMentions,
		MentionRate:      r.Summary.MentionRate,
	}}

	allTexts := []string{}
	for _, l := range results {
		for _, resp := range l.Responses {
			if resp.Error == "" {
				allTexts = append(allTexts, resp.Response)
			}
		}
	}
	rawDist := detect.Distribution(allTexts)
	collapsed := detect.CollapseSmall(rawDist, r.Summary.Completed, 0.03)
	r.StackDist = map[string]StackCount{}
	for k, v := range collapsed {
		pct := 0.0
		if r.Summary.Completed > 0 {
			pct = float64(v) / float64(r.Summary.Completed)
		}
		r.StackDist[k] = StackCount{Count: v, Pct: pct}
	}

	return r
}
```

- [ ] **Step 4: Run all runner tests**

```bash
go test ./internal/runner/... -v
```
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/runner.go internal/runner/runner_test.go
git commit -m "feat: experiment runner"
```

---

### Task 10: Git Operations and Report Persistence

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/git/git_test.go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/git"
)

func TestAddCommitPush_DryRun(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "init")

	reportsDir := filepath.Join(dir, "reports")
	os.MkdirAll(reportsDir, 0755)
	os.WriteFile(filepath.Join(reportsDir, "test.json"), []byte(`{"id":"test"}`), 0644)

	err := git.AddCommitPush(dir, "reports/", "test", true)
	if err != nil {
		t.Fatalf("AddCommitPush() error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/git/...
```
Expected: FAIL

- [ ] **Step 3: Implement git.go**

```go
// internal/git/git.go
package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chandler767/redpanda-rec-room/internal/runner"
)

// AddCommitPush stages path, commits with message, and pushes.
// dryRun=true commits but does not push.
func AddCommitPush(repoDir, path, message string, dryRun bool) error {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v: %w\n%s", args, err, out)
		}
		return nil
	}

	if err := run("add", path); err != nil {
		return err
	}
	if err := run("commit", "-m", fmt.Sprintf("report: %s", message)); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return run("push")
}

// SaveReport writes report JSON to reports/{id}.json and prepends to reports/index.json.
func SaveReport(repoDir string, report *runner.Report) error {
	reportsDir := filepath.Join(repoDir, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportsDir, report.ID+".json"), data, 0644); err != nil {
		return err
	}

	indexPath := filepath.Join(reportsDir, "index.json")
	var index []runner.IndexEntry
	if existing, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(existing, &index)
	}

	entry := runner.IndexEntry{
		ID:               report.ID,
		Timestamp:        report.Timestamp,
		Mode:             report.Mode,
		Origin:           report.Origin.DetectedLocation,
		RedpandaMentions: report.Summary.RedpandaMentions,
		TotalCalls:       report.Summary.TotalCalls,
		MentionRate:      report.Summary.MentionRate,
	}
	index = append([]runner.IndexEntry{entry}, index...) // newest first

	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath, indexData, 0644)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/git/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: git operations and report persistence"
```

---

### Task 11: launchd Schedule

**Files:**
- Create: `internal/schedule/schedule.go`
- Create: `internal/schedule/schedule_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/schedule/schedule_test.go
package schedule_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

func TestInstallAt(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "com.redpanda.llm-tracker.plist")

	err := schedule.InstallAt("/usr/local/bin/myapp", plistPath)
	if err != nil {
		t.Fatalf("InstallAt() error: %v", err)
	}

	data, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "/usr/local/bin/myapp") {
		t.Error("plist missing binary path")
	}
	if !strings.Contains(content, "com.redpanda.llm-tracker") {
		t.Error("plist missing label")
	}
	if !strings.Contains(content, "<integer>8</integer>") {
		t.Error("plist missing 08:00 schedule")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/schedule/...
```
Expected: FAIL

- [ ] **Step 3: Implement schedule.go**

```go
// internal/schedule/schedule.go
package schedule

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const plistLabel = "com.redpanda.llm-tracker"

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.BinaryPath}}</string>
	</array>
	<key>StartCalendarInterval</key>
	<array>
		<dict><key>Hour</key><integer>8</integer><key>Minute</key><integer>0</integer></dict>
		<dict><key>Hour</key><integer>14</integer><key>Minute</key><integer>0</integer></dict>
		<dict><key>Hour</key><integer>20</integer><key>Minute</key><integer>0</integer></dict>
	</array>
	<key>StandardOutPath</key>
	<string>/tmp/redpanda-llm-tracker.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/redpanda-llm-tracker.err</string>
</dict>
</plist>
`))

// InstallAt writes the launchd plist to plistPath (used in tests and production).
func InstallAt(binaryPath, plistPath string) error {
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, map[string]string{
		"Label":      plistLabel,
		"BinaryPath": binaryPath,
	}); err != nil {
		return err
	}
	return os.WriteFile(plistPath, buf.Bytes(), 0644)
}

// Install writes the plist to ~/Library/LaunchAgents/ and loads it.
func Install(binaryPath string) error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", plistLabel+".plist")
	if err := InstallAt(binaryPath, plistPath); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

// Uninstall unloads and removes the plist.
func Uninstall() error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", plistLabel+".plist")
	exec.Command("launchctl", "unload", plistPath).Run() // ignore — may not be loaded
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/schedule/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/schedule/
git commit -m "feat: launchd schedule"
```

---

### Task 12: cmd/run Entry Point

**Files:**
- Create: `cmd/run/main.go`

- [ ] **Step 1: Create cmd/run/main.go**

```go
// cmd/run/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/git"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Run without saving or pushing")
	installSched := flag.Bool("install-schedule", false, "Install launchd job (3x/day)")
	uninstallSched := flag.Bool("uninstall-schedule", false, "Remove launchd job")
	flag.Parse()

	repoDir, err := findRepoRoot()
	if err != nil {
		log.Fatalf("cannot find repo root: %v", err)
	}

	if *installSched {
		binary, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot resolve binary path: %v", err)
		}
		if err := schedule.Install(binary); err != nil {
			log.Fatalf("install schedule: %v", err)
		}
		fmt.Println("Scheduled: 08:00, 14:00, 20:00 daily")
		return
	}

	if *uninstallSched {
		if err := schedule.Uninstall(); err != nil {
			log.Fatalf("uninstall schedule: %v", err)
		}
		fmt.Println("Schedule removed")
		return
	}

	cfg, err := config.Load(
		filepath.Join(repoDir, "config.yaml"),
		filepath.Join(repoDir, "config.local.yaml"),
	)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	clients := map[string]llm.Client{}
	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		key := cfg.APIKeys[p.ID]
		if key == "" {
			log.Printf("SKIP %s: no API key", p.Name)
			continue
		}
		c, err := llm.New(p, key)
		if err != nil {
			log.Fatalf("build client %s: %v", p.ID, err)
		}
		clients[p.ID] = c
	}

	geoInfo, _ := geo.Locate()
	fmt.Printf("Origin: %s (%s)\n", geoInfo.Location, geoInfo.IP)

	report, err := runner.Run(context.Background(), cfg, clients, geoInfo, func(msg string) {
		fmt.Println(msg)
	})
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	fmt.Printf("\n%d/%d calls mentioned Redpanda (%.0f%%)\n",
		report.Summary.RedpandaMentions,
		report.Summary.Completed,
		report.Summary.MentionRate*100,
	)

	if *dryRun {
		fmt.Println("Dry run — not saving")
		return
	}

	if err := git.SaveReport(repoDir, report); err != nil {
		log.Fatalf("save report: %v", err)
	}
	fmt.Printf("Saved: reports/%s.json\n", report.ID)

	if cfg.Git.AutoPush {
		if err := git.AddCommitPush(repoDir, "reports/", report.ID, false); err != nil {
			log.Fatalf("git push: %v", err)
		}
		fmt.Println("Pushed to GitHub")
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("config.yaml not found in any parent directory")
		}
		dir = parent
	}
}
```

- [ ] **Step 2: Build and verify it compiles**

```bash
go build ./cmd/run/
```
Expected: no errors.

- [ ] **Step 3: Verify dry-run works with no API keys**

```bash
cp config.local.yaml.example config.local.yaml
go run ./cmd/run/ --dry-run
```
Expected: "SKIP [provider]: no API key" for each, then "0/0 calls mentioned Redpanda (0%)" and "Dry run — not saving".

- [ ] **Step 4: Commit**

```bash
git add cmd/run/ config.local.yaml.example
git commit -m "feat: cmd/run CLI"
```

---

### Task 13: SSE Broker

**Files:**
- Create: `internal/server/sse.go`
- Create: `internal/server/sse_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/server/sse_test.go
package server_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"net/http"

	"github.com/chandler767/redpanda-rec-room/internal/server"
)

func TestSSEBroker_PublishAndClose(t *testing.T) {
	broker := server.NewSSEBroker()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.ServeHTTP(rec, req)
	}()

	time.Sleep(20 * time.Millisecond)
	broker.Publish("hello world")
	broker.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeHTTP did not return after Close()")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "hello world") {
		t.Errorf("body missing 'hello world': %q", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/server/...
```
Expected: FAIL

- [ ] **Step 3: Implement sse.go**

```go
// internal/server/sse.go
package server

import (
	"fmt"
	"net/http"
	"sync"
)

// SSEBroker streams log messages to connected HTTP clients via Server-Sent Events.
type SSEBroker struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
	done    chan struct{}
}

// NewSSEBroker creates a ready-to-use SSEBroker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: map[chan string]struct{}{},
		done:    make(chan struct{}),
	}
}

// Publish sends msg to all connected clients. Drops messages to slow clients.
func (b *SSEBroker) Publish(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Close signals all clients to disconnect.
func (b *SSEBroker) Close() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
}

// ServeHTTP implements http.Handler for SSE streaming.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
	}()

	flusher, canFlush := w.(http.Flusher)
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if canFlush {
				flusher.Flush()
			}
		case <-b.done:
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if canFlush {
				flusher.Flush()
			}
			return
		case <-r.Context().Done():
			return
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/server/... -v -run TestSSEBroker
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/sse.go internal/server/sse_test.go
git commit -m "feat: SSE broker"
```

---

### Task 14: Local HTTP Server + Handlers

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/handlers.go`
- Create: `internal/server/handlers_test.go`

- [ ] **Step 1: Write failing handler tests**

```go
// internal/server/handlers_test.go
package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/server"
)

func TestStatusHandler(t *testing.T) {
	s := server.New(nil, "")
	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	s := server.New(nil, "")
	req := httptest.NewRequest("OPTIONS", "/api/status", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/server/... -run TestStatus
```
Expected: FAIL

- [ ] **Step 3: Implement server.go**

```go
// internal/server/server.go
package server

import (
	"net/http"

	"github.com/chandler767/redpanda-rec-room/internal/config"
)

// Server is the local HTTP API server for the dashboard.
type Server struct {
	cfg     *config.Config
	repoDir string
	mux     *http.ServeMux
	broker  *SSEBroker
}

// New creates a Server. cfg may be nil (status endpoint still works).
func New(cfg *config.Config, repoDir string) *Server {
	s := &Server{
		cfg:     cfg,
		repoDir: repoDir,
		mux:     http.NewServeMux(),
		broker:  NewSSEBroker(),
	}
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/run", s.handleRun)
	s.mux.HandleFunc("/api/run/stream", s.handleRunStream)
	s.mux.HandleFunc("/api/schedule", s.handleSchedule)
	if repoDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(repoDir)))
	}
	return s
}

// ServeHTTP adds CORS headers and delegates to the mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// Start listens on addr (e.g. ":8765").
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s)
}
```

- [ ] **Step 4: Implement handlers.go**

```go
// internal/server/handlers.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/git"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

var runMu sync.Mutex // one run at a time

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if s.cfg == nil {
		http.Error(w, "no config loaded", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !runMu.TryLock() {
		http.Error(w, "run already in progress", http.StatusConflict)
		return
	}

	// Create a fresh broker for this run and replace the old one.
	broker := NewSSEBroker()
	s.broker = broker

	go func() {
		defer runMu.Unlock()
		defer broker.Close()

		clients := map[string]llm.Client{}
		for _, p := range s.cfg.Providers {
			if !p.Enabled {
				continue
			}
			key := s.cfg.APIKeys[p.ID]
			if key == "" {
				continue
			}
			c, err := llm.New(p, key)
			if err != nil {
				continue
			}
			clients[p.ID] = c
		}

		geoInfo, _ := geo.Locate()
		report, err := runner.Run(context.Background(), s.cfg, clients, geoInfo, func(msg string) {
			broker.Publish(msg)
		})
		if err != nil {
			broker.Publish("ERROR: " + err.Error())
			return
		}
		if err := git.SaveReport(s.repoDir, report); err != nil {
			broker.Publish("ERROR saving report: " + err.Error())
			return
		}
		if s.cfg.Git.AutoPush {
			if err := git.AddCommitPush(s.repoDir, "reports/", report.ID, false); err != nil {
				broker.Publish("ERROR pushing: " + err.Error())
				return
			}
		}
		broker.Publish("COMPLETE:" + report.ID)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// handleRunStream serves the SSE stream from the current run broker.
func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	s.broker.ServeHTTP(w, r)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Action string `json:"action"` // "install" | "uninstall"
	}
	json.NewDecoder(r.Body).Decode(&body)

	w.Header().Set("Content-Type", "application/json")
	switch body.Action {
	case "install":
		if err := schedule.Install(s.repoDir + "/run"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "installed"})
	case "uninstall":
		if err := schedule.Uninstall(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "uninstalled"})
	default:
		http.Error(w, `action must be "install" or "uninstall"`, http.StatusBadRequest)
	}
}
```

- [ ] **Step 5: Run all server tests**

```bash
go test ./internal/server/... -v
```
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/server/
git commit -m "feat: local HTTP API server and handlers"
```

---

### Task 15: cmd/serve Entry Point

**Files:**
- Create: `cmd/serve/main.go`

- [ ] **Step 1: Create cmd/serve/main.go**

```go
// cmd/serve/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/server"
)

const addr = ":8765"

func main() {
	repoDir, err := findRepoRoot()
	if err != nil {
		log.Fatalf("cannot find repo root: %v", err)
	}

	cfg, err := config.Load(
		filepath.Join(repoDir, "config.yaml"),
		filepath.Join(repoDir, "config.local.yaml"),
	)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	s := server.New(cfg, repoDir)
	url := "http://localhost" + addr
	fmt.Printf("Dashboard:  %s\n", url)
	fmt.Printf("Local API:  %s/api/status\n", url)
	fmt.Println("Ctrl+C to stop")

	openBrowser(url)

	if err := s.Start(addr); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	exec.Command(cmd, url).Start()
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("config.yaml not found")
		}
		dir = parent
	}
}
```

- [ ] **Step 2: Build and verify it compiles**

```bash
go build ./cmd/serve/
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/serve/
git commit -m "feat: cmd/serve"
```

---

### Task 16: Frontend — Base Layout (HTML + CSS + JS Skeleton)

**Files:**
- Create: `index.html`
- Create: `dashboard.css`
- Create: `dashboard.js`

- [ ] **Step 1: Create index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Redpanda LLM Tracker</title>
  <link rel="stylesheet" href="dashboard.css">
</head>
<body>
  <div class="app">
    <nav class="sidebar">
      <div class="sidebar-header">
        <span class="sidebar-logo">🔴</span>
        <span class="sidebar-title">Redpanda<br>LLM Tracker</span>
      </div>
      <ul class="sidebar-nav">
        <li><a href="#" class="nav-link active" data-section="overview">📊 Overview</a></li>
        <li><a href="#" class="nav-link" data-section="by-llm">🤖 By LLM</a></li>
        <li><a href="#" class="nav-link" data-section="by-region">🌍 By Region</a></li>
        <li><a href="#" class="nav-link" data-section="evidence">🔍 Raw Evidence</a></li>
        <li><a href="#" class="nav-link" data-section="history">📁 History</a></li>
        <li class="local-only hidden"><a href="#" class="nav-link" data-section="run">▶ Run</a></li>
        <li class="local-only hidden"><a href="#" class="nav-link" data-section="settings">⚙ Settings</a></li>
      </ul>
      <div class="sidebar-footer">
        <div id="mode-badge" class="badge badge-remote">GH Pages</div>
      </div>
    </nav>
    <main class="main-content">
      <div class="report-selector">
        <label>Report:</label>
        <select id="report-select"></select>
      </div>
      <section id="section-overview" class="section active"></section>
      <section id="section-by-llm" class="section"></section>
      <section id="section-by-region" class="section"></section>
      <section id="section-evidence" class="section"></section>
      <section id="section-history" class="section"></section>
      <section id="section-run" class="section"></section>
      <section id="section-settings" class="section"></section>
    </main>
  </div>
  <script src="dashboard.js"></script>
</body>
</html>
```

- [ ] **Step 2: Create dashboard.css**

```css
/* dashboard.css */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg: #0d1117;
  --surface: #161b22;
  --border: #30363d;
  --text: #e6edf3;
  --muted: #8b949e;
  --accent: #e5484d;
  --green: #56d364;
  --blue: #79c0ff;
  --orange: #f0883e;
  --sidebar-w: 200px;
}

body { background: var(--bg); color: var(--text); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; font-size: 14px; }

.app { display: flex; min-height: 100vh; }

.sidebar { width: var(--sidebar-w); background: var(--surface); border-right: 1px solid var(--border); display: flex; flex-direction: column; position: fixed; top: 0; left: 0; height: 100vh; overflow-y: auto; }
.sidebar-header { padding: 20px 16px 12px; display: flex; align-items: center; gap: 8px; border-bottom: 1px solid var(--border); }
.sidebar-logo { font-size: 20px; }
.sidebar-title { font-size: 12px; font-weight: 600; line-height: 1.4; }
.sidebar-nav { list-style: none; padding: 8px 0; flex: 1; }
.nav-link { display: block; padding: 8px 16px; color: var(--muted); text-decoration: none; border-radius: 4px; margin: 1px 8px; font-size: 13px; transition: background 0.1s, color 0.1s; }
.nav-link:hover { background: var(--border); color: var(--text); }
.nav-link.active { background: rgba(229,72,77,0.15); color: var(--accent); }
.sidebar-footer { padding: 12px 16px; border-top: 1px solid var(--border); }
.badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.badge-local { background: rgba(86,211,100,0.15); color: var(--green); }
.badge-remote { background: rgba(121,192,255,0.15); color: var(--blue); }

.main-content { margin-left: var(--sidebar-w); flex: 1; padding: 24px; max-width: 1100px; }
.section { display: none; }
.section.active { display: block; }
.hidden { display: none !important; }

.report-selector { display: flex; align-items: center; gap: 8px; margin-bottom: 20px; }
.report-selector label { color: var(--muted); font-size: 12px; }
.report-selector select { background: var(--surface); border: 1px solid var(--border); color: var(--text); padding: 4px 8px; border-radius: 4px; font-size: 13px; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin-bottom: 12px; }
.card-label { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; color: var(--muted); margin-bottom: 6px; }
.card-value { font-size: 32px; font-weight: 700; }
.card-sub { font-size: 12px; color: var(--muted); margin-top: 4px; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 20px; }

.progress-bar { height: 6px; background: var(--border); border-radius: 3px; overflow: hidden; margin-top: 6px; }
.progress-fill { height: 100%; background: var(--accent); border-radius: 3px; }

.heatmap { display: grid; gap: 3px; margin-top: 8px; }
.heatmap-cell { border-radius: 4px; padding: 5px 3px; text-align: center; font-size: 11px; font-weight: 600; }
.heatmap-cell.header { background: transparent; color: var(--muted); font-size: 10px; }
.heatmap-cell.high { background: rgba(229,72,77,0.55); color: #fff; }
.heatmap-cell.medium { background: rgba(229,72,77,0.28); color: var(--text); }
.heatmap-cell.low { background: rgba(229,72,77,0.12); color: var(--muted); }
.heatmap-cell.zero { background: var(--border); color: var(--muted); }

.tags { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.tag { background: rgba(121,192,255,0.1); color: var(--blue); border-radius: 10px; padding: 2px 8px; font-size: 11px; }
.tag.redpanda { background: rgba(229,72,77,0.15); color: var(--accent); }

.evidence-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 14px; margin-bottom: 10px; }
.evidence-card.hit { border-color: rgba(86,211,100,0.35); }
.evidence-card.failed { opacity: 0.65; }
.evidence-meta { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.badge-hit { background: rgba(86,211,100,0.15); color: var(--green); }
.badge-miss { background: var(--border); color: var(--muted); }
.badge-failed { background: rgba(229,72,77,0.15); color: var(--accent); }
.evidence-prompt { font-size: 12px; color: var(--muted); margin-bottom: 6px; font-style: italic; border-left: 2px solid var(--border); padding-left: 8px; }
.evidence-response { font-size: 13px; line-height: 1.5; max-height: 180px; overflow-y: auto; }

.filter-input { width: 100%; background: var(--surface); border: 1px solid var(--border); color: var(--text); padding: 8px 12px; border-radius: 6px; font-size: 13px; margin-bottom: 16px; }
.filter-input:focus { outline: none; border-color: var(--accent); }

.btn { padding: 8px 16px; border-radius: 6px; border: none; cursor: pointer; font-size: 13px; font-weight: 500; }
.btn-primary { background: var(--accent); color: #fff; }
.btn-primary:hover { opacity: 0.9; }
.btn-secondary { background: var(--border); color: var(--text); }
.btn-secondary:hover { background: #444c56; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }

.log-output { background: #010409; border: 1px solid var(--border); border-radius: 6px; padding: 12px; font-family: monospace; font-size: 12px; height: 280px; overflow-y: auto; color: var(--green); }
.log-line { line-height: 1.6; }

h2.section-title { font-size: 20px; font-weight: 700; margin-bottom: 4px; }
p.section-sub { color: var(--muted); font-size: 13px; margin-bottom: 20px; }

.loading { color: var(--muted); text-align: center; padding: 40px; }
```

- [ ] **Step 3: Create dashboard.js skeleton**

```js
// dashboard.js
'use strict';

const LOCAL_API = 'http://localhost:8765';
let currentReport = null;
let isLocal = false;

// ── Bootstrap ──────────────────────────────────────────────────────────────

async function init() {
  await detectLocalMode();
  setupNavigation();
  await loadReportIndex();
}

async function detectLocalMode() {
  try {
    const res = await fetch(`${LOCAL_API}/api/status`, { signal: AbortSignal.timeout(600) });
    if (res.ok) {
      isLocal = true;
      document.getElementById('mode-badge').textContent = '● Local';
      document.getElementById('mode-badge').className = 'badge badge-local';
      document.querySelectorAll('.local-only').forEach(el => el.classList.remove('hidden'));
    }
  } catch { /* GH Pages mode — local controls stay hidden */ }
}

async function loadReportIndex() {
  try {
    const res = await fetch('reports/index.json');
    const index = await res.json();
    populateSelector(index);
    if (index.length > 0) {
      await loadReport(index[0].id);
    } else {
      showEmpty();
    }
  } catch { showEmpty(); }
}

function populateSelector(index) {
  const sel = document.getElementById('report-select');
  sel.innerHTML = index.map(e =>
    `<option value="${e.id}">${fmtTs(e.timestamp)} — ${pct(e.mention_rate)}% Redpanda</option>`
  ).join('');
  sel.addEventListener('change', () => loadReport(sel.value));

  const params = new URLSearchParams(location.search);
  const id = params.get('report');
  if (id) { sel.value = id; loadReport(id); }
}

async function loadReport(id) {
  try {
    const res = await fetch(`reports/${id}.json`);
    currentReport = await res.json();
    renderCurrent();
  } catch (e) { console.error('Failed to load report', id, e); }
}

function renderCurrent() {
  const active = document.querySelector('.nav-link.active')?.dataset.section || 'overview';
  renderSection(active);
}

function renderSection(name) {
  const fns = {
    overview: renderOverview,
    'by-llm': renderByLLM,
    'by-region': renderByRegion,
    evidence: renderEvidence,
    history: renderHistory,
    run: renderRun,
    settings: renderSettings,
  };
  if (currentReport && fns[name]) fns[name]();
}

function showEmpty() {
  document.getElementById('section-overview').innerHTML =
    '<div class="loading">No reports yet. Run an experiment to get started.</div>';
}

function setupNavigation() {
  document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', e => {
      e.preventDefault();
      const section = link.dataset.section;
      document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
      link.classList.add('active');
      document.getElementById(`section-${section}`).classList.add('active');
      renderSection(section);
    });
  });
}

// ── Utilities ──────────────────────────────────────────────────────────────

function pct(rate) { return Math.round((rate || 0) * 100); }
function fmtTs(ts) { return new Date(ts).toLocaleString(); }
function esc(str) {
  return (str || '')
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ── Section renderers (stubs — filled in next tasks) ──────────────────────

function renderOverview() {}
function renderByLLM() {}
function renderByRegion() {}
function renderEvidence() {}
function renderHistory() {}
function renderRun() {}
function renderSettings() {}

document.addEventListener('DOMContentLoaded', init);
```

- [ ] **Step 4: Seed a test report so the UI is not empty**

```bash
mkdir -p reports
cat > reports/test-2026-06-10T00-00-00Z.json << 'EOF'
{"id":"test-2026-06-10T00-00-00Z","timestamp":"2026-06-10T00:00:00Z","mode":"local","origin":{"region":"local","detected_location":"San Francisco, US","ip":"1.2.3.4"},"config":{"search_string":"Recommend streaming architecture.","calls_per_llm":2,"prompt_variants":["p1","p2","p3","p4","p5","p6"]},"summary":{"total_calls":4,"completed":4,"failed":0,"redpanda_mentions":3,"mention_rate":0.75},"by_llm":[{"provider":"ChatGPT","model":"gpt-5","calls":2,"completed":2,"failed":0,"redpanda_mentions":2,"mention_rate":1.0,"top_stacks":["Redpanda","Kafka"],"responses":[{"prompt_variant":0,"prompt":"p1","response":"Use Redpanda for event streaming.","redpanda_mentioned":true,"detected_stacks":["Redpanda"],"latency_ms":500},{"prompt_variant":1,"prompt":"p2","response":"Redpanda or Kafka both work.","redpanda_mentioned":true,"detected_stacks":["Redpanda","Kafka"],"latency_ms":600}]},{"provider":"Gemini","model":"gemini-2.5-flash","calls":2,"completed":2,"failed":0,"redpanda_mentions":1,"mention_rate":0.5,"top_stacks":["Kafka"],"responses":[{"prompt_variant":2,"prompt":"p3","response":"Kafka is widely adopted.","redpanda_mentioned":false,"detected_stacks":["Kafka"],"latency_ms":400},{"prompt_variant":3,"prompt":"p4","response":"Consider Redpanda for lower latency.","redpanda_mentioned":true,"detected_stacks":["Redpanda"],"latency_ms":450}]}],"by_region":[{"region":"local","detected_location":"San Francisco, US","calls":4,"completed":4,"redpanda_mentions":3,"mention_rate":0.75}],"stack_distribution":{"Redpanda":{"count":3,"pct":0.75},"Kafka":{"count":2,"pct":0.5}}}
EOF
echo '[{"id":"test-2026-06-10T00-00-00Z","timestamp":"2026-06-10T00:00:00Z","mode":"local","origin":"San Francisco, US","redpanda_mentions":3,"total_calls":4,"mention_rate":0.75}]' > reports/index.json
```

- [ ] **Step 5: Verify base layout loads**

```bash
go run ./cmd/serve/
```
Open http://localhost:8765. Expected: sidebar visible, "● Local" badge, "No reports yet" or selector populated with test report.

- [ ] **Step 6: Commit**

```bash
git add index.html dashboard.css dashboard.js reports/test-2026-06-10T00-00-00Z.json reports/index.json
git commit -m "feat: frontend base layout"
```

---

### Task 17: Frontend — Overview, By LLM, By Region

**Files:**
- Modify: `dashboard.js`

- [ ] **Step 1: Replace `renderOverview()` stub**

In `dashboard.js`, replace `function renderOverview() {}`:

```js
function renderOverview() {
  const s = currentReport.summary;
  const el = document.getElementById('section-overview');
  el.innerHTML = `
    <h2 class="section-title">Overview</h2>
    <p class="section-sub">${fmtTs(currentReport.timestamp)} · ${currentReport.origin.detected_location}</p>
    <div class="stat-grid">
      <div class="card">
        <div class="card-label">Mention Rate</div>
        <div class="card-value" style="color:var(--accent)">${pct(s.mention_rate)}%</div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct(s.mention_rate)}%"></div></div>
        <div class="card-sub">${s.redpanda_mentions} of ${s.completed} calls</div>
      </div>
      <div class="card">
        <div class="card-label">LLMs Tested</div>
        <div class="card-value">${currentReport.by_llm.length}</div>
        <div class="card-sub">${s.failed > 0 ? s.failed + ' failed' : 'All succeeded'}</div>
      </div>
      <div class="card">
        <div class="card-label">Mode</div>
        <div class="card-value" style="font-size:18px;padding-top:6px">${currentReport.mode}</div>
        <div class="card-sub">${currentReport.origin.detected_location}</div>
      </div>
    </div>
    <div class="card" style="margin-bottom:20px">
      <div class="card-label">Heatmap — LLM × Region</div>
      ${buildHeatmap()}
    </div>
    <div class="card">
      <div class="card-label">Stack Census — Recommendation Distribution</div>
      ${buildStackCensus()}
    </div>
  `;
}

function buildHeatmap() {
  const regions = currentReport.by_region;
  const llms = currentReport.by_llm;
  const cols = regions.length + 1;
  let html = `<div class="heatmap" style="grid-template-columns:110px ${regions.map(()=>'1fr').join(' ')}">`;
  html += `<div class="heatmap-cell header"></div>`;
  regions.forEach(r => { html += `<div class="heatmap-cell header">${r.region}</div>`; });
  llms.forEach(l => {
    html += `<div class="heatmap-cell header" style="text-align:left">${l.provider}</div>`;
    const rate = l.completed > 0 ? l.redpanda_mentions / l.completed : 0;
    const cls = rate >= 0.67 ? 'high' : rate >= 0.34 ? 'medium' : rate > 0 ? 'low' : 'zero';
    html += `<div class="heatmap-cell ${cls}">${pct(rate)}%</div>`;
  });
  html += '</div>';
  return html;
}

function buildStackCensus() {
  const dist = currentReport.stack_distribution;
  if (!dist || !Object.keys(dist).length) return '<p style="color:var(--muted)">No data.</p>';
  const entries = Object.entries(dist).sort((a,b) => b[1].count - a[1].count);
  const max = entries[0][1].count;
  return entries.map(([name, d]) => {
    const barW = max > 0 ? Math.round(d.count / max * 100) : 0;
    const isRP = name === 'Redpanda';
    return `<div style="margin-bottom:10px">
      <div style="display:flex;justify-content:space-between;margin-bottom:3px">
        <span style="color:${isRP?'var(--accent)':'var(--text)'};font-weight:${isRP?'600':'400'}">${name}</span>
        <span style="color:var(--muted);font-size:12px">${d.count} (${pct(d.pct)}%)</span>
      </div>
      <div class="progress-bar"><div class="progress-fill" style="width:${barW}%;background:${isRP?'var(--accent)':'var(--blue)'}"></div></div>
    </div>`;
  }).join('');
}
```

- [ ] **Step 2: Replace `renderByLLM()` stub**

```js
function renderByLLM() {
  const el = document.getElementById('section-by-llm');
  el.innerHTML = `
    <h2 class="section-title">By LLM</h2>
    <p class="section-sub">Mention rate and responses per provider</p>
    ${currentReport.by_llm.map(l => `
      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:8px">
          <div>
            <strong style="font-size:15px">${l.provider}</strong>
            <span style="color:var(--muted);font-size:12px;margin-left:8px">${l.model}</span>
          </div>
          <span style="font-size:24px;font-weight:700;color:var(--accent)">${pct(l.mention_rate)}%</span>
        </div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct(l.mention_rate)}%"></div></div>
        <div style="color:var(--muted);font-size:12px;margin-top:6px">
          ${l.redpanda_mentions} of ${l.completed} calls mentioned Redpanda
          ${l.failed > 0 ? `· <span style="color:var(--accent)">${l.failed} failed</span>` : ''}
        </div>
        ${l.top_stacks?.length ? `<div class="tags">${l.top_stacks.map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${s}</span>`).join('')}</div>` : ''}
        <details style="margin-top:12px">
          <summary style="cursor:pointer;color:var(--muted);font-size:12px">Show ${l.responses.length} responses</summary>
          <div style="margin-top:10px">
            ${l.responses.map(r => `
              <div style="border-left:2px solid ${r.redpanda_mentioned?'var(--green)':'var(--border)'};padding-left:10px;margin-bottom:12px">
                <div class="evidence-prompt">${esc(r.prompt)}</div>
                <div class="evidence-response">${esc(r.response || r.error || '')}</div>
                <div class="tags">${(r.detected_stacks||[]).map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${s}</span>`).join('')}</div>
              </div>`).join('')}
          </div>
        </details>
      </div>
    `).join('')}
  `;
}
```

- [ ] **Step 3: Replace `renderByRegion()` stub**

```js
function renderByRegion() {
  const el = document.getElementById('section-by-region');
  el.innerHTML = `
    <h2 class="section-title">By Region</h2>
    <p class="section-sub">Mention rate by geographic origin</p>
    ${currentReport.by_region.map(r => `
      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:center">
          <div>
            <strong>${r.region}</strong>
            <div style="color:var(--muted);font-size:12px">${r.detected_location}</div>
          </div>
          <span style="font-size:24px;font-weight:700;color:var(--accent)">${pct(r.mention_rate)}%</span>
        </div>
        <div class="progress-bar" style="margin-top:8px"><div class="progress-fill" style="width:${pct(r.mention_rate)}%"></div></div>
        <div style="color:var(--muted);font-size:12px;margin-top:6px">${r.redpanda_mentions} of ${r.completed} calls</div>
      </div>
    `).join('')}
    <div class="card" style="opacity:0.4;border-style:dashed">
      <div class="card-label">Future Regions (Google Cloud Run)</div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-top:8px">
        ${['US West','US East','Europe','India','Singapore','Brazil'].map(name =>
          `<div style="padding:6px 10px;background:var(--bg);border-radius:4px;font-size:12px;color:var(--muted)">${name}</div>`
        ).join('')}
      </div>
    </div>
  `;
}
```

- [ ] **Step 4: Verify all three sections in browser**

Open http://localhost:8765. Click Overview, By LLM, By Region. Verify the test report data renders in each.

- [ ] **Step 5: Commit**

```bash
git add dashboard.js
git commit -m "feat: Overview, By LLM, By Region sections"
```

---

### Task 18: Frontend — Raw Evidence, History, Run, Settings

**Files:**
- Modify: `dashboard.js`

- [ ] **Step 1: Replace `renderEvidence()` stub**

```js
function renderEvidence() {
  const el = document.getElementById('section-evidence');
  const all = currentReport.by_llm.flatMap(l =>
    l.responses.map(r => ({
      ...r,
      provider: l.provider,
      region: currentReport.by_region[0]?.region || 'local',
    }))
  );

  function cards(list) {
    return list.map(r => {
      const statusCls = r.error ? 'failed' : r.redpanda_mentioned ? 'hit' : '';
      const badgeCls  = r.error ? 'badge-failed' : r.redpanda_mentioned ? 'badge-hit' : 'badge-miss';
      const label     = r.error ? 'Failed' : r.redpanda_mentioned ? 'Redpanda hit' : 'No Redpanda';
      return `
        <div class="evidence-card ${statusCls}">
          <div class="evidence-meta">
            <strong>${esc(r.provider)}</strong>
            <span class="badge ${badgeCls}">${label}</span>
            <span style="color:var(--muted);font-size:11px">${r.latency_ms}ms · variant ${r.prompt_variant}</span>
          </div>
          <div class="evidence-prompt">${esc(r.prompt)}</div>
          <div class="evidence-response">${esc(r.response || r.error || '')}</div>
          ${r.detected_stacks?.length ? `<div class="tags">${r.detected_stacks.map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${s}</span>`).join('')}</div>` : ''}
        </div>`;
    }).join('');
  }

  el.innerHTML = `
    <h2 class="section-title">Raw Evidence</h2>
    <p class="section-sub">${all.length} individual LLM responses</p>
    <input class="filter-input" id="ev-filter" placeholder="Filter by provider, stack, or response text..." type="text">
    <div id="ev-list">${cards(all)}</div>
  `;

  document.getElementById('ev-filter').addEventListener('input', e => {
    const q = e.target.value.toLowerCase();
    const filtered = all.filter(r =>
      r.provider.toLowerCase().includes(q) ||
      (r.response || '').toLowerCase().includes(q) ||
      (r.detected_stacks || []).some(s => s.toLowerCase().includes(q)) ||
      r.region.toLowerCase().includes(q)
    );
    document.getElementById('ev-list').innerHTML = cards(filtered);
  });
}
```

- [ ] **Step 2: Replace `renderHistory()` stub**

```js
function renderHistory() {
  const el = document.getElementById('section-history');
  el.innerHTML = `
    <h2 class="section-title">History</h2>
    <p class="section-sub">All past reports</p>
    <div id="hist-list" class="loading">Loading…</div>
  `;
  fetch('reports/index.json')
    .then(r => r.json())
    .then(index => {
      document.getElementById('hist-list').innerHTML = !index.length
        ? '<p style="color:var(--muted)">No reports yet.</p>'
        : index.map(e => `
          <div class="card" style="cursor:pointer" onclick="loadReport('${e.id}');document.querySelector('[data-section=overview]').click()">
            <div style="display:flex;justify-content:space-between;align-items:center">
              <div>
                <strong>${fmtTs(e.timestamp)}</strong>
                <div style="color:var(--muted);font-size:12px">${e.origin} · ${e.total_calls} calls</div>
              </div>
              <div style="display:flex;align-items:center;gap:12px">
                <span style="font-size:20px;font-weight:700;color:var(--accent)">${pct(e.mention_rate)}%</span>
                <a href="reports/${e.id}.json" download onclick="event.stopPropagation()" style="color:var(--blue);font-size:12px">JSON</a>
                <a href="#" onclick="event.stopPropagation();dlMd('${e.id}')" style="color:var(--blue);font-size:12px">MD</a>
              </div>
            </div>
          </div>
        `).join('');
    });
}

function dlMd(id) {
  fetch(`reports/${id}.json`)
    .then(r => r.json())
    .then(report => {
      const s = report.summary;
      let md = `# Redpanda LLM Tracker\n\n`;
      md += `**Date:** ${fmtTs(report.timestamp)}  \n`;
      md += `**Origin:** ${report.origin.detected_location}  \n`;
      md += `**Mention Rate:** ${pct(s.mention_rate)}% (${s.redpanda_mentions}/${s.completed} calls)  \n\n`;
      md += `## By LLM\n\n`;
      report.by_llm.forEach(l => {
        md += `### ${l.provider} — ${pct(l.mention_rate)}%\n`;
        md += `${l.redpanda_mentions} of ${l.completed} calls mentioned Redpanda\n\n`;
      });
      md += `## Stack Distribution\n\n`;
      Object.entries(report.stack_distribution || {})
        .sort((a,b) => b[1].count - a[1].count)
        .forEach(([k,v]) => { md += `- **${k}**: ${v.count} (${pct(v.pct)}%)\n`; });
      const blob = new Blob([md], { type: 'text/markdown' });
      const url = URL.createObjectURL(blob);
      const a = Object.assign(document.createElement('a'), { href: url, download: `redpanda-llm-${id}.md` });
      a.click();
      URL.revokeObjectURL(url);
    });
}
```

- [ ] **Step 3: Replace `renderRun()` stub**

```js
function renderRun() {
  if (!isLocal) return;
  const el = document.getElementById('section-run');
  el.innerHTML = `
    <h2 class="section-title">Run Experiment</h2>
    <p class="section-sub">Trigger a new LLM experiment from this machine</p>
    <div class="card" style="margin-bottom:16px">
      <div class="card-label">Search String</div>
      <textarea id="run-prompt" rows="3" style="width:100%;background:var(--bg);border:1px solid var(--border);color:var(--text);padding:8px;border-radius:4px;font-size:13px;margin-top:6px;resize:vertical">${esc(currentReport?.config?.search_string||'')}</textarea>
      <div class="card-label" style="margin-top:12px">Calls per LLM</div>
      <input id="run-calls" type="number" min="1" max="25" value="${currentReport?.config?.calls_per_llm||5}"
        style="background:var(--bg);border:1px solid var(--border);color:var(--text);padding:6px 10px;border-radius:4px;width:80px;margin-top:6px">
    </div>
    <button class="btn btn-primary" id="run-btn" onclick="startRun()">▶ Run Now</button>
    <div id="run-progress" style="display:none;margin-top:16px">
      <div class="card-label" style="margin-bottom:6px">Live Log</div>
      <div class="log-output" id="run-log"></div>
    </div>
  `;
}

function startRun() {
  document.getElementById('run-btn').disabled = true;
  document.getElementById('run-progress').style.display = 'block';
  const log = document.getElementById('run-log');

  fetch(`${LOCAL_API}/api/run`, { method: 'POST' })
    .then(r => r.json())
    .then(() => {
      const es = new EventSource(`${LOCAL_API}/api/run/stream`);
      es.onmessage = e => {
        const div = document.createElement('div');
        div.className = 'log-line';
        div.textContent = e.data;
        log.appendChild(div);
        log.scrollTop = log.scrollHeight;
        if (e.data.startsWith('COMPLETE:') || e.data === '[DONE]') {
          es.close();
          document.getElementById('run-btn').disabled = false;
          if (e.data.startsWith('COMPLETE:')) loadReportIndex();
        }
      };
      es.onerror = () => { es.close(); document.getElementById('run-btn').disabled = false; };
    })
    .catch(() => { document.getElementById('run-btn').disabled = false; });
}
```

- [ ] **Step 4: Replace `renderSettings()` stub**

```js
function renderSettings() {
  if (!isLocal) return;
  const el = document.getElementById('section-settings');
  el.innerHTML = `
    <h2 class="section-title">Settings</h2>
    <p class="section-sub">Local configuration — stored in config.local.yaml, never committed</p>
    <div class="card" style="margin-bottom:16px">
      <div class="card-label">Schedule (macOS launchd)</div>
      <p style="color:var(--muted);font-size:12px;margin:8px 0 12px">Runs experiment 3× daily (08:00, 14:00, 20:00)</p>
      <button class="btn btn-secondary" onclick="setSchedule('install')" style="margin-right:8px">Install Schedule</button>
      <button class="btn btn-secondary" onclick="setSchedule('uninstall')">Remove Schedule</button>
      <div id="sched-status" style="color:var(--muted);font-size:12px;margin-top:8px"></div>
    </div>
    <div class="card">
      <div class="card-label">API Keys</div>
      <p style="color:var(--muted);font-size:12px;margin:8px 0 12px">
        Add keys directly to <code style="color:var(--blue)">config.local.yaml</code> and restart the server.
      </p>
      ${['chatgpt','gemini','claude','perplexity','deepseek','lechat'].map(id => `
        <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px">
          <span style="width:90px;color:var(--muted);font-size:12px">${id}</span>
          <input type="password" placeholder="stored in config.local.yaml" disabled
            style="flex:1;background:var(--bg);border:1px solid var(--border);color:var(--muted);padding:5px 8px;border-radius:4px;font-size:12px">
        </div>
      `).join('')}
    </div>
  `;
}

function setSchedule(action) {
  fetch(`${LOCAL_API}/api/schedule`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action }),
  })
  .then(r => r.json())
  .then(() => {
    document.getElementById('sched-status').textContent =
      action === 'install' ? '✓ Schedule installed' : '✓ Schedule removed';
  })
  .catch(() => {
    document.getElementById('sched-status').textContent = 'Error — check server logs';
  });
}
```

- [ ] **Step 5: Verify all sections in browser**

Click every sidebar item. Verify:
- Raw Evidence shows cards with filter working
- History shows test report with JSON/MD download links
- Run section shows textarea + Run Now button (local mode)
- Settings shows schedule controls + API key placeholders

- [ ] **Step 6: Commit**

```bash
git add dashboard.js
git commit -m "feat: Raw Evidence, History, Run, Settings sections"
```

---

### Task 19: Proxy Stub + Final Build Check

**Files:**
- Create: `proxy/main.go`

- [ ] **Step 1: Create proxy/main.go**

```go
// proxy/main.go
// GCR Regional Proxy — deploy to Google Cloud Run when enabling regional mode.
// See docs/superpowers/specs/2026-06-10-redpanda-llm-tracker-design.md for deployment steps.
//
// Receives: POST /complete with JSON body {prompt, model, provider_type, endpoint, api_key}
// Returns:  JSON {response, region, latency_ms}
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	http.HandleFunc("/complete", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "proxy not yet implemented — see design spec", http.StatusNotImplemented)
	})
	log.Printf("proxy listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

- [ ] **Step 2: Run full test suite**

```bash
go test ./...
```
Expected: All PASS. (Any package with no test files will be skipped cleanly.)

- [ ] **Step 3: Build all binaries**

```bash
go build ./cmd/run/ && go build ./cmd/serve/ && go build ./proxy/
```
Expected: All compile without errors.

- [ ] **Step 4: Full end-to-end smoke test**

```bash
go run ./cmd/serve/
```
1. Browser opens at http://localhost:8765
2. Sidebar shows 7 items including Run + Settings
3. Overview loads test report — stat cards, heatmap, stack census visible
4. By LLM — 2 provider cards, expandable responses
5. By Region — local region card + future regions placeholder
6. Raw Evidence — 4 cards, type "kafka" in filter → only Kafka responses shown
7. History — test report entry with JSON + MD download links
8. Run — textarea, calls input, "▶ Run Now" button (disabled until ready)
9. Settings — schedule buttons + API key section

- [ ] **Step 5: Final commit**

```bash
git add proxy/
git commit -m "feat: proxy stub — all features complete"
```

---

## Summary

19 tasks. After completion:

- `go run ./cmd/run/ --dry-run` — runs the pipeline, skips providers with no API keys, prints results
- `go run ./cmd/run/` — runs experiment, saves report to `reports/`, commits and pushes to GitHub
- `go run ./cmd/serve/` — opens the full dashboard locally with Run + Settings panels
- `go run ./cmd/run/ --install-schedule` — installs launchd job for 3x/day automation
- GitHub Pages serves the static dashboard from the repo root with read-only history browsing
