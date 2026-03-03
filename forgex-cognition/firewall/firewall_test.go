package firewall_test

import (
	"context"
	"strings"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/firewall"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// MockProvider generates a deterministic response: ALLOW if text contains "database", else REJECT.
type MockProvider struct{}

func (m *MockProvider) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	text := messages[0].Content
	// Default to REJECT, only ALLOW if we find target word "DB_DSN" which is unique to our test valid string
	if strings.Contains(text, "DB_DSN") {
		return &provider.Response{Content: `{"result": "ALLOW"}`}, nil
	}
	return &provider.Response{Content: `{"result": "REJECT"}`}, nil
}
func (m *MockProvider) Embed(ctx context.Context, text string, opts *provider.EmbeddingOpts) ([]float32, error) {
	return nil, nil // not needed
}

func TestFirewall_Heuristics(t *testing.T) {
	manager := firewall.NewManager(nil) // nil LLM, only heuristics

	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Empty string", "", false},
		{"Whitespace", "   \n\t ", false},
		{"Irrelevant word", "ok", false},
		{"Runtime panic", "panic: runtime error: index out of range", false},
		{"Valid useful text", "The server runs on port 8080 by default.", true}, // Passes heuristics, no LLM call
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := manager.Validate(ctx, tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if valid != tc.expected {
				t.Errorf("Validation got %v, want %v", valid, tc.expected)
			}
		})
	}
}

func TestFirewall_LLMCheck(t *testing.T) {
	manager := firewall.NewManager(&MockProvider{})
	ctx := context.Background()

	// Text over 50 chars that triggers LLM.
	longValidText := "We found out that the database connection string is configured via the DB_DSN environment variable."
	longInvalidText := "Error: something failed badly and I do not know why it did. File not found probably. Stopping now."

	valid, err := manager.Validate(ctx, longValidText)
	if err != nil || !valid {
		t.Errorf("expected longValidText to pass (contains 'database' -> Mock returns ALLOW)")
	}

	valid, _ = manager.Validate(ctx, longInvalidText)
	if valid {
		t.Errorf("expected longInvalidText to be blocked by Mock LLM (returns REJECT)")
	}
}
