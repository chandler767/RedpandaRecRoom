# Redpanda LLM Tracker — Design Spec
_Date: 2026-06-10_

## Overview

A self-hosted version of the [Redpanda Recommendation Lab](https://redpanda-llm-experiment.vercel.app). Tests whether major LLMs recommend Redpanda when asked architecture questions, compares results across 6 LLM providers, and publishes reports to GitHub Pages.

The user runs experiments locally (or on a schedule), results are committed as JSON to the repo, and a vanilla JS static site serves the dashboard. Regional distribution (Google Cloud Run) is designed in but disabled initially — local mode with IP autodetection runs instead.

---

## Architecture

```
YOUR MACHINE
  cmd/run       — orchestrates LLM calls, assembles report JSON, git commits + pushes
  cmd/serve     — local HTTP API server; enables Run/Settings UI in the dashboard

FUTURE (disabled now)
  proxy/        — Go HTTP proxy, deployable to Google Cloud Run in 6 regions

GITHUB PAGES (static, always-on)
  index.html + dashboard.js + dashboard.css
  reports/      — timestamped JSON reports + index.json manifest
```

### Local vs Regional Mode

Controlled by `config.yaml`:

- **Local mode (default):** CLI makes LLM calls directly from the machine. IP geolocation detects location (via `ipinfo.io`). Report records `"region": "local"` + detected location.
- **Regional mode (future):** CLI fans out to 6 GCR proxy endpoints in parallel. Each proxy makes LLM calls from its geography and returns normalized results.

Report JSON schema is identical in both modes — the frontend requires no changes when switching.

---

## Repository Structure

```
cmd/
  run/main.go       — experiment runner CLI
  serve/main.go     — local API server
proxy/
  main.go           — GCR proxy (disabled; deploy later)
config.yaml         — prompts, model list, region config, thresholds
config.local.yaml   — API keys (gitignored)
reports/
  index.json        — manifest of all reports (newest first)
  2026-06-10T14-00-00Z.json
  ...
index.html          — GH Pages entry point
dashboard.js        — all visualization logic
dashboard.css       — styles
.gitignore          — config.local.yaml, .superpowers/
```

---

## Go CLIs

### `cmd/run` — Experiment Runner

| Command | Description |
|---|---|
| `go run ./cmd/run` | Run experiment, save report, git commit + push |
| `go run ./cmd/run --dry-run` | Run without saving or pushing |
| `go run ./cmd/run --install-schedule` | Install launchd job (3x/day) |
| `go run ./cmd/run --uninstall-schedule` | Remove launchd job |

### `cmd/serve` — Local Dashboard Server

| Command | Description |
|---|---|
| `go run ./cmd/serve` | Start local API server + open dashboard in browser |

### Experiment Run Flow

1. Load `config.yaml` + `config.local.yaml` (API keys)
2. Detect location via `ipinfo.io` (local mode)
3. Build 6 prompt variants from the configured search string
4. For each selected LLM: fan out N calls (default 5) with randomized prompt variants
5. Parse each response: detect Redpanda mention, classify stack technologies
6. Assemble report JSON
7. Write `reports/{timestamp}.json`
8. Update `reports/index.json` manifest
9. `git add reports/ && git commit -m "report: {timestamp}" && git push`

### Prompt Variants

Six variants are generated from the base search string, covering:
1. Base prompt as-is
2. Stack breadth framing ("What are the best options for...")
3. Production-ready framing ("For a production system that needs...")
4. Event/analytics framing ("For real-time event processing and analytics...")
5. Neutral advisor framing ("As a neutral architect, what would you recommend...")
6. Tradeoffs framing ("Compare the tradeoffs between options for...")

### Technology Detection

Regex-based classification of LLM response text:

| Bucket | Detection |
|---|---|
| Redpanda | `\bredpanda\b` |
| Kafka | `\bkafka\b` |
| Confluent | `\bconfluent\b` |
| MSK | `\b(msk\|amazon msk\|aws msk)\b` |
| Google Pub/Sub | `\bpub[/ -]?sub\b` |
| Other | RabbitMQ, Pulsar, NATS, Azure Service Bus, Event Hubs, SQS, SNS, Kinesis, Redis Streams, ActiveMQ |

Items under 3% of total mentions collapse into "Other" in the distribution display.

---

## Report JSON Schema

```json
{
  "id": "2026-06-10T14-00-00Z",
  "timestamp": "2026-06-10T14:00:00Z",
  "mode": "local",
  "origin": {
    "region": "local",
    "detected_location": "San Francisco, US",
    "ip": "1.2.3.4"
  },
  "config": {
    "search_string": "Recommend software architecture for...",
    "calls_per_llm": 5,
    "prompt_variants": ["...", "...", "...", "...", "...", "..."]
  },
  "summary": {
    "total_calls": 30,
    "completed": 28,
    "failed": 2,
    "redpanda_mentions": 21,
    "mention_rate": 0.75
  },
  "by_llm": [
    {
      "provider": "ChatGPT",
      "model": "gpt-5",
      "calls": 5,
      "completed": 5,
      "failed": 0,
      "redpanda_mentions": 4,
      "mention_rate": 0.80,
      "top_stacks": ["Redpanda", "Kafka", "Confluent"],
      "responses": [
        {
          "prompt_variant": 2,
          "prompt": "...",
          "response": "...",
          "redpanda_mentioned": true,
          "detected_stacks": ["Redpanda", "Kafka"],
          "latency_ms": 1240
        }
      ]
    }
  ],
  "by_region": [
    {
      "region": "local",
      "detected_location": "San Francisco, US",
      "calls": 30,
      "completed": 28,
      "redpanda_mentions": 21,
      "mention_rate": 0.75
    }
  ],
  "stack_distribution": {
    "Redpanda": { "count": 21, "pct": 0.75 },
    "Kafka": { "count": 18, "pct": 0.64 },
    "Confluent": { "count": 8, "pct": 0.29 },
    "Other": { "count": 4, "pct": 0.14 }
  }
}
```

---

## Local API Server (`cmd/serve`)

Runs on `localhost:8765`. Provides endpoints the dashboard calls when running locally.

| Endpoint | Method | Description |
|---|---|---|
| `/api/status` | GET | Health check — frontend uses this to detect local mode |
| `/api/config` | GET | Returns current config (models, prompts, schedule status) |
| `/api/config` | POST | Save config changes |
| `/api/run` | POST | Trigger an experiment run |
| `/api/run/stream` | GET | SSE stream of live log output during a run |
| `/api/schedule` | POST | Install or uninstall launchd job |

Frontend fetches `http://localhost:8765/api/status` on load. If it responds, local controls (Run, Settings) are shown. If it times out, they are hidden.

---

## Frontend (`index.html` + `dashboard.js`)

Vanilla JS, no build step. Served by GitHub Pages from repo root.

### Sidebar Nav Sections

| Section | GH Pages | Local |
|---|---|---|
| Overview | ✓ | ✓ |
| By LLM | ✓ | ✓ |
| By Region | ✓ | ✓ |
| Raw Evidence | ✓ | ✓ |
| History | ✓ | ✓ |
| Run | — | ✓ |
| Settings | — | ✓ |

### Overview
- Overall mention rate (large stat)
- Heatmap: LLM × Region grid (color-coded: ≥67% high, ≥34% medium, >0% low, 0% zero)
- Trend sparkline across last N reports
- Stack Census: distribution bar chart (Redpanda vs Kafka, Confluent, MSK, Pub/Sub, Other)
- Last run timestamp + origin

### By LLM
- One card per provider: mention rate, progress bar, mention count, failed count
- Top 5 stack tag cloud per provider
- Expandable to show individual responses

### By Region
- Bar list: per-region mention rate + count
- Region metadata: detected location, IP, latency
- Placeholder cards for future GCR regions (greyed out)

### Raw Evidence
- Live text filter across: provider, region, stack tags, response text
- One card per LLM call: prompt used, full response, detected stacks
- Badges: "Redpanda hit" (green) / "No Redpanda" (grey) / "Failed" (red)

### History
- Chronological list from `reports/index.json`
- Click to load any past report into dashboard
- Trend chart across all runs
- Download as JSON or Markdown per report

### Run (local only)
- Custom search string input
- Calls per LLM (1–25, default 5)
- LLM provider checkboxes
- Shows 6 prompt variants that will be used
- Run Now button → SSE stream shows live log
- Progress indicator per LLM
- Auto-reloads dashboard on completion

### Settings (local only)
- API keys per provider (saved to `config.local.yaml`, never committed)
- Prompt library: edit/add/remove base prompts
- Schedule: install/uninstall launchd job (3x/day)
- Region endpoints: toggle local vs GCR per region
- Git remote: set repo URL for auto-push

### Report Loading
Frontend loads `reports/index.json` on startup to get the manifest. Loads the most recent report by default. Supports `?report=2026-06-10T14-00-00Z` query parameter for direct links to historical reports.

---

## Automation

### launchd Job (macOS)

Installed via `go run ./cmd/run --install-schedule`. Creates `~/Library/LaunchAgents/com.redpanda.llm-tracker.plist` configured to run 3x/day (08:00, 14:00, 20:00 local time).

### GitHub Actions

Not used for running experiments (single-origin limitation). Used only for GitHub Pages deployment — GH Pages auto-deploys from the `main` branch root on every push.

---

## LLM Providers

| Provider | API | Model | Key env var |
|---|---|---|---|
| ChatGPT | OpenAI API | gpt-5 | `OPENAI_API_KEY` |
| Gemini | Google Gemini API | gemini-2.5-flash | `GEMINI_API_KEY` |
| Claude | Anthropic Messages API | claude-sonnet-4-5 | `ANTHROPIC_API_KEY` |
| Perplexity | Perplexity API | sonar | `PERPLEXITY_API_KEY` |
| DeepSeek | DeepSeek API | deepseek-chat | `DEEPSEEK_API_KEY` |
| Le Chat | Mistral API | mistral-large-latest | `MISTRAL_API_KEY` |

---

## Future: Google Cloud Run Regional Proxies

When ready to enable regional mode:

1. Deploy `proxy/` as a Go container to 6 GCR services, one per region:
   - `us-west1` (US West)
   - `us-east1` (US East)
   - `europe-west1` (Europe)
   - `asia-south1` (India)
   - `asia-southeast1` (Singapore)
   - `southamerica-east1` (Brazil)
2. Add GCR endpoint URLs to `config.yaml` under `regions`
3. Toggle `mode: regional` in `config.yaml`
4. No frontend changes required — report schema is identical

The proxy receives `{prompt, model, api_key}` per request, calls the LLM API, and returns `{response, region, latency_ms}`. API keys travel encrypted per-request from the local CLI — never stored on GCR.

---

## Error Handling

- Per-LLM call failures are recorded in the report (`failed: N`) and shown in the UI — they do not abort the run
- If all calls for an LLM fail, the LLM is marked as fully failed in the report
- Git push failure logs an error but does not delete the local report file
- IP geolocation failure falls back to `"detected_location": "unknown"`

---

## What Is Not In Scope

- User authentication
- Multi-user / team support
- Real-time collaborative runs
- Database storage (JSON files only)
- Windows support for launchd scheduling (launchd is macOS-only; cron instructions provided in README instead)
