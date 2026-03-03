// Package firewall prevents noisy or failed executions from polluting the Agent's permanent memory.
// It acts as a gatekeeper between the short-term DraftStore and long-term FactStore/InferenceStore.
package firewall

import (
	"context"
	"fmt"
	"strings"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// Policy defines a rule that a piece of information must pass.
type Policy func(text string) bool

// Manager coordinates the validation of facts before they become permanent.
type Manager struct {
	llm      provider.Provider
	policies []Policy
}

// NewManager creates a firewall manager with default policies.
func NewManager(llm provider.Provider) *Manager {
	return &Manager{
		llm: llm,
		policies: []Policy{
			NoErrorsPolicy,
			NoEmptyPolicy,
			NoIrrelevantPolicy,
		},
	}
}

// AddPolicy registers a custom policy rule.
func (m *Manager) AddPolicy(p Policy) {
	m.policies = append(m.policies, p)
}

// Validate checks if a piece of text (usually from a task outcome) is safe to memorize.
func (m *Manager) Validate(ctx context.Context, text string) (bool, error) {
	if text == "" {
		return false, nil
	}

	// 1. Fast heuristics validations
	for _, p := range m.policies {
		if !p(text) {
			logger.L().Debugw("🧱 Firewall blocked by fast heuristic policy")
			return false, nil
		}
	}

	// 2. Slow LLM-based verification (if text is sufficiently complex or needs semantic validation)
	if m.llm != nil && len(text) > 50 {
		return m.validateWithLLM(ctx, text)
	}

	return true, nil
}

// validateWithLLM uses an LLM call to classify the given content as valid learning material or noise.
func (m *Manager) validateWithLLM(ctx context.Context, text string) (bool, error) {
	prompt := `You are an AI Memory Firewall. 
Your job is to decide if the following piece of execution result should be stored in the AI's long-term memory.
You should REJECT it if:
- It's a blatant error stack trace without a resolution.
- It's an empty or obviously useless observation ("File not found" and nothing else).
- It failed the test and the task was aborted.

You should ALLOW it if:
- It describes a successful code change.
- It is a meaningful finding ("We discovered the database name is actually postgres_dev not postgres").
- It is a successfully executed command result.

Respond ONLY with {"result": "ALLOW"} or {"result": "REJECT"}.

Content:
` + text

	resp, err := m.llm.Generate(ctx, []provider.Message{{Role: provider.RoleUser, Content: prompt}}, &provider.Options{JSONMode: true})
	if err != nil {
		return false, fmt.Errorf("llm firewall validation failed: %w", err)
	}

	if strings.Contains(strings.ToUpper(resp.Content), "ALLOW") {
		return true, nil
	}

	logger.L().Debugw("🧱 Firewall blocked by LLM reasoning", "reason", resp.Content)
	return false, nil
}

// --- Built-in heuristic policies ---

// NoEmptyPolicy rejects whitespace-only text.
func NoEmptyPolicy(text string) bool {
	return len(strings.TrimSpace(text)) > 0
}

// NoErrorsPolicy rejects common raw error signatures.
func NoErrorsPolicy(text string) bool {
	lower := strings.ToLower(text)
	// Example heuristics:
	if strings.Contains(lower, "panic: runtime error") ||
		strings.Contains(lower, "segmentation fault") ||
		strings.Contains(lower, "error: command not found") {
		return false
	}
	return true
}

// NoIrrelevantPolicy rejects extremely short uninformative outputs.
func NoIrrelevantPolicy(text string) bool {
	t := strings.TrimSpace(text)
	if t == "ok" || t == "done" || t == "success" {
		// Too short to be useful as an actual learning fact
		return false
	}
	return true
}
