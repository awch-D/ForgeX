package protocol_test

import (
	"testing"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
)

func TestExtractJSON_Plain(t *testing.T) {
	input := `{"done": true, "summary": "All good"}`
	got := protocol.ExtractJSON(input)
	if got != input {
		t.Errorf("Expected plain JSON unchanged, got %q", got)
	}
}

func TestExtractJSON_WithCodeFence(t *testing.T) {
	input := "```json\n{\"done\": true}\n```"
	expected := `{"done": true}`
	got := protocol.ExtractJSON(input)
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_WithCodeFenceNoLang(t *testing.T) {
	input := "```\n{\"key\": \"value\"}\n```"
	expected := `{"key": "value"}`
	got := protocol.ExtractJSON(input)
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_WithWhitespace(t *testing.T) {
	input := "  \n  ```json\n{\"a\": 1}\n```  \n  "
	expected := `{"a": 1}`
	got := protocol.ExtractJSON(input)
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestExtractJSON_EmptyString(t *testing.T) {
	got := protocol.ExtractJSON("")
	if got != "" {
		t.Errorf("Expected empty string, got %q", got)
	}
}

func TestExtractJSON_NoFence(t *testing.T) {
	input := `  {"tool_calls": []}  `
	expected := `{"tool_calls": []}`
	got := protocol.ExtractJSON(input)
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}
