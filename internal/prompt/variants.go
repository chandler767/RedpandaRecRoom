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
