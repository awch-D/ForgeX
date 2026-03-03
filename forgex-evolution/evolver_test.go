package evolution

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvolver_EvaluateGoProject_Pass(t *testing.T) {
	// Create a temporary Go project that compiles and passes tests
	dir, _ := os.MkdirTemp("", "evo-test-pass")
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main
func main() {}
func Add(a, b int) int { return a + b }
`), 0644)
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(`package main
import "testing"
func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 { t.Fatal("wrong") }
}
`), 0644)

	e := NewEvolver()
	score := e.Evaluate(dir)

	if !score.CompilePass {
		t.Errorf("Expected compile to pass, errors: %s", score.Errors)
	}
	if !score.TestPass {
		t.Errorf("Expected tests to pass, errors: %s", score.Errors)
	}
	if score.Total < 0.7 {
		t.Errorf("Expected high score, got %.2f", score.Total)
	}
	if e.ShouldRetry(score) {
		t.Errorf("Should not retry on high score")
	}
}

func TestEvolver_EvaluateGoProject_CompileFail(t *testing.T) {
	dir, _ := os.MkdirTemp("", "evo-test-fail")
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main
func main() {
	undefinedFunc()  // This will fail to compile
}
`), 0644)

	e := NewEvolver()
	score := e.Evaluate(dir)

	if score.CompilePass {
		t.Error("Expected compile to fail")
	}
	if score.Total != 0.0 {
		t.Errorf("Expected score 0.0 for compile failure, got %.2f", score.Total)
	}
	if !e.ShouldRetry(score) {
		t.Error("Should retry on compile failure")
	}
}

func TestEvolver_BuildRetryPrompt(t *testing.T) {
	e := NewEvolver()

	score := Score{
		CompilePass: false,
		Errors:      "undefined: undefinedFunc",
	}

	prompt := e.BuildRetryPrompt(score)

	if prompt == "" {
		t.Error("Expected non-empty retry prompt")
	}
	if !contains(prompt, "COMPILATION FAILED") {
		t.Error("Expected prompt to mention compilation failure")
	}
	if !contains(prompt, "undefinedFunc") {
		t.Error("Expected prompt to contain error details")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
