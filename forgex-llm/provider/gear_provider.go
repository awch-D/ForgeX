// Package provider defines the standard interface for all LLM providers.
package provider

import "context"

// GearProvider is a decorator that automatically injects the GearLevel
// into every LLM request, enabling the Router to select the appropriate
// model tier based on task complexity.
type GearProvider struct {
	inner     Provider
	gearLevel int
}

// WithGear wraps a Provider and injects the given gearLevel into every
// Generate call's Options, so the Router can apply gear-aware model selection.
func WithGear(p Provider, level int) Provider {
	return &GearProvider{inner: p, gearLevel: level}
}

// Generate injects GearLevel into opts and delegates to the inner Provider.
func (g *GearProvider) Generate(ctx context.Context, messages []Message, opts *Options) (*Response, error) {
	merged := &Options{}
	if opts != nil {
		*merged = *opts
	}
	// Only inject if GearLevel is not already set by the caller.
	if merged.GearLevel == 0 {
		merged.GearLevel = g.gearLevel
	}
	return g.inner.Generate(ctx, messages, merged)
}

// Embed delegates to the inner Provider unchanged (gear level doesn't affect embeddings).
func (g *GearProvider) Embed(ctx context.Context, text string, opts *EmbeddingOpts) ([]float32, error) {
	return g.inner.Embed(ctx, text, opts)
}
