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
