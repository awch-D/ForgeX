package router

import (
	"testing"

	"github.com/awch-D/ForgeX/forgex-core/config"
)

func makeTestRouter(strategy string) *Router {
	cfg := &config.RouterConfig{
		Strategy: strategy,
		Models: []config.ModelConfig{
			{Name: "cheap-model", Endpoint: "http://fake:4000", APIKey: "k1", Tier: "low"},
			{Name: "strong-model", Endpoint: "http://fake:4000", APIKey: "k2", Tier: "high"},
		},
	}
	defaultLLM := &config.LLMConfig{MaxTokens: 4096, Temperature: 0.7}
	return New(cfg, defaultLLM)
}

func TestRouter_GearStrategy(t *testing.T) {
	r := makeTestRouter("gear")

	// Simple tasks (gear 1-2) should pick low-tier
	model := r.SelectModel(1)
	if model != "cheap-model" {
		t.Errorf("Gear L1: expected cheap-model, got %s", model)
	}
	model = r.SelectModel(2)
	if model != "cheap-model" {
		t.Errorf("Gear L2: expected cheap-model, got %s", model)
	}

	// Complex tasks (gear 3+) should pick high-tier
	model = r.SelectModel(3)
	if model != "strong-model" {
		t.Errorf("Gear L3: expected strong-model, got %s", model)
	}
	model = r.SelectModel(5)
	if model != "strong-model" {
		t.Errorf("Gear L5: expected strong-model, got %s", model)
	}
}

func TestRouter_CheapestStrategy(t *testing.T) {
	r := makeTestRouter("cheapest")

	// Should always pick cheap
	for _, g := range []int{1, 3, 5} {
		model := r.SelectModel(g)
		if model != "cheap-model" {
			t.Errorf("Cheapest gear %d: expected cheap-model, got %s", g, model)
		}
	}
}

func TestRouter_FallbackModel(t *testing.T) {
	r := makeTestRouter("fallback")

	if r.FallbackModel() != "strong-model" {
		t.Errorf("Expected fallback to be strong-model, got %s", r.FallbackModel())
	}
}

func TestRouter_ClientResolution(t *testing.T) {
	r := makeTestRouter("gear")

	// Verify that clients are registered
	if len(r.clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(r.clients))
	}
	if _, ok := r.clients["cheap-model"]; !ok {
		t.Error("cheap-model client not found")
	}
	if _, ok := r.clients["strong-model"]; !ok {
		t.Error("strong-model client not found")
	}
}
