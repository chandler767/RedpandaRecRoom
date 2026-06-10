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
