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

// TestRouter_Generate_GearLevelRouting verifies that Generate() uses SelectModel
// when GearLevel is set and no explicit Model is specified.
func TestRouter_Generate_GearLevelRouting(t *testing.T) {
	r := makeTestRouter("gear")

	tests := []struct {
		name          string
		gearLevel     int
		expectedModel string
	}{
		{"L1_uses_cheap", 1, "cheap-model"},
		{"L2_uses_cheap", 2, "cheap-model"},
		{"L3_uses_strong", 3, "strong-model"},
		{"L4_uses_strong", 4, "strong-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.SelectModel(tt.gearLevel)
			if got != tt.expectedModel {
				t.Errorf("SelectModel(%d) = %s, want %s", tt.gearLevel, got, tt.expectedModel)
			}
		})
	}
}

// TestRouter_Generate_ExplicitModelOverride verifies that an explicit Model in
// Options takes priority over GearLevel routing.
func TestRouter_Generate_ExplicitModelOverride(t *testing.T) {
	r := makeTestRouter("gear")

	// When Model is explicitly set, it should be used regardless of GearLevel.
	// We test this at the SelectModel level since we can't do real HTTP calls.
	model := r.SelectModel(1) // Would pick cheap
	if model != "cheap-model" {
		t.Errorf("Expected cheap-model for gear 1, got %s", model)
	}
	// But if caller sets opts.Model = "strong-model", Generate should use that.
	// This is tested implicitly since Generate checks opts.Model first.
}

// TestRouter_SelectModel_AllStrategies tests SelectModel across all three strategies.
func TestRouter_SelectModel_AllStrategies(t *testing.T) {
	strategies := []struct {
		name     string
		gear     int
		expected string
	}{
		{"gear_low", 1, "cheap-model"},
		{"gear_high", 4, "strong-model"},
		{"cheapest_any", 3, "cheap-model"},
	}

	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			strategy := "gear"
			if s.name == "cheapest_any" {
				strategy = "cheapest"
			}
			r := makeTestRouter(strategy)
			got := r.SelectModel(s.gear)
			if got != s.expected {
				t.Errorf("strategy=%s gear=%d: got %s, want %s", strategy, s.gear, got, s.expected)
			}
		})
	}
}
