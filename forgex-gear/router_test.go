package gear_test

import (
	"testing"

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
		CoreIntent:    "create a REST API with jwt auth",
		TechStack:     []string{"golang", "gin"},
		ExecutionPlan: []string{"main.go", "handler.go", "auth.go"},
	}

	level := gear.Evaluate(analysis)
	if level < gear.L2Medium {
		t.Errorf("expected at least L2Medium, got %s", level)
	}
}

func TestEvaluate_L3Complex(t *testing.T) {
	analysis := &parser.TaskAnalysis{
		CoreIntent:    "build a microservice with database, jwt authentication, and concurrent workers",
		TechStack:     []string{"golang", "grpc", "postgresql", "redis"},
		ExecutionPlan: []string{"cmd/main.go", "internal/auth/jwt.go", "internal/db/db.go", "internal/worker/pool.go", "internal/handler/api.go", "internal/model/user.go"},
	}

	level := gear.Evaluate(analysis)
	if level < gear.L3Complex {
		t.Errorf("expected at least L3Complex, got %s", level)
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
