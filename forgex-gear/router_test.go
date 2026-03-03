package gear_test

import (
	"testing"

	"github.com/awch-D/ForgeX/forgex-core/types"
	gear "github.com/awch-D/ForgeX/forgex-gear"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
)

func TestEvaluate_L1Simple(t *testing.T) {
	analysis := &parser.TaskAnalysis{
		CoreIntent:    "print hello world",
		TechStack:     []string{"golang"},
		ExecutionPlan: []string{"main.go"},
	}

	level := gear.Evaluate(analysis)
	if level != gear.L1Simple {
		t.Errorf("expected L1Simple, got %s", level)
	}
}

func TestEvaluate_L2Medium(t *testing.T) {
	analysis := &parser.TaskAnalysis{
		CoreIntent:    "create a REST API with database",
		TechStack:     []string{"golang", "gin", "sqlite"},
		ExecutionPlan: []string{"main.go", "handler.go", "db.go"},
	}

	level := gear.Evaluate(analysis)
	if level < gear.L2Medium {
		t.Errorf("expected at least L2Medium, got %s", level)
	}
}

func TestEvaluate_L3Complex(t *testing.T) {
	analysis := &parser.TaskAnalysis{
		CoreIntent:     "build a microservice with database, jwt authentication, and concurrent workers",
		TechStack:      []string{"golang", "grpc", "postgresql", "redis", "docker"},
		ExecutionPlan:  []string{"cmd/main.go", "internal/auth/jwt.go", "internal/db/db.go", "internal/worker/pool.go", "internal/handler/api.go", "internal/model/user.go"},
		FilesToModify:  []string{"go.mod", "docker-compose.yml", "Makefile"},
		EstimatedLevel: types.L3,
	}

	level := gear.Evaluate(analysis)
	if level < gear.L3Complex {
		t.Errorf("expected at least L3Complex, got %s", level)
	}
}

func TestEvaluate_LLMEstimationBoost(t *testing.T) {
	// Same task but with high LLM estimation should score higher
	base := &parser.TaskAnalysis{
		CoreIntent:    "create a CLI tool",
		TechStack:     []string{"golang"},
		ExecutionPlan: []string{"main.go", "cmd.go"},
	}

	boosted := &parser.TaskAnalysis{
		CoreIntent:     "create a CLI tool",
		TechStack:      []string{"golang"},
		ExecutionPlan:  []string{"main.go", "cmd.go"},
		EstimatedLevel: types.L3,
	}

	baseLevel := gear.Evaluate(base)
	boostedLevel := gear.Evaluate(boosted)

	if boostedLevel < baseLevel {
		t.Errorf("LLM estimation boost should increase level: base=%s, boosted=%s", baseLevel, boostedLevel)
	}
}

func TestEvaluate_FilesToModifyImpact(t *testing.T) {
	// Many files to modify should increase complexity
	analysis := &parser.TaskAnalysis{
		CoreIntent:    "refactor the auth module",
		TechStack:     []string{"golang"},
		ExecutionPlan: []string{"auth.go"},
		FilesToModify: []string{
			"handler/login.go", "handler/register.go", "handler/profile.go",
			"middleware/jwt.go", "middleware/cors.go", "middleware/rate.go",
			"model/user.go", "model/session.go",
		},
	}

	level := gear.Evaluate(analysis)
	if level < gear.L2Medium {
		t.Errorf("many files to modify should bump level to at least L2, got %s", level)
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    gear.Level
		expected string
	}{
		{gear.L1Simple, "L1-Simple"},
		{gear.L2Medium, "L2-Medium"},
		{gear.L3Complex, "L3-Complex"},
		{gear.L4Advanced, "L4-Advanced"},
		{gear.Level(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestLevel_NeedsMultiAgent(t *testing.T) {
	if gear.L1Simple.NeedsMultiAgent() {
		t.Error("L1 should not need multi-agent")
	}
	if gear.L2Medium.NeedsMultiAgent() {
		t.Error("L2 should not need multi-agent")
	}
	if !gear.L3Complex.NeedsMultiAgent() {
		t.Error("L3 should need multi-agent")
	}
	if !gear.L4Advanced.NeedsMultiAgent() {
		t.Error("L4 should need multi-agent")
	}
}
