// Package provider defines the standard interface for all LLM providers.
package provider

import "context"

// Role defines the message author role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role
	Content string
}

// Response represents the output from an LLM.
type Response struct {
	Content      string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	Model        string // The actual model that served the response
}

// Options allows overriding default behavior for a specific completion.
type Options struct {
	Model       string
	Temperature float64
	MaxTokens   int
	JSONMode    bool // If true, requires the LLM to output valid JSON
	GearLevel   int  // Complexity level (1-4) used by Router to select the appropriate model tier
}

// EmbeddingOpts configuration for vector embedding generation.
type EmbeddingOpts struct {
	Model string
}

// Provider is the interface that varying LLM clients must implement.
type Provider interface {
	// Generate performs a single completion call.
	Generate(ctx context.Context, messages []Message, opts *Options) (*Response, error)
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string, opts *EmbeddingOpts) ([]float32, error)
}
