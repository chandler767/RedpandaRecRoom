// internal/runner/report.go
package runner

import "time"

// Report is the full output of one experiment run.
type Report struct {
	ID        string                `json:"id"`
	Timestamp time.Time             `json:"timestamp"`
	Mode      string                `json:"mode"` // "local" | "regional"
	Origin    Origin                `json:"origin"`
	Config    RunConfig             `json:"config"`
	Summary   Summary               `json:"summary"`
	ByLLM     []LLMResult           `json:"by_llm"`
	ByRegion  []RegionResult        `json:"by_region"`
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
