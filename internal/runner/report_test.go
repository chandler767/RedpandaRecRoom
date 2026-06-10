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
