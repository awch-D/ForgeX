// Package protocol defines the inter-agent communication primitives.
package protocol

import "strings"

// ExtractJSON strips Markdown code fences from raw LLM output,
// returning a clean JSON string for unmarshalling.
// This is the single canonical implementation; all agents should use this
// instead of maintaining their own copy.
func ExtractJSON(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		idx := strings.Index(s, "\n")
		if idx != -1 {
			s = s[idx+1:]
		}
		if lastIdx := strings.LastIndex(s, "```"); lastIdx != -1 {
			s = s[:lastIdx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}
