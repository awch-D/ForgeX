// Package router provides an intelligent multi-model LLM routing layer.
// It implements provider.Provider and dispatches requests to downstream
// clients based on configurable strategies (gear, cheapest, fallback).
package router

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/awch-D/ForgeX/forgex-core/config"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// Strategy defines how the router selects a model.
type Strategy string

const (
	StrategyGear     Strategy = "gear"
	StrategyCheapest Strategy = "cheapest"
	StrategyFallback Strategy = "fallback"
)

// Router implements provider.Provider and delegates to multiple downstream clients.
type Router struct {
	mu       sync.RWMutex
	clients  map[string]*litellm.Client // model name -> client
	tiers    map[string]string          // model name -> tier ("high"/"low")
	models   []string                   // ordered model names
	strategy Strategy
	fallback string // default model name (first high-tier or first model)
}

// New creates a Router from the router config section.
func New(routerCfg *config.RouterConfig, defaultLLM *config.LLMConfig) *Router {
	r := &Router{
		clients: make(map[string]*litellm.Client),
		tiers:   make(map[string]string),
	}

	// Parse strategy
	switch strings.ToLower(routerCfg.Strategy) {
	case "cheapest":
		r.strategy = StrategyCheapest
	case "fallback":
		r.strategy = StrategyFallback
	default:
		r.strategy = StrategyGear
	}

	// Build downstream clients
	for _, m := range routerCfg.Models {
		cfg := &config.LLMConfig{
			Endpoint:    m.Endpoint,
			APIKey:      m.APIKey,
			Model:       m.Name,
			MaxTokens:   defaultLLM.MaxTokens,
			Temperature: defaultLLM.Temperature,
		}
		client := litellm.NewClient(cfg)
		r.clients[m.Name] = client
		r.tiers[m.Name] = strings.ToLower(m.Tier)
		r.models = append(r.models, m.Name)
	}

	// Pick fallback: first high-tier model, or first model
	for _, name := range r.models {
		if r.tiers[name] == "high" {
			r.fallback = name
			break
		}
	}
	if r.fallback == "" && len(r.models) > 0 {
		r.fallback = r.models[0]
	}

	logger.L().Infow("🔀 LLM Router initialized",
		"strategy", string(r.strategy),
		"models", r.models,
		"fallback", r.fallback,
	)

	return r
}

// SelectModel picks the best model based on strategy and gear level.
// gearLevel: 1-5, where 1 is simplest and 5 is most complex.
func (r *Router) SelectModel(gearLevel int) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch r.strategy {
	case StrategyGear:
		if gearLevel <= 2 {
			// Simple task → pick a low-tier model
			for _, name := range r.models {
				if r.tiers[name] == "low" {
					return name
				}
			}
		}
		// Complex task or no low-tier available → high-tier
		for _, name := range r.models {
			if r.tiers[name] == "high" {
				return name
			}
		}
		return r.fallback

	case StrategyCheapest:
		// Always pick the first low-tier, fallback to any
		for _, name := range r.models {
			if r.tiers[name] == "low" {
				return name
			}
		}
		return r.fallback

	case StrategyFallback:
		// Return the first low-tier model; caller should retry with fallback on error
		for _, name := range r.models {
			if r.tiers[name] == "low" {
				return name
			}
		}
		return r.fallback

	default:
		return r.fallback
	}
}

// FallbackModel returns the high-tier fallback model name.
func (r *Router) FallbackModel() string {
	return r.fallback
}

// Generate implements provider.Provider.
// It routes the request to the appropriate downstream client.
// If opts.GearLevel > 0 and no explicit Model is set, model selection
// is delegated to SelectModel to apply the configured routing strategy.
func (r *Router) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	model := r.fallback
	if opts != nil && opts.Model != "" {
		// Caller explicitly requested a model; honour it.
		model = opts.Model
	} else if opts != nil && opts.GearLevel > 0 {
		// Gear-aware routing: pick the model tier based on task complexity.
		model = r.SelectModel(opts.GearLevel)
		logger.L().Infow("🔀 Router: gear-aware model selected",
			"gear_level", opts.GearLevel,
			"model", model,
			"strategy", string(r.strategy),
		)
	}

	client, ok := r.clients[model]
	if !ok {
		// Unknown model → try fallback
		logger.L().Warnw("Router: unknown model, using fallback", "requested", model, "fallback", r.fallback)
		client, ok = r.clients[r.fallback]
		if !ok {
			return nil, fmt.Errorf("router: no client available for model %q or fallback %q", model, r.fallback)
		}
		model = r.fallback
	}

	// Ensure opts carries the correct model name
	routedOpts := &provider.Options{
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
	if opts != nil {
		if opts.Temperature != 0 {
			routedOpts.Temperature = opts.Temperature
		}
		if opts.MaxTokens != 0 {
			routedOpts.MaxTokens = opts.MaxTokens
		}
		routedOpts.JSONMode = opts.JSONMode
	}

	resp, err := client.Generate(ctx, messages, routedOpts)

	// Fallback strategy: if the primary model fails, try the fallback
	if err != nil && r.strategy == StrategyFallback && model != r.fallback {
		logger.L().Warnw("Router: primary model failed, trying fallback",
			"primary", model, "fallback", r.fallback, "error", err)
		fbClient := r.clients[r.fallback]
		routedOpts.Model = r.fallback
		return fbClient.Generate(ctx, messages, routedOpts)
	}

	return resp, err
}

// Embed implements provider.Provider.
func (r *Router) Embed(ctx context.Context, text string, opts *provider.EmbeddingOpts) ([]float32, error) {
	// For embedding, we just use the fallback client
	client, ok := r.clients[r.fallback]
	if !ok {
		return nil, fmt.Errorf("router: no client available for embedding")
	}
	return client.Embed(ctx, text, opts)
}
