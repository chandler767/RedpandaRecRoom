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
