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
